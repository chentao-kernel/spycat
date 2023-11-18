package exporter

import "github.com/chentao-kernel/spycat/pkg/component/consumer"

type Exporter interface {
	consumer.Consumer
}
