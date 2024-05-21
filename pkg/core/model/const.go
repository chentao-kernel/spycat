package model

// expoter type
const (
	LOKI       = "loki"
	PROMETHEUS = "prometheus"
	DISK       = "disk"
	INFLUXDB   = "influxdb"
	PYROSCOPE  = "pyroscope"
)

const (
	OffCpu     = "offcpu"
	IrqOff     = "irqoff"
	OnCpu      = "oncpu"
	FutexSnoop = "futexsnoop"
	Syscall    = "syscall"
	OtherEvent = "other_event"
)

// for metric
const (
	OffCpuMetricName     = "offcpu_dur_ms"
	FutexMaxUerCountName = "max_futex_user_cnt"
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

	// for futexsnoop
	UserCnt    = "user_cnt"
	MaxUserCnt = "max_user_cnt"
	LockAddr   = "lock_addr"
	MinDur     = "min_dur"
	MaxDur     = "max_dur"
	DeltaDur   = "delta_dur"
	AvgDur     = "avg_dur"
	LockCnt    = "lock_cnt"
	Stack      = "stack"

	// for syscall
	DurMs = "dur_ms"
)

const (
	CpuEventBlockName = "cpu_event_block"
)
