package panel

import (
	"context"
	"fmt"
	"os"
)

// FromEnv builds a client from VPN_PANEL_URL/USERNAME/PASSWORD and authenticates
// it. Credentials come from the environment only — never from files or flags — so
// the same contract serves both the vpn CLI and the vpnbot.
func FromEnv(ctx context.Context) (*Client, error) {
	base := os.Getenv("VPN_PANEL_URL")
	user := os.Getenv("VPN_PANEL_USERNAME")
	pass := os.Getenv("VPN_PANEL_PASSWORD")
	if base == "" || user == "" || pass == "" {
		return nil, fmt.Errorf("set VPN_PANEL_URL, VPN_PANEL_USERNAME and VPN_PANEL_PASSWORD")
	}
	client := New(base)
	if err := client.Authenticate(ctx, user, pass); err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}
	return client, nil
}
