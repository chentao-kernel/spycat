package consumer

import "github.com/chentao-kernel/spycat/pkg/core/model"

type Consumer interface {
	Consume(data *model.DataBlock) error
}
