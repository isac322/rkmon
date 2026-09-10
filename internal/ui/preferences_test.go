package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestDisplayPreferencesRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rkmon", "config.json")
	sections := DefaultSections()
	sections[SecGPU] = false
	tiers := [3]int8{TierOn, TierOff, TierAuto}

	if err := saveDisplayPreferences(path, sections, tiers); err != nil {
		t.Fatal(err)
	}
	loadedSections, loadedTiers := loadDisplayPreferences(path, DefaultSections(), [3]int8{})
	if loadedSections != sections {
		t.Fatalf("sections = %v, want %v", loadedSections, sections)
	}
	if loadedTiers != tiers {
		t.Fatalf("tiers = %v, want %v", loadedTiers, tiers)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %o, want 600", info.Mode().Perm())
	}
}

func TestModelPersistsVisibilityChanges(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	model := NewModel(nil, time.Second, false)

	updated, _ := model.Update(keyPress('c'))
	model = updated.(Model)
	_, _ = model.Update(keyPress('i'))

	reloaded := NewModel(nil, time.Second, false)
	if reloaded.sections[SecCPU] {
		t.Fatal("CPU section was not restored as hidden")
	}
	if reloaded.tiers[0] != TierOn {
		t.Fatalf("I/O tier = %d, want %d", reloaded.tiers[0], TierOn)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "rkmon", "config.json")); err != nil {
		t.Fatalf("persisted config: %v", err)
	}
}

func TestExplicitTiersOverrideSavedTiers(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := preferencesPath()
	if err := saveDisplayPreferences(path, DefaultSections(), [3]int8{TierOff, TierOff, TierOff}); err != nil {
		t.Fatal(err)
	}

	model := NewModelWithTiers(nil, time.Second, false, [3]int8{TierOn, TierOn, TierOn})
	if model.tiers != ([3]int8{TierOn, TierOn, TierOn}) {
		t.Fatalf("tiers = %v, want explicit all-on override", model.tiers)
	}
}

func keyPress(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code, Text: string(code)})
}
