# Changelog

All notable changes to **rkmon** are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-06-12

Stack refresh and CI/CD modernization.

### Changed

- **bubbletea v1 → v2** (`charm.land/bubbletea/v2 v2.0.7`) — migrated to the declarative `View() tea.View` API; `AltScreen` is now a field on the returned view instead of a `tea.WithAltScreen()` program option.
- **lipgloss v1 → v2** (`charm.land/lipgloss/v2 v2.0.4`) — pure-render mode, automatic color downsampling driven by Bubble Tea.
- **Go floor 1.24 → 1.26**, plus `go get -u ./... && go mod tidy` for every transitive dependency.
- **GitHub Actions pins** — `actions/checkout v4→v6`, `actions/setup-go v5→v6`, `goreleaser/goreleaser-action v6→v7`, `golangci/golangci-lint-action v6→v9`, `actions/upload-artifact v4→v7`.
- **CI workflow** — dropped standalone `gofmt`/`go vet` bash steps; both run inside `golangci-lint v2.12` via the new `.golangci.yml` config. Replaced the inline `go build … | grep statically` bash with `goreleaser build --snapshot` and `actions/upload-artifact@v7` so every CI run publishes the linux/arm64 binary for inspection.
- **Release workflow** — split into `goreleaser release --skip=publish` → verify static linkage → `goreleaser release --skip=build` so the static-link check now happens **before** publication, not after.

### Added

- **`.github/dependabot.yml`** — weekly grouped updates for `gomod` (charm.land/*, golang.org/x/*, go-runtime) and `github-actions` (actions/*, ecosystem), plus monthly no-op coverage for `gitsubmodule`. All groups emit Conventional Commits with scoped labels.
- **`.golangci.yml`** — explicit v2 config; default `none` plus stable allow-list (`errcheck`, `govet` (enable-all minus fieldalignment), `ineffassign`, `staticcheck`, `unused`, `misspell`, `unconvert`) and `gofmt` + `gofumpt` as formatters.

### Security

- **Redacted a private SSH hostname** from the Makefile and `docs/demo.tape`; rewrote the entire git history with `git filter-repo --replace-text` so no historical commit contains it.
- **Rewrote author/committer email** across history from a personal address to the GitHub noreply alias (`isac322@users.noreply.github.com`) via `--mailmap`, so the public commit log no longer leaks the maintainer's primary mailbox.

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

[Unreleased]: https://github.com/isac322/rkmon/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/isac322/rkmon/releases/tag/v0.3.0
[0.2.0]: https://github.com/isac322/rkmon/releases/tag/v0.2.0
