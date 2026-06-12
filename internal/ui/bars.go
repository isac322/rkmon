package ui

import (
	"strings"
)

var (
	fullBlock24 = strings.Repeat("█", 24)
	dashBlock24 = strings.Repeat("─", 24)
)

func renderBar(s Styles, pct float64, width int) string {
	if width <= 0 {
		return ""
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	exact := pct * float64(width) / 100.0
	full := int(exact)
	if full > width {
		full = width
	}
	frac := exact - float64(full)

	idx := int(pct / 5)
	if idx > 20 {
		idx = 20
	}
	fillPrefix := s.pctANSI[idx]
	emptyPrefix := s.barEmptyANSI
	reset := s.ansiReset

	var sb strings.Builder
	sb.Grow(width*3 + 32)
	if full > 0 {
		sb.WriteString(fillPrefix)
		writeBlocks(&sb, "█", full)
		sb.WriteString(reset)
	}
	remaining := width - full
	if remaining > 0 {
		partial := ""
		switch {
		case frac >= 7.0/8.0:
			partial = "▉"
		case frac >= 6.0/8.0:
			partial = "▊"
		case frac >= 5.0/8.0:
			partial = "▋"
		case frac >= 4.0/8.0:
			partial = "▌"
		case frac >= 3.0/8.0:
			partial = "▍"
		case frac >= 2.0/8.0:
			partial = "▎"
		case frac >= 1.0/8.0:
			partial = "▏"
		}
		if partial != "" {
			sb.WriteString(fillPrefix)
			sb.WriteString(partial)
			sb.WriteString(reset)
			remaining--
		}
		if remaining > 0 {
			sb.WriteString(emptyPrefix)
			writeBlocks(&sb, "─", remaining)
			sb.WriteString(reset)
		}
	}
	return sb.String()
}

func writeBlocks(sb *strings.Builder, ch string, n int) {
	switch ch {
	case "█":
		if n <= 24 {
			sb.WriteString(fullBlock24[:n*len("█")])
			return
		}
	case "─":
		if n <= 24 {
			sb.WriteString(dashBlock24[:n*len("─")])
			return
		}
	}
	for i := 0; i < n; i++ {
		sb.WriteString(ch)
	}
}

func renderBarStale(s Styles, width int) string {
	if width <= 0 {
		return ""
	}
	var sb strings.Builder
	sb.Grow(width*3 + 16)
	sb.WriteString(s.staleANSI)
	for i := 0; i < width; i++ {
		sb.WriteString("▒")
	}
	sb.WriteString(s.ansiReset)
	return sb.String()
}
