# sysprobe — CLI Agent Context

## What This Is
A standalone Go package that extracts system metric collection logic from the [Beszel](https://github.com/phanmn/beszel) monitoring agent. Provides CPU, memory, disk I/O + space, network I/O, and GPU (nvidia-smi) metrics via a single `Collect()` call. Mirrors Beszel's metric types exactly.

## Module
- **Path**: `github.com/phanmn/sysprobe`
- **Go version**: 1.26+
- **Single dependency**: `github.com/shirou/gopsutil/v4` (all metric sources)
- **Layout**: Flat single package at module root — no subdirectories, no internal packages

## Public API (`sysprobe.go`)
| Function | Description |
|---|---|
| `Collect(opts Options, prev PreviousState) (*Metrics, PreviousState, error)` | Collect all enabled subsystems. Returns new metrics + state to pass back on next tick. Delta-based collectors (CPU, disk I/O, network) require previous state for rate calculation. First tick returns zeroed delta metrics so callers see which interfaces/disks/cores exist from the start. |
| `GPUCollect() ([]GPUInfo, error)` | Returns latest GPU snapshot from background nvidia-smi poller. Call after `GPUStart()`. |
| `GPUStart(ctx context.Context, interval time.Duration) error` | Starts async nvidia-smi polling goroutine (~5s default). |
| `GPUStop()` | Stops the GPU polling goroutine. |
| `JSONExport(metrics *Metrics) (string, error)` | Convenience helper — marshals metrics to JSON string. |
| `JSONExportGPU(gpus []GPUInfo) (string, error)` | Convenience helper — marshals GPU info to JSON string. |

## Metric Kinds
| Collector | Kind | Units | Notes |
|---|---|---|---|
| CPU | Delta | % usage per core + average | `CPUTimes(true)` with time diff between samples |
| Memory | Absolute | Bytes (uint64) for total/used/available/swap; float64 % for used/swap percent | No rounding on byte values |
| Disk I/O | Delta | ReadBps / WriteBps (bytes/sec), IOPSRead / IOPSWrite | Timestamp-tracked deltas per device |
| Disk Space | Absolute | Bytes for total/used/free | Per mount point, no delta needed |
| Network | Delta | SentBps / ReceivedBps (bytes/sec) + `HasPublicIP` bool | Timestamp-tracked deltas; public IP detection covers both IPv4 and IPv6 non-private ranges |
| GPU | Async snapshot | Memory %, Temp °C, Utilization %, Power W | Parsed from `nvidia-smi` CSV output, background goroutine |

## Key Conventions
- **Delta collectors** (CPU, disk I/O, network) track timestamps between samples. First tick always returns zeroed metrics — never empty arrays — so callers know which items exist.
- **Network filtering**: Skip loopback unless `IncludeLoopback=true`. Skip tunnel interfaces on Linux/unless `IncludeTunnel=true`. Min 6-byte MAC filter catches synthetic interfaces. Tunnel detection is name-based (no netlink dependency).
- **Public IP detection** (`HasPublicIP`): Checks all addresses on the interface. IPv4 excludes 10.x, 172.16-31.x, 192.168.x, 127.x, 169.254.x. IPv6 excludes ::, ::1, fe80::/10, fc00::/7, ff00::/8.
- **GPU**: Fire-and-forget background goroutine. `GPUCollect()` returns the latest snapshot (may be stale if nvidia-smi is slow or fails).
- **Rounding**: `roundTwo()` applies only to percentages and byte/sec rates. Byte values are returned as exact uint64.
- **No external config files, no CLI** — pure library package.

## Platform-Specific Files
| File | Build Tag | Purpose |
|---|---|---|
| `types_linux.go` | `//go:build linux` | Name-based tunnel detection (tun, tap, gre, vxlan, etc.) + net.InterfaceByName |
| `types_windows.go` | `//go:build windows` | Simple net.InterfaceByName fallback, no tunnel filtering |
| `types_darwin.go` | `//go:build darwin` | macOS interface helpers with utun/awdl/virtual prefix tunnel detection |

## Build / Test / Lint
```bash
# Custom wrapper handles build + test + vet in one command
rtk go build .       # build
rtk go test -v .     # run tests (6 tests)
rtk go vet .         # static analysis
```

## Critical Context for LLMs
- Module path is `github.com/phanmn/sysprobe`. Import with `go get` after pushing to GitHub, or use `go.work` for local dev.
- All metric struct fields have `json:` tags — callers can marshal directly with `encoding/json` without using the provided helpers.
- `PreviousState` is a single struct that holds all delta-tracking state (CPU, disk I/O, network counters) plus GPU snapshot. Caller passes it back verbatim on each tick.
- No git submodules, no vendoring, no Makefile — keep it minimal.
