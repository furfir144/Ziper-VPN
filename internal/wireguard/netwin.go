// netwin.go — Windows-мережеві операції для WireGuard тунелю:
// призначення IP-адрес на Wintun адаптер, DNS, маршрути для AllowedIPs
// та прив'язка UDP-сокетів тунелю до фізичного інтерфейсу (захист від
// «петлі» маршрутів при full-tunnel конфігурації 0.0.0.0/0).
package wireguard

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"ziper-vpn-go/internal/i18n"
	"ziper-vpn-go/internal/win"
)

// psEncodedCommand — кодує скрипт PowerShell у base64 (UTF-16LE),
// щоб викликати його через `powershell -EncodedCommand` без проблем
// з лапками, пробілами та кирилицею в іменах адаптерів.
func psEncodedCommand(script string) string {
	u16 := utf16.Encode([]rune(script))
	b := make([]byte, len(u16)*2)
	for i, v := range u16 {
		b[i*2] = byte(v)
		b[i*2+1] = byte(v >> 8)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// runPS — запускає PowerShell-скрипт і повертає вивід.
func runPS(script string) string {
	script = "$ProgressPreference = 'SilentlyContinue';\n" + script
	out, _, _ := win.RunHidden("powershell", "-NoProfile", "-NonInteractive", "-EncodedCommand", psEncodedCommand(script))
	return out
}

// psQuote — екранує лапки для PowerShell-рядка.
func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// ─── Wintun.dll ───────────────────────────────────────────────────────────

// ensureWintunDLL — гарантує, що wintun.dll лежить у папці застосунку
// (бібліотека wireguard-go шукає DLL саме там). За потреби копіює з
// папки openVPN або завантажує з wintun.net.
func ensureWintunDLL() error {
	dst := filepath.Join(appDir, "wintun.dll")
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	// 1. wintun.dll уже міг бути завантажений OpenVPN-модулем
	if data, err := os.ReadFile(filepath.Join(appDir, "openVPN", "wintun.dll")); err == nil {
		if err := os.WriteFile(dst, data, 0o644); err == nil {
			logf(i18n.Tr("wintun.dll скопійовано до папки застосунку."))
			return nil
		}
	}
	// 2. завантаження з wintun.net (офіційні білди)
	path := vpnDownloadWintun()
	if path == "" {
		return fmt.Errorf("не вдалося завантажити драйвер Wintun")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("не вдалося прочитати wintun.dll: %w", err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("не вдалося скопіювати wintun.dll: %w", err)
	}
	logf(i18n.Tr("Драйвер Wintun встановлено: ") + dst)
	return nil
}

// ─── Адреса, DNS, маршрути ────────────────────────────────────────────────

// assignAddress — призначає IP-адресу Wintun адаптеру.
// Перша адреса адаптера є джерелом вихідних пакетів; наступні —
// з SkipAsSource (як в офіційному WireGuard клієнті).
func assignAddress(alias, cidr string, first bool) error {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("некоректна адреса %s: %w", cidr, err)
	}
	plen, _ := ipnet.Mask.Size()
	skip := ""
	if !first {
		skip = " -SkipAsSource $true"
	}
	script := fmt.Sprintf(`$ip = %s
$a = Get-NetIPAddress -InterfaceAlias %s -IPAddress $ip -ErrorAction SilentlyContinue
if (-not $a) {
  New-NetIPAddress -InterfaceAlias %s -IPAddress $ip -PrefixLength %d%s -ErrorAction Stop | Out-Null
}
Write-Output 'OK'`, psQuote(ip.String()), psQuote(alias), psQuote(alias), plen, skip)
	out := runPS(script)
	if !strings.Contains(out, "OK") {
		return fmt.Errorf("не вдалося призначити адресу %s адаптеру «%s»", cidr, alias)
	}
	logf(i18n.Trf("Адресу {addr} призначено адаптеру «{name}».",
		map[string]string{"addr": cidr, "name": alias}))
	return nil
}

// setAdapterMetric — знижує метрику адаптера, щоб його DNS та маршрути
// мали пріоритет над фізичним інтернетом (як в офіційному клієнті).
func setAdapterMetric(alias string) {
	script := fmt.Sprintf(`try {
  Set-NetIPInterface -InterfaceAlias %s -InterfaceMetric 3 -ErrorAction Stop
  Write-Output 'OK'
} catch { Write-Output 'FAIL' }`, psQuote(alias))
	if out := runPS(script); strings.Contains(out, "OK") {
		logf(i18n.Tr("Метрику адаптера тунелю знижено (пріоритет над фізичним інтернетом)."))
	}
}

// setAdapterDNS — прописує DNS сервери на адаптері тунелю.
func setAdapterDNS(alias string, servers []string) error {
	if len(servers) == 0 {
		return nil
	}
	var quoted []string
	for _, s := range servers {
		if net.ParseIP(s) == nil {
			continue
		}
		quoted = append(quoted, psQuote(s))
	}
	if len(quoted) == 0 {
		return nil
	}
	script := fmt.Sprintf(`try {
  Set-DnsClientServerAddress -InterfaceAlias %s -ServerAddresses @(%s) -ErrorAction Stop
  Write-Output 'OK'
} catch { Write-Output ('FAIL|' + $_.Exception.Message) }`,
		psQuote(alias), strings.Join(quoted, ","))
	out := runPS(script)
	if !strings.Contains(out, "OK") {
		return fmt.Errorf("не вдалося прописати DNS на адаптері тунелю")
	}
	logf(i18n.Trf("DNS тунелю: {servers}",
		map[string]string{"servers": strings.Join(servers, ", ")}))
	return nil
}

// addOnLinkRoute — додає маршрут-онлінк (без шлюзу) через адаптер тунелю.
// nexthop у netsh опційний — відсутність означає «on-link».
func addOnLinkRoute(family, prefix, alias string) bool {
	fam := "ipv4"
	if family == "6" {
		fam = "ipv6"
	}
	_, rc, err := win.RunHidden("netsh", "interface", fam, "add", "route",
		"prefix="+prefix, "interface="+alias, "metric=1", "store=active")
	return err == nil && rc == 0
}

// installAllowedIPRoutes — розкладає AllowedIPs усіх пірів на маршрути
// через адаптер тунелю. 0.0.0.0/0 та ::/0 розбиваються на /1 підмережі
// (стандартний прийом WireGuard, щоб перекрити дефолтний маршрут).
func installAllowedIPRoutes(alias string, peers []PeerConf) {
	seen := map[string]bool{}
	for _, p := range peers {
		for _, item := range p.AllowedIPs {
			if seen[item] {
				continue
			}
			seen[item] = true
			prefix, err := netip.ParsePrefix(item)
			if err != nil {
				continue
			}
			family := "4"
			if !prefix.Addr().Is4() {
				family = "6"
			}
			// розбиття дефолтних маршрутів на половини
			if prefix.Bits() == 0 {
				if family == "4" {
					addOnLinkRoute(family, "0.0.0.0/1", alias)
					addOnLinkRoute(family, "128.0.0.0/1", alias)
					logf(i18n.Tr("Full-tunnel: маршрути 0.0.0.0/1 + 128.0.0.0/1 через тунель."))
				} else {
					addOnLinkRoute(family, "::/1", alias)
					addOnLinkRoute(family, "8000::/1", alias)
					logf(i18n.Tr("Full-tunnel (IPv6): маршрути ::/1 + 8000::/1 через тунель."))
				}
				continue
			}
			if !addOnLinkRoute(family, prefix.String(), alias) {
				logf("[WARN] " + i18n.Trf("Маршрут {p} не додано (можливо, вже існує).",
					map[string]string{"p": prefix.String()}))
			}
		}
	}
}

// ─── Фізичний інтерфейс ───────────────────────────────────────────────────

// defaultRouteInfo — дані дефолтного маршруту фізичного інтерфейсу.
type defaultRouteInfo struct {
	IfIndex int
	Alias   string
	NextHop string
}

// physicalDefaultRoute — шукає дефолтний маршрут фізичного інтерфейсу
// (VPN-адаптери виключаються фільтром).
func physicalDefaultRoute(family string) *defaultRouteInfo {
	prefix := "0.0.0.0/0"
	if family == "6" {
		prefix = "::/0"
	}
	script := fmt.Sprintf(`Get-NetRoute -DestinationPrefix %s -AddressFamily IPv%s -ErrorAction SilentlyContinue |
  Where-Object { %s } |
  Sort-Object { $_.RouteMetric + $_.InterfaceMetric } |
  Select-Object -First 1 |
  ForEach-Object { Write-Output ('IF|' + $_.InterfaceIndex + '|' + $_.InterfaceAlias + '|' + $_.NextHop) }`,
		psQuote(prefix), family, wgVPNAdapterFilter())
	out := runPS(script)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(strings.TrimSpace(line), "\r")
		if !strings.HasPrefix(line, "IF|") {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(line, "IF|"), "|", 3)
		if len(parts) != 3 {
			continue
		}
		idx := 0
		if _, err := fmt.Sscanf(parts[0], "%d", &idx); err != nil || idx <= 0 {
			continue
		}
		return &defaultRouteInfo{IfIndex: idx, Alias: parts[1], NextHop: parts[2]}
	}
	return nil
}

// wgVPNAdapterFilter — умова «адаптер — не VPN-тунель» у PowerShell
// (використовується для пошуку фізичного інтернет-адаптера).
func wgVPNAdapterFilter() string {
	return `$_.InterfaceAlias -notmatch 'Ziper|Wintun|TAP|Loopback|vEthernet|OpenVPN|WireGuard|IPSec|VPN'`
}
