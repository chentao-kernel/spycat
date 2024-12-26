package sqlitexporter

import "time"

type Config struct {
	Path            string
	BatchWait       time.Duration
	BatchEntriesNum int
}

func NewConfig() *Config {
	return &Config{
		Path:            "/tmp/spycat/spycat.db",
		BatchWait:       10 * time.Second,
		BatchEntriesNum: 1000,
	}
}
