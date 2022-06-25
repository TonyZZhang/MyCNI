package main

import (
	"github.com/containernetworking/cni/pkg/version"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
	"mycni/skel"
	utils "mycni/util"
)

func cmdAdd(args *skel.CmdArgs) error {
	utils.WriteLog("进入到 cmdAdd")
	utils.WriteLog(
		"这里的 CmdArgs 是: ", "ContainerID: ", args.ContainerID,
		"Netns: ", args.Netns,
		"IfName: ", args.IfName,
		"Args: ", args.Args,
		"Path: ", args.Path,
		"StdinData: ", string(args.StdinData))
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
