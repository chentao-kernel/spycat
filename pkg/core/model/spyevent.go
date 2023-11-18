package model

import "fmt"

type Protocol uint32
type ValueType int32

const (
	ValueType_NONE    ValueType = 0
	ValueType_INT8    ValueType = 1
	ValueType_INT16   ValueType = 2
	ValueType_INT32   ValueType = 3
	ValueType_INT64   ValueType = 4
	ValueType_UINT8   ValueType = 5
	ValueType_UINT16  ValueType = 6
	ValueType_UINT32  ValueType = 7
	ValueType_UINT64  ValueType = 8
	ValueType_CHARBUF ValueType = 9
	ValueType_BYTEBUF ValueType = 10
	ValueType_FLOAT   ValueType = 11
	ValueType_DOUBLE  ValueType = 12
	ValueType_BOOL    ValueType = 13
)

var ValueType_name = map[int32]string{
	0:  "NONE",
	1:  "INT8",
	2:  "INT16",
	3:  "INT32",
	4:  "INT64",
	5:  "UINT8",
	6:  "UINT16",
	7:  "UINT32",
	8:  "UINT64",
	9:  "CHARBUF",
	10: "BYTEBUF",
	11: "FLOAT",
	12: "DOUBLE",
	13: "BOOL",
}

const (
	ProtoUnknown = 0
	ProtoTcp     = 1
	ProtoUdp     = 2
	ProtoIcmp    = 3
	ProtoRaw     = 4
)

var ValueType_value = map[string]int32{
	"NONE":    0,
	"INT8":    1,
	"INT16":   2,
	"INT32":   3,
	"INT64":   4,
	"UINT8":   5,
	"UINT16":  6,
	"UINT32":  7,
	"UINT64":  8,
	"CHARBUF": 9,
	"BYTEBUF": 10,
	"FLOAT":   11,
	"DOUBLE":  12,
	"BOOL":    13,
}

type TaskInfo struct {
	Pid           uint32
	Tid           uint32
	Comm          string
	ContainerId   string
	ContainerName string
	NameSpace     string
	Stack         uint8
	Latency       uint32
}

type IP []uint32

type NetInfo struct {
	Proto Protocol
	Sip   IP
	Dip   IP
	Sport uint16
	Dport uint16
}

type EventData struct {
	Task    TaskInfo
	Net     NetInfo
	RawData []uint8
}

type ClassInfo struct {
	Name  string // 表示归属对应的detecor
	Event string // 表示对应的事件
}

type KeyValue struct {
	Key       string
	ValueType ValueType
	Value     []byte
}

type CpuEvent struct {
}

type SpyEvent struct {
	Name      string
	TimeStamp uint64
	Class     ClassInfo
	Task      TaskInfo
	// UserAttributes 数量
	ParamsCnt      uint16
	UserAttributes [16]KeyValue
}

func (s *SpyEvent) GetUserAttributes() *[16]KeyValue {
	return &s.UserAttributes
}

func (s *SpyEvent) SetUserAttribute(kv KeyValue) error {
	if s.ParamsCnt > 15 {
		return fmt.Errorf("exceeded the max user attrubute size:%d", s.ParamsCnt)
	}
	s.UserAttributes[s.ParamsCnt] = kv
	s.ParamsCnt++
	return nil
}

func (s *SpyEvent) GetUserAttribute(key string) *KeyValue {
	if s.ParamsCnt == 0 {
		return nil
	}

	for id, kv := range s.UserAttributes {
		if id+1 > int(s.ParamsCnt) {
			break
		}
		if kv.Key == key {
			return &kv
		}
	}
	return nil
}

func (kv *KeyValue) GetKey() string {
	if kv != nil {
		return kv.Key
	}
	return ""
}

func (kv *KeyValue) GetValueType() ValueType {
	if kv != nil {
		return kv.ValueType
	}
	return ValueType_NONE
}

func (kv *KeyValue) GetValue() []byte {
	if kv != nil {
		return kv.Value
	}
	return nil
}
