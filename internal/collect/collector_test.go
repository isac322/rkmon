package collect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFirstUsableDevfreqPath(t *testing.T) {
	root := t.TempDir()
	panthor := filepath.Join(root, "fb000000.gpu-panthor")
	mali := filepath.Join(root, "fb000000.gpu-mali")
	generic := filepath.Join(root, "fb000000.gpu")
	for _, path := range []string{panthor, mali, generic} {
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeDevfreqFiles := func(path string) {
		t.Helper()
		for name, content := range map[string]string{"load": "9@200000000Hz\n", "cur_freq": "200000000\n"} {
			if err := os.WriteFile(filepath.Join(path, name), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	writeDevfreqFiles(panthor)
	writeDevfreqFiles(mali)
	writeDevfreqFiles(generic)

	if got := firstUsableDevfreqPath(panthor, mali, generic); got != panthor {
		t.Fatalf("want preferred path %q, got %q", panthor, got)
	}
	if err := os.Remove(filepath.Join(panthor, "cur_freq")); err != nil {
		t.Fatal(err)
	}
	if got := firstUsableDevfreqPath(panthor, mali, generic); got != mali {
		t.Fatalf("want fallback path %q, got %q", mali, got)
	}
	if err := os.RemoveAll(mali); err != nil {
		t.Fatal(err)
	}
	if got := firstUsableDevfreqPath(panthor, mali, generic); got != generic {
		t.Fatalf("want generic GPU path %q, got %q", generic, got)
	}
	if err := os.Remove(filepath.Join(generic, "load")); err != nil {
		t.Fatal(err)
	}
	if got := firstUsableDevfreqPath(panthor, mali, generic); got != "" {
		t.Fatalf("want no usable path, got %q", got)
	}
}

func TestReadDevfreqUsesCurFreq(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "load"), []byte("9@200000000Hz\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cur_freq"), []byte("300000000\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	collector := New()
	t.Cleanup(collector.Close)
	got := collector.readDevfreq(root, "GPU", "gpu-thermal")
	if got.PctUsed != 9 || got.FreqHz != 300000000 {
		t.Fatalf("want load 9 and cur_freq 300000000, got %+v", got)
	}
}

func TestRGAClockName(t *testing.T) {
	tests := map[string]string{
		"rga3":       "clk_rga3_0_core",
		"rga3_core0": "clk_rga3_0_core",
		"rga3_1":     "clk_rga3_1_core",
		"rga3_core1": "clk_rga3_1_core",
		"rga2":       "clk_rga2_core",
		"unknown":    "",
	}
	for name, want := range tests {
		if got := rgaClockName(name); got != want {
			t.Errorf("rgaClockName(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestVPUClockNames(t *testing.T) {
	tests := map[string]string{
		"rkvdec-core0": "clk_rkvdec0_core",
		"rkvdec-core1": "clk_rkvdec1_core",
		"rkvenc-core0": "clk_rkvenc0_core",
		"rkvenc-core1": "clk_rkvenc1_core",
		"av1d0":        "aclk_av1",
		"jpegd0":       "aclk_jpeg_decoder",
		"jpege-core0":  "aclk_jpeg_encoder0",
		"jpege-core1":  "aclk_jpeg_encoder1",
		"jpege-core2":  "aclk_jpeg_encoder2",
		"jpege-core3":  "aclk_jpeg_encoder3",
	}
	for engine, want := range tests {
		if got := vpuClockNames[engine]; got != want {
			t.Errorf("vpuClockNames[%q] = %q, want %q", engine, got, want)
		}
	}
}
