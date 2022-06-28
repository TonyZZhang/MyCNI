package ipam

import "testing"

func TestName(t *testing.T) {
	ipam := &Ipam{
		subnet: "192.168.0.0/16",
	}

	ipam.SetSegmentMask()
	ipam.GenerateNodeNetSegmentPool()
}
