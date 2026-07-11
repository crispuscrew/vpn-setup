package awg

import (
	"strings"
	"testing"
)

func sampleProfile() ServerProfile {
	return ServerProfile{
		PublicKey:    "c2VydmVycHVia2V5MDAwMDAwMDAwMDAwMDAwMDAwMDA=",
		PresharedKey: "cHNrMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDA=",
		Endpoint:     "203.0.113.7:51820",
		DNS:          "1.1.1.1, 1.0.0.1",
		Protocol:     "2",
		Jc:           4, Jmin: 99, Jmax: 1180,
		S1: 39, S2: 43, S3: 46, S4: 6,
		H1: 472878318, H2: 965718118, H3: 1293990745, H4: 1920741330,
	}
}

func TestClientConfigContainsEssentials(t *testing.T) {
	conf := sampleProfile().ClientConfig("Y2xpZW50cHJpdmtleTAwMDAwMDAwMDAwMDAwMDAwMDA=", "10.8.0.5")
	for _, want := range []string{
		"[Interface]",
		"PrivateKey = Y2xpZW50cHJpdmtleTAwMDAwMDAwMDAwMDAwMDAwMDA=",
		"Address = 10.8.0.5/32",
		"DNS = 1.1.1.1, 1.0.0.1",
		"Jc = 4", "S3 = 46", "S4 = 6", "H4 = 1920741330",
		"[Peer]",
		"PublicKey = c2VydmVycHVia2V5MDAwMDAwMDAwMDAwMDAwMDAwMDA=",
		"PresharedKey = cHNrMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDA=",
		"AllowedIPs = 0.0.0.0/0, ::/0",
		"Endpoint = 203.0.113.7:51820",
		"PersistentKeepalive = 25",
	} {
		if !strings.Contains(conf, want) {
			t.Errorf("client config missing %q\n---\n%s", want, conf)
		}
	}
}

func TestClientConfigProtocol15OmitsS3S4(t *testing.T) {
	profile := sampleProfile()
	profile.Protocol = "1.5"
	conf := profile.ClientConfig("priv", "10.8.0.5")
	if strings.Contains(conf, "S3 =") || strings.Contains(conf, "S4 =") {
		t.Errorf("protocol 1.5 must omit S3/S4\n%s", conf)
	}
	if !strings.Contains(conf, "S2 = 43") {
		t.Error("protocol 1.5 should still carry S1/S2")
	}
}

func TestClientConfigOmitsEmptyPSK(t *testing.T) {
	profile := sampleProfile()
	profile.PresharedKey = ""
	conf := profile.ClientConfig("priv", "10.8.0.5")
	if strings.Contains(conf, "PresharedKey") {
		t.Errorf("empty PSK should be omitted\n%s", conf)
	}
}
