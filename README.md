# sysprobe

Platform-agnostic system metric collection for Go. Collects CPU, memory, disk I/O, disk space, network I/O, and NVIDIA GPU metrics with delta-based rate calculation where needed.

## Install

```bash
go get github.com/henrygd/sysprobe
```

Requires Go 1.26+.

## Usage

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/henrygd/sysprobe"
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

	for {
		metrics, newState, err := sysprobe.Collect(opts, prev)
		if err != nil {
			log.Fatal(err)
		}
		prev = newState

		fmt.Printf("CPU avg: %.2f%%\n", metrics.CPU.Average)
		fmt.Printf("Memory: %.0f / %.0f MB (%.1f%%)\n",
			metrics.Memory.Used, metrics.Memory.Total, metrics.Memory.UsedPercent)

		for _, d := range metrics.DiskIO {
			fmt.Printf("Disk %s: %s %.2f MB/s, %s %.2f MB/s\n",
				d.Name, "R", d.ReadMB, "W", d.WriteMB)
		}

		for _, n := range metrics.Network {
	fmt.Printf("Net %s: TX %.0f B/s, RX %.0f B/s\n",
			n.Name, n.SentBytes, n.ReceivedBytes)
		}

		time.Sleep(5 * time.Second)
	}
}
```

### JSON Export

```go
b, err := sysprobe.JSONExport(metrics)
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(b)) // pretty-printed JSON

// GPU metrics
gpu, _ := sysprobe.GPUCollect()
gb, _ := sysprobe.JSONExportGPU(gpu)
```

All metric structs have `json:` tags, so you can also use `encoding/json` directly.

### GPU Metrics

GPU polling runs asynchronously via nvidia-smi. Start it on first call and fetch the latest snapshot each tick:

```go
gpu, err := sysprobe.GPUCollect()
if err != nil {
    log.Printf("gpu error: %v", err)
} else {
    fmt.Printf("GPU: %.0fC, %.0f%% util, %.0fW\n",
        gpu.Temperature, gpu.UtilizationGPU, gpu.Power)
}

// Stop when done
defer sysprobe.GPUStop()
```

## Metric Kinds

| Kind | Delta or Absolute | Notes |
|------|-------------------|-------|
| CPU | Delta | Per-core + average usage % |
| Memory | Absolute | Total/used/available in MB |
| Disk I/O | Delta | Read/write MB/s and IOPS per device |
| Disk Space | Absolute | Total/free/used per mount point |
| Network | Delta | Sent/received MB/s per interface |
| GPU | Absolute | Async nvidia-smi polling (~5s) |

Delta-based metrics require passing `PreviousState` from the prior call. On the first call, pass an empty `PreviousState{}` - deltas will be zero until the second tick.

## Network Filtering

By default, loopback and tunnel/virtual interfaces are excluded. Control with options:

```go
opts := sysprobe.Options{
    Network:         true,
    IncludeLoopback: true, // include lo
    IncludeTunnel:   true, // include tun/tap/vxlan etc. (Linux only)
    MACMinLength:    0,    // 0 = no MAC filter
}
```

## Platform Support

- **Linux**: Full support including netlink-based interface detection
- **Windows**: Full support with simplified interface filtering
- **macOS**: CPU, memory, disk, and network supported
