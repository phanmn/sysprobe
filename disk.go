package sysprobe

import (
	"time"

	"github.com/shirou/gopsutil/v4/disk"
)

func diskIOCollect(prev DiskIOTickState) ([]DiskIOMetric, DiskIOTickState, error) {
	ioCounters, err := disk.IOCounters()
	if err != nil {
		return nil, prev, err
	}

	var metrics []DiskIOMetric
	now := time.Now()
	newCounters := make(map[string]diskIOCounters)

	for name, c := range ioCounters {
		prevC, hasPrev := prev.Counters[name]
		if hasPrev && prevC.ReadBytes >= c.ReadBytes && prevC.WriteBytes >= c.WriteBytes {
			var dt float64
			if !prevC.Time.IsZero() {
				dt = now.Sub(prevC.Time).Seconds()
			} else {
				dt = 1
			}

			readDelta := float64(c.ReadBytes-prevC.ReadBytes) / 1048576 / dt
			writeDelta := float64(c.WriteBytes-prevC.WriteBytes) / 1048576 / dt
			iopsRead := float64(c.ReadCount-prevC.ReadCount) / dt
			iopsWrite := float64(c.WriteCount-prevC.WriteCount) / dt

			metrics = append(metrics, DiskIOMetric{
				Name:      name,
				ReadMB:    roundTwo(readDelta),
				WriteMB:   roundTwo(writeDelta),
				IOPSRead:  roundTwo(iopsRead),
				IOPSWrite: roundTwo(iopsWrite),
			})
		} else if !hasPrev {
			metrics = append(metrics, DiskIOMetric{
				Name:      name,
				ReadMB:    0,
				WriteMB:   0,
				IOPSRead:  0,
				IOPSWrite: 0,
			})
		}

		newCounters[name] = diskIOCounters{
			ReadBytes:  c.ReadBytes,
			WriteBytes: c.WriteBytes,
			ReadCount:  c.ReadCount,
			WriteCount: c.WriteCount,
			Time:       now,
		}
	}

	return metrics, DiskIOTickState{Counters: newCounters}, nil
}

func diskSpaceCollect() ([]DiskSpaceMetric, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	var metrics []DiskSpaceMetric
	for _, p := range partitions {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}

		totalGB := roundTwo(float64(usage.Total) / 1073741824)
		freeGB := roundTwo(float64(usage.Free) / 1073741824)
		usedGB := roundTwo(float64(usage.Used) / 1073741824)

		metrics = append(metrics, DiskSpaceMetric{
			Path:        p.Mountpoint,
			Device:      p.Device,
			FSType:      p.Fstype,
			Total:       totalGB,
			Free:        freeGB,
			Used:        usedGB,
			UsedPercent: roundTwo(usage.UsedPercent),
		})
	}

	return metrics, nil
}
