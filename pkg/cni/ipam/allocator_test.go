package ipam

import (
	"net"
	"testing"

	"github.com/containernetworking/cni/pkg/types"
	fakestore "github.com/containernetworking/plugins/plugins/ipam/host-local/backend/testing"
	"github.com/stretchr/testify/assert"
)

func TestIPAllocation(t *testing.T) {
	t.Run("must return error when container has IP allocated", func(t *testing.T) {
		netRange := &Range{Subnet: mustSubnet(t, "192.168.1.0/26"), Gateway: net.ParseIP("192.168.1.5")}
		store := fakestore.NewFakeStore(map[string]string{
			"192.168.1.3": "192.168.1.3",
		}, map[string]net.IP{})
		allocator := NewIPAllocator(netRange, store)
		v, err := allocator.AllocateIP("192.168.1.3", "", nil)
		assert.Nil(t, v)
		assert.ErrorContains(t, err, "192.168.1.3 has been allocated to 192.168.1.3, duplicate allocation is not allowed")
	})
	t.Run("must allocate the first IP in the range when lastAllocated IP is empty", func(t *testing.T) {
		netRange := &Range{Subnet: mustSubnet(t, "192.168.1.0/26"), Gateway: net.ParseIP("192.168.1.5")}
		store := fakestore.NewFakeStore(map[string]string{}, map[string]net.IP{})
		allocator := NewIPAllocator(netRange, store)
		v, err := allocator.AllocateIP("192.168.1.3", "", nil)
		assert.Nil(t, err)
		assert.Equal(t, "192.168.1.1", v.Address.IP.String())
	})
	t.Run("must allocate the next IP in the range when lastAllocated IP is not empty", func(t *testing.T) {
		netRange := &Range{Subnet: mustSubnet(t, "192.168.1.0/26"), Gateway: net.ParseIP("192.168.1.5")}
		store := fakestore.NewFakeStore(map[string]string{}, map[string]net.IP{
			"0": net.ParseIP("192.168.1.6"),
		})
		allocator := NewIPAllocator(netRange, store)
		v, err := allocator.AllocateIP("id", "", nil)
		assert.Nil(t, err)
		assert.Equal(t, "192.168.1.7", v.Address.IP.String())
	})
	t.Run("must return error when there's no available IP addresses in range set", func(t *testing.T) {
		netRange := &Range{Subnet: mustSubnet(t, "192.168.1.0/26"), Gateway: net.ParseIP("192.168.1.5")}
		store := fakestore.NewFakeStore(map[string]string{}, map[string]net.IP{
			"0": net.ParseIP("192.168.1.62"),
		})
		allocator := NewIPAllocator(netRange, store)
		v, err := allocator.AllocateIP("id", "", nil)
		assert.Nil(t, v)
		assert.ErrorContains(t, err, "no available IP addresses in range set")
	})
}
func mustSubnet(t *testing.T, s string) types.IPNet {
	n, err := types.ParseCIDR(s)
	assert.Nil(t, err, "failed to parse CIDR")
	canonicalizeIP(&n.IP)
	return types.IPNet(*n)
}

func networkSubnet(t *testing.T, s string) types.IPNet {
	net := mustSubnet(t, s)
	net.IP = net.IP.Mask(net.Mask)
	return net
}
