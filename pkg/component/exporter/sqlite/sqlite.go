package sqlitexporter

import (
	"sync"
	"time"

	"github.com/chentao-kernel/spycat/pkg/core/model"
	"github.com/chentao-kernel/spycat/pkg/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var batchCacheSize = 200

type SqliteExporter struct {
	name      string
	config    *Config
	waitGroup sync.WaitGroup
	quit      chan struct{}
	datas     chan Item
	DB        *gorm.DB
}

func NewSqliteExporter(cfg interface{}) *SqliteExporter {
	config, _ := cfg.(*Config)
	var err error
	server := &SqliteExporter{
		name:   "sqlite_exporter",
		config: config,
		quit:   make(chan struct{}),
		datas:  make(chan Item, 5000),
	}
	server.DB, err = gorm.Open(sqlite.Open(server.config.Path), &gorm.Config{})
	if err != nil {
		log.Loger.Error("sqlite open failed:%v", err)
	}
	// wait groutine
	server.waitGroup.Add(1)

	// migrate table
	server.DB.AutoMigrate(&OFFCPU{})
	server.DB.AutoMigrate(&ONCPU{})
	server.DB.AutoMigrate(&SYSCALL{})
	server.DB.AutoMigrate(&FUTEXSNOOP{})
	server.DB.AutoMigrate(&CACHESTAT{})

	go server.Run()

	return server
}

func (s *SqliteExporter) ShutDown() {
	// notify Run grountine quit
	close(s.quit)
	// wait until Run() grountine quit
	s.waitGroup.Wait()
	// close db connection
	db, _ := s.DB.DB()
	db.Close()
}

// once a new table added in table.go. it also should be added here
func (s *SqliteExporter) Flush(batch []Item) {
	var offcpus []OFFCPU
	var oncpus []ONCPU
	var syscalls []SYSCALL
	var cachestats []CACHESTAT
	var futexsnoops []FUTEXSNOOP
	var table string

	if batch == nil {
		return
	}
	// just for different tables
	for _, val := range batch {
		if val.TableName() == "OFFCPU" {
			item, ok := val.(*OFFCPU)
			if ok {
				offcpus = append(offcpus, *item)
			}
		}
		if val.TableName() == "ONCPU" {
			oncpu, ok := val.(*ONCPU)
			if ok {
				oncpus = append(oncpus, *oncpu)
			}
		}
		if val.TableName() == "SYSCALL" {
			syscall, ok := val.(*SYSCALL)
			if ok {
				syscalls = append(syscalls, *syscall)
			}
		}
		if val.TableName() == "CACHESTAT" {
			cachestat, ok := val.(*CACHESTAT)
			if ok {
				cachestats = append(cachestats, *cachestat)
			}
		}
		if val.TableName() == "FUTEXSNOOP" {
			futexsnoop, ok := val.(*FUTEXSNOOP)
			if ok {
				futexsnoops = append(futexsnoops, *futexsnoop)
			}
		}
	}

	if len(offcpus) > 0 {
		table = "OFFCPU"
		result := s.DB.Table(table).CreateInBatches(offcpus, batchCacheSize)
		if result.Error != nil {
			log.Loger.Error("table:%s, sqlite flush failed:%v\n", table, result.Error)
		}
	}
	if len(oncpus) > 0 {
		table = "ONCPU"
		result := s.DB.Table(table).CreateInBatches(oncpus, batchCacheSize)
		if result.Error != nil {
			log.Loger.Error("table:%s, sqlite flush failed:%v\n", table, result.Error)
		}
	}
	if len(syscalls) > 0 {
		table = "SYSCALL"
		result := s.DB.Table(table).CreateInBatches(syscalls, batchCacheSize)
		if result.Error != nil {
			log.Loger.Error("table:%s, sqlite flush failed:%v\n", table, result.Error)
		}
	}
	if len(cachestats) > 0 {
		table = "CACHESTAT"
		result := s.DB.Table(table).CreateInBatches(cachestats, batchCacheSize)
		if result.Error != nil {
			log.Loger.Error("table:%s, sqlite flush failed:%v\n", table, result.Error)
		}
	}
	if len(futexsnoops) > 0 {
		table = "FUTEXSNOOP"
		result := s.DB.Table(table).CreateInBatches(futexsnoops, batchCacheSize)
		if result.Error != nil {
			log.Loger.Error("table:%s, sqlite flush failed:%v\n", table, result.Error)
		}
	}

	// just debug to check data write to db success or not
	// var count int64
	// s.DB.Table("OFFCPU").Count(&count)
	// fmt.Printf("tao records in %s:%d\n", "OFFCPU", count)
}

func (s *SqliteExporter) Run() {
	var batch []Item
	size := 0

	maxWait := time.NewTimer(s.config.BatchWait)

	defer func() {
		if size > 0 {
			s.Flush(batch)
		}
		s.waitGroup.Done()
		maxWait.Stop()
	}()

	for {
		select {
		case <-s.quit:
		case data := <-s.datas:
			batch = append(batch, data)
			size++
			if size > s.config.BatchEntriesNum {
				s.Flush(batch)
				// init again
				batch = []Item{}
				size = 0
				maxWait.Reset(s.config.BatchWait)
			}
		case <-maxWait.C:
			if size > 0 {
				s.Flush(batch)
				batch = []Item{}
				size = 0
			}
			maxWait.Reset(s.config.BatchWait)
		}
	}
}

// data from processor
func (s *SqliteExporter) Consume(data *model.DataBlock) error {
	var im Item

	switch data.Name {
	case model.OffCpu:
		im = &OFFCPU{
			WakerPid:      uint32(data.Labels.GetIntValue(model.Tid_W)),
			WakerTid:      uint32(data.Labels.GetIntValue(model.Pid_W)),
			WakerComm:     data.Labels.GetStringValue(model.Waker),
			WakerStack:    data.Labels.GetStringValue(model.Stack_W),
			WakeePid:      uint32(data.Labels.GetIntValue(model.Pid_W)),
			WakeeTid:      uint32(data.Labels.GetIntValue(model.Tid_W)),
			WakeeComm:     data.Labels.GetStringValue(model.Wakee),
			WakeeStack:    data.Labels.GetStringValue(model.Stack_T),
			WakeeOffCpuTs: data.Labels.GetStringValue(model.StartTime),
			DurMs:         uint32(data.Labels.GetIntValue(model.DurMs)),
			RunqDurMs:     uint32(data.Labels.GetIntValue(model.RunqDurMs)),
			CacheId:       uint32(data.Labels.GetIntValue(model.CacheId)),
			Prio:          uint32(data.Labels.GetIntValue(model.Prio)),
			Cpu:           uint32(data.Labels.GetIntValue(model.Cpu)),
		}
	case model.OnCpu:
		im = &ONCPU{
			Comm:  data.Labels.GetStringValue(model.Comm),
			Pid:   uint32(data.Labels.GetIntValue(model.Pid)),
			Tid:   uint32(data.Labels.GetIntValue(model.Tid)),
			Stack: data.Labels.GetStringValue(model.Stack),
			Count: uint32(data.Labels.GetIntValue(model.Count)),
		}
	case model.Syscall:
		im = &SYSCALL{
			Comm:    data.Labels.GetStringValue(model.Comm),
			Pid:     uint32(data.Labels.GetIntValue(model.Pid)),
			Tid:     uint32(data.Labels.GetIntValue(model.Tid)),
			Stack:   data.Labels.GetStringValue(model.Stack),
			Syscall: data.Labels.GetStringValue(model.Syscall),
			DurUs:   uint32(data.Labels.GetIntValue(model.DurUs)),
		}
	case model.FutexSnoop:
		im = &FUTEXSNOOP{
			Comm:       data.Labels.GetStringValue(model.Comm),
			Pid:        uint32(data.Labels.GetIntValue(model.Pid)),
			Tid:        uint32(data.Labels.GetIntValue(model.Tid)),
			Stack:      data.Labels.GetStringValue(model.Stack),
			UserCnt:    uint32(data.Labels.GetIntValue(model.UserCnt)),
			MaxUserCnt: uint32(data.Labels.GetIntValue(model.MaxUserCnt)),
			LockAddr:   int64(data.Labels.GetIntValue(model.LockAddr)),
			MinDurUS:   uint32(data.Labels.GetIntValue(model.MinDur)),
			MaxDurUs:   uint32(data.Labels.GetIntValue(model.MaxDur)),
			AvgDurUs:   uint32(data.Labels.GetIntValue(model.AvgDur)),
			DeltaDurUs: uint32(data.Labels.GetIntValue(model.DeltaDur)),
			LockCnt:    uint32(data.Labels.GetIntValue(model.LockCnt)),
		}
	case model.CacheStat:
		im = &CACHESTAT{
			Comm:       data.Labels.GetStringValue(model.Comm),
			Pid:        uint32(data.Labels.GetIntValue(model.Pid)),
			Cpu:        uint32(data.Labels.GetIntValue(model.Cpu)),
			ReadSizeM:  uint32(data.Labels.GetIntValue(model.ReadSizeM)),
			WriteSizeM: uint32(data.Labels.GetIntValue(model.WriteSizeM)),
			File:       data.Labels.GetStringValue(model.File),
		}
	}
	s.datas <- im
	return nil
}

func (s *SqliteExporter) Name() string {
	return s.name
}
