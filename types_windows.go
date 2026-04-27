//go:build windows

package sysprobe

import (
	"net"

	"github.com/shirou/gopsutil/v4/cpu"
)

type linkInfo struct {
	Name         string
	HardwareAddr string
	MTU          int
	IsLoopback   bool
	IsTunnel     bool
}

func getLinkInfo(name string) linkInfo {
	iface, _ := net.InterfaceByName(name)
	return linkInfo{
		Name:         name,
		HardwareAddr: formatMAC(iface.HardwareAddr),
		MTU:          iface.MTU,
		IsLoopback:   iface.Flags&net.FlagLoopback != 0,
		IsTunnel:     false,
	}
}

func formatMAC(hw []byte) string {
	if len(hw) == 0 {
		return ""
	}
	return net.HardwareAddr(hw).String()
}

func cpuTotalTime(t cpu.TimesStat) float64 {
	return t.User + t.System + t.Idle + t.Nice + t.Irq +
		t.Iowait + t.Steal + t.Guest + t.GuestNice
}
