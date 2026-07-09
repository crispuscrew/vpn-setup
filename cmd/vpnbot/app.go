package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	qrcode "github.com/skip2/go-qrcode"
	tele "gopkg.in/telebot.v3"

	"github.com/crispuscrew/vpn-setup/internal/ledger"
	"github.com/crispuscrew/vpn-setup/internal/panel"
)

const (
	opTimeout   = 30 * time.Second
	qrPixelSize = 512
	tokenBytes  = 12
)

// app holds the bot's dependencies; handlers are methods on it.
type app struct {
	ledger         *ledger.Ledger
	admins         map[int64]bool
	defaultService string
	botUsername    string
}

func (a *app) isAdmin(c tele.Context) bool {
	return c.Sender() != nil && a.admins[c.Sender().ID]
}

// newToken returns a random claim token safe to put in a t.me deep link.
func newToken() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// deliver sends username's subscription URL and its QR code to the current chat.
func (a *app) deliver(c tele.Context, username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	client, err := panel.FromEnv(ctx)
	if err != nil {
		return c.Send("The panel is unavailable right now — please try again later.")
	}
	user, err := client.User(ctx, username)
	if err != nil {
		if panel.NotFound(err) {
			return c.Send("Your account was not found. Please contact your administrator.")
		}
		return err
	}
	png, err := qrcode.Encode(user.SubscriptionURL, qrcode.Medium, qrPixelSize)
	if err != nil {
		return err
	}
	caption := "Your VPN subscription. Import this link into your client, or scan the QR:\n\n" + user.SubscriptionURL
	return c.Send(&tele.Photo{File: tele.FromReader(bytes.NewReader(png)), Caption: caption})
}
