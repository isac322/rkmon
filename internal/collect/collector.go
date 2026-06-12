package collect

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Hardware paths on RK3588 BSP kernel.
const (
	GPUDevfreq = "/sys/class/devfreq/fb000000.gpu-mali"
	NPUDevfreq = "/sys/class/devfreq/fdab0000.npu"
	DDRDevfreq = "/sys/class/devfreq/dmc"

	RKNPULoad   = "/sys/kernel/debug/rknpu/load"
	MPPLoad     = "/proc/mpp_service/load"
	MPPInterval = "/proc/mpp_service/load_interval"
	MPPSessions = "/proc/mpp_service/sessions-summary"
	RGALoad     = "/sys/kernel/debug/rkrga/load"
)

// Cluster mapping for RK3588: cpu0-3 = A55 (little), cpu4-7 = A76 (big).
var ClusterOf = []string{"A55", "A55", "A55", "A55", "A76", "A76", "A76", "A76"}
var ClusterThermalZone = map[int]string{
	0: "littlecore-thermal",
	4: "bigcore0-thermal",
	6: "bigcore1-thermal",
}

type Collector struct {
	mu sync.Mutex

	prevCPU      map[string]CPUTimes
	prevSnapshot *Snapshot

	prevTaskCount map[string]uint64
	prevTaskAt    time.Time
	prevNPULoad   int
	npuSameCount  int

	prevDisk   map[string]DiskCounters
	prevDiskAt time.Time
	prevNet    map[string]NetCounters
	prevNetAt  time.Time
	prevCtxt   uint64
	prevCtxtAt time.Time
	prevIRQ    []uint64
	prevIRQAt  time.Time

	cachedDevfreqLimits map[string]freqLimits
	cachedThermalTypes  map[string]string
	cachedPCIe          []PCIeDev
	cachedFanHwmon      string
	mppIntervalChecked  bool
}

type freqLimits struct {
	min, max uint64
}

func New() *Collector {
	return &Collector{
		prevCPU:             make(map[string]CPUTimes),
		prevTaskCount:       make(map[string]uint64),
		prevDisk:            make(map[string]DiskCounters),
		prevNet:             make(map[string]NetCounters),
		cachedDevfreqLimits: make(map[string]freqLimits),
		cachedThermalTypes:  make(map[string]string),
	}
}

// Snapshot reads all hardware state, computes deltas where needed, and returns
// a Snapshot. Safe to call concurrently. Returns a partial Snapshot on a
// missing sysfs node rather than failing hard.
func (c *Collector) Snapshot() (*Snapshot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	snap := &Snapshot{
		CollectedAt: now,
		Thermal:     make(map[string]int),
	}

	c.readHost(snap)
	c.readCPU(snap)
	c.readMem(snap)
	c.readDevfreqs(snap)
	c.readNPUCores(snap)
	c.readVPU(snap, now)
	c.readRGA(snap)
	c.readISP(snap)
	c.readThermal(snap)
	c.readDisks(snap, now)
	c.readNets(snap, now)
	c.readThrottle(snap)
	c.readCMA(snap)
	c.readFan(snap)
	c.readGovernor(snap)
	c.readPCIe(snap)
	c.readCtxIRQ(snap, now)

	c.prevSnapshot = snap
	return snap, nil
}

// --- host -------------------------------------------------------------------

var cachedHostname = func() string {
	h, _ := os.Hostname()
	return h
}()

func (c *Collector) readHost(snap *Snapshot) {
	hostname := cachedHostname
	kernel := unameRelease()

	uptimeRaw, _ := readFile("/proc/uptime")
	loadRaw, _ := readFile("/proc/loadavg")

	avg, running, total := ParseLoadavg(loadRaw)
	snap.Host = HostInfo{
		Hostname:     hostname,
		Kernel:       kernel,
		Uptime:       time.Duration(ParseUptimeSeconds(uptimeRaw) * float64(time.Second)),
		LoadAvg:      avg,
		ProcsRunning: running,
		ProcsTotal:   total,
		IsRoot:       os.Geteuid() == 0,
	}
}

var cachedKernel = func() string {
	if s, err := readFile("/proc/sys/kernel/osrelease"); err == nil {
		return strings.TrimSpace(s)
	}
	return ""
}()

func unameRelease() string { return cachedKernel }

// --- CPU --------------------------------------------------------------------

func (c *Collector) readCPU(snap *Snapshot) {
	raw, err := readFile("/proc/stat")
	if err != nil {
		return
	}
	cur := make(map[string]CPUTimes)
	for _, line := range strings.Split(raw, "\n") {
		if !strings.HasPrefix(line, "cpu") {
			continue
		}
		name, t, ok := ParseCPULine(line)
		if !ok {
			continue
		}
		cur[name] = t
	}

	cores := make([]CPUCore, 8)
	for i := 0; i < 8; i++ {
		name := "cpu" + strconv.Itoa(i)
		core := CPUCore{Index: i, Cluster: ClusterOf[i]}
		if pt, ok := c.prevCPU[name]; ok {
			if ct, ok := cur[name]; ok {
				core.PctUsed = CPUPctFromDelta(pt, ct)
			}
		}
		core.FreqMHz = readCPUFreqMHz(i)
		cores[i] = core
	}
	snap.CPU = cores
	c.prevCPU = cur
}

func readCPUFreqMHz(idx int) int {
	path := "/sys/devices/system/cpu/cpu" + strconv.Itoa(idx) + "/cpufreq/scaling_cur_freq"
	s, err := readFile(path)
	if err != nil {
		return 0
	}
	v, _ := strconv.Atoi(strings.TrimSpace(s))
	return v / 1000
}

// --- Memory -----------------------------------------------------------------

func (c *Collector) readMem(snap *Snapshot) {
	raw, err := readFile("/proc/meminfo")
	if err != nil {
		return
	}
	snap.Mem = ParseMeminfo(raw)
}

// --- Devfreq nodes ----------------------------------------------------------

func (c *Collector) readDevfreqs(snap *Snapshot) {
	snap.GPU = c.readDevfreq(GPUDevfreq, "Mali-G610", "gpu-thermal")
	snap.NPU = c.readDevfreq(NPUDevfreq, "NPU", "npu-thermal")
	snap.DDR = c.readDevfreq(DDRDevfreq, "DDR", "")

	// rknpu_ondemand quirk: load values stick when NPU goes idle. Detect
	// "same value for N samples" sticky and flag stale.
	if c.prevNPULoad == snap.NPU.PctUsed {
		c.npuSameCount++
	} else {
		c.npuSameCount = 0
	}
	c.prevNPULoad = snap.NPU.PctUsed
	if c.npuSameCount >= 3 && snap.NPU.PctUsed == 100 {
		snap.NPU.Stale = true
	}
}

func (c *Collector) readDevfreq(base, name, thermalZone string) Devfreq {
	d := Devfreq{Name: name, ThermalZone: thermalZone}
	if raw, err := readFile(filepath.Join(base, "load")); err == nil {
		d.PctUsed, d.FreqHz = ParseDevfreqLoad(raw)
	}
	if d.FreqHz == 0 {
		if raw, err := readFile(filepath.Join(base, "cur_freq")); err == nil {
			d.FreqHz, _ = strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
		}
	}
	limits, cached := c.cachedDevfreqLimits[base]
	if !cached {
		if raw, err := readFile(filepath.Join(base, "min_freq")); err == nil {
			limits.min, _ = strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
		}
		if raw, err := readFile(filepath.Join(base, "max_freq")); err == nil {
			limits.max, _ = strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
		}
		if limits.min > 0 || limits.max > 0 {
			c.cachedDevfreqLimits[base] = limits
		}
	}
	d.MinHz, d.MaxHz = limits.min, limits.max
	return d
}

// --- NPU per-core (root only) ----------------------------------------------

func (c *Collector) readNPUCores(snap *Snapshot) {
	raw, err := readFile(RKNPULoad)
	if err != nil {
		snap.NPUCores.Available = false
		return
	}
	cores := ParseRKNPULoad(raw)
	snap.NPUCores = NPUCores{Available: len(cores) > 0, Cores: cores}
	// If we have ground truth, override the (possibly stale) aggregate flag.
	if snap.NPUCores.Available {
		maxC := 0
		for _, c := range cores {
			if c > maxC {
				maxC = c
			}
		}
		if maxC < 100 {
			snap.NPU.Stale = false
		}
	}
}

// --- VPU (mpp_service) -----------------------------------------------------

func (c *Collector) readVPU(snap *Snapshot, now time.Time) {
	if !c.mppIntervalChecked {
		c.mppIntervalChecked = true
		if raw, err := readFile(MPPInterval); err == nil {
			if iv, _ := strconv.Atoi(strings.TrimSpace(raw)); iv == 0 {
				_ = os.WriteFile(MPPInterval, []byte("1000\n"), 0o644)
			}
		}
	}

	// Active session count is sudoless.
	if raw, err := readFile(MPPSessions); err == nil {
		snap.VPU.Sessions = ParseMPPSessions(raw)
	}

	// Prefer real %-load from /proc/mpp_service/load (root + interval set).
	if raw, err := readFile(MPPLoad); err == nil {
		entries := ParseMPPLoad(raw)
		if len(entries) > 0 {
			snap.VPU.Mode = "load"
			indexOf := map[string]int{}
			eng := make([]VPUEngine, 0, len(entries))
			for _, e := range entries {
				idx := indexOf[e.Device]
				indexOf[e.Device]++
				eng = append(eng, VPUEngine{
					Name:    e.Device + strconv.Itoa(idx),
					LoadPct: e.LoadPct,
					UtilPct: e.UtilPct,
				})
			}
			snap.VPU.Engines = eng
			return
		}
	}

	// Sudoless fallback: tasks/sec from per-core task_count counters.
	curCounts := map[string]uint64{}
	for _, kind := range []string{"rkvdec-core0", "rkvdec-core1", "rkvenc-core0", "rkvenc-core1"} {
		path := "/proc/mpp_service/" + kind + "/task_count"
		if raw, err := readFile(path); err == nil {
			v, _ := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
			curCounts[kind] = v
		}
	}

	snap.VPU.Mode = "rates"
	dt := now.Sub(c.prevTaskAt).Seconds()
	if c.prevTaskAt.IsZero() || dt <= 0 {
		dt = 1
	}

	names := make([]string, 0, len(curCounts))
	for k := range curCounts {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		cur := curCounts[k]
		prev := c.prevTaskCount[k]
		var rate float64
		if cur > prev {
			rate = float64(cur-prev) / dt
		}
		snap.VPU.Engines = append(snap.VPU.Engines, VPUEngine{
			Name:        k,
			LoadPct:     -1,
			TasksPerSec: rate,
		})
	}
	c.prevTaskCount = curCounts
	c.prevTaskAt = now
}

func (c *Collector) readRGA(snap *Snapshot) {
	raw, err := readFile(RGALoad)
	if err != nil {
		return
	}
	cores := ParseRGALoad(raw)
	snap.RGA = RGAInfo{Available: len(cores) > 0, Cores: cores}
}

func (c *Collector) readISP(snap *Snapshot) {
	seen := map[string]struct{}{}
	for _, glob := range []string{
		"/sys/class/video4linux/video*/name",
		"/sys/class/video4linux/v4l-subdev*/name",
		"/sys/class/media/media*/name",
	} {
		matches, _ := filepath.Glob(glob)
		for _, p := range matches {
			raw, err := readFile(p)
			if err != nil {
				continue
			}
			name := strings.TrimSpace(raw)
			lower := strings.ToLower(name)
			if strings.Contains(lower, "rkisp") || strings.Contains(lower, "rk-isp") {
				if _, ok := seen[name]; ok {
					continue
				}
				seen[name] = struct{}{}
				snap.ISP.Available = true
				snap.ISP.Devices = append(snap.ISP.Devices, name)
			}
		}
	}
	sort.Strings(snap.ISP.Devices)
}

// --- Thermal ---------------------------------------------------------------

func (c *Collector) readThermal(snap *Snapshot) {
	if len(c.cachedThermalTypes) == 0 {
		entries, err := os.ReadDir("/sys/class/thermal")
		if err != nil {
			return
		}
		for _, e := range entries {
			name := e.Name()
			if !strings.HasPrefix(name, "thermal_zone") {
				continue
			}
			base := filepath.Join("/sys/class/thermal", name)
			typ, err := readFile(filepath.Join(base, "type"))
			if err != nil {
				continue
			}
			c.cachedThermalTypes[base] = strings.TrimSpace(typ)
		}
	}
	for base, typ := range c.cachedThermalTypes {
		raw, err := readFile(filepath.Join(base, "temp"))
		if err != nil {
			continue
		}
		if v, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
			snap.Thermal[typ] = v
		}
	}
}

// --- Disk I/O --------------------------------------------------------------

func (c *Collector) readDisks(snap *Snapshot, now time.Time) {
	raw, err := readFile("/proc/diskstats")
	if err != nil {
		return
	}
	all := ParseDiskstats(raw)
	if c.prevDiskAt.IsZero() {
		c.prevDisk = make(map[string]DiskCounters, len(all))
		for _, d := range all {
			c.prevDisk[d.Name] = d
		}
		c.prevDiskAt = now
		return
	}
	dt := now.Sub(c.prevDiskAt).Seconds()
	if dt <= 0 {
		return
	}
	var out []DiskStat
	for _, d := range all {
		if !isWholeDisk(d.Name) {
			continue
		}
		prev, ok := c.prevDisk[d.Name]
		if !ok {
			c.prevDisk[d.Name] = d
			continue
		}
		const sectorBytes = 512
		const bytesPerMB = 1024 * 1024
		rB := float64(d.ReadSectors-prev.ReadSectors) * sectorBytes
		wB := float64(d.WriteSectors-prev.WriteSectors) * sectorBytes
		ioMs := float64(d.IOTimeMillis - prev.IOTimeMillis)
		util := 100.0 * ioMs / (dt * 1000.0)
		if util > 100 {
			util = 100
		}
		out = append(out, DiskStat{
			Name:      d.Name,
			ReadMBps:  rB / bytesPerMB / dt,
			WriteMBps: wB / bytesPerMB / dt,
			UtilPct:   util,
		})
		c.prevDisk[d.Name] = d
	}
	sort.Slice(out, func(i, j int) bool {
		ti := out[i].ReadMBps + out[i].WriteMBps
		tj := out[j].ReadMBps + out[j].WriteMBps
		if ti != tj {
			return ti > tj
		}
		return out[i].Name < out[j].Name
	})
	snap.Disks = out
	c.prevDiskAt = now
}

func isWholeDisk(name string) bool {
	switch {
	case strings.HasPrefix(name, "loop"),
		strings.HasPrefix(name, "zd"),
		strings.HasPrefix(name, "dm-"),
		strings.HasPrefix(name, "ram"),
		strings.HasPrefix(name, "ng"):
		return false
	}
	if strings.HasPrefix(name, "nvme") {
		return !strings.Contains(name, "p")
	}
	if strings.HasPrefix(name, "mmcblk") {
		return !strings.Contains(name, "p")
	}
	if strings.HasPrefix(name, "sd") && len(name) >= 3 {
		last := name[len(name)-1]
		return last < '0' || last > '9'
	}
	return false
}

// --- Network ---------------------------------------------------------------

func isPhysIface(name string) bool {
	switch {
	case name == "lo":
		return false
	case strings.HasPrefix(name, "docker"),
		strings.HasPrefix(name, "cilium"),
		strings.HasPrefix(name, "lxc"),
		strings.HasPrefix(name, "veth"),
		strings.HasPrefix(name, "br-"),
		strings.HasPrefix(name, "tap"),
		strings.HasPrefix(name, "cni"),
		strings.HasPrefix(name, "flannel"),
		strings.HasPrefix(name, "kube"):
		return false
	}
	return true
}

func (c *Collector) readNets(snap *Snapshot, now time.Time) {
	raw, err := readFile("/proc/net/dev")
	if err != nil {
		return
	}
	all := ParseNetDev(raw)
	if c.prevNetAt.IsZero() {
		for _, n := range all {
			c.prevNet[n.Iface] = n
		}
		c.prevNetAt = now
		return
	}
	dt := now.Sub(c.prevNetAt).Seconds()
	if dt <= 0 {
		return
	}
	const bytesPerMB = 1024 * 1024
	var out []NetStat
	for _, n := range all {
		if !isPhysIface(n.Iface) {
			continue
		}
		prev, ok := c.prevNet[n.Iface]
		if !ok {
			c.prevNet[n.Iface] = n
			continue
		}
		out = append(out, NetStat{
			Iface:   n.Iface,
			RxMBps:  float64(n.RxBytes-prev.RxBytes) / bytesPerMB / dt,
			TxMBps:  float64(n.TxBytes-prev.TxBytes) / bytesPerMB / dt,
			RxTotal: n.RxBytes,
			TxTotal: n.TxBytes,
		})
		c.prevNet[n.Iface] = n
	}
	sort.Slice(out, func(i, j int) bool {
		return (out[i].RxMBps + out[i].TxMBps) > (out[j].RxMBps + out[j].TxMBps)
	})
	snap.Nets = out
	c.prevNetAt = now
}

// --- Throttle / cooling devices --------------------------------------------

func (c *Collector) readThrottle(snap *Snapshot) {
	entries, err := os.ReadDir("/sys/class/thermal")
	if err != nil {
		return
	}
	var out []CoolingDev
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "cooling_device") {
			continue
		}
		base := filepath.Join("/sys/class/thermal", name)
		typ, _ := readFile(filepath.Join(base, "type"))
		typStr := strings.TrimSpace(typ)
		if strings.HasPrefix(typStr, "pwm-fan") || strings.HasPrefix(typStr, "pwmfan") {
			continue
		}
		curS, _ := readFile(filepath.Join(base, "cur_state"))
		maxS, _ := readFile(filepath.Join(base, "max_state"))
		cur, _ := strconv.Atoi(strings.TrimSpace(curS))
		mx, _ := strconv.Atoi(strings.TrimSpace(maxS))
		out = append(out, CoolingDev{Type: typStr, Cur: cur, Max: mx})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type < out[j].Type })
	snap.Throttle = out
}

// --- CMA -------------------------------------------------------------------

func (c *Collector) readCMA(snap *Snapshot) {
	raw, err := readFile("/proc/meminfo")
	if err != nil {
		return
	}
	snap.CMA = ParseCMA(raw)
}

// --- Fan -------------------------------------------------------------------

func (c *Collector) readFan(snap *Snapshot) {
	if c.cachedFanHwmon == "" {
		matches, _ := filepath.Glob("/sys/class/hwmon/hwmon*/name")
		for _, p := range matches {
			n, _ := readFile(p)
			n = strings.TrimSpace(n)
			if strings.HasPrefix(n, "pwmfan") || strings.HasPrefix(n, "pwm-fan") {
				c.cachedFanHwmon = filepath.Dir(p)
				break
			}
		}
		if c.cachedFanHwmon == "" {
			return
		}
	}
	pwmRaw, err := readFile(filepath.Join(c.cachedFanHwmon, "pwm1"))
	if err != nil {
		return
	}
	pwm, err := strconv.Atoi(strings.TrimSpace(pwmRaw))
	if err != nil {
		return
	}
	rpm := 0
	if rpmRaw, err := readFile(filepath.Join(c.cachedFanHwmon, "fan1_input")); err == nil {
		rpm, _ = strconv.Atoi(strings.TrimSpace(rpmRaw))
	}
	snap.Fan = FanInfo{
		Available: true,
		PWM:       pwm,
		PWMPct:    pwm * 100 / 255,
		RPM:       rpm,
	}
}

// --- Governor --------------------------------------------------------------

func (c *Collector) readGovernor(snap *Snapshot) {
	read := func(cpu int) string {
		raw, _ := readFile("/sys/devices/system/cpu/cpu" + strconv.Itoa(cpu) + "/cpufreq/scaling_governor")
		return strings.TrimSpace(raw)
	}
	snap.Governor = GovernorInfo{
		A55: read(0), A76_0: read(4), A76_1: read(6),
	}
}

// --- PCIe ------------------------------------------------------------------

func (c *Collector) readPCIe(snap *Snapshot) {
	if c.cachedPCIe != nil {
		snap.PCIe = c.cachedPCIe
		return
	}
	matches, _ := filepath.Glob("/sys/bus/pci/devices/*")
	var out []PCIeDev
	for _, p := range matches {
		spdRaw, err := readFile(filepath.Join(p, "current_link_speed"))
		if err != nil {
			continue
		}
		spd := strings.TrimSpace(spdRaw)
		if spd == "Unknown speed" || spd == "" {
			continue
		}
		widRaw, _ := readFile(filepath.Join(p, "current_link_width"))
		wid, _ := strconv.Atoi(strings.TrimSpace(widRaw))
		clsRaw, _ := readFile(filepath.Join(p, "class"))
		out = append(out, PCIeDev{
			BusID:     filepath.Base(p),
			LinkSpeed: spd,
			LinkWidth: wid,
			Class:     strings.TrimSpace(clsRaw),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].BusID < out[j].BusID })
	c.cachedPCIe = out
	snap.PCIe = out
}

// --- Ctxsw / IRQ -----------------------------------------------------------

func (c *Collector) readCtxIRQ(snap *Snapshot, now time.Time) {
	statRaw, _ := readFile("/proc/stat")
	curCtxt, _ := ParseStatCtxt(statRaw)
	intRaw, _ := readFile("/proc/interrupts")
	curIRQ := ParseInterruptsPerCPU(intRaw)

	if c.prevCtxtAt.IsZero() {
		c.prevCtxt = curCtxt
		c.prevCtxtAt = now
		c.prevIRQ = curIRQ
		c.prevIRQAt = now
		return
	}
	dt := now.Sub(c.prevCtxtAt).Seconds()
	if dt > 0 && curCtxt >= c.prevCtxt {
		snap.CtxSwitch = uint64(float64(curCtxt-c.prevCtxt) / dt)
	}
	c.prevCtxt = curCtxt
	c.prevCtxtAt = now

	if len(curIRQ) == len(c.prevIRQ) && len(curIRQ) > 0 {
		dtI := now.Sub(c.prevIRQAt).Seconds()
		if dtI > 0 {
			out := make([]uint64, len(curIRQ))
			for i, v := range curIRQ {
				if v >= c.prevIRQ[i] {
					out[i] = uint64(float64(v-c.prevIRQ[i]) / dtI)
				}
			}
			snap.IRQPerCPU = out
		}
	}
	c.prevIRQ = curIRQ
	c.prevIRQAt = now
}

// --- helpers ---------------------------------------------------------------

func readFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
