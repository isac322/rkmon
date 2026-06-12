package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type helpTab struct {
	title string
	body  string
}

var helpTabs = []helpTab{
	{
		title: "Keybinds",
		body: `Section toggles  (order matches the screen)
  c     CPU per-core panel
  m     Memory (MEM / SWAP / DDR ctrl)
  g     GPU (Mali-G610)
  n     NPU (RKNPU)
  v     VPU (mpp_service)
  a     RGA (2D accel)
        Footer shows: [X]NAME:on|off

Tier toggles  (binary on/off, shown after core sections on screen)
  i     I/O    — Disk I/O · Network · Throttle (cooling devices)
  s     System — CMA pool · Fan PWM · CPU governor · PCIe links
  k     Kernel — Context switches/sec · IRQ per CPU
        1 / 2 / 3 still accepted as aliases of i / s / k.
        Footer shows: [i]I/O:on|off  [s]Sys:on|off  [k]Krn:on|off
        Initial state is "auto" (height-based) until first key press.
        Note: at width >= 150, PCIe shows in the right column always.

Controls
  q / ctrl+c     Quit
  + / =          Refresh rate -100ms (faster)
  - / _          Refresh rate +100ms (slower)
  r              Force redraw / re-collect
  ?              Open / close this help

Scrolling (main view)
  ↑ ↓ / j        Scroll one line
  pgup pgdn      Scroll 10 lines (space also = pgdn)
  home / end     Jump to top / bottom of frame

Scrolling (help view)
  tab / ← →      Cycle help tabs
  ↑ ↓ / k j      Scroll one line
  pgup pgdn      Scroll 10 lines (space / b / f / ctrl+u / ctrl+d)
  g / G          Jump to top / bottom of help

CLI flags
  --refresh=DUR        100ms..60s (default 1s)
  --width=N            override auto-detected width
  --tiers=STR          "" | all | none | "i,k" | "1,3" (default auto)
  --no-color           disable ANSI colors
  --once               single snapshot to stdout, exit
`,
	},
	{
		title: "Metrics",
		body: `CPU per-core
  Utilization % via /proc/stat delta (excludes guest/guest_nice;
  iowait treated as idle; NO_HZ negative deltas clamped to 0).
  Cluster: A55 cpu0-3 (little) or A76 cpu4-7 (big).
  Freq from /sys/devices/system/cpu/cpuN/cpufreq/scaling_cur_freq.
  Temp on cpu0/cpu4/cpu6 via littlecore/bigcore0/bigcore1 zones.

Memory
  MEM   = (MemTotal - MemAvailable) / MemTotal from /proc/meminfo.
  SWAP  = (SwapTotal - SwapFree) / SwapTotal.
  DDR ctrl is the LPDDR4/5 memory CONTROLLER utilization (bandwidth,
  not capacity) from /sys/class/devfreq/dmc/load (N@FreqHz format).

GPU (Mali-G610)
  /sys/class/devfreq/fb000000.gpu-mali/load (utilization %).
  Freq + range from cur_freq / min_freq / max_freq.

NPU (RKNPU)
  Aggregate via /sys/class/devfreq/fdab0000.npu/load.
  Per-core (root only) via /sys/kernel/debug/rknpu/load
  parsed as "Core0: N%, Core1: N%, Core2: N%".
  Some BSPs stick at 100% under rknpu_ondemand — flagged as "raw".

VPU (mpp_service)
  Rockchip MPP umbrella for HW codecs (rkvdec/rkvenc/av1d/jpeg).
  Root mode: writes load_interval=1000 then reads
  /proc/mpp_service/load → per-engine load% and util%.
  User mode: parses sessions-summary count + delta task_count
  per rkvdec-core{0,1}/task_count (= tasks/s rate).

RGA (2D accel)
  /sys/kernel/debug/rkrga/load (root) gives per-scheduler load%.
  RK3588 exposes rga3 x2 + rga2 x1 (rga3 duplicate suffix _1).

Thermal zones
  Millidegree C from /sys/class/thermal/thermal_zoneN/temp.
  Standard 7 zones: soc, bigcore0, bigcore1, littlecore, center,
  gpu, npu.

Disk I/O (tier i = I/O)
  Per whole-disk (nvme*n*, sd[a-z], mmcblk*; partitions filtered):
  read MB/s + write MB/s + util% from /proc/diskstats deltas.
  sector_size = 512 bytes; util% = ioticks_delta / interval.

Network (tier i = I/O)
  Per physical interface (lxc/docker/cilium filtered):
  RX/TX MB/s from /proc/net/dev byte deltas + cumulative totals.

Throttle (tier i = I/O)
  /sys/class/thermal/cooling_device*: cur_state / max_state per
  device (cpu cluster freq caps, GPU cap, DMC cap). >0 = active.
  PWM fan excluded (shown separately under Fan panel).

CMA / Fan / Gov / PCIe (tier s = System)
  CMA: CmaTotal/CmaFree from /proc/meminfo.
  Fan: hwmon pwmfan/pwm1 PWM (0-255) + fan1_input RPM if exposed.
  Governor: cpu0/4/6/cpufreq/scaling_governor per cluster.
  PCIe: current_link_speed × current_link_width per device, with
  class code mapped to PCI bridge/NVMe/WiFi/Ethernet labels.

Ctxsw / IRQ (tier k = Kernel)
  Ctxsw: /proc/stat ctxt delta per second.
  IRQ:   /proc/interrupts per-CPU column sums delta per second.
`,
	},
	{
		title: "Stress tests",
		body: `Drive each metric to verify rkmon shows it correctly.

CPU
  for i in 1 2 3 4 5 6 7 8; do yes >/dev/null & done
  # all 8 cores go to 100%. Kill: pkill -f "^yes$"
  stress-ng --cpu 8 --timeout 60   # if installed

Memory (capacity)
  dd if=/dev/zero of=/dev/shm/big bs=1M count=8000
  # MEM jumps ~8GB. Clean: rm /dev/shm/big

DDR ctrl (memory bandwidth, separate from RAM fill)
  Run anything that streams large data through RAM:
  - ffmpeg HW transcode (also drives VPU)
  - big file copy: dd if=/dev/sda of=/dev/null bs=1M count=4096
  - rsync large dir between disks
  MEM stays flat; DDR ctrl % rises.

GPU (Mali-G610)
  glmark2-es2 --off-screen            # if installed
  vkmark                              # Vulkan benchmark
  Wayland + sway under heavy compositing

NPU (RKNPU)
  rknn_benchmark <model.rknn>         # Rockchip RKNN toolkit
  rknn_yolo_demo / rknn_mobilenet_demo
  (Requires Rockchip RKNN runtime + RKNN model file)

VPU (mpp_service) — proven workload
  # On a host with h264_rkmpp ffmpeg (Rockchip BSP build):
  ffmpeg -hide_banner -loglevel error -re -stream_loop 50 \
    -i src.mp4 -c:v h264_rkmpp -f null -
  # Watch rkvenc/rkvdec engines + session count.

RGA (2D accel)
  ffmpeg ... -vf scale_rkrga=W:H ...    # scale_rkrga driver
  rga_demo / rga_test from rockchip-rga github
  (Most easily exercised via ffmpeg with scale_rkrga filter)

Network
  iperf3 -s                              # server side
  iperf3 -c <peer> -t 30                 # client side
  # Or: scp a 10G file between hosts
  # Or: nc -l 5001 > /dev/null ; cat bigfile | nc peer 5001

Disk I/O
  fio --name=randrw --rw=randrw --bs=4k --size=1G --runtime=30
  dd if=/dev/zero of=/path/big.bin bs=1M count=4096 oflag=direct
  hdparm -t /dev/nvme0n1                 # sequential read bench

Thermal + Throttle
  Sustain CPU stress (above) for 30-60s. Watch SoC/bigcore temps
  climb, then cooling_device state goes 0 → N as kernel caps freq.

CMA pool
  Hard to drive without RKNN/camera workloads. CMA is allocated by
  drivers; rkmon just shows current allocation.

PCIe / Fan / Governor
  Static metrics — change boot-time only (PCIe link), or via
  thermal feedback (Fan PWM), or sysfs write (governor).

Ctxsw / IRQ
  Network workloads drive IRQs (NIC interrupt per packet).
  Many small processes drive context switches:
  for i in $(seq 1 1000); do (sleep 0.1) & done
`,
	},
}

func renderHelp(s Styles, width, height, tabIdx, scroll int) string {
	if width < 60 {
		width = 60
	}
	if tabIdx < 0 || tabIdx >= len(helpTabs) {
		tabIdx = 0
	}
	t := helpTabs[tabIdx]
	bodyLines := strings.Split(strings.TrimRight(t.body, "\n"), "\n")
	total := len(bodyLines)

	const chrome = 5
	bodyCap := total
	if height > 0 {
		bodyCap = height - chrome
		if bodyCap < 1 {
			bodyCap = 1
		}
		if bodyCap > total {
			bodyCap = total
		}
	}
	maxScroll := total - bodyCap
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}
	end := scroll + bodyCap
	if end > total {
		end = total
	}
	visible := bodyLines[scroll:end]

	var rows []string
	rows = append(rows, helpTopBorder(s, width, tabIdx))
	rows = append(rows, helpTabBar(s, width, tabIdx, scroll, maxScroll))
	rows = append(rows, helpDivider(s, width))
	for _, line := range visible {
		rows = append(rows, helpContentRow(s, width, line))
	}
	rows = append(rows, helpBottomBorder(s, width))
	rows = append(rows, helpFooterHint(s, scroll, maxScroll, end, total))

	return strings.Join(rows, "\n")
}

func helpMaxScroll(tabIdx, height int) int {
	if tabIdx < 0 || tabIdx >= len(helpTabs) {
		return 0
	}
	total := len(strings.Split(strings.TrimRight(helpTabs[tabIdx].body, "\n"), "\n"))
	const chrome = 5
	bodyCap := total
	if height > 0 {
		bodyCap = height - chrome
		if bodyCap < 1 {
			bodyCap = 1
		}
		if bodyCap > total {
			bodyCap = total
		}
	}
	m := total - bodyCap
	if m < 0 {
		m = 0
	}
	return m
}

func clampHelpScroll(v, tabIdx, height int) int {
	if v < 0 {
		return 0
	}
	if max := helpMaxScroll(tabIdx, height); v > max {
		return max
	}
	return v
}

func helpFooterHint(s Styles, scroll, maxScroll, end, total int) string {
	pos := fmt.Sprintf("(line %d-%d / %d)", scroll+1, end, total)
	keys := "[tab / ← →] tabs   [↑↓ / pgup pgdn / g G] scroll   [esc / ? / q] close"
	return s.hint(keys) + "   " + s.dim(pos)
}

func helpTopBorder(s Styles, width, tabIdx int) string {
	title := fmt.Sprintf("rkmon help · tab %d/%d: %s", tabIdx+1, len(helpTabs), helpTabs[tabIdx].title)
	prefix := s.border("┌─ ") + s.title(title) + " "
	visible := lipgloss.Width("┌─ " + title + " ")
	fill := width - visible - 1
	if fill < 1 {
		fill = 1
	}
	return prefix + s.border(strings.Repeat("─", fill)+"┐")
}

func helpTabBar(s Styles, width, tabIdx, scroll, maxScroll int) string {
	var sb strings.Builder
	sb.WriteString(s.BorderBarL)
	parts := make([]string, 0, len(helpTabs))
	for i, ht := range helpTabs {
		label := fmt.Sprintf(" %d. %s ", i+1, ht.title)
		if i == tabIdx {
			parts = append(parts, s.title("►"+label+"◄"))
		} else {
			parts = append(parts, s.dim(" "+label+" "))
		}
	}
	bar := strings.Join(parts, "  ")
	scrollHint := ""
	switch {
	case scroll > 0 && scroll < maxScroll:
		scrollHint = s.dim("↑↓")
	case scroll > 0:
		scrollHint = s.dim("↑ ")
	case maxScroll > 0:
		scrollHint = s.dim(" ↓")
	}
	avail := width - 4 - lipgloss.Width(scrollHint)
	if avail < 1 {
		avail = 1
	}
	if w := lipgloss.Width(bar); w < avail {
		bar += strings.Repeat(" ", avail-w)
	} else if w > avail {
		bar = ansi.Truncate(bar, avail, "…")
	}
	sb.WriteString(bar)
	sb.WriteString(scrollHint)
	sb.WriteString(s.BorderBarR)
	return sb.String()
}

func helpDivider(s Styles, width int) string {
	return s.border("├" + strings.Repeat("─", width-2) + "┤")
}

func helpContentRow(s Styles, width int, line string) string {
	inner := width - 4
	if lipgloss.Width(line) > inner {
		line = ansi.Truncate(line, inner, "…")
	}
	pad := inner - lipgloss.Width(line)
	if pad < 0 {
		pad = 0
	}
	return s.BorderBarL + line + strings.Repeat(" ", pad) + s.BorderBarR
}

func helpBottomBorder(s Styles, width int) string {
	return s.border("└" + strings.Repeat("─", width-2) + "┘")
}
