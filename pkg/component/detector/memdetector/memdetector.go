package memdetector

import (
	"fmt"

	"github.com/chentao-kernel/spycat/pkg/component/consumer"
	"github.com/chentao-kernel/spycat/pkg/component/detector"
	"github.com/chentao-kernel/spycat/pkg/core/model"
	"github.com/chentao-kernel/spycat/pkg/log"
)

const (
	DetectorMemType string = "detector_mem"
)

type MemDetector struct {
	cfg       *Config
	eventChan chan *model.SpyEvent
	stopChan  chan struct{}
	consumers []consumer.Consumer
}

func NewMemDetector(cfg any, consumers []consumer.Consumer) detector.Detector {
	config, _ := cfg.(*Config)

	mem := &MemDetector{
		cfg:       config,
		eventChan: make(chan *model.SpyEvent, config.EventChanSize),
		consumers: consumers,
		stopChan:  make(chan struct{}),
	}

	return mem
}

func (mem *MemDetector) sendToConsumers() {

}

func (mem *MemDetector) Start() error {
	go mem.ConsumeChanEvents()
	return nil
}

func (mem *MemDetector) Stop() error {
	return nil
}

func (mem *MemDetector) Name() string {
	return DetectorMemType
}

func (mem *MemDetector) ProcessEvent(e *model.SpyEvent) error {
	fmt.Printf("cpu process event\n")
	return nil
}

func (mem *MemDetector) ConsumeChanEvents() {
	for {
		select {
		case e := <-mem.eventChan:
			err := mem.ProcessEvent(e)
			if err != nil {
				log.Loger.Error("cpu detector process event failed:%v", err)
			}
		case <-mem.stopChan:
			log.Loger.Info("detector event channel stoped.")
			return
		}
	}
}

// 公共接口
func (mem *MemDetector) ConsumeEvent(*model.SpyEvent) error {
	return nil
}

// hard code
func (mem *MemDetector) OwnedEvents() []string {
	return []string{}
}
