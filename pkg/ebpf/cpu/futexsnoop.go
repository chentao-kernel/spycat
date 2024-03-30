package cpu

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/binary"
	"fmt"
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

type LockStat struct {
	UserCnt    uint32
	MaxUserCnt uint32
	Uaddr      uint64
	Comm       [16]byte
	PidTgid    uint64
	Ts         uint64
}

type TaskLockStat struct {
	Slots        [24]uint32
	Comm         [16]byte
	PidTgid      uint64
	Uaddr        uint64
	UserStackId  int32
	Pad          uint32
	Contended    uint64
	TotalElapsed uint64
	MinDur       uint64
	MaxDur       uint64
	DeltaDur     uint64
	MaxDurTs     uint64
}

type FutexSnoopArgs struct {
	Tid              uint32
	Pid              uint32
	TargetLock       uint64
	Stack            bool
	MinDurMs         uint32
	MaxDurMs         uint32
	MaxLockHoldUsers uint32
}

type FutexSnoopSession struct {
	Session    *core.Session
	PerfBuffer *bpf.PerfBuffer
	Module     *bpf.Module
	SymSession *symtab.SymSession
	mapStacks  *bpf.BPFMap
	Args       FutexSnoopArgs
}

func NewFutexSnoopSession(name string, cfg *config.FUTEXSNOOP, buf chan *model.SpyEvent) core.BpfSpyer {
	symSession, err := symtab.NewSymSession(cfg.SymbolCacheSize)
	if err != nil {
		log.Loger.Error("sym sessoin failed")
		return nil
	}
	return &FutexSnoopSession{
		Session:    core.NewSession(name, &core.SessionConfig{}, buf),
		SymSession: symSession,
		Args: FutexSnoopArgs{
			Pid:              uint32(cfg.Pid),
			Tid:              uint32(cfg.Tid),
			MinDurMs:         uint32(cfg.MinDurMs),
			MaxDurMs:         uint32(cfg.MaxDurMs),
			TargetLock:       uint64(cfg.TargetLock),
			Stack:            cfg.Stack,
			MaxLockHoldUsers: uint32(cfg.MaxLockHoldUsers),
		},
	}
}

func (b *FutexSnoopSession) attachProgs() error {
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

func (b *FutexSnoopSession) initArgsMap() error {
	var id uint32 = 0
	args := &FutexSnoopArgs{
		Pid:              b.Args.Pid,
		Tid:              b.Args.Tid,
		MinDurMs:         b.Args.MinDurMs,
		MaxDurMs:         b.Args.MaxDurMs,
		MaxLockHoldUsers: b.Args.MaxLockHoldUsers,
	}
	maps, err := b.Module.GetMap("args_map")
	if err != nil {
		return fmt.Errorf("get map failed:%v", err)
	}
	maps.Update(unsafe.Pointer(&id), unsafe.Pointer(args))
	fmt.Printf("update futexsnoop args succeess:\n")
	util.PrintStructFields(b.Args)
	return nil
}

func (b *FutexSnoopSession) pollData() {
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

func (b *FutexSnoopSession) Start() error {
	fmt.Println("futexsnoop start trace")
	err := unix.Setrlimit(unix.RLIMIT_MEMLOCK, &unix.Rlimit{
		Cur: unix.RLIM_INFINITY,
		Max: unix.RLIM_INFINITY,
	})
	if err != nil {
		return err
	}

	args := bpf.NewModuleArgs{BPFObjBuff: futexSnoopBpf}

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

func (b *FutexSnoopSession) Stop() error {
	if b.Module != nil {
		b.Module.Close()
	}
	if b.PerfBuffer != nil {
		b.PerfBuffer.Close()
	}
	return nil
}

func (b *FutexSnoopSession) getStack(stackId int32) []byte {
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

func (b *FutexSnoopSession) ResolveStack(syms *bytes.Buffer, stackId int32, pid uint32, userspace bool) error {
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

func (b *FutexSnoopSession) HandleEvent(data []byte) {
	sizeLock := unsafe.Sizeof(LockStat{})
	sizeTask := unsafe.Sizeof(TaskLockStat{})
	sizeData := len(data)
	spyEvent := &model.SpyEvent{}

	spyEvent.Name = model.FutexSnoop
	spyEvent.TimeStamp = uint64(time.Now().Unix())
	spyEvent.Class.Name = cpudetector.DetectorCpuType
	spyEvent.Class.Event = model.FutexSnoop

	if sizeData >= int(sizeLock) && sizeData < int(sizeTask) {
		var lock LockStat
		err := binary.Read(bytes.NewBuffer(data), binary.LittleEndian, &lock)
		if err != nil {
			log.Loger.Error("parse event: %v", err)
		}
		spyEvent.Task.Pid = uint32(lock.PidTgid >> 32)
		spyEvent.Task.Tid = uint32(lock.PidTgid)
		spyEvent.Task.Comm = string(lock.Comm[:])
		spyEvent.SetUserAttributeWithUint32("user_cnt", lock.UserCnt)
		spyEvent.SetUserAttributeWithUint32("max_user_cnt", lock.MaxUserCnt)
		spyEvent.SetUserAttributeWithUint64("lock_addr", lock.Uaddr)
		spyEvent.SetUserAttributeWithUint64("min_dur", 0)
		spyEvent.SetUserAttributeWithUint64("max_dur", 0)
		spyEvent.SetUserAttributeWithUint64("delta_dur", 0)
		spyEvent.SetUserAttributeWithUint64("avg_dur", 0)
		spyEvent.SetUserAttributeWithUint64("lock_cnt", 0)
		spyEvent.SetUserAttributeWithByteBuf("stack", []byte(""))
		//fmt.Printf("pid:%d,tid:%d,comm:%s,max_user_cnt:%d\n", spyEvent.Task.Pid,
		//	spyEvent.Task.Tid, spyEvent.Task.Comm, lock.MaxUserCnt)
	}

	if sizeData >= int(sizeTask) {
		var task TaskLockStat
		err := binary.Read(bytes.NewBuffer(data), binary.LittleEndian, &task)
		if err != nil {
			log.Loger.Error("parse event: %v", err)
		}
		var avg uint64
		if task.Contended == 0 {
			avg = 0
		} else {
			avg = task.TotalElapsed / task.Contended
		}
		spyEvent.Task.Pid = uint32(task.PidTgid >> 32)
		spyEvent.Task.Tid = uint32(task.PidTgid)
		spyEvent.Task.Comm = string(task.Comm[:])
		spyEvent.SetUserAttributeWithUint32("user_cnt", 0)
		spyEvent.SetUserAttributeWithUint32("max_user_cnt", 0)
		spyEvent.SetUserAttributeWithUint64("lock_addr", task.Uaddr)
		spyEvent.SetUserAttributeWithUint64("min_dur", task.MinDur)
		spyEvent.SetUserAttributeWithUint64("max_dur", task.MaxDur)
		spyEvent.SetUserAttributeWithUint64("delta_dur", task.DeltaDur)
		spyEvent.SetUserAttributeWithUint64("avg_dur", avg)
		spyEvent.SetUserAttributeWithUint64("lock_cnt", task.Contended)
		syms := bytes.NewBuffer(nil)
		b.ResolveStack(syms, task.UserStackId, spyEvent.Task.Pid, true)
		spyEvent.SetUserAttributeWithByteBuf("stack", syms.Bytes())
		// fmt.Printf("addr:%x, min_dur:%d, max_dur:%d, avg_dur:%d, delta:_dur:%d, comm:%s\n", task.Uaddr, task.MinDur,
		// 	task.MaxDur, avg, task.DeltaDur, spyEvent.Task.Comm)
	}

	b.Session.DataBuffer <- spyEvent
}

func (b *FutexSnoopSession) ReadEvent() error {
	return nil
}

func (b *FutexSnoopSession) ConsumeEvent() error {
	return nil
}

func (b *FutexSnoopSession) Name() string {
	return b.Session.Name()
}

//go:embed futexsnoop/futexsnoop.bpf.o
var futexSnoopBpf []byte
