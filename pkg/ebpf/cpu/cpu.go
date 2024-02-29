package cpu

import (
	"time"

	bpf "github.com/aquasecurity/libbpfgo"
	"github.com/chentao-kernel/spycat/pkg/core"
	"github.com/chentao-kernel/spycat/pkg/core/model"
	"github.com/chentao-kernel/spycat/pkg/log"
	"github.com/chentao-kernel/spycat/pkg/symtab"
)

type BpfSession struct {
	Session *core.Session
	// inner field
	PerfBuffer *bpf.PerfBuffer
	// inner field
	Module *bpf.Module
}

func NewBpfSession(name string, config *core.SessionConfig, buf chan *model.SpyEvent) core.BpfSpyer {

	symSession, err := symtab.NewSymSession()
	if err != nil {
		log.Loger.Error("sym session failed")
		return nil
	}
	if name == model.OffCpu {
		return &OffcpuSession{
			Session:    core.NewSession(name, config, buf),
			SymSession: symSession,
		}
	} else if name == model.OnCpu {
		return &OncpuSession{
			Session:    core.NewSession(name, config, buf),
			SymSession: symSession,
			sampleRate: 100,
			// 10秒上传一次数据
			ticker: time.NewTicker(10 * time.Second),
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
