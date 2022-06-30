package main

import (
	"encoding/json"
	"fmt"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/version"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
	"github.com/vishvananda/netlink"
	"mycni/netlinktool"
	"mycni/skel"
	utils "mycni/util"
	"net"
)

type PluginConf struct {
	types.NetConf
	// 这里可以自由定义自己的 plugin 中配置了的参数然后自由处理
	Bridge string `json:"bridge"`
	Subnet string `json:"subnet"`
}

func cmdAdd(args *skel.CmdArgs) error {
	utils.WriteLog("进入到 cmdAdd")
	utils.WriteLog(
		"这里的 CmdArgs 是: ", "ContainerID: ", args.ContainerID,
		"Netns: ", args.Netns,
		"IfName: ", args.IfName,
		"Args: ", args.Args,
		"Path: ", args.Path,
		"StdinData: ", string(args.StdinData))

	pluginConfig := &PluginConf{}
	err := json.Unmarshal(args.StdinData, pluginConfig)
	if err != nil {
		utils.WriteLog(err.Error())
		return err
	}
	//1.给Pod分配IP
	//2.实现同一Node之间Pod网络互通
	//3.实现不同Node之间Pod网络互通

	//每一个Node分配同一个子网下的不同网段
	//pluginConfig.Subnet
	IP := ""
	//实现同一Node之间Pod网络互通
	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return err
	}

	err = netlinktool.CreateBridgeAndSetupVeth(pluginConfig.Bridge, netns, args.IfName)
	if err != nil {
		return err
	}

	//Setup a IP address
	err = netns.Do(func(hostNS ns.NetNS) error {
		// create the veth pair in the container and move host end into host netns

		link, err := netlink.LinkByName(args.IfName)
		if err != nil {
			return err
		}
		ipv4Addr, ipv4Net, err := net.ParseCIDR(IP)
		addr := &netlink.Addr{IPNet: ipv4Net, Label: ""}
		ipv4Net.IP = ipv4Addr
		if err = netlink.AddrAdd(link, addr); err != nil {
			return fmt.Errorf("failed to add IP addr %v to %q: %v", ipv4Net, args.IfName, err)
		}
		return nil
	})


	return nil
}

func cmdDel(args *skel.CmdArgs) error {
	return nil
}

func cmdCheck(args *skel.CmdArgs) error {
	return nil
}
func main()  {
	utils.WriteLog("-----")
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString("testcni"))
}
