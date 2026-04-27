package sysprobe

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const gpuPollInterval = 5 * time.Second

func gpuPoll(done <-chan struct{}) {
	ticker := time.NewTicker(gpuPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			data, err := fetchGPUData()
			gpuMu.Lock()
			gpuData = data
			gpuError = err
			gpuMu.Unlock()
		}
	}
}

func fetchGPUData() (GPUMetrics, error) {
	out, err := exec.Command("nvidia-smi",
		"--query-gpu=temperature.gpu,clocks.current.graphics,memory.used,memory.total,power.draw,fan.speed,utilization.gpu,utilization.memory",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return GPUMetrics{}, fmt.Errorf("nvidia-smi: %w", err)
	}

	fields := splitCSV(string(out))
	if len(fields) < 8 {
		return GPUMetrics{}, fmt.Errorf("nvidia-smi: expected 8 fields, got %d", len(fields))
	}

	temp, _ := strconv.ParseFloat(strings.TrimSpace(fields[0]), 64)
	clock, _ := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64)
	memUsed, _ := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64)
	memTotal, _ := strconv.ParseFloat(strings.TrimSpace(fields[3]), 64)
	power, _ := strconv.ParseFloat(strings.TrimSpace(fields[4]), 64)
	fan, _ := strconv.ParseFloat(strings.TrimSpace(fields[5]), 64)
	utilGPU, _ := strconv.ParseFloat(strings.TrimSpace(fields[6]), 64)
	utilMem, _ := strconv.ParseFloat(strings.TrimSpace(fields[7]), 64)

	return GPUMetrics{
		Timestamp:      time.Now(),
		Temperature:    roundTwo(temp),
		ClockFreq:      roundTwo(clock),
		MemoryUsed:     roundTwo(memUsed),
		MemoryTotal:    roundTwo(memTotal),
		Power:          roundTwo(power),
		FanSpeed:       roundTwo(fan),
		UtilizationGPU:  roundTwo(utilGPU),
		UtilizationMem:  roundTwo(utilMem),
	}, nil
}

func splitCSV(s string) []string {
	var fields []string
	var current strings.Builder
	inQuotes := false

	for _, r := range s {
		switch r {
		case '"':
			inQuotes = !inQuotes
		case ',':
			if !inQuotes {
				fields = append(fields, current.String())
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}
	fields = append(fields, current.String())
	return fields
}
