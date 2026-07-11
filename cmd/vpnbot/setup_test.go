package main

import (
	"reflect"
	"testing"
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

// setupMenu must expose exactly one button per platform, so no device is
// unreachable from the picker.
func TestSetupMenuCoversEveryPlatform(t *testing.T) {
	buttons := 0
	for _, row := range setupMenu().InlineKeyboard {
		buttons += len(row)
	}
	if buttons != len(platforms) {
		t.Errorf("setupMenu has %d buttons, want %d", buttons, len(platforms))
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
