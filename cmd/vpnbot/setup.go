package main

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v3"

	"github.com/crispuscrew/vpn-setup/internal/panel"
)

const setupUnique = "setup"

// platforms drives the picker layout: order, key, and button label. The label
// (emoji + device name) is language-neutral; the step text lives in the catalog.
var platforms = []struct{ key, label string }{
	{"ios", "🍎 iOS"},
	{"android", "🤖 Android"},
	{"windows", "🪟 Windows"},
	{"macos", "💻 macOS"},
	{"linux", "🐧 Linux"},
}

// setupBtn only registers the callback endpoint; every picker button shares its
// "setup" unique, so they all route to onSetupPick.
var setupBtn = tele.Btn{Unique: setupUnique}

// setupMenu builds the inline platform picker, two buttons per row.
func setupMenu() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	var rows []tele.Row
	for i := 0; i < len(platforms); i += 2 {
		row := []tele.Btn{markup.Data(platforms[i].label, setupUnique, platforms[i].key)}
		if i+1 < len(platforms) {
			row = append(row, markup.Data(platforms[i+1].label, setupUnique, platforms[i+1].key))
		}
		rows = append(rows, markup.Row(row...))
	}
	markup.Inline(rows...)
	return markup
}

// onSetup shows the platform picker on demand.
func (a *app) onSetup(c tele.Context) error {
	return c.Send(tr(a.langOf(c)).setupChoose, setupMenu())
}

// stepsFor returns the setup instructions for a platform key in a language.
func stepsFor(l lang, key string) (string, bool) {
	steps, ok := tr(l).steps[key]
	return steps, ok
}

// onSetupPick answers a tapped platform button with that platform's instructions,
// prefixed with the user's own link when their chat is already bound.
func (a *app) onSetupPick(c tele.Context) error {
	m := tr(a.langOf(c))
	steps, ok := stepsFor(a.langOf(c), c.Data())
	if !ok {
		return c.Respond(&tele.CallbackResponse{Text: m.unknownPlatform})
	}
	if err := c.Respond(); err != nil {
		return err
	}
	steps = steps + "\n\n" + m.multiServerNote
	if link, ok := a.subURLForChat(c.Chat().ID); ok {
		steps = fmt.Sprintf(m.subLinkPrefix, link) + steps
	}
	return c.Send(steps)
}

// subURLForChat returns the subscription URL of the user bound to a chat, if any.
func (a *app) subURLForChat(chatID int64) (string, bool) {
	entry, ok := a.ledger.ByChat(chatID)
	if !ok {
		return "", false
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	client, err := panel.FromEnv(ctx)
	if err != nil {
		return "", false
	}
	user, err := client.User(ctx, entry.Username)
	if err != nil {
		return "", false
	}
	return user.SubscriptionURL, true
}
