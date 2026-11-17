package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"ziper-vpn-go/internal/settings"
	"ziper-vpn-go/internal/tray"
	"ziper-vpn-go/internal/win"
)

//go:embed all:frontend/dist
var assets embed.FS

// fitWindow — обмежує розмір вікна робочою областю екрана (без панелі задач),
// щоб на маленьких моніторах вікно вміщалося повністю і не доводилось
// стискати його вручну. На великих екранах повертає заданий розмір без змін.
func fitWindow(w, h int) (int, int) {
	type rect struct {
		Left, Top, Right, Bottom int32
	}
	var r rect
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("SystemParametersInfoW")
	const spiGetWorkArea = 0x0030
	ret, _, _ := proc.Call(uintptr(spiGetWorkArea), 0, uintptr(unsafe.Pointer(&r)), 0)
	if ret == 0 {
		return w, h
	}
	workW := int(r.Right - r.Left)
	workH := int(r.Bottom - r.Top)
	// Запас: по горизонталі — рамки вікна; по вертикалі — заголовок вікна.
	const marginX = 24
	const marginY = 48
	if w > workW-marginX {
		w = workW - marginX
	}
	if h > workH-marginY {
		h = workH - marginY
	}
	if w < 640 {
		w = 640
	}
	if h < 480 {
		h = 480
	}
	return w, h
}

func main() {
	noElevate := flag.Bool("no-elevate", false, "Не запитувати UAC при старті")
	autostart := flag.Bool("autostart", false, "Запуск у режимі автозапуску (згорнуте вікно)")
	flag.Parse()
	_ = autostart

	// Підвищення прав: Wintun адаптер та маршрутизація вимагають адміна.
	if !*noElevate {
		if !win.IsAdmin() {
			if win.Elevate() {
				// процес перезапущено від імені адміністратора — виходимо
				return
			}
		}
	}

	app := NewApp()

	// Стартуємо трей у фоновому горутині
	tray.SetCallbacks(
		func() { app.ToggleVPN() },                           // toggle
		func() { app.showWindow() },                          // show
		func() { app.quitApp() },                             // quit
		func() { app.trayUpload() },                          // upload ovpn
		func() { app.ToggleTor() },                           // tor
		func() { app.RefreshServers() },                      // refresh servers
		func(name string) { app.SelectCustomProfile(name) },  // load profile
	)
	tray.Init()

	// Налаштування
	settings.Init()

	// Компактне вікно за замовчуванням; на маленьких екранах розмір
	// автоматично обмежується робочою областю, щоб не стискати вручну.
	width, height := fitWindow(980, 700)
	minWidth, minHeight := 820, 560
	if minWidth > width {
		minWidth = width
	}
	if minHeight > height {
		minHeight = height
	}

	// Wails — нативне вікно з webview
	err := wails.Run(&options.App{
		Title:     "Ziper VPN",
		Width:     width,
		Height:    height,
		MinWidth:  minWidth,
		MinHeight: minHeight,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 11, G: 11, B: 11, A: 1},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			Theme:                windows.Dark,
		},
		OnStartup: app.startup,
		OnBeforeClose: func(ctx context.Context) bool {
			// ховаємо вікно в трей замість реального закриття
			app.hideWindow()
			return true
		},
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		fmt.Println("Error:", err.Error())
	}
}
