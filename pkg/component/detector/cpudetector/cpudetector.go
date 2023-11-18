package cpudetector

import (
	"strings"
	"sync"
	"time"

	"github.com/chentao-kernel/spycat/pkg/component/consumer"
	"github.com/chentao-kernel/spycat/pkg/component/detector"
	"github.com/chentao-kernel/spycat/pkg/core/model"
	"github.com/chentao-kernel/spycat/pkg/log"
)

const (
	DetectorCpuType string = "detector_cpu"
)

var nanoToSeconds uint64 = 1e9
var enableProfile bool = true

type CpuDetector struct {
	cfg             *Config
	cpuPidEvents    map[uint32]map[uint32]*model.TimeSegments
	eventChan       chan *model.SpyEvent
	stopChan        chan struct{}
	consumers       []consumer.Consumer
	tidExpiredQueue *tidDeleteQueue
	lock            sync.RWMutex
}

func NewCpuDetector(cfg any, consumers []consumer.Consumer) detector.Detector {
	config, _ := cfg.(*Config)

	cd := &CpuDetector{
		cfg:             config,
		cpuPidEvents:    make(map[uint32]map[uint32]*model.TimeSegments, 10000),
		eventChan:       make(chan *model.SpyEvent, config.EventChanSize),
		consumers:       consumers,
		tidExpiredQueue: newTidDeleteQueue(),
		stopChan:        make(chan struct{}),
	}

	return cd
}

func (c *CpuDetector) sendToConsumers() {

}

func (c *CpuDetector) Start() error {
	go c.ConsumeChanEvents()
	//go c.TidRemove(30*time.Second, 10*time.Second)
	return nil
}

func (c *CpuDetector) Stop() error {
	close(c.stopChan)
	return nil
}

func (c *CpuDetector) Name() string {
	return DetectorCpuType
}

// send to next consumer
func (c *CpuDetector) ProcessEvent(e *model.SpyEvent) error {
	var dataBlock *model.DataBlock
	var err error
	switch e.Name {
	case model.OffCpu:
		dataBlock, err = c.offcpuHandler(e)
	case model.OnCpu:
		dataBlock, err = c.oncpuHandler(e)
	case model.IrqOff:
		dataBlock, err = c.irqoffHandler(e)
	default:
		return nil
	}
	if err != nil {
		return err
	}

	if dataBlock == nil {
		return nil
	}
	// next consumer is default processor
	for _, con := range c.consumers {
		err := con.Consume(dataBlock)
		if err != nil {
			log.Loger.Error("consumer consume event failed:%v", err)
			return err
		}
	}
	return nil
}

func (c *CpuDetector) ConsumeChanEvents() {
	for {
		select {
		case e := <-c.eventChan:
			err := c.ProcessEvent(e)
			if err != nil {
				log.Loger.Error("cpu detector process event failed:%v", err)
			}
		case <-c.stopChan:
			log.Loger.Info("detector event channel stoped.")
			return
		}
	}
}

func (c *CpuDetector) ConsumeCommEvent(e *model.SpyEvent) error {
	return nil
}

func (c *CpuDetector) irqoffHandler(e *model.SpyEvent) (*model.DataBlock, error) {
	return nil, nil
}

func (c *CpuDetector) formatOffcpuLabels(e *model.SpyEvent) (*model.AttributeMap, error) {
	labels := model.NewAttributeMap()
	for i := 0; i < int(e.ParamsCnt); i++ {
		userAttributes := e.UserAttributes[i]
		switch {
		case userAttributes.GetKey() == "t_start_time":
			labels.AddIntValue(model.StartTime, int64(userAttributes.GetUintValue()))
		case userAttributes.GetKey() == "t_end_time":
			labels.AddIntValue(model.EndTime, int64(userAttributes.GetUintValue()))
		case userAttributes.GetKey() == "ts":
			labels.AddIntValue(model.TimeStamp, int64(userAttributes.GetUintValue()))
		case userAttributes.GetKey() == "w_comm":
			labels.AddStringValue(model.Waker, strings.Replace(string(userAttributes.GetValue()), "\u0000", "", -1))
		case userAttributes.GetKey() == "w_pid":
			labels.AddIntValue(model.Pid_W, int64(userAttributes.GetUintValue()))
		case userAttributes.GetKey() == "w_stack":
			labels.AddStringValue(model.Stack_W, string(userAttributes.GetValue()))
		case userAttributes.GetKey() == "w_onrq_oncpu":
			labels.AddIntValue(model.RunqLatUs_W, int64(userAttributes.GetUintValue()/1000))
		case userAttributes.GetKey() == "w_offcpu_oncpu":
			labels.AddIntValue(model.CpuOffUs_W, int64(userAttributes.GetUintValue()/1000))
		case userAttributes.GetKey() == "wt_comm":
			labels.AddStringValue(model.Target, strings.Replace(string(userAttributes.GetValue()), "\u0000", "", -1))
		case userAttributes.GetKey() == "wt_pid":
			labels.AddIntValue(model.Pid_T, int64(userAttributes.GetUintValue()))
		case userAttributes.GetKey() == "t_stack":
			labels.AddStringValue(model.Stack_T, string(userAttributes.GetValue()))
		case userAttributes.GetKey() == "t_onrq_oncpu":
			labels.AddIntValue(model.RunqLatUs_T, int64(userAttributes.GetUintValue()/1000))
		case userAttributes.GetKey() == "t_offcpu_oncpu":
			labels.AddIntValue(model.CpuOffUs_T, int64(userAttributes.GetUintValue()/1000))
		}
	}
	return labels, nil
}

func (c *CpuDetector) offcpuHandler(e *model.SpyEvent) (*model.DataBlock, error) {
	labels, _ := c.formatOffcpuLabels(e)
	//util.PrintStructFields(*ev)
	//c.PutEventToSegments(e.GetPid(), e.GetTid(), e.GetComm(), ev)
	val := e.GetUintUserAttribute("t_offcpu_oncpu")
	metric := model.NewIntMetric(model.OffCpuMetricName, int64(val))
	return model.NewDataBlock(model.OffCpu, labels, e.TimeStamp, metric), nil
}

func (c *CpuDetector) oncpuHandler(e *model.SpyEvent) (*model.DataBlock, error) {
	return nil, nil
}

// called by receiver
func (c *CpuDetector) ConsumeEvent(e *model.SpyEvent) error {
	c.eventChan <- e
	return nil
}

// hard code
func (c *CpuDetector) OwnedEvents() []string {
	return []string{model.OffCpu, model.IrqOff, model.OnCpu}
}

func (c *CpuDetector) PutEventToSegments(pid uint32, tid uint32, threadName string, event model.TimedEvent) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if !enableProfile {
		return
	}
	tidCpuEvents, exist := c.cpuPidEvents[pid]
	if !exist {
		tidCpuEvents = make(map[uint32]*model.TimeSegments)
		c.cpuPidEvents[pid] = tidCpuEvents
	}
	timeSegments, exist := tidCpuEvents[tid]
	// 配置的最大值
	maxSegmentSize := c.cfg.SegmentSize
	if exist {
		endOffset := int(event.EndTimestamp()/nanoToSeconds - timeSegments.BaseTime)
		if endOffset < 0 {
			log.Loger.Debug("endoffset is negative, endtimestamp=%d, basetime=%d", event.EndTimestamp(), timeSegments.BaseTime)
			return
		}
		startOffset := int(event.StartTimestamp()/nanoToSeconds - timeSegments.BaseTime)
		if startOffset < 0 {
			startOffset = 0
		}
		if startOffset >= maxSegmentSize || endOffset > maxSegmentSize {
			if startOffset*2 >= 3*maxSegmentSize {
				log.Loger.Debug("pid=%d,tid=%d,com=%s, reset basetime from %d to %d, startoffset:%d, maxsegmentSize:%d",
					pid, tid, threadName, timeSegments.BaseTime, event.StartTimestamp()/nanoToSeconds, startOffset, maxSegmentSize)
				// clear all elements
				timeSegments.Segments.Clear()
				// basetime 等于开始时间
				timeSegments.BaseTime = event.StartTimestamp() / nanoToSeconds
				endOffset = endOffset - startOffset
				startOffset = 0
				for i := 0; i < maxSegmentSize; i++ {
					segment := model.NewSegment((timeSegments.BaseTime+uint64(i))*nanoToSeconds,
						// basetime 间隔1秒
						(timeSegments.BaseTime+uint64(i+1))*nanoToSeconds)
					timeSegments.Segments.UpdateByIndex(i, segment)
				}
			} else {
				// Clear half of the elements
				clearSize := maxSegmentSize / 2
				log.Loger.Debug("pid=%d,tid=%d,com=%s, reset basetime from %d to %d, startoffset:%d, maxsegmentSize:%d",
					pid, tid, threadName, timeSegments.BaseTime, event.StartTimestamp()/nanoToSeconds, startOffset, maxSegmentSize)

				timeSegments.BaseTime = timeSegments.BaseTime + uint64(clearSize)
				startOffset -= clearSize
				if startOffset < 0 {
					startOffset = 0
				}
				endOffset -= clearSize
				for i := 0; i < clearSize; i++ {
					movedIndex := i + clearSize
					val := timeSegments.Segments.GetByIndex(movedIndex)
					timeSegments.Segments.UpdateByIndex(i, val)
					segmentTmp := model.NewSegment((timeSegments.BaseTime+uint64(movedIndex))*nanoToSeconds,
						(timeSegments.BaseTime+uint64(movedIndex+1))*nanoToSeconds)
					timeSegments.Segments.UpdateByIndex(movedIndex, segmentTmp)
				}
			}
		}
		// Update the thread name immediately
		// startOffset和endOffset差的不大
		timeSegments.UpdateThreadName(threadName)
		for i := startOffset; i <= endOffset && i < maxSegmentSize; i++ {
			val := timeSegments.Segments.GetByIndex(i)
			segment := val.(*model.Segment)
			segment.PutTimedEvent(event)
			segment.IsSend = 0
			timeSegments.Segments.UpdateByIndex(i, segment)
		}

	} else {
		/*
			TimeSegments
				Segments-->CircleQueue
							data[0] 			--> Segment
							data[1] 			--> Segment
							... 				...
							data[maxSegmentSize] --> Segment
		*/
		newTimeSegments := &model.TimeSegments{
			Pid:        pid,
			Tid:        tid,
			ThreadName: threadName,
			// 初始化起始时间
			BaseTime: event.StartTimestamp() / nanoToSeconds,
			Segments: model.NewCircleQueue(maxSegmentSize),
		}
		for i := 0; i < maxSegmentSize; i++ {
			segment := model.NewSegment((newTimeSegments.BaseTime+uint64(i))*nanoToSeconds,
				(newTimeSegments.BaseTime+uint64(i+1))*nanoToSeconds)
			// 将segment放到Segments中
			newTimeSegments.Segments.UpdateByIndex(i, segment)
		}

		endOffset := int(event.EndTimestamp()/nanoToSeconds - newTimeSegments.BaseTime)

		for i := 0; i <= endOffset && i < maxSegmentSize; i++ {
			val := newTimeSegments.Segments.GetByIndex(i)
			segment := val.(*model.Segment)
			segment.PutTimedEvent(event)
			segment.IsSend = 0
		}
		//  一个tid对应一个ringbuffer
		tidCpuEvents[tid] = newTimeSegments
	}
}

// todo: 增加添加tidExpiredQueue的逻辑
func (c *CpuDetector) TidRemove(interval time.Duration, expiredDuration time.Duration) {
	for {
		select {
		case <-c.stopChan:
			return
		case <-time.After(interval):
			now := time.Now()
			func() {
				c.tidExpiredQueue.queueMutex.Lock()
				defer c.tidExpiredQueue.queueMutex.Unlock()
				for {
					// 取出队列中的所有元素, 如果过期了就删除掉
					elem := c.tidExpiredQueue.GetFront()
					if elem == nil {
						break
					}
					if elem.exitTime.Add(expiredDuration).After(now) {
						break
					}
					//Delete expired threads (current_time >= thread_exit_time + interval_time).
					func() {
						// 获取map的锁
						c.lock.Lock()
						defer c.lock.Unlock()
						tidEventsMap := c.cpuPidEvents[elem.pid]
						if tidEventsMap == nil {
							c.tidExpiredQueue.Pop()
						} else {
							log.Loger.Debug("delete expired thread,pid=%d,tid=%d", elem.pid, elem.tid)
							delete(tidEventsMap, elem.tid)
							c.tidExpiredQueue.Pop()
						}
					}()
				}
			}()
		}
	}

}
