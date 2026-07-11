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
	return c.Send("Welcome. Send the code your administrator gave you:\n/start <code>")
}

// onHelp lists the commands available to the sender (admins see more).
func (a *app) onHelp(c tele.Context) error {
	var out strings.Builder
	out.WriteString("Commands:\n")
	out.WriteString("/start <code> — claim your subscription (a bare /start re-shows it)\n")
	out.WriteString("/setup — how to connect on your device\n")
	out.WriteString("/help — show this message\n")
	if a.isAdmin(c) {
		out.WriteString("\nAdmin:\n")
		out.WriteString("/add <username> — create a user and get a one-time claim link\n")
		out.WriteString("/list — show tracked users and their delivery status\n")
		out.WriteString("/revoke <username> — rotate a user's key so their link stops working\n")
	}
	return c.Send(out.String())
}

func (a *app) claim(c tele.Context, token string) error {
	entry, first, err := a.ledger.Claim(token, c.Chat().ID)
	switch {
	case err == ledger.ErrNotFound:
		return c.Send("That code isn't valid. Check it with your administrator.")
	case err != nil:
		return err
	case !first && entry.ChatID != c.Chat().ID:
		return c.Send("That code has already been claimed.")
	}
	return a.deliver(c, entry.Username)
}

// onAdd is admin-only: create a panel user and issue a one-time claim link.
func (a *app) onAdd(c tele.Context) error {
	if !a.isAdmin(c) {
		return c.Send("Not authorised.")
	}
	args := c.Args()
	if len(args) != 1 {
		return c.Send("usage: /add <username>")
	}
	username := strings.ToLower(args[0])

	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	client, err := panel.FromEnv(ctx)
	if err != nil {
		return err
	}
	serviceID, err := a.serviceID(ctx, client)
	if err != nil {
		return c.Send(err.Error())
	}
	if _, err := client.CreateUser(ctx, username, panel.ExpireNever, []int{serviceID}, ""); err != nil {
		if _, exists := client.User(ctx, username); exists != nil {
			return c.Send("Could not create user: " + err.Error())
		}
		// User already existed — reuse it and issue a fresh claim token.
	}

	token, err := newToken()
	if err != nil {
		return err
	}
	if err := a.ledger.Add(username, token); err != nil {
		return c.Send(err.Error())
	}
	link := fmt.Sprintf("https://t.me/%s?start=%s", a.botUsername, token)
	return c.Send(fmt.Sprintf("Created %s.\nSend them this link:\n%s\n\n(or the code: %s)", username, link, token))
}

// onList is admin-only: show tracked users and their claim status.
func (a *app) onList(c tele.Context) error {
	if !a.isAdmin(c) {
		return c.Send("Not authorised.")
	}
	entries := a.ledger.List()
	if len(entries) == 0 {
		return c.Send("No users tracked yet.")
	}
	var out strings.Builder
	out.WriteString("Tracked users:\n")
	for _, entry := range entries {
		status := "unclaimed"
		if entry.Status == ledger.Delivered {
			status = fmt.Sprintf("delivered → chat %d", entry.ChatID)
		}
		fmt.Fprintf(&out, "• %s — %s\n", entry.Username, status)
	}
	return c.Send(out.String())
}

// onRevoke is admin-only: rotate a user's key so their link stops working.
func (a *app) onRevoke(c tele.Context) error {
	if !a.isAdmin(c) {
		return c.Send("Not authorised.")
	}
	args := c.Args()
	if len(args) != 1 {
		return c.Send("usage: /revoke <username>")
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
			return c.Send("No such user: " + username)
		}
		return c.Send("Revoke failed: " + err.Error())
	}
	if _, err := a.ledger.Remove(username); err != nil {
		return err
	}
	return c.Send(fmt.Sprintf("Revoked %s. Their old subscription link no longer works.", username))
}

// serviceID resolves the configured default service name to its panel id.
func (a *app) serviceID(ctx context.Context, client *panel.Client) (int, error) {
	services, err := client.Services(ctx)
	if err != nil {
		return 0, err
	}
	for _, service := range services {
		if service.Name == a.defaultService {
			return service.ID, nil
		}
	}
	return 0, fmt.Errorf("service %q not found — run `vpn apply` first", a.defaultService)
}
