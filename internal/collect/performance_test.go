package collect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPerformanceGovernorPaths(t *testing.T) {
	root := t.TempDir()
	paths := []string{
		"sys/devices/system/cpu/cpufreq/policy0/scaling_governor",
		"sys/devices/system/cpu/cpufreq/policy4/scaling_governor",
		"sys/class/devfreq/fb000000.gpu-panthor/governor",
		"sys/class/devfreq/fdab0000.npu/governor",
		"sys/class/devfreq/dmc/governor",
	}
	for _, path := range paths {
		fullPath := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte("ondemand"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := performanceGovernorPaths(root)
	if len(got) != len(paths) {
		t.Fatalf("want %d governor paths, got %d: %v", len(paths), len(got), got)
	}
}

func TestRestoreGovernorSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "governor")
	if err := os.WriteFile(path, []byte("performance"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := restoreGovernorSettings([]governorSetting{{path: path, value: "schedutil"}}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "schedutil" {
		t.Fatalf("want restored governor schedutil, got %q", got)
	}
}

func TestToggleMaxPerformanceAndRestore(t *testing.T) {
	root := t.TempDir()
	governorPath := filepath.Join(root, "sys/class/devfreq/fdab0000.npu/governor")
	if err := os.MkdirAll(filepath.Dir(governorPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(governorPath, []byte("rknpu_ondemand"), 0o644); err != nil {
		t.Fatal(err)
	}

	collector := New()
	collector.performanceRoot = root
	enabled, err := collector.ToggleMaxPerformance()
	if err != nil || !enabled {
		t.Fatalf("enable = %v, %v; want true, nil", enabled, err)
	}
	assertFileContent(t, governorPath, "performance")

	enabled, err = collector.ToggleMaxPerformance()
	if err != nil || enabled {
		t.Fatalf("disable = %v, %v; want false, nil", enabled, err)
	}
	assertFileContent(t, governorPath, "rknpu_ondemand")
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", path, got, want)
	}
}
