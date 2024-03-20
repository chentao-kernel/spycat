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
}

type OffCpuEvent struct {
	Waker         Waker
	Target        Target
	Offtime_delta uint64
	Ts            uint64
}

type UserArgs struct {
	Pid           uint32
	Tgid          uint32
	Min_offcpu_ms uint32
	Max_offcpu_ms uint32
	Onrq_us       uint32
}

type OffcpuSession struct {
	Session *core.Session
	// inner filed
	PerfBuffer *bpf.PerfBuffer
	// inner filed
	Module     *bpf.Module
	SymSession *symtab.SymSession
	mapStacks  *bpf.BPFMap
	Args       UserArgs
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
		Args: UserArgs{
			Pid:           math.MaxUint32,
			Tgid:          uint32(cfg.Pid),
			Min_offcpu_ms: uint32(cfg.MinOffcpuMs),
			Max_offcpu_ms: uint32(cfg.MaxOffcpuMs),
			Onrq_us:       uint32(cfg.OnRqUs),
		},
	}
}

func (b *OffcpuSession) attachProgs() error {
	progIter := b.Module.Iterator()
	for {
		prog := progIter.NextProgram()
		if prog == nil {
			break
		}
		if _, err := prog.AttachGeneric(); err != nil {
			return err
		}
		fmt.Printf("prog:%s attach success\n", prog.GetName())
	}
	return nil
}

func (b *OffcpuSession) initArgsMap() error {
	var id uint32 = 0
	args := &UserArgs{
		Pid:           b.Args.Pid,
		Tgid:          b.Args.Tgid,
		Min_offcpu_ms: b.Args.Min_offcpu_ms,
		Max_offcpu_ms: b.Args.Max_offcpu_ms,
		Onrq_us:       b.Args.Onrq_us,
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
	//var module *bpf.Module
	//var prog *bpf.BPFProg
	fmt.Println("offcpu start trace")
	err := unix.Setrlimit(unix.RLIMIT_MEMLOCK, &unix.Rlimit{
		Cur: unix.RLIM_INFINITY,
		Max: unix.RLIM_INFINITY,
	})
	if err != nil {
		return err
	}

	args := bpf.NewModuleArgs{BPFObjBuff: offcpuBpf}

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

func (b *OffcpuSession) HandleEvent(data []byte) {
	var event OffCpuEvent
	spyEvent := &model.SpyEvent{}
	if err := binary.Read(bytes.NewBuffer(data), binary.LittleEndian, &event); err != nil {
		log.Loger.Error("parse event: %s", err)
	}
	//util.PrintStructFields(event)
	spyEvent.Name = "offcpu"
	spyEvent.TimeStamp = uint64(time.Now().Unix())
	spyEvent.Class.Name = cpudetector.DetectorCpuType
	spyEvent.Class.Event = model.OffCpu
	spyEvent.Task.Pid = event.Target.Tgid
	spyEvent.Task.Tid = event.Target.Pid
	//spyEvent.Task.Comm = strings.ReplaceAll(string(event.Target.Comm[:]), "\u0000", "")
	spyEvent.Task.Comm = string(event.Target.Comm[:])
	spyEvent.SetUserAttributeWithUint32("w_pid", event.Waker.Pid)
	spyEvent.SetUserAttributeWithUint32("w_tgid", event.Waker.Tgid)
	spyEvent.SetUserAttributeWithUint32("w_onrq_oncpu", uint32(event.Waker.Oncpu_ns-event.Waker.Onrq_ns))
	spyEvent.SetUserAttributeWithUint32("w_offcpu_oncpu", uint32(event.Waker.Oncpu_ns-event.Waker.Offcpu_ns))
	// todo
	spyEvent.SetUserAttributeWithByteBuf("w_comm", event.Waker.Comm[:])
	syms := bytes.NewBuffer(nil)
	b.ResolveStack(syms, event.Waker.Kern_stack_id, 0, false)
	// split kernel and user stack info
	syms.Write([]byte("----"))
	b.ResolveStack(syms, event.Waker.User_stack_id, event.Waker.Tgid, true)
	spyEvent.SetUserAttributeWithByteBuf("w_stack", syms.Bytes())

	spyEvent.SetUserAttributeWithUint32("wt_pid", event.Waker.T_pid)
	spyEvent.SetUserAttributeWithByteBuf("wt_comm", event.Waker.T_Comm[:])
	spyEvent.SetUserAttributeWithUint32("t_pid", event.Target.Pid)
	spyEvent.SetUserAttributeWithUint32("t_tgid", event.Target.Tgid)
	spyEvent.SetUserAttributeWithUint32("t_onrq_oncpu", uint32(event.Target.Oncpu_ns-event.Target.Onrq_ns))
	spyEvent.SetUserAttributeWithUint32("t_offcpu_oncpu", uint32(event.Target.Oncpu_ns-event.Target.Offcpu_ns))
	// todo
	spyEvent.SetUserAttributeWithByteBuf("t_comm", event.Target.Comm[:])
	spyEvent.SetUserAttributeWithUint64("t_start_time", event.Target.Offcpu_ns)
	spyEvent.SetUserAttributeWithUint64("t_end_time", event.Target.Oncpu_ns)
	spyEvent.SetUserAttributeWithUint64("ts", event.Ts)

	syms.Reset()
	b.ResolveStack(syms, event.Target.Kern_stack_id, 0, false)
	syms.Write([]byte("----"))
	b.ResolveStack(syms, event.Target.User_stack_id, event.Target.Tgid, true)
	spyEvent.SetUserAttributeWithByteBuf("t_stack", syms.Bytes())

	/*
		fmt.Printf("waker:%s, pid:%d, tgid:%d, target:%s, t_pid:%d, uid:%d, kid:%d\n", string(event.Waker.Comm[:]),
			event.Waker.Pid, event.Waker.Tgid, string(event.Waker.T_Comm[:]), event.Waker.Pid, event.Waker.User_stack_id, event.Waker.Kern_stack_id)
		fmt.Printf("target:%s, pid:%d, tgid:%d, uid:%d, kid:%d\n", string(event.Target.Comm[:]),
			event.Target.Pid, event.Target.Tgid, event.Target.User_stack_id, event.Target.Kern_stack_id)
	*/
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
