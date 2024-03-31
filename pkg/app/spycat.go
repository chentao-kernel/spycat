package app

import (
	"fmt"

	appspy "github.com/chentao-kernel/spycat/internal/app"
	"github.com/chentao-kernel/spycat/pkg/core"
	"github.com/chentao-kernel/spycat/pkg/core/model"
	"github.com/chentao-kernel/spycat/pkg/ebpf/cpu"
	"github.com/chentao-kernel/spycat/pkg/log"
)

func Start() {
	spy, err := appspy.NewAppSpy()
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

	bpfSpyers := []core.BpfSpyer{
		cpu.NewOffCpuBpfSession(model.OnCpu, nil, receiver.RcvChan()),
		cpu.NewOnCpuBpfSession(model.OffCpu, nil, receiver.RcvChan()),
	}

	for _, spyer := range bpfSpyers {
		// just for lint-check
		tmp := spyer
		if tmp.Name() != model.OnCpu {
			continue
		}
		go func() {
			err := tmp.Start()
			if err != nil {
				log.Loger.Error("bpfspy:{%s}, start failed:%v\n", tmp.Name(), err)
			}
		}()
		fmt.Printf("trace event:%s start\n", tmp.Name())
	}
	fmt.Println("Bpf Spy Start All")
	log.Loger.Info("Bpf Spy Start All.")
}
