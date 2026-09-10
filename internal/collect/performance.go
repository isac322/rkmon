package collect

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type governorSetting struct {
	path  string
	value string
}

func (c *Collector) ToggleMaxPerformance() (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.savedGovernors) > 0 {
		return false, c.restoreGovernors()
	}

	targets := performanceGovernorPaths(c.performanceRoot)
	if len(targets) == 0 {
		return false, fmt.Errorf("no supported CPU or devfreq governors found")
	}

	saved := make([]governorSetting, 0, len(targets))
	for _, path := range targets {
		current, err := os.ReadFile(path)
		if err != nil {
			_ = restoreGovernorSettings(saved)
			return false, fmt.Errorf("read %s: %w", path, err)
		}
		value := strings.TrimSpace(string(current))
		if value == "performance" {
			continue
		}
		if err := os.WriteFile(path, []byte("performance"), 0); err != nil {
			_ = restoreGovernorSettings(saved)
			return false, fmt.Errorf("set %s: %w (run rkmon with sudo)", path, err)
		}
		saved = append(saved, governorSetting{path: path, value: value})
	}

	// Keep a sentinel when every domain was already at performance so the next
	// toggle still disables the lock without changing pre-existing settings.
	if len(saved) == 0 {
		saved = append(saved, governorSetting{})
	}
	c.savedGovernors = saved
	return true, nil
}

func performanceGovernorPaths(root string) []string {
	patterns := []string{
		rootPath(root, "/sys/devices/system/cpu/cpufreq/policy*/scaling_governor"),
		rootPath(root, gpuDevfreqPanthor+"/governor"),
		rootPath(root, gpuDevfreqVendorMali+"/governor"),
		rootPath(root, gpuDevfreqGeneric+"/governor"),
		rootPath(root, NPUDevfreq+"/governor"),
		rootPath(root, DDRDevfreq+"/governor"),
	}
	var paths []string
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		paths = append(paths, matches...)
	}
	return paths
}

func rootPath(root, abs string) string {
	if root == "" {
		return abs
	}
	return filepath.Join(root, strings.TrimPrefix(abs, "/"))
}

func (c *Collector) restoreGovernors() error {
	err := restoreGovernorSettings(c.savedGovernors)
	if err == nil {
		c.savedGovernors = nil
	}
	return err
}

func restoreGovernorSettings(settings []governorSetting) error {
	var errs []string
	for i := len(settings) - 1; i >= 0; i-- {
		setting := settings[i]
		if setting.path == "" {
			continue
		}
		if err := os.WriteFile(setting.path, []byte(setting.value), 0); err != nil {
			errs = append(errs, fmt.Sprintf("restore %s: %v", setting.path, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}
