package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
	tele "gopkg.in/telebot.v3"

	"github.com/crispuscrew/vpn-setup/internal/awg"
	"github.com/crispuscrew/vpn-setup/internal/ledger"
	"github.com/crispuscrew/vpn-setup/internal/panel"
)

const awgLocUnique = "awgloc"

// awgLocBtn registers the callback endpoint; every location button shares its
// unique and routes to onAWGPick.
var awgLocBtn = tele.Btn{Unique: awgLocUnique}

// awgConfigured reports whether any AmneziaWG nodes are wired up for the bot.
func (a *app) awgConfigured() bool { return len(a.awgNodes) > 0 && a.awgAgent != nil }

// onAWG shows the caller their AmneziaWG-capable locations. With no argument it
// acts for the account bound to this chat; an admin may pass a username to act for.
func (a *app) onAWG(c tele.Context) error {
	m := tr(a.langOf(c))
	if !a.awgConfigured() {
		return c.Send(m.awgNotConfigured)
	}
	username, ok := a.awgTarget(c)
	if !ok {
		return c.Send(m.awgClaimFirst)
	}

	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	client, err := panel.FromEnv(ctx)
	if err != nil {
		return c.Send(m.panelDown)
	}
	locations, err := a.awgLocationsFor(ctx, client, username)
	if err != nil {
		return c.Send(m.panelDown)
	}
	if len(locations) == 0 {
		return c.Send(m.awgNoLocations)
	}
	return c.Send(m.awgChoose, awgLocationMarkup(username, locations))
}

// awgTarget resolves whose config to act on: an admin's explicit username argument,
// else the account bound to this chat.
func (a *app) awgTarget(c tele.Context) (string, bool) {
	if args := c.Args(); len(args) == 1 && a.isAdmin(c) {
		return strings.ToLower(args[0]), true
	}
	if entry, ok := a.ledger.ByChat(c.Chat().ID); ok {
		return entry.Username, true
	}
	return "", false
}

// awgLocationsFor returns the locations a user may reach that also have an AWG node
// configured, so per-location access still governs AmneziaWG.
func (a *app) awgLocationsFor(ctx context.Context, client *panel.Client, username string) ([]string, error) {
	user, err := client.User(ctx, username)
	if err != nil {
		return nil, err
	}
	granted, err := grantedNames(ctx, client, user.ServiceIDs)
	if err != nil {
		return nil, err
	}
	locations := make([]string, 0, len(granted))
	for _, name := range granted {
		if _, ok := a.awgNodes[name]; ok {
			locations = append(locations, name)
		}
	}
	sort.Strings(locations)
	return locations, nil
}

func awgLocationMarkup(username string, locations []string) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0, len(locations))
	for _, loc := range locations {
		rows = append(rows, markup.Row(markup.Data("🔒 "+loc, awgLocUnique, username, loc)))
	}
	markup.Inline(rows...)
	return markup
}

// onAWGPick provisions (or re-sends) the AmneziaWG config for the tapped location.
func (a *app) onAWGPick(c tele.Context) error {
	m := tr(a.langOf(c))
	if !a.awgConfigured() {
		return c.Respond(&tele.CallbackResponse{Text: m.awgNotConfigured})
	}
	username, location, ok := parseUserLocation(c.Data())
	if !ok {
		return c.Respond(&tele.CallbackResponse{Text: m.addBadRequest})
	}
	if !a.isAdmin(c) {
		if entry, bound := a.ledger.ByChat(c.Chat().ID); !bound || entry.Username != username {
			return c.Respond(&tele.CallbackResponse{Text: m.notAuthorised})
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	client, err := panel.FromEnv(ctx)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: m.panelShort})
	}
	// Re-check the grant so a stale button can't provision a revoked location.
	host, ok := a.awgNodes[location]
	if !ok || !a.stillGranted(ctx, client, username, location) {
		return c.Respond(&tele.CallbackResponse{Text: m.awgNoLocations})
	}
	_ = c.Respond(&tele.CallbackResponse{Text: m.awgProvisioning})

	conf, err := a.provisionAWG(ctx, client, username, location, host)
	if err != nil {
		log.Printf("awg provision %s@%s: %v", username, location, err)
		return c.Send(m.awgFailed)
	}
	return a.sendAWGConfig(c, location, conf)
}

// stillGranted reports whether username currently has access to location.
func (a *app) stillGranted(ctx context.Context, client *panel.Client, username, location string) bool {
	locations, err := a.awgLocationsFor(ctx, client, username)
	if err != nil {
		return false
	}
	for _, loc := range locations {
		if loc == location {
			return true
		}
	}
	return false
}

// provisionAWG reuses the user's stored peer for a location or mints one, ensures
// it exists on the node (idempotent), and returns the rendered client config.
func (a *app) provisionAWG(ctx context.Context, _ *panel.Client, username, location, host string) (string, error) {
	// Serialise per user so a double-tap can't mint two peers for one location.
	unlock := a.lockUser(username)
	defer unlock()
	peer, ok := a.ledger.AWGPeer(username, location)
	if !ok {
		kp, err := awg.GenerateKeypair()
		if err != nil {
			return "", err
		}
		peer = ledger.AWGPeer{Username: username, Location: location, PublicKey: kp.PublicKey, PrivateKey: kp.PrivateKey}
	}
	profile, err := a.awgAgent.ServerProfile(ctx, host)
	if err != nil {
		return "", err
	}
	address, err := a.awgAgent.AddPeer(ctx, host, awg.PeerName(username, location), peer.PublicKey)
	if err != nil {
		return "", err
	}
	peer.Address = address
	if err := a.ledger.SaveAWGPeer(peer); err != nil {
		return "", err
	}
	return profile.ClientConfig(peer.PrivateKey, address), nil
}

// sendAWGConfig delivers the config as an importable .conf file plus a scannable QR.
func (a *app) sendAWGConfig(c tele.Context, location, conf string) error {
	m := tr(a.langOf(c))
	doc := &tele.Document{
		File:     tele.FromReader(strings.NewReader(conf)),
		FileName: "amneziawg-" + location + ".conf",
		Caption:  fmt.Sprintf(m.awgCaption, location),
	}
	if err := c.Send(doc); err != nil {
		return err
	}
	if png, err := qrcode.Encode(conf, qrcode.Medium, qrPixelSize); err == nil {
		return c.Send(&tele.Photo{File: tele.FromReader(bytes.NewReader(png)), Caption: m.awgQRCaption})
	}
	return nil
}

func parseUserLocation(data string) (string, string, bool) {
	parts := strings.SplitN(data, "|", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
