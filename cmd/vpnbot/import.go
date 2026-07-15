package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	tele "gopkg.in/telebot.v3"

	"github.com/crispuscrew/vpn-setup/internal/panel"
)

// onImport is admin-only: reconcile the ledger with the panel. Every panel user the
// bot does not yet track gets a pending claim entry and a one-time link, so users
// created outside the bot (e.g. by `vpn apply`) become deliverable and appear in
// /list. Already-tracked users are left untouched, so it is safe to re-run.
func (a *app) onImport(c tele.Context) error {
	m := tr(a.langOf(c))
	if !a.isAdmin(c) {
		return c.Send(m.notAuthorised)
	}

	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	client, err := panel.FromEnv(ctx)
	if err != nil {
		return c.Send(m.panelDown)
	}
	users, err := client.Users(ctx)
	if err != nil {
		return c.Send(m.panelDown)
	}

	names := make([]string, len(users))
	for i, user := range users {
		names[i] = user.Username
	}
	tracked := func(name string) bool {
		_, ok := a.ledger.ByUsername(name)
		return ok
	}
	done, failed := planImports(names, tracked, a.tokenFor, a.botUsername)
	for _, name := range failed {
		log.Printf("import %s: could not mint claim token", name)
	}

	if len(done) == 0 && len(failed) == 0 {
		return c.Send(m.importNone)
	}
	var out strings.Builder
	fmt.Fprintf(&out, m.importHeader, len(done))
	for _, imp := range done {
		fmt.Fprintf(&out, m.importLine, imp.Username, imp.Link)
	}
	for _, name := range failed {
		fmt.Fprintf(&out, m.importFailed, name)
	}
	return sendChunked(c, out.String())
}

// importResult pairs an imported user with the claim link to forward to them.
type importResult struct {
	Username string
	Link     string
}

// planImports mints a claim link for each username the bot does not already track
// and reports the rest as failures. It is pure over its injected lookups so the
// classify/mint logic is testable without a panel or a live ledger.
func planImports(usernames []string, tracked func(string) bool, tokenFor func(string) (string, error), botUser string) (done []importResult, failed []string) {
	for _, name := range usernames {
		if tracked(name) {
			continue
		}
		token, err := tokenFor(name)
		if err != nil {
			failed = append(failed, name)
			continue
		}
		done = append(done, importResult{Username: name, Link: fmt.Sprintf("https://t.me/%s?start=%s", botUser, token)})
	}
	sort.Slice(done, func(i, j int) bool { return done[i].Username < done[j].Username })
	sort.Strings(failed)
	return done, failed
}

// sendChunked splits a long reply on line boundaries so it stays under Telegram's
// per-message limit; a large roster of claim links can exceed it in one message.
func sendChunked(c tele.Context, text string) error {
	const maxLen = 4000 // hard limit is 4096; leave headroom
	for len(text) > maxLen {
		cut := strings.LastIndex(text[:maxLen], "\n")
		if cut <= 0 {
			cut = maxLen
		}
		if err := c.Send(text[:cut]); err != nil {
			return err
		}
		text = strings.TrimPrefix(text[cut:], "\n")
	}
	if text != "" {
		return c.Send(text)
	}
	return nil
}
