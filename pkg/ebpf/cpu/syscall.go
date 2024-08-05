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
	"runtime"
	"strconv"
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

// #include "syscall/syscall_table.h"
// #include <stdlib.h>
import "C"

type SyscallEvent struct {
	Pid       uint32
	Tid       uint32
	TsNs      uint64
	DurUs     uint64
	UStackId  int64
	KStackId  int64
	SyscallId uint32
	Comm      [16]byte
}

type SyscallArgs struct {
	Pid       uint32
	Tid       uint32
	MinDurMs  uint32
	MaxDurMs  uint32
	Stack     bool
	SyscallId uint32
}

type SyscallSession struct {
	Session    *core.Session
	PerfBuffer *bpf.PerfBuffer
	Module     *bpf.Module
	SymSession *symtab.SymSession
	mapStacks  *bpf.BPFMap
	Args       SyscallArgs
	BtfPath    string
	Arch       string
}

func NewSyscallSession(name string, cfg *config.SYSCALL, buf chan *model.SpyEvent) core.BpfSpyer {
	symSession, err := symtab.NewSymSession(cfg.SymbolCacheSize)
	if err != nil {
		log.Loger.Error("sym sessoin failed")
		return nil
	}
	return &SyscallSession{
		Session:    core.NewSession(name, &core.SessionConfig{}, buf),
		SymSession: symSession,
		Args: SyscallArgs{
			Pid:       uint32(cfg.Pid),
			Tid:       uint32(cfg.Tid),
			MinDurMs:  uint32(cfg.MinDurMs),
			MaxDurMs:  uint32(cfg.MaxDurMs),
			Stack:     cfg.Stack,
			SyscallId: SyscallNameToId(cfg.Syscall),
		},
		BtfPath: cfg.BtfPath,
		Arch:    runtime.GOARCH,
	}
}

func (b *SyscallSession) attachProgs() error {
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

func (b *SyscallSession) initArgsMap() error {
	var id uint32
	args := &SyscallArgs{
		Pid:       b.Args.Pid,
		Tid:       b.Args.Tid,
		MinDurMs:  b.Args.MinDurMs,
		MaxDurMs:  b.Args.MaxDurMs,
		Stack:     b.Args.Stack,
		SyscallId: b.Args.SyscallId,
	}
	maps, err := b.Module.GetMap("args_map")
	if err != nil {
		return fmt.Errorf("get map failed:%v", err)
	}
	maps.Update(unsafe.Pointer(&id), unsafe.Pointer(args))
	fmt.Printf("update syscall args succeess:\n")
	util.PrintStructFields(b.Args)
	return nil
}

func (b *SyscallSession) pollData() {
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

func (b *SyscallSession) Start() error {
	fmt.Println("syscall start trace")
	err := unix.Setrlimit(unix.RLIMIT_MEMLOCK, &unix.Rlimit{
		Cur: unix.RLIM_INFINITY,
		Max: unix.RLIM_INFINITY,
	})
	if err != nil {
		return err
	}

	args := bpf.NewModuleArgs{
		BPFObjBuff: syscallBpf,
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

	err = b.attachProgs()
	if err != nil {
		return fmt.Errorf("attach program failed:%v", err)
	}

	b.pollData()
	return nil
}

func (b *SyscallSession) Stop() error {
	if b.Module != nil {
		b.Module.Close()
	}
	if b.PerfBuffer != nil {
		b.PerfBuffer.Close()
	}
	return nil
}

func (b *SyscallSession) getStack(stackId int32) []byte {
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

func (b *SyscallSession) ResolveStack(syms *bytes.Buffer, stackId int32, pid uint32, userspace bool) error {
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

func (b *SyscallSession) HandleEvent(data []byte) {
	spyEvent := &model.SpyEvent{}

	spyEvent.Name = model.Syscall
	spyEvent.TimeStamp = uint64(time.Now().Unix())
	spyEvent.Class.Name = cpudetector.DetectorCpuType
	spyEvent.Class.Event = model.Syscall

	var event SyscallEvent
	err := binary.Read(bytes.NewBuffer(data), binary.LittleEndian, &event)
	if err != nil {
		log.Loger.Error("parse event: %v", err)
	}

	spyEvent.Task.Pid = uint32(event.Pid)
	spyEvent.Task.Tid = uint32(event.Tid)
	spyEvent.Task.Comm = string(event.Comm[:])
	spyEvent.SetUserAttributeWithUint64("dur_us", event.DurUs)
	spyEvent.SetUserAttributeWithUint32("pid", uint32(event.Pid))
	spyEvent.SetUserAttributeWithByteBuf("comm", event.Comm[:])
	spyEvent.SetUserAttributeWithByteBuf("syscall", []byte(b.SyscallIdToName(uint32(event.SyscallId))))
	syms := bytes.NewBuffer(nil)
	b.ResolveStack(syms, int32(event.KStackId), 0, false)
	syms.Write([]byte("----"))
	b.ResolveStack(syms, int32(event.UStackId), spyEvent.Task.Pid, true)
	spyEvent.SetUserAttributeWithByteBuf("stack", syms.Bytes())
	// fmt.Printf("addr:%x, min_dur:%d, max_dur:%d, avg_dur:%d, delta:_dur:%d, comm:%s\n", task.Uaddr, task.MinDur,
	// 	task.MaxDur, avg, task.DeltaDur, spyEvent.Task.Comm)

	b.Session.DataBuffer <- spyEvent
}

func (b *SyscallSession) ReadEvent() error {
	return nil
}

func (b *SyscallSession) ConsumeEvent() error {
	return nil
}

func (b *SyscallSession) Name() string {
	return b.Session.Name()
}

func (b *SyscallSession) SyscallIdToName(id uint32) string {
	switch b.Arch {
	case "amd64":
		if int(id) >= len(C.syscall_table_x86) {
			return "unknown_" + strconv.Itoa(int(id))
		}
		raw_name := C.syscall_table_x86[C.int(id)]
		name := C.GoString(raw_name)
		return name
	case "arm":
		fallthrough
	case "arm64":
		if int(id) >= len(C.syscall_table_arm) {
			return "unknown_" + strconv.Itoa(int(id))
		}
		raw_name := C.syscall_table_arm[C.int(id)]
		name := C.GoString(raw_name)
		return name
	default:
		log.Loger.Error("Unknown arch:%s\n", b.Arch)
	}
	return "unknown"
}

func SyscallNameToId(name string) uint32 {
	switch runtime.GOARCH {
	case "amd64":
		for i := 0; i < len(C.syscall_table_x86); i++ {
			c_name := C.syscall_table_x86[C.int(i)]
			go_name := C.GoString(c_name)
			if name == go_name {
				return uint32(i)
			}
		}
	case "arm":
		fallthrough
	case "arm64":
		for i := 0; i < len(C.syscall_table_arm); i++ {
			c_name := C.syscall_table_arm[C.int(i)]
			go_name := C.GoString(c_name)
			if go_name == name {
				return uint32(i)
			}
		}
	default:
		log.Loger.Error("Unknown arch")
	}
	return math.MaxUint32
}

//go:embed syscall/syscall.bpf.o
var syscallBpf []byte
