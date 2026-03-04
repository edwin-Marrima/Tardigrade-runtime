package ipam

import (
	"fmt"
	"net"

	"github.com/containernetworking/cni/pkg/types"
)

type IPAMConfig struct {
	Range Range `json:"range"`
}

type Range struct {
	Subnet  types.IPNet `json:"subnet"`
	Gateway net.IP      `json:"gateway,omitempty"`
}

func (r *Range) Contains(addr net.IP) bool {
	if err := canonicalizeIP(&addr); err != nil {
		return false
	}

	subnet := (net.IPNet)(r.Subnet)

	// Not the same address family
	if len(addr) != len(r.Subnet.IP) {
		return false
	}

	// Not in network
	if !subnet.Contains(addr) {
		return false
	}

	return true
}

func canonicalizeIP(ip *net.IP) error {
	if ip.To4() != nil {
		*ip = ip.To4()
		return nil
	} else if ip.To16() != nil {
		*ip = ip.To16()
		return nil
	}
	return fmt.Errorf("IP %s not v4 nor v6", *ip)
}
