package ipam

import (
	"encoding/json"
	"fmt"
	"net"
	"syscall"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/netlinksafe"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/containernetworking/plugins/plugins/ipam/host-local/backend/disk"
	"github.com/vishvananda/netlink"

	"github.com/containernetworking/plugins/plugins/ipam/host-local/backend/allocator"
)

type Bridge struct {
	BrName string `json:"bridge"`
	IsGW   bool   `json:"isGateway"`
	MTU    int    `json:"mtu"`
}

type NetConf struct {
	types.NetConf
	IPAM          allocator.IPAMConfig `json:"ipam,omitempty"`
	Bridge        Bridge               `json:"bridge,omitempty"`
	IPMasq        bool                 `json:"ipMasq"`
	IPMasqBackend *string              `json:"ipMasqBackend,omitempty"`
	MTU           int                  `json:"mtu"`
}

func setupContainerVeth(netns ns.NetNS, ifName string, mtu int, pr *current.Result) (*current.Interface, *current.Interface, error) {
	// The IPAM result will be something like IP=192.168.3.5/24, GW=192.168.3.1.
	// What we want is really a point-to-point link but veth does not support IFF_POINTTOPOINT.
	// Next best thing would be to let it ARP but set interface to 192.168.3.5/32 and
	// add a route like "192.168.3.0/24 via 192.168.3.1 dev $ifName".
	// Unfortunately that won't work as the GW will be outside the interface's subnet.

	// Our solution is to configure the interface with 192.168.3.5/24, then delete the
	// "192.168.3.0/24 dev $ifName" route that was automatically added. Then we add
	// "192.168.3.1/32 dev $ifName" and "192.168.3.0/24 via 192.168.3.1 dev $ifName".
	// In other words we force all traffic to ARP via the gateway except for GW itself.

	hostInterface := &current.Interface{}
	containerInterface := &current.Interface{}

	err := netns.Do(func(hostNS ns.NetNS) error {
		hostVeth, contVeth0, err := ip.SetupVeth(ifName, mtu, "", hostNS)
		if err != nil {
			return err
		}

		hostInterface.Name = hostVeth.Name
		hostInterface.Mac = hostVeth.HardwareAddr.String()
		containerInterface.Name = contVeth0.Name
		containerInterface.Mac = contVeth0.HardwareAddr.String()
		containerInterface.Sandbox = netns.Path()

		for _, ipc := range pr.IPs {
			// All addresses apply to the container veth interface
			ipc.Interface = current.Int(1)
		}

		pr.Interfaces = []*current.Interface{hostInterface, containerInterface}

		// should not assign IP address to veth pair within namespace
		//if err = ipam.ConfigureIface(ifName, pr); err != nil {
		//	return err
		//}
		// create tap device
		tapName := "tap0"
		tap := &netlink.Tuntap{
			LinkAttrs: netlink.LinkAttrs{
				Name: tapName,
			},
			Mode:       netlink.TUNTAP_MODE_TAP,
			NonPersist: true,
		}
		if err := netlink.LinkAdd(tap); err != nil {
			return fmt.Errorf("failed to create TAP device %s: %w", tapName, err)
		}
		tapLink, err := netlink.LinkByName(tapName)
		if err != nil {
			return fmt.Errorf("failed to get TAP device %s: %w", tapName, err)
		}
		if err := netlink.LinkSetUp(tapLink); err != nil {
			return fmt.Errorf("failed to bring up TAP device %s: %w", tapName, err)
		}
		//  create bridge within namespace
		bridgeName := "br0"
		linkAttrs := netlink.NewLinkAttrs()
		linkAttrs.Name = bridgeName
		br := &netlink.Bridge{
			LinkAttrs: linkAttrs,
		}
		err = netlink.LinkAdd(br)
		if err != nil && err != syscall.EEXIST {
			return fmt.Errorf("could not add bridge %q: %v", bridgeName, err)
		}
		if err := netlink.LinkSetUp(br); err != nil {
			return fmt.Errorf("failed to bring up bridge %q: %v", bridgeName, err)
		}
		// assign ip to bridge
		for _, ipc := range pr.IPs {
			ipn := &net.IPNet{
				IP:   ipc.Address.IP,
				Mask: ipc.Address.Mask,
			}
			addr := &netlink.Addr{IPNet: ipn, Label: ""}
			if err = netlink.AddrAdd(br, addr); err != nil {
				return fmt.Errorf("failed to add IP addr (%#v) to container bridge %s: %v", ipn, bridgeName, err)
			}
		}
		// bind veth to bridge
		containerVeth, err := netlinksafe.LinkByName(contVeth0.Name)
		if err != nil {
			return fmt.Errorf("failed to lookup %q: %v", contVeth0.Name, err)
		}
		err = netlink.LinkSetMaster(containerVeth, br)
		if err != nil {
			return fmt.Errorf("failed to attach veth '%s' to bridge %s: %v", contVeth0.Name, br.Name, err)
		}
		// bind tap device to bridge
		err = netlink.LinkSetMaster(tapLink, br)
		if err != nil {
			return fmt.Errorf("failed to attach tap '%s' to bridge %s: %v", tapName, br.Name, err)
		}

		for _, ipc := range pr.IPs {
			// Delete the route that was automatically added when the IP was
			// assigned to br (auto-added on br.Index, not on the veth).
			route := netlink.Route{
				LinkIndex: br.Index,
				Dst: &net.IPNet{
					IP:   ipc.Address.IP.Mask(ipc.Address.Mask),
					Mask: ipc.Address.Mask,
				},
				Scope: netlink.SCOPE_NOWHERE,
			}

			if err := netlink.RouteDel(&route); err != nil {
				return fmt.Errorf("failed to delete route %v: %v", route, err)
			}

			addrBits := 32
			if ipc.Address.IP.To4() == nil {
				addrBits = 128
			}

			for _, r := range []netlink.Route{
				//route add ${IP_GW}/32 dev $BR_NS
				{
					LinkIndex: br.Index,
					Dst: &net.IPNet{
						IP:   ipc.Gateway,
						Mask: net.CIDRMask(addrBits, addrBits),
					},
					Scope: netlink.SCOPE_LINK,
				},
				// route add IP/mask via $IP_GW dev $BR_NS
				{
					LinkIndex: br.Index,
					Dst: &net.IPNet{
						IP:   ipc.Address.IP.Mask(ipc.Address.Mask),
						Mask: ipc.Address.Mask,
					},
					Scope: netlink.SCOPE_UNIVERSE,
					Gw:    ipc.Gateway,
					Src:   ipc.Address.IP,
				},
				//route add default via $IP_GW dev $BR_NS
				{
					LinkIndex: br.Index,
					Dst:       nil, // A nil Dst in netlink automatically means 0.0.0.0/0 (Default Route)
					Gw:        ipc.Gateway,
				},
			} {
				if err := netlink.RouteAdd(&r); err != nil {
					return fmt.Errorf("failed to add route %v: %v", r, err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return hostInterface, containerInterface, nil
}

func setupHostVeth(vethName string, br *netlink.Bridge) error {
	veth, err := netlinksafe.LinkByName(vethName)
	if err != nil {
		return fmt.Errorf("failed to lookup %q: %v", vethName, err)
	}
	err = netlink.LinkSetMaster(veth, br)
	if err != nil {
		return fmt.Errorf("failed to attach %s to bridge %s: %v", vethName, br.Name, err)
	}
	return nil
}
func bridgeByName(name string) (*netlink.Bridge, error) {
	l, err := netlinksafe.LinkByName(name)
	if err != nil {
		return nil, fmt.Errorf("could not lookup %q: %v", name, err)
	}
	br, ok := l.(*netlink.Bridge)
	if !ok {
		return nil, fmt.Errorf("%q already exists but is not a bridge", name)
	}
	return br, nil
}
func setupHostBridge(brName string, mtu int, isGw bool, pr *current.Result) (*netlink.Bridge, error) {
	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = brName
	linkAttrs.MTU = mtu
	br := &netlink.Bridge{
		LinkAttrs: linkAttrs,
	}
	err := netlink.LinkAdd(br)
	if err != nil && err != syscall.EEXIST {
		return nil, fmt.Errorf("could not add %q: %v", brName, err)
	}
	br, err = bridgeByName(brName)
	if err != nil {
		return nil, err
	}

	// Turn off IPv6 auto-configuration (accept_ra = 0).
	// If left on, the kernel might listen to stray Router Advertisements on the network
	// and alter the routing table without our permission. We set this to 0 so we
	// explicitly own and manually program all routes for this bridge.
	_, _ = sysctl.Sysctl(fmt.Sprintf("net/ipv6/conf/%s/accept_ra", brName), "0")

	if isGw {
		for _, ipc := range pr.IPs {
			ipn := &net.IPNet{
				IP:   ipc.Gateway,
				Mask: ipc.Address.Mask,
			}
			addr := &netlink.Addr{IPNet: ipn, Label: ""}
			if err = netlink.AddrAdd(br, addr); err != nil {
				return nil, fmt.Errorf("failed to add IP addr (%#v) to host bridge %s: %v", ipn, brName, err)
			}
		}
	}

	if err := netlink.LinkSetUp(br); err != nil {
		return nil, err
	}

	return br, nil
}
func cmdAdd(args *skel.CmdArgs) error {
	conf := NetConf{}
	if err := json.Unmarshal(args.StdinData, &conf); err != nil {
		return fmt.Errorf("failed to load netconf: %v", err)
	}
	result := &current.Result{CNIVersion: current.ImplementedSpecVersion}
	store, err := disk.New(conf.Name, "")
	if err != nil {
		return err
	}
	ipRange := *conf.IPAM.Range
	rangeSet := allocator.RangeSet{ipRange}
	alloc := allocator.NewIPAllocator(&rangeSet, store, 0)
	// no custom IP is being requested
	ipConf, err := alloc.Get(args.ContainerID, args.IfName, nil)
	if err != nil {
		return fmt.Errorf("failed to allocate for range %s: %v", ipRange.Subnet.IP.String(), err)
	}

	result.IPs = append(result.IPs, ipConf)
	result.Routes = conf.IPAM.Routes

	// create bridge on host namespace
	bridge, err := setupHostBridge(conf.Bridge.BrName, conf.Bridge.MTU, conf.Bridge.IsGW, result)
	if err != nil {
		return fmt.Errorf("failed to setup host bridge: %v", err)
	}
	// setup veth-pairs
	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer netns.Close()
	hostInterface, _, err := setupContainerVeth(netns, args.IfName, conf.MTU, result)
	if err != nil {
		return err
	}

	if err = setupHostVeth(hostInterface.Name, bridge); err != nil {
		return err
	}

	return types.PrintResult(result, conf.CNIVersion)
}
