package sysprobe

import (
	"net"
	"strings"
	"time"

	psnet "github.com/shirou/gopsutil/v4/net"
)

func networkCollect(opts Options, prev NetworkPreviousState) ([]NetworkMetric, NetworkPreviousState, error) {
	counters, err := psnet.IOCounters(true)
	if err != nil {
		return nil, prev, err
	}

	minMAC := opts.MACMinLength
	if minMAC == 0 {
		minMAC = 6
	}

	// Build map of interface name → has public IP (v4 or v6) address
	publicIPMap := make(map[string]bool)
	ifaces, err := psnet.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			for _, addr := range iface.Addrs {
				ipStr := strings.Split(addr.Addr, "/")[0]
				ip := net.ParseIP(ipStr)
				if ip != nil && isPublicIP(ip) {
					publicIPMap[iface.Name] = true
				}
			}
		}
	}

	var metrics []NetworkMetric
	now := time.Now()
	newCounters := make(map[string]netCounters)

	for _, c := range counters {
		info := getLinkInfo(c.Name)

		// Skip loopback unless explicitly requested
		if info.IsLoopback && !opts.IncludeLoopback {
			continue
		}

		// Skip tunnel interfaces on Linux unless explicitly requested
		if info.IsTunnel && !opts.IncludeTunnel {
			continue
		}

		// Skip interfaces with short MAC addresses (tunnels, etc.)
		if len(info.HardwareAddr) < minMAC {
			continue
		}

		prevC, hasPrev := prev.Counters[c.Name]
		if hasPrev {
			var dt float64
			if !prevC.Time.IsZero() {
				dt = now.Sub(prevC.Time).Seconds()
			} else {
				dt = 1
			}

			sentBps := float64(c.BytesSent-prevC.Sent) / dt
			receivedBps := float64(c.BytesRecv-prevC.Received) / dt

			metrics = append(metrics, NetworkMetric{
				Name:        c.Name,
				MAC:         info.HardwareAddr,
				MTU:         info.MTU,
				SentBps:     roundTwo(sentBps),
				ReceivedBps: roundTwo(receivedBps),
				HasPublicIP: publicIPMap[c.Name],
			})
		} else {
			metrics = append(metrics, NetworkMetric{
				Name:        c.Name,
				MAC:         info.HardwareAddr,
				MTU:         info.MTU,
				SentBps:     0,
				ReceivedBps: 0,
				HasPublicIP: publicIPMap[c.Name],
			})
		}

		newCounters[c.Name] = netCounters{
			Sent:     c.BytesSent,
			Received: c.BytesRecv,
			Time:     now,
		}
	}

	return metrics, NetworkPreviousState{Counters: newCounters}, nil
}

// isPublicIP returns true if the address is globally routable (not private/reserved).
func isPublicIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		return !isPrivateIPv4(ip4)
	}
	// IPv6 — check for reserved ranges
	return !isPrivateIPv6(ip)
}

func isPrivateIPv4(ip net.IP) bool {
	switch {
	case ip[0] == 10: // 10.0.0.0/8
		return true
	case ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31: // 172.16.0.0/12
		return true
	case ip[0] == 192 && ip[1] == 168: // 192.168.0.0/16
		return true
	case ip[0] == 127: // loopback
		return true
	case ip[0] == 169 && ip[1] == 254: // link-local
		return true
	default:
		return false
	}
}

func isPrivateIPv6(ip net.IP) bool {
	switch {
	case ip.Equal(net.IPv6zero) || ip.Equal(net.IPv6loopback): // ::/128, ::1/128
		return true
	case ip[0] == 0xfe && ip[1]&0xc0 == 0x80: // fe80::/10 — link-local
		return true
	case ip[0]&0xfe == 0xfc: // fc00::/7 — unique local unicast
		return true
	case ip[0] == 0xff: // ff00::/8 — multicast
		return true
	default:
		return false
	}
}
