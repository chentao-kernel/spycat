package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

var (
	configPath   = "spycat.yaml"
	ConfigGlobal *Config
)

type Config struct {
	Log struct {
		Path  string `yaml:"Path"`
		Level string `yaml:"Level"`
	} `yaml:"log"`
	Inputs []struct {
		Type   string   `yaml:"Type"`
		Common struct{} `yaml:"Common,omitempty"`
	} `yaml:"inputs"`
	Processors []struct {
		Type   string   `yaml:"Type"`
		Common struct{} `yaml:"Common,omitempty"`
	} `yaml:"processors"`
	Exporters []struct {
		Type   string   `yaml:"Type"`
		Common struct{} `yaml:"Common,omitempty"`
	} `yaml:"exporters"`
}

var defaultConfig = Config{
	Log: struct {
		Path  string `yaml:"Path"`
		Level string `yaml:"Level"`
	}{
		Path:  "/tmp/spycat/",
		Level: "INFO",
	},
	Inputs: []struct {
		Type   string   `yaml:"Type"`
		Common struct{} `yaml:"Common,omitempty"`
	}{
		{Type: "default"},
	},
	Processors: []struct {
		Type   string   `yaml:"Type"`
		Common struct{} `yaml:"Common,omitempty"`
	}{
		{Type: "processor_default"},
	},
	Exporters: []struct {
		Type   string   `yaml:"Type"`
		Common struct{} `yaml:"Common,omitempty"`
	}{
		{Type: "exporter_stdout"},
	},
}

func NewConfig() *Config {
	return &Config{
		Log:        defaultConfig.Log,
		Inputs:     defaultConfig.Inputs,
		Processors: defaultConfig.Processors,
		Exporters:  defaultConfig.Exporters,
	}
}

func ConfigInit() error {
	ConfigGlobal = NewConfig()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		data, err := yaml.Marshal(defaultConfig)
		if err != nil {
			return fmt.Errorf("yaml marshal failed:%v", err)
		}
		err = os.WriteFile(configPath, data, 0644)
		if err != nil {
			return fmt.Errorf("yaml write failed:%v", err)
		}
	} else {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("yaml read failed:%v", err)
		}
		err = yaml.Unmarshal(data, ConfigGlobal)
		if err != nil {
			return fmt.Errorf("yaml unmarshal failed:%v", err)
		}
	}
	return nil
}

func GetConfig() *Config {
	return ConfigGlobal
}

func (c *Config) ConfigLog() {

}

func (c *Config) ConfigInputs() {

}

func (c *Config) ConfigProcessors() {

}

func (c *Config) ConfigExporters() {

}
