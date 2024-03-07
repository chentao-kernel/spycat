package detector

import (
	"fmt"

	"github.com/chentao-kernel/spycat/pkg/log"
)

type DetectorManager struct {
	detectors        []Detector
	eventDetectorMap map[string][]Detector
}

func NewDetectorManager(detectors ...Detector) (*DetectorManager, error) {
	if len(detectors) == 0 {
		return nil, fmt.Errorf("detectors is nil")
	}
	detectorMap := make(map[string][]Detector)
	for _, detector := range detectors {
		// detecotr owned events
		events := detector.OwnedEvents()
		for _, event := range events {
			mapDetectors, ok := detectorMap[event]
			if !ok {
				mapDetectors = make([]Detector, 0)
			}
			// normally, one event to one detector
			mapDetectors = append(mapDetectors, detector)
			detectorMap[event] = mapDetectors
		}
		// detector:detector_cpu, owned events:[offcpu, irqoff oncpu]
		log.Loger.Info("detector:%s, owned events:%s\n", detector.Name(), events)
	}

	return &DetectorManager{
		detectors:        detectors,
		eventDetectorMap: detectorMap,
	}, nil
}

func (d *DetectorManager) InitAllDetectors(cfg any) error {
	for _, detector := range d.detectors {
		log.Loger.Info("init detector:%s", detector.Name())
		if err := detector.Init(cfg); err != nil {
			return err
		}
	}
	return nil
}

func (d *DetectorManager) StartAllDetectors() error {
	for _, detector := range d.detectors {
		log.Loger.Info("start detector:%s", detector.Name())
		if err := detector.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (d *DetectorManager) StopAllDetectors() error {
	for _, detector := range d.detectors {
		log.Loger.Info("stop detector:%s", detector.Name())
		if err := detector.Stop(); err != nil {
			return err
		}
	}
	return nil
}

func (d *DetectorManager) GetDetectors(name string) []Detector {
	detecotrs, ok := d.eventDetectorMap[name]
	if ok {
		return detecotrs
	}
	return nil
}
