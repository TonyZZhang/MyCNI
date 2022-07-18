package netlinktool

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/coreos/go-iptables/iptables"
	"github.com/vishvananda/netlink"
	utils "mycni/util"
	"net"
	"os"
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
	mut int
}

func NewPodNet(netNamespace, iframe, IP string, mut int)*podNet{
	return &podNet{
		netNamespace: netNamespace,
		iframe: iframe,
		IP: IP,
		mut: mut,
	}
}

type nodeBridge struct {
	name string
	IP string
	mut int
}

func NewNodeBridge(name, IP string, mut int) *nodeBridge {
	return &nodeBridge{
		name: name,
		IP: IP,
		mut: mut,
	}
}

func CreatePodNetInSameNode(b *nodeBridge, p *podNet)error {
	// 先创建网桥
	br, err := b.createBridgeOnHost()
	if err != nil {
		utils.WriteLog("创建网卡失败, err: ", err.Error())
		return err
	}

	netns, err := ns.GetNS(p.netNamespace)
	if err != nil {
		return err
	}
	err = netns.Do(func(hostNs ns.NetNS) error {
		// 创建一对儿 veth 设备
		containerVeth, hostVeth, err := p.createVethPair()
		if err != nil {
			utils.WriteLog("创建 veth 失败, err: ", err.Error())
			return err
		}

		// 把随机起名的 veth 那头放在主机上
		err = setVethNsFd(hostVeth, hostNs)
		if err != nil {
			utils.WriteLog("把 veth 设置到 ns 下失败: ", err.Error())
			return err
		}

		// 然后把要被放到 pod 中的那头 veth 塞上 podIP
		err = setIpForVeth(containerVeth, p.IP)
		if err != nil {
			utils.WriteLog("给 veth 设置 ip 失败, err: ", err.Error())
			return err
		}

		// 然后启动它
		err = setUpVeth(containerVeth)
		if err != nil {
			utils.WriteLog("启动 veth pair 失败, err: ", err.Error())
			return err
		}

		// 启动之后给这个 netns 设置默认路由 以便让其他网段的包也能从 veth 走到网桥
		// TODO: 实测后发现还必须得写在这里, 如果写在下面 hostNs.Do 里头会报错目标 network 不可达(why?)
		gwNetIP, _, err := net.ParseCIDR(b.IP)
		if err != nil {
			utils.WriteLog("转换 gwip 失败, err:", err.Error())
			return err
		}

		// 给 pod(net ns) 中加一个默认路由规则, 该规则让匹配了 0.0.0.0 的都走上边创建的那个 container veth
		err = setDefaultRouteToVeth(gwNetIP, containerVeth)
		if err != nil {
			utils.WriteLog("SetDefaultRouteToVeth 时出错, err: ", err.Error())
			return err
		}

		hostNs.Do(func(_ ns.NetNS) error {
			// 重新获取一次 host 上的 veth, 因为 hostVeth 发生了改变
			_hostVeth, err := netlink.LinkByName(hostVeth.Attrs().Name)
			hostVeth = _hostVeth.(*netlink.Veth)
			if err != nil {
				utils.WriteLog("重新获取 hostVeth 失败, err: ", err.Error())
				return err
			}
			// 启动它
			err = setUpVeth(hostVeth)
			if err != nil {
				utils.WriteLog("启动 veth pair 失败, err: ", err.Error())
				return err
			}

			// 把它塞到网桥上
			err = setVethMaster(hostVeth, br)
			if err != nil {
				utils.WriteLog("挂载 veth 到网桥失败, err: ", err.Error())
				return err
			}

			// 都完事儿之后理论上同一台主机下的俩 netns(pod) 就能通信了
			// 如果无法通信, 有可能是 iptables 被设置了 forward drop
			// 需要用 iptables 允许网桥做转发
			err = setIptablesForBridgeToForwardAccept(br)
			if err != nil {
				return err
			}

			return nil
		})

		return nil
	})

	if err != nil {
		return err
	}
	return nil
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

func (p * podNet) createVethPair() (*netlink.Veth, *netlink.Veth, error) {
	vethPairName := ""
	for {
		_vname, err := randomVethName()
		vethPairName = _vname
		if err != nil {
			utils.WriteLog("生成随机 veth pair 名字失败, err: ", err.Error())
			return nil, nil, err
		}

		_, err = netlink.LinkByName(vethPairName)
		if err != nil && !os.IsExist(err) {
			// 上面生成随机名字可能会重名, 所以这里先尝试按照这个名字获取一下
			// 如果没有这个名字的设备, 那就可以 break 了
			break
		}
	}

	if vethPairName == "" {
		return nil, nil, errors.New("生成 veth pair name 失败")
	}

	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: p.iframe,
			// Flags:     net.FlagUp,
			MTU: p.mut,
			// Namespace: netlink.NsFd(int(ns.Fd())), // 先不设置 ns, 要不一会儿下头 LinkByName 时候找不到
		},
		PeerName: vethPairName,
		// PeerNamespace: netlink.NsFd(int(ns.Fd())),
	}

	// 创建 veth pair
	err := netlink.LinkAdd(veth)

	if err != nil {
		utils.WriteLog("创建 veth 设备失败, err: ", err.Error())
		return nil, nil, err
	}

	// 尝试重新获取 veth 设备看是否能成功
	veth1, err := netlink.LinkByName(p.iframe) // veth1 一会儿要在 pod(net ns) 里
	if err != nil {
		// 如果获取失败就尝试删掉
		netlink.LinkDel(veth1)
		utils.WriteLog("创建完 veth 但是获取失败, err: ", err.Error())
		return nil, nil, err
	}

	// 尝试重新获取 veth 设备看是否能成功
	veth2, err := netlink.LinkByName(vethPairName) // veth2 在主机上
	if err != nil {
		// 如果获取失败就尝试删掉
		netlink.LinkDel(veth2)
		utils.WriteLog("创建完 veth 但是获取失败, err: ", err.Error())
		return nil, nil, err
	}

	return veth1.(*netlink.Veth), veth2.(*netlink.Veth), nil
}

func randomVethName() (string, error) {
	entropy := make([]byte, 4)
	_, err := rand.Read(entropy)
	if err != nil {
		return "", fmt.Errorf("failed to generate random veth name: %v", err)
	}

	// NetworkManager (recent versions) will ignore veth devices that start with "veth"
	return fmt.Sprintf("veth%x", entropy), nil
}

func setVethNsFd(veth *netlink.Veth, ns ns.NetNS) error {
	err := netlink.LinkSetNsFd(veth, int(ns.Fd()))
	if err != nil {
		return fmt.Errorf("把 veth %q 干到 netns 上失败: %v", veth.Attrs().Name, err)
	}
	return nil
}

func setVethMaster(veth *netlink.Veth, br *netlink.Bridge) error {
	err := netlink.LinkSetMaster(veth, br)
	if err != nil {
		return fmt.Errorf("把 veth %q 干到网桥上失败: %v", veth.Attrs().Name, err)
	}
	return nil
}

func setIpForVeth(veth *netlink.Veth, podIP string) error {
	// 给 veth1 也就是 pod(net ns) 里的设备添加上 podIP
	ipaddr, ipnet, err := net.ParseCIDR(podIP)
	if err != nil {
		utils.WriteLog("转换 podIP 为 net 类型失败: ", podIP, " err: ", err.Error())
		return err
	}
	ipnet.IP = ipaddr
	err = netlink.AddrAdd(veth, &netlink.Addr{IPNet: ipnet})
	if err != nil {
		utils.WriteLog("给 veth 添加 podIP 失败, podIP 是: ", podIP, " err: ", err.Error())
		return err
	}

	return nil
}

func setUpVeth(veth ...*netlink.Veth) error {
	for _, v := range veth {
		// 启动 veth 设备
		err := netlink.LinkSetUp(v)
		if err != nil {
			utils.WriteLog("启动 veth1 失败, err: ", err.Error())
			return err
		}
	}
	return nil
}

func setDefaultRouteToVeth(gwIP net.IP, veth netlink.Link) error {
	return addDefaultRoute(gwIP, veth)
}

func addDefaultRoute(gw net.IP, dev netlink.Link) error {
	_, defNet, _ := net.ParseCIDR("0.0.0.0/0")
	return addRoute(defNet, gw, dev)
}

func addRoute(ipn *net.IPNet, gw net.IP, dev netlink.Link) error {
	return netlink.RouteAdd(&netlink.Route{
		LinkIndex: dev.Attrs().Index,
		Scope:     netlink.SCOPE_UNIVERSE,
		Dst:       ipn,
		Gw:        gw,
	})
}

func setIptablesForBridgeToForwardAccept(br *netlink.Bridge) error {
	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		utils.WriteLog("这里 NewWithProtocol 失败, err: ", err.Error())
		return err
	}
	err = ipt.Append("filter", "FORWARD", "-i", br.Attrs().Name, "-j", "ACCEPT")
	if err != nil {
		utils.WriteLog("这里 ipt.Append 失败, err: ", err.Error())
		return err
	}
	return nil
}