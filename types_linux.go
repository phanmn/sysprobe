//go:build linux

package sysprobe

import (
	"net"
	"strings"

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
	info := linkInfo{
		Name:         name,
		HardwareAddr: formatMAC(iface.HardwareAddr),
		MTU:          iface.MTU,
		IsLoopback:   iface.Flags&net.FlagLoopback != 0,
	}

	// Detect tunnel interfaces by name patterns (no MAC addr + common prefixes)
	if info.HardwareAddr == "" && !info.IsLoopback {
		lower := strings.ToLower(name)
		info.IsTunnel = strings.HasPrefix(lower, "tun") ||
			strings.HasPrefix(lower, "tap") ||
			strings.HasPrefix(lower, "gre") ||
			strings.HasPrefix(lower, "gretap") ||
			strings.HasPrefix(lower, "ipip") ||
			strings.HasPrefix(lower, "sit") ||
			strings.HasPrefix(lower, "vti") ||
			strings.HasPrefix(lower, "erspan") ||
			strings.HasPrefix(lower, "ip6tnl") ||
			strings.HasPrefix(lower, "vxlan") ||
			strings.HasPrefix(lower, "gretap")
	}

	return info
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
