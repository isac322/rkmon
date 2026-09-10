package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const preferencesVersion = 1

var (
	sectionPreferenceNames = [SecCount]string{"cpu", "memory", "gpu", "npu", "mpp", "rga"}
	tierPreferenceNames    = [3]string{"io", "system", "kernel"}
)

type displayPreferences struct {
	Version  int             `json:"version"`
	Sections map[string]bool `json:"sections"`
	Tiers    map[string]int8 `json:"tiers"`
}

func preferencesPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(configDir, "rkmon", "config.json")
}

func loadDisplayPreferences(path string, sections [SecCount]bool, tiers [3]int8) ([SecCount]bool, [3]int8) {
	if path == "" {
		return sections, tiers
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return sections, tiers
	}
	var preferences displayPreferences
	if json.Unmarshal(raw, &preferences) != nil {
		return sections, tiers
	}
	for index, name := range sectionPreferenceNames {
		if visible, ok := preferences.Sections[name]; ok {
			sections[index] = visible
		}
	}
	for index, name := range tierPreferenceNames {
		if state, ok := preferences.Tiers[name]; ok && state >= TierOff && state <= TierOn {
			tiers[index] = state
		}
	}
	return sections, tiers
}

func saveDisplayPreferences(path string, sections [SecCount]bool, tiers [3]int8) error {
	if path == "" {
		return nil
	}
	preferences := displayPreferences{
		Version:  preferencesVersion,
		Sections: make(map[string]bool, SecCount),
		Tiers:    make(map[string]int8, len(tiers)),
	}
	for index, name := range sectionPreferenceNames {
		preferences.Sections[name] = sections[index]
	}
	for index, name := range tierPreferenceNames {
		preferences.Tiers[name] = tiers[index]
	}
	raw, err := json.MarshalIndent(preferences, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if mkdirErr := os.MkdirAll(filepath.Dir(path), 0o700); mkdirErr != nil {
		return mkdirErr
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".config-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(raw); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
