package consumer

import "github.com/chentao-kernel/spycat/pkg/core/model"

type Consumer interface {
	Consume(data *model.DataBlock) error
	// todo 增加Stop接口清理资源, 如fileoutput中的fd
}
