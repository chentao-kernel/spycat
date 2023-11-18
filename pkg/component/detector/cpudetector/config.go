package cpudetector

type Config struct {
	SessionInfos  []SubSession `mapstructure:"sessioninfo"`
	EventChanSize int
}

type SubSession struct {
	Class  string            `mapstructure:"class"`
	Name   string            `mapstructure:"name"`
	Params map[string]string `mapstructure:"params"`
}

func NewConfig() *Config {
	return &Config{
		SessionInfos: []SubSession{
			{
				Name:  "net",
				Class: "net",
			},
			{
				Name:  "io",
				Class: "io",
			},
			{
				Name:  "cpu",
				Class: "cpu",
			},
			{
				Name:  "mem",
				Class: "mem",
			},
		},
		EventChanSize: 1000,
	}
}
