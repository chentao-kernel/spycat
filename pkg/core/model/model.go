package model

import (
	"encoding/json"
)

type TimedEventKind int

const (
	TimedCpuEventKind TimedEventKind = iota
)

const (
	CpuEventLabel = "cpuEvents"
)

type TimedEvent interface {
	StartTimestamp() uint64
	EndTimestamp() uint64
	Kind() TimedEventKind
}

type Segment struct {
	StartTime      uint64       `json:"startTime"`
	EndTime        uint64       `json:"endTime"`
	CpuEvents      []TimedEvent `json:"cpuEvents"`
	IsSend         int
	IndexTimestamp string `json:"indexTimestamp"`
}

type TimeSegments struct {
	Pid        uint32       `json:"pid"`
	Tid        uint32       `json:"tid"`
	ThreadName string       `json:"threadName"`
	BaseTime   uint64       `json:"baseTime"`
	Segments   *CircleQueue `json:"segments"`
}

func (t *TimeSegments) UpdateThreadName(threadName string) {
	t.ThreadName = threadName
}

func NewSegment(startTime uint64, endTime uint64) *Segment {
	return &Segment{
		StartTime:      startTime,
		EndTime:        endTime,
		CpuEvents:      make([]TimedEvent, 0),
		IsSend:         0,
		IndexTimestamp: "",
	}
}

func (s *Segment) PutTimedEvent(event TimedEvent) {
	switch event.Kind() {
	case TimedCpuEventKind:
		s.CpuEvents = append(s.CpuEvents, event)
	}
}

func (s *Segment) toDataBlock(parent *TimeSegments) *DataBlock {
	labels := NewAttributeMap()
	labels.AddIntValue(Pid, int64(parent.Pid))
	labels.AddIntValue(Tid, int64(parent.Tid))
	labels.AddIntValue(IsSent, int64(s.IsSend))
	labels.AddStringValue(ThreadName, parent.ThreadName)
	labels.AddIntValue(StartTime, int64(s.StartTime))
	labels.AddIntValue(EndTime, int64(s.EndTime))
	cpuEventString, err := json.Marshal(s.CpuEvents)
	if err == nil {
		labels.AddStringValue(CpuEventLabel, string(cpuEventString))
	}
	return NewDataBlock(CpuEventBlockName, labels, s.StartTime)
}

func (s *Segment) isNotEmpty() bool {
	return len(s.CpuEvents) > 0
}

func (s *Segment) UnmarshalJSON(data []byte) error {
	events := make(map[string]json.RawMessage)
	err := json.Unmarshal(data, &events)

	if err != nil {
		return err
	}

	for k, v := range events {
		switch k {
		case "startTime":
			var t uint64
			err := json.Unmarshal(v, &t)
			if err != nil {
				return err
			}
			s.StartTime = t
		case "endTime":
			var t uint64
			err := json.Unmarshal(v, &t)
			if err != nil {
				return err
			}
			s.EndTime = t
		case CpuEventLabel:
			var e []CpuEvent
			err := json.Unmarshal(v, &e)
			if err != nil {
				return err
			}
			for i, _ := range e {
				s.CpuEvents = append(s.CpuEvents, &e[i])
			}
		default:
			//return errors.New("unrecognized key")
		}
	}
	return nil
}

type CPUType uint8

const (
	CPUType_ON CPUType = 0
)

func (ct CPUType) MarshalJSON() ([]byte, error) {
	return json.Marshal(uint16(ct))
}

func (ct *CPUType) UnmarshalJSON(data []byte) error {
	var val uint16
	err := json.Unmarshal(data, &val)
	if err != nil {
		return err
	}
	*ct = CPUType(val)
	return nil
}

type CpuEvent struct {
	TimeStamp  uint64
	StartTime  uint64
	EndTime    uint64
	Waker      string
	Pid_W      uint32
	Target     string
	Pid_WT     uint32
	WTarget    string
	Pid_T      uint32
	IrqOffUs_W uint32
	CpuOffUs_W uint32
	RunLatUs_W uint32
	Stack_W    string
	IrqOffUs_T uint32
	CpuOffUs_T uint32
	RunLatUs_T uint32
	Stack_T    string
	Log        string
}

func (c *CpuEvent) StartTimestamp() uint64 {
	return c.StartTime
}

func (c *CpuEvent) EndTimestamp() uint64 {
	return c.EndTime
}

func (c *CpuEvent) Kind() TimedEventKind {
	return TimedCpuEventKind
}

type CircleQueue struct {
	length int
	data   []interface{}
}

func NewCircleQueue(length int) *CircleQueue {
	return &CircleQueue{
		length: length,
		data:   make([]interface{}, length),
	}
}

func (s *CircleQueue) GetByIndex(index int) interface{} {
	return s.data[index]
}

func (s *CircleQueue) UpdateByIndex(index int, val interface{}) {
	s.data[index] = val
	return
}

func (s *CircleQueue) Clear() {
	s.data = make([]interface{}, s.length)
}
