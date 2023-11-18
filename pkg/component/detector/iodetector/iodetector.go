package iodetector

import (
	"fmt"

	"github.com/chentao-kernel/spycat/pkg/component/consumer"
	"github.com/chentao-kernel/spycat/pkg/component/detector"
	"github.com/chentao-kernel/spycat/pkg/core/model"
	"github.com/chentao-kernel/spycat/pkg/log"
)

const (
	DetectorIoType string = "detector_io"
)

type IoDetector struct {
	cfg       *Config
	eventChan chan *model.SpyEvent
	stopChan  chan struct{}
	consumers []consumer.Consumer
}

func NewIoDetector(cfg any, consumers []consumer.Consumer) detector.Detector {
	config, _ := cfg.(*Config)

	io := &IoDetector{
		cfg:       config,
		eventChan: make(chan *model.SpyEvent, config.EventChanSize),
		consumers: consumers,
		stopChan:  make(chan struct{}),
	}

	return io
}

func (io *IoDetector) Start() error {
	go io.ConsumeChanEvents()
	return nil
}

func (io *IoDetector) Stop() error {
	return nil
}

func (io *IoDetector) Name() string {
	return DetectorIoType
}

func (io *IoDetector) ProcessEvent(e *model.SpyEvent) error {
	fmt.Printf("io process event\n")
	return nil
}

func (io *IoDetector) ConsumeChanEvents() {
	for {
		select {
		case e := <-io.eventChan:
			err := io.ProcessEvent(e)
			if err != nil {
				log.Loger.Error("cpu detector process event failed:%v", err)
			}
		case <-io.stopChan:
			log.Loger.Info("detector event channel stoped.")
			return
		}
	}
}

// 公共接口
func (io *IoDetector) ConsumeEvent(e *model.SpyEvent) error {
	return nil
}

// hard code
func (io *IoDetector) OwnedEvents() []string {
	return []string{}
}
