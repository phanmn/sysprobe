package sysprobe

import (
	"github.com/shirou/gopsutil/v4/mem"
)

func memoryCollect() (*MemoryMetrics, error) {
	vmem, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	swap, err := mem.SwapMemory()
	if err != nil {
		return nil, err
	}

	return &MemoryMetrics{
		Total:       vmem.Total,
		Used:        vmem.Used,
		UsedPercent: roundTwo(vmem.UsedPercent),
		Available:   vmem.Available,
		SwapTotal:   swap.Total,
		SwapUsed:    swap.Used,
		SwapUsedPct: roundTwo(swap.UsedPercent),
	}, nil
}
