//go:build darwin

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

	// Detect tunnel/virtual interfaces by common macOS prefixes
	if info.HardwareAddr == "" && !info.IsLoopback {
		lower := strings.ToLower(name)
		info.IsTunnel = strings.HasPrefix(lower, "utun") ||
			strings.HasPrefix(lower, "awdl") ||
			strings.HasPrefix(lower, "llw") ||
			strings.HasPrefix(lower, "bridge") ||
			strings.HasPrefix(lower, "veth") ||
			strings.HasPrefix(lower, "gif") ||
			strings.HasPrefix(lower, "stf") ||
			strings.HasPrefix(lower, "tun") ||
			strings.HasPrefix(lower, "tap") ||
			strings.HasPrefix(lower, "ipsec") ||
			strings.HasPrefix(lower, "key") ||
			strings.HasPrefix(lower, "p2p")
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
