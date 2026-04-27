# Public API — Component Guide

## Overview

The `sysprobe.Collect()` function is the main entry point. It dispatches to individual collectors based on enabled options, aggregates results into a single `Metrics` struct, and manages delta state.

Source file: `sysprobe.go` (257 lines)

---

## Collect Function

### Signature

```go
func Collect(opts Options, prev PreviousState) (Metrics, PreviousState, error)
```

### Parameters

| Param | Purpose |
|---|---|
| `opts Options` | Controls which subsystems to enable and interface filtering rules |
| `prev PreviousState` | State from previous call for delta calculation. Zero value on first call. |

### Returns

| Return | Description |
|---|---|
| `Metrics` | Aggregated metrics for this tick. Fields are nil/empty for disabled collectors. |
| `PreviousState` | New state to pass back on next call. Contains raw counters and timestamps. |
| `error` | Non-nil if any enabled collector fails. Failed collectors abort the entire tick. |

### Collection Order

1. CPU (if `opts.CPU`)
2. Memory (if `opts.Memory`)
3. Disk I/O (if `opts.DiskIO`)
4. Disk Space (if `opts.DiskSpace`)
5. Network (if `opts.Network`)

If any step fails, remaining collectors are **not** executed for that tick.

---

## Options Struct

```go
type Options struct {
    DiskIO          bool // enable disk I/O metrics
    DiskSpace       bool // enable disk space per mount point
    Network         bool // enable network I/O metrics
    CPU             bool // enable CPU usage metrics
    Memory          bool // enable memory + swap metrics

    IncludeLoopback bool // include loopback interfaces (default: false)
    IncludeTunnel   bool // include tunnel interfaces (default: false, Linux/macOS only)

    MACMinLength    int  // min MAC address length to include interface (default: 6)
}
```

---

## Metrics Struct

```go
type Metrics struct {
    Timestamp time.Time       // collection time
    CPU       *CPUMetrics     // nil if opts.CPU == false
    Memory    *MemoryMetrics  // nil if opts.Memory == false
    DiskIO    []DiskIOMetric  // empty if opts.DiskIO == false
    DiskSpace []DiskSpaceMetric // empty if opts.DiskSpace == false
    Network   []NetworkMetric // empty if opts.Network == false
}
```

---

## JSON Export Helpers

```go
// Convenience wrappers — all struct fields have json: tags, so callers
// can also use encoding/json directly.
func JSONExport(m Metrics) ([]byte, error)
func JSONExportGPU(g GPUMetrics) ([]byte, error)
```

---

## Full Usage Pattern

```go
package main

import (
    "fmt"
    "time"

    "github.com/x16z/sysprobe"
)

func main() {
    opts := sysprobe.Options{
        CPU:       true,
        Memory:    true,
        DiskIO:    true,
        DiskSpace: true,
        Network:   true,
    }

    var prev sysprobe.PreviousState
    defer sysprobe.GPUStop()

    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        metrics, newState, err := sysprobe.Collect(opts, prev)
        if err != nil {
            fmt.Printf("collect error: %v\n", err)
            continue
        }
        prev = newState // required for delta calculation on next tick

        gpu, gpuErr := sysprobe.GPUCollect()
        if gpuErr == nil {
            fmt.Printf("GPU temp: %.0f°C\n", gpu.Temperature)
        }

        jsonBytes, _ := sysprobe.JSONExport(metrics)
        fmt.Printf("%s\n", jsonBytes)
    }
}
```

## Notes for LLMs

- `Collect()` is **not** concurrent-safe — do not call from multiple goroutines simultaneously.
- The returned `PreviousState` must be passed back verbatim on the next call. Modifying it will break delta calculations.
- Disabled collectors return nil (for pointers) or empty slices (for arrays), not error.
- Timestamp is set to `time.Now()` at the end of collection, after all collectors run.
- GPU is collected separately via `GPUCollect()` — it's not part of the main `Collect()` call because it uses a background poller.
