package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/isac322/rkmon/internal/collect"
)

// Layout encodes a responsive layout decision derived from term width.
type Layout struct {
	Width     int // outer frame width
	BarW      int // bar width used for ALL bars in the frame
	LabelW    int // label column width
	Narrow    bool
	Wide      bool
	ShortTime bool // omit seconds in footer
	TwoCol    bool // right column for Thermal + PCIe
	LeftW     int  // left box outer width when TwoCol
	RightW    int  // right box outer width when TwoCol
}

const (
	MinWidth     = 60
	TwoColThresh = 150
	TwoColRightW = 48
)

func layoutFor(width int) Layout {
	if width < MinWidth {
		width = MinWidth
	}
	l := Layout{Width: width, LabelW: 11}
	if width >= TwoColThresh {
		l.TwoCol = true
		l.RightW = TwoColRightW
		l.LeftW = width - TwoColRightW
	}
	contentW := width
	if l.TwoCol {
		contentW = l.LeftW
	}
	switch {
	case contentW < 75:
		l.Narrow = true
		l.BarW = 8
		l.LabelW = 9
		l.ShortTime = true
	case contentW < 90:
		l.Narrow = true
		l.BarW = 12
		l.ShortTime = true
	case contentW < 130:
		l.BarW = 24
	default:
		l.Wide = true
		l.BarW = 36
	}
	return l
}

const (
	tier1ThreshH = 44
	tier2ThreshH = 57
	tier3ThreshH = 60
)

func Render(s Styles, snap *collect.Snapshot, refresh time.Duration, tick, width, height int, tiers [3]int8, sections [SecCount]bool, scroll int) string {
	if width > 0 && width < MinWidth {
		return fmt.Sprintf("rkmon: terminal too narrow (%d cols); need >=%d", width, MinWidth)
	}
	if width <= 0 {
		width = 100
	}
	l := layoutFor(width)
	t1 := tierVisible(tiers[0], height, 0)
	t2 := tierVisible(tiers[1], height, 1)
	t3 := tierVisible(tiers[2], height, 2)

	leftL := l
	if l.TwoCol {
		leftL = layoutFor(l.LeftW)
	}
	leftRows := buildMainRows(s, leftL, snap, refresh, tick, tiers, sections, t1, t2, t3, !l.TwoCol)

	var rows []string
	if l.TwoCol {
		rightL := rightColLayout(l.RightW)
		rightRows := buildRightRows(s, rightL, snap)
		rows = mergeTwoCol(s, leftRows, rightRows, leftL.Width, rightL.Width)
	} else {
		rows = leftRows
	}

	total := len(rows)
	if height > 0 && total >= 2 {
		const chrome = 4
		visibleBody := height - chrome
		if visibleBody < 1 {
			visibleBody = 1
		}
		top := rows[0]
		bottom := rows[total-1]
		body := rows[1 : total-1]
		bodyLen := len(body)
		maxScroll := bodyLen - visibleBody
		if maxScroll < 0 {
			maxScroll = 0
		}
		if scroll > maxScroll {
			scroll = maxScroll
		}
		if scroll < 0 {
			scroll = 0
		}
		end := scroll + visibleBody
		if end > bodyLen {
			end = bodyLen
		}
		visible := body[scroll:end]
		rows = make([]string, 0, len(visible)+2)
		rows = append(rows, top)
		rows = append(rows, visible...)
		rows = append(rows, bottom)
	}
	rows = append(rows, renderFooter(s, l, tiers, sections, height))

	return strings.Join(rows, "\n")
}

func mainBodyLen(s Styles, snap *collect.Snapshot, refresh time.Duration, tick, width, height int, tiers [3]int8, sections [SecCount]bool) int {
	if width > 0 && width < MinWidth {
		return 0
	}
	if width <= 0 {
		width = 100
	}
	l := layoutFor(width)
	t1 := tierVisible(tiers[0], height, 0)
	t2 := tierVisible(tiers[1], height, 1)
	t3 := tierVisible(tiers[2], height, 2)
	leftL := l
	if l.TwoCol {
		leftL = layoutFor(l.LeftW)
	}
	leftRows := buildMainRows(s, leftL, snap, refresh, tick, tiers, sections, t1, t2, t3, !l.TwoCol)
	total := len(leftRows)
	if l.TwoCol {
		rightL := rightColLayout(l.RightW)
		rightRows := buildRightRows(s, rightL, snap)
		if len(rightRows) > total {
			total = len(rightRows)
		}
	}
	if total < 2 {
		return 0
	}
	return total - 2
}

func mainMaxScroll(bodyLen, height int) int {
	const chrome = 4
	visibleBody := height - chrome
	if visibleBody < 1 {
		visibleBody = 1
	}
	max := bodyLen - visibleBody
	if max < 0 {
		max = 0
	}
	return max
}

func clampMainScroll(v, bodyLen, height int) int {
	if v < 0 {
		return 0
	}
	max := mainMaxScroll(bodyLen, height)
	if v > max {
		return max
	}
	return v
}

func buildMainRows(s Styles, l Layout, snap *collect.Snapshot, refresh time.Duration, tick int, tiers [3]int8, sections [SecCount]bool, t1, t2, t3, includeRightCol bool) []string {
	var rows []string
	rows = append(rows, renderTopBorder(s, l, snap))
	rows = append(rows, renderStatusRow(s, l, snap, refresh, tick))
	if sectionVisible(sections, SecCPU) {
		rows = append(rows, renderDivider(s, l, "CPU per-core"))
		rows = append(rows, renderCPU(s, l, snap)...)
	}
	if sectionVisible(sections, SecMEM) {
		rows = append(rows, renderDivider(s, l, "Memory"))
		rows = append(rows, renderMem(s, l, snap)...)
	}
	if sectionVisible(sections, SecGPU) {
		rows = append(rows, renderDivider(s, l, "GPU"))
		rows = append(rows, renderGPU(s, l, snap)...)
	}
	if sectionVisible(sections, SecNPU) {
		rows = append(rows, renderDivider(s, l, "NPU"))
		rows = append(rows, renderNPU(s, l, snap)...)
	}
	if sectionVisible(sections, SecVPU) {
		rows = append(rows, renderDivider(s, l, "VPU (mpp_service)"))
		rows = append(rows, renderVPU(s, l, snap)...)
	}
	if sectionVisible(sections, SecRGA) {
		if snap != nil && snap.RGA.Available {
			rows = append(rows, renderDivider(s, l, "RGA (2D accel)"))
			rows = append(rows, renderRGA(s, l, snap)...)
		} else if snap != nil && !snap.Host.IsRoot {
			rows = append(rows, renderDivider(s, l, "RGA (2D accel)"))
			rows = append(rows, contentRow(s, l, s.hint("RGA load needs sudo (debugfs rkrga/load)")))
		}
	}
	if snap != nil && snap.ISP.Available {
		rows = append(rows, renderDivider(s, l, "ISP (camera pipeline)"))
		rows = append(rows, renderISP(s, l, snap))
	}
	if includeRightCol {
		rows = append(rows, renderDivider(s, l, "Thermal zones"))
		rows = append(rows, renderThermal(s, l, snap))
	}

	if t1 {
		rows = append(rows, renderDivider(s, l, "Disk I/O"))
		rows = append(rows, renderDiskIO(s, l, snap)...)
		rows = append(rows, renderDivider(s, l, "Network"))
		rows = append(rows, renderNetwork(s, l, snap)...)
		rows = append(rows, renderDivider(s, l, "Throttle (cooling devices)"))
		rows = append(rows, renderThrottle(s, l, snap))
	}
	if t2 {
		rows = append(rows, renderDivider(s, l, "CMA / Fan / Governor"))
		rows = append(rows, renderCMA(s, l, snap))
		rows = append(rows, renderFan(s, l, snap))
		rows = append(rows, renderGovernor(s, l, snap))
		if includeRightCol {
			rows = append(rows, renderDivider(s, l, "PCIe links"))
			rows = append(rows, renderPCIe(s, l, snap)...)
		}
	}
	if t3 {
		rows = append(rows, renderDivider(s, l, "Ctxsw / IRQ"))
		rows = append(rows, renderCtxIRQ(s, l, snap)...)
	}
	rows = append(rows, renderBottomBorder(s, l))
	return rows
}

func rightColLayout(width int) Layout {
	return Layout{Width: width, LabelW: 11, BarW: 8, Narrow: true, ShortTime: true}
}

func buildRightRows(s Styles, l Layout, snap *collect.Snapshot) []string {
	var rows []string
	rows = append(rows, renderSideTopBorder(s, l, "Thermal zones"))
	if snap == nil || len(snap.Thermal) == 0 {
		rows = append(rows, contentRow(s, l, s.hint("no thermal data")))
	} else {
		rows = append(rows, renderThermalVertical(s, l, snap)...)
	}
	rows = append(rows, renderDivider(s, l, "PCIe links"))
	rows = append(rows, renderPCIe(s, l, snap)...)
	rows = append(rows, renderBottomBorder(s, l))
	return rows
}

func mergeTwoCol(s Styles, left, right []string, leftW, rightW int) []string {
	if len(left) == 0 || len(right) == 0 {
		return left
	}
	leftContent := left[:len(left)-1]
	leftBottom := left[len(left)-1]
	rightContent := right[:len(right)-1]
	rightBottom := right[len(right)-1]
	blankL := s.border("│") + strings.Repeat(" ", leftW-2) + s.border("│")
	blankR := s.border("│") + strings.Repeat(" ", rightW-2) + s.border("│")
	n := len(leftContent)
	if len(rightContent) > n {
		n = len(rightContent)
	}
	out := make([]string, 0, n+1)
	for i := 0; i < n; i++ {
		l := blankL
		if i < len(leftContent) {
			l = leftContent[i]
		}
		r := blankR
		if i < len(rightContent) {
			r = rightContent[i]
		}
		out = append(out, l+r)
	}
	out = append(out, leftBottom+rightBottom)
	return out
}

func renderSideTopBorder(s Styles, l Layout, title string) string {
	titleColored := s.title(title)
	prefix := s.border("┌─ ") + titleColored + " "
	visiblePrefix := lipgloss.Width("┌─ " + title + " ")
	fill := l.Width - visiblePrefix - 1
	if fill < 1 {
		fill = 1
	}
	return prefix + s.border(strings.Repeat("─", fill)+"┐")
}

func renderThermalVertical(s Styles, l Layout, snap *collect.Snapshot) []string {
	order := []string{
		"soc-thermal", "bigcore0-thermal", "bigcore1-thermal",
		"littlecore-thermal", "center-thermal", "gpu-thermal", "npu-thermal",
	}
	var rows []string
	for _, k := range order {
		v, ok := snap.Thermal[k]
		if !ok || v == 0 {
			continue
		}
		labelText := strings.TrimSuffix(k, "-thermal")
		label := padVisible(s.label(labelText), l.LabelW)
		rows = append(rows, contentRow(s, l, label+fmtTempCell(s, v)))
	}
	return rows
}

// --- borders & rows ---------------------------------------------------------

func renderTopBorder(s Styles, l Layout, snap *collect.Snapshot) string {
	host := "?"
	kernel := "?"
	up := "?"
	if snap != nil {
		host = snap.Host.Hostname
		kernel = snap.Host.Kernel
		up = fmtUptime(snap.Host.Uptime)
	}
	var title string
	if l.Narrow {
		title = fmt.Sprintf("rkmon · %s · up %s", host, up)
	} else {
		title = fmt.Sprintf("rkmon · %s · RK3588 · %s · uptime %s", host, kernel, up)
	}
	titleColored := s.title(title)
	// "┌─ <title> " + fill "─" + "─┐"
	prefix := s.border("┌─ ") + titleColored + " "
	visiblePrefix := lipgloss.Width("┌─ "+title+" ") + 0
	fill := l.Width - visiblePrefix - 1 // 1 for the closing ┐
	if fill < 1 {
		fill = 1
	}
	return prefix + s.border(strings.Repeat("─", fill)+"┐")
}

func renderBottomBorder(s Styles, l Layout) string {
	return s.border("└" + strings.Repeat("─", l.Width-2) + "┘")
}

func renderDivider(s Styles, l Layout, title string) string {
	titleColored := s.label(title)
	prefix := s.border("├─ ") + titleColored + " "
	visiblePrefix := lipgloss.Width("├─ "+title+" ") + 0
	fill := l.Width - visiblePrefix - 1
	if fill < 1 {
		fill = 1
	}
	return prefix + s.border(strings.Repeat("─", fill)+"┤")
}

func contentRow(s Styles, l Layout, text string) string {
	inner := l.Width - 4
	w := lipgloss.Width(text)
	if w > inner {
		text = ansi.Truncate(text, inner, "…")
		w = lipgloss.Width(text)
	}
	pad := inner - w
	if pad < 0 {
		pad = 0
	}
	return s.BorderBarL + text + strings.Repeat(" ", pad) + s.BorderBarR
}

// --- status row -------------------------------------------------------------

func renderStatusRow(s Styles, l Layout, snap *collect.Snapshot, refresh time.Duration, tick int) string {
	if snap == nil {
		return contentRow(s, l, s.hint("collecting first sample..."))
	}
	la := snap.Host.LoadAvg
	loadStr := fmt.Sprintf("%.2f %.2f %.2f", la[0], la[1], la[2])
	tasks := fmt.Sprintf("%d/%d", snap.Host.ProcsRunning, snap.Host.ProcsTotal)
	left := fmt.Sprintf("%s %s  %s %s",
		s.label("Load"), s.value(loadStr),
		s.label("Tasks"), s.value(tasks))
	right := fmt.Sprintf("%s %s  %s %d",
		s.label("refresh"), s.value(refresh.String()),
		s.label("tick"), tick)
	inner := l.Width - 4
	gap := inner - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return contentRow(s, l, left+strings.Repeat(" ", gap)+right)
}

// --- CPU panel --------------------------------------------------------------

func renderCPU(s Styles, l Layout, snap *collect.Snapshot) []string {
	if snap == nil || len(snap.CPU) == 0 {
		return []string{contentRow(s, l, s.hint("no CPU data"))}
	}
	rows := make([]string, 0, len(snap.CPU))
	for _, core := range snap.CPU {
		label := fmt.Sprintf("%s cpu%d", s.cluster(core.Cluster), core.Index)
		labelPadded := padVisible(label, l.LabelW)
		bar := renderBar(s, float64(core.PctUsed), l.BarW)
		pct := pctText(s, core.PctUsed)
		freq := s.label(fmt.Sprintf("%4d MHz", core.FreqMHz))

		var temp string
		if zone, ok := collect.ClusterThermalZone[core.Index]; ok {
			temp = fmtTempCell(s, snap.Thermal[zone])
		} else {
			temp = strings.Repeat(" ", 6)
		}
		line := fmt.Sprintf("%s%s %s %s  %s", labelPadded, bar, pct, freq, temp)
		rows = append(rows, contentRow(s, l, line))
	}
	return rows
}

// --- Memory panel -----------------------------------------------------------

func renderMem(s Styles, l Layout, snap *collect.Snapshot) []string {
	if snap == nil {
		return []string{contentRow(s, l, s.hint("no mem data"))}
	}
	m := snap.Mem
	used := uint64(0)
	if m.TotalKiB > m.AvailableKiB {
		used = m.TotalKiB - m.AvailableKiB
	}
	pct := 0
	if m.TotalKiB > 0 {
		pct = int(used * 100 / m.TotalKiB)
	}
	memLine := fmt.Sprintf(
		"%s%s %s %s / %s",
		padVisible(s.label("MEM"), l.LabelW),
		renderBar(s, float64(pct), l.BarW),
		pctText(s, pct),
		s.value(fmtKiB(used)),
		s.value(fmtKiB(m.TotalKiB)),
	)
	if !l.Narrow {
		memLine += "  " + s.dim(fmt.Sprintf("(buf %s  cache %s)", fmtKiB(m.BuffersKiB), fmtKiB(m.CachedKiB)))
	}

	sused := uint64(0)
	if m.SwapTotalKiB > m.SwapFreeKiB {
		sused = m.SwapTotalKiB - m.SwapFreeKiB
	}
	spct := 0
	if m.SwapTotalKiB > 0 {
		spct = int(sused * 100 / m.SwapTotalKiB)
	}
	swapLine := fmt.Sprintf(
		"%s%s %s %s / %s",
		padVisible(s.label("SWAP"), l.LabelW),
		renderBar(s, float64(spct), l.BarW),
		pctText(s, spct),
		s.value(fmtKiB(sused)),
		s.value(fmtKiB(m.SwapTotalKiB)),
	)

	ddrLine := contentRow(s, l, accelLine(s, l, "DDR ctrl", snap.DDR, 0, false))
	return []string{contentRow(s, l, memLine), contentRow(s, l, swapLine), ddrLine}
}

// --- GPU panel -------------------------------------------------------------

func renderGPU(s Styles, l Layout, snap *collect.Snapshot) []string {
	if snap == nil {
		return []string{contentRow(s, l, s.hint("no gpu data"))}
	}
	return []string{contentRow(s, l, accelLine(s, l, "Mali-G610", snap.GPU, snap.Thermal[snap.GPU.ThermalZone], false))}
}

// --- NPU panel -------------------------------------------------------------

func renderNPU(s Styles, l Layout, snap *collect.Snapshot) []string {
	if snap == nil {
		return []string{contentRow(s, l, s.hint("no npu data"))}
	}
	rows := make([]string, 0, 5)
	if snap.NPUCores.Available && len(snap.NPUCores.Cores) > 0 {
		maxC := 0
		for _, c := range snap.NPUCores.Cores {
			if c > maxC {
				maxC = c
			}
		}
		live := collect.Devfreq{
			Name: "NPU max", PctUsed: maxC,
			FreqHz: snap.NPU.FreqHz, MinHz: snap.NPU.MinHz, MaxHz: snap.NPU.MaxHz,
			ThermalZone: "npu-thermal",
		}
		rows = append(rows, contentRow(s, l, accelLine(s, l, "NPU max ", live, snap.Thermal["npu-thermal"], false)))
		for i, c := range snap.NPUCores.Cores {
			label := padVisible(s.label(fmt.Sprintf("NPU c%d ", i)), l.LabelW)
			bar := renderBar(s, float64(c), l.BarW)
			pct := pctText(s, c)
			rows = append(rows, contentRow(s, l, fmt.Sprintf("%s%s %s", label, bar, pct)))
		}
		return rows
	}
	stale := snap.NPU.Stale
	label := "NPU agg "
	if stale {
		label = "NPU raw "
	}
	rows = append(rows, contentRow(s, l, accelLine(s, l, label, snap.NPU, snap.Thermal["npu-thermal"], stale)))
	if !l.Narrow && stale {
		rows = append(rows, contentRow(s, l, s.hint("note: rknpu_ondemand load can stick at last sample; sudo for per-core truth")))
	}
	return rows
}

func accelLine(s Styles, l Layout, label string, d collect.Devfreq, thermalMilliC int, stale bool) string {
	labelPadded := padVisible(s.label(label), l.LabelW)
	var bar string
	if stale {
		bar = renderBarStale(s, l.BarW)
	} else {
		bar = renderBar(s, float64(d.PctUsed), l.BarW)
	}
	var pct string
	if stale {
		pct = s.staleANSI + fmt.Sprintf("%3d%%", d.PctUsed) + s.ansiReset
	} else {
		pct = pctText(s, d.PctUsed)
	}
	freq := s.label(fmt.Sprintf("%4d MHz", int(d.FreqHz/1_000_000)))
	rangeStr := ""
	if !l.Narrow && d.MaxHz > 0 {
		rangeStr = " " + s.dim(fmt.Sprintf("(%d-%d)", int(d.MinHz/1_000_000), int(d.MaxHz/1_000_000)))
	}
	temp := ""
	if thermalMilliC > 0 {
		temp = "  " + fmtTempCell(s, thermalMilliC)
	}
	return fmt.Sprintf("%s%s %s %s%s%s", labelPadded, bar, pct, freq, rangeStr, temp)
}

// --- VPU panel --------------------------------------------------------------

func renderVPU(s Styles, l Layout, snap *collect.Snapshot) []string {
	if snap == nil {
		return []string{contentRow(s, l, s.hint("no vpu data"))}
	}
	v := snap.VPU
	sessionsTag := fmt.Sprintf("%s %d", s.label("sess"), v.Sessions)

	if v.Mode == "load" && len(v.Engines) > 0 {
		shown := filterVPUEngines(v.Engines)
		if l.Narrow {
			parts := []string{sessionsTag}
			for _, e := range shown {
				parts = append(parts, fmt.Sprintf("%s %s",
					shortVPUName(e.Name),
					pctText(s, int(e.LoadPct+0.5))))
			}
			return []string{contentRow(s, l, strings.Join(parts, "  "))}
		}
		labelW := vpuLabelW(shown)
		rows := []string{contentRow(s, l, sessionsTag)}
		for _, e := range shown {
			label := padVisible(s.label(e.Name), labelW)
			bar := renderBar(s, e.LoadPct, l.BarW)
			pct := pctText(s, int(e.LoadPct+0.5))
			util := s.dim(fmt.Sprintf("util %.1f%%", e.UtilPct))
			rows = append(rows, contentRow(s, l, fmt.Sprintf("%s%s %s  %s", label, bar, pct, util)))
		}
		return rows
	}

	// rates mode (sudoless fallback): show tasks/s for active engines
	parts := []string{sessionsTag}
	for _, e := range v.Engines {
		if e.TasksPerSec < 0.01 && v.Sessions == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %s",
			shortVPUName(e.Name),
			s.value(fmt.Sprintf("%.0f/s", e.TasksPerSec))))
	}
	if len(parts) == 1 {
		// nothing happening; show all engines compactly
		for _, e := range v.Engines {
			parts = append(parts, fmt.Sprintf("%s %s",
				shortVPUName(e.Name),
				s.dim("0/s")))
		}
	}
	if !l.Narrow {
		hint := "(load% needs sudo init)"
		if snap.Host.IsRoot {
			hint = "(load% initializing, will populate next tick)"
		}
		parts = append(parts, s.hint(hint))
	}
	return []string{contentRow(s, l, strings.Join(parts, "  "))}
}

func renderRGA(s Styles, l Layout, snap *collect.Snapshot) []string {
	if !snap.RGA.Available {
		return nil
	}
	if l.Narrow {
		parts := []string{}
		for _, c := range snap.RGA.Cores {
			parts = append(parts, fmt.Sprintf("%s %s",
				s.label(shortRGAName(c.Name)),
				pctText(s, c.LoadPct)))
		}
		return []string{contentRow(s, l, strings.Join(parts, "  "))}
	}
	rows := make([]string, 0, len(snap.RGA.Cores))
	for _, c := range snap.RGA.Cores {
		label := padVisible(s.label(c.Name), l.LabelW)
		bar := renderBar(s, float64(c.LoadPct), l.BarW)
		pct := pctText(s, c.LoadPct)
		rows = append(rows, contentRow(s, l, fmt.Sprintf("%s%s %s", label, bar, pct)))
	}
	return rows
}

func renderISP(s Styles, l Layout, snap *collect.Snapshot) string {
	devs := snap.ISP.Devices
	if len(devs) == 0 {
		return contentRow(s, l, s.hint("ISP available but no devices listed"))
	}
	count := fmt.Sprintf("%s %d active", s.label("nodes"), len(devs))
	if l.Narrow {
		return contentRow(s, l, count)
	}
	preview := strings.Join(devs, "  ")
	max := l.Width - 4 - lipgloss.Width(count) - 3
	if max > 0 && lipgloss.Width(preview) > max {
		preview = preview[:max] + "…"
	}
	return contentRow(s, l, count+"   "+s.dim(preview))
}

func shortRGAName(n string) string {
	switch n {
	case "rga3_core0":
		return "rga3c0"
	case "rga3_core1":
		return "rga3c1"
	case "rga2_core0", "rga2":
		return "rga2"
	default:
		return n
	}
}

func shortVPUName(n string) string {
	n = strings.TrimPrefix(n, "rkv")
	n = strings.TrimPrefix(n, "rk")
	return n
}

// filterVPUEngines: rkvdec/rkvenc always; others only when load>0 (BSP exposes 13 engines).
func filterVPUEngines(all []collect.VPUEngine) []collect.VPUEngine {
	out := make([]collect.VPUEngine, 0, len(all))
	for _, e := range all {
		base := strings.TrimRightFunc(e.Name, func(r rune) bool { return r >= '0' && r <= '9' })
		switch base {
		case "rkvdec-core", "rkvenc-core":
			out = append(out, e)
		default:
			if e.LoadPct > 0.5 || e.UtilPct > 0.5 {
				out = append(out, e)
			}
		}
	}
	return out
}

func vpuLabelW(engines []collect.VPUEngine) int {
	max := 0
	for _, e := range engines {
		if w := lipgloss.Width(e.Name); w > max {
			max = w
		}
	}
	return max + 2
}

// --- Disk I/O panel --------------------------------------------------------

func renderDiskIO(s Styles, l Layout, snap *collect.Snapshot) []string {
	if snap == nil || len(snap.Disks) == 0 {
		return []string{contentRow(s, l, s.hint("no disk samples yet"))}
	}
	rows := make([]string, 0, len(snap.Disks))
	for _, d := range snap.Disks {
		label := padVisible(s.label(d.Name), l.LabelW)
		bar := renderBar(s, d.UtilPct, l.BarW)
		pct := pctText(s, int(d.UtilPct+0.5))
		rates := fmt.Sprintf("r %5.1f  w %5.1f MB/s", d.ReadMBps, d.WriteMBps)
		rows = append(rows, contentRow(s, l, fmt.Sprintf("%s%s %s  %s", label, bar, pct, s.value(rates))))
	}
	return rows
}

// --- Network panel ----------------------------------------------------------

func renderNetwork(s Styles, l Layout, snap *collect.Snapshot) []string {
	if snap == nil || len(snap.Nets) == 0 {
		return []string{contentRow(s, l, s.hint("no net samples yet"))}
	}
	rows := make([]string, 0, len(snap.Nets))
	for _, n := range snap.Nets {
		label := padVisible(s.label(n.Iface), l.LabelW)
		rates := fmt.Sprintf("↓ %6.2f  ↑ %6.2f MB/s   total ↓%s ↑%s",
			n.RxMBps, n.TxMBps, fmtBytes(n.RxTotal), fmtBytes(n.TxTotal))
		rows = append(rows, contentRow(s, l, label+s.value(rates)))
	}
	return rows
}

// --- Throttle panel --------------------------------------------------------

func renderThrottle(s Styles, l Layout, snap *collect.Snapshot) string {
	if snap == nil || len(snap.Throttle) == 0 {
		return contentRow(s, l, s.hint("no cooling devices"))
	}
	parts := make([]string, 0, len(snap.Throttle))
	for _, d := range snap.Throttle {
		name := shortCoolingName(d.Type)
		state := fmt.Sprintf("%d/%d", d.Cur, d.Max)
		if d.Cur == 0 {
			parts = append(parts, fmt.Sprintf("%s %s", s.label(name), s.dim(state)))
		} else {
			pct := 0
			if d.Max > 0 {
				pct = d.Cur * 100 / d.Max
			}
			parts = append(parts, fmt.Sprintf("%s %s", s.label(name), pctText(s, pct)+s.dim(" ("+state+")")))
		}
	}
	return contentRow(s, l, strings.Join(parts, "  "))
}

func shortCoolingName(t string) string {
	switch {
	case strings.HasPrefix(t, "cpufreq-cpu"):
		return "cpu" + strings.TrimPrefix(t, "cpufreq-cpu")
	case strings.HasPrefix(t, "devfreq-dmc"):
		return "dmc"
	case strings.HasPrefix(t, "devfreq-") && strings.Contains(t, "gpu"):
		return "gpu"
	case strings.HasPrefix(t, "devfreq-") && strings.Contains(t, "npu"):
		return "npu"
	case strings.HasPrefix(t, "pwm-fan"), strings.HasPrefix(t, "pwmfan"):
		return "fan"
	default:
		return t
	}
}

// --- CMA panel -------------------------------------------------------------

func renderCMA(s Styles, l Layout, snap *collect.Snapshot) string {
	if snap == nil || !snap.CMA.Available {
		return contentRow(s, l, s.hint("CMA pool not exposed by kernel"))
	}
	pct := 0
	if snap.CMA.TotalKiB > 0 {
		pct = int(snap.CMA.AllocatedKiB * 100 / snap.CMA.TotalKiB)
	}
	label := padVisible(s.label("CMA"), l.LabelW)
	bar := renderBar(s, float64(pct), l.BarW)
	val := fmt.Sprintf("%s / %s", fmtKiB(snap.CMA.AllocatedKiB), fmtKiB(snap.CMA.TotalKiB))
	return contentRow(s, l, fmt.Sprintf("%s%s %s %s", label, bar, pctText(s, pct), s.value(val)))
}

// --- Fan panel -------------------------------------------------------------

func renderFan(s Styles, l Layout, snap *collect.Snapshot) string {
	if snap == nil || !snap.Fan.Available {
		return contentRow(s, l, s.hint("no PWM fan detected"))
	}
	label := padVisible(s.label("FAN PWM"), l.LabelW)
	bar := renderBar(s, float64(snap.Fan.PWMPct), l.BarW)
	val := fmt.Sprintf("%3d/255", snap.Fan.PWM)
	right := s.value(val)
	if snap.Fan.RPM > 0 {
		right += "  " + s.dim(fmt.Sprintf("%d RPM", snap.Fan.RPM))
	}
	return contentRow(s, l, fmt.Sprintf("%s%s %s %s", label, bar, pctText(s, snap.Fan.PWMPct), right))
}

// --- Governor panel --------------------------------------------------------

func renderGovernor(s Styles, l Layout, snap *collect.Snapshot) string {
	if snap == nil {
		return contentRow(s, l, s.hint("no governor data"))
	}
	g := snap.Governor
	text := fmt.Sprintf("%s %s   %s %s   %s %s",
		s.label("A55"), s.value(g.A55),
		s.label("A76-0"), s.value(g.A76_0),
		s.label("A76-1"), s.value(g.A76_1))
	return contentRow(s, l, text)
}

// --- PCIe panel ------------------------------------------------------------

func renderPCIe(s Styles, l Layout, snap *collect.Snapshot) []string {
	if snap == nil || len(snap.PCIe) == 0 {
		return []string{contentRow(s, l, s.hint("no PCIe devices"))}
	}
	rows := make([]string, 0, len(snap.PCIe))
	for _, d := range snap.PCIe {
		text := fmt.Sprintf("%s  %s x%d  %s",
			s.label(d.BusID),
			s.value(shortLinkSpeed(d.LinkSpeed)),
			d.LinkWidth,
			s.dim(pciClassShort(d.Class)))
		rows = append(rows, contentRow(s, l, text))
	}
	return rows
}

func shortLinkSpeed(s string) string {
	s = strings.TrimSuffix(s, " PCIe")
	s = strings.ReplaceAll(s, " GT/s", "GT/s")
	return s
}

func pciClassShort(c string) string {
	if len(c) < 6 {
		return c
	}
	switch c[:6] {
	case "0x0604", "060400":
		return "PCI bridge"
	case "0x0108", "010802":
		return "NVMe"
	case "0x0280", "028000":
		return "WiFi"
	case "0x0200", "020000":
		return "Ethernet"
	case "0x0c03":
		return "USB"
	}
	return c
}

// --- CtxSwitch / IRQ panel -------------------------------------------------

func renderCtxIRQ(s Styles, l Layout, snap *collect.Snapshot) []string {
	if snap == nil {
		return nil
	}
	rows := []string{}
	ctxLine := fmt.Sprintf("%s %s",
		s.label("ctxsw"), s.value(fmt.Sprintf("%d/s", snap.CtxSwitch)))
	if len(snap.IRQPerCPU) > 0 {
		var sum uint64
		for _, v := range snap.IRQPerCPU {
			sum += v
		}
		ctxLine += "   " + s.label("irq total") + " " + s.value(fmt.Sprintf("%d/s", sum))
	}
	rows = append(rows, contentRow(s, l, ctxLine))
	if len(snap.IRQPerCPU) > 0 && !l.Narrow {
		parts := make([]string, 0, len(snap.IRQPerCPU))
		for i, v := range snap.IRQPerCPU {
			parts = append(parts, fmt.Sprintf("%s %s",
				s.label(fmt.Sprintf("c%d", i)), s.value(fmt.Sprintf("%d", v))))
		}
		rows = append(rows, contentRow(s, l, strings.Join(parts, "  ")))
	}
	return rows
}

func fmtBytes(b uint64) string {
	const (
		KiB = 1024.0
		MiB = KiB * 1024.0
		GiB = MiB * 1024.0
		TiB = GiB * 1024.0
	)
	f := float64(b)
	switch {
	case f >= TiB:
		return fmt.Sprintf("%.1fT", f/TiB)
	case f >= GiB:
		return fmt.Sprintf("%.1fG", f/GiB)
	case f >= MiB:
		return fmt.Sprintf("%.0fM", f/MiB)
	case f >= KiB:
		return fmt.Sprintf("%.0fK", f/KiB)
	default:
		return fmt.Sprintf("%dB", b)
	}
}

// --- Thermal panel ----------------------------------------------------------

func renderThermal(s Styles, l Layout, snap *collect.Snapshot) string {
	if snap == nil || len(snap.Thermal) == 0 {
		return contentRow(s, l, s.hint("no thermal data"))
	}
	order := []string{
		"soc-thermal", "bigcore0-thermal", "bigcore1-thermal",
		"littlecore-thermal", "center-thermal", "gpu-thermal", "npu-thermal",
	}
	// Stable iteration also for any extra zones we don't know about.
	extras := []string{}
	for k := range snap.Thermal {
		known := false
		for _, o := range order {
			if o == k {
				known = true
				break
			}
		}
		if !known {
			extras = append(extras, k)
		}
	}
	sort.Strings(extras)
	all := append(order, extras...)

	var parts []string
	for _, k := range all {
		v, ok := snap.Thermal[k]
		if !ok || v == 0 {
			continue
		}
		short := thermalShort(k)
		if l.Narrow {
			parts = append(parts, fmt.Sprintf("%s %s", s.label(short), fmtTempCell(s, v)))
		} else {
			parts = append(parts, fmt.Sprintf("%s %s", s.label(strings.TrimSuffix(k, "-thermal")), fmtTempCell(s, v)))
		}
	}
	sep := "  "
	if l.Narrow {
		sep = " "
	}
	return contentRow(s, l, strings.Join(parts, sep))
}

func thermalShort(k string) string {
	switch k {
	case "soc-thermal":
		return "SoC"
	case "bigcore0-thermal":
		return "B0"
	case "bigcore1-thermal":
		return "B1"
	case "littlecore-thermal":
		return "L"
	case "center-thermal":
		return "Cen"
	case "gpu-thermal":
		return "G"
	case "npu-thermal":
		return "N"
	default:
		return strings.TrimSuffix(k, "-thermal")
	}
}

// --- Footer (outside the box) ----------------------------------------------

func renderFooter(s Styles, l Layout, tiers [3]int8, sections [SecCount]bool, height int) string {
	secEntries := []struct {
		key  string
		name string
		idx  int
	}{
		{"[c]", "CPU", SecCPU},
		{"[m]", "MEM", SecMEM},
		{"[g]", "GPU", SecGPU},
		{"[n]", "NPU", SecNPU},
		{"[v]", "VPU", SecVPU},
		{"[a]", "RGA", SecRGA},
	}
	secParts := make([]string, 0, len(secEntries))
	for _, e := range secEntries {
		state := "on"
		if !sections[e.idx] {
			state = "off"
		}
		secParts = append(secParts, s.label(e.key+e.name+":")+s.value(state))
	}
	tierMap := fmt.Sprintf("%s%s  %s%s  %s%s",
		s.label("[i]I/O:"), s.value(footerStateLabel(tiers[0], height, 0)),
		s.label("[s]Sys:"), s.value(footerStateLabel(tiers[1], height, 1)),
		s.label("[k]Krn:"), s.value(footerStateLabel(tiers[2], height, 2)))
	now := time.Now().Format("15:04:05")
	if l.Narrow {
		now = time.Now().Format("15:04")
	}
	line1 := strings.Join(secParts, "  ") + "  " + tierMap + "  " + s.dim("· "+now)
	line2 := s.hint("[q]quit  [+/-]refresh  [r]redraw  [?]help  [↑↓/pgup/pgdn/home/end]scroll")
	if l.Width > 0 {
		if lipgloss.Width(line1) > l.Width {
			line1 = ansi.Truncate(line1, l.Width, "…")
		}
		if lipgloss.Width(line2) > l.Width {
			line2 = ansi.Truncate(line2, l.Width, "…")
		}
	}
	return line1 + "\n" + line2
}

// --- formatters -------------------------------------------------------------

func pctText(s Styles, pct int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	idx := pct / 5
	return s.pctANSI[idx] + fmt.Sprintf("%3d%%", pct) + s.ansiReset
}

func fmtTempCell(s Styles, milliC int) string {
	if milliC <= 0 {
		return s.dim("     ")
	}
	c := float64(milliC) / 1000.0
	return tempANSI(s, c) + fmt.Sprintf("%4.0f°C", c) + s.ansiReset
}

func tempANSI(s Styles, c float64) string {
	switch {
	case c < 35:
		return s.tempANSI[0]
	case c < 50:
		return s.tempANSI[2]
	case c < 65:
		return s.tempANSI[3]
	case c < 78:
		return s.tempANSI[4]
	default:
		return s.tempANSI[5]
	}
}

func fmtKiB(kib uint64) string {
	const (
		MiB = 1024.0
		GiB = 1024.0 * 1024.0
	)
	f := float64(kib)
	switch {
	case f >= GiB:
		return fmt.Sprintf("%.1fG", f/GiB)
	case f >= MiB:
		return fmt.Sprintf("%.0fM", f/MiB)
	default:
		return fmt.Sprintf("%dK", kib)
	}
}

func fmtUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

// padVisible right-pads text to exactly `width` visible (non-ANSI) cells.
func padVisible(text string, width int) string {
	w := lipgloss.Width(text)
	if w >= width {
		return text
	}
	return text + strings.Repeat(" ", width-w)
}
