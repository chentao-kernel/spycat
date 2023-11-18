package cpu

import (
	bpf "github.com/aquasecurity/libbpfgo"
	"github.com/chentao-kernel/spycat/pkg/core"
	"github.com/chentao-kernel/spycat/pkg/core/model"
	"github.com/chentao-kernel/spycat/pkg/log"
)

type BpfSession struct {
	Session *core.Session
	// inner filed
	PerfBuffer *bpf.PerfBuffer
	// inner filed
	Module *bpf.Module
}

func NewBpfSession(name string, config *core.SessionConfig, buf chan *model.SpyEvent) core.BpfSpyer {
	if name == "offcpu" {
		return &OffcpuSession{
			Session: core.NewSession(name, config, buf),
		}
	} else {
		log.Loger.Error("session name:%s unknown\n", name)
	}
	return nil
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
