package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v3"

	"github.com/crispuscrew/vpn-setup/internal/panel"
)

const (
	addLocUnique  = "addloc"
	addDoneUnique = "adddone"
)

// addLocBtn/addDoneBtn register the callback endpoints; each picker button shares
// the addloc unique, and the finish button the adddone unique.
var (
	addLocBtn  = tele.Btn{Unique: addLocUnique}
	addDoneBtn = tele.Btn{Unique: addDoneUnique}
)

// onAdd (admin) creates the panel user if new, then shows a location picker. The
// admin toggles which services (locations) the user may reach and taps Done to get
// the one-time claim link. A user's granted services are the source of truth, so
// the toggles need no in-memory session state.
func (a *app) onAdd(c tele.Context) error {
	m := tr(a.langOf(c))
	if !a.isAdmin(c) {
		return c.Send(m.notAuthorised)
	}
	args := c.Args()
	if len(args) != 1 {
		return c.Send(m.addUsage)
	}
	username := strings.ToLower(args[0])

	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	client, err := panel.FromEnv(ctx)
	if err != nil {
		return c.Send(m.panelDown)
	}
	if _, err := client.User(ctx, username); panel.NotFound(err) {
		if _, err := client.CreateUser(ctx, username, panel.ExpireNever, []int{}, ""); err != nil {
			return c.Send(fmt.Sprintf(m.addCreateFail, err.Error()))
		}
	} else if err != nil {
		return err
	}

	markup, err := a.locationMarkup(ctx, client, username, a.langOf(c))
	if err != nil {
		return c.Send(m.panelDown)
	}
	return c.Send(fmt.Sprintf(m.addGrantPrompt, username), markup)
}

// locationMarkup fetches the services and the user's current grants and builds the
// toggle keyboard: every service, checked when granted, plus a Done button.
func (a *app) locationMarkup(ctx context.Context, client *panel.Client, username string, l lang) (*tele.ReplyMarkup, error) {
	services, err := client.Services(ctx)
	if err != nil {
		return nil, err
	}
	user, err := client.User(ctx, username)
	if err != nil {
		return nil, err
	}
	granted := make(map[int]bool, len(user.ServiceIDs))
	for _, id := range user.ServiceIDs {
		granted[id] = true
	}
	return buildLocationMarkup(username, services, granted, tr(l).btnDone), nil
}

func buildLocationMarkup(username string, services []panel.Service, granted map[int]bool, doneLabel string) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	btns := make([]tele.Btn, 0, len(services))
	for _, svc := range services {
		label := svc.Name
		if granted[svc.ID] {
			label = "✅ " + label
		}
		btns = append(btns, markup.Data(label, addLocUnique, username, strconv.Itoa(svc.ID)))
	}
	var rows []tele.Row
	for i := 0; i < len(btns); i += 2 {
		if i+1 < len(btns) {
			rows = append(rows, markup.Row(btns[i], btns[i+1]))
		} else {
			rows = append(rows, markup.Row(btns[i]))
		}
	}
	rows = append(rows, markup.Row(markup.Data(doneLabel, addDoneUnique, username)))
	markup.Inline(rows...)
	return markup
}

// onAddToggle flips one service on the user and redraws the picker's checkmarks.
func (a *app) onAddToggle(c tele.Context) error {
	m := tr(a.langOf(c))
	if !a.isAdmin(c) {
		return c.Respond(&tele.CallbackResponse{Text: m.notAuthorised})
	}
	username, serviceID, ok := parseUserService(c.Data())
	if !ok {
		return c.Respond(&tele.CallbackResponse{Text: m.addBadRequest})
	}

	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	client, err := panel.FromEnv(ctx)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: m.panelShort})
	}
	user, err := client.User(ctx, username)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: m.addNoUser})
	}
	next, added := toggleID(user.ServiceIDs, serviceID)
	if _, err := client.UpdateUser(ctx, username, user.ExpireStrategy, next); err != nil {
		log.Printf("add toggle %s service %d: %v", username, serviceID, err)
		return c.Respond(&tele.CallbackResponse{Text: m.addUpdateFail})
	}

	if markup, err := a.locationMarkup(ctx, client, username, a.langOf(c)); err == nil {
		_ = c.Edit(markup)
	}
	if added {
		return c.Respond(&tele.CallbackResponse{Text: m.addGranted})
	}
	return c.Respond(&tele.CallbackResponse{Text: m.addRemoved})
}

// onAddDone issues the one-time claim link once at least one location is granted,
// reusing the user's existing token if they were added before.
func (a *app) onAddDone(c tele.Context) error {
	m := tr(a.langOf(c))
	if !a.isAdmin(c) {
		return c.Respond(&tele.CallbackResponse{Text: m.notAuthorised})
	}
	username := c.Data()

	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	client, err := panel.FromEnv(ctx)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: m.panelShort})
	}
	user, err := client.User(ctx, username)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: m.addNoUser})
	}
	if len(user.ServiceIDs) == 0 {
		return c.Respond(&tele.CallbackResponse{Text: m.addPickFirst})
	}

	token, err := a.tokenFor(username)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: m.panelShort})
	}
	locations, err := grantedNames(ctx, client, user.ServiceIDs)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: m.panelShort})
	}

	link := fmt.Sprintf("https://t.me/%s?start=%s", a.botUsername, token)
	_ = c.Respond()
	return c.Edit(fmt.Sprintf(m.addDone, username, strings.Join(locations, ", "), link, token), &tele.ReplyMarkup{})
}

// tokenFor returns the user's existing claim token, or mints and records a new one.
func (a *app) tokenFor(username string) (string, error) {
	if entry, ok := a.ledger.ByUsername(username); ok {
		return entry.Token, nil
	}
	token, err := newToken()
	if err != nil {
		return "", err
	}
	if err := a.ledger.Add(username, token); err != nil {
		return "", err
	}
	return token, nil
}

// grantedNames maps a user's service ids to their names for display.
func grantedNames(ctx context.Context, client *panel.Client, serviceIDs []int) ([]string, error) {
	services, err := client.Services(ctx)
	if err != nil {
		return nil, err
	}
	nameByID := make(map[int]string, len(services))
	for _, svc := range services {
		nameByID[svc.ID] = svc.Name
	}
	names := make([]string, 0, len(serviceIDs))
	for _, id := range serviceIDs {
		if name, ok := nameByID[id]; ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

func parseUserService(data string) (string, int, bool) {
	parts := strings.SplitN(data, "|", 2)
	if len(parts) != 2 {
		return "", 0, false
	}
	id, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, false
	}
	return parts[0], id, true
}

// toggleID adds id to ids if absent (returns added=true) or removes it if present.
func toggleID(ids []int, id int) ([]int, bool) {
	out := make([]int, 0, len(ids)+1)
	found := false
	for _, existing := range ids {
		if existing == id {
			found = true
			continue
		}
		out = append(out, existing)
	}
	if !found {
		out = append(out, id)
	}
	return out, !found
}
