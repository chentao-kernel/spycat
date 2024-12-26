package sqlitexporter

import (
	"testing"
	"time"

	"github.com/chentao-kernel/spycat/pkg/core/model"
	"github.com/chentao-kernel/spycat/pkg/log"
)

func init() {
	log.LogInit()
}

func TestSqlite(t *testing.T) {
	log.LogInit()

	conf := NewConfig()
	sqlite := NewSqliteExporter(conf)
	if sqlite == nil {
		t.Errorf("new sqlite exporter failed")
	}

	// test for offcpu
	// 998|2024-12-26 21:54:01.591039107+08:00|111|112|waker|a;b;c|112|111|wakee|d;e;f||20|10|996|0|120
	// 999|2024-12-26 21:54:01.591039107+08:00|111|112|waker|a;b;c|112|111|wakee|d;e;f||20|10|997|0|120
	// 1000|2024-12-26 21:54:01.591039107+08:00|111|112|waker|a;b;c|112|111|wakee|d;e;f||20|10|998|0|120
	for i := 0; i < 1002; i++ {
		labels := model.NewAttributeMap()
		labels.AddIntValue(model.Cpu, int64(i))
		labels.AddIntValue(model.Prio, 120)
		labels.AddIntValue(model.CacheId, 0)
		labels.AddIntValue(model.DurMs, 20)
		labels.AddIntValue(model.RunqDurMs, 10)
		labels.AddStringValue(model.Waker, "waker")
		labels.AddIntValue(model.Tid_W, 111)
		labels.AddIntValue(model.Pid_W, 112)
		labels.AddStringValue(model.Stack_W, "a;b;c")
		labels.AddStringValue(model.Wakee, "wakee")
		labels.AddIntValue(model.Tid_T, 113)
		labels.AddIntValue(model.Pid_T, 114)
		labels.AddStringValue(model.Stack_T, "d;e;f")
		labels.AddIntValue(model.RunqLatUs_T, 2222)
		labels.AddIntValue(model.CpuOffUs_T, 3333)
		labels.AddStringValue(model.StartTime, "2024-12-12")

		metric := model.NewIntMetric(model.OffCpuMetricName, 12)

		dataBlock := model.NewDataBlock(model.OffCpu, labels, 123123123123, metric)

		sqlite.Consume(dataBlock)
	}

	// test for oncpu
	for i := 0; i < 1002; i++ {
		labels := model.NewAttributeMap()
		labels.AddIntValue(model.Cpu, 0)
		labels.AddIntValue(model.Pid, 111)
		labels.AddIntValue(model.Tid, 112)
		labels.AddStringValue(model.Comm, "target")
		metric := model.NewIntMetric(model.OnCpuMetricName, 111)
		dataBlock := model.NewDataBlock(model.OnCpu, labels, 2312341234, metric)
		sqlite.Consume(dataBlock)
	}
	time.Sleep(time.Second * 2)
}
