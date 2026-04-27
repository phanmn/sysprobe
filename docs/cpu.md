# CPU Collector — Component Guide

## Overview

Delta-based CPU usage collector that computes per-core and average utilization percentages from `gopsutil/v4/cpu.Times(true)`.

Source file: `cpu.go` (44 lines)

## How It Works

1. Calls `cpu.Times(true)` to get cumulative time stats per core
2. For each core, computes deltas between current and previous sample:
   - `userDelta = t.User - p.User`
   - `systemDelta = t.System - p.System`
   - `coreDelta = cpuTotalTime(t) - cpuTotalTime(p)`
3. Usage percentage = `(userDelta + systemDelta) / coreDelta * 100`
4. Average = mean of all core percentages

## Key Behavior

| Scenario | Output |
|---|---|
| First tick (no previous state) | Zeroed metrics for all cores, 0% average |
| Normal tick | Computed percentages rounded to 2 decimals |
| Core count changed | New cores get 0%, existing cores compute normally |

## Types

```go
type CPUMetrics struct {
    Average float64   // mean of all cores, rounded 2 decimals
    Cores   []float64 // per-core usage %, length = core count
}

type CPUPreviousState struct {
    Times []cpu.TimesStat // cumulative times from prior sample
}
```

## Usage Pattern

```go
var prev sysprobe.PreviousState

for range ticker.C {
    metrics, newState, err := sysprobe.Collect(sysprobe.Options{CPU: true}, prev)
    if err != nil {
        log.Printf("cpu collect error: %v", err)
        continue
    }
    prev = newState // CRITICAL: pass back for next delta calculation

    fmt.Printf("avg: %.2f%%\n", metrics.CPU.Average)
    for i, core := range metrics.CPU.Cores {
        fmt.Printf("core %d: %.2f%%\n", i, core)
    }
}
```

## Internal Functions

| Function | Location | Purpose |
|---|---|---|
| `cpuCollect(prev)` | cpu.go:7 | Main collection logic |
| `cpuTotalTime(t)` | sysprobe.go (implicit via gopsutil fields) | Sums all time fields in a TimesStat |

## Notes for LLMs

- CPU metrics are **delta-based** — the first tick always returns zeros. This is intentional so callers know which cores exist.
- `roundTwo()` applies to both per-core and average values.
- No filtering or interface selection — all online cores are reported.
- The `CPUPreviousState.Times` slice is replaced entirely each tick (not mutated in place).
