package model

const (
	OffCpu     = "offcpu"
	IrqOff     = "irqoff"
	OnCpu      = "oncpu"
	OtherEvent = "other_event"
)

// for metric
const (
	OffCpuMetricName = "offcpu_dur_ms"
)

// for labels
const (
	Comm       = "comm"
	Pid        = "pid"
	Tid        = "tid"
	StartTime  = "startTime"
	EndTime    = "endTime"
	IsSent     = "isSent"
	ThreadName = "threadName"
	TimeStamp  = "timestamp"
	Pid_W      = "pid_w"
	Tgid_W     = "tgid_w"
	Waker      = "waker"

	WTarget = "wtarget"
	Pid_WT  = "pid_wt"

	Target      = "target"
	Pid_T       = "pid_t"
	Tgid_T      = "tgid_t"
	IrqOffUs_W  = "irqoffus_w"
	CpuOffUs_W  = "offcpuus_w"
	RunqLatUs_W = "runqlatus_w"
	Stack_W     = "stack_w"
	IrqOffUs_T  = "irqoffus_t"
	CpuOffUs_T  = "offcpuus_t"
	RunqLatUs_T = "runqlatus_t"
	Stack_T     = "stack_t"
)

const (
	CpuEventBlockName = "cpu_event_block"
)
