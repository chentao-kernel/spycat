package lokiexporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/chentao-kernel/spycat/pkg/core/model"
	"github.com/chentao-kernel/spycat/pkg/log"
)

// https://github.com/afiskon/promtail-client/blob/master/promtail/common.go#L45
type LogEntry struct {
	Ts   time.Time `json:"ts"`
	Line string    `json:"line"`
}

type LogStream struct {
	Labels  string      `json:"labels"`
	Entries []*LogEntry `json:"entries"`
}

type LogMsg struct {
	Streams []LogStream `json:"streams"`
}

type LokiExporter struct {
	name      string
	config    *Config
	quit      chan struct{}
	waitGroup sync.WaitGroup
	client    http.Client
	entries   chan *LogEntry // entry buffer
}

func NewLokiExporter(cfg interface{}) *LokiExporter {
	config, _ := cfg.(*Config)
	client := &LokiExporter{
		name:    "loki_exporter",
		config:  config,
		quit:    make(chan struct{}),
		entries: make(chan *LogEntry, 5000),
		client:  http.Client{},
	}
	// wait grountine
	client.waitGroup.Add(1)
	go client.Run()
	return client
}

func (l *LokiExporter) write(content string) {
	l.entries <- &LogEntry{
		Ts:   time.Now(),
		Line: content,
	}
}

func (l *LokiExporter) Flush(entries []*LogEntry) {
	var streams []LogStream

	streams = append(streams, LogStream{
		Labels:  l.config.Labels,
		Entries: entries,
	})

	msg := LogMsg{Streams: streams}
	jmsg, err := json.Marshal(msg)
	if err != nil {
		log.Loger.Error("loki exporter unable to marshal:%v", err)
		return
	}

	resp, body, err := l.flushJson("POST", l.config.Url, "application/json", jmsg)
	if err != nil {
		log.Loger.Error("loki exporter log flush failed:%v", err)
		return
	}
	if resp.StatusCode != 204 {
		log.Loger.Error("loki exporter unexpected http status code:%d, msg:%s\n", resp.StatusCode, body)
		return
	}
}

func (l *LokiExporter) Run() {
	var batch []*LogEntry
	batchSize := 0
	maxWait := time.NewTimer(l.config.BatchWait)

	defer func() {
		if batchSize > 0 {
			l.Flush(batch)
		}
		l.waitGroup.Done()
		maxWait.Stop()
	}()

	for {
		select {
		case <-l.quit:
			return
		case entry := <-l.entries:
			batch = append(batch, entry)
			batchSize++
			if batchSize > l.config.BatchEntriesNum {
				l.Flush(batch)
				// init again
				batch = []*LogEntry{}
				batchSize = 0
				maxWait.Reset(l.config.BatchWait)
			}
		case <-maxWait.C:
			if batchSize > 0 {
				l.Flush(batch)
				batch = []*LogEntry{}
				batchSize = 0
			}
			maxWait.Reset(l.config.BatchWait)
		}
	}
}

func (l *LokiExporter) ShutDown() {
	close(l.quit)
	l.waitGroup.Wait()
}

func (l *LokiExporter) flushJson(method string, url string, ctype string, reqBody []byte) (resp *http.Response, resBody []byte, err error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", ctype)

	resp, err = l.client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	resBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	return resp, resBody, nil
}

// common api
func (l *LokiExporter) Consume(data *model.DataBlock) error {
	jsondata, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("loki marshal data failed:%v", err)
	}
	l.write(string(jsondata))
	return nil
}

func (l *LokiExporter) Name() string {
	return l.name
}
