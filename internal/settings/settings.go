// Package settings — налаштування програми, автозапуск, власні .ovpn профілі.
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"ziper-vpn-go/internal/win"
)

// Defaults — значення за замовчуванням (ідентичні Python-версії).
var Defaults = map[string]interface{}{
	"language":            "uk",
	"autostart":           false,
	"autoconnect":         false,
	"auto_server":         true,
	"selected_country":    "All",
	"selected_server_ip":  "",
	"selected_server_name": "",
	"protocol":            "Auto",
	"dns_leak_protection": true,
	"dns_resolver":        "Cloudflare (DoH)",
	"custom_dns":          "",
	"kill_switch":         false,
	"tor_enabled":         false,
	"tor_socks_port":      9050,
	"block_ads":           true,
	"notifications":       true,
	"custom_ovpn_name":    "",
	"tor_bridges_enabled": false,
	"tor_bridges":         "",
	"wg_profile":          "",
}

var (
	ConfigDir         string
	SettingsFile      string
	CustomProfilesDir string
	WireGuardDir      string
)

// Init — визначає робочі шляхи (поруч із exe у зібраній версії).
func Init() {
	exe, err := os.Executable()
	if err != nil {
		exe, _ = os.Getwd()
	}
	dir := filepath.Dir(exe)
	ConfigDir = filepath.Join(dir, "config")
	SettingsFile = filepath.Join(ConfigDir, "settings.json")
	CustomProfilesDir = filepath.Join(ConfigDir, "profiles")
	WireGuardDir = filepath.Join(ConfigDir, "wireguard")
}

// Settings — типізовані налаштування (для біндингу у Wails).
type Settings struct {
	Language           string `json:"language"`
	Autostart          bool   `json:"autostart"`
	Autoconnect        bool   `json:"autoconnect"`
	AutoServer         bool   `json:"auto_server"`
	SelectedCountry    string `json:"selected_country"`
	SelectedServerIP   string `json:"selected_server_ip"`
	SelectedServerName string `json:"selected_server_name"`
	Protocol           string `json:"protocol"`
	DNSLeakProtection  bool   `json:"dns_leak_protection"`
	DNSResolver        string `json:"dns_resolver"`
	CustomDNS          string `json:"custom_dns"`
	KillSwitch         bool   `json:"kill_switch"`
	TorEnabled         bool   `json:"tor_enabled"`
	TorSocksPort       int    `json:"tor_socks_port"`
	BlockAds           bool   `json:"block_ads"`
	Notifications      bool   `json:"notifications"`
	CustomOvpnName     string `json:"custom_ovpn_name"`
	TorBridgesEnabled  bool   `json:"tor_bridges_enabled"`
	TorBridges         string `json:"tor_bridges"`
	WireGuardProfile   string `json:"wg_profile"`
}

// Load — читає settings.json; синхронізує autostart із реєстром Windows.
func Load() Settings {
	s := Settings{
		Language: "uk", Autoconnect: false, AutoServer: true,
		SelectedCountry: "All", Protocol: "Auto",
		DNSLeakProtection: true, DNSResolver: "Cloudflare (DoH)",
		TorSocksPort: 9050, BlockAds: true, Notifications: true,
	}
	data, err := os.ReadFile(SettingsFile)
	if err == nil {
		_ = json.Unmarshal(data, &s)
	}
	// Реєстр — джерело істини для autostart
	s.Autostart = win.IsAutostartEnabled()
	return s
}

// Save — зберігає налаштування та оновлює реєстр автозапуску.
func Save(s Settings) bool {
	_ = os.MkdirAll(ConfigDir, 0o755)
	_ = win.SetAutostart(s.Autostart)
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return false
	}
	return os.WriteFile(SettingsFile, data, 0o644) == nil
}

// ─── Власні .ovpn профілі ─────────────────────────────────────────────────

// SanitizeProfileName — прибирає шляхи та небезпечні символи з імені файлу.
func SanitizeProfileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	var b strings.Builder
	for _, ch := range name {
		ok := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') ||
			ch == ' ' || ch == '-' || ch == '_' || ch == '.'
		if ok {
			b.WriteRune(ch)
		}
	}
	safe := strings.Join(strings.Fields(b.String()), " ")
	if safe == "" {
		safe = "custom.ovpn"
	}
	if !strings.HasSuffix(strings.ToLower(safe), ".ovpn") {
		safe += ".ovpn"
	}
	if len(safe) > 80 {
		safe = safe[:80]
	}
	return safe
}

// ProfilePath — повний шлях до збереженого власного профілю.
func ProfilePath(name string) string {
	return filepath.Join(CustomProfilesDir, SanitizeProfileName(name))
}

// SaveCustomProfile — зберігає власний .ovpn конфіг; повертає збережене ім'я або "".
func SaveCustomProfile(name, content string) string {
	if err := os.MkdirAll(CustomProfilesDir, 0o755); err != nil {
		return ""
	}
	safe := SanitizeProfileName(name)
	if err := os.WriteFile(ProfilePath(safe), []byte(content), 0o644); err != nil {
		return ""
	}
	return safe
}

// ListCustomProfiles — відсортований список збережених .ovpn профілів.
func ListCustomProfiles() []string {
	entries, err := os.ReadDir(CustomProfilesDir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		n := e.Name()
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(n), ".ovpn") {
			out = append(out, n)
		}
	}
	// сортування
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if strings.ToLower(out[j]) < strings.ToLower(out[i]) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// ReadCustomProfile — читає збережений профіль.
func ReadCustomProfile(name string) (string, error) {
	data, err := os.ReadFile(ProfilePath(name))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DeleteCustomProfile — видаляє збережений профіль.
func DeleteCustomProfile(name string) bool {
	return os.Remove(ProfilePath(name)) == nil
}

// ─── WireGuard .conf профілі ──────────────────────────────────────────────

// SanitizeWGName — прибирає шляхи та небезпечні символи з імені .conf файлу.
func SanitizeWGName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	var b strings.Builder
	for _, ch := range name {
		ok := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') ||
			ch == ' ' || ch == '-' || ch == '_' || ch == '.'
		if ok {
			b.WriteRune(ch)
		}
	}
	safe := strings.Join(strings.Fields(b.String()), " ")
	if safe == "" {
		safe = "custom.conf"
	}
	if !strings.HasSuffix(strings.ToLower(safe), ".conf") {
		safe += ".conf"
	}
	if len(safe) > 80 {
		safe = safe[:80]
	}
	return safe
}

// WGProfilePath — повний шлях до збереженого WireGuard профілю.
func WGProfilePath(name string) string {
	return filepath.Join(WireGuardDir, SanitizeWGName(name))
}

// SaveWGProfile — зберігає .conf конфіг; повертає збережене ім'я або "".
func SaveWGProfile(name, content string) string {
	if err := os.MkdirAll(WireGuardDir, 0o755); err != nil {
		return ""
	}
	safe := SanitizeWGName(name)
	if err := os.WriteFile(WGProfilePath(safe), []byte(content), 0o600); err != nil {
		return ""
	}
	return safe
}

// ListWGProfiles — відсортований список збережених .conf профілів.
func ListWGProfiles() []string {
	entries, err := os.ReadDir(WireGuardDir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		n := e.Name()
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(n), ".conf") {
			out = append(out, n)
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if strings.ToLower(out[j]) < strings.ToLower(out[i]) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// ReadWGProfile — читає збережений профіль.
func ReadWGProfile(name string) (string, error) {
	data, err := os.ReadFile(WGProfilePath(name))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DeleteWGProfile — видаляє збережений профіль.
func DeleteWGProfile(name string) bool {
	return os.Remove(WGProfilePath(name)) == nil
}
