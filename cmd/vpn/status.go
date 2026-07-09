package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/crispuscrew/vpn-setup/internal/panel"
)

func runStatus(args []string) error {
	ctx, cancel := commandContext()
	defer cancel()
	client, err := panelClient(ctx)
	if err != nil {
		return err
	}

	inbounds, err := client.Inbounds(ctx)
	if err != nil {
		return err
	}
	services, err := client.Services(ctx)
	if err != nil {
		return err
	}
	users, err := client.Users(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("inbounds (%d):\n", len(inbounds))
	for _, inbound := range inbounds {
		fmt.Printf("  #%-3d %-16s %s\n", inbound.ID, inbound.Tag, inbound.Protocol)
	}
	fmt.Printf("services (%d):\n", len(services))
	for _, service := range services {
		fmt.Printf("  #%-3d %-16s inbounds=%v users=%d\n", service.ID, service.Name, service.InboundIDs, len(service.UserIDs))
	}
	fmt.Printf("users (%d):\n", len(users))
	for _, user := range users {
		state := "enabled"
		if !user.Enabled {
			state = "disabled"
		}
		fmt.Printf("  %-16s %-8s services=%v\n", user.Username, state, user.ServiceIDs)
	}
	return nil
}

func runSub(args []string) error {
	// The username is the first positional; flags follow it (stdlib flag stops
	// parsing at the first non-flag token, so parse the username out by hand).
	if len(args) < 1 {
		return fmt.Errorf("usage: vpn sub <username> [--format links]")
	}
	username := args[0]
	flags := flag.NewFlagSet("sub", flag.ContinueOnError)
	format := flags.String("format", "", "fetch and print the subscription body in this format (links, sing-box, xray, clash, v2ray)")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	ctx, cancel := commandContext()
	defer cancel()
	client, err := panelClient(ctx)
	if err != nil {
		return err
	}
	user, err := client.User(ctx, username)
	if err != nil {
		if panel.NotFound(err) {
			return fmt.Errorf("no such user: %s", username)
		}
		return err
	}
	if *format == "" {
		fmt.Println(user.SubscriptionURL)
		return nil
	}
	return fetchSubscription(ctx, user.SubscriptionURL, *format)
}

// fetchSubscription prints the subscription body for one format variant.
func fetchSubscription(ctx context.Context, base, format string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/"+format, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("subscription %q returned %d", format, resp.StatusCode)
	}
	_, err = io.Copy(os.Stdout, resp.Body)
	return err
}
