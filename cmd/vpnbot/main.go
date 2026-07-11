// Command vpnbot is the Telegram delivery bot for vpn-setup. Admins create panel
// users with /add and hand out a one-time claim link; the owner sends /start <code>
// and receives their subscription URL + QR exactly once (tracked in a durable
// ledger), re-shown on demand. /list and /revoke round out the admin face.
//
// All secrets come from the environment, never files: VPNBOT_TOKEN (from BotFather),
// VPNBOT_ADMINS (comma-separated Telegram user ids), and the VPN_PANEL_* credentials
// the panel client reads. Optional: VPNBOT_LEDGER (default /state/ledger.json).
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"

	"github.com/crispuscrew/vpn-setup/internal/awg"
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
	awgNodes, err := parseAWGNodes(os.Getenv("VPNBOT_AWG_NODES"))
	if err != nil {
		return err
	}
	var awgAgent *awg.NodeAgent
	if len(awgNodes) > 0 {
		awgAgent = awg.NewNodeAgent(awgKeyPath(), os.Getenv("VPNBOT_SSH_USER"), os.Getenv("VPNBOT_AWG_SCRIPT"))
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
		ledger:      led,
		admins:      admins,
		botUsername: bot.Me.Username,
		awgNodes:    awgNodes,
		awgAgent:    awgAgent,
	}
	bot.Handle("/start", application.onStart)
	bot.Handle("/setup", application.onSetup)
	bot.Handle("/lang", application.onLang)
	bot.Handle("/help", application.onHelp)
	bot.Handle("/awg", application.onAWG)
	bot.Handle("/add", application.onAdd)
	bot.Handle("/list", application.onList)
	bot.Handle("/revoke", application.onRevoke)
	bot.Handle(&setupBtn, application.onSetupPick)
	bot.Handle(&langBtn, application.onLangPick)
	bot.Handle(&addLocBtn, application.onAddToggle)
	bot.Handle(&addDoneBtn, application.onAddDone)
	bot.Handle(&awgLocBtn, application.onAWGPick)

	// Advertise the user-facing commands in Telegram's "/" menu, in English by
	// default and Russian for ru users; admin commands stay unlisted (they answer
	// "Not authorised." for everyone else).
	commandsFor := func(l lang) []tele.Command {
		m := tr(l)
		cmds := []tele.Command{
			{Text: "start", Description: m.cmdStart},
			{Text: "setup", Description: m.cmdSetup},
		}
		if application.awgConfigured() {
			cmds = append(cmds, tele.Command{Text: "awg", Description: m.cmdAwg})
		}
		return append(cmds,
			tele.Command{Text: "lang", Description: m.cmdLang},
			tele.Command{Text: "help", Description: m.cmdHelp},
		)
	}
	if err := bot.SetCommands(commandsFor(langEN)); err != nil {
		log.Printf("set commands: %v", err)
	}
	if err := bot.SetCommands(commandsFor(langRU), "ru"); err != nil {
		log.Printf("set ru commands: %v", err)
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

// parseAWGNodes reads VPNBOT_AWG_NODES ("Location=host,Location=host") into a
// location→host map. Each location must match a panel service/node name.
func parseAWGNodes(raw string) (map[string]string, error) {
	nodes := make(map[string]string)
	for _, field := range strings.Split(raw, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		name, host, ok := strings.Cut(field, "=")
		name, host = strings.TrimSpace(name), strings.TrimSpace(host)
		if !ok || name == "" || host == "" {
			return nil, fmt.Errorf("VPNBOT_AWG_NODES: %q is not Location=host", field)
		}
		nodes[name] = host
	}
	return nodes, nil
}

// awgKeyPath is the SSH key the bot uses to reach the node agents, defaulting to
// the same key that deployed the nodes.
func awgKeyPath() string {
	if key := os.Getenv("VPNBOT_SSH_KEY"); key != "" {
		return key
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".ssh", "amnezia-ansible")
	}
	return ""
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
	fmt.Print(`vpnbot - Telegram subscription-delivery bot for vpn-setup

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
  VPNBOT_AWG_NODES          AmneziaWG nodes, "Location=host,Location=host"
  VPNBOT_SSH_KEY            SSH key for the node peer agents (default ~/.ssh/amnezia-ansible)
  VPNBOT_SSH_USER           SSH user for the node peer agents (default root)
  VPNBOT_AWG_SCRIPT         node peer-agent path (default /usr/local/sbin/awg-peer)
`)
}
