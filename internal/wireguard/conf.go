// Package wireguard — повноцінний WireGuard тунель на базі wireguard-go
// (golang.zx2c4.com/wireguard) з Wintun-адаптером.
// Формат профілів — стандартні .conf файли WireGuard ([Interface]/[Peer]).
package wireguard

import (
	"encoding/base64"
	"fmt"
	"net"
	"strings"
)

// Conf — розібрана конфігурація WireGuard (.conf файл).
type Conf struct {
	PrivateKey string   // base64 приватного ключа
	ListenPort int      // 0 — випадковий порт
	Addresses  []string // CIDR-адреси інтерфейсу ("10.2.0.2/32")
	DNS        []string // DNS сервери з конфіга
	MTU        int      // 0 — стандартні 1420
	Peers      []PeerConf
}

// PeerConf — секція [Peer].
type PeerConf struct {
	PublicKey    string   // base64 публічного ключа
	PresharedKey string   // base64 (порожньо — якщо не задано)
	Endpoint     string   // host:port або [host]:port
	AllowedIPs   []string // CIDR
	Keepalive    int      // persistent keepalive, сек (0 — вимкнено)
}

// base64KeyLen — довжина декодованого ключа WireGuard (32 байти).
const base64KeyLen = 32

// ParseConf — розбирає текст стандартного WireGuard .conf.
// Повертає помилку, якщо відсутні обов'язкові поля (Private key,
// PublicKey кожного піра) або вони некоректні.
func ParseConf(content string) (*Conf, error) {
	c := &Conf{}
	section := ""
	lineNo := 0
	for _, raw := range strings.Split(content, "\n") {
		lineNo++
		line := strings.TrimSpace(raw)
		// прибираємо коментарі (# та ;) — включно з «інлайн» після значення
		if i := strings.IndexAny(line, "#;"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			// кожна нова секція [Peer] — новий пір
			if section == "peer" {
				c.Peers = append(c.Peers, PeerConf{})
			}
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("рядок %d: некоректний синтаксис «%s»", lineNo, line)
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)

		switch section {
		case "interface":
			switch key {
			case "privatekey":
				if err := checkKey(value); err != nil {
					return nil, fmt.Errorf("рядок %d: PrivateKey: %w", lineNo, err)
				}
				c.PrivateKey = value
			case "listenport":
				p := 0
				if _, err := fmt.Sscanf(value, "%d", &p); err != nil || p <= 0 || p > 65535 {
					return nil, fmt.Errorf("рядок %d: некоректний ListenPort «%s»", lineNo, value)
				}
				c.ListenPort = p
			case "address":
				for _, addr := range strings.Split(value, ",") {
					a := strings.TrimSpace(addr)
					if a == "" {
						continue
					}
					if _, _, err := net.ParseCIDR(a); err != nil {
						if net.ParseIP(a) == nil {
							return nil, fmt.Errorf("рядок %d: некоректна Address «%s»", lineNo, a)
						}
						// IP без маски — вважаємо /32 або /128
						if strings.Contains(a, ":") {
							a += "/128"
						} else {
							a += "/32"
						}
					}
					c.Addresses = append(c.Addresses, a)
				}
			case "dns":
				for _, addr := range strings.Split(value, ",") {
					a := strings.TrimSpace(addr)
					if a == "" {
						continue
					}
					if net.ParseIP(a) == nil {
						// DNS може бути ім'ям хоста — пропускаємо некоректні
						continue
					}
					c.DNS = append(c.DNS, a)
				}
			case "mtu":
				p := 0
				if _, err := fmt.Sscanf(value, "%d", &p); err != nil || p < 576 || p > 65535 {
					return nil, fmt.Errorf("рядок %d: некоректний MTU «%s»", lineNo, value)
				}
				c.MTU = p
			}
		case "peer":
			if len(c.Peers) == 0 {
				break
			}
			peer := &c.Peers[len(c.Peers)-1]
			switch key {
			case "publickey":
				if err := checkKey(value); err != nil {
					return nil, fmt.Errorf("рядок %d: PublicKey: %w", lineNo, err)
				}
				peer.PublicKey = value
			case "presharedkey":
				if err := checkKey(value); err != nil {
					return nil, fmt.Errorf("рядок %d: PresharedKey: %w", lineNo, err)
				}
				peer.PresharedKey = value
			case "endpoint":
				if _, _, err := net.SplitHostPort(value); err != nil {
					return nil, fmt.Errorf("рядок %d: некоректний Endpoint «%s» (очікується host:port)", lineNo, value)
				}
				peer.Endpoint = value
			case "allowedips":
				for _, item := range strings.Split(value, ",") {
					a := strings.TrimSpace(item)
					if a == "" {
						continue
					}
					if _, _, err := net.ParseCIDR(a); err != nil {
						return nil, fmt.Errorf("рядок %d: некоректні AllowedIPs «%s»", lineNo, a)
					}
					peer.AllowedIPs = append(peer.AllowedIPs, a)
				}
			case "persistentkeepalive":
				p := 0
				if _, err := fmt.Sscanf(value, "%d", &p); err != nil || p < 0 || p > 65535 {
					return nil, fmt.Errorf("рядок %d: некоректний PersistentKeepalive «%s»", lineNo, value)
				}
				peer.Keepalive = p
			}
		}
		// невідомі ключі ігноруємо (SaveConfig, PreUp, Table тощо)
	}

	if c.PrivateKey == "" {
		return nil, fmt.Errorf("у конфізі відсутній PrivateKey (секція [Interface])")
	}
	if len(c.Addresses) == 0 {
		return nil, fmt.Errorf("у конфізі відсутній Address (секція [Interface])")
	}
	if len(c.Peers) == 0 {
		return nil, fmt.Errorf("у конфізі відсутня секція [Peer]")
	}
	for i, p := range c.Peers {
		if p.PublicKey == "" {
			return nil, fmt.Errorf("пір %d: відсутній PublicKey", i+1)
		}
		if len(p.AllowedIPs) == 0 {
			return nil, fmt.Errorf("пір %d: відсутні AllowedIPs", i+1)
		}
	}
	return c, nil
}

// checkKey — перевіряє, що значення — коректний base64 ключ WireGuard.
func checkKey(value string) error {
	b, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return fmt.Errorf("ключ має бути base64: %w", err)
	}
	if len(b) != base64KeyLen {
		return fmt.Errorf("ключ має містити %d байти, а містить %d", base64KeyLen, len(b))
	}
	return nil
}

// keyHex — конвертує base64 ключ у hex (формат UAPI).
func keyHex(b64 string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

// Summary — короткий опис конфіга для UI (перша адреса, endpoint першого
// піра, DNS, кількість пірів).
type Summary struct {
	Addresses []string `json:"addresses"`
	DNS       []string `json:"dns"`
	Endpoint  string   `json:"endpoint"`
	Peers     int      `json:"peers"`
	MTU       int      `json:"mtu"`
}

// Describe — будує Summary для відображення у інтерфейсі.
func Describe(c *Conf) Summary {
	out := Summary{Peers: len(c.Peers), MTU: c.MTU}
	out.Addresses = append(out.Addresses, c.Addresses...)
	out.DNS = append(out.DNS, c.DNS...)
	for _, p := range c.Peers {
		if p.Endpoint != "" && out.Endpoint == "" {
			out.Endpoint = p.Endpoint
			break
		}
	}
	return out
}
