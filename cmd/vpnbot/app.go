package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	qrcode "github.com/skip2/go-qrcode"
	tele "gopkg.in/telebot.v3"

	"github.com/crispuscrew/vpn-setup/internal/awg"
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
	ledger      *ledger.Ledger
	admins      map[int64]bool
	botUsername string
	// awgNodes maps a location (panel service/node name) to that node's public
	// host; awgAgent runs the node-side peer agent over SSH. Both empty when
	// AmneziaWG delivery is not configured.
	awgNodes map[string]string
	awgAgent *awg.NodeAgent
	// userLocks serialises a single user's read-modify-write operations (panel
	// service toggles, AWG provisioning, revoke) so concurrent Telegram callbacks
	// for that user can't race; different users still proceed in parallel.
	userLocks sync.Map
}

func (a *app) isAdmin(c tele.Context) bool {
	return c.Sender() != nil && a.admins[c.Sender().ID]
}

// lockUser locks the per-user mutex and returns its unlock func. telebot runs each
// update handler in its own goroutine, so a check-then-write across panel/ledger
// (e.g. mint-or-reuse an AWG peer, toggle a service) would otherwise interleave.
func (a *app) lockUser(username string) func() {
	value, _ := a.userLocks.LoadOrStore(username, &sync.Mutex{})
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	return mutex.Unlock
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
	m := tr(a.langOf(c))
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	client, err := panel.FromEnv(ctx)
	if err != nil {
		return c.Send(m.panelDown)
	}
	user, err := client.User(ctx, username)
	if err != nil {
		if panel.NotFound(err) {
			return c.Send(m.accountNotFound)
		}
		return err
	}
	png, err := qrcode.Encode(user.SubscriptionURL, qrcode.Medium, qrPixelSize)
	if err != nil {
		return err
	}
	caption := fmt.Sprintf(m.deliverCaption, user.SubscriptionURL)
	return c.Send(&tele.Photo{File: tele.FromReader(bytes.NewReader(png)), Caption: caption}, setupMenu())
}
