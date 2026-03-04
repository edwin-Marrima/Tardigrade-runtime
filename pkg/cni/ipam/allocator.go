package ipam

import (
	"fmt"

	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/plugins/ipam/host-local/backend"
	log "github.com/sirupsen/logrus"

	"net"
	"os"
	//_ "github.com/containernetworking/plugins/plugins/ipam/host-local/backend/disk"
)

type IPAllocator struct {
	netRange *Range
	store    backend.Store
	rangeID  string // Used for tracking last reserved ip
}

func NewIPAllocator(s *Range, store backend.Store) *IPAllocator {
	return &IPAllocator{
		netRange: s,
		store:    store,
		rangeID:  "0",
	}
}

func (c *IPAllocator) AllocateIP(id string, ifname string, requestedIP net.IP) (*current.IPConfig, error) {
	// get allocated IP for given ID in order to avoid duplicated allocations
	allocatedIps := c.store.GetByID(id, ifname)
	for _, allocatedIP := range allocatedIps {
		// check whether the existing IP belong to this range set
		fmt.Println(c.netRange.Contains(allocatedIP))
		if c.netRange.Contains(allocatedIP) {
			return nil, fmt.Errorf("%s has been allocated to %s, duplicate allocation is not allowed", allocatedIP.String(), id)
		}
	}

	// retrieve latest allocated IP
	lastReservedIP, err := c.store.LastReservedIP(c.rangeID)
	if err != nil && !os.IsNotExist(err) {
		log.WithField("id", id).WithError(err).Error("Error retrieving last reserved ip")
	}

	subnet := (net.IPNet)(c.netRange.Subnet)
	gateway := c.netRange.Gateway
	broadcastIP := broadcastAddr(subnet)
	// Initialize the candidate IP.
	// Note: Ensure your ip.NextIP function doesn't mutate lastReservedIP in place if you need to keep it.
	var candidateIP net.IP
	if lastReservedIP != nil {
		candidateIP = ip.NextIP(lastReservedIP)
	} else {
		// Fallback to the start of the range if no IP was previously reserved
		// Ignore error because the validation is done before
		firstUsableIPAddr, _ := firstUsableIP(subnet)
		candidateIP = firstUsableIPAddr
	}
	for {
		// 1. Check if the candidate IP is still within the network range limits
		if !c.netRange.Contains(candidateIP) || candidateIP.Equal(broadcastIP) {
			break // Reached the end of this range
		}

		// 2. Check if the candidate IP is the gateway IP
		if candidateIP.Equal(gateway) {
			candidateIP = ip.NextIP(candidateIP)
			continue
		}

		// 3. Attempt to reserve the IP in the datastore
		reserved, err := c.store.Reserve(id, ifname, candidateIP, c.rangeID)
		if err != nil {
			return nil, fmt.Errorf("error reserving IP %s: %v", candidateIP.String(), err)
		}

		if reserved {
			// Successfully allocated! Build and return the IP configuration
			return &current.IPConfig{
				Address: net.IPNet{IP: candidateIP, Mask: subnet.Mask},
				Gateway: gateway,
			}, nil
		}

		// If not reserved, try the next one
		candidateIP = ip.NextIP(candidateIP)
	}

	return nil, fmt.Errorf("no available IP addresses in range set")
}
func (c *IPAllocator) Release(id string, ifname string) error {
	c.store.Lock()
	defer c.store.Unlock()
	return c.store.ReleaseByID(id, ifname)
}
func firstUsableIP(ipnet net.IPNet) (net.IP, error) {
	ip := ipnet.IP.To4()
	if ip == nil {
		return nil, fmt.Errorf("not an IPv4 subnet")
	}

	maskSize, bits := ipnet.Mask.Size()

	// /32 -> single host
	if maskSize == bits {
		return ip, nil
	}

	// /31 -> both addresses usable (RFC 3021)
	if maskSize == bits-1 {
		return ip, nil
	}

	// Normal subnet -> network + 1
	first := make(net.IP, len(ip))
	copy(first, ip)
	first[3]++

	return first, nil
}
func broadcastAddr(n net.IPNet) net.IP {
	ip := n.IP.To4()
	mask := n.Mask

	broadcast := make(net.IP, len(ip))
	for i := 0; i < len(ip); i++ {
		broadcast[i] = ip[i] | ^mask[i]
	}

	return broadcast
}
