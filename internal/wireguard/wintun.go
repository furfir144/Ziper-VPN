// wintun.go — завантаження драйвера Wintun для WireGuard тунелю
// (використовує завантажувач з OpenVPN-модуля: той самий офіційний
// архів wintun.net, DLL кладеться в openVPN/wintun.dll).
package wireguard

import (
	"ziper-vpn-go/internal/vpn"
)

// vpnDownloadWintun — гарантує наявність wintun.dll у папці openVPN;
// повертає шлях до нього або "".
func vpnDownloadWintun() string {
	return vpn.DownloadWintun()
}
