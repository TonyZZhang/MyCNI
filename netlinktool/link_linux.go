package netlinktool

import (
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"
	"net"
)

func makeVethPair(name, peer string, mtu int) (netlink.Link, error){
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: name,
			Flags: net.FlagUp,
			MTU: mtu,
		},
		PeerName: peer,
	}
	if err := netlink.LinkAdd(veth); err != nil {
		return nil, err
	}

	return veth, nil
}

func peerExists(name string) bool {
	if _, err := netlink.LinkByName(name); err != nil {
		return false
	}
	return true
}

func SetupVeth(contVethName string, mtu int, hostNS ns.NetNS)(netlink.Veth, netlink.Veth){
	peerName := ""
	contVeth, err := makeVethPair(contVethName, peerName, mtu)
	if err != nil {
		return nil, nil
	}

	err = netlink.LinkSetUp(contVeth)
	if err != nil {
		return nil, nil
	}

	hostVeth, err := netlink.LinkByName(peerName)
	if err != nil {
		return nil, nil
	}

	if err = netlink.LinkSetNsFd(hostVeth, int(hostNS.Fd())); err != nil {
		return nil, nil
	}

	err = hostNS.Do(func(_ ns.NetNS) error {
		hostVeth, err = netlink.LinkByName(peerName)
		if err != nil {
			return err
		}

		if err = netlink.LinkSetUp(hostVeth); err != nil {
			return err
		}
		return nil
	})

	return hostVeth, contVeth
}

func createBridge(name string) (*netlink.Bridge, error) {
	br := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: name,
			MTU:  1500,
			// Let kernel use default txqueuelen; leaving it unset
			// means 0, and a zero-length TX queue messes up FIFO
			// traffic shapers which use TX queue length as the
			// default packet limit
			TxQLen: -1,
		},
	}

	err := netlink.LinkAdd(br)
	if err != nil && err != syscall.EEXIST {
		return nil, err
	}

	//Fetch the bridge Object, we need to use it for the veth
	l, err := netlink.LinkByName(name)
	if err != nil {
		return nil, fmt.Errorf("could not lookup %q: %v", name, err)
	}
	newBr, ok := l.(*netlink.Bridge)
	if !ok {
		return nil, fmt.Errorf("%q already exists but is not a bridge", name)
	}

	if err := netlink.LinkSetUp(br); err != nil {
		return nil, err
	}

	return newBr, nil
}

func CreateBridgeAndSetupVeth(bridgeName string, contNS ns.NetNS, ifName string) error{
	var hostVethName string
	mut := 1500
	bridge := createBridge(bridgeName)
	err := contNS.Do(func(hostNS ns.NetNS) error{
		hostVeth, contVeth := SetupVeth(ifName, mut, hostNS)
		hostVethName = hostVeth.Name
	})
	if err != nil {
		return err
	}

	hostVeth := netlink.LinkByName(hostVethName)
	if err = netlink.LinkSetMaster(hostVeth, bridge); err != nil {
		return err
	}

	return nil
}