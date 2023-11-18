package cpu

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"
	"unsafe"

	bpf "github.com/aquasecurity/libbpfgo"
	"github.com/chentao-kernel/spycat/pkg/component/detector/cpudetector"
	"github.com/chentao-kernel/spycat/pkg/core"
	"github.com/chentao-kernel/spycat/pkg/core/model"
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
}

type OffcpuSession struct {
	Session *core.Session
	// inner filed
	PerfBuffer *bpf.PerfBuffer
	// inner filed
	Module *bpf.Module
}

func attachProgs(bpfModule *bpf.Module) error {
	progIter := bpfModule.Iterator()
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

func initArgsMap(bpfModule *bpf.Module) error {
	var id uint32 = 0
	args := &UserArgs{
		Pid:           math.MaxUint32,
		Tgid:          math.MaxUint32,
		Min_offcpu_ms: 1,
		Max_offcpu_ms: 1000000,
	}

	maps, err := bpfModule.GetMap("args_map")
	if err != nil {
		return fmt.Errorf("get map failed:%v", err)
	}
	maps.Update(unsafe.Pointer(&id), unsafe.Pointer(args))
	fmt.Printf("update user_args succeess:\n")
	util.PrintStructFields(*args)
	return nil
}

func (b *OffcpuSession) bpfSessionInit(module *bpf.Module, perf *bpf.PerfBuffer) {
	b.Module = module
	b.PerfBuffer = perf
}

func (b *OffcpuSession) pollData(bpfModule *bpf.Module) {
	dataChan := make(chan []byte)
	lostChan := make(chan uint64)

	pb, err := bpfModule.InitPerfBuf("perf_map", dataChan, lostChan, 1)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	b.bpfSessionInit(bpfModule, pb)
	pb.Start()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer func() {
		pb.Stop()
		pb.Close()
		stop()
	}()
	fmt.Printf("%-8s %-6s %-6s %-5s %-5s %-5s %-5s\n",
		"TIME", "PID", "CPU1", "CPU2", "SPORT", "DPORT", "DELAY(us)")
loop:
	for {
		select {
		case data := <-dataChan:
			b.ReadData(data)
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

	err := unix.Setrlimit(unix.RLIMIT_MEMLOCK, &unix.Rlimit{
		Cur: unix.RLIM_INFINITY,
		Max: unix.RLIM_INFINITY,
	})
	if err != nil {
		return err
	}

	args := bpf.NewModuleArgs{BPFObjBuff: offcpuBpf}

	module, err := bpf.NewModuleFromBufferArgs(args)
	if err != nil {
		return fmt.Errorf("load module failed:%v", err)
	}

	err = module.BPFLoadObject()
	if err != nil {
		return fmt.Errorf("load object failed:%v", err)
	}

	err = initArgsMap(module)
	if err != nil {
		return fmt.Errorf("init args map failed:%v", err)
	}

	err = attachProgs(module)
	if err != nil {
		return fmt.Errorf("attach program failed:%v", err)
	}
	b.pollData(module)
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

func (b *OffcpuSession) ResolveStack() []byte {
	return nil
}

func (b *OffcpuSession) ReadData(data []byte) {
	var event OffCpuEvent
	spyEvent := &model.SpyEvent{}
	if err := binary.Read(bytes.NewBuffer(data), binary.LittleEndian, &event); err != nil {
		log.Printf("parse event: %s", err)
	}
	//util.PrintStructFields(event)
	spyEvent.Name = "offcpu"
	spyEvent.TimeStamp = event.Ts
	spyEvent.Class.Name = cpudetector.DetectorCpuType
	spyEvent.Class.Event = model.OffCpu
	spyEvent.Task.Pid = event.Target.Tgid
	spyEvent.Task.Tid = event.Target.Pid
	spyEvent.Task.Comm = string(event.Target.Comm[:])
	spyEvent.SetUserAttributeWithUint32("w_pid", event.Waker.Pid)
	spyEvent.SetUserAttributeWithUint32("w_onrq_oncpu", uint32(event.Waker.Oncpu_ns-event.Waker.Onrq_ns))
	spyEvent.SetUserAttributeWithUint32("w_offcpu_oncpu", uint32(event.Waker.Oncpu_ns-event.Waker.Offcpu_ns))
	// todo
	spyEvent.SetUserAttributeWithByteBuf("w_stack", event.Waker.Comm[:])
	spyEvent.SetUserAttributeWithByteBuf("w_comm", event.Waker.Comm[:])

	spyEvent.SetUserAttributeWithUint32("wt_pid", event.Waker.T_pid)
	spyEvent.SetUserAttributeWithByteBuf("wt_comm", event.Waker.T_Comm[:])
	spyEvent.SetUserAttributeWithUint32("t_pid", event.Target.Pid)
	spyEvent.SetUserAttributeWithUint32("t_onrq_oncpu", uint32(event.Target.Oncpu_ns-event.Target.Onrq_ns))
	spyEvent.SetUserAttributeWithUint32("t_offcpu_oncpu", uint32(event.Target.Oncpu_ns-event.Target.Offcpu_ns))
	// todo
	spyEvent.SetUserAttributeWithByteBuf("t_stack", event.Target.Comm[:])
	spyEvent.SetUserAttributeWithByteBuf("t_comm", event.Target.Comm[:])
	spyEvent.SetUserAttributeWithUint64("t_start_time", event.Target.Offcpu_ns)
	spyEvent.SetUserAttributeWithUint64("t_end_time", event.Target.Oncpu_ns)
	spyEvent.SetUserAttributeWithUint64("ts", event.Ts)
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
