# Memory Collector — Component Guide

## Overview

Absolute memory and swap usage collector using `gopsutil/v4/mem.VirtualMemory()` and `mem.SwapMemory()`. Returns exact byte counts (uint64) with no rounding or unit conversion.

Source file: `memory.go` (27 lines)

## How It Works

1. Calls `mem.VirtualMemory()` for RAM stats
2. Calls `mem.SwapMemory()` for swap stats
3. Returns raw values — no delta computation needed

## Key Behavior

| Field | Source | Type | Notes |
|---|---|---|---|
| Total | `vmem.Total` | uint64 bytes | Exact, no rounding |
| Used | `vmem.Used` | uint64 bytes | Exact, no rounding |
| Available | `vmem.Available` | uint64 bytes | Exact, no rounding |
| UsedPercent | `vmem.UsedPercent` | float64 % | Rounded to 2 decimals |
| SwapTotal | `swap.Total` | uint64 bytes | Exact, no rounding |
| SwapUsed | `swap.Used` | uint64 bytes | Exact, no rounding |
| SwapUsedPct | `swap.UsedPercent` | float64 % | Rounded to 2 decimals |

## Types

```go
type MemoryMetrics struct {
    Total        uint64  // total RAM in bytes
    Used         uint64  // used RAM in bytes
    UsedPercent  float64 // used RAM percentage (0-100)
    Available    uint64  // available RAM in bytes
    SwapTotal    uint64  // total swap in bytes
    SwapUsed     uint64  // used swap in bytes
    SwapUsedPct  float64 // used swap percentage (0-100)
}

type MemoryPreviousState struct{} // unused — no delta tracking
```

## Usage Pattern

```go
metrics, _, err := sysprobe.Collect(sysprobe.Options{Memory: true}, prev)
if err != nil {
    log.Printf("memory collect error: %v", err)
    return
}

m := metrics.Memory
fmt.Printf("RAM: %.1f GB / %.1f GB (%.1f%%)\n",
    float64(m.Used)/1e9, float64(m.Total)/1e9, m.UsedPercent)
fmt.Printf("Swap: %.1f MB / %.1f MB (%.1f%%)\n",
    float64(m.SwapUsed)/1e6, float64(m.SwapTotal)/1e6, m.SwapUsedPct)
```

## Notes for LLMs

- Memory metrics are **absolute** — no previous state needed, no delta computation.
- Byte values are exact uint64 — callers handle their own unit conversions (GB, MB, etc.).
- Only percentages (`UsedPercent`, `SwapUsedPct`) pass through `roundTwo()`.
- If either `VirtualMemory()` or `SwapMemory()` fails, the entire memory collection fails.
