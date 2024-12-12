package spyexporter

type Config struct {
	OutPuter     string
	BaseFilePath string
}

type FileConfig struct {
	fileName string
}

func NewConfig() *Config {
	return &Config{
		OutPuter:     "FileOutputer",
		BaseFilePath: "/var/log/spycat",
	}
}
