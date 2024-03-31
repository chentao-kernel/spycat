package core

import (
	"github.com/chentao-kernel/spycat/pkg/log"
	"github.com/cilium/ebpf/perf"
	"github.com/cilium/ebpf/ringbuf"
	"golang.org/x/time/rate"
)

type ReaderType int

const (
	PerfReader ReaderType = iota
	RingBufReader
)

type BpfReader struct {
	PerfReader    *perf.Reader
	RingBufReader *ringbuf.Reader
}

type SessionReader struct {
	ReaderType ReaderType
	Limiter    *rate.Limiter
	BpfReader  BpfReader
	BufSize    int
	Data       any
	DataChan   chan any
}

func NewSessionReader(bufSize int, perfReader *perf.Reader, ringBuf *ringbuf.Reader) *SessionReader {
	var reader ReaderType

	if perfReader != nil {
		reader = PerfReader
	} else if ringBuf != nil {
		reader = RingBufReader
	} else {
		log.Loger.Error("perf:{}, ringbuf:P{}", perfReader, ringBuf)
	}

	return &SessionReader{
		ReaderType: reader,
		Limiter:    rate.NewLimiter(2, 1),
		BpfReader: BpfReader{
			PerfReader:    perfReader,
			RingBufReader: ringBuf,
		},
		BufSize:  bufSize,
		DataChan: make(chan any, 1000),
	}
}

func (reader *SessionReader) DataRead() (any, error) {
	return nil, nil
}
