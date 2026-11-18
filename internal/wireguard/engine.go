// engine.go — рушій WireGuard тунелю на базі wireguard-go:
// створення Wintun адаптера (tun.CreateTUN), налаштування device
// через UAPI (IpcSet), прив'язка UDP-сокетів до фізичного інтерфейсу,
// призначення адреси/DNS/маршрутів та моніторинг хендшейків.
package wireguard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"

	"ziper-vpn-go/internal/i18n"
	"ziper-vpn-go/internal/win"
)

// ─── Глобальний стан ──────────────────────────────────────────────────────

const adapterName = "ZiperVPN WG"

var (
	appDir        string
	globalMu      sync.Mutex
	curEngine     *engine
	logCallback   func(string)
	connectedCb   func()
	disconnectedCb func(reason string)
	statsCb       func(State)
)

func init() {
	exe, _ := os.Executable()
	appDir = filepath.Dir(exe)
}

// SetCallbacks — обробники подій тунелю (викликається з app.go).
func SetCallbacks(log func(string), onConnected func(), onDisconnected func(string), onStats func(State)) {
	logCallback = log
	connectedCb = onConnected
	disconnectedCb = onDisconnected
	statsCb = onStats
}

func logf(msg string) {
	if logCallback != nil {
		logCallback(msg)
	}
}

// State — стан тунелю для UI.
type State struct {
	Status        string   `json:"status"` // DISCONNECTED | CONNECTING | CONNECTED
	Profile       string   `json:"profile"`
	Address       string   `json:"address"`
	Endpoint      string   `json:"endpoint"`
	DNS           []string `json:"dns"`
	RXBytes       int64    `json:"rx"`
	TXBytes       int64    `json:"tx"`
	LastHandshake int64    `json:"last_handshake"` // сек тому; -1 — ще не було
}

// engine — активна сесія тунелю.
type engine struct {
	mu         sync.Mutex
	dev        *device.Device
	tunDev     tun.Device
	status     string // CONNECTING | CONNECTED
	profile    string
	conf       *Conf
	dnsServers []string
	endpoint   string // resolved "ip:port" першого піра
	address    string // перша IPv4 адреса конфіга
	rx, tx     int64
	handshake  time.Time // zero — ще не було
	startedAt  time.Time
	cancelReq  bool
	closing    bool
}

// IsActive — чи активний тунель (підключення або з'єднано).
func IsActive() bool {
	globalMu.Lock()
	defer globalMu.Unlock()
	return curEngine != nil
}

// CurrentState — знімок стану для біндингу.
func CurrentState() State {
	globalMu.Lock()
	e := curEngine
	globalMu.Unlock()
	if e == nil {
		return State{Status: "DISCONNECTED", LastHandshake: -1}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	st := State{
		Status:  e.status,
		Profile: e.profile,
		RXBytes: e.rx,
		TXBytes: e.tx,
	}
	if e.address != "" {
		st.Address = e.address
	}
	if e.endpoint != "" {
		st.Endpoint = e.endpoint
	}
	st.DNS = append(st.DNS, e.dnsServers...)
	if !e.handshake.IsZero() {
		st.LastHandshake = int64(time.Since(e.handshake).Seconds())
	} else {
		st.LastHandshake = -1
	}
	return st
}

// RequestDisconnect — поза чергою: запит на розрив тунелю.
func RequestDisconnect() {
	globalMu.Lock()
	e := curEngine
	globalMu.Unlock()
	if e != nil {
		e.mu.Lock()
		e.cancelReq = true
		e.mu.Unlock()
	}
}

// Disconnect — грасіозне завершення тунелю (закриває адаптер).
func Disconnect() {
	globalMu.Lock()
	e := curEngine
	globalMu.Unlock()
	if e == nil {
		return
	}
	e.mu.Lock()
	if e.closing {
		e.mu.Unlock()
		return
	}
	e.closing = true
	e.cancelReq = true
	e.mu.Unlock()

	logf(i18n.Tr("Закриття WireGuard тунелю..."))
	if e.dev != nil {
		e.dev.Close() // закриває і TUN адаптер, і сокети
	}
	// монітор завершиться сам (IpcGet поверне помилку)
	globalMu.Lock()
	if curEngine == e {
		curEngine = nil
	}
	globalMu.Unlock()
	e.mu.Lock()
	wasStatus := e.status
	e.status = "DISCONNECTED"
	e.mu.Unlock()
	if wasStatus != "DISCONNECTED" {
		if disconnectedCb != nil {
			disconnectedCb("Відключено користувачем")
		}
		emitState()
	}
}

// emitState — надсилає знімок стану у UI (якщо підписано).
func emitState() {
	if statsCb != nil {
		statsCb(CurrentState())
	}
}

// ─── Підключення ──────────────────────────────────────────────────────────

// Connect — встановлює WireGuard тунель із .conf конфігурації.
// fallbackDNS використовується, якщо у конфізі немає DNS.
func Connect(profileName, content string, fallbackDNS []string) error {
	// конфлікти з іншими режимами прибирає app.go; тут лише стан
	globalMu.Lock()
	prev := curEngine
	globalMu.Unlock()
	if prev != nil {
		Disconnect()
		time.Sleep(300 * time.Millisecond)
	}

	// 1. Розбір конфіга
	c, err := ParseConf(content)
	if err != nil {
		logf("[ERROR] " + i18n.Trf("Некоректний WireGuard конфіг: {ex}", map[string]string{"ex": err.Error()}))
		return err
	}

	e := &engine{
		status:    "CONNECTING",
		profile:   profileName,
		conf:      c,
		startedAt: time.Now(),
	}
	e.dnsServers = append(e.dnsServers, c.DNS...)
	if len(e.dnsServers) == 0 {
		e.dnsServers = append(e.dnsServers, fallbackDNS...)
	}
	for _, a := range c.Addresses {
		if ip := net.ParseIP(strings.Split(a, "/")[0]); ip != nil && ip.To4() != nil {
			e.address = a
			break
		}
	}

	globalMu.Lock()
	curEngine = e
	globalMu.Unlock()
	emitState()

	logf(i18n.Tr("=== Запуск WireGuard тунелю ==="))
	logf(i18n.Trf("Профіль: {name} | Адреса тунелю: {addr} | Пірів: {peers}",
		map[string]string{"name": profileName, "addr": e.address, "peers": fmt.Sprint(len(c.Peers))}))

	// 2. Endpoint: доменні імена розпізнаємо самі (bind приймає лише IP)
	endpoints := make([]string, len(c.Peers))
	for i, p := range c.Peers {
		ep, err := resolveEndpoint(p.Endpoint, 8*time.Second)
		if err != nil {
			return e.fail(i18n.Trf("Не вдалося розпізнати Endpoint «{ep}»: {ex}",
				map[string]string{"ep": p.Endpoint, "ex": err.Error()}))
		}
		endpoints[i] = ep
		if e.endpoint == "" {
			e.endpoint = ep
		}
		logf(i18n.Trf("Endpoint {i}: {ep}", map[string]string{"i": fmt.Sprint(i + 1), "ep": ep}))
	}
	if isCancelled(e) {
		return e.fail("Підключення відмінено користувачем")
	}

	// 3. wintun.dll поруч із застосунком
	if err := ensureWintunDLL(); err != nil {
		return e.fail(err.Error())
	}

	// 4. Wintun адаптер + wireguard device
	mtu := 1420
	if c.MTU > 0 {
		mtu = c.MTU
	}
	tunDev, err := tun.CreateTUN(adapterName, mtu)
	if err != nil {
		return e.fail(i18n.Trf("Не вдалося створити Wintun адаптер: {ex} (потрібні права адміністратора)",
			map[string]string{"ex": err.Error()}))
	}
	e.mu.Lock()
	e.tunDev = tunDev
	e.mu.Unlock()

	dev := device.NewDevice(tunDev, conn.NewDefaultBind(), &device.Logger{
		Verbosef: func(format string, args ...any) { logf("[WG] " + fmt.Sprintf(format, args...)) },
		Errorf:   func(format string, args ...any) { logf("[WG-ERR] " + fmt.Sprintf(format, args...)) },
	})
	e.mu.Lock()
	e.dev = dev
	e.mu.Unlock()
	logf(i18n.Trf("Wintun адаптер «{name}» створено (MTU {mtu}).",
		map[string]string{"name": adapterName, "mtu": fmt.Sprint(mtu)}))

	// 5. UAPI конфігурація
	uapi, err := buildUAPI(c, endpoints)
	if err != nil {
		return e.fail(err.Error())
	}
	if err := dev.IpcSet(uapi); err != nil {
		return e.fail(i18n.Trf("Помилка конфігурації тунелю: {ex}", map[string]string{"ex": err.Error()}))
	}

	// 6. Підйом тунелю (відкриваються UDP сокети)
	if err := dev.Up(); err != nil {
		return e.fail(i18n.Trf("Не вдалося запустити тунель: {ex}", map[string]string{"ex": err.Error()}))
	}

	// 7. Прив'язка сокетів до фізичного інтерфейсу — щоб UDP пакети
	// хендшейків не пішли у щойно створений тунель (full-tunnel loop)
	bindToPhysical(dev.Bind())

	// 8. Мережеві налаштування адаптера
	if err := e.setupNetwork(); err != nil {
		return e.fail(err.Error())
	}

	logf(i18n.Tr("Тунель ініціалізовано, очікую хендшейк з сервером..."))
	go e.monitor()
	return nil
}

// fail — помилка підключення: чистка та повідомлення.
func (e *engine) fail(reason string) error {
	logf("[ERROR] " + reason)
	e.mu.Lock()
	if e.closing {
		e.mu.Unlock()
		return errors.New(reason)
	}
	e.closing = true
	dev := e.dev
	e.mu.Unlock()
	if dev != nil {
		dev.Close()
	}
	globalMu.Lock()
	if curEngine == e {
		curEngine = nil
	}
	globalMu.Unlock()
	if disconnectedCb != nil {
		disconnectedCb(reason)
	}
	emitState()
	return errors.New(reason)
}

// isCancelled — чи запитано відміну.
func isCancelled(e *engine) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.cancelReq
}

// resolveEndpoint — перетворює "host:port" на числовий "ip:port"
// ([v6] у дужках), розпізнаючи DNS-імена.
func resolveEndpoint(endpoint string, timeout time.Duration) (string, error) {
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return "", fmt.Errorf("%w", err)
	}
	if ip := net.ParseIP(host); ip != nil {
		return formatEndpoint(ip, port), nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return "", err
	}
	if len(addrs) == 0 {
		return "", fmt.Errorf("хост %s не розпізнано", host)
	}
	// пріоритет IPv4
	for _, a := range addrs {
		if a.IP.To4() != nil {
			return formatEndpoint(a.IP, port), nil
		}
	}
	return formatEndpoint(addrs[0].IP, port), nil
}

func formatEndpoint(ip net.IP, port string) string {
	if ip.To4() == nil {
		return "[" + ip.String() + "]:" + port
	}
	return ip.String() + ":" + port
}

// buildUAPI — конвертує .conf у формат UAPI (IpcSet).
func buildUAPI(c *Conf, endpoints []string) (string, error) {
	priv, err := keyHex(c.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("некоректний PrivateKey: %w", err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "private_key=%s\n", priv)
	if c.ListenPort > 0 {
		fmt.Fprintf(&b, "listen_port=%d\n", c.ListenPort)
	}
	b.WriteString("replace_peers=true\n")
	for i, p := range c.Peers {
		pub, err := keyHex(p.PublicKey)
		if err != nil {
			return "", fmt.Errorf("пір %d: некоректний PublicKey: %w", i+1, err)
		}
		fmt.Fprintf(&b, "public_key=%s\n", pub)
		if p.PresharedKey != "" {
			psk, err := keyHex(p.PresharedKey)
			if err != nil {
				return "", fmt.Errorf("пір %d: некоректний PresharedKey: %w", i+1, err)
			}
			fmt.Fprintf(&b, "preshared_key=%s\n", psk)
		}
		if endpoints[i] != "" {
			fmt.Fprintf(&b, "endpoint=%s\n", endpoints[i])
		}
		if p.Keepalive > 0 {
			fmt.Fprintf(&b, "persistent_keepalive_interval=%d\n", p.Keepalive)
		}
		b.WriteString("replace_allowed_ips=true\n")
		for _, ip := range p.AllowedIPs {
			fmt.Fprintf(&b, "allowed_ip=%s\n", ip)
		}
	}
	return b.String(), nil
}

// bindToPhysical — прив'язує UDP сокети до фізичного інтерфейсу.
// Це механізм офіційного WireGuard клієнта проти «петлі» маршрутів.
func bindToPhysical(bind conn.Bind) {
	for _, family := range []string{"4", "6"} {
		r := physicalDefaultRoute(family)
		if r == nil {
			continue
		}
		binder, ok := bind.(conn.BindSocketToInterface)
		if !ok {
			logf("[WARN] Bind не підтримує прив'язку до інтерфейсу.")
			return
		}
		var err error
		if family == "4" {
			err = binder.BindSocketToInterface4(uint32(r.IfIndex), false)
		} else {
			err = binder.BindSocketToInterface6(uint32(r.IfIndex), false)
		}
		if err != nil {
			logf("[WARN] " + i18n.Trf("Сокети не прив'язано до «{name}»: {ex}",
				map[string]string{"name": r.Alias, "ex": err.Error()}))
		} else {
			logf(i18n.Trf("UDP сокети тунелю прив'язано до фізичного інтерфейсу «{name}».",
				map[string]string{"name": r.Alias}))
		}
	}
}

// setupNetwork — адреса, метрика, DNS та маршрути адаптера.
func (e *engine) setupNetwork() error {
	first := true
	for _, addr := range e.conf.Addresses {
		if isCancelled(e) {
			return errors.New("Підключення відмінено користувачем")
		}
		if err := assignAddress(adapterName, addr, first); err != nil {
			return err
		}
		first = false
	}
	setAdapterMetric(adapterName)
	if err := setAdapterDNS(adapterName, e.dnsServers); err != nil {
		logf("[WARN] " + err.Error())
	}
	if !win.IsAdmin() {
		logf("[WARN] " + i18n.Tr("Без прав адміністратора маршрути не встановити."))
	}
	installAllowedIPRoutes(adapterName, e.conf.Peers)
	return nil
}

// ─── Моніторинг ───────────────────────────────────────────────────────────

// peerStats — статистика з IpcGet.
type peerStats struct {
	rx, tx    int64
	handshake time.Time
}

// readStats — читає статистику через IpcGet (rx/tx/хендшейк).
func (e *engine) readStats() *peerStats {
	e.mu.Lock()
	dev := e.dev
	e.mu.Unlock()
	if dev == nil {
		return nil
	}
	s, err := dev.IpcGet()
	if err != nil {
		return nil // тунель закрився
	}
	ps := &peerStats{}
	var hsSec int64
	for _, line := range strings.Split(s, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch key {
		case "rx_bytes":
			fmt.Sscanf(value, "%d", &ps.rx)
		case "tx_bytes":
			fmt.Sscanf(value, "%d", &ps.tx)
		case "last_handshake_time_sec":
			fmt.Sscanf(value, "%d", &hsSec)
		}
	}
	if hsSec > 0 {
		t := time.Unix(hsSec, 0)
		if ps.handshake.Before(t) {
			ps.handshake = t
		}
	}
	return ps
}

// monitor — цикл стану: хендшейк → CONNECTED, статистика, детекція збоїв.
func (e *engine) monitor() {
	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()
	handshakeDeadline := time.Now().Add(30 * time.Second)
	lastRx, lastTx := int64(0), int64(0)
	warnedStale := false
	for {
		select {
		case <-ticker.C:
		}
		ps := e.readStats()
		if ps == nil {
			return // тунель закрито
		}
		e.mu.Lock()
		e.rx, e.tx = ps.rx, ps.tx
		if ps.handshake.After(e.handshake) {
			e.handshake = ps.handshake
		}
		status := e.status
		e.mu.Unlock()

		// перехід у CONNECTED після першого хендшейку
		if status == "CONNECTING" && !ps.handshake.IsZero() {
			e.mu.Lock()
			e.status = "CONNECTED"
			e.mu.Unlock()
			status = "CONNECTED"
			logf(i18n.Tr("WireGuard хендшейк успішний — тунель активний! ✓"))
			if connectedCb != nil {
				connectedCb()
			}
		} else if status == "CONNECTING" {
			// детекція «хендшейк не вдався»
			if time.Now().After(handshakeDeadline) {
				e.mu.Lock()
				txMoved := e.tx > 0
				kaOn := false
				for _, p := range e.conf.Peers {
					if p.Keepalive > 0 {
						kaOn = true
					}
				}
				e.mu.Unlock()
				keepTrying := time.Since(e.startedAt) < 90*time.Second && !txMoved && !kaOn
				if !keepTrying {
					e.fail(i18n.Tr("Хендшейк не вдався: сервер недоступний або ключі невірні."))
					return
				}
				if time.Since(e.startedAt) >= 30*time.Second {
					logf("[WARN] " + i18n.Tr("Хендшейк ще не отримано — трафік через тунель не генерується..."))
					handshakeDeadline = time.Now().Add(60 * time.Second)
				}
			}
		}

		// попередження про застарілий хендшейк (сервер «відвалився»)
		if status == "CONNECTED" && !ps.handshake.IsZero() {
			stale := time.Since(ps.handshake)
			idle := ps.rx == lastRx && ps.tx == lastTx
			if stale > 180*time.Second && idle && !warnedStale {
				logf("[WARN] " + i18n.Tr("Хендшейк застарілий (>180с) — сервер може бути недоступним."))
				warnedStale = true
			} else if !idle {
				warnedStale = false
			}
		}
		lastRx, lastTx = ps.rx, ps.tx

		emitState()
	}
}
