package lokiexporter

import (
	"os"
	"time"
)

type Config struct {
	Url             string
	Labels          string
	BatchWait       time.Duration
	BatchEntriesNum int
}

func NewConfig() *Config {
	host, _ := os.Hostname()
	labels := "{host=" + host + "}"
	return &Config{
		Url:             "http://localhost:3100/api/prom/push",
		Labels:          labels,
		BatchWait:       5 * time.Second,
		BatchEntriesNum: 10000,
	}
}
