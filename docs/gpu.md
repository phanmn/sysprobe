# GPU Collector — Component Guide

## Overview

Async NVIDIA GPU telemetry collector that polls `nvidia-smi` in a background goroutine at ~5s intervals. Parses CSV output for temperature, clock, memory, power, fan speed, and utilization metrics.

Source file: `gpu.go` (88 lines)

---

## How It Works

### Lifecycle

1. First call to `GPUCollect()` starts the background poller goroutine automatically
2. Poller runs on a 5s ticker, calling `fetchGPUData()` each tick
3. Results stored in package-level variables protected by `gpuMu` mutex
4. Callers read the latest snapshot via `GPUCollect()` on their own schedule
5. Call `GPUStop()` to shut down the poller

### nvidia-smi Command

```bash
nvidia-smi --query-gpu=temperature.gpu,clocks.current.graphics,memory.used,memory.total,power.draw,fan.speed,utilization.gpu,utilization.memory --format=csv,noheader,nounits
```

Output is CSV with 8 fields per GPU (comma-separated, no header, no units).

### CSV Parsing

Custom `splitCSV()` parser handles quoted fields (nvidia-smi wraps some values in quotes). Does NOT use `encoding/csv` — avoids newline handling overhead for single-line output.

---

## Types

```go
type GPUMetrics struct {
    Timestamp      time.Time // when the sample was taken
    Temperature    float64   // GPU temperature in °C
    ClockFreq      float64   // current graphics clock in MHz
    MemoryUsed     float64   // used GPU memory in MB
    MemoryTotal    float64   // total GPU memory in MB
    Power          float64   // current power draw in watts
    FanSpeed       float64   // fan speed as percentage
    UtilizationGPU  float64   // GPU compute utilization %
    UtilizationMem  float64   // GPU memory utilization %
}

type GPUTickState struct{} // unused — metrics are absolute snapshots
```

---

## API

```go
// GPUCollect starts the poller if not running, returns latest snapshot.
func GPUCollect() (GPUMetrics, error)

// GPUStop stops the background poller.
func GPUStop()
```

---

## Usage Pattern

```go
// Start polling (happens automatically on first GPUCollect call)
gpu, err := sysprobe.GPUCollect()
if err != nil {
    log.Printf("gpu init error: %v", err)
    // poller will retry in background
}

// Periodic reads (e.g., every 10s)
for range ticker.C {
    gpu, err := sysprobe.GPUCollect()
    if err != nil {
        log.Printf("gpu error: %v", err)
        continue
    }
    fmt.Printf("GPU: %.0f°C, %.0f MHz, %.0f/%.0f MB, %.0fW, fan %.0f%%\n",
        gpu.Temperature, gpu.ClockFreq, gpu.MemoryUsed, gpu.MemoryTotal,
        gpu.Power, gpu.FanSpeed)
    fmt.Printf("  Util: GPU %.0f%%, Mem %.0f%%\n",
        gpu.UtilizationGPU, gpu.UtilizationMem)
}

// Cleanup on exit
defer sysprobe.GPUStop()
```

## Notes for LLMs

- GPU polling is **fire-and-forget** — `GPUCollect()` returns the latest snapshot, which may be stale if nvidia-smi fails or is slow.
- Poll interval is hardcoded at 5 seconds (`gpuPollInterval` constant).
- Package-level state (`gpuData`, `gpuError`, `gpuDone`) is protected by `gpuMu`.
- If `nvidia-smi` binary is not found or fails, the error is stored and returned on next `GPUCollect()` call. The poller continues retrying.
- Multi-GPU support: current implementation parses first GPU line only. nvidia-smi output with multiple GPUs would need CSV line splitting (comma within quotes).
- All float values pass through `roundTwo()`.
- `GPUStop()` is safe to call multiple times — checks for nil before closing channel.
