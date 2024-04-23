package appspy

import (
	"fmt"
	"os"
	"time"

	"github.com/chentao-kernel/spycat/pkg/component/consumer"
	"github.com/chentao-kernel/spycat/pkg/component/detector"
	cpu "github.com/chentao-kernel/spycat/pkg/component/detector/cpudetector"
	io "github.com/chentao-kernel/spycat/pkg/component/detector/iodetector"
	mem "github.com/chentao-kernel/spycat/pkg/component/detector/memdetector"
	net "github.com/chentao-kernel/spycat/pkg/component/detector/netdetector"
	"github.com/chentao-kernel/spycat/pkg/component/exporter"
	lokiexporter "github.com/chentao-kernel/spycat/pkg/component/exporter/loki"
	spyexporter "github.com/chentao-kernel/spycat/pkg/component/exporter/spy"
	"github.com/chentao-kernel/spycat/pkg/component/processor"
	"github.com/chentao-kernel/spycat/pkg/component/receiver"
	"github.com/chentao-kernel/spycat/pkg/core/model"
)

type AppSpy struct {
	componentsFactory *ComponentsFactory
	receiver          receiver.Receiver
	detecorManager    *detector.DetectorManager
}

func NewAppSpy(cfg *Config) (*AppSpy, error) {
	app := &AppSpy{
		componentsFactory: NewConpnentsFactory(),
	}

	app.registerFactory()

	if err := app.createPipeline(cfg); err != nil {
		return nil, fmt.Errorf("Create piplined failed:%v", err)
	}
	return app, nil
}

func (a *AppSpy) registerFactory() {
	a.componentsFactory.RegisterReceiver(receiver.ReceiverCiliumType, receiver.NewCiliumReceiver, receiver.NewConfig())
	a.componentsFactory.RegisterDetector(cpu.DetectorCpuType, cpu.NewCpuDetector, cpu.NewConfig())
	a.componentsFactory.RegisterDetector(io.DetectorIoType, io.NewIoDetector, io.NewConfig())
	a.componentsFactory.RegisterDetector(mem.DetectorMemType, mem.NewMemDetector, mem.NewConfig())
	a.componentsFactory.RegisterDetector(net.DetectorNetType, net.NewNetDetector, net.NewConfig())
	a.componentsFactory.RegisterProcessor(processor.ProcessorDefaultType, processor.NewDefaultProcessor, processor.NewConfig())
}

func (a *AppSpy) GetReceiver() receiver.Receiver {
	return a.receiver
}

func (a *AppSpy) Init(cfg any) error {
	return a.detecorManager.InitAllDetectors(cfg)
}

func (a *AppSpy) Start() error {
	err := a.detecorManager.StartAllDetectors()
	if err != nil {
		return fmt.Errorf("detector start failed:%v", err)
	}
	err = a.receiver.Start()
	if err != nil {
		return fmt.Errorf("receiver start failed:%v", err)
	}
	return nil
}

func (a *AppSpy) Stop() error {
	err := a.receiver.Stop()
	if err != nil {
		return fmt.Errorf("receiver stop failed:%v", err)
	}
	err = a.detecorManager.StopAllDetectors()
	if err != nil {
		return fmt.Errorf("detector stop failed:%v", err)
	}
	return nil
}

func NewExporter(cfg *Config) exporter.Exporter {
	var eper exporter.Exporter
	switch cfg.Exporter {
	case model.DISK:
		var conf *spyexporter.Config

		if cfg.Server == "" {
			conf = &spyexporter.Config{
				OutPuter: "TerminalOutputer",
			}
		} else {
			conf = &spyexporter.Config{
				OutPuter:     "FileOutputer",
				BaseFilePath: cfg.Server,
			}
		}
		eper = spyexporter.NewSpyExporter(conf)
	case model.LOKI:
		host, _ := os.Hostname()
		labels := "{host=\"" + host + "\"}"
		conf := &lokiexporter.Config{
			Url:             cfg.Server + "/api/prom/push",
			Labels:          labels,
			BatchWait:       5 * time.Second,
			BatchEntriesNum: 10000,
		}
		eper = lokiexporter.NewLokiExporter(conf)
	// pyroscope exporter init in cpudetector
	// only support for oncpu, prometheus influxdb no support
	case model.PYROSCOPE, model.PROMETHEUS, model.INFLUXDB:
		fallthrough
	default:
		conf := &spyexporter.Config{
			OutPuter: "TerminalOutputer",
		}
		eper = spyexporter.NewSpyExporter(conf)
	}
	return eper
}

func (a *AppSpy) createPipeline(cfg *Config) error {
	// 1. create Exporter
	eper := NewExporter(cfg)
	// 2. create Processor, used for data aggregate
	defaultProcessor := processor.NewDefaultProcessor(processor.NewConfig(), eper)
	// 3. create Detector
	cpuDetectorFactory := a.componentsFactory.Detectors[cpu.DetectorCpuType]
	cpuDetector := cpuDetectorFactory.NewComponentMember(cpuDetectorFactory.Config, []consumer.Consumer{defaultProcessor})
	ioDetectorFactory := a.componentsFactory.Detectors[io.DetectorIoType]
	ioDetector := ioDetectorFactory.NewComponentMember(ioDetectorFactory.Config, []consumer.Consumer{defaultProcessor})
	netDetectorFactory := a.componentsFactory.Detectors[net.DetectorNetType]
	netDetector := netDetectorFactory.NewComponentMember(netDetectorFactory.Config, []consumer.Consumer{defaultProcessor})
	memDetectorFactory := a.componentsFactory.Detectors[mem.DetectorMemType]
	memDetector := memDetectorFactory.NewComponentMember(memDetectorFactory.Config, []consumer.Consumer{defaultProcessor})
	detectorManager, err := detector.NewDetectorManager(cpuDetector, ioDetector, netDetector, memDetector)
	if err != nil {
		return fmt.Errorf("new detector manager failed:%v", err)
	}
	a.detecorManager = detectorManager

	// 4. create Receiver
	receiverFactory := a.componentsFactory.Receivers[receiver.ReceiverCiliumType]
	reciever := receiverFactory.NewComponentMember(receiver.NewConfig(), detectorManager)
	a.receiver = reciever

	return nil
}
