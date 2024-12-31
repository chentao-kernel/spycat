package mem

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
	"github.com/chentao-kernel/spycat/pkg/util"
	"golang.org/x/sys/unix"
)

// #cgo CFLAGS: -I./cachestat/
// #include <linux/types.h>
// #include "cachestat.h"
import "C"

type CacheStat struct {
	Pid       uint32
	Cpu       uint32
	Comm      [16]byte
	File      [64]byte
	ReadSize  uint64
	WriteSize uint64
	Cnt       uint64
}

type CacheStatArgs struct {
	Pid       uint32
	CacheType uint32
	Cpu       uint32
	Pad       uint32
}

type CacheStatSession struct {
	Session      *core.Session
	Module       *bpf.Module
	mapCacheStat *bpf.BPFMap
	Args         CacheStatArgs
	ticker       *time.Ticker
	BtfPath      string
	modMutex     sync.Mutex
}

func NewCacheStatBpfSession(name string, cfg *config.CACHESTAT, buf chan *model.SpyEvent) core.BpfSpyer {
	return &CacheStatSession{
		Session: core.NewSession(name, &core.SessionConfig{}, buf),
		Args: CacheStatArgs{
			Pid:       uint32(cfg.Pid),
			CacheType: uint32(cfg.CacheType),
			Cpu:       uint32(cfg.Cpu),
		},
		BtfPath: cfg.BtfPath,
		ticker:  time.NewTicker(cfg.UploadRate),
	}
}

func (b *CacheStatSession) attachProgs() error {
	progIter := b.Module.Iterator()
	for {
		prog := progIter.NextProgram()
		if prog == nil {
			break
		}
		if _, err := prog.AttachGeneric(); err != nil {
			return err
		}
		fmt.Printf("cachestat prog:%s attach success\n", prog.GetName())
	}
	return nil
}

func (b *CacheStatSession) initArgsMap() error {
	var id uint32
	args := &CacheStatArgs{
		Pid:       b.Args.Pid,
		CacheType: b.Args.CacheType,
		Cpu:       b.Args.Cpu,
	}
	maps, err := b.Module.GetMap("args_map")
	if err != nil {
		return fmt.Errorf("get args_map failed:%v", err)
	}
	maps.Update(unsafe.Pointer(&id), unsafe.Pointer(args))
	fmt.Printf("update user_args success\n")
	util.PrintStructFields(b.Args)
	return nil
}

// revive:disable:function-result-limit
func (b *CacheStatSession) getCacheStatMapValues() (keys [][]byte, values [][]byte, batch bool, err error) {
	var (
		mapSize = C.CACHESTAT_MAP_SIZE
		keySize = int(unsafe.Sizeof(C.struct_cache_key{}))
		allKeys = make([]byte, mapSize*keySize)
		pKeys   = unsafe.Pointer(&allKeys[0])
		nextKey = C.struct_cache_key{}
	)
	values, err = b.mapCacheStat.GetValueBatch(pKeys, nil, unsafe.Pointer(&nextKey), uint32(mapSize))
	if len(values) > 0 {
		keys = collectBatchValues(allKeys, len(values), keySize)
		return keys, values, true, nil
	}
	// try iterating
	it := b.mapCacheStat.Iterator()
	for it.Next() {
		key := it.Key()
		v, err := b.mapCacheStat.GetValue(unsafe.Pointer(&key[0]))
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

func (b *CacheStatSession) Reset() error {
	_, values, _, err := b.getCacheStatMapValues()
	if err != nil {
		return err
	}

	for _, value := range values {
		v := (*C.struct_cache_info)(unsafe.Pointer(&value[0]))
		pid := uint32(v.pid)
		cpu := uint32(v.cpu)
		cnt := uint64(v.cnt)
		comm := C.GoString(&v.comm[0])
		file := C.GoString(&v.file[0])
		read_size := uint64(v.read_size)
		write_size := uint64(v.write_size)

		event := &model.SpyEvent{}
		event.Name = model.CacheStat
		event.Class.Event = model.CacheStat
		event.TimeStamp = uint64(time.Now().Unix())

		event.SetUserAttributeWithByteBuf("comm", []byte(comm))
		event.SetUserAttributeWithByteBuf("file", []byte(file))
		event.SetUserAttributeWithUint32("pid", pid)
		event.SetUserAttributeWithUint32("cpu", cpu)
		event.SetUserAttributeWithUint64("cnt", cnt)
		event.SetUserAttributeWithUint64("read_size_m", read_size)
		event.SetUserAttributeWithUint64("write_size_m", write_size)
		// fmt.Printf("pid:%d,comm:%s,file:%s,read_size:%d,write_size:%d\n", pid, comm, file, read_size, write_size)
		b.Session.DataBuffer <- event
	}
	return nil
}

func (b *CacheStatSession) PollData() {
	for {
		select {
		case <-b.ticker.C:
			b.Reset()
		}
	}
}

func (b *CacheStatSession) Start() error {
	var err error

	fmt.Println("cachestat start trace")
	if err = unix.Setrlimit(unix.RLIMIT_MEMLOCK, &unix.Rlimit{
		Cur: unix.RLIM_INFINITY,
		Max: unix.RLIM_INFINITY,
	}); err != nil {
		return err
	}

	b.modMutex.Lock()
	defer b.modMutex.Unlock()

	args := bpf.NewModuleArgs{
		BPFObjBuff: cachestatBpf,
		BTFObjPath: b.BtfPath,
	}

	if b.Module, err = bpf.NewModuleFromBufferArgs(args); err != nil {
		return err
	}
	if err = b.Module.BPFLoadObject(); err != nil {
		return err
	}

	err = b.initArgsMap()
	if err != nil {
		return fmt.Errorf("init args map failed:%v", err)
	}

	b.mapCacheStat, err = b.Module.GetMap("cachestat_map")
	if err != nil {
		return fmt.Errorf("cachestat get cachestat map failed:%v", err)
	}

	err = b.attachProgs()
	if err != nil {
		return fmt.Errorf("cachestat attach program failed:%v", err)
	}

	b.PollData()
	return nil
}

func (b *CacheStatSession) HandleEvent(data []byte) {
	var event CacheStat

	spyEvent := &model.SpyEvent{}
	if err := binary.Read(bytes.NewBuffer(data), binary.LittleEndian, &event); err != nil {
		log.Loger.Error("cachestat parse event: %s", err)
	}
	b.Session.DataBuffer <- spyEvent
}
func (b *CacheStatSession) Stop() error {
	if b.Module != nil {
		b.Module.Close()
	}
	return nil
}

func (b *CacheStatSession) ReadEvent() error {
	return nil
}

func (b *CacheStatSession) ConsumeEvent() error {
	return nil
}

func (b *CacheStatSession) Name() string {
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

//go:embed cachestat/cachestat.bpf.o
var cachestatBpf []byte
