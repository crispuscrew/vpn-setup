// Command vpnbot is the Telegram delivery bot for vpn-setup. Admins create panel
// users with /add and hand out a one-time claim link; the owner sends /start <code>
// and receives their subscription URL + QR exactly once (tracked in a durable
// ledger), re-shown on demand. /list and /revoke round out the admin face.
//
// All secrets come from the environment, never files: VPNBOT_TOKEN (from BotFather),
// VPNBOT_ADMINS (comma-separated Telegram user ids), and the VPN_PANEL_* credentials
// the panel client reads. Optional: VPNBOT_LEDGER (default /state/ledger.json),
// VPNBOT_DEFAULT_SERVICE (default "all").
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"

	"github.com/crispuscrew/vpn-setup/internal/buildinfo"
	"github.com/crispuscrew/vpn-setup/internal/ledger"
)

const defaultLedgerPath = "/state/ledger.json"

func main() {
	switch arg := firstArg(); arg {
	case "version", "--version", "-v":
		fmt.Printf("vpnbot %s\n", buildinfo.Version)
	case "help", "-h", "--help":
		usage()
	case "", "run":
		if err := run(); err != nil {
			fmt.Fprintln(os.Stderr, "vpnbot:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", arg)
		usage()
		os.Exit(2)
	}
}

func run() error {
	token := os.Getenv("VPNBOT_TOKEN")
	if token == "" {
		return fmt.Errorf("set VPNBOT_TOKEN (from @BotFather)")
	}
	admins, err := parseAdmins(os.Getenv("VPNBOT_ADMINS"))
	if err != nil {
		return err
	}
	ledgerPath := envOr("VPNBOT_LEDGER", defaultLedgerPath)
	led, err := ledger.Open(ledgerPath)
	if err != nil {
		return fmt.Errorf("open ledger: %w", err)
	}

	bot, err := tele.NewBot(tele.Settings{
		Token: token,
		// Request callback_query explicitly: with an empty list Telegram keeps the
		// token's previous allowed_updates, which may omit button taps.
		Poller: &tele.LongPoller{
			Timeout:        10 * time.Second,
			AllowedUpdates: []string{"message", "callback_query"},
		},
		OnError: func(err error, _ tele.Context) { log.Printf("handler error: %v", err) },
	})
	if err != nil {
		return fmt.Errorf("connect to Telegram: %w", err)
	}

	application := &app{
		ledger:         led,
		admins:         admins,
		defaultService: envOr("VPNBOT_DEFAULT_SERVICE", "all"),
		botUsername:    bot.Me.Username,
	}
	bot.Handle("/start", application.onStart)
	bot.Handle("/setup", application.onSetup)
	bot.Handle("/help", application.onHelp)
	bot.Handle("/add", application.onAdd)
	bot.Handle("/list", application.onList)
	bot.Handle("/revoke", application.onRevoke)
	bot.Handle(&setupBtn, application.onSetupPick)

	// Advertise the user-facing commands in Telegram's "/" menu; admin commands
	// stay unlisted (they answer "Not authorised." for everyone else).
	if err := bot.SetCommands([]tele.Command{
		{Text: "start", Description: "Claim or re-show your subscription"},
		{Text: "setup", Description: "How to connect on your device"},
		{Text: "help", Description: "Show available commands"},
	}); err != nil {
		log.Printf("set commands: %v", err)
	}

	log.Printf("vpnbot @%s ready; %d admin(s); ledger %s", bot.Me.Username, len(admins), ledgerPath)
	bot.Start()
	return nil
}

func parseAdmins(raw string) (map[int64]bool, error) {
	admins := make(map[int64]bool)
	for _, field := range strings.Split(raw, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		id, err := strconv.ParseInt(field, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("VPNBOT_ADMINS: %q is not a numeric Telegram id", field)
		}
		admins[id] = true
	}
	if len(admins) == 0 {
		return nil, fmt.Errorf("set VPNBOT_ADMINS to at least one numeric Telegram id")
	}
	return admins, nil
}

func firstArg() string {
	if len(os.Args) > 1 {
		return os.Args[1]
	}
	return ""
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func usage() {
	fmt.Print(`vpnbot — Telegram subscription-delivery bot for vpn-setup

usage:
  vpnbot            run the delivery daemon (long-polls Telegram)
  vpnbot version    print the tool version
  vpnbot help       show this help

required environment:
  VPNBOT_TOKEN              bot token from @BotFather
  VPNBOT_ADMINS             comma-separated admin Telegram user ids
  VPN_PANEL_URL/USERNAME/PASSWORD   panel API credentials
optional environment:
  VPNBOT_LEDGER             delivery ledger path (default /state/ledger.json)
  VPNBOT_DEFAULT_SERVICE    service granted to new users (default "all")
`)
}
