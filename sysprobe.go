// Package sysprobe provides platform-agnostic system metric collection.
//
// It mirrors the metric types used by Beszel agents so external projects can
// collect identical metrics without depending on the Beszel codebase.
package sysprobe

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
)

// Options controls which subsystems to collect and interface filtering.
type Options struct {
	// DiskIO enables disk I/O metrics collection.
	DiskIO bool
	// DiskSpace enables disk space metrics per mount point.
	DiskSpace bool
	// Network enables network I/O metrics collection.
	Network bool
	// CPU enables CPU usage metrics collection.
	CPU bool
	// Memory enables memory metrics collection.
	Memory bool

	// IncludeLoopback includes loopback interfaces in network metrics.
	IncludeLoopback bool
	// IncludeTunnel includes tunnel interfaces in network metrics (Linux only).
	IncludeTunnel bool

	// MACMinLength is the minimum MAC address length to include an interface.
	// Interfaces with shorter MACs are excluded (default 6).
	MACMinLength int
}

// Metrics holds all collected system metrics for a single tick.
type Metrics struct {
	Timestamp    time.Time `json:"timestamp"`
	CPU          *CPUMetrics
	Memory       *MemoryMetrics
	DiskIO       []DiskIOMetric
	DiskSpace    []DiskSpaceMetric
	Network      []NetworkMetric
}

// CPUMetrics holds CPU usage percentages.
type CPUMetrics struct {
	Average float64   `json:"average"`
	Cores   []float64 `json:"cores"`
}

// MemoryMetrics holds memory and swap usage in bytes.
type MemoryMetrics struct {
	Total        uint64  `json:"total_bytes"`
	Used         uint64  `json:"used_bytes"`
	UsedPercent  float64 `json:"used_percent"`
	Available    uint64  `json:"available_bytes"`
	SwapTotal    uint64  `json:"swap_total_bytes"`
	SwapUsed     uint64  `json:"swap_used_bytes"`
	SwapUsedPct  float64 `json:"swap_used_percent"`
}

// DiskIOMetric holds delta I/O counters for a single disk device.
type DiskIOMetric struct {
	Name      string  `json:"name"`
	ReadMB    float64 `json:"read_mb"`
	WriteMB   float64 `json:"write_mb"`
	IOPSRead  float64 `json:"iops_read"`
	IOPSWrite float64 `json:"iops_write"`
}

// DiskSpaceMetric holds usage for a single mount point.
type DiskSpaceMetric struct {
	Path        string  `json:"path"`
	Device      string  `json:"device"`
	FSType      string  `json:"fstype"`
	Total       float64 `json:"total_gb"`
	Free        float64 `json:"free_gb"`
	Used        float64 `json:"used_gb"`
	UsedPercent float64 `json:"used_percent"`
}

// NetworkMetric holds bandwidth for a single network interface (bytes/sec).
type NetworkMetric struct {
	Name          string  `json:"name"`
	MAC           string  `json:"mac"`
	MTU           int     `json:"mtu"`
	SentBps       float64 `json:"sent_bps"`
	ReceivedBps   float64 `json:"received_bps"`
	HasPublicIP   bool    `json:"has_public_ip"`
}

// TickState holds the previous tick's raw counters for delta calculation.
type TickState struct {
	CPU       CPUTickState
	Memory    MemoryTickState
	DiskIO    DiskIOTickState
	DiskSpace DiskSpaceTickState
	Network   NetworkTickState
}

// CPUTickState stores per-CPU times for usage rate calculation.
type CPUTickState struct {
	Times []cpu.TimesStat
}

// MemoryTickState is unused (memory metrics are absolute, not delta-based).
type MemoryTickState struct{}

// DiskIOTickState stores previous I/O counters keyed by device name.
type DiskIOTickState struct {
	Counters map[string]diskIOCounters
}

// diskIOCounters holds raw cumulative counters for a disk device.
type diskIOCounters struct {
	ReadBytes  uint64
	WriteBytes uint64
	ReadCount  uint64
	WriteCount uint64
	Time       time.Time
}

// DiskSpaceTickState is unused (disk space metrics are absolute).
type DiskSpaceTickState struct{}

// NetworkTickState stores previous counters keyed by interface name.
type NetworkTickState struct {
	Counters map[string]netCounters
}

// netCounters holds raw cumulative byte counters for a network interface.
type netCounters struct {
	Sent     uint64
	Received uint64
	Time     time.Time
}

// GPUMetrics holds NVIDIA GPU telemetry.
type GPUMetrics struct {
	Timestamp   time.Time `json:"timestamp"`
	Temperature float64   `json:"temperature_c"`
	ClockFreq   float64   `json:"clock_freq_mhz"`
	MemoryUsed  float64   `json:"memory_used_mb"`
	MemoryTotal float64   `json:"memory_total_mb"`
	Power       float64   `json:"power_watts"`
	FanSpeed    float64   `json:"fan_speed_percent"`
	UtilizationGPU  float64 `json:"utilization_gpu_percent"`
	UtilizationMem  float64 `json:"utilization_mem_percent"`
}

// GPUTickState is unused (GPU metrics are polled absolutely).
type GPUTickState struct{}

var (
	gpuMu    sync.Mutex
	gpuData  GPUMetrics
	gpuError error
	gpuDone  chan struct{}
)

// Collect gathers all enabled metrics for a single tick.
// Pass the TickState returned from the prior call to compute deltas.
// Returns the new TickState for the next call.
func Collect(opts Options, prev TickState) (Metrics, TickState, error) {
	now := time.Now()
	var m Metrics
	var ps TickState

	if opts.CPU {
		cpuMet, cpuPs, err := cpuCollect(prev.CPU)
		if err != nil {
			return m, ps, err
		}
		m.CPU = cpuMet
		ps.CPU = cpuPs
	}

	if opts.Memory {
		memMet, err := memoryCollect()
		if err != nil {
			return m, ps, err
		}
		m.Memory = memMet
	}

	if opts.DiskIO {
		diskIOMet, diskIOps, err := diskIOCollect(prev.DiskIO)
		if err != nil {
			return m, ps, err
		}
		m.DiskIO = diskIOMet
		ps.DiskIO = diskIOps
	}

	if opts.DiskSpace {
		diskSpaceMet, err := diskSpaceCollect()
		if err != nil {
			return m, ps, err
		}
		m.DiskSpace = diskSpaceMet
	}

	if opts.Network {
		netMet, netPs, err := networkCollect(opts, prev.Network)
		if err != nil {
			return m, ps, err
		}
		m.Network = netMet
		ps.Network = netPs
	}

	m.Timestamp = now
	return m, ps, nil
}

// GPUCollect starts async NVIDIA GPU polling if not already running, then
// returns the latest snapshot. Callers should invoke this on each tick to
// keep the background poller alive. Returns (metrics, error).
func GPUCollect() (GPUMetrics, error) {
	gpuMu.Lock()
	defer gpuMu.Unlock()

	if gpuDone == nil {
		gpuDone = make(chan struct{})
		go gpuPoll(gpuDone)
	}

	return gpuData, gpuError
}

// GPUStop stops the background GPU poller.
func GPUStop() {
	gpuMu.Lock()
	defer gpuMu.Unlock()

	if gpuDone != nil {
		close(gpuDone)
		gpuDone = nil
	}
}

// JSONExport returns pretty-printed JSON bytes for the given metrics.
func JSONExport(m Metrics) ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// JSONExportGPU returns pretty-printed JSON bytes for the given GPU metrics.
func JSONExportGPU(g GPUMetrics) ([]byte, error) {
	return json.MarshalIndent(g, "", "  ")
}

func roundTwo(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
