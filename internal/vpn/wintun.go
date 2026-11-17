package vpn

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"ziper-vpn-go/internal/i18n"
)

// Wintun wrapper — створення/видалення адаптера через wintun.dll.
// Аналог _wintun_ensure_adapter/_wintun_release_adapter з Python-версії.

var (
	wintunMu      sync.Mutex
	wintunHandle  uintptr
	wintunAdapter string
)

const adapterName = "ZiperVPN"
const tunnelType = "OpenVPN Tunnels"

// (функція openVPNDir видалена — використовується глобальна змінна з vpn.go)

func wintunDLLPath() string {
	return filepath.Join(openVPNDir, "wintun.dll")
}

type wintunAPI struct {
	dll           *syscall.DLL
	createAdapter *syscall.Proc
	closeAdapter  *syscall.Proc
}

var wintun *wintunAPI

func loadWintun(log func(string)) (*wintunAPI, error) {
	if wintun != nil {
		return wintun, nil
	}
	path := wintunDLLPath()
	if log != nil { log("loadWintun: path=" + path) }
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("wintun.dll не знайдено: %s", path)
	}
	if log != nil { log("loadWintun: calling LoadDLL") }
	dll, err := syscall.LoadDLL(path)
	if err != nil {
		return nil, err
	}
	if log != nil { log("loadWintun: LoadDLL ok, FindProc create") }
	create, err := dll.FindProc("WintunCreateAdapter")
	if err != nil {
		return nil, err
	}
	if log != nil { log("loadWintun: FindProc close") }
	close, err := dll.FindProc("WintunCloseAdapter")
	if err != nil {
		return nil, err
	}
	wintun = &wintunAPI{dll: dll, createAdapter: create, closeAdapter: close}
	if log != nil { log("loadWintun: done") }
	return wintun, nil
}

// ReleaseWintunAdapter — закриває handle створеного адаптера → видаляє його.
// Викликати ПІСЛЯ завершення openvpn, не під час активного тунелю.
func ReleaseWintunAdapter() {
	wintunMu.Lock()
	defer wintunMu.Unlock()
	releaseWintunAdapterLocked()
}

func releaseWintunAdapterLocked() {
	if wintunHandle == 0 {
		return
	}
	if wintun != nil {
		_, _, _ = wintun.closeAdapter.Call(wintunHandle)
	}
	wintunHandle = 0
}

// EnsureWintunAdapter — створює свіжий Wintun адаптер (кожне підключення).
func EnsureWintunAdapter(log func(string)) bool {
	wintunMu.Lock()
	defer wintunMu.Unlock()
	log("EnsureWintunAdapter: start")
	releaseWintunAdapterLocked()
	log("EnsureWintunAdapter: after release")
	api, err := loadWintun(log)
	if err != nil {
		if log != nil {
			log(fmt.Sprintf("[WARN] %s — перехід на tap-windows6.", err))
		}
		return false
	}
	log("loadWintun OK, викликаю WintunCreateAdapter")
	namePtr, _ := syscall.UTF16PtrFromString(adapterName)
	typePtr, _ := syscall.UTF16PtrFromString(tunnelType)
	log(i18n.Trf("Виклик WintunCreateAdapter для {name}...", map[string]string{"name": adapterName}))
	// WintunCreateAdapter(Name, TunnelType, RequestedGUID *GUID) HANDLE
	h, _, callErr := api.createAdapter.Call(
		uintptr(unsafe.Pointer(namePtr)),
		uintptr(unsafe.Pointer(typePtr)),
		0,
	)
	if h != 0 {
		wintunHandle = h
		if log != nil {
			log(i18n.Trf("Створено Wintun адаптер \"{name}\".", map[string]string{"name": adapterName}))
		}
		return true
	}
	if log != nil {
		log(i18n.Trf("WintunCreateAdapter відхилено (код помилки Windows: {code}).",
			map[string]string{"code": fmt.Sprintf("%v", callErr)}))
	}
	return false
}

// WintunDLLExists — перевіряє наявність wintun.dll.
func WintunDLLExists() bool {
	_, err := os.Stat(wintunDLLPath())
	return err == nil
}

// ─── tapctl — створення адаптерів альтернативним способом ─────────────────

func tapctlPath() string {
	return filepath.Join(openVPNDir, "tapctl.exe")
}

// TapctlCreateAdapter — створює адаптер через tapctl.exe ('wintun'/'tap').
func TapctlCreateAdapter(kind string, log func(string)) bool {
	path := tapctlPath()
	if _, err := os.Stat(path); err != nil {
		if log != nil {
			log(i18n.Tr("tapctl.exe не знайдено в папці openVPN."))
		}
		return false
	}
	hwid := "root\\tap0901"
	if kind == "wintun" {
		hwid = "wintun"
	}
	out, rc, err := runTool(path, "create", "--hwid", hwid, "--name", "Ziper VPN")
	if err != nil && log != nil {
		log(i18n.Trf("tapctl {kind}: {out}", map[string]string{"kind": kind, "out": truncate(out, 300)}))
		return false
	}
	if rc == 0 && (strings.Contains(out, "{") || strings.Contains(out, "Created")) {
		if log != nil {
			log(i18n.Trf("Створено {kind} адаптер \"Ziper VPN\" (tapctl).",
				map[string]string{"kind": kind}))
		}
		return true
	}
	// tapctl іноді повертає rc!=0 навіть при успіху — перевіряємо GUID
	if strings.Contains(out, "{") {
		if log != nil {
			log(i18n.Trf("tapctl повернув код {code}, але адаптер створено за GUID.",
				map[string]string{"code": fmt.Sprintf("%d", rc)}))
		}
		return true
	}
	if log != nil {
		log(i18n.Trf("tapctl {kind}: {out}", map[string]string{"kind": kind, "out": truncate(out, 300)}))
	}
	return false
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " | ")
	if len(s) > n {
		return s[:n]
	}
	return s
}
