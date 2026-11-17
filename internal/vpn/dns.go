// dns.go — DNS-логіка: вибір серверів за пріоритетом, застосування DNS
// на адаптер тунелю та системно на інтернет-адаптерах (працює без VPN).
// Застосування йде через PowerShell (Set-DnsClientServerAddress) з
// EncodedCommand — це уникає проблем з лапками, пробілами та кирилицею
// в іменах адаптерів.
package vpn

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"unicode/utf16"

	"ziper-vpn-go/internal/i18n"
	"ziper-vpn-go/internal/win"
)

// dnsResolvers — відомі резолвери з налаштувань.
var dnsResolvers = map[string][]string{
	"Cloudflare (DoH)": {"1.1.1.1", "1.0.0.1"},
	"Cloudflare":       {"1.1.1.1", "1.0.0.1"},
	"Quad9":            {"9.9.9.9", "149.112.112.112"},
	"Google":           {"8.8.8.8", "8.8.4.4"},
	"AdGuard":          {"94.140.14.14", "94.140.15.15"},
}

// DNSServers — обчислює список DNS за пріоритетом:
// власні DNS → явний резолвер → блокування реклами (AdGuard, лише в «Авто»).
// Повертає (список, джерело): джерело "" означає «автоматично (провайдер)».
func DNSServers(blockAds bool, resolver, customDNS string) ([]string, string) {
	if customDNS != "" {
		var servers []string
		for _, token := range strings.FieldsFunc(customDNS, func(r rune) bool {
			return r == ',' || r == ';' || r == ' '
		}) {
			if net.ParseIP(token) == nil {
				logf(i18n.Tr("Пропущено некоректну DNS адресу: ") + token)
				continue
			}
			servers = append(servers, token)
		}
		if len(servers) > 0 {
			return servers, "custom"
		}
	}
	// явний вибір резолвера має пріоритет над блокуванням реклами
	if resolver != "" && resolver != "Auto" {
		if r, ok := dnsResolvers[resolver]; ok {
			return append([]string(nil), r...), "resolver"
		}
	}
	if blockAds {
		return []string{"94.140.14.14", "94.140.15.15"}, "blockads"
	}
	return nil, ""
}

// psEncodedCommand — кодує скрипт PowerShell у base64 (UTF-16LE),
// щоб викликати його через `powershell -EncodedCommand` без проблем
// з лапками та спецсимволами.
func psEncodedCommand(script string) string {
	u16 := utf16.Encode([]rune(script))
	b := make([]byte, len(u16)*2)
	for i, v := range u16 {
		binary.LittleEndian.PutUint16(b[i*2:], v)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// psTunnelFilter — умова «адаптер — не тунель» у PowerShell.
const psTunnelFilter = `$t = $_.InterfaceAlias; if ($t -match 'Ziper|Wintun|TAP|Loopback|vEthernet|OpenVPN') { return }`

// internetAdaptersScript — скрипт виводу інтернет-адаптерів.
const internetAdaptersScript = `Get-NetIPConfiguration | Where-Object { $_.IPv4DefaultGateway -and $_.NetAdapter.Status -eq 'Up' } | ForEach-Object { Write-Output ('ADAPTER|' + $_.InterfaceAlias) }`

// runPS — запускає PowerShell-скрипт і повертає вивід. Прогрес-записи
// глушаться ($ProgressPreference), щоб CLIXML-сміття зі stderr не
// перемішувалося з корисним виводом у CombinedOutput.
func runPS(script string) string {
	script = "$ProgressPreference = 'SilentlyContinue';\n" + script
	out, rc, err := runTool("powershell", "-NoProfile", "-NonInteractive", "-EncodedCommand", psEncodedCommand(script))
	_ = rc
	if err != nil {
		return out
	}
	return out
}

// ─── Застосування DNS на адаптер тунелю ────────────────────────────────────

// adapterCandidates — можливі імена VPN-адаптера (Wintun-адаптер і
// адаптер, створений через tapctl).
func adapterCandidates() []string {
	return []string{adapterName, "Ziper VPN"}
}

// applySessionDNS — прописує DNS-сервери сесії на адаптері тунелю.
// dhcp-option з Wintun-адаптером працює не завжди, тому дублюємо явно.
func applySessionDNS(s *session) {
	s.mu.Lock()
	servers := append([]string(nil), s.dnsServers...)
	s.mu.Unlock()
	if len(servers) == 0 {
		return
	}

	// будуємо PowerShell-скрипт: спробуємо обидва імена адаптера
	var addrList []string
	for _, dns := range servers {
		addrList = append(addrList, "'" + dns + "'")
	}
	quoted := strings.Join(addrList, ",")
	var names []string
	for _, name := range adapterCandidates() {
		names = append(names, "'" + strings.ReplaceAll(name, "'", "''") + "'")
	}
	script := fmt.Sprintf(`$servers = @(%s)
foreach ($name in @(%s)) {
  $a = Get-NetAdapter -ErrorAction SilentlyContinue | Where-Object { $_.InterfaceAlias -eq $name -or $_.Name -eq $name }
  if (-not $a) { continue }
  try {
    Set-DnsClientServerAddress -InterfaceAlias $name -ServerAddresses $servers -ErrorAction Stop
    Write-Output ('OK|' + $name)
  } catch { Write-Output ('FAIL|' + $name + '|' + $_.Exception.Message) }
}`, quoted, strings.Join(names, ","))

	out := runPS(script)
	if strings.Contains(out, "OK|") {
		logf(i18n.Tr("DNS застосовано на адаптері тунелю: {servers}",
			map[string]string{"servers": strings.Join(servers, ", ")}))
	} else {
		logf("[WARN] Не вдалося прописати DNS на адаптері тунелю.")
	}
}

// resetAdapterDNS — повертає адаптери тунелю до «DNS автоматично».
func resetAdapterDNS() {
	var names []string
	for _, name := range adapterCandidates() {
		names = append(names, "'"+strings.ReplaceAll(name, "'", "''")+"'")
	}
	script := fmt.Sprintf(`foreach ($name in @(%s)) {
  try { Set-DnsClientServerAddress -InterfaceAlias $name -ResetServerAddresses -ErrorAction Stop } catch { }
}`, strings.Join(names, ","))
	runPS(script)
}

// ─── Системний DNS (працює навіть без VPN) ─────────────────────────────────

// systemInternetAdapters — імена активних адаптерів з default gateway
// (фізичний інтернет), без VPN-адаптерів та віртуальних.
func systemInternetAdapters() []string {
	out := runPS(internetAdaptersScript)
	var names []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(strings.TrimSpace(line), "\r")
		if !strings.HasPrefix(line, "ADAPTER|") {
			continue
		}
		name := strings.TrimPrefix(line, "ADAPTER|")
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	return names
}

// SystemDNSApply — прописує DNS на всіх інтернет-адаптерах системи,
// тож фільтрація (Quad9 / AdGuard / власні DNS) працює навіть без VPN.
// Повертає nil, якщо DNS застосовано хоча б на одному адаптері.
func SystemDNSApply(servers []string) error {
	if len(servers) == 0 {
		return fmt.Errorf("список DNS порожній")
	}
	if !win.IsAdmin() {
		return fmt.Errorf("потрібні права адміністратора")
	}
	var addrList []string
	for _, dns := range servers {
		addrList = append(addrList, "'"+dns+"'")
	}
	script := fmt.Sprintf(`$servers = @(%s)
Get-NetIPConfiguration | Where-Object { $_.IPv4DefaultGateway -and $_.NetAdapter.Status -eq 'Up' } | ForEach-Object {
  %s
  try {
    Set-DnsClientServerAddress -InterfaceAlias $_.InterfaceAlias -ServerAddresses $servers -ErrorAction Stop
    Write-Output ('OK|' + $_.InterfaceAlias)
  } catch { Write-Output ('FAIL|' + $_.InterfaceAlias + '|' + $_.Exception.Message) }
}`, strings.Join(addrList, ","), psTunnelFilter)

	out := runPS(script)
	var ok, failed []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(strings.TrimSpace(line), "\r")
		switch {
		case strings.HasPrefix(line, "OK|"):
			name := strings.TrimPrefix(line, "OK|")
			ok = append(ok, name)
			logf(i18n.Tr("Системний DNS {servers} застосовано на адаптері «{name}» (працює без VPN).",
				map[string]string{"servers": strings.Join(servers, ", "), "name": name}))
		case strings.HasPrefix(line, "FAIL|"):
			parts := strings.SplitN(strings.TrimPrefix(line, "FAIL|"), "|", 2)
			failed = append(failed, parts[0])
			if len(parts) == 2 && parts[1] != "" {
				logf("[WARN] " + parts[0] + ": " + parts[1])
			}
		}
	}
	if len(ok) == 0 {
		return fmt.Errorf("не вдалося змінити DNS на жодному адаптері (адаптери: %s)",
			strings.Join(append(failed, failedNamesFromAdapters(systemInternetAdapters())...), ", "))
	}
	return nil
}

// failedNamesFromAdapters — хелпер для повідомлення про помилку.
func failedNamesFromAdapters(names []string) []string { return names }

// SystemDNSReset — повертає всі інтернет-адаптери до DNS провайдера —
// варіант «Автоматично».
func SystemDNSReset() error {
	if !win.IsAdmin() {
		return fmt.Errorf("потрібні права адміністратора")
	}
	script := `Get-NetIPConfiguration | Where-Object { $_.IPv4DefaultGateway -and $_.NetAdapter.Status -eq 'Up' } | ForEach-Object {
  ` + psTunnelFilter + `
  try {
    Set-DnsClientServerAddress -InterfaceAlias $_.InterfaceAlias -ResetServerAddresses -ErrorAction Stop
    Write-Output ('OK|' + $_.InterfaceAlias)
  } catch { Write-Output ('FAIL|' + $_.InterfaceAlias) }
}`
	out := runPS(script)
	ok := false
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "OK|") {
			ok = true
			logf(i18n.Tr("DNS на адаптері «{name}» повернуто до автоматичного (провайдер).",
				map[string]string{"name": strings.TrimPrefix(strings.TrimSpace(line), "OK|")}))
		}
	}
	if !ok {
		return fmt.Errorf("не вдалося відновити автоматичний DNS")
	}
	return nil
}
