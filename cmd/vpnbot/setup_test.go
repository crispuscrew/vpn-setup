package main

import (
	"reflect"
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestStepsForKnownPlatforms(t *testing.T) {
	for _, l := range []lang{langEN, langRU} {
		for _, platform := range platforms {
			steps, ok := stepsFor(l, platform.key)
			if !ok {
				t.Errorf("stepsFor(%d, %q): not found", l, platform.key)
			}
			if steps == "" {
				t.Errorf("stepsFor(%d, %q): empty steps", l, platform.key)
			}
		}
	}
}

func TestStepsForUnknownPlatform(t *testing.T) {
	if _, ok := stepsFor(langEN, "blackberry"); ok {
		t.Error("stepsFor unknown: want not found, got found")
	}
}

func TestPlatformKeysUnique(t *testing.T) {
	seen := make(map[string]bool, len(platforms))
	for _, platform := range platforms {
		if seen[platform.key] {
			t.Errorf("duplicate platform key %q", platform.key)
		}
		seen[platform.key] = true
	}
}

// connectMenu must expose exactly one button per platform, so no device is
// unreachable from the picker, plus one AmneziaWG button when AWG is available.
func TestConnectMenuCoversEveryPlatform(t *testing.T) {
	countButtons := func(markup *tele.ReplyMarkup) int {
		buttons := 0
		for _, row := range markup.InlineKeyboard {
			buttons += len(row)
		}
		return buttons
	}
	if got := countButtons(connectMenu(false)); got != len(platforms) {
		t.Errorf("connectMenu(false) has %d buttons, want %d", got, len(platforms))
	}
	if got := countButtons(connectMenu(true)); got != len(platforms)+1 {
		t.Errorf("connectMenu(true) has %d buttons, want %d", got, len(platforms)+1)
	}
}

func TestLangFromCode(t *testing.T) {
	cases := map[string]lang{"ru": langRU, "ru-RU": langRU, "RU": langRU, "en": langEN, "en-US": langEN, "": langEN}
	for code, want := range cases {
		if got := langFromCode(code); got != want {
			t.Errorf("langFromCode(%q) = %d, want %d", code, got, want)
		}
	}
}

// Every string field in both catalogs must be filled, so no message falls back to
// an empty string in either language.
func TestCatalogsComplete(t *testing.T) {
	for _, l := range []lang{langEN, langRU} {
		v := reflect.ValueOf(tr(l))
		typ := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if field.Kind() == reflect.String && field.String() == "" {
				t.Errorf("catalog lang %d: field %s is empty", l, typ.Field(i).Name)
			}
		}
	}
}
