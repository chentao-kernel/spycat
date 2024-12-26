package sqlitexporter

import (
	"time"
)

type Item interface {
	TableName() string
}

// .tables
// .schema OFFCPU
// SELECT * FROM OFFCPU;
type OFFCPU struct {
	ID            uint64    `gorm:"primaryKey"`
	CreateAt      time.Time `gorm:"autoCreateTime"`
	WakerPid      uint32
	WakerTid      uint32
	WakerComm     string `grom:"size:16"`
	WakerStack    string `gorm:"size:100"`
	WakeePid      uint32
	WakeeTid      uint32
	WakeeComm     string `grom:"size:16"`
	WakeeStack    string `gorm:"size:100"`
	WakeeOffCpuTs string
	// wakee dur
	DurMs uint32
	// wakee runq dur
	RunqDurMs uint32
	// just for runq latency info
	Cpu     uint32
	CacheId uint32
	// task prio
	Prio uint32
}

func (o *OFFCPU) TableName() string {
	return "OFFCPU"
}

type ONCPU struct {
	ID       uint64    `gorm:"primaryKey"`
	CreateAt time.Time `gorm:"autoCreateTime"`
	Comm     string    `gorm:"size:16"`
	Pid      uint32
	Tid      uint32
	Stack    string `gorm:"size:100"`
	Count    uint32
}

func (o *ONCPU) TableName() string {
	return "ONCPU"
}
