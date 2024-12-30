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
		labels.AddStringValue(model.Stack, "a;b;c")
		metric := model.NewIntMetric(model.OnCpuMetricName, 111)
		dataBlock := model.NewDataBlock(model.OnCpu, labels, 2312341234, metric)
		sqlite.Consume(dataBlock)
	}

	// test for futexsnoop
	for i := 0; i < 1002; i++ {
		labels := model.NewAttributeMap()
		labels.AddIntValue(model.UserCnt, 20)
		labels.AddIntValue(model.MaxUserCnt, 100)
		labels.AddIntValue(model.LockAddr, 12312313)
		labels.AddIntValue(model.MinDur, 50)
		labels.AddIntValue(model.MaxDur, 200)
		labels.AddIntValue(model.DeltaDur, 120)
		labels.AddIntValue(model.AvgDur, 120)
		labels.AddIntValue(model.LockCnt, 10)
		labels.AddStringValue(model.Stack, "a;b;c")
		labels.AddStringValue(model.Comm, "comm")
		labels.AddIntValue(model.Pid, 111)
		labels.AddIntValue(model.Tid, 112)

		metric := model.NewIntMetric(model.FutexMaxUerCountName, 22)
		dataBlock := model.NewDataBlock(model.FutexSnoop, labels, 1231231223, metric)
		sqlite.Consume(dataBlock)
	}

	// test for syscall
	for i := 0; i < 1002; i++ {
		labels := model.NewAttributeMap()
		labels.AddIntValue(model.DurUs, 1000)
		labels.AddStringValue(model.Syscall, "write")
		labels.AddStringValue(model.Stack, "a;b;c")
		labels.AddIntValue(model.Pid, 111)
		labels.AddIntValue(model.Tid, 112)
		labels.AddStringValue(model.Comm, "comm")

		metric := model.NewIntMetric(model.SyscallMetricName, 1212)
		dataBlock := model.NewDataBlock(model.Syscall, labels, 12313123412, metric)
		sqlite.Consume(dataBlock)
	}

	// test for cachestat
	for i := 0; i < 1002; i++ {
		labels := model.NewAttributeMap()
		labels.AddIntValue(model.Pid, 1234)
		labels.AddIntValue(model.Cpu, 0)
		labels.AddIntValue(model.Count, 100000)
		labels.AddIntValue(model.ReadSizeM, 1000)
		labels.AddIntValue(model.WriteSizeM, 2000)
		labels.AddStringValue(model.Comm, "cachestat")
		labels.AddStringValue(model.File, "cache.log")

		metric := model.NewIntMetric(model.CacheStatMetricName, 10000)
		dataBlock := model.NewDataBlock(model.CacheStat, labels, 1232141234, metric)
		sqlite.Consume(dataBlock)
	}

	time.Sleep(time.Second * 2)
}
