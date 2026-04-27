# Disk Collector — Component Guide

## Overview

Two-part collector: **disk I/O** (delta-based read/write rates and IOPS) and **disk space** (absolute usage per mount point). Uses `gopsutil/v4/disk`.

Source file: `disk.go` (92 lines)

---

## Disk I/O

### How It Works

1. Calls `disk.IOCounters()` to get cumulative counters per device
2. For each device, computes deltas from previous sample:
   - `readDelta = (current.ReadBytes - prev.ReadBytes) / interval_seconds` → bytes/sec, then divided by 1048576 for MB/s
   - `writeDelta` — same logic for writes
   - `iopsRead = (current.ReadCount - prev.ReadCount) / interval_seconds`
   - `iopsWrite` — same logic for writes
3. Tracks timestamps per device in `diskIOCounters`

### Key Behavior

| Scenario | Output |
|---|---|
| First tick (no previous state) | Zeroed metrics for all devices (ReadMB=0, WriteMB=0, IOPSRead=0, IOPSWrite=0) |
| Normal tick | Computed MB/s and IOPS, rounded to 2 decimals |
| Device disappears | Removed from output on next tick |
| New device appears | Added with zeroed metrics |

### Types

```go
type DiskIOMetric struct {
    Name      string  // device name (e.g., "sda", "nvme0n1")
    ReadMB    float64 // read throughput in MB/s
    WriteMB   float64 // write throughput in MB/s
    IOPSRead  float64 // read operations per second
    IOPSWrite float64 // write operations per second
}

type DiskIOPreviousState struct {
    Counters map[string]diskIOCounters // keyed by device name
}

type diskIOCounters struct { // internal, not exported
    ReadBytes  uint64
    WriteBytes uint64
    ReadCount  uint64
    WriteCount uint64
    Time       time.Time
}
```

---

## Disk Space

### How It Works

1. Calls `disk.Partitions(false)` to list mount points (no pseudo-filesystems)
2. For each partition, calls `disk.Usage(mountpoint)` for total/used/free
3. Converts bytes to GB and rounds to 2 decimals

### Types

```go
type DiskSpaceMetric struct {
    Path        string  // mount point path
    Device      string  // backing device (e.g., "/dev/sda1")
    FSType      string  // filesystem type (e.g., "ext4", "apfs")
    Total       float64 // total space in GB
    Free        float64 // free space in GB
    Used        float64 // used space in GB
    UsedPercent float64 // used percentage (0-100)
}

type DiskSpacePreviousState struct{} // unused — no delta tracking
```

---

## Usage Pattern

```go
metrics, newState, err := sysprobe.Collect(sysprobe.Options{
    DiskIO:    true,
    DiskSpace: true,
}, prev)
if err != nil {
    log.Printf("disk collect error: %v", err)
    return
}
prev = newState

// Disk I/O
for _, d := range metrics.DiskIO {
    fmt.Printf("%s: R=%.2f MB/s W=%.2f MB/s (IOPS: R=%.0f W=%.0f)\n",
        d.Name, d.ReadMB, d.WriteMB, d.IOPSRead, d.IOPSWrite)
}

// Disk Space
for _, s := range metrics.DiskSpace {
    fmt.Printf("%s (%s): %.1f GB / %.1f GB (%.1f%%)\n",
        s.Path, s.Device, s.Used, s.Total, s.UsedPercent)
}
```

## Notes for LLMs

- Disk I/O is **delta-based** — first tick returns zeros. Pass back `DiskIOPreviousState` each tick.
- Disk space is **absolute** — no previous state needed.
- Disk I/O throughput is in MB/s (divides by 1048576 = 2^20), not raw bytes/sec.
- Error handling: if `disk.Usage()` fails for a partition, that partition is skipped (`continue`), not fatal.
- All float values pass through `roundTwo()`.
