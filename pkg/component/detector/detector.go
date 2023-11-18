package detector

import "github.com/chentao-kernel/spycat/pkg/core/model"

type Detector interface {
	Start() error
	Stop() error
	ConsumeEvent(e *model.SpyEvent) error
	OwnedEvents() []string
	Name() string
}
