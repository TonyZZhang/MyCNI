package netlinktool

import "fmt"

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

