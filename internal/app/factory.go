package appspy

import (
	"github.com/chentao-kernel/spycat/pkg/component/consumer"
	"github.com/chentao-kernel/spycat/pkg/component/detector"
	"github.com/chentao-kernel/spycat/pkg/component/exporter"
	"github.com/chentao-kernel/spycat/pkg/component/processor"
	"github.com/chentao-kernel/spycat/pkg/component/receiver"
)

// data folw of pipline:
// receiver --> detector --> processor --> exporter
type ComponentsFactory struct {
	// receiver used to receive ebpf data from cilium etc.
	Receivers map[string]ReceiverFactory
	// detector used to handle receiver data
	Detectors map[string]DetectorFactory
	// processor used to handle detector data as aggregator
	Processors map[string]ProcessorFactory
	// exporter used to handler processor data and send trace metric to
	// server like prometheus etc.
	Exporters map[string]ExporterFactory
}

type NewReceiverCb func(cfg any, managers *detector.DetectorManager) receiver.Receiver
type NewDetectorCb func(cfg any, consumers []consumer.Consumer) detector.Detector
type NewProcessorCb func(cfg any, consumers consumer.Consumer) processor.Processor
type NewExporterCb func(cfg any, consumers consumer.Consumer) exporter.Exporter

type ReceiverFactory struct {
	NewComponentMember NewReceiverCb
	Config             any
}

type DetectorFactory struct {
	NewComponentMember NewDetectorCb
	Config             any
}

type ProcessorFactory struct {
	NewComponentMember NewProcessorCb
	Config             any
}

type ExporterFactory struct {
	NewComponentMember NewExporterCb
	Config             any
}

func NewConpnentsFactory() *ComponentsFactory {
	return &ComponentsFactory{
		Receivers:  make(map[string]ReceiverFactory),
		Detectors:  make(map[string]DetectorFactory),
		Processors: make(map[string]ProcessorFactory),
		Exporters:  make(map[string]ExporterFactory),
	}
}

func (c *ComponentsFactory) RegisterReceiver(name string, cb NewReceiverCb, config any) {
	c.Receivers[name] = ReceiverFactory{
		NewComponentMember: cb,
		Config:             config,
	}
}

func (c *ComponentsFactory) RegisterDetector(name string, cb NewDetectorCb, config any) {
	c.Detectors[name] = DetectorFactory{
		NewComponentMember: cb,
		Config:             config,
	}
}

func (c *ComponentsFactory) RegisterProcessor(name string, cb NewProcessorCb, config any) {
	c.Processors[name] = ProcessorFactory{
		NewComponentMember: cb,
		Config:             config,
	}
}

func (c *ComponentsFactory) RegisterExporter(name string, cb NewExporterCb, config any) {
	c.Exporters[name] = ExporterFactory{
		NewComponentMember: cb,
		Config:             config,
	}
}
