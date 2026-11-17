package vpn

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Server — запис сервера VPN Gate (ідентичні поля Python-версії).
type Server struct {
	HostName          string `json:"HostName"`
	IP                string `json:"IP"`
	Score             int64  `json:"Score"`
	Ping              int    `json:"Ping"`
	Speed             int64  `json:"Speed"`
	CountryLong       string `json:"CountryLong"`
	CountryShort      string `json:"CountryShort"`
	NumVpnSessions    int    `json:"NumVpnSessions"`
	Uptime            string `json:"Uptime"`
	TotalUsers        string `json:"TotalUsers"`
	Operator          string `json:"Operator"`
	Message           string `json:"Message"`
	OpenVPNConfigB64  string `json:"OpenVPN_ConfigData_Base64"`
}

// ServerCacheFile — шлях до локального кешу серверів.
func ServerCacheFile() string {
	return filepath.Join(appDir, "config", "servers_cache.json")
}

// LoadCachedServers — читає збережений список серверів.
func LoadCachedServers() []Server {
	data, err := os.ReadFile(ServerCacheFile())
	if err != nil {
		return nil
	}
	var servers []Server
	if err := json.Unmarshal(data, &servers); err != nil || len(servers) == 0 {
		return nil
	}
	return servers
}

func saveServersCache(servers []Server) {
	_ = os.MkdirAll(filepath.Dir(ServerCacheFile()), 0o755)
	data, err := json.Marshal(servers)
	if err != nil {
		return
	}
	_ = os.WriteFile(ServerCacheFile(), data, 0o644)
}

func sortServers(servers []Server) {
	sort.SliceStable(servers, func(i, j int) bool {
		if servers[i].Score != servers[j].Score {
			return servers[i].Score > servers[j].Score
		}
		return servers[i].Ping < servers[j].Ping
	})
}

// ParseJSONServers — парсить сервери зі структури JSON дзеркала auto-ovpn.
func ParseJSONServers(data []byte) []Server {
	var root interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil
	}
	var rawList []map[string]interface{}
	switch v := root.(type) {
	case []interface{}:
		if len(v) > 0 {
			if m, ok := v[0].(map[string]interface{}); ok {
				if list, ok := m["servers"].([]interface{}); ok {
					rawList = normalizeList(list)
				}
			}
		}
	case map[string]interface{}:
		if list, ok := v["servers"].([]interface{}); ok {
			rawList = normalizeList(list)
		}
	}
	var servers []Server
	for _, s := range rawList {
		cfgB64 := getStr(s, "openvpn_configdata_base64", "OpenVPN_ConfigData_Base64", "OpenVPN_ConfigData_B64")
		if cfgB64 == "" {
			continue
		}
		servers = append(servers, Server{
			HostName:         getStr(s, "hostname", "HostName"),
			IP:               getStr(s, "ip", "IP"),
			Score:            getInt(s, "score", "Score"),
			Ping:             int(getInt(s, "ping", "Ping")),
			Speed:            getInt(s, "speed", "Speed"),
			CountryLong:      firstNonEmpty(getStr(s, "countrylong", "CountryLong"), "Unknown"),
			CountryShort:     getStr(s, "countryshort", "CountryShort"),
			NumVpnSessions:   int(getInt(s, "numvpnsessions", "NumVpnSessions")),
			Uptime:           getStr(s, "uptime", "Uptime"),
			TotalUsers:       getStr(s, "totalusers", "TotalUsers"),
			Operator:         getStr(s, "operator", "Operator"),
			Message:          getStr(s, "message", "Message"),
			OpenVPNConfigB64: cfgB64,
		})
	}
	return servers
}

func normalizeList(list []interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(list))
	for _, item := range list {
		if m, ok := item.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}
	return out
}

func getStr(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch x := v.(type) {
			case string:
				return x
			case float64:
				return fmt.Sprint(int64(x))
			}
		}
	}
	return ""
}

func getInt(m map[string]interface{}, keys ...string) int64 {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch x := v.(type) {
			case float64:
				return int64(x)
			case string:
				n, _ := strconv.ParseInt(x, 10, 64)
				return n
			}
		}
	}
	return 0
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// ParseCSVServers — парсить сервери з офіційного CSV формату VPN Gate.
func ParseCSVServers(csvText string) []Server {
	var servers []Server
	for _, line := range strings.Split(csvText, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, "*") {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		row := strings.Split(line, ",")
		if len(row) > 14 {
			score, _ := strconv.ParseInt(row[2], 10, 64)
			ping, _ := strconv.Atoi(row[3])
			speed, _ := strconv.ParseInt(row[4], 10, 64)
			sessions, _ := strconv.Atoi(row[7])
			servers = append(servers, Server{
				HostName:         row[0],
				IP:               row[1],
				Score:            score,
				Ping:             ping,
				Speed:            speed,
				CountryLong:      firstNonEmpty(row[5], "Unknown"),
				CountryShort:     row[6],
				NumVpnSessions:   sessions,
				Uptime:           row[8],
				TotalUsers:       row[9],
				Operator:         row[11],
				Message:          row[12],
				OpenVPNConfigB64: row[14],
			})
		}
	}
	return servers
}

// FetchServers — завантажує та парсить список VPN серверів:
// 1. Швидкі надійні дзеркала (GitHub / jsDelivr CDN)
// 2. Прямі ендпоінти VPN Gate CSV
// 3. Tor SOCKS5 проксі (якщо активний)
// 4. Локальний збережений кеш
func FetchServers() []Server {
	jsonMirrors := []string{
		"https://raw.githubusercontent.com/9xN/auto-ovpn/main/json/data.json",
		"https://cdn.jsdelivr.net/gh/9xN/auto-ovpn@main/json/data.json",
	}
	for _, url := range jsonMirrors {
		resp, err := fetch(url, 7*time.Second)
		if err != nil {
			logf(fmt.Sprintf("[DEBUG] Дзеркало %s недоступне: %v", url, err))
			continue
		}
		servers := ParseJSONServers(resp)
		if len(servers) > 0 {
			sortServers(servers)
			saveServersCache(servers)
			logf(fmt.Sprintf("[INFO] Завантажено %d серверів з онлайн-дзеркала [OK]", len(servers)))
			return servers
		}
	}

	csvEndpoints := []string{
		"https://www.vpngate.net/api/iphone/",
		"http://www.vpngate.net/api/iphone/",
	}
	for _, ep := range csvEndpoints {
		resp, err := fetch(ep, 20*time.Second)
		if err != nil {
			logf(fmt.Sprintf("[DEBUG] VPN Gate (%s) недоступне: %v", ep, err))
			continue
		}
		servers := ParseCSVServers(string(resp))
		if len(servers) > 0 {
			sortServers(servers)
			saveServersCache(servers)
			logf(fmt.Sprintf("[INFO] Завантажено %d серверів з VPN Gate [OK]", len(servers)))
			return servers
		}
	}

	// 3. Через Tor SOCKS5, якщо порт 9050 відкритий
	if isPortOpen("127.0.0.1", 9050, 500*time.Millisecond) {
		if resp, err := fetchViaTor("http://www.vpngate.net/api/iphone/", 15*time.Second); err == nil {
			servers := ParseCSVServers(string(resp))
			if len(servers) > 0 {
				sortServers(servers)
				saveServersCache(servers)
				logf(fmt.Sprintf("[INFO] Завантажено %d серверів через Tor [OK]", len(servers)))
				return servers
			}
		}
	}

	// 4. Локальний кеш
	cached := LoadCachedServers()
	if len(cached) > 0 {
		logf(fmt.Sprintf("[WARN] Сервери завантажено з локального кешу (%d шт.)", len(cached)))
		return cached
	}
	return nil
}

func fetch(url string, timeout time.Duration) ([]byte, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func fetchViaTor(url string, timeout time.Duration) ([]byte, error) {
	// SOCKS5 через Tor: використовуємо net.Dial через SOCKS5 proxy
	proxy := &socks5Dialer{addr: "127.0.0.1:9050"}
	client := &http.Client{Timeout: timeout, Transport: &http.Transport{Dial: proxy.Dial}}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// socks5Dialer — простий SOCKS5 dialer (без автентифікації).
type socks5Dialer struct{ addr string }

func (d *socks5Dialer) Dial(network, addr string) (net.Conn, error) {
	proxyConn, err := net.Dial("tcp", d.addr)
	if err != nil {
		return nil, err
	}
	// SOCKS5 handshake: version 5, 1 auth method, no auth
	_, err = proxyConn.Write([]byte{0x05, 0x01, 0x00})
	if err != nil {
		proxyConn.Close()
		return nil, err
	}
	buf := make([]byte, 2)
	if _, err = proxyConn.Read(buf); err != nil || buf[0] != 0x05 {
		proxyConn.Close()
		return nil, fmt.Errorf("SOCKS5 handshake failed")
	}
	// CONNECT request
	host, port, _ := splitHostPort(addr)
	req := append([]byte{0x05, 0x01, 0x00, 0x03}, byte(len(host)))
	req = append(req, host...)
	req = append(req, byte(port>>8), byte(port&0xff))
	if _, err = proxyConn.Write(req); err != nil {
		proxyConn.Close()
		return nil, err
	}
	resp := make([]byte, 10)
	if _, err = proxyConn.Read(resp); err != nil {
		proxyConn.Close()
		return nil, err
	}
	if resp[1] != 0x00 {
		proxyConn.Close()
		return nil, fmt.Errorf("SOCKS5 connect rejected")
	}
	return proxyConn, nil
}

func splitHostPort(addr string) (string, int, error) {
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return "", 0, fmt.Errorf("bad addr")
	}
	port, _ := strconv.Atoi(addr[idx+1:])
	return addr[:idx], port, nil
}

func isPortOpen(host string, port int, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// ─── IP / гео ──────────────────────────────────────────────────────────────

// IPInfo — публічна IP-адреса та геодані.
type IPInfo struct {
	IP          string `json:"ip"`
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`
	City        string `json:"city"`
	ISP         string `json:"isp"`
}

// FetchCurrentIPInfo — отримує поточну публічну IP-адресу та геодані.
func FetchCurrentIPInfo() IPInfo {
	// ip-api.com (HTTP — безкоштовний)
	if data, err := fetch("http://ip-api.com/json/?fields=status,message,country,countryCode,regionName,city,isp,query", 5*time.Second); err == nil {
		var m map[string]interface{}
		if json.Unmarshal(data, &m) == nil && m["status"] == "success" {
			return IPInfo{
				IP:          getStr(m, "query"),
				Country:     getStr(m, "country"),
				CountryCode: getStr(m, "countryCode"),
				City:        getStr(m, "city"),
				ISP:         getStr(m, "isp"),
			}
		}
	}
	if data, err := fetch("https://ipinfo.io/json", 5*time.Second); err == nil {
		var m map[string]interface{}
		if json.Unmarshal(data, &m) == nil {
			return IPInfo{
				IP:          getStr(m, "ip"),
				Country:     getStr(m, "country"),
				CountryCode: getStr(m, "country"),
				City:        getStr(m, "city"),
				ISP:         getStr(m, "org"),
			}
		}
	}
	return IPInfo{IP: "Не визначено", Country: "Не визначено"}
}

// DecodeConfig — декодує base64 конфіг сервера.
func DecodeConfig(b64 string) string {
	if b64 == "" {
		return ""
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return ""
	}
	return string(data)
}
