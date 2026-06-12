# rkmon

TUI hardware monitor for Rockchip RK3588 (Radxa Rock 5B+).

## Install

```sh
make build               # cross-compiles to build/rkmon-linux-arm64
make deploy              # scp to your-rk3588-host:~/.local/bin/rkmon
```

## Run

```sh
rkmon                    # default 1s refresh, all tiers in auto (height-based)
rkmon --refresh=500ms
rkmon --no-color         # plain output, still keeps box layout
rkmon --once             # single snapshot to stdout, no TUI (defaults to --tiers=all)
rkmon --width=160        # override auto-detected width
rkmon --tiers=i,k        # show I/O and Kernel tiers only (1,3 also accepted)
rkmon --tiers=all        # show every tier
rkmon --tiers=none       # core panels only
sudo rkmon               # unlock root-only metrics (see below)
```

Keys:
- `q`/`ctrl+c` quit · `+`/`-` adjust refresh · `r` force redraw
- `c`/`m`/`g`/`n`/`v`/`a` toggle CPU/MEM/GPU/NPU/VPU/RGA core sections.
- `i`/`s`/`k` independently toggle the three tier panel groups (`1`/`2`/`3` also accepted as numeric aliases). Binary on/off, no multi-step cycle. Initial state is `auto` (height-based) until first press. Footer shows effective visibility per tier such as `[i]I/O:on  [s]Sys:off  [k]Krn:on`. Restart with `--tiers=""` to reset all back to auto.
- `↑`/`↓`/`j` line scroll · `pgup`/`pgdn`/`space` page scroll · `home`/`end` top/bottom — scroll the main view body when content exceeds the terminal height. Top border, bottom border, and footer stay pinned.
- `?` open multi-tab help. Inside help:
  - `tab`/`←`/`→` cycle tabs (`shift+tab`/`h`/`p` reverse)
  - `↑`/`↓`/`k`/`j` scroll line · `pgup`/`pgdn`/`b`/`f`/`space` scroll page · `g`/`G` top/bottom
  - `esc`/`q`/`?` close help
  - Tabs cover keybinds, metric definitions, and per-metric stress-test recipes.

## Panels and tiers

Core panels (CPU/MEM/GPU/NPU/VPU/RGA) can each be toggled via `c`/`m`/`g`/`n`/`v`/`a` keys. Remaining panels are grouped into 3 tiers — I/O, System, and Kernel — toggled via `i`/`s`/`k` keys (or the `--tiers` CLI flag); the legacy `1`/`2`/`3` numeric keys are still accepted as aliases. Higher tiers auto-show when terminal height permits.

**Core (toggleable):** CPU per-core · Memory (MEM/SWAP/DDR ctrl) · GPU (Mali-G610) · NPU (RKNPU per-core or aggregate) · VPU (mpp_service) · RGA (2D accel) · ISP (auto-shown when camera attached).

**Wide-screen layout:** when terminal width is `>= 150`, Thermal zones and PCIe links are moved to a separate right-side column (vertical), keeping core panels in the left column.

**I/O tier (`i`, auto when height ≥ 44):** Disk I/O · Network · Throttle (cooling devices).

**System tier (`s`, auto when height ≥ 57):** CMA pool · Fan PWM · CPU governor · PCIe link speed/width.

**Kernel tier (`k`, auto when height ≥ 60):** Context switches/sec · IRQ counts per CPU.

## Metric matrix

| Panel | User mode | Root mode | Dynamic QA |
|---|---|---|---|
| CPU per-core (% / MHz / cluster temp) | ok | ok | proven via `yes >/dev/null` x8 → all cores 100% |
| MEM / SWAP | ok | ok | proven via tmpfs alloc → 71% → 77% |
| GPU Mali-G610 (% / MHz / temp) | ok | ok | **NOT proven dynamically** (no glmark2/vulkan workload) |
| NPU agg | stale flag if sticky | per-core via debugfs | **NOT proven** (no RKNN inference workload) |
| DDR ctrl | ok | ok | proven via ffmpeg HW transcode → 9% → 30% |
| VPU (mpp_service) | sessions + tasks/s | full load% via `mpp_service/load` | proven via ffmpeg HW transcode → rkvenc-core0 0% → 21%, sess 0 → 1 |
| RGA (2D accel) | hint only (needs sudo) | load% per core | **NOT proven** (no RGA test workload) |
| ISP camera | auto-hidden when no camera | same | auto-hidden on this host (HDMI-only video0) |
| Thermal zones | 7 zones (SoC/B0/B1/L/Cen/G/N) | same | proven via CPU stress → SoC 42°C → 46°C |
| Disk I/O (`i`) | all whole disks (nvme/sd/mmcblk) — r/w MB/s + util% | same | proven via mmcblk1 18% util on idle host |
| Network (`i`) | physical interfaces (eth/wifi/wg), filters LXC/docker noise — RX/TX MB/s + total | same | proven via eth0 baseline traffic |
| Throttle (`i`) | cooling_device states per CPU cluster / GPU / DMC | same | shown as cur/max ratio; >0 means active throttle |
| CMA pool (`s`) | Allocated / Total from /proc/meminfo | same | proven via meminfo read |
| Fan PWM (`s`) | hwmon `pwmfan` PWM 0..255 + RPM if exposed | same | rock5bp exposes PWM only, no fan tachometer |
| CPU governor (`s`) | per-cluster scaling_governor (A55 / A76-0 / A76-1) | same | proven |
| PCIe links (`s`) | every PCIe device's current speed × width × class | same | proven via 8 devices on rock5bp (NVMe x2, WiFi, Ethernet, bridges) |
| Ctxsw/IRQ (`k`) | ctxsw delta + per-CPU IRQ delta from /proc/stat + /proc/interrupts | same | proven via /proc deltas |

Metrics marked "NOT proven dynamically" read from the same kernel paths as the proven ones; behavior is symmetric, but no workload generator was available in the test environment to drive them above idle.

## Root mode

Some metrics require root (debugfs access):

- NPU per-core load via `/sys/kernel/debug/rknpu/load`
- VPU per-engine load% via `/proc/mpp_service/load` (otherwise falls back to sessions+tasks/s)
- RGA per-core load via `/sys/kernel/debug/rkrga/load`

Run as `sudo rkmon` to unlock.

## Layout

- < 60 cols: "terminal too narrow" message (hard minimum)
- 60-74 cols: narrow mode, 8-cell bars, short labels
- 75-89 cols: narrow mode, 12-cell bars
- 90-129 cols: normal mode, 24-cell bars
- ≥ 130 cols: wide mode, 36-cell bars

If terminal height is shorter than the rendered frame, the body becomes scrollable: top border, bottom border, and the 2-line footer stay pinned, while panel rows scroll via the `↑`/`↓`/`j`/`pgup`/`pgdn`/`space`/`home`/`end` keys.
