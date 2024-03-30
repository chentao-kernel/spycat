package receiver

import (
	"sync"

	"github.com/chentao-kernel/spycat/pkg/component/detector"
	"github.com/chentao-kernel/spycat/pkg/core/model"
	"github.com/chentao-kernel/spycat/pkg/log"
)

const (
	ReceiverCiliumType = "receiver_cilium"
)

type Receiver interface {
	Start() error
	Stop() error
	RcvChan() chan *model.SpyEvent
}

type CiliumReceiver struct {
	cfg             *Config
	detectorManager *detector.DetectorManager
	stopCh          chan struct{}
	stopWG          sync.WaitGroup
	// data from cilium bpf to detector
	eventCh chan *model.SpyEvent
	stats   eventstat
}

func EbpfRecevierHandler() *model.SpyEvent {
	return &model.SpyEvent{}
}

func NewCiliumReceiver(config any, detectorManager *detector.DetectorManager) Receiver {
	cfg, ok := config.(*Config)
	if !ok {
		log.Loger.Error("Convert %v config failed", cfg)
		return nil
	}
	ciliumReceiver := &CiliumReceiver{
		cfg:             cfg,
		detectorManager: detectorManager,
		stopCh:          make(chan struct{}, 1),
		eventCh:         make(chan *model.SpyEvent, 2e5),
	}
	ciliumReceiver.stats = NewEventStats(cfg.SessionInfos)
	return ciliumReceiver
}

func (c *CiliumReceiver) Start() error {
	log.Loger.Info("Cilium Receiver Start!")

	go c.consumeEvents()
	//go c.receiveEvents()

	return nil
}

func (c *CiliumReceiver) RcvChan() chan *model.SpyEvent {
	return c.eventCh
}

// Receive events from cilium bpf framework
func (c *CiliumReceiver) receiveEvents() {
	c.stopWG.Add(1)
	for {
		select {
		case <-c.stopCh:
			c.stopWG.Done()
			return
			// receive from cilium
		default:
			// todo receive real data
			event := &model.SpyEvent{}
			c.eventCh <- event
			c.stats.Add(event.Class.Name, 1)
		}
	}
}

// data entry for all ebpf events
func (c *CiliumReceiver) consumeEvents() {
	c.stopWG.Add(1)
	for {
		select {
		case <-c.stopCh:
			c.stopWG.Done()
			return
		case e := <-c.eventCh:
			err := c.sendToConsumers(e)
			if err != nil {
				log.Loger.Error("Failed to send events to next consumer:%v", err)
			}
		}
	}
}

func (c *CiliumReceiver) sendToConsumers(e *model.SpyEvent) error {
	// Class.Name no used
	// GetDetectors from OwnedEvents
	detectors := c.detectorManager.GetDetectors(e.Class.Event)
	if len(detectors) == 0 {
		return nil
	}
	for _, detector := range detectors {
		if err := detector.ConsumeEvent(e); err != nil {
			log.Loger.Warn("failed to send event:%v", err)
		}
	}
	return nil
}

func (c *CiliumReceiver) Stop() error {
	close(c.stopCh)
	c.stopWG.Wait()
	return nil
}
