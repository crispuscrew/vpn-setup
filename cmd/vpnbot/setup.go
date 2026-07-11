package main

import (
	"context"

	tele "gopkg.in/telebot.v3"

	"github.com/crispuscrew/vpn-setup/internal/panel"
)

const setupUnique = "setup"

// multiServerNote is appended to every platform's steps: the subscription carries
// all of our nodes, and clients group them so the fastest is used automatically.
const multiServerNote = `ℹ️ Using several servers
We run more than one server, and they are all in your subscription. In your app pick "Auto" / "Best Latency" (a group, not a single server) to always use the fastest one. It switches over on its own if a server goes down. You can still pick a specific server by name.`

// platforms drives the picker layout (order + labels) and holds the per-platform
// setup steps. Every client below imports a standard subscription link.
var platforms = []struct{ key, label, steps string }{
	{"ios", "🍎 iOS", `🍎 iOS setup

Recommended app: Streisand (free) or Hiddify, both on the App Store.

1. Install Streisand from the App Store.
2. Copy your subscription link (send /start if you need it again).
3. Open Streisand, tap ＋ (top-right), then "Add from Clipboard".
   Or tap ＋ → "Scan QR Code" and scan the QR from your /start message.
4. Select the config and tap Connect.`},

	{"android", "🤖 Android", `🤖 Android setup

Recommended app: Hiddify or v2rayNG (both free).

1. Install Hiddify from Google Play, or v2rayNG from GitHub.
2. Copy your subscription link (send /start if you need it again).
3. Open the app, tap ＋, then "Add from clipboard" / "Import from link".
   Or tap ＋ → "Scan QR code" and scan the QR from your /start message.
4. Tap the power button to connect.`},

	{"windows", "🪟 Windows", `🪟 Windows setup

Recommended app: Hiddify (hiddify.com) or v2rayN.

1. Download and run Hiddify for Windows.
2. Copy your subscription link (send /start if you need it again).
3. In Hiddify: New Profile → paste the link → Add.
4. Select the profile and click Connect.`},

	{"macos", "💻 macOS", `💻 macOS setup

Recommended app: Hiddify (hiddify.com), Apple Silicon and Intel.

1. Install Hiddify for macOS.
2. Copy your subscription link (send /start if you need it again).
3. In Hiddify: New Profile → paste the link → Add.
4. Select the profile and click Connect.`},

	{"linux", "🐧 Linux", `🐧 Linux setup

Recommended app: Hiddify (AppImage from hiddify.com), or the sing-box CLI.

Hiddify:
1. Download the Hiddify AppImage, make it executable, and run it.
2. New Profile → paste your subscription link → Add → Connect.

sing-box (CLI): append /sing-box to your link and run:
  curl -L "YOUR_LINK/sing-box" -o config.json
  sing-box run -c config.json
Your link is shown above, or send /start to get it.`},
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
	return c.Send("Choose your device to see setup steps:", setupMenu())
}

// stepsFor returns the setup instructions for a platform key.
func stepsFor(key string) (string, bool) {
	for _, platform := range platforms {
		if platform.key == key {
			return platform.steps, true
		}
	}
	return "", false
}

// onSetupPick answers a tapped platform button with that platform's instructions,
// prefixed with the user's own link when their chat is already bound.
func (a *app) onSetupPick(c tele.Context) error {
	steps, ok := stepsFor(c.Data())
	if !ok {
		return c.Respond(&tele.CallbackResponse{Text: "Unknown platform"})
	}
	if err := c.Respond(); err != nil {
		return err
	}
	steps = steps + "\n\n" + multiServerNote
	if link, ok := a.subURLForChat(c.Chat().ID); ok {
		steps = "Your subscription link:\n" + link + "\n\n" + steps
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
