package cpu

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"fmt"
	"sync"
	"time"
	"unsafe"

	bpf "github.com/aquasecurity/libbpfgo"
	"github.com/chentao-kernel/spycat/pkg/app/config"
	"github.com/chentao-kernel/spycat/pkg/core"
	"github.com/chentao-kernel/spycat/pkg/core/model"
	"github.com/chentao-kernel/spycat/pkg/log"
	"github.com/chentao-kernel/spycat/pkg/symtab"
	"github.com/chentao-kernel/spycat/pkg/util"
	"golang.org/x/sys/unix"
)

//#cgo CFLAGS: -I./oncpu/
//#include <linux/types.h>
//#include "oncpu.h"
import "C"

type OnCpuEvent struct {
	Pid       uint32
	KernStack int64
	UserStack int64
	Comm      [16]byte
}

type TaskCounter struct {
	counts uint64
	delta  uint64
}

type OnCpuUserArgs struct {
	cpu        int
	sampleRate int
}

type OncpuSession struct {
	Session *core.Session
	// inner filed
	PerfBuffer *bpf.PerfBuffer
	// inner filed
	Module        *bpf.Module
	SymSession    *symtab.SymSession
	mapStacks     *bpf.BPFMap
	mapCounts     *bpf.BPFMap
	prog          *bpf.BPFProg
	modMutex      sync.Mutex
	perfEventFds  []int
	lastTaskInfos map[OnCpuEvent]TaskCounter
	ticker        *time.Ticker
	cpus          []uint
	sampleRate    uint32
}

func NewOnCpuBpfSession(name string, cfg *config.ONCPU, buf chan *model.SpyEvent) core.BpfSpyer {

	symSession, err := symtab.NewSymSession(cfg.SymbolCacheSize)
	if err != nil {
		log.Loger.Error("sym session failed")
		return nil
	}

	return &OncpuSession{
		Session:    core.NewSession(name, &core.SessionConfig{}, buf),
		SymSession: symSession,
		sampleRate: uint32(cfg.SampleRate),
		// 10秒上传一次数据
		ticker: time.NewTicker(cfg.UploadRate),
	}
}

func (b *OncpuSession) attachPerfEvent() error {
	var err error

	if b.cpus == nil {
		if b.cpus, err = util.CpuGet(); err != nil {
			return err
		}
	}

	// 指定cpu profile
	for _, cpu := range b.cpus {
		attr := unix.PerfEventAttr{
			Type:   unix.PERF_TYPE_SOFTWARE,
			Config: unix.PERF_COUNT_SW_CPU_CLOCK,
			Bits:   unix.PerfBitFreq,
			Sample: uint64(b.sampleRate),
		}
		fd, err := unix.PerfEventOpen(&attr, -1, int(cpu), -1, unix.PERF_FLAG_FD_CLOEXEC)
		if err != nil {
			return err
		}
		b.perfEventFds = append(b.perfEventFds, fd)
		if _, err = b.prog.AttachPerfEvent(fd); err != nil {
			return err
		}
	}
	return nil
}

func (b *OncpuSession) PollData() {
	for {
		select {
		case <-b.ticker.C:
			b.Reset()
		}
	}
}

func (b *OncpuSession) getCountsMapValues() (keys [][]byte, values [][]byte, batch bool, err error) {
	// try lookup_and_delete_batch
	var (
		mapSize = C.PROFILE_MAPS_SIZE
		keySize = int(unsafe.Sizeof(C.struct_profile_key_t{}))
		allKeys = make([]byte, mapSize*keySize)
		pKeys   = unsafe.Pointer(&allKeys[0])
		nextKey = C.struct_profile_key_t{}
	)
	values, err = b.mapCounts.GetValueAndDeleteBatch(pKeys, nil, unsafe.Pointer(&nextKey), uint32(mapSize))
	if len(values) > 0 {
		keys = collectBatchValues(allKeys, len(values), keySize)
		return keys, values, true, nil
	}

	// batch failed or unsupported or just unlucky and got 0 stack-traces
	// try iterating
	it := b.mapCounts.Iterator()
	for it.Next() {
		key := it.Key()
		v, err := b.mapCounts.GetValue(unsafe.Pointer(&key[0]))
		if err != nil {
			return nil, nil, false, err
		}
		keyCopy := make([]byte, len(key)) // The slice is valid only until the next call to Next.
		copy(keyCopy, key)
		keys = append(keys, keyCopy)
		values = append(values, v)
	}
	return keys, values, false, nil
}

func (b *OncpuSession) clearCountsMap(keys [][]byte, batch bool) error {
	if len(keys) == 0 {
		return nil
	}
	if batch {
		// do nothing, already deleted with GetValueAndDeleteBatch in getCountsMapValues
		return nil
	}
	for _, key := range keys {
		err := b.mapCounts.DeleteKey(unsafe.Pointer(&key[0]))
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *OncpuSession) clearStacksMap(knownKeys map[uint32]bool) error {
	m := b.mapStacks
	cnt := 0
	errs := 0
	if b.SymSession.RoundNumber%10 == 0 {
		// do a full reset once in a while
		it := m.Iterator()
		var keys [][]byte
		for it.Next() {
			key := it.Key()
			keyCopy := make([]byte, len(key)) // The slice is valid only until the next call to Next.
			copy(keyCopy, key)
			keys = append(keys, keyCopy)
		}
		for _, key := range keys {
			if err := m.DeleteKey(unsafe.Pointer(&key[0])); err != nil {
				errs += 1
			} else {
				cnt += 1
			}
		}
		return nil
	}
	for stackId := range knownKeys {
		k := stackId
		if err := m.DeleteKey(unsafe.Pointer(&k)); err != nil {
			errs += 1
		} else {
			cnt += 1
		}
	}
	return nil
}

func (s *OncpuSession) Start() error {
	var err error

	fmt.Println("oncpu start trace")
	if err = unix.Setrlimit(unix.RLIMIT_MEMLOCK, &unix.Rlimit{
		Cur: unix.RLIM_INFINITY,
		Max: unix.RLIM_INFINITY,
	}); err != nil {
		return err
	}

	s.modMutex.Lock()
	defer s.modMutex.Unlock()

	args := bpf.NewModuleArgs{BPFObjBuff: onCpuBpf}
	if s.Module, err = bpf.NewModuleFromBufferArgs(args); err != nil {
		return err
	}

	if err = s.Module.BPFLoadObject(); err != nil {
		return err
	}

	if s.prog, err = s.Module.GetProgram("do_perf_event"); err != nil {
		return err
	}

	s.mapCounts, err = s.Module.GetMap("counts")
	if err != nil {
		return fmt.Errorf("get counts map failed:%v", err)
	}

	s.mapStacks, err = s.Module.GetMap("stacks")
	if err != nil {
		return fmt.Errorf("get stacks map failed:%v", err)
	}

	if err = s.attachPerfEvent(); err != nil {
		return err
	}

	s.PollData()
	return nil
}

func (b *OncpuSession) Reset() error {
	b.SymSession.RoundNumber += 1

	keys, values, batch, err := b.getCountsMapValues()
	if err != nil {
		return err
	}

	type sf struct {
		pid    uint32
		count  uint32
		kStack []byte
		uStack []byte
		comm   string
	}

	var sfs []sf
	knownStacks := map[uint32]bool{}
	for i, key := range keys {
		ck := (*C.struct_profile_key_t)(unsafe.Pointer(&key[0]))
		value := values[i]

		pid := uint32(ck.pid)
		kStackID := int64(ck.kern_stack)
		uStackID := int64(ck.user_stack)
		count := binary.LittleEndian.Uint32(value)
		var comm string = C.GoString(&ck.comm[0])
		if uStackID >= 0 {
			knownStacks[uint32(uStackID)] = true
		}
		if kStackID >= 0 {
			knownStacks[uint32(kStackID)] = true
		}

		uStack := b.getStack(int32(uStackID))
		kStack := b.getStack(int32(kStackID))
		sfs = append(sfs, sf{pid: pid, uStack: uStack, kStack: kStack, count: count, comm: comm})
	}
	for _, it := range sfs {
		buf := bytes.NewBuffer(nil)
		buf.Write([]byte(it.comm))
		buf.Write([]byte{';'})
		b.SymSession.WalkStack(buf, it.uStack, it.pid, true)
		b.SymSession.WalkStack(buf, it.kStack, 0, false)

		event := &model.SpyEvent{}
		event.Name = model.OnCpu
		event.Class.Event = model.OnCpu
		event.TimeStamp = uint64(time.Now().Unix())
		event.SetUserAttributeWithByteBuf("stack", buf.Bytes())
		event.SetUserAttributeWithUint32("count", it.count)
		event.SetUserAttributeWithUint32("pid", it.pid)
		event.SetUserAttributeWithByteBuf("comm", []byte(it.comm))
		event.SetUserAttributeWithUint32("sampleRate", b.sampleRate)

		b.Session.DataBuffer <- event
	}

	if err = b.clearCountsMap(keys, batch); err != nil {
		return err
	}
	if err = b.clearStacksMap(knownStacks); err != nil {
		return err
	}
	return nil
}

func (b *OncpuSession) Stop() error {
	if b.Module != nil {
		b.Module.Close()
	}
	if b.PerfBuffer != nil {
		b.PerfBuffer.Close()
	}
	return nil
}

func (b *OncpuSession) getStack(stackId int32) []byte {
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

func (b *OncpuSession) ResolveStack(syms *bytes.Buffer, stackId int32, pid uint32, userspace bool) error {
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

func (b *OncpuSession) HandleEvent(data []byte) {
	var event OnCpuEvent
	spyEvent := &model.SpyEvent{}
	if err := binary.Read(bytes.NewBuffer(data), binary.LittleEndian, &event); err != nil {
		log.Loger.Error("parse event: %s", err)
	}

	b.Session.DataBuffer <- spyEvent
}

func (b *OncpuSession) ReadEvent() error {
	return nil
}

func (b *OncpuSession) ConsumeEvent() error {
	return nil
}

func (b *OncpuSession) Name() string {
	return b.Session.Name()
}

func collectBatchValues(values []byte, count int, valueSize int) [][]byte {
	var value []byte
	var collected [][]byte
	for i := 0; i < count*valueSize; i += valueSize {
		value = values[i : i+valueSize]
		collected = append(collected, value)
	}
	return collected
}

//go:embed oncpu/oncpu.bpf.o
var onCpuBpf []byte
