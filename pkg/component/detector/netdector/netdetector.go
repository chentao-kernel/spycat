package netdetector

import (
	"github.com/chentao-kernel/spycat/pkg/component/consumer"
	"github.com/chentao-kernel/spycat/pkg/component/detector"
	"github.com/chentao-kernel/spycat/pkg/core/model"
)

const (
	DetectorNetType string = "detector_net"
)

type NetDetector struct {
	cfg       *Config
	eventChan chan *model.SpyEvent
	stopChan  chan struct{}
	cousumers []consumer.Consumer
}

func NewNetDetector(cfg any, consumers []consumer.Consumer) detector.Detector {
	config, _ := cfg.(*Config)

	c := &NetDetector{
		cfg: config,
	}

	return c
}

func (c *NetDetector) Start() error {
	return nil
}

func (c *NetDetector) Stop() error {
	return nil
}

func (c *NetDetector) Name() string {
	return DetectorNetType
}

// 公共接口
func (c *NetDetector) ConsumeEvent(e *model.SpyEvent) error {
	return nil
}

// hard code
func (c *NetDetector) OwnedEvents() []string {
	return []string{}
}
