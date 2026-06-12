package ui

import (
	"strings"
	"testing"
)

func TestRenderFooterNarrowKeysTruthful(t *testing.T) {
	s := NewStyles(true)
	l := Layout{Width: 60, Narrow: true}
	got := renderFooter(s, l, [3]int8{0, 0, 0}, DefaultSections(), 60)

	if !strings.Contains(got, "[q]quit") {
		t.Fatalf("narrow footer missing [q]quit: %q", got)
	}
	if strings.Contains(got, "[q]rate") {
		t.Fatalf("narrow footer contains misleading [q]rate: %q", got)
	}
}

func TestRenderFooterEffectiveStateMatchesVisibility(t *testing.T) {
	s := NewStyles(true)
	l := Layout{Width: 200}

	got := renderFooter(s, l, [3]int8{0, 0, 0}, DefaultSections(), 100)
	for _, key := range []string{"[i]I/O:on", "[s]Sys:on", "[k]Krn:on"} {
		if !strings.Contains(got, key) {
			t.Fatalf("auto+h=100 footer missing %q: %q", key, got)
		}
	}

	got = renderFooter(s, l, [3]int8{0, 0, 0}, DefaultSections(), 0)
	for _, key := range []string{"[i]I/O:off", "[s]Sys:off", "[k]Krn:off"} {
		if !strings.Contains(got, key) {
			t.Fatalf("auto+h=0 footer missing %q: %q", key, got)
		}
	}

	got = renderFooter(s, l, [3]int8{1, -1, 0}, DefaultSections(), 0)
	if !strings.Contains(got, "[i]I/O:on") {
		t.Fatalf("forced-on tier should show on: %q", got)
	}
	if !strings.Contains(got, "[s]Sys:off") {
		t.Fatalf("forced-off tier should show off: %q", got)
	}
}

func TestRenderFooterScreenOrderSectionsBeforeTiers(t *testing.T) {
	s := NewStyles(true)
	l := Layout{Width: 200}
	got := renderFooter(s, l, [3]int8{0, 0, 0}, DefaultSections(), 100)
	cpuIdx := strings.Index(got, "[c]CPU")
	ioIdx := strings.Index(got, "[i]I/O")
	if cpuIdx < 0 || ioIdx < 0 {
		t.Fatalf("footer missing required labels: %q", got)
	}
	if cpuIdx > ioIdx {
		t.Fatalf("expected sections before tiers: cpu@%d > io@%d :: %q", cpuIdx, ioIdx, got)
	}
}

func TestRenderFooterShowsAllSectionKeys(t *testing.T) {
	s := NewStyles(true)
	l := Layout{Width: 200}
	got := renderFooter(s, l, [3]int8{0, 0, 0}, DefaultSections(), 100)
	for _, want := range []string{"[c]CPU", "[m]MEM", "[g]GPU", "[n]NPU", "[v]VPU", "[a]RGA"} {
		if !strings.Contains(got, want) {
			t.Fatalf("footer missing section key %q: %q", want, got)
		}
	}
}
