package main

import (
	"encoding/json"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/version"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
	"mycni/skel"
	utils "mycni/util"
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
