package sysprobe

import (
	"testing"
	"time"
)

func TestCollectBasic(t *testing.T) {
	opts := Options{
		CPU:    true,
		Memory: true,
		DiskIO: true,
	}

	var prev PreviousState

	metrics, newState, err := Collect(opts, prev)
	if err != nil {
		t.Fatalf("first collect failed: %v", err)
	}

	if metrics.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}

	// First tick CPU deltas are zero since there's no previous state
	if metrics.CPU == nil {
		t.Fatal("CPU metrics should not be nil")
	}
	if len(metrics.CPU.Cores) == 0 {
		t.Error("expected at least one CPU core")
	}

	if metrics.Memory == nil {
		t.Fatal("memory metrics should not be nil")
	}
	if metrics.Memory.Total <= 0 {
		t.Error("total memory should be positive")
	}

	// Second tick should have non-zero deltas
	time.Sleep(500 * time.Millisecond)

	metrics2, _, err := Collect(opts, newState)
	if err != nil {
		t.Fatalf("second collect failed: %v", err)
	}

	// CPU average should be >= 0 (may still be 0 on idle system, that's fine)
	if metrics2.CPU.Average < 0 {
		t.Error("CPU average should not be negative")
	}
}

func TestCollectMemoryOnly(t *testing.T) {
	opts := Options{
		Memory: true,
	}

	metrics, _, err := Collect(opts, PreviousState{})
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}

	if metrics.CPU != nil {
		t.Error("CPU should be nil when not enabled")
	}
	if metrics.Memory == nil {
		t.Fatal("memory should not be nil when enabled")
	}
	if metrics.Memory.UsedPercent < 0 || metrics.Memory.UsedPercent > 100 {
		t.Errorf("used percent out of range: %.2f", metrics.Memory.UsedPercent)
	}
}

func TestCollectDiskSpace(t *testing.T) {
	opts := Options{
		DiskSpace: true,
	}

	metrics, _, err := Collect(opts, PreviousState{})
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}

	if len(metrics.DiskSpace) == 0 {
		t.Fatal("expected at least one disk space metric")
	}

	for _, d := range metrics.DiskSpace {
		if d.Path == "" {
			t.Error("disk space path should not be empty")
		}
		if d.UsedPercent < 0 || d.UsedPercent > 100 {
			t.Errorf("used percent out of range for %s: %.2f", d.Path, d.UsedPercent)
		}
	}
}

func TestCollectNetwork(t *testing.T) {
	opts := Options{
		Network: true,
	}

	var prev PreviousState

	_, newState, err := Collect(opts, prev)
	if err != nil {
		t.Fatalf("first collect failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	metrics2, _, err := Collect(opts, newState)
	if err != nil {
		t.Fatalf("second collect failed: %v", err)
	}

	for _, n := range metrics2.Network {
		if n.Name == "" {
			t.Error("network name should not be empty")
		}
		if n.SentBps < 0 || n.ReceivedBps < 0 {
			t.Errorf("negative network metrics for %s", n.Name)
		}
	}
}

func TestOptionsDisableAll(t *testing.T) {
	opts := Options{}

	metrics, _, err := Collect(opts, PreviousState{})
	if err != nil {
		t.Fatalf("collect with no options failed: %v", err)
	}

	// All metric fields should be nil/empty when nothing is enabled
	if metrics.CPU != nil {
		t.Error("CPU should be nil")
	}
	if metrics.Memory != nil {
		t.Error("memory should be nil")
	}
	if len(metrics.DiskIO) > 0 {
		t.Error("disk IO should be empty")
	}
	if len(metrics.DiskSpace) > 0 {
		t.Error("disk space should be empty")
	}
	if len(metrics.Network) > 0 {
		t.Error("network should be empty")
	}
}

func TestRoundTwo(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{1.234, 1.23},
		{1.235, 1.24},
		{1.236, 1.24},
		{0.0, 0.0},
		{99.999, 100.0},
	}

	for _, tt := range tests {
		got := roundTwo(tt.input)
		if got != tt.expected {
			t.Errorf("roundTwo(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
