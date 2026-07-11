package main

import "testing"

func TestStepsForKnownPlatforms(t *testing.T) {
	for _, platform := range platforms {
		steps, ok := stepsFor(platform.key)
		if !ok {
			t.Errorf("stepsFor(%q): not found", platform.key)
		}
		if steps == "" {
			t.Errorf("stepsFor(%q): empty steps", platform.key)
		}
	}
}

func TestStepsForUnknownPlatform(t *testing.T) {
	if _, ok := stepsFor("blackberry"); ok {
		t.Error("stepsFor(\"blackberry\"): want not found, got found")
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
