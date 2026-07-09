package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/crispuscrew/vpn-setup/internal/panel"
)

const commandTimeout = 120 * time.Second

// panelClient builds a client from VPN_PANEL_* env and authenticates it.
func panelClient(ctx context.Context) (*panel.Client, error) {
	base := os.Getenv("VPN_PANEL_URL")
	user := os.Getenv("VPN_PANEL_USERNAME")
	pass := os.Getenv("VPN_PANEL_PASSWORD")
	if base == "" || user == "" || pass == "" {
		return nil, fmt.Errorf("set VPN_PANEL_URL, VPN_PANEL_USERNAME and VPN_PANEL_PASSWORD")
	}
	client := panel.New(base)
	if err := client.Authenticate(ctx, user, pass); err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}
	return client, nil
}

// commandContext returns a context bounded by commandTimeout.
func commandContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), commandTimeout)
}
