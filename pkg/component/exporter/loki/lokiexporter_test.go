package lokiexporter

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestLokiClient(t *testing.T) {
	host, _ := os.Hostname()
	labels := "{tao=\"" + host + "\",job=\"" + host + "\"}"

	conf := &Config{
		Url:             "http://localhost:3100/api/prom/push",
		Labels:          labels,
		BatchWait:       5 * time.Second,
		BatchEntriesNum: 10000,
	}
	client := NewLokiExporter(conf)
	for i := 0; i < 1000; i++ {
		client.Write(fmt.Sprintf("value=%d", i))
	}
	time.Sleep(10 * time.Second)
	client.ShutDown()
}

func nameIsValid(name string) bool {
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			(c == '-') || (c == '_')) {
			return false
		}
	}
	return true
}
