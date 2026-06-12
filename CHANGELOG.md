# Changelog

All notable changes to **rkmon** are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-06-12

Initial public release.

### Added

- **Core panels** — CPU per-core (big.LITTLE-aware), Memory (MEM/SWAP/DDR controller), GPU (Mali-G610), NPU (RKNPU with per-core load when run as root), VPU (Rockchip MPP umbrella: rkvenc/rkvdec/jpeg/av1d), RGA (2D accelerator), ISP (auto-shown when a camera is attached), thermal zones across all 7 sensors.
- **I/O tier** — per-disk read/write MB/s + utilization%, per-interface RX/TX MB/s, cooling-device throttle states (CPU/GPU/DMC).
- **System tier** — CMA pool, fan PWM, per-cluster CPU governor, PCIe link speed × width per device.
- **Kernel tier** — context switches/sec, per-CPU IRQ/sec.
- **Responsive layout** — auto-narrow (60–89 cols), normal (90–129), wide (130+), two-column (150+) with thermal/PCIe pinned to the right side.
- **Independent toggles** — `c m g n v a` for core sections, `i s k` (or `1 2 3`) for tiers; per-tier auto-show by terminal height when nothing is forced.
- **Scrollable main view** — `↑ ↓ j pgup pgdn space home end`; top/bottom border and 2-line footer pinned.
- **Multi-tab help** (`?`): Keybinds · Metrics · Stress tests (with concrete stress-ng/ffmpeg/iperf3/fio recipes per metric).
- **CLI** — `--once`, `--width`, `--refresh`, `--tiers`, `--no-color`, `--version`, `--help`.

[Unreleased]: https://github.com/isac322/rkmon/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/isac322/rkmon/releases/tag/v0.2.0
