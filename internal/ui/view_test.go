package ui

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/isac322/rkmon/internal/collect"
)

func TestRenderVPUAndRGAClocks(t *testing.T) {
	snapshot := &collect.Snapshot{
		VPU: collect.VPUInfo{
			Mode: "load",
			Engines: []collect.VPUEngine{
				{Name: "rkvdec-core0", LoadPct: 25, UtilPct: 12.5, ClockHz: 594_000_000},
				{Name: "av1d0", LoadPct: 10, UtilPct: 9, ClockHz: 400_000_000},
				{Name: "jpegd0", LoadPct: 8, UtilPct: 7, ClockHz: 600_000_000},
				{Name: "jpege-core0", LoadPct: 6, UtilPct: 5, ClockHz: 500_000_000},
			},
		},
		RGA: collect.RGAInfo{
			Available: true,
			Cores: []collect.RGACore{
				{Name: "rga3", LoadPct: 50, ClockHz: 750_000_000},
			},
		},
	}
	sections := [SecCount]bool{}
	sections[SecVPU] = true
	sections[SecRGA] = true

	got := Render(NewStyles(true), snapshot, time.Second, 0, 100, 0, [3]int8{}, sections, 0)
	for _, want := range []string{
		"VPU (mpp_service)", "rkvdec-core0", "594 MHz",
		"av1d0", "400 MHz", "jpegd0", "600 MHz", "jpege-core0", "500 MHz",
		"rga3", "750 MHz",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("render missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "MPP Load") {
		t.Errorf("render contains MPP Load heading:\n%s", got)
	}
}

func TestFormatClockMHzReservesColumn(t *testing.T) {
	s := NewStyles(true)
	if lipgloss.Width(formatClockMHz(s, 0)) != lipgloss.Width(formatClockMHz(s, 594_000_000)) {
		t.Fatalf("empty clock width %d, 594 MHz width %d",
			lipgloss.Width(formatClockMHz(s, 0)),
			lipgloss.Width(formatClockMHz(s, 594_000_000)))
	}
}

func TestRenderShowsMaxPerformanceInStatusRow(t *testing.T) {
	snapshot := &collect.Snapshot{
		Host: collect.HostInfo{MaxPerformance: true},
	}

	got := Render(NewStyles(true), snapshot, time.Second, 1, 100, 0, [3]int8{}, DefaultSections(), 0)
	if !strings.Contains(got, "PERF MAX") {
		t.Fatalf("active max-performance state missing from status row:\n%s", got)
	}
}
