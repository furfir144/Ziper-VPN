package wireguard

import (
	"strings"
	"testing"
)

const sampleConf = `
# Comment line
[Interface]
PrivateKey = yAnz5h3q7Q0R0+8K7vZ0e3mS9jZ1QaPq7kYb4wG2cHA=   # inline comment
Address = 10.2.0.2/32
DNS = 1.1.1.1, 9.9.9.9
MTU = 1280

[Peer]
PublicKey = yAnz5h3q7Q0R0+8K7vZ0e3mS9jZ1QaPq7kYb4wG2cHA=
PresharedKey = yAnz5h3q7Q0R0+8K7vZ0e3mS9jZ1QaPq7kYb4wG2cHA=
Endpoint = vpn.example.com:51820
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25

[Peer]
PublicKey = yAnz5h3q7Q0R0+8K7vZ0e3mS9jZ1QaPq7kYb4wG2cHA=
Endpoint = [2001:db8::1]:51820
AllowedIPs = 10.8.0.0/24
`

func TestParseConf(t *testing.T) {
	c, err := ParseConf(sampleConf)
	if err != nil {
		t.Fatalf("ParseConf: %v", err)
	}
	if c.PrivateKey != "yAnz5h3q7Q0R0+8K7vZ0e3mS9jZ1QaPq7kYb4wG2cHA=" {
		t.Errorf("PrivateKey = %q", c.PrivateKey)
	}
	if len(c.Addresses) != 1 || c.Addresses[0] != "10.2.0.2/32" {
		t.Errorf("Addresses = %v", c.Addresses)
	}
	if len(c.DNS) != 2 {
		t.Errorf("DNS = %v", c.DNS)
	}
	if c.MTU != 1280 {
		t.Errorf("MTU = %d", c.MTU)
	}
	if len(c.Peers) != 2 {
		t.Fatalf("Peers = %d", len(c.Peers))
	}
	if c.Peers[0].Endpoint != "vpn.example.com:51820" {
		t.Errorf("Endpoint = %q", c.Peers[0].Endpoint)
	}
	if len(c.Peers[0].AllowedIPs) != 2 {
		t.Errorf("AllowedIPs = %v", c.Peers[0].AllowedIPs)
	}
	if c.Peers[0].Keepalive != 25 {
		t.Errorf("Keepalive = %d", c.Peers[0].Keepalive)
	}
	if c.Peers[1].Keepalive != 0 || len(c.Peers[1].AllowedIPs) != 1 {
		t.Errorf("Peer2 = %+v", c.Peers[1])
	}

	// UAPI
	eps := []string{"203.0.113.5:51820", "192.0.2.1:443"}
	uapi, err := buildUAPI(c, eps)
	if err != nil {
		t.Fatalf("buildUAPI: %v", err)
	}
	if strings.Contains(uapi, "\n\n") {
		t.Errorf("UAPI містить порожній рядок:\n%s", uapi)
	}
	if !strings.Contains(uapi, "private_key=") || !strings.Contains(uapi, "endpoint=203.0.113.5:51820") ||
		!strings.Contains(uapi, "allowed_ip=0.0.0.0/0") || !strings.Contains(uapi, "allowed_ip=::/0") ||
		!strings.Contains(uapi, "persistent_keepalive_interval=25") ||
		!strings.Contains(uapi, "preshared_key=") {
		t.Errorf("UAPI неповний:\n%s", uapi)
	}
	t.Logf("UAPI:\n%s", uapi)
}

func TestParseConfInvalid(t *testing.T) {
	if _, err := ParseConf("[Interface]\nAddress = 10.0.0.2/32\n"); err == nil {
		t.Error("очікували помилку без PrivateKey")
	}
	if _, err := ParseConf("[Interface]\nPrivateKey = bad!!\nAddress = 10.0.0.2/32\n[Peer]\nPublicKey = x\nAllowedIPs = 0.0.0.0/0\n"); err == nil {
		t.Error("очікували помилку з некоректним ключем")
	}
	if _, err := ParseConf(""); err == nil {
		t.Error("очікували помилку на порожньому конфізі")
	}
}

func TestResolveEndpointFormat(t *testing.T) {
	out, err := resolveEndpoint("[2001:db8::1]:51820", 0)
	if err != nil {
		t.Fatalf("resolveEndpoint: %v", err)
	}
	if out != "[2001:db8::1]:51820" {
		t.Errorf("v6 format = %q", out)
	}
	out, _ = resolveEndpoint("192.0.2.9:443", 0)
	if out != "192.0.2.9:443" {
		t.Errorf("v4 format = %q", out)
	}
}
