// Package vpn — VPN логіка з OpenVPN Management Interface.
// Керування через TCP-сокет: openvpn --management 127.0.0.1 7505.
// Підтримка Wintun драйвера, блокування витоків DNS та вибору протоколу.
package vpn

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"ziper-vpn-go/internal/i18n"
	"ziper-vpn-go/internal/win"
)

// ─── Константи та шляхи ────────────────────────────────────────────────────

const (
	mgmtHost = "127.0.0.1"
	mgmtPort = 7505
)

var (
	appDir     string
	openVPNDir string
)

func init() {
	exe, _ := os.Executable()
	appDir = filepath.Dir(exe)
	openVPNDir = filepath.Join(appDir, "openVPN")
}

func localOpenVPN() string { return filepath.Join(openVPNDir, "openvpn.exe") }

var systemOpenVPNPaths = []string{
	`C:\Program Files\OpenVPN\bin\openvpn.exe`,
	`C:\Program Files (x86)\OpenVPN\bin\openvpn.exe`,
}

const (
	openVPNMsiURL  = "https://swupdate.openvpn.org/community/releases/OpenVPN-2.6.12-I001-amd64.msi"
	wintunZipURL   = "https://www.wintun.net/builds/wintun-0.14.1.zip"
	tapWindowsURL  = "https://build.openvpn.net/downloads/releases/tap-windows-9.24.7-I601-Win10.exe"
)

// ─── Стан сесії ────────────────────────────────────────────────────────────

type session struct {
	mu           sync.Mutex
	process      *exec.Cmd
	procHandle   *os.Process
	mgmtConn     net.Conn
	cancelReq    bool
	disconnecting bool
	counter      int
	lastState    string
	dnsServers   []string // DNS, який треба прописати на адаптері тунелю
}

var (
	globalMu        sync.Mutex
	curSession      = &session{lastState: "NONE"}
	logCallback     func(string)
	onConnectedCb    func()
	onDisconnectedCb func(string)
)

// SetCallbacks — встановлює обробники подій VPN (викликається з app.go).
func SetCallbacks(log func(string), onConnected func(), onDisconnected func(string)) {
	logCallback = log
	onConnectedCb = onConnected
	onDisconnectedCb = onDisconnected
}

func logf(msg string) {
	if logCallback != nil {
		logCallback(msg)
	}
}

// LastMgmtState — останній реальний стан тунелю (джерело істини для UI).
func LastMgmtState() string {
	globalMu.Lock()
	defer globalMu.Unlock()
	return curSession.lastState
}

// RequestCancel — запитує відміну поточної спроби підключення.
func RequestCancel() {
	globalMu.Lock()
	s := curSession
	s.mu.Lock()
	s.cancelReq = true
	s.disconnecting = true
	s.mu.Unlock()
	globalMu.Unlock()
}

func isCancelled(s *session) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cancelReq
}

// FindOpenVPN — повертає шлях до робочого openvpn.exe або "".
func FindOpenVPN() string {
	if _, err := os.Stat(localOpenVPN()); err == nil {
		return localOpenVPN()
	}
	for _, p := range systemOpenVPNPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// ─── Завантаження OpenVPN / Wintun / TAP ───────────────────────────────────

// DownloadWintun — завантажує драйвер Wintun (wintun.dll).
func DownloadWintun() string {
	dest := filepath.Join(openVPNDir, "wintun.dll")
	if _, err := os.Stat(dest); err == nil {
		return dest
	}
	logf(i18n.Tr("Завантажую драйвер Wintun..."))
	resp, err := http.Get(wintunZipURL)
	if err != nil {
		logf(i18n.Tr("Не вдалося завантажити Wintun: ") + err.Error())
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		logf(i18n.Tr("Не вдалося завантажити Wintun: HTTP ") + fmt.Sprint(resp.StatusCode))
		return ""
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return ""
	}
	for _, f := range zr.File {
		if strings.Contains(f.Name, "amd64/wintun.dll") {
			_ = os.MkdirAll(openVPNDir, 0o755)
			rc, err := f.Open()
			if err != nil {
				return ""
			}
			content, _ := io.ReadAll(rc)
			rc.Close()
			if err := os.WriteFile(dest, content, 0o644); err == nil {
				logf(i18n.Tr("Wintun успішно встановлено: ") + dest)
				return dest
			}
		}
	}
	return ""
}

// DownloadOpenVPN — завантажує офіційний MSI OpenVPN та розпаковує бінарники.
func DownloadOpenVPN() string {
	_ = os.MkdirAll(openVPNDir, 0o755)
	tmp := os.TempDir()
	msiPath := filepath.Join(tmp, "openvpn_installer.msi")
	extractDir := filepath.Join(tmp, "openvpn_extract")

	logf(i18n.Tr("Завантажую OpenVPN MSI (~8 МБ)..."))
	resp, err := http.Get(openVPNMsiURL)
	if err != nil {
		logf(i18n.Tr("Помилка: ") + err.Error())
		return ""
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	if err := os.WriteFile(msiPath, data, 0o644); err != nil {
		return ""
	}

	_ = os.MkdirAll(extractDir, 0o755)
	logf(i18n.Tr("Розпаковую OpenVPN..."))
	out, rc, _ := win.RunHidden("msiexec", "/a", msiPath, "/qn", "TARGETDIR="+extractDir)
	_ = out
	if rc != 0 && rc != 1638 {
		logf(i18n.Tr("msiexec завершився з кодом ") + fmt.Sprint(rc))
		return ""
	}

	// шукаємо openvpn.exe у розпакованому дереві
	var binDir string
	_ = filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if strings.EqualFold(info.Name(), "openvpn.exe") {
			binDir = filepath.Dir(path)
			return filepath.SkipAll
		}
		return nil
	})
	if binDir == "" {
		logf("[ERROR] openvpn.exe не знайдено в інсталяторі.")
		return ""
	}
	entries, _ := os.ReadDir(binDir)
	for _, e := range entries {
		if !e.IsDir() {
			src := filepath.Join(binDir, e.Name())
			dst := filepath.Join(openVPNDir, e.Name())
			content, err := os.ReadFile(src)
			if err == nil {
				_ = os.WriteFile(dst, content, 0o644)
			}
		}
	}
	DownloadWintun()
	_ = os.Remove(msiPath)
	_ = os.RemoveAll(extractDir)
	logf(i18n.Tr("OpenVPN готовий до роботи: ") + localOpenVPN())
	return localOpenVPN()
}

// EnsureOpenVPN — гарантує наявність openvpn.exe та wintun.dll.
func EnsureOpenVPN() string {
	path := FindOpenVPN()
	if path != "" {
		dir := filepath.Dir(path)
		wintunPath := filepath.Join(dir, "wintun.dll")
		if _, err := os.Stat(wintunPath); err != nil {
			if dir == openVPNDir {
				DownloadWintun()
			} else {
				srcWintun := filepath.Join(appDir, "wintun.dll")
				if content, err := os.ReadFile(srcWintun); err == nil {
					_ = os.WriteFile(wintunPath, content, 0o644)
					logf(i18n.Tr("Копіюю wintun.dll до ") + dir)
				}
			}
		}
		return path
	}
	return DownloadOpenVPN()
}

// ─── Адаптери та cleanup ───────────────────────────────────────────────────

// KillLingeringOpenVPN — прибиває залишені процеси openvpn.exe.
func KillLingeringOpenVPN() {
	win.KillProcessByName("openvpn.exe")
	time.Sleep(300 * time.Millisecond)
}

// WaitOpenVPNExited — чекає завершення ВСІХ старих openvpn.exe.
func WaitOpenVPNExited(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !win.IsProcessRunning("openvpn.exe") {
			return true
		}
		KillLingeringOpenVPN()
		time.Sleep(300 * time.Millisecond)
	}
	logf(i18n.Tr("Старі процеси openvpn.exe не завершились — можлива помилка адаптера."))
	return false
}

// EnsureInteractiveService — встановлює та запускає OpenVPNServiceInteractive.
func EnsureInteractiveService() {
	if !win.IsAdmin() {
		return
	}
	servExe := filepath.Join(openVPNDir, "openvpnserv.exe")
	if _, err := os.Stat(servExe); err != nil {
		return
	}
	out, rc, _ := win.RunHidden("sc.exe", "query", "OpenVPNServiceInteractive")
	if rc != 0 || strings.Contains(out, "FAILED 1060") {
		logf(i18n.Tr("Реєстрація системної служби OpenVPN Interactive Service..."))
		_, _, _ = win.RunHidden(servExe, "-install")
		_, _, _ = win.RunHidden(servExe, "-start", "interactive")
	} else if !strings.Contains(out, "RUNNING") {
		_, _, _ = win.RunHidden("net", "start", "OpenVPNServiceInteractive")
	}
}

// CleanupWintunGhosts — видаляє фантомні "OpenVPN Tunnels" Wintun-пристрої.
func CleanupWintunGhosts() {
	names := listSWDWintunNames()
	var ghosts []string
	for _, inst := range names {
		fn := getDeviceFriendlyName(inst)
		if strings.Contains(fn, tunnelType) || strings.Contains(fn, "ZiperVPN") || strings.Contains(fn, adapterName) {
			ghosts = append(ghosts, inst)
		}
	}
	if len(ghosts) == 0 {
		logf(i18n.Tr("Фанітомних Wintun пристроїв не виявлено."))
		return
	}
	for _, inst := range ghosts {
		logf(i18n.Tr("Видалення фантомного Wintun пристрою ") + inst + "...")
		_, rc, _ := win.RunHidden("pnputil", "/remove-device", `SWD\Wintun\`+inst)
		if rc != 0 {
			logf(i18n.Tr("Не вдалося видалити фантомний пристрій ") + inst + " (код " + fmt.Sprint(rc) + ").")
		}
	}
}

func listSWDWintunNames() []string {
	var out []string
	path := `SYSTEM\CurrentControlSet\Enum\SWD\Wintun`
	k, err := regOpenKey(regHKLM, path, regKeyRead)
	if err != nil {
		return out
	}
	defer k.Close()
	names, err := k.ReadSubKeyNames(0)
	if err != nil && len(names) == 0 {
		return out
	}
	return names
}

func getDeviceFriendlyName(instance string) string {
	k, err := regOpenKey(regHKLM, `SYSTEM\CurrentControlSet\Enum\SWD\Wintun\`+instance, regKeyRead)
	if err != nil {
		return ""
	}
	defer k.Close()
	val, _ := regGetString(k, "FriendlyName")
	return val
}

// EnsureNetworkAdapter — пірамідна логіка дій над мережевим адаптером.
// Повертає ім'я драйвера ('wintun'/'tap-windows6') або "".
func EnsureNetworkAdapter() string {
	CleanupWintunGhosts()
	if EnsureWintunAdapter(logf) {
		return "wintun"
	}
	if TapctlCreateAdapter("wintun", logf) {
		return "wintun"
	}
	if ensureTapDriver() && TapctlCreateAdapter("tap", logf) {
		return "tap-windows6"
	}
	logf(i18n.Tr("Не вдалося підготувати жодний мережевий адаптер."))
	return ""
}

func ensureTapDriver() bool {
	// драйвер tap0901 вже зареєстрований?
	if _, err := regOpenKey(regHKLM, `SYSTEM\CurrentControlSet\Services\tap0901`, regKeyRead); err == nil {
		logf(i18n.Tr("Драйвер tap0901 вже встановлений в системі."))
		return true
	}
	logf(i18n.Tr("Завантаження офіційного драйвера tap0901 (tap-windows6)..."))
	resp, err := http.Get(tapWindowsURL)
	if err != nil {
		logf(i18n.Tr("Не вдалося встановити драйвер tap0901: ") + err.Error())
		return false
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	tmp := os.TempDir()
	exePath := filepath.Join(tmp, "tap-windows-installer.exe")
	if err := os.WriteFile(exePath, data, 0o755); err != nil {
		return false
	}
	defer func() { _ = os.Remove(exePath) }()
	logf(i18n.Tr("Встановлення драйвера tap0901 (тихий режим)..."))
	_, _, _ = win.RunHidden(exePath, "/S")
	// верифікація
	if _, err := regOpenKey(regHKLM, `SYSTEM\CurrentControlSet\Services\tap0901`, regKeyRead); err == nil {
		logf(i18n.Tr("Драйвер tap0901 (tap-windows6) встановлено."))
		return true
	}
	logf(i18n.Tr("Не вдалося встановити драйвер tap0901: код ") + fmt.Sprint(-1))
	return false
}

func runTool(name string, args ...string) (string, int, error) {
	return win.RunHidden(name, args...)
}

// ─── Management Interface ──────────────────────────────────────────────────

func mgmtConnect(s *session, retries int, delay time.Duration) (net.Conn, error) {
	var lastErr error
	for i := 0; i < retries; i++ {
		if isCancelled(s) {
			return nil, errCancelled
		}
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(mgmtHost, fmt.Sprint(mgmtPort)), 2*time.Second)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		time.Sleep(delay)
	}
	if isCancelled(s) {
		return nil, errCancelled
	}
	return nil, fmt.Errorf("Не вдалося підключитися до Management Interface %s:%d: %v", mgmtHost, mgmtPort, lastErr)
}

var errCancelled = fmt.Errorf("Підключення відмінено користувачем")

func mgmtReadUntilPrompt(conn net.Conn) string {
	var lines []string
	buf := make([]byte, 0, 4096)
	deadline := time.Now().Add(3 * time.Second)
	conn.SetReadDeadline(deadline)
	tmp := make([]byte, 1024)
	for {
		n, err := conn.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			// шукаємо prompt '>' або END/SUCCESS/ERROR
			text := string(buf)
			if strings.Contains(text, "\n>") || strings.HasSuffix(text, ">") ||
				strings.Contains(text, "END") || strings.Contains(text, "SUCCESS") ||
				strings.Contains(text, "ERROR") {
				break
			}
		}
		if err != nil {
			break
		}
		if time.Now().After(deadline) {
			break
		}
	}
	lines = append(lines, string(buf))
	return strings.Join(lines, "\n")
}

func mgmtSend(conn net.Conn, cmd string) error {
	conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	_, err := conn.Write([]byte(cmd + "\n"))
	return err
}

// mgmtListenLoop — безперервно читає повідомлення з management interface.
func mgmtListenLoop(s *session, conn net.Conn) {
	logf("[DEBUG] mgmtListenLoop START")
	lastState := ""
	var waitStart time.Time
	var handledDisconnect bool
	defer func() {
		logf("[DEBUG] mgmtListenLoop EXIT")
		conn.Close()
		globalMu.Lock()
		isCurrent := curSession == s
		globalMu.Unlock()
		if isCurrent && !s.disconnecting && !handledDisconnect {
			if onDisconnectedCb != nil {
				onDisconnectedCb("DISCONNECTED")
			}
		}
		if isCurrent {
			globalMu.Lock()
			s.lastState = "DISCONNECTED"
			globalMu.Unlock()
		}
	}()
	buf := make([]byte, 0, 8192)
	tmp := make([]byte, 4096)
	for {
		if s.disconnecting {
			return
		}
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, err := conn.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			for {
				idx := bytes.IndexByte(buf, '\n')
				if idx < 0 {
					break
				}
				line := strings.TrimRight(string(buf[:idx]), "\r")
				buf = buf[idx+1:]
				if line == "" {
					continue
				}
				logf("[VPN] " + line)

				// Фатальні помилки
				if strings.HasPrefix(line, ">FATAL:") {
					fatal := line
					if i := strings.Index(line, ":"); i >= 0 {
						fatal = line[i+1:]
					}
					friendly := fatal
					low := strings.ToLower(fatal)
					if strings.Contains(low, "wintun adapters") {
						friendly = "Усі адаптери Wintun зайняті або потрібні права адміністратора для створення мережевого інтерфейсу."
					} else if strings.Contains(low, "tap-windows6 adapters") {
						friendly = "Мережевий адаптер tap-windows6 не знайдено або зайнятий."
					}
					logf("[FATAL] " + friendly)
					if !s.disconnecting && onDisconnectedCb != nil {
						handledDisconnect = true
						onDisconnectedCb("Помилка: " + friendly)
					}
					return
				}

				// Стани підключення
				if strings.Contains(line, ">STATE:") {
					parts := strings.Split(line, ",")
					if len(parts) >= 2 {
						state := parts[1]
						logf("[DEBUG] parsed state=" + state + " line=" + line)
						lastState = state
						if !strings.HasPrefix(line, ">LOG:") {
							globalMu.Lock()
							s.lastState = state
							globalMu.Unlock()
						}
						switch state {
						case "WAIT":
							waitStart = time.Now()
						case "AUTH":
							logf(i18n.Tr("Сервер відгукнувся! Перевірка авторизації (AUTH)..."))
						case "GET_CONFIG":
							logf(i18n.Tr("Отримання мережевих параметрів від сервера (GET_CONFIG)..."))
						case "ASSIGN_IP":
							logf(i18n.Tr("Налаштування віртуального Wintun адаптера та маршрутів..."))
						case "CONNECTED":
							logf("[DEBUG] STATE CONNECTED received")
							// DNS прописуємо одразу після підключення адаптера
							go applySessionDNS(s)
							if !s.disconnecting && onConnectedCb != nil {
								logf("[DEBUG] calling onConnectedCb")
								onConnectedCb()
								logf("[DEBUG] onConnectedCb returned")
							} else {
								logf("[DEBUG] skipping onConnectedCb: disconnecting=" + fmt.Sprint(s.disconnecting) + " cbNil=" + fmt.Sprint(onConnectedCb == nil))
							}
						case "DISCONNECTED", "EXITING", "RECONNECTING":
							if !s.disconnecting && onDisconnectedCb != nil {
								handledDisconnect = true
								reason := ""
								if len(parts) > 2 {
									reason = parts[2]
								}
								code := state
								if reason != "" {
									code = state + "," + reason
								}
								if state == "RECONNECTING" {
									logf(i18n.Tr("Втрата зв'язку з сервером (RECONNECTING)..."))
								}
								onDisconnectedCb(code)
							}
						}
					}
				}

				// Застереження про зависання у WAIT
				if lastState == "WAIT" && !waitStart.IsZero() {
					elapsed := time.Since(waitStart).Seconds()
					if elapsed >= 10 && elapsed <= 14 {
						logf(i18n.Tr("Сервер не відповідає на запити TLS handshake. Вузол може бути перевантажений або вимкнений..."))
					}
				}
			}
		}
		if err != nil {
			logf("[DEBUG] mgmtListenLoop read err: " + err.Error())
			// EOF або таймаут читання — продовжуємо цикл на таймауті, виходимо на EOF
			var ne net.Error
			if ok := errors.As(err, &ne); ok && ne.Timeout() {
				continue
			}
			return
		}
		if s.disconnecting {
			return
		}
	}
}

// ─── Підключення ───────────────────────────────────────────────────────────

// Connect — підключається до VPN через OpenVPN Management Interface.
func Connect(configData, protocol string, dnsLeakProtection, blockAds bool, dnsResolver, customDNS string) error {
	Disconnect()

	// Чекаємо смерть старих процесів ДО роботи з адаптерами
	WaitOpenVPNExited(6 * time.Second)

	globalMu.Lock()
	curSession = &session{lastState: "NONE", counter: curSession.counter + 1}
	s := curSession
	globalMu.Unlock()

	if win.IsAdmin() {
		logf(i18n.Tr("Очищення фантомних Wintun адаптерів..."))
		CleanupWintunGhosts()
		EnsureInteractiveService()
	} else {
		logf(i18n.Tr("Увага: Ziper VPN запущено без прав адміністратора. Створення Wintun адаптера вимагає підвищених привілеїв Windows."))
	}

	// 1. openvpn.exe
	logf(i18n.Tr("Ініціалізація OpenVPN рушія..."))
	openvpnCmd := EnsureOpenVPN()
	if isCancelled(s) {
		return errCancelled
	}

	// 2. Конфіг у тимчасовий файл
	configPath := filepath.Join(os.TempDir(), "ziper_vpn.ovpn")
	if err := os.WriteFile(configPath, []byte(configData), 0o644); err != nil {
		return err
	}

	// 3. Команда запуску
	cmd := exec.Command(openvpnCmd,
		"--config", configPath,
		"--management", mgmtHost, fmt.Sprint(mgmtPort),
		"--management-query-passwords",
		"--management-hold",
		"--connect-retry-max", "1",
		"--hand-window", "30",
		"--verb", "3",
	)
	cmd.Dir = filepath.Dir(openvpnCmd)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}

	// Вибір драйвера
	wintunDLL := filepath.Join(filepath.Dir(openvpnCmd), "wintun.dll")
	driverName := ""
	if win.IsAdmin() {
		logf(i18n.Trf("IsAdmin=true. wintunDLL={p}", map[string]string{"p": wintunDLL}))
		if _, err := os.Stat(wintunDLL); err == nil {
			logf("wintun.dll знайдено, викликаю EnsureNetworkAdapter...")
			driverName = EnsureNetworkAdapter()
		} else {
			logf("wintun.dll НЕ знайдено: " + err.Error())
		}
	} else {
		logf("IsAdmin=false — пропускаю Wintun.")
	}
	if driverName != "" {
		if driverName == "tap-windows6" {
			logf("Створюю tap-windows6 адаптер через tapctl...")
			TapctlCreateAdapter("tap", logf)
		}
		cmd.Args = append(cmd.Args, "--windows-driver", driverName)
		logf(i18n.Tr("Використовую ") + driverName + i18n.Tr(" драйвер."))
	} else {
		cmd.Args = append(cmd.Args, "--windows-driver", "tap-windows6")
		logf(i18n.Tr("Адаптер підготувати не вдалося — віддаю вибір драйвера OpenVPN."))
	}

	// Протокол: якщо профіль сам задає proto — пріоритет має профіль
	if protocol != "" && strings.EqualFold(protocol, "UDP") || strings.EqualFold(protocol, "TCP") {
		profileProto := ""
		for _, raw := range strings.Split(configData, "\n") {
			line := strings.TrimSpace(strings.ToLower(raw))
			if strings.HasPrefix(line, "proto ") {
				word := strings.Fields(line)[1]
				if strings.HasPrefix(word, "udp") {
					profileProto = "udp"
				} else if strings.HasPrefix(word, "tcp") {
					profileProto = "tcp"
				}
				break
			}
		}
		if profileProto != "" {
			if profileProto != strings.ToLower(protocol) {
				logf(i18n.Trf("Профіль сам задає протокол {p} — налаштування '{p2}' для цього профілю проігноровано.",
					map[string]string{"p": strings.ToUpper(profileProto), "p2": strings.ToUpper(protocol)}))
			}
		} else {
			cmd.Args = append(cmd.Args, "--proto", strings.ToLower(protocol))
		}
	}

	// Захист від витоку DNS
	if dnsLeakProtection {
		cmd.Args = append(cmd.Args, "--block-outside-dns")
	}

	// DNS сервери (пріоритет: власні DNS → блокування реклами → резолвер)
	dnsServers, dnsSource := DNSServers(blockAds, dnsResolver, customDNS)
	switch dnsSource {
	case "custom":
		logf(i18n.Tr("Використовується власний DNS: ") + strings.Join(dnsServers, ", "))
		if blockAds {
			logf(i18n.Tr("Блокування реклами пропущено — пріоритет має власний DNS."))
		}
	case "blockads":
		// Блокування реклами/трекерів/загроз на рівні DNS через AdGuard DNS.
		logf(i18n.Tr("🚫 Блокування реклами та трекерів: використовую AdGuard DNS (94.140.14.14)."))
	}
	for _, dns := range dnsServers {
		cmd.Args = append(cmd.Args, "--dhcp-option", "DNS", dns)
	}

	// Запам'ятовуємо DNS сесії — після CONNECTED прописуємо його
	// на адаптері тунелю явно (dhcp-option з Wintun працює не завжди).
	s.mu.Lock()
	s.dnsServers = dnsServers
	s.mu.Unlock()

	logf(i18n.Trf("Запуск тунелю OpenVPN (порт management: {port})...",
		map[string]string{"port": fmt.Sprint(mgmtPort)}))

	// 4. Запуск процесу
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	s.mu.Lock()
	s.process = cmd
	s.procHandle = cmd.Process
	s.mu.Unlock()

	// Читаємо stdout у фоні
	go func() {
		scanner := make([]byte, 4096)
		for {
			n, err := stdoutPipe.Read(scanner)
			if n > 0 {
				logf("[OVPN] " + strings.TrimRight(string(scanner[:n]), "\r\n"))
			}
			if err != nil {
				break
			}
			if s.disconnecting {
				return
			}
		}
		rc := cmd.Wait()
		if rc != nil && !s.disconnecting {
			if onDisconnectedCb != nil {
				onDisconnectedCb(fmt.Sprintf("OpenVPN завершився з кодом %d", cmd.ProcessState.ExitCode()))
			}
		}
	}()

	// 5. Management Interface
	logf(i18n.Tr("Очікую Management Interface..."))
	conn, err := mgmtConnect(s, 30, 500*time.Millisecond)
	if err != nil {
		if isCancelled(s) {
			Disconnect()
			return errCancelled
		}
		return fmt.Errorf("Помилка сокету керування: %v", err)
	}
	s.mu.Lock()
	s.mgmtConn = conn
	s.mu.Unlock()

	mgmtReadUntilPrompt(conn)
	_ = mgmtSend(conn, "state on")
	_ = mgmtSend(conn, "log on")
	_ = mgmtSend(conn, "hold release")
	if isCancelled(s) {
		Disconnect()
		return errCancelled
	}

	logf(i18n.Tr("Сесію ініційовано. Встановлюю з'єднання..."))
	go mgmtListenLoop(s, conn)

	return nil
}

// Disconnect — безпечне відключення через Management Interface.
func Disconnect() {
	globalMu.Lock()
	s := curSession
	globalMu.Unlock()
	s.mu.Lock()
	s.disconnecting = true
	s.lastState = "DISCONNECTED"
	s.mu.Unlock()

	// SIGTERM через management — грасіозний вихід
	if s.mgmtConn != nil {
		_ = mgmtSend(s.mgmtConn, "signal SIGTERM")
		time.Sleep(100 * time.Millisecond)
		_ = s.mgmtConn.Close()
		s.mu.Lock()
		s.mgmtConn = nil
		s.mu.Unlock()
	}

	// Очікуємо грасіозний вихід (4 с), потім terminate/kill
	if s.procHandle != nil {
		deadline := time.Now().Add(4 * time.Second)
		for time.Now().Before(deadline) {
			if !win.IsProcessRunning("openvpn.exe") {
				break
			}
			time.Sleep(150 * time.Millisecond)
		}
		if win.IsProcessRunning("openvpn.exe") {
			_ = s.procHandle.Kill()
			time.Sleep(200 * time.Millisecond)
		}
		s.mu.Lock()
		s.procHandle = nil
		s.process = nil
		s.mu.Unlock()
	}

	KillLingeringOpenVPN()
	ReleaseWintunAdapter()
	// адаптер (наприклад tapctl) міг лишитися в системі — скидаємо його DNS
	resetAdapterDNS()
}
