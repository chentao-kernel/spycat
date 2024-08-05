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
	StartTime  = "start_ts"
	EndTime    = "end_ts"
	IsSent     = "isSent"
	ThreadName = "threadName"
	TimeStamp  = "ts"
	Tid_W      = "waker_tid"
	Pid_W      = "waker_pid"
	Waker      = "waker"
	Wakee      = "wakee"
	Cpu        = "cpu"
	Prio       = "prio"
	CacheId    = "cache_id"

	WTarget = "wtarget"
	Pid_WT  = "pid_wt"
	Tid_WT  = "tid_wt"

	Target      = "target"
	Pid_T       = "wakee_pid"
	Tid_T       = "wakee_tid"
	IrqOffUs_W  = "wakee_irqoff_us"
	CpuOffUs_W  = "waker_offcpu_us"
	RunqLatUs_W = "waker_runqlat_us"
	Stack_W     = "waker_stack"
	IrqOffUs_T  = "wakee_irqoff_us"
	CpuOffUs_T  = "wakee_offcpu_us"
	RunqLatUs_T = "wakee_runqlat_us"
	Stack_T     = "wakee_stack"
	RunqDurMs   = "rq_dur_ms"

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
	DurUs = "dur_us"
)

const (
	CpuEventBlockName = "cpu_event_block"
)
