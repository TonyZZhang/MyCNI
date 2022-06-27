package ipam

import "strings"


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

func (i *Ipam) SetNodeNetSegment(nodeName string) {
	
}

}