package cpudetector

type Config struct {
	EventName     string
	SessionInfos  []SubSession `mapstructure:"sessioninfo"`
	EventChanSize int
	SegmentSize   int `mapstrucrure:"segmentSize"`
}

type SubSession struct {
	Class  string            `mapstructure:"class"`
	Name   string            `mapstructure:"name"`
	Params map[string]string `mapstructure:"params"`
}

func NewConfig() *Config {
	return &Config{
		EventChanSize: 10000,
		SegmentSize:   40,
	}
}
