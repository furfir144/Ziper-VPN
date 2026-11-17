// Package tray — значок системного трея з меню для Ziper VPN.
package tray

import (
	"embed"
	"fmt"
	"sync"

	"github.com/getlantern/systray"

	"ziper-vpn-go/internal/i18n"
)

//go:embed icon.ico
var iconData embed.FS

var (
	mu              sync.Mutex
	instance        *Tray
	onToggle        func()
	onShow          func()
	onQuit          func()
	onUploadOvpn    func()
	onTorToggle     func()
	onRefreshServer func()
	onLoadProfile   func(string)
)

// Tray — інстанс трей-меню.
type Tray struct {
	itemToggle    *systray.MenuItem
	itemShow      *systray.MenuItem
	itemTor       *systray.MenuItem
	itemRefresh   *systray.MenuItem
	itemUpload    *systray.MenuItem
	profileItems  []*systray.MenuItem
	currentStatus string
	torActive     bool
}

// SetCallbacks — встановлює обробники подій трей.
func SetCallbacks(toggle, show, quit, upload, torToggle, refresh func(), loadProfile func(string)) {
	mu.Lock()
	defer mu.Unlock()
	onToggle = toggle
	onShow = show
	onQuit = quit
	onUploadOvpn = upload
	onTorToggle = torToggle
	onRefreshServer = refresh
	onLoadProfile = loadProfile
}

// Init — запускає значок трея (в окремому горутині).
func Init() {
	go systray.Run(onReady, onExit)
}

func onReady() {
	icon, err := iconData.ReadFile("icon.ico")
	if err != nil {
		icon = []byte{}
	}
	systray.SetIcon(icon)
	systray.SetTitle("Ziper VPN")
	systray.SetTooltip("Ziper VPN")
	tray := &Tray{}

	tray.itemShow = systray.AddMenuItem(i18n.Tr("Відкрити вікно"), "Show main window")
	systray.AddSeparator()
	tray.itemToggle = systray.AddMenuItem("○ "+i18n.Tr("Підключити"), "Toggle VPN connection")
	tray.itemRefresh = systray.AddMenuItem(i18n.Tr("Оновити список серверів"), "Refresh server list")
	tray.itemTor = systray.AddMenuItem("○ "+i18n.Tr("Tor SOCKS5"), "Toggle Tor SOCKS5")
	tray.itemUpload = systray.AddMenuItem(i18n.Tr("Завантажити свій .ovpn конфіг"), "Upload custom .ovpn profile")
	systray.AddSeparator()
	quit := systray.AddMenuItem(i18n.Tr("Вийти"), "Exit the application")

	tray.currentStatus = "DISCONNECTED"

	mu.Lock()
	instance = tray
	mu.Unlock()

	go func() {
		for range tray.itemShow.ClickedCh {
			if onShow != nil {
				onShow()
			}
		}
	}()
	go func() {
		for range tray.itemToggle.ClickedCh {
			if onToggle != nil {
				onToggle()
			}
		}
	}()
	go func() {
		for range tray.itemRefresh.ClickedCh {
			if onRefreshServer != nil {
				onRefreshServer()
			}
		}
	}()
	go func() {
		for range tray.itemTor.ClickedCh {
			if onTorToggle != nil {
				onTorToggle()
			}
		}
	}()
	go func() {
		for range tray.itemUpload.ClickedCh {
			if onUploadOvpn != nil {
				onUploadOvpn()
			}
		}
	}()
	go func() {
		for range quit.ClickedCh {
			if onQuit != nil {
				onQuit()
			}
		}
	}()
}

func onExit() {
	mu.Lock()
	instance = nil
	mu.Unlock()
}

// GetInstance — поточний інстанс трей (nil, якщо трей не запущено).
func GetInstance() *Tray {
	mu.Lock()
	defer mu.Unlock()
	return instance
}

// UpdateState — оновлює меню під статус VPN.
func (t *Tray) UpdateState(status string) {
	mu.Lock()
	defer mu.Unlock()
	t.currentStatus = status
	switch status {
	case "CONNECTED":
		t.itemToggle.SetTitle("● " + i18n.Tr("Відключити"))
		t.itemToggle.SetTooltip(i18n.Tr("Ziper VPN — підключено"))
		systray.SetTooltip("Ziper VPN — підключено")
	case "CONNECTING":
		t.itemToggle.SetTitle("◐ " + i18n.Tr("Відключити"))
		t.itemToggle.SetTooltip(i18n.Tr("Ziper VPN — підключення виконується..."))
		systray.SetTooltip("Ziper VPN — підключення виконується...")
	default:
		t.itemToggle.SetTitle("○ " + i18n.Tr("Підключити"))
		t.itemToggle.SetTooltip(i18n.Tr("Ziper VPN — відключено"))
		systray.SetTooltip("Ziper VPN — відключено")
	}
}

// UpdateTorState — оновлює позначку Tor (● увімкнено / ○ вимкнено).
func (t *Tray) UpdateTorState(active bool) {
	mu.Lock()
	defer mu.Unlock()
	t.torActive = active
	dot := "○"
	if active {
		dot = "●"
	}
	t.itemTor.SetTitle(dot + " " + i18n.Tr("Tor SOCKS5"))
}

// RefreshProfiles — оновлює список профілів у меню.
func (t *Tray) RefreshProfiles(names []string) {
	mu.Lock()
	defer mu.Unlock()
	// старі елементи прибираємо
	for _, item := range t.profileItems {
		item.Hide()
	}
	t.profileItems = t.profileItems[:0]
	for _, name := range names {
		item := systray.AddMenuItem(i18n.Trf("Профіль {name}", map[string]string{"name": name}),
			fmt.Sprintf("Load %s", name))
		item.Show()
		t.profileItems = append(t.profileItems, item)
		n := name
		go func() {
			for range item.ClickedCh {
				if onLoadProfile != nil {
					onLoadProfile(n)
				}
			}
		}()
	}
}
