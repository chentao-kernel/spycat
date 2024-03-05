package cpu

import (
	bpf "github.com/aquasecurity/libbpfgo"
	"github.com/chentao-kernel/spycat/pkg/core"
)

type BpfSession struct {
	Session *core.Session
	// inner field
	PerfBuffer *bpf.PerfBuffer
	// inner field
	Module *bpf.Module
}

func (s *BpfSession) Start() error {
	return nil
}

func (s *BpfSession) Stop() error {
	return nil
}

func (s *BpfSession) Name() string {
	return s.Session.Name()
}

func (s *BpfSession) ConsumeEvent() error {
	return nil
}
