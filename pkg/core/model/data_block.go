package model

import (
	"encoding/json"
	"fmt"
	"strings"
)

type DataBlock struct {
	Name      string        `json:"name"`
	Metrics   []*Metric     `json:"metrics"`
	Labels    *AttributeMap `json:"labels"`
	Timestamp uint64        `json:"timestamp"`
}

func NewDataBlock(name string, labels *AttributeMap, timestamp uint64, values ...*Metric) *DataBlock {
	return &DataBlock{
		Name:      name,
		Metrics:   values,
		Labels:    labels,
		Timestamp: timestamp,
	}
}

func (d *DataBlock) GetMetric(name string) (*Metric, bool) {
	for _, metric := range d.Metrics {
		if metric.Name == name {
			return metric, true
		}
	}
	return nil, false
}

func (d *DataBlock) AddIntMetricWithName(name string, value int64) {
	d.AddMetric(NewIntMetric(name, value))
}

func (d *DataBlock) AddMetric(metric *Metric) {
	if d.Metrics == nil {
		d.Metrics = make([]*Metric, 0)
	}
	d.Metrics = append(d.Metrics, metric)
}

// UpdateAddIntMetric overwrite the metric with the key of 'name' if existing, or adds the metric if not existing.
func (d *DataBlock) UpdateAddIntMetric(name string, value int64) {
	if metric, ok := d.GetMetric(name); ok {
		metric.Data = &Int{Value: value}
	} else {
		d.AddIntMetricWithName(name, value)
	}
}

func (d *DataBlock) RemoveMetric(name string) {
	newValues := make([]*Metric, 0)
	for _, value := range d.Metrics {
		if value.Name == name {
			continue
		}
		newValues = append(newValues, value)
	}
	d.Metrics = newValues
}

func (d DataBlock) String() string {
	var str strings.Builder
	str.WriteString(fmt.Sprintln("DataBlock:"))
	str.WriteString(fmt.Sprintf("\tName: %s\n", d.Name))
	str.WriteString(fmt.Sprintln("\tValues:"))
	for _, v := range d.Metrics {
		switch v.DataType() {
		case IntMetricType:
			str.WriteString(fmt.Sprintf("\t\t\"%s\": %d\n", v.Name, v.GetInt().Value))
		case HistogramMetricType:
			histogram := v.GetHistogram()
			str.WriteString(fmt.Sprintf("\t\t\"%s\": \n\t\t\tSum: %d\n\t\t\tCount: %d\n\t\t\tExplicitBoundaries: %v\n\t\t\tBucketCount: %v\n",
				v.Name, histogram.Sum, histogram.Count, histogram.ExplicitBoundaries, histogram.BucketCounts))
		}
	}
	if labelsStr, err := json.MarshalIndent(d.Labels, "\t", "\t"); err == nil {
		str.WriteString(fmt.Sprintf("\tLabels:\n\t%v\n", string(labelsStr)))
	} else {
		str.WriteString(fmt.Sprintln("\tLabels: marshal Failed"))
	}
	str.WriteString(fmt.Sprintf("\tTimestamp: %d\n", d.Timestamp))
	return str.String()
}

func (d *DataBlock) Reset() {
	d.Name = ""
	for _, v := range d.Metrics {
		v.Clear()
	}
	d.Labels.ResetValues()
	d.Timestamp = 0
}

func (d *DataBlock) Clone() *DataBlock {
	metrics := make([]*Metric, len(d.Metrics))
	for i, v := range d.Metrics {
		metrics[i] = v.Clone()
	}
	return NewDataBlock(d.Name, d.Labels.Clone(), d.Timestamp, metrics...)
}
