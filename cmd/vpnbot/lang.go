package main

import (
	"log"

	tele "gopkg.in/telebot.v3"
)

const langUnique = "lang"

// langBtn registers the callback endpoint for the language picker buttons.
var langBtn = tele.Btn{Unique: langUnique}

// onLang shows a language picker; the choice is saved per chat and overrides the
// auto-detected Telegram locale.
func (a *app) onLang(c tele.Context) error {
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(
		markup.Data("🇷🇺 Русский", langUnique, "ru"),
		markup.Data("🇬🇧 English", langUnique, "en"),
	))
	return c.Send(tr(a.langOf(c)).chooseLang, markup)
}

// onLangPick saves the chosen language for the chat and confirms in that language.
func (a *app) onLangPick(c tele.Context) error {
	code := c.Data()
	if code != "ru" && code != "en" {
		return c.Respond(&tele.CallbackResponse{Text: tr(a.langOf(c)).addBadRequest})
	}
	if c.Chat() != nil {
		if err := a.ledger.SetLang(c.Chat().ID, code); err != nil {
			log.Printf("set lang for chat %d: %v", c.Chat().ID, err)
			return c.Respond(&tele.CallbackResponse{Text: tr(a.langOf(c)).panelShort})
		}
	}
	if err := c.Respond(); err != nil {
		return err
	}
	return c.Edit(tr(langFromCode(code)).langSet)
}
