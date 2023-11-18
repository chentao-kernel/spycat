package processor

import (
	"sync"

	"github.com/chentao-kernel/spycat/pkg/core/model"
)

type valueRecorder struct {
	name string
	// aggValuesMap is responsible for its own thread-safe access.
	labelValues sync.Map
	aggKindMap  map[string][]KindConfig
}

func newValueRecorder(recorderName string, aggKindMap map[string][]KindConfig) *valueRecorder {
	return &valueRecorder{
		name:        recorderName,
		labelValues: sync.Map{},
		aggKindMap:  aggKindMap,
	}
}

func (r *valueRecorder) Record(key *LabelKeys, metricValues []*model.Metric, timestamp uint64) {
	if key == nil {
		return
	}
	aggValues, ok := r.labelValues.Load(*key)
	if !ok {
		aggValues, _ = r.labelValues.LoadOrStore(*key, newAggValuesMap(metricValues, r.aggKindMap))
	}
	for _, metric := range metricValues {
		aggValues.(aggValuesMap).calculate(metric, timestamp)
	}
}

func (r *valueRecorder) dump() []*model.DataBlock {
	ret := make([]*model.DataBlock, 0)
	r.labelValues.Range(func(key, value interface{}) bool {
		k := key.(LabelKeys)
		v := value.(aggValuesMap)
		dataGroup := model.NewDataBlock(r.name, k.GetLabels(), v.getTimestamp(), v.getAll()...)
		ret = append(ret, dataGroup)
		return true
	})
	return ret
}

func (r *valueRecorder) reset() {
	r.labelValues = sync.Map{}
}
