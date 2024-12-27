package config

import (
	"time"
)

type Config struct {
	Version bool `mapstructure:"version"`

	ONCPU      ONCPU      `skip:"true" mapstructure:",squash"`
	OFFCPU     OFFCPU     `skip:"true" mapstructure:",squash"`
	FUTEXSNOOP FUTEXSNOOP `skip:"true" mapstructure:",squash"`
	SYSCALL    SYSCALL    `skip:"true" mapstructure:",squash"`
	CACHESTAT  CACHESTAT  `skip:"true" mapstructure:",squash"`

	ONCPU_YAML      ONCPU_YAML
	OFFCPU_YAML     OFFCPU_YAML
	FUTEXSNOOP_YAML FUTEXSNOOP_YAML
	SYSCALL_YAML    SYSCALL_YAML
	CACHESTAT_YAML  CACHESTAT_YAML
}

type FUTEXSNOOP struct {
	LogLevel         string `def:"INFO" desc:"log level: DEBUG|INFO|WARN|ERROR" mapstructure:"log-level"`
	AppName          string `def:"" desc:"application name used when uploading profiling data" mapstructure:"app-name"`
	Server           string `def:"http://localhost:4040" desc:"the server address" mapstructure:"server"`
	Pid              int    `def:"0" desc:"pid to trace, -1 to trace all pids" mapstructure:"pid"`
	Tid              int    `def:"0" desc:"tid to trace, -1 to trace all tids" mapstructure:"tid"`
	MaxDurMs         uint   `def:"1000000" desc:"max time(ms) wait unlock" mapstructure:"max_dur_ms"`
	MinDurMs         uint   `def:"1000" desc:"min time(ms) wait unlock" mapstructure:"min_dur_ms"`
	SymbolCacheSize  int    `def:"256" desc:"max size of symbols cache" mapstructure:"symbol-cache-size"`
	MaxLockHoldUsers uint   `def:"100" desc:"max users hold the same lock" mapstructure:"max-lock-hold-users"`
	TargetLock       uint   `def:"0" desc:"target lock addr" mapstructure:"target-lock"`
	Stack            bool   `def:"false" desc:"get stack INFO or not" mapstructure:"stack"`
	BtfPath          string `def:"" desc:"btf file path" mapstructure:"btf-path"`
	Exporter         string `def:"" desc:"data exporter: loki,pyroscoe,prometheus,disk,etc." mapstructure:"exporter"`
	Status           string
}

type FUTEXSNOOP_YAML struct {
	LogLevel         string `yaml:"log_level"`
	AppName          string `yaml:"app_name"`
	Server           string `yaml:"server"`
	Pid              int    `yaml:"pid"`
	Tid              int    `yaml:"tid"`
	MaxDurMs         uint   `yaml:"max_dur_ms"`
	MinDurMs         uint   `yaml:"min_dur_ms"`
	SymbolCacheSize  int    `yaml:"symbol_cache_size"`
	MaxLockHoldUsers uint   `yaml:"max_lock_hold_users"`
	TargetLock       bool   `yaml:"target_lock"`
	BtfPath          string `yaml:"btf_path"`
	Exporter         string `yaml:"exporter"`
	Status           string `yaml:"status"`
}

type SYSCALL struct {
	LogLevel        string `def:"INFO" desc:"log level: DEBUG|INFO|WARN|ERROR" mapstructure:"log_level"`
	AppName         string `def:"" desc:"application name used when uploading profiling data" mapstructure:"app_name"`
	Server          string `def:"http://localhost:4040" desc:"the server address" mapstructure:"server"`
	Pid             int    `def:"0" desc:"pid to trace, 0 to trace all pids" mapstructure:"pid"`
	Tid             int    `def:"0" desc:"tid to trace, 0 to trace all tids" mapstructure:"tid"`
	MaxDurMs        uint   `def:"10000" desc:"max time(ms) syscall used" mapstructure:"max_dur_ms"`
	MinDurMs        uint   `def:"1" desc:"min time(ms) syscall used" mapstructure:"min_dur_ms"`
	Syscall         string `def:"" desc:"set syscall to filter event" mapstructure:"syscall"`
	SymbolCacheSize int    `def:"256" desc:"max size of symbols cache" mapstructure:"symbol_cache_size"`
	Stack           bool   `def:"false" desc:"get stack INFO or not" mapstructure:"stack"`
	BtfPath         string `def:"" desc:"btf file path" mapstructure:"btf-path"`
	Exporter        string `def:"" desc:"data exporter: loki,pyroscoe,prometheus,disk,etc." mapstructure:"exporter"`
	Status          string
}

// SYSCALL_YAML for yaml config, SYSCALL for cmd config
// If we use yaml, SYSCALL_YAML will be assigned to SYSCALL which is the final parameter
// to eBPF tools
type SYSCALL_YAML struct {
	LogLevel        string `yaml:"log_level"`
	AppName         string `yaml:"app_name"`
	Server          string `yaml:"server"`
	Pid             int    `yaml:"pid"`
	Tid             int    `yaml:"tid"`
	MaxDurMs        uint   `yaml:"max_dur_ms"`
	MinDurMs        uint   `yaml:"min_dur_ms"`
	Syscall         string `yaml:"syscall"`
	SymbolCacheSize int    `yaml:"symbol_cache_size"`
	Stack           bool   `yaml:"stack"`
	BtfPath         string `yaml:"btf_path"`
	Exporter        string `yaml:"exporter"`
	Status          string `yaml:"status"`
}

type OFFCPU struct {
	LogLevel        string `def:"INFO" desc:"log level: DEBUG|INFO|WARN|ERROR" mapstructure:"log_level"`
	AppName         string `def:"" desc:"application name used when uploading profiling data" mapstructure:"app_name"`
	Server          string `def:"http://localhost:4040" desc:"the server address" mapstructure:"server"`
	Pid             int    `def:"-1" desc:"pid to trace, -1 to trace all pids" mapstructure:"pid"`
	MaxOffcpuMs     uint   `def:"100000000" desc:"max offcpu ms" mapstructure:"max_offcpu"`
	MinOffcpuMs     uint   `def:"0" desc:"min offcpu ms" mapstructure:"min_offcpu"`
	SymbolCacheSize int    `def:"256" desc:"max size of symbols cache" mapstructure:"symbol_cache_size"`
	RqDurMs         uint   `def:"0" desc:"min onrq ms" mapstructure:"rq_dur"`
	BtfPath         string `def:"" desc:"btf file path" mapstructure:"btf_path"`
	Exporter        string `def:"" desc:"data exporter: loki,pyroscoe,prometheus,disk,etc." mapstructure:"exporter"`
	Status          string
}

type OFFCPU_YAML struct {
	LogLevel        string `yaml:"log_level"`
	AppName         string `yaml:"app_name"`
	Server          string `yaml:"server"`
	Pid             int    `yaml:"pid"`
	MaxOffcpuMs     uint   `yaml:"max_offcpu_ms"`
	MinOffcpuMs     uint   `yaml:"min_offcpu_ms"`
	SymbolCacheSize int    `yaml:"symbol_cache_size"`
	RqDurMs         uint   `yaml:"rq_dur_ms"`
	BtfPath         string `yaml:"btf_path"`
	Exporter        string `yaml:"exporter"`
	Status          string `yaml:"status"`
}

type ONCPU struct {
	LogLevel string `def:"INFO" desc:"log level: DEBUG|INFO|WARN|ERROR" mapstructure:"log_level"`
	// Spy configuration
	AppName    string `def:"" desc:"application name used when uploading profiling data" mapstructure:"app_name"`
	SampleRate uint   `def:"100" desc:"sample rate for the profiler in Hz. 100 means reading 100 times per second" mapstructure:"sample_rate"`

	// Remote upstream configuration
	Server string `def:"http://localhost:4040" desc:"the server address" mapstructure:"server"`
	// AuthToken              string        `def:"" desc:"authorization token used to upload profiling data" mapstructure:"auth-token"`
	UploadThreads int           `def:"4" desc:"number of upload threads" mapstructure:"upload_threads"`
	UploadTimeout time.Duration `def:"10s" desc:"profile upload timeout" mapstructure:"upload_timeout"`
	UploadRate    time.Duration `def:"10s" desc:"profile upload rate " mapstructure:"upload_rate"`

	Cpu string `def:"-1" desc:"Number of cpu you want to profile, like:1,2,4; -1 to profile the whole system" mapstructure:"cpu"`
	// DetectSubprocesses bool   `def:"false" desc:"makes keep track of and profile subprocesses of the main process" mapstructure:"detect-subprocesses"`
	SymbolCacheSize int    `def:"256" desc:"max size of symbols cache" mapstructure:"symbol_cache_size"`
	BtfPath         string `def:"" desc:"btf file path" mapstructure:"btf-path"`
	Exporter        string `def:"" desc:"data exporter: loki,pyroscoe,prometheus,disk,etc." mapstructure:"exporter"`
	Status          string
}

type ONCPU_YAML struct {
	LogLevel        string        `yaml:"log_level"`
	AppName         string        `yaml:"app_name"`
	SampleRate      uint          `yaml:"sample_rate"`
	Server          string        `yaml:"server"`
	UploadThreads   int           `yaml:"upload_threads"`
	UploadTimeout   time.Duration `yaml:"upload_timeout"`
	UploadRate      time.Duration `yaml:"upload_rate"`
	Cpu             string        `yaml:"cpu"`
	SymbolCacheSize int           `yaml:"symbol_cache_size"`
	BtfPath         string        `yaml:"btf_path"`
	Exporter        string        `yaml:"exporter"`
	Status          string        `yaml:"status"`
}

type CACHESTAT struct {
	LogLevel   string        `def:"INFO" desc:"log level: DEBUG|INFO|WARN|ERROR" mapstructure:"log_level"`
	AppName    string        `def:"" desc:"application name used when uploading profiling data" mapstructure:"app_name"`
	Server     string        `def:"http://localhost:4040" desc:"the server address" mapstructure:"server"`
	UploadRate time.Duration `def:"30s" desc:"upload for the cachestat data. 30 means 30s upload the trace data from kernel" mapstructure:"upload_rate"`
	Pid        int           `def:"-1" desc:"pid to trace, -1 to trace all pids" mapstructure:"pid"`
	CacheType  int           `def:"0" desc:"cache type to trace, 0:read/write cache, 1:read cache, 2:write cache" mapstructure:"cache_type"`
	Cpu        int           `def:"-1" desc:"target cpu to trace. -1 to trace all cpus" mapstructure:"cpu"`
	BtfPath    string        `def:"" desc:"btf file path" mapstructure:"btf_path"`
	Exporter   string        `def:"" desc:"data exporter: loki,pyroscoe,prometheus,disk,etc." mapstructure:"exporter"`
	Status     string
}

type CACHESTAT_YAML struct {
	LogLevel   string        `yaml:"log_level"`
	AppName    string        `yaml:"app_name"`
	Server     string        `yaml:"server"`
	UploadRate time.Duration `yaml:"upload_rate"`
	Pid        int           `yaml:"pid"`
	CacheType  int           `yaml:"cache_type"`
	Cpu        int           `yaml:"cpu"`
	BtfPath    string        `yaml:"btf_path"`
	Exporter   string        `yaml:"exporter"`
	Status     string        `yaml:"status"`
}

type NET struct {
	LogLevel string `def:"INFO" desc:"log level: DEBUG|INFO|WARN|ERROR" mapstructure:"log_level"`

	// Spy configuration
	AppName    string `def:"" desc:"application name used when uploading profiling data" mapstructure:"app_name"`
	SampleRate uint   `def:"100" desc:"sample rate for the profiler in Hz. 100 means reading 100 times per second" mapstructure:"sample_rate"`

	// Remote upstream configuration
	Server        string        `def:"http://localhost:4040" desc:"the server address" mapstructure:"server"`
	UploadThreads int           `def:"4" desc:"number of upload threads" mapstructure:"upload_threads"`
	UploadTimeout time.Duration `def:"10s" desc:"profile upload timeout" mapstructure:"upload_timeout"`
	UploadRate    time.Duration `def:"10s" desc:"profile upload rate " mapstructure:"upload_rate"`

	Tags            map[string]string `name:"tag" def:"" desc:"tag in key=value form. The flag may be specified multiple times" mapstructure:"tags"`
	Pid             int               `def:"-1" desc:"PID you want to profile. -1 to profile the whole system" mapstructure:"pid"`
	Cpu             int               `def:"-1" desc:"Number of cpu you want to profile. -1 to profile the whole system" mapstructure:"cpu"`
	Dport           int               `def:"-1" desc:"Dport you want profile." mapstructure:"dport"`
	Sport           int               `def:"-1" desc:"Sport you want profile." mapstructure:"sport"`
	Delay           int               `def:"100" desc:"User take packet delay(ms)." mapstructure:"delay"`
	ExitTime        int               `def:"2" desc:"time of days the profiling to exit, default 2 days" mapstructure:"exitTime"`
	SymbolCacheSize int               `def:"256" desc:"max size of symbols cache" mapstructure:"symbol_cache_size"`
	SLS             string            `def:"unuser" desc:"producer/consumer/produceraw data to/from SLS" mapstructure:"sls"`
	Endpoint        string            `def:"endpoint" desc:"SLS Endpoint" mapstructure:"endpoint"`
	AKID            string            `def:"akid" desc:"SLS AccessKeyID" mapstructure:"akid"`
	AKSE            string            `def:"akse" desc:"SLS AccessKeySecret" mapstructure:"akse"`
	Project         string            `def:"akid" desc:"SLS Project" mapstructure:"project"`
	Logstore        string            `def:"logstore" desc:"SLS Logstore" mapstructure:"logstore"`
	Encrypt         string            `def:"base64" desc:"Encryte ak/sk" mapstructure:"encrypt"`
}
