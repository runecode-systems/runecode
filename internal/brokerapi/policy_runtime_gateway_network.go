package brokerapi

import (
	"net"
	"sync"
)

func isDeniedRuntimeIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, cidr := range runtimeDeniedIPRanges() {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

var (
	runtimeDeniedOnce sync.Once
	runtimeDeniedNets []*net.IPNet
)

func runtimeDeniedIPRanges() []*net.IPNet {
	runtimeDeniedOnce.Do(func() {
		for _, raw := range []string{
			"0.0.0.0/8",
			"100.64.0.0/10",
			"169.254.0.0/16",
			"224.0.0.0/4",
			"240.0.0.0/4",
			"::/128",
			"::1/128",
			"fc00::/7",
			"fe80::/10",
		} {
			_, network, err := net.ParseCIDR(raw)
			if err == nil {
				runtimeDeniedNets = append(runtimeDeniedNets, network)
			}
		}
	})
	return runtimeDeniedNets
}
