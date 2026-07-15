package main

import (
	"strings"

	tele "gopkg.in/telebot.v3"
)

// lang is a supported bot language.
type lang int

const (
	langEN lang = iota
	langRU
)

// langFromCode maps a Telegram/user language code to a supported language;
// anything starting with "ru" is Russian, everything else English.
func langFromCode(code string) lang {
	if strings.HasPrefix(strings.ToLower(code), "ru") {
		return langRU
	}
	return langEN
}

// langOf resolves the language for a message: a /lang override for the chat wins,
// otherwise the sender's Telegram locale, otherwise English.
func (a *app) langOf(c tele.Context) lang {
	if c.Chat() != nil {
		if code, ok := a.ledger.Lang(c.Chat().ID); ok {
			return langFromCode(code)
		}
	}
	if sender := c.Sender(); sender != nil {
		return langFromCode(sender.LanguageCode)
	}
	return langEN
}

// tr returns the message catalog for a language.
func tr(l lang) msg {
	if m, ok := catalog[l]; ok {
		return m
	}
	return catalog[langEN]
}

// msg is every user-facing string, one instance per language. Fields ending in a
// verb-like name carry %s/%d placeholders filled with fmt.Sprintf.
type msg struct {
	welcome          string
	codeInvalid      string
	codeClaimed      string
	helpUser         string
	helpAdmin        string
	notAuthorised    string
	panelDown        string
	panelShort       string
	listEmpty        string
	listHeader       string
	listUnclaimed    string
	listDelivered    string // takes chat id
	listLine         string // takes name, status
	importNone       string
	importHeader     string // takes count
	importLine       string // takes username, link
	importFailed     string // takes username
	revokeUsage      string
	revokeNoUser     string // takes username
	revokeFailed     string // takes error
	revokeOK         string // takes username
	accountNotFound  string
	deliverCaption   string // takes subscription URL
	deliverAwgNote   string // appended to delivery when AmneziaWG is available
	setupChoose      string
	unknownPlatform  string
	subLinkPrefix    string // takes link
	multiServerNote  string
	steps            map[string]string
	addUsage         string
	addCreateFail    string // takes error
	addGrantPrompt   string // takes username
	addBadRequest    string
	addNoUser        string
	addUpdateFail    string
	addGranted       string
	addRemoved       string
	addPickFirst     string
	addDone          string // takes username, locations, link, code
	btnDone          string
	chooseLang       string
	langSet          string
	awgNotConfigured string
	awgClaimFirst    string
	awgNoLocations   string
	awgChoose        string
	awgProvisioning  string
	awgFailed        string
	awgCaption       string // takes location
	awgQRCaption     string
	cmdStart         string
	cmdSetup         string
	cmdHelp          string
	cmdLang          string
	cmdAwg           string
}

var catalog = map[lang]msg{
	langEN: enMsg,
	langRU: ruMsg,
}
