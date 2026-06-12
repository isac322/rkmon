package collect

import (
	"regexp"
	"strconv"
	"strings"
)

// CPUTimes is the raw delta-accounting unit for a CPU line in /proc/stat.
// Fields exclude guest/guest_nice (they are double-counted in user/nice).
// Iowait is treated as idle (htop convention).
type CPUTimes struct {
	User, Nice, System, Idle, IOWait, IRQ, SoftIRQ, Steal uint64
}

func (t CPUTimes) Total() uint64 {
	return t.User + t.Nice + t.System + t.Idle + t.IOWait + t.IRQ + t.SoftIRQ + t.Steal
}

func (t CPUTimes) Active() uint64 {
	return t.User + t.Nice + t.System + t.IRQ + t.SoftIRQ + t.Steal
}

// ParseCPULine parses one "cpu" or "cpuN" line of /proc/stat.
func ParseCPULine(line string) (name string, t CPUTimes, ok bool) {
	fs := strings.Fields(line)
	if len(fs) < 5 || !strings.HasPrefix(fs[0], "cpu") {
		return "", CPUTimes{}, false
	}
	name = fs[0]
	getU64 := func(i int) uint64 {
		if i >= len(fs) {
			return 0
		}
		v, _ := strconv.ParseUint(fs[i], 10, 64)
		return v
	}
	t = CPUTimes{
		User: getU64(1), Nice: getU64(2), System: getU64(3),
		Idle: getU64(4), IOWait: getU64(5), IRQ: getU64(6),
		SoftIRQ: getU64(7), Steal: getU64(8),
	}
	return name, t, true
}

// CPUPctFromDelta returns 0..100 from two consecutive CPUTimes reads.
// Handles NO_HZ negative delta (clamp 0) and zero-total (returns 0).
func CPUPctFromDelta(prev, cur CPUTimes) int {
	curT, prevT := cur.Total(), prev.Total()
	curA, prevA := cur.Active(), prev.Active()
	if curT <= prevT {
		return 0
	}
	dt := curT - prevT
	var da uint64
	if curA > prevA {
		da = curA - prevA
	}
	if dt == 0 {
		return 0
	}
	p := int((da*100 + dt/2) / dt) // rounded
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	return p
}

// ParseMeminfo parses /proc/meminfo into KiB values.
func ParseMeminfo(s string) MemInfo {
	m := MemInfo{}
	for _, line := range strings.Split(s, "\n") {
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		key := line[:colon]
		val := strings.TrimSpace(line[colon+1:])
		val = strings.TrimSuffix(val, " kB")
		v, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			continue
		}
		switch key {
		case "MemTotal":
			m.TotalKiB = v
		case "MemAvailable":
			m.AvailableKiB = v
		case "Buffers":
			m.BuffersKiB = v
		case "Cached":
			m.CachedKiB = v
		case "SwapTotal":
			m.SwapTotalKiB = v
		case "SwapFree":
			m.SwapFreeKiB = v
		}
	}
	return m
}

// ParseDevfreqLoad handles the "N@FreqHz" format used by Mali / NPU / DDR
// devfreq nodes on RK3588 BSP kernels (e.g. "22@300000000Hz").
func ParseDevfreqLoad(raw string) (pct int, freqHz uint64) {
	raw = strings.TrimSpace(raw)
	at := strings.IndexByte(raw, '@')
	if at < 0 {
		// Fallback: plain integer (some drivers omit the @freq part)
		if n, err := strconv.Atoi(raw); err == nil {
			return clampPct(n), 0
		}
		return 0, 0
	}
	pctStr := strings.TrimRight(raw[:at], "% ")
	hzStr := strings.TrimRight(raw[at+1:], "Hz \t\r\n")
	if n, err := strconv.Atoi(pctStr); err == nil {
		pct = clampPct(n)
	}
	freqHz, _ = strconv.ParseUint(hzStr, 10, 64)
	return pct, freqHz
}

var npuCoreRE = regexp.MustCompile(`Core(\d+):\s*(\d+)%`)

// ParseRKNPULoad parses /sys/kernel/debug/rknpu/load:
//   "NPU load:  Core0:  0%, Core1: 12%, Core2:  0%,"
func ParseRKNPULoad(s string) []int {
	var out []int
	for _, m := range npuCoreRE.FindAllStringSubmatch(s, -1) {
		if v, err := strconv.Atoi(m[2]); err == nil {
			out = append(out, clampPct(v))
		}
	}
	return out
}

// mppLoadRE captures (device_name, load%, util%).
// Example line: "fdc38100.rkvdec-core      load:  17.41% utilization:  17.09%"
var mppLoadRE = regexp.MustCompile(`(\S+)\.(\S+?)\s+load:\s*([\d.]+)%\s+utilization:\s*([\d.]+)%`)

type MPPLoadEntry struct {
	DTNode  string // e.g. "fdc38100"
	Device  string // e.g. "rkvdec-core"
	LoadPct float64
	UtilPct float64
}

func ParseMPPLoad(s string) []MPPLoadEntry {
	var out []MPPLoadEntry
	for _, m := range mppLoadRE.FindAllStringSubmatch(s, -1) {
		l, _ := strconv.ParseFloat(m[3], 64)
		u, _ := strconv.ParseFloat(m[4], 64)
		out = append(out, MPPLoadEntry{
			DTNode:  m[1],
			Device:  m[2],
			LoadPct: l,
			UtilPct: u,
		})
	}
	return out
}

// ParseMPPSessions counts active "device:" lines in sessions-summary.
func ParseMPPSessions(s string) int {
	n := 0
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, "device:") {
			n++
		}
	}
	return n
}

var rgaLoadRE = regexp.MustCompile(`scheduler\[(\d+)\]:\s*(\S+)\s+load\s*=\s*(\d+)%`)

// ParseRGALoad parses /sys/kernel/debug/rkrga/load (root only).
// Disambiguates duplicate scheduler names (e.g. two "rga3") via the index.
func ParseRGALoad(s string) []RGACore {
	var out []RGACore
	seen := map[string]int{}
	for _, m := range rgaLoadRE.FindAllStringSubmatch(s, -1) {
		base := m[2]
		v, _ := strconv.Atoi(m[3])
		name := base
		if seen[base] > 0 {
			name = base + "_" + strconv.Itoa(seen[base])
		}
		seen[base]++
		out = append(out, RGACore{Name: name, LoadPct: clampPct(v)})
	}
	return out
}

// ParseLoadavg parses /proc/loadavg.
func ParseLoadavg(s string) (avg [3]float64, running, total int) {
	fs := strings.Fields(s)
	if len(fs) < 4 {
		return
	}
	for i := 0; i < 3; i++ {
		v, _ := strconv.ParseFloat(fs[i], 64)
		avg[i] = v
	}
	if slash := strings.IndexByte(fs[3], '/'); slash >= 0 {
		running, _ = strconv.Atoi(fs[3][:slash])
		total, _ = strconv.Atoi(fs[3][slash+1:])
	}
	return
}

// ParseUptimeSeconds parses /proc/uptime first field.
func ParseUptimeSeconds(s string) float64 {
	fs := strings.Fields(s)
	if len(fs) == 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(fs[0], 64)
	return v
}

// DiskCounters is one row of /proc/diskstats relevant to throughput/util.
type DiskCounters struct {
	Name           string
	ReadSectors    uint64
	WriteSectors   uint64
	IOTimeMillis   uint64
}

// ParseDiskstats returns one entry per device line. Caller filters whole-disk
// vs partition names.
func ParseDiskstats(s string) []DiskCounters {
	var out []DiskCounters
	for _, line := range strings.Split(s, "\n") {
		fs := strings.Fields(line)
		if len(fs) < 14 {
			continue
		}
		rs, _ := strconv.ParseUint(fs[5], 10, 64)
		ws, _ := strconv.ParseUint(fs[9], 10, 64)
		ioms, _ := strconv.ParseUint(fs[12], 10, 64)
		out = append(out, DiskCounters{
			Name: fs[2], ReadSectors: rs, WriteSectors: ws, IOTimeMillis: ioms,
		})
	}
	return out
}

// NetCounters tracks one interface's cumulative byte totals.
type NetCounters struct {
	Iface           string
	RxBytes, TxBytes uint64
}

// ParseNetDev parses /proc/net/dev (skips 2 header lines).
func ParseNetDev(s string) []NetCounters {
	var out []NetCounters
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if i < 2 {
			continue
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		iface := strings.TrimSpace(line[:colon])
		fs := strings.Fields(line[colon+1:])
		if len(fs) < 16 {
			continue
		}
		rx, _ := strconv.ParseUint(fs[0], 10, 64)
		tx, _ := strconv.ParseUint(fs[8], 10, 64)
		out = append(out, NetCounters{Iface: iface, RxBytes: rx, TxBytes: tx})
	}
	return out
}

// ParseCMA extracts CMA pool stats from /proc/meminfo content.
func ParseCMA(s string) CMAInfo {
	var ci CMAInfo
	for _, line := range strings.Split(s, "\n") {
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		key := line[:colon]
		val := strings.TrimSpace(line[colon+1:])
		val = strings.TrimSuffix(val, " kB")
		v, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			continue
		}
		switch key {
		case "CmaTotal":
			ci.TotalKiB = v
			ci.Available = true
		case "CmaFree":
			ci.FreeKiB = v
		}
	}
	if ci.Available && ci.TotalKiB >= ci.FreeKiB {
		ci.AllocatedKiB = ci.TotalKiB - ci.FreeKiB
	}
	return ci
}

// ParseStatCtxt extracts ctxt and processes fields from /proc/stat.
func ParseStatCtxt(s string) (ctxt, processes uint64) {
	for _, line := range strings.Split(s, "\n") {
		switch {
		case strings.HasPrefix(line, "ctxt "):
			ctxt, _ = strconv.ParseUint(strings.TrimPrefix(line, "ctxt "), 10, 64)
		case strings.HasPrefix(line, "processes "):
			processes, _ = strconv.ParseUint(strings.TrimPrefix(line, "processes "), 10, 64)
		}
	}
	return
}

// ParseInterruptsPerCPU sums each per-CPU column across all IRQ rows.
// Returns one uint64 per CPU, length = number of CPU columns in the header.
func ParseInterruptsPerCPU(s string) []uint64 {
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return nil
	}
	header := strings.Fields(lines[0])
	n := 0
	for _, h := range header {
		if strings.HasPrefix(h, "CPU") {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	sums := make([]uint64, n)
	for _, line := range lines[1:] {
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		fs := strings.Fields(line[colon+1:])
		if len(fs) < n {
			continue
		}
		for i := 0; i < n; i++ {
			v, err := strconv.ParseUint(fs[i], 10, 64)
			if err != nil {
				continue
			}
			sums[i] += v
		}
	}
	return sums
}

func clampPct(n int) int {
	if n < 0 {
		return 0
	}
	if n > 100 {
		return 100
	}
	return n
}
