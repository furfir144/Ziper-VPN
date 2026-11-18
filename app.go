package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"ziper-vpn-go/internal/i18n"
	"ziper-vpn-go/internal/settings"
	"ziper-vpn-go/internal/tor"
	"ziper-vpn-go/internal/tray"
	"ziper-vpn-go/internal/win"
	"ziper-vpn-go/internal/vpn"
	"ziper-vpn-go/internal/wireguard"
)

// App — об'єкт, який біндиться у фронтенд.
type App struct {
	ctx context.Context

	mu          sync.Mutex
	vpnStatus   string // DISCONNECTED | CONNECTING | CONNECTED
	torActive   bool
	logs        []string
	torLogs     []string
	wgLogs      []string
	connectedAt time.Time
	failoverMu  sync.Mutex
	failoverIdx int
	lastAttempt time.Time
}

func NewApp() *App { return &App{vpnStatus: "DISCONNECTED"} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	settings.Init()

	vpn.SetCallbacks(a.log, a.onVPNConnected, a.onVPNDisconnected)
	tor.SetLogCallback(a.torLog)
	wireguard.SetCallbacks(a.wgLog, a.onWGConnected, a.onWGDisconnected, a.onWGStats)

	// відновлюємо мову з налаштувань
	s := settings.Load()
	i18n.SetLanguage(s.Language)

	// ініціалізуємо трей-стан
	if tr := tray.GetInstance(); tr != nil {
		tr.UpdateState("DISCONNECTED")
	}

	// автопідключення при старті
	if s.Autoconnect {
		a.log("⚡ Автопідключення VPN активовано...")
		go a.connectVPN("")
	}

	// Repair: якщо Kill Switch вимкнено в налаштуваннях, але політика
	// брандмауера залишилась від минулого запуску (наприклад після збою) —
	// тихо відновлюємо стандартну політику та прибираємо правила-винятки.
	if !s.KillSwitch {
		_ = win.DisableKillSwitch()
	}

	// Застосовуємо обраний DNS одразу — фільтрація (Quad9 / AdGuard /
	// власні адреси) працює навіть без VPN.
	go a.applySystemDNS()
}

// showWindow — показує головне вікно з трея.
func (a *App) showWindow() {
	if a.ctx != nil {
		runtime.WindowShow(a.ctx)
		runtime.WindowUnminimise(a.ctx)
	}
}

// hideWindow — ховає вікно (close → tray).
func (a *App) hideWindow() {
	if a.ctx != nil {
		runtime.WindowHide(a.ctx)
	}
}

// quitApp — повне завершення програми (з трея).
func (a *App) quitApp() {
	a.DisconnectVPN()
	wireguard.Disconnect()
	tor.Stop()
	// знімаємо Kill Switch, щоб після виходу інтернет не лишився заблокованим
	_ = win.DisableKillSwitch()
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
	os.Exit(0)
}

// trayUpload — завантаження .ovpn з трея (через діалог файлів).
func (a *App) trayUpload() {
	a.showWindow()
	a.SaveProfileDialog()
}

// ─── Логи та події ─────────────────────────────────────────────────────────

func (a *App) log(msg string) {
	a.mu.Lock()
	a.logs = append(a.logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg))
	if len(a.logs) > 2000 {
		a.logs = a.logs[len(a.logs)-2000:]
	}
	a.mu.Unlock()
	runtime.EventsEmit(a.ctx, "vpn:log", msg)
}

func (a *App) torLog(msg string) {
	a.mu.Lock()
	a.torLogs = append(a.torLogs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg))
	if len(a.torLogs) > 2000 {
		a.torLogs = a.torLogs[len(a.torLogs)-2000:]
	}
	a.mu.Unlock()
	runtime.EventsEmit(a.ctx, "tor:log", msg)
}

func (a *App) setStatus(status string) {
	a.mu.Lock()
	a.vpnStatus = status
	a.mu.Unlock()
	if status == "CONNECTED" {
		a.mu.Lock()
		a.connectedAt = time.Now()
		a.mu.Unlock()
	}
	runtime.EventsEmit(a.ctx, "vpn:state", status)
}

func (a *App) onVPNConnected() {
	a.setStatus("CONNECTED")
	a.log("VPN тунель успішно встановлено! ✓")
	info := vpn.FetchCurrentIPInfo()
	runtime.EventsEmit(a.ctx, "vpn:ipinfo", info)
	a.notifyUser(i18n.Trf("VPN підключено через {country}!", map[string]string{"country": info.Country}), "positive")
	// трей
	if tr := tray.GetInstance(); tr != nil {
		tr.UpdateState("CONNECTED")
	}
}

func (a *App) onVPNDisconnected(reason string) {
	a.setStatus("DISCONNECTED")
	a.log("VPN відключено (" + reason + ")")
	if tr := tray.GetInstance(); tr != nil {
		tr.UpdateState("DISCONNECTED")
	}
}

func (a *App) notifyUser(message, kind string) {
	// Перемикач «Спливаючі сповіщення»: вимкнені — показуємо лише помилки.
	if s := settings.Load(); !s.Notifications && kind != "negative" {
		a.log("🔕 Сповіщення вимкнено: " + message)
		return
	}
	runtime.EventsEmit(a.ctx, "vpn:notification", map[string]string{"message": message, "kind": kind})
}

// ─── Налаштування ──────────────────────────────────────────────────────────

// GetSettings повертає поточні налаштування.
func (a *App) GetSettings() settings.Settings {
	return settings.Load()
}

// SaveSettings зберігає налаштування.
func (a *App) SaveSettings(s settings.Settings) settings.Settings {
	settings.Save(s)
	if s.Language != "" {
		i18n.SetLanguage(s.Language)
	}
	return settings.Load()
}

// UpdateSetting — оновлює одне налаштування.
func (a *App) UpdateSetting(key, value string) {
	s := settings.Load()
	switch key {
	case "language":
		s.Language = value
		i18n.SetLanguage(value)
	case "autoconnect":
		s.Autoconnect = value == "true"
	case "auto_server":
		s.AutoServer = value == "true"
	case "selected_country":
		s.SelectedCountry = value
	case "selected_server_ip":
		s.SelectedServerIP = value
	case "selected_server_name":
		s.SelectedServerName = value
	case "protocol":
		s.Protocol = value
	case "dns_leak_protection":
		s.DNSLeakProtection = value == "true"
	case "dns_resolver":
		s.DNSResolver = value
		// обраний резолвер застосовується системно одразу (навіть без VPN)
		go a.applySystemDNS()
	case "custom_dns":
		s.CustomDNS = value
		go a.applySystemDNS()
	case "block_ads":
		s.BlockAds = value == "true"
		go a.applySystemDNS()
	case "kill_switch":
		s.KillSwitch = value == "true"
		if !s.KillSwitch {
			// вимкнули тумблер — одразу відновлюємо політику брандмауера
			if err := win.DisableKillSwitch(); err == nil && win.IsAdmin() {
				a.log("🛡 Kill Switch вимкнено — стандартну політику брандмауера відновлено.")
			}
		}
	case "tor_enabled":
		s.TorEnabled = value == "true"
	case "tor_socks_port":
		var p int
		fmt.Sscanf(value, "%d", &p)
		s.TorSocksPort = p
	case "notifications":
		s.Notifications = value == "true"
	case "tor_bridges_enabled":
		s.TorBridgesEnabled = value == "true"
	case "tor_bridges":
		s.TorBridges = value
	case "custom_ovpn_name":
		s.CustomOvpnName = value
	case "wg_profile":
		s.WireGuardProfile = value
	}
	settings.Save(s)
}

// applySystemDNS — застосовує обраний DNS системно (працює без VPN):
// «Авто» → повертає DNS провайдера, інакше прописує обрані сервери.
func (a *App) applySystemDNS() {
	s := settings.Load()
	servers, _ := vpn.DNSServers(s.BlockAds, s.DNSResolver, s.CustomDNS)
	if len(servers) == 0 {
		if err := vpn.SystemDNSReset(); err != nil {
			a.log("[WARN] Автоматичний DNS не відновлено: " + err.Error())
		} else {
			a.log("DNS відновлено на автоматичний (провайдер).")
		}
		return
	}
	if err := vpn.SystemDNSApply(servers); err != nil {
		a.log("[WARN] Системний DNS не застосовано: " + err.Error())
	}
}

// ─── VPN ───────────────────────────────────────────────────────────────────

// GetVPNState — поточний статус VPN.
func (a *App) GetVPNState() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.vpnStatus
}

// GetLastMgmtState — джерело істини для самозлагодження UI.
func (a *App) GetLastMgmtState() string {
	return vpn.LastMgmtState()
}

// GetVPNLogs — логи тунелю.
func (a *App) GetVPNLogs() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]string, len(a.logs))
	copy(out, a.logs)
	return out
}

// ClearVPNLogs — очищає логи.
func (a *App) ClearVPNLogs() {
	a.mu.Lock()
	a.logs = nil
	a.mu.Unlock()
}

// IsAdmin — чи запущено з правами адміністратора.
func (a *App) IsAdmin() bool { return win.IsAdmin() }

// Elevate — запитує UAC і перезапускає процес.
func (a *App) Elevate() bool {
	return win.Elevate()
}

// ToggleVPN — підключення/відключення VPN.
func (a *App) ToggleVPN() {
	a.mu.Lock()
	status := a.vpnStatus
	a.lastAttempt = time.Now()
	a.mu.Unlock()

	if status == "DISCONNECTED" {
		go a.connectVPN("")
	} else {
		go a.DisconnectVPN()
	}
}

// DisconnectVPN — відключення VPN.
func (a *App) DisconnectVPN() {
	a.log("Ініціація відключення VPN...")
	go func() {
		vpn.Disconnect()
		a.setStatus("DISCONNECTED")
		a.log("VPN-тунель повністю зупинено (process openvpn завершено, маршрути знято).")
		a.notifyUser(i18n.Tr("VPN успішно відключено."), "positive")
		if tr := tray.GetInstance(); tr != nil {
			tr.UpdateState("DISCONNECTED")
		}
	}()
}

// ConnectServer — підключення до конкретного сервера.
func (a *App) ConnectServer(serverIP string) {
	a.mu.Lock()
	status := a.vpnStatus
	a.mu.Unlock()
	if status == "DISCONNECTED" {
		go a.connectVPN(serverIP)
	}
}

// connectVPN — основна логіка підключення (з авто-підбором сервера).
func (a *App) connectVPN(serverIP string) {
	// захист від швидкого повторного кліку
	a.mu.Lock()
	if a.vpnStatus == "CONNECTING" {
		a.mu.Unlock()
		return
	}
	a.mu.Unlock()

	// Деактивуємо Tor якщо активний (як у Python-версії)
	if tor.IsActive() {
		tor.Stop()
	}

	// WireGuard і OpenVPN — взаємовиключні режими тунелю
	if wireguard.IsActive() {
		wireguard.Disconnect()
		time.Sleep(300 * time.Millisecond)
	}

	s := settings.Load()
	a.setStatus("CONNECTING")
	a.log("=== Запуск захищеної сесії Ziper VPN ===")

	// Вибір конфіг: власний профіль → обраний сервер → авто-підбір
	configData := ""
	serverDesc := ""

	if s.CustomOvpnName != "" {
		content, err := settings.ReadCustomProfile(s.CustomOvpnName)
		if err != nil {
			a.log("[ERROR] Не вдалося прочитати збережений .ovpn: " + err.Error())
			a.notifyUser(i18n.Tr("Помилка: Не вдалося підключитися."), "negative")
			a.setStatus("DISCONNECTED")
			return
		}
		configData = content
		a.log("Використовую вбудований власний профіль: " + s.CustomOvpnName)
	} else {
		// Отримуємо список серверів
		a.log("Режим: Автоматичний підбір сервера (VPN Gate)...")
		servers := vpn.FetchServers()
		if len(servers) == 0 {
			a.log("[ERROR] Не вдалося завантажити список серверів з мережі!")
			a.notifyUser(i18n.Tr("Помилка зв'язку з VPN Gate. Перевірте з'єднання з мережею."), "negative")
			a.setStatus("DISCONNECTED")
			return
		}
		a.log(i18n.Trf("Оновлено: знайдено {count} активних серверів",
			map[string]string{"count": fmt.Sprint(len(servers))}))

		// вибір сервера: явний IP → обраний у налаштуваннях → авто (перший у списку)
		var selected *vpn.Server
		targetIP := serverIP
		if targetIP == "" && !s.AutoServer {
			targetIP = s.SelectedServerIP
		}
		if targetIP != "" {
			for i := range servers {
				if servers[i].IP == targetIP {
					selected = &servers[i]
					break
				}
			}
			if selected == nil {
				a.log(i18n.Trf("Обраний сервер {ip} недоступний у списку — переходимо на авто-підбір.",
					map[string]string{"ip": targetIP}))
			}
		}
		if selected == nil {
			selected = &servers[0]
		}
		serverDesc = fmt.Sprintf("%s %s (%s)", selected.HostName, selected.CountryLong, selected.IP)
		a.log(i18n.Trf("Обрано сервер: {desc} | Ping: {ping} ms | Швидкість: {speed} Mbps",
			map[string]string{"desc": serverDesc, "ping": fmt.Sprint(selected.Ping), "speed": fmt.Sprint(selected.Speed / 1000000)}))
		configData = vpn.DecodeConfig(selected.OpenVPNConfigB64)
	}

	if configData == "" {
		a.log("[ERROR] Порожній конфіг, підключення неможливе.")
		a.notifyUser(i18n.Tr("Помилка: Не вдалося підключитися."), "negative")
		a.setStatus("DISCONNECTED")
		return
	}

	a.log("Підготовка конфігурації та вибір сервера...")

	// Протокол
	protocol := s.Protocol
	if protocol == "" {
		protocol = "Auto"
	}

	// Kill Switch: перед підключенням блокуємо весь трафік поза тунелем
	// (залишається активним і після відключення, поки тумблер увімкнений).
	if s.KillSwitch {
		if err := win.EnableKillSwitch(vpn.FindOpenVPN()); err != nil {
			a.log("🛡 [WARN] Kill Switch не активовано: " + err.Error())
		} else {
			a.log("🛡 Kill Switch активовано: трафік поза тунелем заблоковано брандмауером.")
		}
	}

	err := vpn.Connect(configData, protocol, s.DNSLeakProtection, s.BlockAds, s.DNSResolver, s.CustomDNS)
	if err != nil {
		a.log("[ERROR] Не вдалося підключитися: " + err.Error())
		a.notifyUser(i18n.Tr("Помилка: {ex}", map[string]string{"ex": err.Error()}), "negative")
		a.setStatus("DISCONNECTED")
	}
}

// ─── Failover ──────────────────────────────────────────────────────────────

// StartFailover — перебирає сервери списку по колу.
func (a *App) StartFailover() {
	a.failoverMu.Lock()
	defer a.failoverMu.Unlock()
	if a.vpnStatus == "CONNECTED" {
		return
	}
	servers := vpn.FetchServers()
	if len(servers) == 0 {
		a.notifyUser(i18n.Tr("Список серверів порожній."), "negative")
		return
	}
	for i := 0; i < len(servers) && a.vpnStatus != "CONNECTED"; i++ {
		if a.vpnStatus == "DISCONNECTED" && a.vpnStatus != "CONNECTING" {
			break
		}
		srv := servers[i]
		desc := fmt.Sprintf("%s %s (%s)", srv.HostName, srv.CountryLong, srv.IP)
		a.log(i18n.Trf("Failover → {desc}", map[string]string{"desc": desc}))
		a.mu.Lock()
		a.vpnStatus = "CONNECTING"
		a.mu.Unlock()
		cfg := vpn.DecodeConfig(srv.OpenVPNConfigB64)
		if cfg == "" {
			a.log("[WARN] Сервер " + srv.IP + " відсутній client, пробуємо наступний...")
			continue
		}
		s := settings.Load()
		if s.KillSwitch {
			if err := win.EnableKillSwitch(vpn.FindOpenVPN()); err != nil {
				a.log("🛡 [WARN] Kill Switch не активовано: " + err.Error())
			}
		}
		if err := vpn.Connect(cfg, s.Protocol, s.DNSLeakProtection, s.BlockAds, s.DNSResolver, s.CustomDNS); err != nil {
			a.log("[ERROR] Failover помилка: " + err.Error())
			continue
		}
		// очікуємо результату до 30 с
		for j := 0; j < 60; j++ {
			time.Sleep(500 * time.Millisecond)
			a.mu.Lock()
			st := a.vpnStatus
			a.mu.Unlock()
			if st == "CONNECTED" {
				return
			}
		}
		vpn.Disconnect()
	}
	a.setStatus("DISCONNECTED")
	a.notifyUser(i18n.Tr("Вичерпано всі сервери зі списку."), "negative")
}

// ─── Сервери ───────────────────────────────────────────────────────────────

// GetServers — повертає список серверів (з кешу або з мережі).
func (a *App) GetServers() []vpn.Server {
	return vpn.LoadCachedServers()
}

// RefreshServers — завантажує список серверів з мережі.
func (a *App) RefreshServers() []vpn.Server {
	go func() {
		servers := vpn.FetchServers()
		if len(servers) == 0 {
			a.notifyUser(i18n.Tr("Не вдалося завантажити список серверів!"), "negative")
			runtime.EventsEmit(a.ctx, "servers:refreshed", []vpn.Server{})
			return
		}
		a.notifyUser(i18n.Trf("Оновлено: знайдено {count} активних серверів",
			map[string]string{"count": fmt.Sprint(len(servers))}), "positive")
		runtime.EventsEmit(a.ctx, "servers:refreshed", servers)
	}()
	return vpn.LoadCachedServers()
}

// ─── IP ────────────────────────────────────────────────────────────────────

// GetIPInfo — поточний IP та гео.
func (a *App) GetIPInfo() vpn.IPInfo {
	return vpn.FetchCurrentIPInfo()
}

// ─── Tor ───────────────────────────────────────────────────────────────────

// TorState — стан Tor для біндингу.
type TorState struct {
	Active    bool `json:"active"`
	Port      int  `json:"port"`
	Available bool `json:"available"`
}

// GetTorState — чи активний Tor та порт.
func (a *App) GetTorState() TorState {
	return TorState{
		Active:    tor.IsActive(),
		Port:      tor.GetPort(),
		Available: tor.IsAvailable(),
	}
}

// ToggleTor — запускає/зупиняє Tor SOCKS5.
func (a *App) ToggleTor() {
	a.mu.Lock()
	active := a.torActive
	a.mu.Unlock()
	if active {
		go func() {
			tor.Stop()
			a.mu.Lock()
			a.torActive = false
			a.mu.Unlock()
			runtime.EventsEmit(a.ctx, "tor:state", a.GetTorState())
			a.notifyUser(i18n.Tr("Службу Tor зупинено"), "info")
			if tr := tray.GetInstance(); tr != nil {
				tr.UpdateTorState(false)
			}
		}()
	} else {
		go func() {
			s := settings.Load()
			// Власні мости Tor (якщо увімкнені) — беруться при кожному запуску
			var bridges []string
			if s.TorBridgesEnabled {
				bridges = tor.ParseBridgeLines(s.TorBridges)
				if len(bridges) == 0 {
					a.log("[TOR WARN] Власні мости увімкнено, але список порожній — Tor запускається без мостів.")
				}
			}
			ok := tor.Start(s.TorSocksPort, bridges)
			a.mu.Lock()
			a.torActive = ok
			a.mu.Unlock()
			runtime.EventsEmit(a.ctx, "tor:state", a.GetTorState())
			if ok {
				a.notifyUser(i18n.Trf("Tor SOCKS5 успішно активовано на порту {port}!",
					map[string]string{"port": fmt.Sprint(s.TorSocksPort)}), "positive")
				if tr := tray.GetInstance(); tr != nil {
					tr.UpdateTorState(true)
				}
			} else {
				a.notifyUser(i18n.Tr("Не вдалося запустити Tor"), "negative")
			}
		}()
	}
}

// GetTorLogs — логи Tor.
func (a *App) GetTorLogs() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]string, len(a.torLogs))
	copy(out, a.torLogs)
	return out
}

// ClearTorLogs — очищає логи Tor.
func (a *App) ClearTorLogs() {
	a.mu.Lock()
	a.torLogs = nil
	a.mu.Unlock()
}

// ─── WireGuard ─────────────────────────────────────────────────────────────

// wgLog — лог тунелю WireGuard (окремий буфер і окрема подія UI).
func (a *App) wgLog(msg string) {
	a.mu.Lock()
	a.wgLogs = append(a.wgLogs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg))
	if len(a.wgLogs) > 2000 {
		a.wgLogs = a.wgLogs[len(a.wgLogs)-2000:]
	}
	a.mu.Unlock()
	runtime.EventsEmit(a.ctx, "wg:log", msg)
}

// onWGConnected — тунель WireGuard піднято (хендшейк успішний).
func (a *App) onWGConnected() {
	a.wgLog("WireGuard тунель успішно встановлено! ✓")
	info := vpn.FetchCurrentIPInfo()
	runtime.EventsEmit(a.ctx, "vpn:ipinfo", info)
	a.notifyUser(i18n.Trf("WireGuard підключено через {country}!", map[string]string{"country": info.Country}), "positive")
	if tr := tray.GetInstance(); tr != nil {
		tr.UpdateState("CONNECTED")
	}
}

// onWGDisconnected — тунель WireGuard обірвано/закрито.
func (a *App) onWGDisconnected(reason string) {
	a.wgLog("WireGuard відключено (" + reason + ")")
	if !strings.Contains(reason, "користувачем") {
		a.notifyUser(i18n.Trf("WireGuard відключено: {ex}", map[string]string{"ex": reason}), "negative")
	} else {
		a.notifyUser(i18n.Tr("WireGuard успішно відключено."), "positive")
	}
	// трей показує DISCONNECTED лише тоді, коли і OpenVPN не активний
	a.mu.Lock()
	st := a.vpnStatus
	a.mu.Unlock()
	if tr := tray.GetInstance(); tr != nil && st == "DISCONNECTED" {
		tr.UpdateState("DISCONNECTED")
	}
}

// onWGStats — періодична статистика тунелю у UI.
func (a *App) onWGStats(st wireguard.State) {
	runtime.EventsEmit(a.ctx, "wg:stats", st)
}

// GetWireGuardState — поточний стан WireGuard тунелю.
func (a *App) GetWireGuardState() wireguard.State {
	return wireguard.CurrentState()
}

// ConnectWireGuard — запускає WireGuard тунель з активного профілю.
func (a *App) ConnectWireGuard() {
	st := wireguard.CurrentState()
	if st.Status != "DISCONNECTED" {
		return
	}
	s := settings.Load()
	name := s.WireGuardProfile
	if name == "" {
		a.wgLog("[WARN] " + i18n.Tr("Профіль WireGuard не обрано."))
		a.notifyUser(i18n.Tr("Спочатку оберіть .conf профіль на вкладці WireGuard."), "negative")
		return
	}
	content, err := settings.ReadWGProfile(name)
	if err != nil {
		a.wgLog("[ERROR] Не вдалося прочитати профіль " + name + ": " + err.Error())
		a.notifyUser(i18n.Tr("Помилка: Не вдалося підключитися."), "negative")
		return
	}

	// WireGuard — єдиний активний тунель: зупиняємо інші режими
	if tor.IsActive() {
		tor.Stop()
	}
	a.mu.Lock()
	vpnStatus := a.vpnStatus
	a.mu.Unlock()
	if vpnStatus != "DISCONNECTED" {
		a.log("WireGuard запускається — зупиняю OpenVPN тунель...")
		vpn.Disconnect()
		a.setStatus("DISCONNECTED")
		time.Sleep(600 * time.Millisecond)
	}

	// DNS з налаштувань застосунку — якщо профіль не задає власний
	fallbackDNS, _ := vpn.DNSServers(s.BlockAds, s.DNSResolver, s.CustomDNS)
	go func() {
		if err := wireguard.Connect(name, content, fallbackDNS); err != nil {
			a.wgLog("[ERROR] " + err.Error())
		}
	}()
}

// DisconnectWireGuard — розриває WireGuard тунель.
func (a *App) DisconnectWireGuard() {
	go wireguard.Disconnect()
}

// ToggleWireGuard — підключення/відключення WireGuard.
func (a *App) ToggleWireGuard() {
	st := wireguard.CurrentState()
	if st.Status == "DISCONNECTED" {
		a.ConnectWireGuard()
	} else {
		a.DisconnectWireGuard()
	}
}

// ListWireGuardProfiles — список збережених .conf профілів.
func (a *App) ListWireGuardProfiles() []string {
	return settings.ListWGProfiles()
}

// ImportWireGuardProfileDialog — діалог імпорту .conf профілю.
func (a *App) ImportWireGuardProfileDialog() string {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: i18n.Tr("Виберіть свій .conf конфіг WireGuard"),
		Filters: []runtime.FileFilter{
			{DisplayName: "WireGuard конфігурації (*.conf)", Pattern: "*.conf"},
		},
	})
	if err != nil || path == "" {
		return ""
	}
	content, err := os.ReadFile(path)
	if err != nil {
		a.wgLog("[ERROR] Імпорт конфіга не вдався: " + err.Error())
		return ""
	}
	// перевірка конфігурації ДО збереження
	if _, err := wireguard.ParseConf(string(content)); err != nil {
		a.wgLog("[ERROR] Некоректний конфіг " + filepath.Base(path) + ": " + err.Error())
		a.notifyUser(i18n.Trf("Некоректний .conf конфіг: {ex}", map[string]string{"ex": err.Error()}), "negative")
		return ""
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	saved := settings.SaveWGProfile(name, string(content))
	if saved != "" {
		a.wgLog("Збережено WireGuard профіль: config/wireguard/" + saved)
		a.notifyUser(i18n.Trf("Завантажено: {name}", map[string]string{"name": saved}), "positive")
	} else {
		a.wgLog("[ERROR] Не вдалося зберегти конфіг " + path)
	}
	return saved
}

// SelectWireGuardProfile — встановлює активний WireGuard профіль.
func (a *App) SelectWireGuardProfile(name string) {
	a.UpdateSetting("wg_profile", name)
	a.wgLog("WireGuard профіль обрано: " + name)
}

// DeselectWireGuardProfile — прибирає активний WireGuard профіль.
func (a *App) DeselectWireGuardProfile(name string) {
	s := settings.Load()
	if s.WireGuardProfile == name {
		a.UpdateSetting("wg_profile", "")
		a.wgLog("WireGuard профіль скинуто: " + name)
	}
}

// DeleteWireGuardProfile — видаляє профіль WireGuard.
func (a *App) DeleteWireGuardProfile(name string) bool {
	ok := settings.DeleteWGProfile(name)
	if ok {
		a.wgLog("Видалено WireGuard профіль: " + name)
		if s := settings.Load(); s.WireGuardProfile == name {
			a.UpdateSetting("wg_profile", "")
		}
	} else {
		a.wgLog("[ERROR] Не вдалося видалити профіль: " + name)
	}
	return ok
}

// WGProfileInfo — короткий опис .conf профілю для UI.
type WGProfileInfo struct {
	OK        bool     `json:"ok"`
	Addresses []string `json:"addresses"`
	DNS       []string `json:"dns"`
	Endpoint  string   `json:"endpoint"`
	Peers     int      `json:"peers"`
	MTU       int      `json:"mtu"`
}

// GetWireGuardProfileInfo — короткий опис .conf профілю для UI.
func (a *App) GetWireGuardProfileInfo(name string) WGProfileInfo {
	out := WGProfileInfo{}
	content, err := settings.ReadWGProfile(name)
	if err != nil {
		return out
	}
	c, err := wireguard.ParseConf(content)
	if err != nil {
		return out
	}
	sum := wireguard.Describe(c)
	out.OK = true
	out.Addresses = sum.Addresses
	out.DNS = sum.DNS
	out.Endpoint = sum.Endpoint
	out.Peers = sum.Peers
	out.MTU = sum.MTU
	return out
}

// GetWireGuardLogs — логи WireGuard.
func (a *App) GetWireGuardLogs() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]string, len(a.wgLogs))
	copy(out, a.wgLogs)
	return out
}

// ClearWireGuardLogs — очищає логи WireGuard.
func (a *App) ClearWireGuardLogs() {
	a.mu.Lock()
	a.wgLogs = nil
	a.mu.Unlock()
}

// ─── Власні .ovpn профілі ──────────────────────────────────────────────────

// ListProfiles — список збережених профілів.
func (a *App) ListProfiles() []string {
	return settings.ListCustomProfiles()
}

// SaveProfileDialog — відкриває діалог вибору файлу .ovpn і зберігає його.
func (a *App) SaveProfileDialog() string {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: i18n.Tr("Виберіть свій .ovpn конфіг"),
		Filters: []runtime.FileFilter{
			{DisplayName: "OpenVPN конфігурації (*.ovpn)", Pattern: "*.ovpn"},
		},
	})
	if err != nil || path == "" {
		return ""
	}
	content, err := os.ReadFile(path)
	if err != nil {
		a.log("[ERROR] Імпорт конфіга не вдався: " + err.Error())
		return ""
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	saved := settings.SaveCustomProfile(name, string(content))
	if saved != "" {
		a.log("Збережено власний .ovpn конфіг: config/profiles/" + saved)
		a.UpdateSetting("custom_ovpn_name", saved)
		a.notifyUser(i18n.Trf("Завантажено: {name}", map[string]string{"name": saved}), "positive")
	} else {
		a.log("[ERROR] Не вдалося зберегти конфіг " + path)
	}
	return saved
}

// DeleteProfile — видаляє профіль.
func (a *App) DeleteProfile(name string) bool {
	ok := settings.DeleteCustomProfile(name)
	if ok {
		a.log("Видалено власний .ovpn профіль: " + name)
		// Якщо видалили АКТИВНИЙ профіль — скидаємо вибір. Інакше
		// settings.json вказував би на неіснуючий файл і підключення
		// завершувалося б помилкою читання конфіга.
		if s := settings.Load(); s.CustomOvpnName == name {
			a.DeselectCustomProfile()
		}
	} else {
		a.log("[ERROR] Не вдалося видалити профіль: " + name)
	}
	return ok
}

// LoadProfile — читає профіль для підключення.
func (a *App) LoadProfile(name string) (string, bool) {
	content, err := settings.ReadCustomProfile(name)
	if err != nil {
		return "", false
	}
	return content, true
}

// SelectCustomProfile — встановлює активний власний профіль. Вибір профілю
// та вибір сервера VPN Gate взаємовиключні, тому обраний сервер скидається.
func (a *App) SelectCustomProfile(name string) {
	a.ClearSelectedServer()
	a.UpdateSetting("custom_ovpn_name", name)
	a.log("Власний .ovpn профіль обрано: " + name + " (обраний сервер VPN Gate скинуто).")
}

// SelectServer — обирає сервер VPN Gate зі списку та скидає активний власний
// .ovpn профіль (взаємовиключні джерела конфігурації підключення).
func (a *App) SelectServer(ip, name string) {
	a.DeselectCustomProfile()
	a.UpdateSetting("selected_server_ip", ip)
	a.UpdateSetting("selected_server_name", name)
	a.UpdateSetting("auto_server", "false")
	a.log("Обрано сервер VPN Gate: " + name + " (власний .ovpn профіль скинуто).")
}

// DeselectCustomProfile — прибирає активний власний .ovpn профіль і повертає
// авто-підбір серверів (VPN Gate) — так само, як відміна обраного сервера.
func (a *App) DeselectCustomProfile() {
	a.UpdateSetting("custom_ovpn_name", "")
	a.log("Власний .ovpn профіль скинуто — активний автоматичний підбір серверів (VPN Gate).")
}

// ClearSelectedServer — прибирає обраний сервер і повертає авто-підбір (VPN Gate).
func (a *App) ClearSelectedServer() {
	a.UpdateSetting("selected_server_ip", "")
	a.UpdateSetting("selected_server_name", "")
	a.UpdateSetting("auto_server", "true")
	a.log("Обраний сервер скинуто — активний автоматичний підбір серверів (VPN Gate).")
}

// ─── Мова ──────────────────────────────────────────────────────────────────

// SetLanguage — встановлює мову UI.
func (a *App) SetLanguage(lang string) string {
	cur := i18n.SetLanguage(lang)
	a.UpdateSetting("language", cur)
	runtime.EventsEmit(a.ctx, "language:changed", cur)
	return cur
}

// GetLanguage — поточна мова.
func (a *App) GetLanguage() string { return i18n.GetLanguage() }
