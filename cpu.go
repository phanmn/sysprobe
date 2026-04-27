package sysprobe

import (
	"github.com/shirou/gopsutil/v4/cpu"
)

func cpuCollect(prev CPUTickState) (*CPUMetrics, CPUTickState, error) {
	times, err := cpu.Times(true)
	if err != nil {
		return nil, prev, err
	}

	metrics := &CPUMetrics{
		Cores: make([]float64, len(times)),
	}

	coreTotals := make([]float64, len(times))
	for i, t := range times {
		if i < len(prev.Times) {
			p := prev.Times[i]
			userDelta := t.User - p.User
			systemDelta := t.System - p.System
			prevTotal := cpuTotalTime(p)
			coreDelta := cpuTotalTime(t) - prevTotal
			if coreDelta > 0 {
				usage := (userDelta + systemDelta) / coreDelta * 100
				metrics.Cores[i] = roundTwo(usage)
			}
		}
		coreTotals[i] = metrics.Cores[i]
	}

	// Average is mean of all cores
	if len(coreTotals) > 0 {
		var sum float64
		for _, c := range coreTotals {
			sum += c
		}
		metrics.Average = roundTwo(sum / float64(len(coreTotals)))
	}

	newPrev := CPUTickState{Times: times}
	return metrics, newPrev, nil
}
