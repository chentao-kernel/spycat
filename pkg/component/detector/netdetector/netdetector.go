package netdetector

import (
	"github.com/chentao-kernel/spycat/pkg/component/consumer"
	"github.com/chentao-kernel/spycat/pkg/component/detector"
	"github.com/chentao-kernel/spycat/pkg/core/model"
	"github.com/chentao-kernel/spycat/pkg/log"
)

const (
	DetectorNetType string = "detector_net"
)

type NetDetector struct {
	cfg       *Config
	eventChan chan *model.SpyEvent
	stopChan  chan struct{}
	consumers []consumer.Consumer
}

func NewNetDetector(cfg any, consumers []consumer.Consumer) detector.Detector {
	config, _ := cfg.(*Config)

	n := &NetDetector{
		cfg:       config,
		eventChan: make(chan *model.SpyEvent, config.EventChanSize),
		consumers: consumers,
		stopChan:  make(chan struct{}),
	}

	return n
}

func (n *NetDetector) sendToConsumers() {

}

func (n *NetDetector) Start() error {
	go n.ConsumeChanEvents()
	return nil
}

func (n *NetDetector) Stop() error {
	return nil
}

func (n *NetDetector) Name() string {
	return DetectorNetType
}

func (n *NetDetector) ProcessEvent(e *model.SpyEvent) error {
	return nil
}

func (n *NetDetector) ConsumeChanEvents() {
	for {
		select {
		case e := <-n.eventChan:
			err := n.ProcessEvent(e)
			if err != nil {
				log.Loger.Error("cpu detector process event failed:%v", err)
			}
		case <-n.stopChan:
			log.Loger.Info("detector event channel stoped.")
			return
		}
	}
}

// 公共接口
func (n *NetDetector) ConsumeEvent(e *model.SpyEvent) error {
	return nil
}

// hard code
func (n *NetDetector) OwnedEvents() []string {
	return []string{}
}
