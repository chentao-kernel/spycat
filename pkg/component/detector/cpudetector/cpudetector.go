package cpudetector

import (
	"fmt"
	"sync"

	"github.com/chentao-kernel/spycat/pkg/component/consumer"
	"github.com/chentao-kernel/spycat/pkg/component/detector"
	"github.com/chentao-kernel/spycat/pkg/core/model"
	"github.com/chentao-kernel/spycat/pkg/log"
)

const (
	DetectorCpuType string = "detector_cpu"
)

type CpuDetector struct {
	cfg       *Config
	eventChan chan *model.SpyEvent
	stopChan  chan struct{}
	consumers []consumer.Consumer
	lock      sync.RWMutex
}

func NewCpuDetector(cfg any, consumers []consumer.Consumer) detector.Detector {
	config, _ := cfg.(*Config)

	cd := &CpuDetector{
		cfg:       config,
		eventChan: make(chan *model.SpyEvent, config.EventChanSize),
		consumers: consumers,
		stopChan:  make(chan struct{}),
	}

	return cd
}

func (c *CpuDetector) sendToConsumers() {

}

func (c *CpuDetector) Start() error {
	go c.ConsumeChanEvents()
	return nil
}

func (c *CpuDetector) Stop() error {
	close(c.stopChan)
	return nil
}

func (c *CpuDetector) Name() string {
	return DetectorCpuType
}

func (c *CpuDetector) ProcessEvent(e *model.SpyEvent) error {
	fmt.Printf("cpu process event\n")
	return nil
}

func (c *CpuDetector) ConsumeChanEvents() {
	for {
		select {
		case e := <-c.eventChan:
			err := c.ProcessEvent(e)
			if err != nil {
				log.Loger.Error("cpu detector process event failed:%v", err)
			}
		case <-c.stopChan:
			log.Loger.Info("detector event channel stoped.")
			return
		}
	}
}

func (c *CpuDetector) ConsumeCommEvent(e *model.SpyEvent) error {
	return nil
}

func (c *CpuDetector) ConsumeProfileCpuEvent(e *model.SpyEvent) error {
	return nil
}

func (c *CpuDetector) ConsumeEvent(e *model.SpyEvent) error {
	c.eventChan <- e
	return nil
}

// hard code
func (c *CpuDetector) OwnedEvents() []string {
	return []string{model.OffCpu, model.IrqOff, model.ProfileCpu}
}
