package detector

import "github.com/chentao-kernel/spycat/pkg/core/model"

type Detector interface {
	Init(cfg any) error
	Start() error
	Stop() error
	ConsumeEvent(e *model.SpyEvent) error
	OwnedEvents() []string
	Name() string
}
