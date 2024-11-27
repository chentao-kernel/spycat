package memdetector

import (
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

func (mem *MemDetector) Init(cfg any) error {
	return nil
}

func (mem *MemDetector) Start() error {
	go mem.ConsumeChanEvents()
	return nil
}

func (mem *MemDetector) Stop() error {
	close(mem.stopChan)
	return nil
}

func (mem *MemDetector) Name() string {
	return DetectorMemType
}

func (mem *MemDetector) formatCacheStat(e *model.SpyEvent) (*model.AttributeMap, error) {
	labels := model.NewAttributeMap()
	for i := 0; i < int(e.ParamsCnt); i++ {
		userAttributes := e.UserAttributes[i]
		switch {
		case userAttributes.GetKey() == "read_size_m":
			labels.AddIntValue(model.ReadSizeM, int64(userAttributes.GetUintValue()))
		case userAttributes.GetKey() == "write_size_m":
			labels.AddIntValue(model.WriteSizeM, int64(userAttributes.GetUintValue()))
		case userAttributes.GetKey() == "pid":
			labels.AddIntValue(model.Pid, int64(userAttributes.GetUintValue()))
		case userAttributes.GetKey() == "comm":
			labels.AddStringValue(model.Comm, string(userAttributes.GetValue()))
		case userAttributes.GetKey() == "file":
			labels.AddStringValue(model.File, string(userAttributes.GetValue()))
		}
	}
	return labels, nil
}

func (mem *MemDetector) cachestatHandler(e *model.SpyEvent) (*model.DataBlock, error) {
	labels, _ := mem.formatCacheStat(e)
	val := e.GetUintUserAttribute("write_size")
	metric := model.NewIntMetric(model.CacheStatMetricName, int64(val))
	return model.NewDataBlock(model.CacheStat, labels, e.TimeStamp, metric), nil
}
func (mem *MemDetector) ProcessEvent(e *model.SpyEvent) error {
	var dataBlock *model.DataBlock
	var err error
	switch e.Name {
	case model.CacheStat:
		dataBlock, err = mem.cachestatHandler(e)
	default:
		return nil
	}
	if err != nil {
		return nil
	}
	if dataBlock == nil {
		return nil
	}
	// next consumer is default processor
	for _, con := range mem.consumers {
		err := con.Consume(dataBlock)
		if err != nil {
			log.Loger.Error("consumer consume event failed:%v", err)
			return err
		}
	}
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
func (mem *MemDetector) ConsumeEvent(e *model.SpyEvent) error {
	mem.eventChan <- e
	return nil
}

// hard code
func (mem *MemDetector) OwnedEvents() []string {
	return []string{model.CacheStat}
}
