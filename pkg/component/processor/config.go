package processor

type Config struct {
	TickerInterval   int                               `mapstructure:"ticker_interval"`
	SessionInfos     []Session                         `mapstructure:"sessioninfo"`
	AggregateKindMap map[string][]AggregatedKindConfig `mapstructure:"aggregate_kind_map"`
}

type AggregatedKindConfig struct {
	OutputName         string  `mapstructure:"output_name"`
	Kind               string  `mapstructure:"kind"`
	ExplicitBoundaries []int64 `mapstructure:"explicit_boundaries"`
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
	}
}
