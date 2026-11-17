// Package win — Windows-специфічні хелпери: права адміністратора,
// UAC-підвищення, реєстр, завершення процесів.
package win

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const createNoWindow = 0x08000000 // CREATE_NO_WINDOW

// IsAdmin — чи запущено процес з правами адміністратора.
func IsAdmin() bool {
	proc := windows.NewLazySystemDLL("shell32.dll").NewProc("IsUserAnAdmin")
	ret, _, _ := proc.Call()
	return ret != 0
}

// Elevate — запитує UAC і перезапускає поточний exe від імені адміністратора.
// Повертає true, якщо процес перезапущено (текучий треба завершувати).
func Elevate() bool {
	if IsAdmin() {
		return true
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	var argsPtr *uint16
	if len(os.Args) > 1 {
		argsPtr, _ = syscall.UTF16PtrFromString(strings.Join(os.Args[1:], " "))
	}
	proc := windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteW")
	ret, _, _ := proc.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(argsPtr)),
		0,
		1, // SW_SHOWNORMAL
	)
	return ret > 32
}

// RunHidden — запускає консольну утиліту без блимання вікна.
func RunHidden(name string, args ...string) (string, int, error) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	out, err := cmd.CombinedOutput()
	code := 0
	if cmd.ProcessState != nil {
		code = cmd.ProcessState.ExitCode()
	}
	return string(out), code, err
}

// KillProcessByName — прибиває всі процеси за ім'ям образу.
func KillProcessByName(image string) {
	_, _, _ = RunHidden("taskkill", "/F", "/IM", image)
}

// IsProcessRunning — чи запущено процес за ім'ям образу.
func IsProcessRunning(image string) bool {
	out, _, err := RunHidden("tasklist", "/FI", "IMAGENAME eq "+image, "/NH")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(out), strings.ToLower(image))
}

// ─── Реєстр: автозапуск ───────────────────────────────────────────────────

const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

const appRegName = "ZiperVPN"

// IsAutostartEnabled — чи є програма в автозапуску Windows.
func IsAutostartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	val, _, err := k.GetStringValue(appRegName)
	if err != nil {
		return false
	}
	return val != ""
}

// SetAutostart — додає/видаляє програму з автозапуску Windows.
func SetAutostart(enable bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if enable {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		cmd := fmt.Sprintf(`"%s" --autostart`, exe)
		return k.SetStringValue(appRegName, cmd)
	}
	err = k.DeleteValue(appRegName)
	if err != nil && err != registry.ErrNotExist {
		return err
	}
	return nil
}

// ─── Kill Switch (брандмауер Windows) ──────────────────────────────────────

// killSwitchAllowRule — ім'я правил-винятків для Kill Switch.
const killSwitchAllowRule = "Ziper VPN Kill Switch Allow"

// EnableKillSwitch — блокує ВЕСЬ вихідний трафік, крім openvpn.exe та
// самого застосунку (політика брандмауера blockoutbound + правила-винятки).
// Потребує прав адміністратора.
func EnableKillSwitch(openvpnPath string) error {
	if !IsAdmin() {
		return fmt.Errorf("потрібні права адміністратора")
	}
	// прибираємо старі правила-винятки
	_, _, _ = RunHidden("netsh", "advfirewall", "firewall", "delete", "rule", "name="+killSwitchAllowRule)

	// дозволяємо openvpn.exe — весь тунельований трафік іде через цей процес
	if openvpnPath != "" {
		_, rc, err := RunHidden("netsh", "advfirewall", "firewall", "add", "rule",
			"name="+killSwitchAllowRule+" (OpenVPN)", "dir=out", "action=allow",
			"enable=yes", "profile=any", "program="+openvpnPath)
		if err != nil || rc != 0 {
			return fmt.Errorf("не вдалося додати виняток для openvpn.exe (код %d)", rc)
		}
	}
	// дозволяємо сам застосунок — інакше він не зможе оновити список серверів
	exe, err := os.Executable()
	if err == nil {
		_, rc, err2 := RunHidden("netsh", "advfirewall", "firewall", "add", "rule",
			"name="+killSwitchAllowRule+" (Ziper)", "dir=out", "action=allow",
			"enable=yes", "profile=any", "program="+exe)
		if err2 != nil || rc != 0 {
			return fmt.Errorf("не вдалося додати виняток для застосунку (код %d)", rc)
		}
	}
	// вмикаємо блокування вихідного трафіку для всіх профілів мережі
	out, rc, err := RunHidden("netsh", "advfirewall", "set", "allprofiles",
		"firewallpolicy", "blockinbound,blockoutbound")
	if err != nil || rc != 0 {
		return fmt.Errorf("не вдалося ввімкнути блокування вихідного трафіку: %s", strings.TrimSpace(out))
	}
	return nil
}

// DisableKillSwitch — відновлює стандартну політику брандмауера
// (blockinbound,allowoutbound) та прибирає правила-винятки Kill Switch.
func DisableKillSwitch() error {
	_, _, _ = RunHidden("netsh", "advfirewall", "firewall", "delete", "rule", "name="+killSwitchAllowRule+" (OpenVPN)")
	_, _, _ = RunHidden("netsh", "advfirewall", "firewall", "delete", "rule", "name="+killSwitchAllowRule+" (Ziper)")
	out, rc, err := RunHidden("netsh", "advfirewall", "set", "allprofiles",
		"firewallpolicy", "blockinbound,allowoutbound")
	if err != nil || rc != 0 {
		return fmt.Errorf("не вдалося відновити політику брандмауера: %s", strings.TrimSpace(out))
	}
	return nil
}
