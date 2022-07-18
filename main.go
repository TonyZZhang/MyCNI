package main

import (
	"encoding/json"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/version"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
	"mycni/netlinktool"
	"mycni/skel"
	utils "mycni/util"
	"os"
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
	//实现思路
	//1)接收创建pod网络的请求，解析出pod的网络namespace以及网卡名
	//2)宿主机上如果没有pod间通信的bridge,则创建bridge
	//3)创建veth pair,一端的名字叫做传进来的IfName，然后将veth pair一端放进pod Namespace，一端放进pod bridge
	//4)为pod以及bridge分配IP

	//3.实现不同Node之间Pod网络互通

	//每一个Node分配同一个子网下的不同网段
	//pluginConfig.Subnet
	//IP := ""
	//实现同一Node之间Pod网络互通
	podIP := ""
	nodeBridgeIP := ""
	hostName,_ := os.Hostname()
	nodeBridgeName := hostName + "_bridge"
	n := netlinktool.NewPodNet(args.Netns, args.IfName, podIP, 50)
	b := netlinktool.NewNodeBridge(nodeBridgeName, nodeBridgeIP, 50)
	err = netlinktool.CreatePodNetInSameNode(b, n)
	if err != nil {
		return err
	}


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
