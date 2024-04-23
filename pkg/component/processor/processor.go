package processor

import (
	"sync"
	"time"

	"github.com/chentao-kernel/spycat/pkg/component/consumer"
	"github.com/chentao-kernel/spycat/pkg/core/model"
	"github.com/chentao-kernel/spycat/pkg/log"
)

const (
	ProcessorDefaultType string = "processor_default"
)

var exponentialInt64Boundaries = []int64{10e6, 20e6, 50e6, 80e6, 130e6, 200e6, 300e6,
	400e6, 500e6, 700e6, 1e9, 2e9, 5e9, 30e9}

type Processor interface {
	consumer.Consumer
}

type Aggregator interface {
	Aggregate(data *model.DataBlock, selectors *LabelSelectors)
	Dump() []*model.DataBlock
}

type DefaultAggregator struct {
	aggregatorMap sync.Map
	mutex         sync.RWMutex
	cfg           *AggregatedConfig
}

type DefaultProcessor struct {
	config     *Config
	consumer   consumer.Consumer
	stopCh     chan struct{}
	aggregator Aggregator
	// todo labels
	ticker *time.Ticker
}

func NewDefaultAggregator(config *AggregatedConfig) *DefaultAggregator {
	return &DefaultAggregator{
		aggregatorMap: sync.Map{},
		cfg:           config,
	}
}

// Aggregate 往aggregatorMap中写聚合数据, 在Consume接口中调用Aggregate聚合
func (d *DefaultAggregator) Aggregate(data *model.DataBlock, selectors *LabelSelectors) {
	name := data.Name
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	result, ok := d.aggregatorMap.Load(name)
	if !ok {
		result, _ = d.aggregatorMap.LoadOrStore(name, newValueRecorder(name, d.cfg.KindMap))
	}
	key := selectors.GetLabelKeys(data.Labels)
	result.(*valueRecorder).Record(key, data.Metrics, data.Timestamp)
}

func (d *DefaultAggregator) Dump() []*model.DataBlock {
	ret := make([]*model.DataBlock, 0)
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.aggregatorMap.Range(func(_, value interface{}) bool {
		recorder := value.(*valueRecorder)
		ret = append(ret, recorder.dump()...)
		// new map
		recorder.reset()
		return true
	})
	return ret
}

func NewDefaultProcessor(cfg any, con consumer.Consumer) Processor {
	config, _ := cfg.(*Config)
	process := &DefaultProcessor{
		config:     config,
		consumer:   con,
		stopCh:     make(chan struct{}),
		aggregator: NewDefaultAggregator(toAggregatedConfig(config.AggregateKindMap)),
		// ticker:     time.NewTicker(time.Duration(config.TickerInterval) * time.Second),
		ticker: time.NewTicker(time.Duration(10) * time.Second),
	}

	go process.runTicker()
	return process
}

func toAggregatedConfig(m map[string][]AggregatedKindConfig) *AggregatedConfig {
	ret := &AggregatedConfig{KindMap: make(map[string][]KindConfig)}
	for k, v := range m {
		kindConfig := make([]KindConfig, len(v))
		for i, kind := range v {
			if kind.OutputName == "" {
				kind.OutputName = k
			}
			kindConfig[i] = newKindConfig(&kind)
		}
		ret.KindMap[k] = kindConfig
	}
	return ret
}

func newKindConfig(rawConfig *AggregatedKindConfig) (kindConfig KindConfig) {
	kind := GetAggregatorKind(rawConfig.Kind)
	switch kind {
	case HistogramKind:
		var boundaries []int64
		if rawConfig.ExplicitBoundaries != nil {
			boundaries = rawConfig.ExplicitBoundaries
		} else {
			boundaries = exponentialInt64Boundaries
		}
		return KindConfig{
			OutputName:         rawConfig.OutputName,
			Kind:               kind,
			ExplicitBoundaries: boundaries,
		}
	default:
		return KindConfig{
			OutputName: rawConfig.OutputName,
			Kind:       kind,
		}
	}
}

// Get data from aggreagte map
func (d *DefaultProcessor) runTicker() {
	for {
		select {
		case <-d.stopCh:
			return
		case <-d.ticker.C:
			aggResults := d.aggregator.Dump()
			for _, agg := range aggResults {
				// here consumer is spyexporter by default
				err := d.consumer.Consume(agg)
				if err != nil {
					log.Loger.Error("consume aggregatorMap failed")
				}
			}
		}
	}
}

// receive data by Consume from upstream and send data by runTicker with
// d.consumer.Consume(agg)
func (d *DefaultProcessor) Consume(data *model.DataBlock) error {
	switch data.Name {
	// offcpu事件不聚合数据直接输出给spyexporter
	case model.OffCpu:
		fallthrough
	case model.FutexSnoop:
		d.consumer.Consume(data)
	// oncpu直接upstream
	case model.OnCpu:
	}
	return nil
}
