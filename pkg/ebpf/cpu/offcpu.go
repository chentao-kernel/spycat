package cpu

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unsafe"

	bpf "github.com/aquasecurity/libbpfgo"
	"github.com/chentao-kernel/spycat/pkg/app/config"
	"github.com/chentao-kernel/spycat/pkg/component/detector/cpudetector"
	"github.com/chentao-kernel/spycat/pkg/core"
	"github.com/chentao-kernel/spycat/pkg/core/model"
	"github.com/chentao-kernel/spycat/pkg/log"
	"github.com/chentao-kernel/spycat/pkg/symtab"
	"github.com/chentao-kernel/spycat/pkg/util"
	"golang.org/x/sys/unix"
)

const (
	SCHED_CACHE_SIZE      = 512
	SCHED_CACHE_DUMP_STEP = 10
)

var (
	bind, configFile string
)

type Waker struct {
	Pid           uint32
	Tgid          uint32
	T_pid         uint32
	Pad           uint32
	T_Comm        [16]byte
	User_stack_id int32
	Kern_stack_id int32
	Comm          [16]byte
	Oncpu_ns      uint64
	Offcpu_ns     uint64
	Onrq_ns       uint64
	Offcpu_id     uint32
	Oncpu_id      uint32
	Run_delay_ns  uint64
}

type Target struct {
	Pid           uint32
	Tgid          uint32
	W_pid         uint32
	Pad           uint32
	W_Comm        [16]byte
	User_stack_id int32
	Kern_stack_id int32
	Comm          [16]byte
	Oncpu_ns      uint64
	Offcpu_ns     uint64
	Onrq_ns       uint64
	Offcpu_id     uint32
	Oncpu_id      uint32
	Run_delay_ns  uint64
}

type SchedRecord struct {
	Pid  uint32
	Prio uint32
	Comm [16]byte
	Ts   uint64
}
type SchedCached struct {
	Status  uint32
	Cpu     uint32
	Id      uint32
	Pad     uint32
	Records [SCHED_CACHE_SIZE]SchedRecord
}

type OffCpuEvent struct {
	Waker            Waker
	Target           Target
	Dur_ms           uint64
	Rq_dur_ms        uint64
	Ts_ns            uint64
	IsSchedCacheDump uint32
	Cpu              uint32
}

type OffCpuArgs struct {
	Pid           uint32
	Tgid          uint32
	Min_offcpu_ms uint32
	Max_offcpu_ms uint32
	Rq_dur_ms     uint32
}

type OffcpuSession struct {
	Session *core.Session
	// inner filed
	PerfBuffer *bpf.PerfBuffer
	// inner filed
	Module            *bpf.Module
	SymSession        *symtab.SymSession
	mapStacks         *bpf.BPFMap
	mapSchedCacheDump *bpf.BPFMap
	Args              OffCpuArgs
	BtfPath           string
}

func NewOffCpuBpfSession(name string, cfg *config.OFFCPU, buf chan *model.SpyEvent) core.BpfSpyer {
	symSession, err := symtab.NewSymSession(cfg.SymbolCacheSize)
	if err != nil {
		log.Loger.Error("sym session failed")
		return nil
	}
	return &OffcpuSession{
		Session:    core.NewSession(name, &core.SessionConfig{}, buf),
		SymSession: symSession,
		Args: OffCpuArgs{
			Pid:           math.MaxUint32,
			Tgid:          uint32(cfg.Pid),
			Min_offcpu_ms: uint32(cfg.MinOffcpuMs),
			Max_offcpu_ms: uint32(cfg.MaxOffcpuMs),
			Rq_dur_ms:     uint32(cfg.RqDurMs),
		},
		BtfPath: cfg.BtfPath,
	}
}

func (b *OffcpuSession) attachProgs() error {
	progIter := b.Module.Iterator()
	for {
		prog := progIter.NextProgram()
		if prog == nil {
			break
		}
		if prog.Name() == "sched_switch_hook" {
			var name string
			ret := util.KprobeExists("finish_task_switch")
			if ret {
				name = "finish_task_switch"
			} else {
				name = "finish_task_switch.isra.0"
			}
			if _, err := prog.AttachKprobe(name); err != nil {
				return err
			}
		} else {
			if _, err := prog.AttachGeneric(); err != nil {
				return err
			}
		}

		fmt.Printf("prog:%s attach success\n", prog.GetName())
	}
	return nil
}

func (b *OffcpuSession) initArgsMap() error {
	var id uint32
	args := &OffCpuArgs{
		Pid:           b.Args.Pid,
		Tgid:          b.Args.Tgid,
		Min_offcpu_ms: b.Args.Min_offcpu_ms,
		Max_offcpu_ms: b.Args.Max_offcpu_ms,
		Rq_dur_ms:     b.Args.Rq_dur_ms,
	}
	maps, err := b.Module.GetMap("args_map")
	if err != nil {
		return fmt.Errorf("get map failed:%v", err)
	}
	maps.Update(unsafe.Pointer(&id), unsafe.Pointer(args))
	fmt.Printf("update user_args succeess:\n")
	util.PrintStructFields(b.Args)
	return nil
}

func (b *OffcpuSession) pollData() {
	dataChan := make(chan []byte)
	lostChan := make(chan uint64)
	var err error

	b.PerfBuffer, err = b.Module.InitPerfBuf("perf_map", dataChan, lostChan, 1)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	b.PerfBuffer.Start()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer func() {
		b.PerfBuffer.Stop()
		b.PerfBuffer.Close()
		stop()
	}()

loop:
	for {
		select {
		case data := <-dataChan:
			b.HandleEvent(data)
		case e := <-lostChan:
			fmt.Printf("Events lost:%d\n", e)
		case <-ctx.Done():
			break loop
		}
	}
}

func (b *OffcpuSession) Start() error {
	fmt.Println("offcpu start trace")
	err := unix.Setrlimit(unix.RLIMIT_MEMLOCK, &unix.Rlimit{
		Cur: unix.RLIM_INFINITY,
		Max: unix.RLIM_INFINITY,
	})
	if err != nil {
		return err
	}

	args := bpf.NewModuleArgs{
		BPFObjBuff: offcpuBpf,
		BTFObjPath: b.BtfPath,
	}

	b.Module, err = bpf.NewModuleFromBufferArgs(args)
	if err != nil {
		return fmt.Errorf("load module failed:%v", err)
	}

	err = b.Module.BPFLoadObject()
	if err != nil {
		return fmt.Errorf("load object failed:%v", err)
	}

	err = b.initArgsMap()
	if err != nil {
		return fmt.Errorf("init args map failed:%v", err)
	}

	b.mapStacks, err = b.Module.GetMap("stack_map")
	if err != nil {
		return fmt.Errorf("get stack map failed:%v", err)
	}

	b.mapSchedCacheDump, err = b.Module.GetMap("sched_cache_backup_map")
	if err != nil {
		return fmt.Errorf("get sched_cache_backup map failed:%v", err)
	}

	err = b.attachProgs()
	if err != nil {
		return fmt.Errorf("attach program failed:%v", err)
	}

	b.pollData()
	return nil
}

func (b *OffcpuSession) Stop() error {
	if b.Module != nil {
		b.Module.Close()
	}
	if b.PerfBuffer != nil {
		b.PerfBuffer.Close()
	}
	return nil
}

func (b *OffcpuSession) getStack(stackId int32) []byte {
	if stackId < 0 {
		return nil
	}

	stackIdU32 := uint32(stackId)
	key := unsafe.Pointer(&stackIdU32)
	stack, err := b.mapStacks.GetValue(key)
	if err != nil {
		return nil
	}

	return stack
}

func (b *OffcpuSession) ResolveStack(syms *bytes.Buffer, stackId int32, pid uint32, userspace bool) error {
	if b.SymSession == nil {
		return fmt.Errorf("sym session nil")
	}
	stack := b.getStack(stackId)
	if stack == nil {
		return fmt.Errorf("stack nil")
	}
	b.SymSession.WalkStack(syms, stack, pid, userspace)
	return nil
}

func (b *OffcpuSession) schedCacheDump(cpu uint32) {
	var cache_event SchedCached

	key := unsafe.Pointer(&cpu)
	value, err := b.mapSchedCacheDump.GetValue(key)
	if err != nil {
		log.Loger.Error("sched cache map get failed: %s", err)
		return
	}

	if err = binary.Read(bytes.NewBuffer(value), binary.LittleEndian, &cache_event); err != nil {
		log.Loger.Error("parse event: %s", err)
	}

	for i := 0; i < SCHED_CACHE_SIZE; i += SCHED_CACHE_DUMP_STEP {
		spyEvent := &model.SpyEvent{}

		spyEvent.Name = model.OffCpu
		spyEvent.TimeStamp = uint64(time.Now().Unix())
		spyEvent.Class.Name = cpudetector.DetectorCpuType
		spyEvent.Class.Event = model.OffCpu
		spyEvent.SetUserAttributeWithUint32("cpu", cpu)
		spyEvent.Task.Tid = cache_event.Records[i].Pid
		spyEvent.Task.Comm = string(cache_event.Records[i].Comm[:])
		spyEvent.SetUserAttributeWithUint32("cache_id", uint32(i))
		spyEvent.SetUserAttributeWithUint64("ts", cache_event.Records[i].Ts)
		spyEvent.SetUserAttributeWithUint32("prio", cache_event.Records[i].Prio)
		b.Session.DataBuffer <- spyEvent
	}
}

func (b *OffcpuSession) HandleEvent(data []byte) {
	spyEvent := &model.SpyEvent{}
	var event OffCpuEvent

	if err := binary.Read(bytes.NewBuffer(data), binary.LittleEndian, &event); err != nil {
		log.Loger.Error("parse event: %s", err)
	}

	if event.IsSchedCacheDump == 1 {
		b.schedCacheDump(event.Cpu)
	}
	// util.PrintStructFields(event)
	spyEvent.Name = model.OffCpu
	spyEvent.TimeStamp = uint64(time.Now().Unix())
	spyEvent.Class.Name = cpudetector.DetectorCpuType
	spyEvent.Class.Event = model.OffCpu
	spyEvent.Task.Pid = event.Target.Tgid
	spyEvent.Task.Tid = event.Target.Pid
	spyEvent.Task.Comm = string(event.Target.Comm[:])

	spyEvent.SetUserAttributeWithUint32("w_tid", event.Waker.Pid)
	spyEvent.SetUserAttributeWithUint32("w_pid", event.Waker.Tgid)
	spyEvent.SetUserAttributeWithUint32("w_onrq_oncpu", uint32(event.Waker.Oncpu_ns-event.Waker.Onrq_ns))
	spyEvent.SetUserAttributeWithUint32("w_offcpu_oncpu", uint32(event.Waker.Oncpu_ns-event.Waker.Offcpu_ns))
	spyEvent.SetUserAttributeWithByteBuf("w_comm", event.Waker.Comm[:])
	syms := bytes.NewBuffer(nil)
	b.ResolveStack(syms, event.Waker.Kern_stack_id, 0, false)
	// split kernel and user stack info
	syms.Write([]byte("----"))
	b.ResolveStack(syms, event.Waker.User_stack_id, event.Waker.Tgid, true)
	spyEvent.SetUserAttributeWithByteBuf("w_stack", syms.Bytes())

	spyEvent.SetUserAttributeWithUint32("t_tid", event.Target.Pid)
	spyEvent.SetUserAttributeWithUint32("t_pid", event.Target.Tgid)
	spyEvent.SetUserAttributeWithUint32("t_onrq_oncpu", uint32(event.Target.Oncpu_ns-event.Target.Onrq_ns))
	spyEvent.SetUserAttributeWithUint32("t_offcpu_oncpu", uint32(event.Target.Oncpu_ns-event.Target.Offcpu_ns))
	spyEvent.SetUserAttributeWithByteBuf("t_comm", event.Target.Comm[:])
	spyEvent.SetUserAttributeWithUint64("t_start_time", event.Target.Offcpu_ns)
	// perf event output time
	spyEvent.SetUserAttributeWithUint64("ts", event.Ts_ns)
	// target offcpu dur
	spyEvent.SetUserAttributeWithUint64("dur_ms", event.Dur_ms)
	// target on rq dur
	spyEvent.SetUserAttributeWithUint64("rq_dur_ms", event.Rq_dur_ms)

	syms.Reset()
	b.ResolveStack(syms, event.Target.Kern_stack_id, 0, false)
	syms.Write([]byte("----"))
	b.ResolveStack(syms, event.Target.User_stack_id, event.Target.Tgid, true)
	spyEvent.SetUserAttributeWithByteBuf("t_stack", syms.Bytes())

	b.Session.DataBuffer <- spyEvent
}

func (b *OffcpuSession) ReadEvent() error {
	return nil
}

func (b *OffcpuSession) ConsumeEvent() error {
	return nil
}

func (b *OffcpuSession) Name() string {
	return b.Session.Name()
}

//go:embed offcpu/offcpu.bpf.o
var offcpuBpf []byte
