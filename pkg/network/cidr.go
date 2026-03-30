package network

import (
	"fmt"
	"net"
)

type CIDRInfo struct {
	IP      net.IP
	Mask    net.IPMask
	Gateway net.IP
}

// ParseCIDR parses a CIDR string (e.g. "192.168.1.5/24") and returns the
// host IP, subnet mask, and the conventional gateway (first host in the subnet).
func ParseCIDR(cidr string) (CIDRInfo, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return CIDRInfo{}, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
	}

	// net.ParseCIDR returns the host address separately from the network
	// address; keep the 4-byte form for IPv4 so callers get consistent types.
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}

	// Derive the gateway as the first host address in the subnet (network
	// address + 1), which is the standard convention used by most hypervisors
	// and CNI plugins.
	gateway := make(net.IP, len(ipNet.IP))
	copy(gateway, ipNet.IP)
	gateway[len(gateway)-1]++
	return CIDRInfo{
		IP:      ip,
		Mask:    ipNet.Mask,
		Gateway: gateway,
	}, nil
}
