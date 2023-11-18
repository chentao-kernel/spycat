package receiver

import (
	"sync/atomic"

	"github.com/chentao-kernel/spycat/pkg/core/model"
)

type eventstat interface {
	Add(name string, value int64)
	GetStats() map[string]int64
}

type EventStats struct {
	stats map[string]*atomicInt64Counter
}

func (e *EventStats) Add(name string, value int64) {
	c, ok := e.stats[name]
	if ok {
		c.add(value)
	} else {
		c = e.stats[model.OtherEvent]
		c.add(value)
	}
}

func (e *EventStats) GetStats() map[string]int64 {
	m := make(map[string]int64, len(e.stats))
	for k, v := range e.stats {
		m[k] = v.get()
	}
	return m
}

func NewEventStats(sessions []Session) *EventStats {
	s := &EventStats{
		stats: make(map[string]*atomicInt64Counter),
	}

	for _, session := range sessions {
		s.stats[session.Name] = &atomicInt64Counter{0}
	}

	s.stats[model.OtherEvent] = &atomicInt64Counter{0}
	return s
}

type atomicInt64Counter struct {
	v int64
}

func (c *atomicInt64Counter) add(value int64) {
	atomic.AddInt64(&c.v, value)
}

func (c *atomicInt64Counter) get() int64 {
	return atomic.LoadInt64(&c.v)
}
