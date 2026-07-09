// Command vpnbot is the Telegram delivery bot for vpn-setup. It reads each user's
// subscription URL from the Marzneshin panel API and delivers it to its owner as a
// URL + QR over Telegram, exactly once, and exposes admin commands (add/list/revoke)
// that call the panel API.
//
// This is the Phase 0 scaffold: only version/help are wired. The delivery runtime
// (Telegram long-poll, exactly-once ledger, panel client) lands in its own phase.
package main

import (
	"fmt"
	"os"

	"github.com/crispuscrew/vpn-setup/internal/buildinfo"
)

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "version", "--version", "-v":
		fmt.Printf("vpnbot %s\n", buildinfo.Version)
	case "", "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Print(`vpnbot — Telegram subscription-delivery bot for vpn-setup

usage:
  vpnbot version       print the tool version
  vpnbot help          show this help

The delivery daemon (Telegram long-poll + exactly-once delivery of each user's
subscription URL and QR) arrives in a later phase; see the workspace plan.
`)
}
