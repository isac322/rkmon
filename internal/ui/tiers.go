package ui

import (
	"fmt"
	"strings"
)

const (
	TierAuto int8 = 0
	TierOn   int8 = 1
	TierOff  int8 = -1
)

const (
	SecCPU = iota
	SecMEM
	SecGPU
	SecNPU
	SecVPU
	SecRGA
	SecCount
)

func sectionVisible(sections [SecCount]bool, idx int) bool {
	if idx < 0 || idx >= SecCount {
		return false
	}
	return sections[idx]
}

func toggleSection(sections [SecCount]bool, idx int) [SecCount]bool {
	if idx >= 0 && idx < SecCount {
		sections[idx] = !sections[idx]
	}
	return sections
}

func DefaultSections() [SecCount]bool {
	var s [SecCount]bool
	for i := range s {
		s[i] = true
	}
	return s
}

func ParseTiersFlag(s string) ([3]int8, error) { return parseTiers(s) }

func parseTiers(s string) ([3]int8, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "":
		return [3]int8{TierAuto, TierAuto, TierAuto}, nil
	case "all":
		return [3]int8{TierOn, TierOn, TierOn}, nil
	case "none":
		return [3]int8{TierOff, TierOff, TierOff}, nil
	}
	out := [3]int8{TierOff, TierOff, TierOff}
	for _, tok := range strings.Split(s, ",") {
		switch strings.ToLower(strings.TrimSpace(tok)) {
		case "1", "i", "io", "i/o":
			out[0] = TierOn
		case "2", "s", "sys", "system":
			out[1] = TierOn
		case "3", "k", "krn", "kernel":
			out[2] = TierOn
		default:
			return [3]int8{}, fmt.Errorf("invalid tier %q (use i,s,k or 1,2,3 or all/none)", tok)
		}
	}
	return out, nil
}

func toggleTier(state int8, height, idx int) int8 {
	if tierVisible(state, height, idx) {
		return TierOff
	}
	return TierOn
}

func tierStateLabel(s int8) string {
	switch s {
	case TierOn:
		return "on"
	case TierOff:
		return "off"
	default:
		return "auto"
	}
}

func footerStateLabel(state int8, height, idx int) string {
	if tierVisible(state, height, idx) {
		return "on"
	}
	return "off"
}

func tierVisible(state int8, height, idx int) bool {
	if state == TierOn {
		return true
	}
	if state == TierOff {
		return false
	}
	thresholds := [3]int{tier1ThreshH, tier2ThreshH, tier3ThreshH}
	if idx < 0 || idx >= len(thresholds) {
		return false
	}
	return height >= thresholds[idx]
}
