package netlinktool

import (
	"fmt"
	"github.com/vishvananda/netlink"
	utils "mycni/util"
	"syscall"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/plugins/pkg/ns"
)

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


func CreateVethPair(ifName string) (*netlink.Veth, *netlink.Veth, error) {
	vethPairName := ""
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: ifName,
			// Flags:     net.FlagUp,
			MTU: 1500,
			// Namespace: netlink.NsFd(int(ns.Fd())), // 先不设置 ns, 要不一会儿下头 LinkByName 时候找不到
		},
		PeerName: vethPairName,
		// PeerNamespace: netlink.NsFd(int(ns.Fd())),
	}

	err := netlink.LinkAdd(veth)
	if err != nil {
		return nil, nil, err
	}

	// 尝试重新获取 veth 设备看是否能成功
	veth1, err := netlink.LinkByName(ifName) // veth1 一会儿要在 pod(net ns) 里
	if err != nil {
		// 如果获取失败就尝试删掉
		netlink.LinkDel(veth1)
		utils.WriteLog("创建完 veth 但是获取失败, err: ", err.Error())
		return nil, nil, err
	}

	veth2, err := netlink.LinkByName(vethPairName) // veth2 在主机上
	if err != nil {
		// 如果获取失败就尝试删掉
		netlink.LinkDel(veth2)
		utils.WriteLog("创建完 veth 但是获取失败, err: ", err.Error())
		return nil, nil, err
	}

	return veth1.(*netlink.Veth), veth2.(*netlink.Veth), nil
}

func setupVeth(netns ns.NetNS, br *netlink.Bridge, ifName string) error {
	guestIface := &current.Interface{}
	hostIface := &current.Interface{}
	mtu := 1500
	err := netns.Do(func(hostNS ns.NetNS) error {
		// create the veth pair in the container and move host end into host netns
		hostVeth, containerVeth, err := ip.SetupVeth(ifName, mtu, hostNS)
		if err != nil {
			return err
		}
		guestIface.Name = containerVeth.Name
		guestIface.Mac = containerVeth.HardwareAddr.String()
		guestIface.Sandbox = netns.Path()
		hostIface.Name = hostVeth.Name
		return nil
	})
	//
	fmt.Println(hostIface)
	if err != nil {
		return err
	}

	// need to lookup hostVeth again as its index has changed during ns move
	hostVeth, err := netlink.LinkByName(hostIface.Name)
	if err != nil {
		return fmt.Errorf("failed to lookup %q: %v", hostIface.Name, err)
	}
	hostIface.Mac = hostVeth.Attrs().HardwareAddr.String()

	// connect host veth end to the bridge
	if err := netlink.LinkSetMaster(hostVeth, br); err != nil {
		return fmt.Errorf("failed to connect %q to bridge %v: %v", hostVeth.Attrs().Name, br.Attrs().Name, err)
	}

	return nil
}
