package collect

import "time"

// Snapshot is a point-in-time read of RK3588 hardware state.
type Snapshot struct {
	Host        HostInfo
	CPU         []CPUCore
	Mem         MemInfo
	GPU         Devfreq
	NPU         Devfreq
	DDR         Devfreq
	NPUCores    NPUCores
	VPU         VPUInfo
	RGA         RGAInfo
	ISP         ISPInfo
	Thermal     map[string]int // milli-Celsius
	Disks       []DiskStat
	Nets        []NetStat
	Throttle    []CoolingDev
	CMA         CMAInfo
	Fan         FanInfo
	Governor    GovernorInfo
	PCIe        []PCIeDev
	CtxSwitch   uint64   // per-second delta
	IRQPerCPU   []uint64 // per-second delta, len = NumCPU
	CollectedAt time.Time
}

type HostInfo struct {
	Hostname     string
	Kernel       string
	Uptime       time.Duration
	LoadAvg      [3]float64
	ProcsRunning int
	ProcsTotal   int
	IsRoot       bool
}

type CPUCore struct {
	Index   int
	Cluster string // "A55" or "A76"
	PctUsed int    // 0..100
	FreqMHz int
}

type MemInfo struct {
	TotalKiB, AvailableKiB, BuffersKiB, CachedKiB uint64
	SwapTotalKiB, SwapFreeKiB                     uint64
}

// Devfreq represents a generic devfreq node (GPU/NPU/DDR controller).
type Devfreq struct {
	Name        string
	PctUsed     int // 0..100; may be sticky for rknpu_ondemand
	FreqHz      uint64
	MinHz       uint64
	MaxHz       uint64
	Stale       bool   // true when value is known to be unreliable (e.g. NPU sticky)
	ThermalZone string // optional thermal zone type to look up
}

// NPUCores is the per-core debugfs read (root required).
type NPUCores struct {
	Available bool
	Cores     []int // each 0..100, len ~ 3 on RK3588
}

// VPUInfo describes Rockchip MPP service state.
type VPUInfo struct {
	Mode     string // "load" (root w/ load_interval set) | "rates" (delta task_count) | "sessions"
	Sessions int
	Engines  []VPUEngine
}

type VPUEngine struct {
	Name        string  // short name (e.g. "rkvdec-core0")
	LoadPct     float64 // -1 if unavailable
	UtilPct     float64
	TasksPerSec float64
}

type RGAInfo struct {
	Available bool
	Cores     []RGACore
}

type RGACore struct {
	Name    string // e.g. "rga3_core0", "rga2"
	LoadPct int    // 0..100
}

type ISPInfo struct {
	Available bool
	Devices   []string
}

type DiskStat struct {
	Name      string
	ReadMBps  float64
	WriteMBps float64
	UtilPct   float64
}

type NetStat struct {
	Iface   string
	RxMBps  float64
	TxMBps  float64
	RxTotal uint64
	TxTotal uint64
}

type CoolingDev struct {
	Type     string
	Cur, Max int
}

type CMAInfo struct {
	Available    bool
	TotalKiB     uint64
	AllocatedKiB uint64
	FreeKiB      uint64
}

type FanInfo struct {
	Available bool
	PWM       int // 0..255
	PWMPct    int // 0..100 (derived)
	RPM       int // 0 if not exposed
}

type GovernorInfo struct {
	A55   string
	A76_0 string
	A76_1 string
}

type PCIeDev struct {
	BusID     string
	LinkSpeed string // "8.0 GT/s PCIe"
	LinkWidth int
	Class     string // class code
}
