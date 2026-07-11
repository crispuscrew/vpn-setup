package awg

import (
	"fmt"
	"strings"
)

// fullTunnel routes all client traffic through the server; AmneziaWG has no
// split-tunnel story here, and the whole point is to exit via the node.
const (
	fullTunnel      = "0.0.0.0/0, ::/0"
	keepaliveSecond = 25
	protocolAWG2    = "2"
)

// ServerProfile is the node identity and obfuscation profile returned by the
// awg-peer agent's server-profile command. Every field is needed to build a
// client config that the server will accept.
type ServerProfile struct {
	PublicKey    string `json:"public_key"`
	PresharedKey string `json:"preshared_key"`
	Endpoint     string `json:"endpoint"`
	DNS          string `json:"dns"`
	Protocol     string `json:"protocol"`
	Jc           int    `json:"jc"`
	Jmin         int    `json:"jmin"`
	Jmax         int    `json:"jmax"`
	S1           int    `json:"s1"`
	S2           int    `json:"s2"`
	S3           int    `json:"s3"`
	S4           int    `json:"s4"`
	H1           int64  `json:"h1"`
	H2           int64  `json:"h2"`
	H3           int64  `json:"h3"`
	H4           int64  `json:"h4"`
}

// ClientConfig renders the AmneziaWG .conf for a peer with private key priv and
// the host address the node assigned it (e.g. "10.8.0.5"). The [Interface]
// obfuscation values must match the server's, so they come from the profile.
func (p ServerProfile) ClientConfig(priv, address string) string {
	var b strings.Builder
	b.WriteString("[Interface]\n")
	fmt.Fprintf(&b, "PrivateKey = %s\n", priv)
	fmt.Fprintf(&b, "Address = %s/32\n", address)
	fmt.Fprintf(&b, "DNS = %s\n", p.DNS)
	fmt.Fprintf(&b, "Jc = %d\n", p.Jc)
	fmt.Fprintf(&b, "Jmin = %d\n", p.Jmin)
	fmt.Fprintf(&b, "Jmax = %d\n", p.Jmax)
	fmt.Fprintf(&b, "S1 = %d\n", p.S1)
	fmt.Fprintf(&b, "S2 = %d\n", p.S2)
	if p.Protocol == protocolAWG2 {
		fmt.Fprintf(&b, "S3 = %d\n", p.S3)
		fmt.Fprintf(&b, "S4 = %d\n", p.S4)
	}
	fmt.Fprintf(&b, "H1 = %d\n", p.H1)
	fmt.Fprintf(&b, "H2 = %d\n", p.H2)
	fmt.Fprintf(&b, "H3 = %d\n", p.H3)
	fmt.Fprintf(&b, "H4 = %d\n\n", p.H4)

	b.WriteString("[Peer]\n")
	fmt.Fprintf(&b, "PublicKey = %s\n", p.PublicKey)
	if p.PresharedKey != "" {
		fmt.Fprintf(&b, "PresharedKey = %s\n", p.PresharedKey)
	}
	fmt.Fprintf(&b, "AllowedIPs = %s\n", fullTunnel)
	fmt.Fprintf(&b, "Endpoint = %s\n", p.Endpoint)
	fmt.Fprintf(&b, "PersistentKeepalive = %d\n", keepaliveSecond)
	return b.String()
}
