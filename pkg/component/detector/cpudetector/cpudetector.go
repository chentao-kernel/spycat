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
	cfg          *Config
	cpuPidEvents map[uint32]map[uint32]*model.TimeSegments
	eventChan    chan *model.SpyEvent
	stopChan     chan struct{}
	consumers    []consumer.Consumer

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
		cpuPidEvents:  make(map[uint32]map[uint32]*model.TimeSegments, 10000),
		eventChan:     make(chan *model.SpyEvent, conf.EventChanSize),
		consumers:     consumers,
		stopChan:      make(chan struct{}),
		previousTries: make(map[string][]*trie.Trie),
		tries:         make(map[string][]*trie.Trie),
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

// all cpu tool config will init here, config.ONCPU, config.ONCPU, config.FutexSnoop, etc
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
			// get stack info and upload
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
					// new a trie to delete the old stack info
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
	case model.Syscall:
		dataBlock, err = c.syscallHandler(e)
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

func (c *CpuDetector) formatSyscallLabels(e *model.SpyEvent) (*model.AttributeMap, error) {
	labels := model.NewAttributeMap()
	for i := 0; i < int(e.ParamsCnt); i++ {
		userAttributes := e.UserAttributes[i]
		switch {
		case userAttributes.GetKey() == "dur_ms":
			labels.AddIntValue(model.DurMs, int64(userAttributes.GetUintValue()))
		case userAttributes.GetKey() == "syscall":
			labels.AddStringValue(model.Syscall, string(userAttributes.GetValue()))
		case userAttributes.GetKey() == "stack":
			labels.AddStringValue(model.Stack, string(userAttributes.GetValue()))
		}
	}
	labels.AddIntValue(model.Pid, int64(e.Task.Pid))
	labels.AddStringValue(model.Comm, strings.Replace(string(e.Task.Comm), "\u0000", "", -1))
	return labels, nil
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

func (c *CpuDetector) syscallHandler(e *model.SpyEvent) (*model.DataBlock, error) {
	labels, _ := c.formatSyscallLabels(e)
	val := e.GetUintUserAttribute("dur_ms")
	metric := model.NewIntMetric(model.Syscall, int64(val))
	return model.NewDataBlock(model.Syscall, labels, e.TimeStamp, metric), nil
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
	return []string{model.OffCpu, model.IrqOff, model.OnCpu, model.FutexSnoop, model.Syscall}
}
