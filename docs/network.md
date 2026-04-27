# Network Collector — Component Guide

## Overview

Delta-based network I/O collector with platform-specific interface filtering and public IP detection. Computes bandwidth in bytes/sec from cumulative counters.

Source file: `network.go` (139 lines)

---

## How It Works

### Collection Flow

1. Calls `psnet.IOCounters(true)` for per-interface cumulative byte counters
2. Calls `psnet.Interfaces()` to build a public IP map per interface name
3. For each interface, applies filtering rules:
   - Skip loopback unless `opts.IncludeLoopback`
   - Skip tunnel interfaces (Linux/macOS) unless `opts.IncludeTunnel`
   - Skip interfaces with MAC length < `opts.MACMinLength` (default 6)
4. Computes bandwidth deltas from previous sample timestamps
5. Returns metrics with `HasPublicIP` flag set from step 2

### Bandwidth Calculation

```
sentBps     = (current.BytesSent   - prev.Sent)     / interval_seconds
receivedBps = (current.BytesRecv   - prev.Received) / interval_seconds
```

---

## Interface Filtering

| Rule | Default Behavior | Override Option |
|---|---|---|
| Loopback (`lo`, `lo0`) | Excluded | `IncludeLoopback: true` |
| Tunnel interfaces (Linux) | Excluded by name prefix | `IncludeTunnel: true` |
| Short MAC (< 6 bytes) | Excluded | `MACMinLength: 0` disables |

### Platform-Specific Tunnel Detection

- **Linux** (`types_linux.go`): Name prefix check — tun, tap, gre, gretap, ipip, sit, vti, erspan, ip6tnl, vxlan
- **macOS** (`types_darwin.go`): Name prefix check — utun, awdl, llw, bridge, veth, gif, stf, tun, tap, ipsec, key, p2p
- **Windows** (`types_windows.go`): No tunnel filtering (all interfaces pass through)

---

## Public IP Detection

### Algorithm

Checks all addresses assigned to the interface. Returns `true` if any address is globally routable.

| Protocol | Excluded Ranges |
|---|---|
| IPv4 | 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 127.0.0.0/8, 169.254.0.0/16 |
| IPv6 | ::/128, ::1/128, fe80::/10 (link-local), fc00::/7 (ULA), ff00::/8 (multicast) |

### Internal Functions

```go
func isPublicIP(ip net.IP) bool           // top-level check
func isPrivateIPv4(ip net.IP) bool        // v4 private/reserved ranges
func isPrivateIPv6(ip net.IP) bool        // v6 reserved ranges
```

---

## Types

```go
type NetworkMetric struct {
    Name          string  // interface name (e.g., "eth0", "en0")
    MAC           string  // hardware address ("xx:xx:xx:xx:xx:xx")
    MTU           int     // maximum transmission unit
    BytesOutPerSec       float64 // outbound bandwidth in bytes/sec
    BytesInPerSec   float64 // inbound bandwidth in bytes/sec
    HasPublicIP   bool    // true if any assigned IP is globally routable
}

type NetworkTickState struct {
    Counters map[string]netCounters // keyed by interface name
}

type netCounters struct { // internal, not exported
    Sent     uint64    // cumulative bytes sent
    Received uint64    // cumulative bytes received
    Time     time.Time // sample timestamp
}
```

---

## Usage Pattern

```go
metrics, newState, err := sysprobe.Collect(sysprobe.Options{
    Network:         true,
    IncludeLoopback: false,
    IncludeTunnel:   false,
}, prev)
if err != nil {
    log.Printf("network collect error: %v", err)
    return
}
prev = newState

for _, n := range metrics.Network {
    pubFlag := ""
    if n.HasPublicIP {
        pubFlag = " [PUBLIC]"
    }
    fmt.Printf("%s (%s, MTU %d): TX=%.0f B/s RX=%.0f B/s%s\n",
        n.Name, n.MAC, n.MTU, n.BytesOutPerSec, n.BytesInPerSec, pubFlag)
}
```

## Notes for LLMs

- Network metrics are **delta-based** — first tick returns zeros. Pass back `NetworkTickState` each tick.
- Bandwidth output is **bytes/sec** (`BytesOutPerSec`, `BytesInPerSec`), not MB/s or cumulative bytes.
- `HasPublicIP` is computed fresh each tick from current interface addresses.
- If `psnet.Interfaces()` fails, public IP detection is silently skipped (map stays empty).
- MAC addresses use standard `net.HardwareAddr.String()` format across all platforms.
- All rate values pass through `roundTwo()`.
