// Command vpn is the operator CLI for vpn-setup. It drives the Marzneshin panel's
// REST API to reconcile the declared config-as-code (services, users) and to read
// each user's subscription URL. Host and node provisioning is done by Ansible; this
// binary owns the panel-side surface.
//
// Panel location and sudo-admin credentials come from the environment, never from
// files or flags: VPN_PANEL_URL, VPN_PANEL_USERNAME, VPN_PANEL_PASSWORD.
package main

import (
	"fmt"
	"os"

	"github.com/crispuscrew/vpn-setup/internal/buildinfo"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]

	var err error
	switch cmd {
	case "apply":
		err = runApply(args)
	case "status":
		err = runStatus(args)
	case "health":
		err = runHealth(args)
	case "sub":
		err = runSub(args)
	case "version", "--version", "-v":
		fmt.Printf("vpn %s\n", buildinfo.Version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "vpn %s: %v\n", cmd, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`vpn — operator CLI for vpn-setup (Marzneshin panel control)

usage:
  vpn apply [-f vpn.yaml]     reconcile the panel to the declared config-as-code
  vpn status                  list discovered inbounds, services, and users
  vpn health                  check panel + node health (non-zero exit if degraded)
  vpn sub <user> [--format]   print a user's subscription URL (or fetch its body)
  vpn version                 print the tool version
  vpn help                    show this help

environment (required, never in files):
  VPN_PANEL_URL               e.g. http://host:8000
  VPN_PANEL_USERNAME          sudo admin username
  VPN_PANEL_PASSWORD          sudo admin password
`)
}
