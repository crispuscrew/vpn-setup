package main

import (
	"bytes"
	"context"
	"errors"
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

const awgStartUnique = "awgstart"

// awgStartBtn is the AmneziaWG button on the delivery/setup menu; tapping it opens
// the same location picker as the /awg command for the account bound to this chat.
var awgStartBtn = tele.Btn{Unique: awgStartUnique}

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

// onAWGMenu opens the AmneziaWG location picker from a tapped menu button, acting
// for the account bound to this chat - the button-driven twin of the /awg command.
func (a *app) onAWGMenu(c tele.Context) error {
	_ = c.Respond()
	return a.onAWG(c)
}

// awgTarget resolves whose config to act on: an admin's explicit username argument,
// else the account bound to this chat.
func (a *app) awgTarget(c tele.Context) (string, bool) {
	entry, hasBound := a.ledger.ByChat(c.Chat().ID)
	return resolveAWGTarget(c.Args(), a.isAdmin(c), entry.Username, hasBound)
}

// resolveAWGTarget picks whose config to act on from the resolved inputs. An admin's
// explicit, non-empty username argument wins; otherwise the caller's own bound account.
// A tapped menu button arrives as a callback whose Args() is [""] (one empty element),
// so the empty-arg check keeps a button tap from resolving to an empty username instead
// of falling through to the caller's account.
func resolveAWGTarget(args []string, isAdmin bool, boundUser string, hasBound bool) (string, bool) {
	if len(args) == 1 && args[0] != "" && isAdmin {
		return strings.ToLower(args[0]), true
	}
	if hasBound {
		return boundUser, true
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
	inbounds, err := client.Inbounds(ctx)
	if err != nil {
		return nil, err
	}
	return awgLocationsFrom(user.ServiceIDs, inbounds, a.awgNodes), nil
}

// awgLocationsFrom resolves a user's granted services to the AWG-capable node names.
// It maps through inbounds (service -> node) rather than matching service names to
// node names, so a multi-node service like "all" expands to its member nodes instead
// of matching nothing - otherwise a user granted only "all" would see no locations.
func awgLocationsFrom(grantedServiceIDs []int, inbounds []panel.Inbound, awgNodes map[string]string) []string {
	granted := make(map[int]bool, len(grantedServiceIDs))
	for _, id := range grantedServiceIDs {
		granted[id] = true
	}
	seen := make(map[string]bool)
	var locations []string
	for _, inbound := range inbounds {
		if !anyGranted(inbound.ServiceIDs, granted) {
			continue
		}
		node := inbound.Node.Name
		if _, ok := awgNodes[node]; ok && !seen[node] {
			seen[node] = true
			locations = append(locations, node)
		}
	}
	sort.Strings(locations)
	return locations
}

// anyGranted reports whether any of a inbound's service ids is one the user holds.
func anyGranted(serviceIDs []int, granted map[int]bool) bool {
	for _, id := range serviceIDs {
		if granted[id] {
			return true
		}
	}
	return false
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
		if errors.Is(err, errRevoked) {
			return c.Send(m.awgNoLocations)
		}
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

// errRevoked signals that a user became ineligible (untracked or no longer granted
// the location) between the pre-lock checks and provisioning - i.e. a concurrent
// /revoke won the race - so no config should be minted or sent.
var errRevoked = errors.New("user no longer eligible for this location")

// provisionAWG reuses the user's stored peer for a location or mints one, ensures
// it exists on the node (idempotent), and returns the rendered client config.
func (a *app) provisionAWG(ctx context.Context, client *panel.Client, username, location, host string) (string, error) {
	// Serialise per user so a double-tap can't mint two peers for one location, and
	// re-validate eligibility under the lock: /revoke holds this same lock while it
	// removes the ledger entry and tears down peers, so a tap that passed the pre-lock
	// checks must not re-provision a just-revoked user (which would hand them a live
	// tunnel after the admin believes they are cut off).
	unlock := a.lockUser(username)
	defer unlock()
	if _, tracked := a.ledger.ByUsername(username); !tracked {
		return "", errRevoked
	}
	if !a.stillGranted(ctx, client, username, location) {
		return "", errRevoked
	}
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
