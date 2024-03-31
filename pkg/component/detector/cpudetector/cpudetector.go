package cpudetector

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chentao-kernel/spycat/pkg/app/config"
	"github.com/chentao-kernel/spycat/pkg/component/consumer"
	"github.com/chentao-kernel/spycat/pkg/component/detector"
	"github.com/chentao-kernel/spycat/pkg/component/detector/cpudetector/upstream"
	"github.com/chentao-kernel/spycat/pkg/component/detector/cpudetector/upstream/remote"
	"github.com/chentao-kernel/spycat/pkg/core/model"
	"github.com/chentao-kernel/spycat/pkg/log"
	"github.com/chentao-kernel/spycat/pkg/util/alignedticker"
	"github.com/chentao-kernel/spycat/pkg/util/trie"

	"github.com/pyroscope-io/pyroscope/pkg/storage/metadata"
	"github.com/pyroscope-io/pyroscope/pkg/storage/segment"
)

const (
	DetectorCpuType string = "detector_cpu"
)

var nanoToSeconds uint64 = 1e9
var enableProfile = true

type CpuDetector struct {
	cfg             *Config
	cpuPidEvents    map[uint32]map[uint32]*model.TimeSegments
	eventChan       chan *model.SpyEvent
	stopChan        chan struct{}
	consumers       []consumer.Consumer
	tidExpiredQueue *tidDeleteQueue

	lock sync.RWMutex

	trieLock           sync.RWMutex
	previousTries      map[string][]*trie.Trie
	tries              map[string][]*trie.Trie
	upstream           upstream.Upstream
	startTimeTruncated time.Time
	uploadRate         time.Duration
	sampleRate         uint32
	Server             string
}

func NewCpuDetector(cfg any, consumers []consumer.Consumer) detector.Detector {
	conf, _ := cfg.(*Config)

	cd := &CpuDetector{
		cfg: conf,
		// noused now
		cpuPidEvents:    make(map[uint32]map[uint32]*model.TimeSegments, 10000),
		eventChan:       make(chan *model.SpyEvent, conf.EventChanSize),
		consumers:       consumers,
		tidExpiredQueue: newTidDeleteQueue(),
		stopChan:        make(chan struct{}),
		previousTries:   make(map[string][]*trie.Trie),
		tries:           make(map[string][]*trie.Trie),
	}
	// only oncpu use
	cd.initializeTries(model.OnCpu)
	cd.uploadRate = 10 * time.Second
	return cd
}

func (c *CpuDetector) sendToConsumers() {

}

func (c *CpuDetector) initializeTries(appName string) {
	if _, ok := c.previousTries[appName]; !ok {
		// TODO Only set the trie if it's not already set
		c.previousTries[appName] = []*trie.Trie{}
		c.tries[appName] = []*trie.Trie{}

		c.previousTries[appName] = append(c.previousTries[appName], nil)
		c.tries[appName] = append(c.tries[appName], trie.New())
	}
}

func (c *CpuDetector) Init(cfg any) error {
	conf, ok := cfg.(*config.ONCPU)
	if ok {
		// init upstream parameter
		rc := remote.RemoteConfig{
			AuthToken:              "", // cfg.AuthToken,
			UpstreamThreads:        conf.UploadThreads,
			UpstreamAddress:        conf.Server,
			UpstreamRequestTimeout: conf.UploadTimeout, // add timeout
		}
		up, err := remote.New(rc)
		if err != nil {
			return fmt.Errorf("new remote upstream: %v", err)
		}
		c.upstream = up
		c.upstream.Start()

		go c.UploadWithTicker()
	}

	return nil
}

func (c *CpuDetector) Start() error {
	// util.PrintStack() for debug
	go c.ConsumeChanEvents()

	return nil
}

func (c *CpuDetector) Stop() error {
	close(c.stopChan)
	c.upstream.Stop()
	return nil
}

func (c *CpuDetector) Name() string {
	return DetectorCpuType
}

func addSuffix(name string, ptype string) (string, error) {
	k, err := segment.ParseKey(name)
	if err != nil {
		return "", err
	}
	k.Add("__name__", k.AppName()+"."+string(ptype))
	return k.Normalized(), nil
}

func (c *CpuDetector) UploadWithTicker() {
	uploadTicker := alignedticker.NewAlignedTicker(c.uploadRate)
	defer uploadTicker.Stop()
	for {
		select {
		case endTimeTruncated := <-uploadTicker.C:
			// 获取栈信息并上传
			c.UploadTries(endTimeTruncated)
		case <-c.stopChan:
			c.Stop()
			return
		}
	}
}

func (c *CpuDetector) UploadTries(endTimeTruncated time.Time) {
	c.trieLock.Lock()
	defer c.trieLock.Unlock()

	if !c.startTimeTruncated.IsZero() {
		// name is app-name
		for name, tarr := range c.tries {
			for i, ti := range tarr {
				if ti != nil {
					endTime := endTimeTruncated
					startTime := endTime.Add(-c.uploadRate)
					uploadTrie := ti
					if !uploadTrie.IsEmpty() {
						nameWithSuffix, _ := addSuffix("aa", "cpu")
						c.upstream.Upload(&upstream.UploadJob{
							Name:            nameWithSuffix,
							StartTime:       startTime,
							EndTime:         endTime,
							SpyName:         "ebpfspy",
							SampleRate:      c.sampleRate,
							Units:           metadata.SamplesUnits,
							AggregationType: metadata.SumAggregationType,
							Trie:            uploadTrie,
						})
					}
					// 新建一个trie树，删除掉历史栈信息
					c.tries[name][i] = trie.New()
				}
			}
		}
	}
	c.startTimeTruncated = endTimeTruncated
}

// send to next consumer
func (c *CpuDetector) ProcessEvent(e *model.SpyEvent) error {
	var dataBlock *model.DataBlock
	var err error
	switch e.Name {
	case model.OffCpu:
		dataBlock, err = c.offcpuHandler(e)
	case model.OnCpu:
		dataBlock, err = c.oncpuHandler(e, func(appName string, stack []byte, v uint64) error {
			if len(stack) > 0 {
				if _, ok := c.tries[appName]; !ok {
					c.initializeTries(appName)
				}
				// insert the stackinfo into trie tree
				c.tries[appName][0].Insert(stack, v, true)
				// fmt.Printf("tao name:%s, stack:%s, count:%d, sample:%d\n", appName, string(stack), v, c.sampleRate)
			}
			return nil
		})
	case model.IrqOff:
		dataBlock, err = c.irqoffHandler(e)
	case model.FutexSnoop:
		dataBlock, err = c.futexsnoopHandler(e)
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

func (c *CpuDetector) formatFutexSnoopLabels(e *model.SpyEvent) (*model.AttributeMap, error) {
	labels := model.NewAttributeMap()
	for i := 0; i < int(e.ParamsCnt); i++ {
		userAttributes := e.UserAttributes[i]
		switch {
		case userAttributes.GetKey() == "user_cnt":
			labels.AddIntValue(model.UserCnt, int64(userAttributes.GetUintValue()))
		case userAttributes.GetKey() == "max_user_cnt":
			labels.AddIntValue(model.MaxUserCnt, int64(userAttributes.GetUintValue()))
		case userAttributes.GetKey() == "lock_addr":
			labels.AddIntValue(model.LockAddr, int64(userAttributes.GetUintValue()))
		case userAttributes.GetKey() == "min_dur":
			labels.AddIntValue(model.MinDur, int64(userAttributes.GetUintValue()))
		case userAttributes.GetKey() == "max_dur":
			labels.AddIntValue(model.MaxDur, int64(userAttributes.GetUintValue()))
		case userAttributes.GetKey() == "delta_dur":
			labels.AddIntValue(model.DeltaDur, int64(userAttributes.GetUintValue()))
		case userAttributes.GetKey() == "avg_dur":
			labels.AddIntValue(model.AvgDur, int64(userAttributes.GetUintValue()))
		case userAttributes.GetKey() == "lock_cnt":
			labels.AddIntValue(model.LockCnt, int64(userAttributes.GetUintValue()))
		case userAttributes.GetKey() == "stack":
			labels.AddStringValue(model.Stack, string(userAttributes.GetValue()))
		}
	}
	return labels, nil
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
		case userAttributes.GetKey() == "w_tgid":
			labels.AddIntValue(model.Tgid_W, int64(userAttributes.GetUintValue()))
		case userAttributes.GetKey() == "w_stack":
			labels.AddStringValue(model.Stack_W, string(userAttributes.GetValue()))
		case userAttributes.GetKey() == "w_onrq_oncpu":
			labels.AddIntValue(model.RunqLatUs_W, int64(userAttributes.GetUintValue()/1000))
		case userAttributes.GetKey() == "w_offcpu_oncpu":
			labels.AddIntValue(model.CpuOffUs_W, int64(userAttributes.GetUintValue()/1000))

		case userAttributes.GetKey() == "wt_comm":
			labels.AddStringValue(model.WTarget, strings.Replace(string(userAttributes.GetValue()), "\u0000", "", -1))
		case userAttributes.GetKey() == "wt_pid":
			labels.AddIntValue(model.Pid_WT, int64(userAttributes.GetUintValue()))

		case userAttributes.GetKey() == "t_comm":
			labels.AddStringValue(model.Target, strings.Replace(string(userAttributes.GetValue()), "\u0000", "", -1))
		case userAttributes.GetKey() == "t_pid":
			labels.AddIntValue(model.Tgid_W, int64(userAttributes.GetUintValue()))
		case userAttributes.GetKey() == "t_tgid":
			labels.AddIntValue(model.Tgid_T, int64(userAttributes.GetUintValue()))
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

func (c *CpuDetector) futexsnoopHandler(e *model.SpyEvent) (*model.DataBlock, error) {
	labels, _ := c.formatFutexSnoopLabels(e)
	val := e.GetUintUserAttribute("max_user_cnt")
	metric := model.NewIntMetric(model.FutexMaxUerCountName, int64(val))
	return model.NewDataBlock(model.FutexSnoop, labels, e.TimeStamp, metric), nil
}

func (c *CpuDetector) offcpuHandler(e *model.SpyEvent) (*model.DataBlock, error) {
	labels, _ := c.formatOffcpuLabels(e)
	// util.PrintStructFields(*ev)
	// c.PutEventToSegments(e.GetPid(), e.GetTid(), e.GetComm(), ev)
	val := e.GetUintUserAttribute("t_offcpu_oncpu")
	metric := model.NewIntMetric(model.OffCpuMetricName, int64(val))
	return model.NewDataBlock(model.OffCpu, labels, e.TimeStamp, metric), nil
}

func (c *CpuDetector) oncpuHandler(e *model.SpyEvent, cb func(string, []byte, uint64) error) (*model.DataBlock, error) {
	var count uint64
	var stack []byte

	c.trieLock.Lock()
	defer c.trieLock.Unlock()
	for i := 0; i < int(e.ParamsCnt); i++ {
		userAttributes := e.UserAttributes[i]
		switch {
		case userAttributes.GetKey() == "count":
			count = userAttributes.GetUintValue()
		case userAttributes.GetKey() == "stack":
			stack = userAttributes.GetValue()
		case userAttributes.GetKey() == "sampleRate":
			c.sampleRate = uint32(userAttributes.GetUintValue())
		}
	}
	cb(e.Name, stack, count)
	// fmt.Printf("pid:%d, comm:%s, count:%d, stack:%s", pid, comm, count, stack)
	return nil, nil
}

// called by receiver
func (c *CpuDetector) ConsumeEvent(e *model.SpyEvent) error {
	c.eventChan <- e
	return nil
}

// Notice: hard code, new tool added update this function
func (c *CpuDetector) OwnedEvents() []string {
	return []string{model.OffCpu, model.IrqOff, model.OnCpu, model.FutexSnoop}
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
					seg := model.NewSegment((timeSegments.BaseTime+uint64(i))*nanoToSeconds,
						// basetime 间隔1秒
						(timeSegments.BaseTime+uint64(i+1))*nanoToSeconds)
					timeSegments.Segments.UpdateByIndex(i, seg)
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
			seg := val.(*model.Segment)
			seg.PutTimedEvent(event)
			seg.IsSend = 0
			timeSegments.Segments.UpdateByIndex(i, seg)
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
			seg := model.NewSegment((newTimeSegments.BaseTime+uint64(i))*nanoToSeconds,
				(newTimeSegments.BaseTime+uint64(i+1))*nanoToSeconds)
			// 将segment放到Segments中
			newTimeSegments.Segments.UpdateByIndex(i, seg)
		}

		endOffset := int(event.EndTimestamp()/nanoToSeconds - newTimeSegments.BaseTime)

		for i := 0; i <= endOffset && i < maxSegmentSize; i++ {
			val := newTimeSegments.Segments.GetByIndex(i)
			seg := val.(*model.Segment)
			seg.PutTimedEvent(event)
			seg.IsSend = 0
		}
		// tid corresponds to ringbuffer one by one
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
					// Delete expired threads (current_time >= thread_exit_time + interval_time).
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
