package main

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v3"

	"github.com/crispuscrew/vpn-setup/internal/ledger"
	"github.com/crispuscrew/vpn-setup/internal/panel"
)

// onStart is the user face: /start <code> claims a subscription, a bare /start
// re-shows an already-claimed one.
func (a *app) onStart(c tele.Context) error {
	if args := c.Args(); len(args) >= 1 {
		return a.claim(c, args[0])
	}
	if entry, ok := a.ledger.ByChat(c.Chat().ID); ok {
		return a.deliver(c, entry.Username)
	}
	return c.Send(tr(a.langOf(c)).welcome)
}

// onHelp lists the commands available to the sender (admins see more).
func (a *app) onHelp(c tele.Context) error {
	m := tr(a.langOf(c))
	out := m.helpUser
	if a.isAdmin(c) {
		out += m.helpAdmin
	}
	return c.Send(out)
}

func (a *app) claim(c tele.Context, token string) error {
	m := tr(a.langOf(c))
	entry, first, err := a.ledger.Claim(token, c.Chat().ID)
	switch {
	case err == ledger.ErrNotFound:
		return c.Send(m.codeInvalid)
	case err != nil:
		return err
	case !first && entry.ChatID != c.Chat().ID:
		return c.Send(m.codeClaimed)
	}
	return a.deliver(c, entry.Username)
}

// onList is admin-only: show tracked users and their claim status.
func (a *app) onList(c tele.Context) error {
	m := tr(a.langOf(c))
	if !a.isAdmin(c) {
		return c.Send(m.notAuthorised)
	}
	entries := a.ledger.List()
	if len(entries) == 0 {
		return c.Send(m.listEmpty)
	}
	var out strings.Builder
	out.WriteString(m.listHeader)
	for _, entry := range entries {
		status := m.listUnclaimed
		if entry.Status == ledger.Delivered {
			status = fmt.Sprintf(m.listDelivered, entry.ChatID)
		}
		fmt.Fprintf(&out, m.listLine, entry.Username, status)
	}
	return c.Send(out.String())
}

// onRevoke is admin-only: rotate a user's key so their link stops working.
func (a *app) onRevoke(c tele.Context) error {
	m := tr(a.langOf(c))
	if !a.isAdmin(c) {
		return c.Send(m.notAuthorised)
	}
	args := c.Args()
	if len(args) != 1 {
		return c.Send(m.revokeUsage)
	}
	username := strings.ToLower(args[0])

	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	client, err := panel.FromEnv(ctx)
	if err != nil {
		return err
	}
	if _, err := client.RevokeSubscription(ctx, username); err != nil {
		if panel.NotFound(err) {
			return c.Send(fmt.Sprintf(m.revokeNoUser, username))
		}
		return c.Send(fmt.Sprintf(m.revokeFailed, err.Error()))
	}
	if _, err := a.ledger.Remove(username); err != nil {
		return err
	}
	return c.Send(fmt.Sprintf(m.revokeOK, username))
}
