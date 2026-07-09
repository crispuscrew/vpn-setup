package main

import (
	"context"
	"time"

	"github.com/crispuscrew/vpn-setup/internal/panel"
)

const commandTimeout = 120 * time.Second

// panelClient builds an authenticated client from the VPN_PANEL_* environment.
func panelClient(ctx context.Context) (*panel.Client, error) {
	return panel.FromEnv(ctx)
}

// commandContext returns a context bounded by commandTimeout.
func commandContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), commandTimeout)
}
