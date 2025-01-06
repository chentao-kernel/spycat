package config

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	appspy "github.com/chentao-kernel/spycat/internal/app"
	"github.com/chentao-kernel/spycat/pkg/app/config"
	"github.com/chentao-kernel/spycat/pkg/core"
	"github.com/chentao-kernel/spycat/pkg/core/model"
	"github.com/chentao-kernel/spycat/pkg/ebpf/cpu"
	"github.com/chentao-kernel/spycat/pkg/ebpf/mem"
	"github.com/chentao-kernel/spycat/pkg/util"
	"gopkg.in/yaml.v3"
)

var (
	configPath       = "spycat.yaml"
	YamlConfigGlobal *Config
	ENABLE           = "enable"
	DISENABLE        = "disable"
)

type ToolsConfig struct {
	Tools      []string // enable tools list
	Futexsnoop config.FUTEXSNOOP_YAML
	Offcpu     config.OFFCPU_YAML
	Syscall    config.SYSCALL_YAML
	Oncpu      config.ONCPU_YAML
	Cachestat  config.CACHESTAT_YAML
}

var ToolsConfigGlobal = ToolsConfig{}

func initToolsConfig(toolsConfig *ToolsConfig, cfg *Config) {
	for _, cf := range cfg.Inputs {
		switch cf.Type {
		case model.OffCpu:
			val, ok := cf.Config.(*config.OFFCPU_YAML)
			if ok {
				toolsConfig.Offcpu = *val
				if val.Status == ENABLE {
					toolsConfig.Tools = append(toolsConfig.Tools, model.OffCpu)
				}
			}
		case model.OnCpu:
			val, ok := cf.Config.(*config.ONCPU_YAML)
			if ok {
				toolsConfig.Oncpu = *val
				if val.Status == ENABLE {
					toolsConfig.Tools = append(toolsConfig.Tools, model.OnCpu)
				}
			}
		case model.CacheStat:
			val, ok := cf.Config.(*config.CACHESTAT_YAML)
			if ok {
				toolsConfig.Cachestat = *val
				if val.Status == ENABLE {
					toolsConfig.Tools = append(toolsConfig.Tools, model.CacheStat)
				}
			}
		case model.Syscall:
			val, ok := cf.Config.(*config.SYSCALL_YAML)
			if ok {
				toolsConfig.Syscall = *val
				if val.Status == ENABLE {
					toolsConfig.Tools = append(toolsConfig.Tools, model.Syscall)
				}
			}
		case model.FutexSnoop:
			val, ok := cf.Config.(*config.FUTEXSNOOP_YAML)
			if ok {
				toolsConfig.Futexsnoop = *val
				if val.Status == ENABLE {
					toolsConfig.Tools = append(toolsConfig.Tools, model.FutexSnoop)
				}
			}
		}
	}
}

func (i *InputConfig) UnmarshalYAML(value *yaml.Node) error {
	// the raw struct layout corresponds to InputConfig
	var raw struct {
		Type   string    `yaml:"type"`
		Config yaml.Node `yaml:"config"`
	}

	if err := value.Decode(&raw); err != nil {
		return err
	}

	i.Type = raw.Type
	switch i.Type {
	case "futexsnoop":
		var cfg config.FUTEXSNOOP_YAML
		if err := raw.Config.Decode(&cfg); err != nil {
			return err
		}
		i.Config = &cfg
	case "offcpu":
		var cfg config.OFFCPU_YAML
		if err := raw.Config.Decode(&cfg); err != nil {
			return err
		}

		i.Config = &cfg
	case "syscall":
		var cfg config.SYSCALL_YAML
		if err := raw.Config.Decode(&cfg); err != nil {
			return err
		}
		i.Config = &cfg
	case "oncpu":
		var cfg config.ONCPU_YAML
		if err := raw.Config.Decode(&cfg); err != nil {
			return err
		}
		i.Config = &cfg
	case "cachestat":
		var cfg config.CACHESTAT_YAML
		if err := raw.Config.Decode(&cfg); err != nil {
			return err
		}
		i.Config = &cfg
	default:
		return fmt.Errorf("unknown input type: %s", i.Type)
	}
	return nil
}

type LogConfig struct {
	Path  string `yaml:"path"`
	Level string `yaml:"level" `
}

type InputType interface{}

// TODO move status filed in xx_YAML struct to InputConfig
type InputConfig struct {
	Type   string    `yaml:"type"`
	Config InputType `yaml:"config"`
	// Syscall *conf.SYSCALL_YAML `yaml:"syscall"`
	// FutexSnoop *conf.FUTEXSNOOP_YAML `yaml:"futexsnoop"`
	// Offcpu *conf.OFFCPU_YAML `yaml:"offcpu"`
	// Oncpu *conf.ONCPU_YAML `yaml:"oncpu"`
	// CacheStat *conf.CACHESTAT_YAML `yaml:"cachestat"`
}

type ProcessorConfig struct {
	Type string `yaml:"type"`
}

type ExporterConfig struct {
	Type string `yaml:"type"`
}

type Config struct {
	Log        LogConfig         `yaml:"log"`
	Inputs     []InputConfig     `yaml:"inputs"`
	Processors []ProcessorConfig `yaml:processors`
	Exporters  []ExporterConfig  `yaml:exporters`
}

var defaultConfig = Config{
	Log: LogConfig{
		Path:  "/tmp/spycat/",
		Level: "INFO",
	},
	Inputs: []InputConfig{
		{
			Type: "futexsnoop",
			Config: &config.FUTEXSNOOP_YAML{
				LogLevel:        "INFO",
				AppName:         "futexsnoop_app",
				Pid:             0,
				Tid:             0,
				MaxDurMs:        1000000,
				MinDurMs:        1000,
				SymbolCacheSize: 256,
				Exporter:        "sqlite",
				Status:          "disable",
			},
		},
		{
			Type: "syscall",
			Config: &config.SYSCALL_YAML{
				LogLevel:        "INFO",
				AppName:         "syscall_app",
				Pid:             0,
				Tid:             0,
				MaxDurMs:        10000,
				MinDurMs:        1,
				SymbolCacheSize: 256,
				Stack:           false,
				Exporter:        "sqlite",
				Status:          "disable",
			},
		},
		{
			Type: "offcpu",
			Config: &config.OFFCPU_YAML{
				LogLevel:        "INFO",
				AppName:         "offcpu_app",
				Pid:             -1,
				MaxOffcpuMs:     1000000000,
				MinOffcpuMs:     0,
				SymbolCacheSize: 256,
				Exporter:        "sqlite",
				Status:          "disable",
			},
		},
		{
			Type: "oncpu",
			Config: &config.ONCPU_YAML{
				LogLevel:        "INFO",
				AppName:         "oncpu_app",
				Cpu:             "-1",
				SymbolCacheSize: 256,
				Exporter:        "sqlite",
				Status:          "disable",
			},
		},
		{
			Type: "cachestat",
			Config: &config.CACHESTAT_YAML{
				LogLevel:  "INFO",
				AppName:   "cachestat_app",
				Pid:       -1,
				CacheType: 0,
				Cpu:       -1,
				Exporter:  "sqlite",
				Status:    "disable",
			},
		},
	},
	Processors: []ProcessorConfig{
		{Type: "processor_default"},
	},
	Exporters: []ExporterConfig{
		{Type: "exporter_stdout"},
	},
}

func NewConfig() *Config {
	return &defaultConfig
}

func ConfigInit() error {
	YamlConfigGlobal = NewConfig()

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
		err = yaml.Unmarshal(data, YamlConfigGlobal)
		if err != nil {
			return fmt.Errorf("yaml unmarshal failed:%v", err)
		}
		initToolsConfig(&ToolsConfigGlobal, YamlConfigGlobal)
	}
	return nil
}

func ConfigWithYaml() bool {
	return len(ToolsConfigGlobal.Tools) > 0
}

func initSpy() error {
	return nil
}

func runSpy(spy *appspy.AppSpy, cfg interface{}, cf *appspy.Config, cb func(interface{}, chan *model.SpyEvent) core.BpfSpyer) error {
	err := spy.Init(cf)
	if err != nil {
		return fmt.Errorf("spy init failed:%v", err)
	}
	err = spy.Start()
	if err != nil {
		spy.Stop()
		return fmt.Errorf("spy start failed:%v", err)
	}
	receiver := spy.GetReceiver()
	spyer := cb(cfg, receiver.RcvChan())

	go func() {
		err := spyer.Start()
		if err != nil {
			fmt.Printf("bpfspy:%s, start failed:%v\n", spyer.Name(), err)
		}
	}()

	fmt.Printf("trace event:%s start\n", spyer.Name())
	return nil
}

func RunWithYaml() error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	cf := &appspy.Config{
		Exporter: "sqlite",
		Server:   "http://localhost:4040",
	}
	spy, err := appspy.NewAppSpy(cf)
	if err != nil {
		return err
	}

	for _, tool := range ToolsConfigGlobal.Tools {
		var err error
		switch tool {
		case model.OffCpu:
			var offcpu config.OFFCPU
			util.CopyStruct(&ToolsConfigGlobal.Offcpu, &offcpu)
			err = runSpy(spy, &offcpu, cf, func(cfg interface{}, buf chan *model.SpyEvent) core.BpfSpyer {
				cf, ok := cfg.(*config.OFFCPU)
				if ok {
					return cpu.NewOffCpuBpfSession(model.OffCpu, cf, buf)
				}
				return nil
			})
		case model.OnCpu:
			var oncpu config.ONCPU
			util.CopyStruct(&ToolsConfigGlobal.Oncpu, &oncpu)
			err = runSpy(spy, &oncpu, cf, func(cfg interface{}, buf chan *model.SpyEvent) core.BpfSpyer {
				cf, ok := cfg.(*config.ONCPU)
				if ok {
					return cpu.NewOnCpuBpfSession(model.OnCpu, cf, buf)
				}
				return nil
			})
		case model.CacheStat:
			var cachestat config.CACHESTAT
			util.CopyStruct(&ToolsConfigGlobal.Cachestat, &cachestat)
			err = runSpy(spy, &cachestat, cf, func(cfg interface{}, buf chan *model.SpyEvent) core.BpfSpyer {
				cf, ok := cfg.(*config.CACHESTAT)
				if ok {
					return mem.NewCacheStatBpfSession(model.CacheStat, cf, buf)
				}
				return nil
			})
		case model.Syscall:
			var syscal config.SYSCALL
			util.CopyStruct(&ToolsConfigGlobal.Syscall, &syscal)
			err = runSpy(spy, &syscal, cf, func(cfg interface{}, buf chan *model.SpyEvent) core.BpfSpyer {
				cf, ok := cfg.(*config.SYSCALL)
				if ok {
					return cpu.NewSyscallSession(model.Syscall, cf, buf)
				}
				return nil
			})
		case model.FutexSnoop:
			var futexsnoop config.FUTEXSNOOP
			util.CopyStruct(&ToolsConfigGlobal.Futexsnoop, &futexsnoop)
			err = runSpy(spy, &futexsnoop, cf, func(cfg interface{}, buf chan *model.SpyEvent) core.BpfSpyer {
				cf, ok := cfg.(*config.FUTEXSNOOP)
				if ok {
					return cpu.NewFutexSnoopSession(model.FutexSnoop, cf, buf)
				}
				return nil
			})
		}

		if err != nil {
			return err
		}
	}
	select {
	case s := <-sig:
		fmt.Printf("Received signal and exit:%d", s)
		os.Exit(0)
	}
	return nil
}

func GetConfig() *Config {
	return YamlConfigGlobal
}

func (c *Config) ConfigLog() {

}

func (c *Config) ConfigInputs() {

}

func (c *Config) ConfigProcessors() {

}

func (c *Config) ConfigExporters() {

}
