package app

import (
	"fmt"

	appspy "github.com/chentao-kernel/spycat/internal/app"
	"github.com/chentao-kernel/spycat/pkg/core"
	"github.com/chentao-kernel/spycat/pkg/ebpf/cpu"
	"github.com/chentao-kernel/spycat/pkg/log"
)

func Start() {
	err, spy := appspy.NewAppSpy()
	if err != nil {
		fmt.Printf("New App Spy failed:%v\n", err)
	}
	err = spy.Start()
	if err != nil {
		fmt.Printf("Spy start failed:%v\n", err)
		spy.Stop()
	}

	fmt.Println("App Spy Start Success")
	log.Loger.Info("App Spy Start Success.")
	receiver := spy.GetReceiver()
	var BpfSpyers = []core.BpfSpyer{
		//uprobe.NewBpfSession("uprobe", &core.SessionConfig{}),
		cpu.NewBpfSession("offcpu", &core.SessionConfig{}, receiver.RcvChan()),
	}
	for _, spyer := range BpfSpyers {
		go func() {
			err := spyer.Start()
			if err != nil {
				fmt.Printf("bpfspy:{%s}, start failed:%v\n", spyer.Name(), err)
				log.Loger.Error("bpfspy:{%s}, start failed:%v\n", spyer.Name(), err)
			}
		}()
	}
	fmt.Println("Bpf Spy Start All")
	log.Loger.Info("Bpf Spy Start All.")
}
