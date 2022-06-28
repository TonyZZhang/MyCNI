package ipam

import (
	"fmt"
	"strconv"
	"strings"
)


type Ipam struct {
	subnet string
	SegmentMask string
}

func (i *Ipam) SetSegmentMask()  {
	strs := strings.Split(i.subnet, "/")
	if len(strs) != 2 {
		return
	}
	i.SegmentMask = strs[0]
}

func (i *Ipam) GenerateNodeNetSegmentPool() string{
	fmt.Println(i.SegmentMask)
	ipSeg := strings.Split(i.SegmentMask, ".")
	var temp int
	var nodeIpSeg, ipPool string
	for k, v := range ipSeg {
		if v == "0" {
			temp = k
			break
		}
	}
	for i:=0; i< 256; i++ {
		ipSeg[temp] = strconv.Itoa(i)
		nodeIpSeg = strings.Join(ipSeg, ".")
		ipPool += nodeIpSeg
		ipPool += ";"
	}
	return ipPool
}

func (i *Ipam) SetNodeNetSegment(nodeName string) {

}