package netlinktool

import (
	"fmt"
	"github.com/vishvananda/netlink"
	utils "mycni/util"
	"net"
)

//闭包的学习
type SampleNs interface {
	Do(toRun func(SampleNs) error) error
}

type sampleNs struct {
	name string
}

func (ns *sampleNs) Do(toRun func(SampleNs) error) error {
	innnerNS := &sampleNs{}
	innnerNS.name = "inner ns"
	toRun(innnerNS)
	return nil
}
func DoDo(){
	myNs := &sampleNs{}
	myNs.name = "myNs"
	err := myNs.Do(func(hostNs SampleNs) error {
		fmt.Println(hostNs)
		//此处打印结果 &{inner ns}
		return nil
	})
	fmt.Println(err)
}

type podNet struct {
	netNamespace string
	iframe string
	IP string
}

type nodeBridge struct {
	name string
	IP string
	mut int
}

//在宿主机上创建bridge
func (b *nodeBridge)createBridgeOnHost() (*netlink.Bridge, error) {
	//检查宿主机上是否已经存在bridge
	l, _ := netlink.LinkByName(b.name)
	br, ok := l.(*netlink.Bridge)
	if ok && br != nil {
		return br, nil
	}

	br = &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name:   b.name,
			MTU:    b.mut,
			TxQLen: -1,
		},
	}

	err := netlink.LinkAdd(br)
	if err != nil {
		utils.WriteLog("无法创建网桥: ", b.name, "err: ", err.Error())
		return nil, err
	}

	// 这里需要通过 netlink 重新获取网桥
	// 否则光创建的话无法从上头拿到其他属性
	l, err = netlink.LinkByName(b.name)

	br, ok = l.(*netlink.Bridge)
	if !ok {
		utils.WriteLog("找到了设备, 但是该设备不是网桥")
		return nil, fmt.Errorf("找到 %q 但该设备不是网桥", b.name)
	}

	// 给网桥绑定 ip 地址, 让网桥作为网关
	ipaddr, ipnet, err := net.ParseCIDR(b.IP)
	if err != nil {
		utils.WriteLog("无法 parse gw 为 ipnet, err: ", err.Error())
		return nil, fmt.Errorf("gatewayIP 转换失败 %q: %v", b.IP, err)
	}
	ipnet.IP = ipaddr
	addr := &netlink.Addr{IPNet: ipnet}
	if err = netlink.AddrAdd(br, addr); err != nil {
		utils.WriteLog("将 gw 添加到 bridge 失败, err: ", err.Error())
		return nil, fmt.Errorf("无法将 %q 添加到网桥设备 %q, err: %v", addr, b.name, err)
	}

	// 然后还要把这个网桥给 up 起来
	if err = netlink.LinkSetUp(br); err != nil {
		utils.WriteLog("启动网桥失败, err: ", err.Error())
		return nil, fmt.Errorf("启动网桥 %q 失败, err: %v", b.name, err)
	}
	return br, nil
}



