package netdetector

type Config struct {
	SessionInfos  []Session `mapstructure:"sessioninfo"`
	EventChanSize int
}

type Session struct {
	Class  string            `mapstructure:"class"`
	Name   string            `mapstructure:"name"`
	Params map[string]string `mapstructure:"params"`
}

func NewConfig() *Config {
	return &Config{
		SessionInfos: []Session{
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
