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
}

type FUTEXSNOOP struct {
	LogLevel         string `def:"info" desc:"log level: debug|info|warn|error" mapstructure:"log-level"`
	AppName          string `def:"" desc:"application name used when uploading profiling data" mapstructure:"app-name"`
	Server           string `def:"http://localhost:4040" desc:"the server address" mapstructure:"server"`
	Pid              int    `def:"0" desc:"pid to trace, -1 to trace all pids" mapstructure:"pid"`
	Tid              int    `def:"0" desc:"tid to trace, -1 to trace all tids" mapstructure:"tid"`
	MaxDurMs         uint   `def:"1000000" desc:"max time(ms) wait unlock" mapstructure:"max_dur_ms"`
	MinDurMs         uint   `def:"1000" desc:"min time(ms) wait unlock" mapstructure:"min_dur_ms"`
	SymbolCacheSize  int    `def:"256" desc:"max size of symbols cache" mapstructure:"symbol-cache-size"`
	MaxLockHoldUsers uint   `def:"100" desc:"max users hold the same lock" mapstructure:"max-lock-hold-users"`
	TargetLock       uint   `def:"0" desc:"target lock addr" mapstructure:"target-lock"`
	Stack            bool   `def:"false" desc:"get stack info or not" mapstructure:"stack"`
	BtfPath          string `def:"" desc:"btf file path" mapstructure:"btf-path"`
	Exporter         string `def:"" desc:"data exporter: loki,pyroscoe,prometheus,disk,etc." mapstructure:"exporter"`
}

type SYSCALL struct {
	LogLevel        string `def:"info" desc:"log level: debug|info|warn|error" mapstructure:"log-level"`
	AppName         string `def:"" desc:"application name used when uploading profiling data" mapstructure:"app-name"`
	Server          string `def:"http://localhost:4040" desc:"the server address" mapstructure:"server"`
	Pid             int    `def:"0" desc:"pid to trace, 0 to trace all pids" mapstructure:"pid"`
	Tid             int    `def:"0" desc:"tid to trace, 0 to trace all tids" mapstructure:"tid"`
	MaxDurMs        uint   `def:"10000" desc:"max time(ms) wait unlock" mapstructure:"max_dur_ms"`
	MinDurMs        uint   `def:"1" desc:"min time(ms) wait unlock" mapstructure:"min_dur_ms"`
	SymbolCacheSize int    `def:"256" desc:"max size of symbols cache" mapstructure:"symbol-cache-size"`
	Stack           bool   `def:"false" desc:"get stack info or not" mapstructure:"stack"`
	BtfPath         string `def:"" desc:"btf file path" mapstructure:"btf-path"`
	Exporter        string `def:"" desc:"data exporter: loki,pyroscoe,prometheus,disk,etc." mapstructure:"exporter"`
}

type OFFCPU struct {
	LogLevel        string `def:"info" desc:"log level: debug|info|warn|error" mapstructure:"log-level"`
	AppName         string `def:"" desc:"application name used when uploading profiling data" mapstructure:"app-name"`
	Server          string `def:"http://localhost:4040" desc:"the server address" mapstructure:"server"`
	Pid             int    `def:"-1" desc:"pid to trace, -1 to trace all pids" mapstructure:"pid"`
	MaxOffcpuMs     uint   `def:"100000000" desc:"max offcpu ms" mapstructure:"max_offcpu"`
	MinOffcpuMs     uint   `def:"0" desc:"min offcpu ms" mapstructure:"min_offcpu"`
	SymbolCacheSize int    `def:"256" desc:"max size of symbols cache" mapstructure:"symbol-cache-size"`
	OnRqUs          uint   `def:"0" desc:"min onrq us" mapstructure:"onrq"`
	BtfPath         string `def:"" desc:"btf file path" mapstructure:"btf-path"`
	Exporter        string `def:"" desc:"data exporter: loki,pyroscoe,prometheus,disk,etc." mapstructure:"exporter"`
}

type ONCPU struct {
	LogLevel string `def:"info" desc:"log level: debug|info|warn|error" mapstructure:"log-level"`
	// Spy configuration
	AppName    string `def:"" desc:"application name used when uploading profiling data" mapstructure:"app-name"`
	SampleRate uint   `def:"100" desc:"sample rate for the profiler in Hz. 100 means reading 100 times per second" mapstructure:"sample-rate"`

	// Remote upstream configuration
	Server string `def:"http://localhost:4040" desc:"the server address" mapstructure:"server"`
	// AuthToken              string        `def:"" desc:"authorization token used to upload profiling data" mapstructure:"auth-token"`
	UploadThreads int           `def:"4" desc:"number of upload threads" mapstructure:"upload-threads"`
	UploadTimeout time.Duration `def:"10s" desc:"profile upload timeout" mapstructure:"upload-timeout"`
	UploadRate    time.Duration `def:"10s" desc:"profile upload rate " mapstructure:"upload-rate"`

	Cpu string `def:"-1" desc:"Number of cpu you want to profile, like:1,2,4; -1 to profile the whole system" mapstructure:"cpu"`
	// DetectSubprocesses bool   `def:"false" desc:"makes keep track of and profile subprocesses of the main process" mapstructure:"detect-subprocesses"`
	SymbolCacheSize int    `def:"256" desc:"max size of symbols cache" mapstructure:"symbol-cache-size"`
	BtfPath         string `def:"" desc:"btf file path" mapstructure:"btf-path"`
	Exporter        string `def:"" desc:"data exporter: loki,pyroscoe,prometheus,disk,etc." mapstructure:"exporter"`
}

type NET struct {
	LogLevel string `def:"info" desc:"log level: debug|info|warn|error" mapstructure:"log-level"`

	// Spy configuration
	AppName    string `def:"" desc:"application name used when uploading profiling data" mapstructure:"app-name"`
	SampleRate uint   `def:"100" desc:"sample rate for the profiler in Hz. 100 means reading 100 times per second" mapstructure:"sample-rate"`

	// Remote upstream configuration
	Server        string        `def:"http://localhost:4040" desc:"the server address" mapstructure:"server"`
	UploadThreads int           `def:"4" desc:"number of upload threads" mapstructure:"upload-threads"`
	UploadTimeout time.Duration `def:"10s" desc:"profile upload timeout" mapstructure:"upload-timeout"`
	UploadRate    time.Duration `def:"10s" desc:"profile upload rate " mapstructure:"upload-rate"`

	Tags            map[string]string `name:"tag" def:"" desc:"tag in key=value form. The flag may be specified multiple times" mapstructure:"tags"`
	Pid             int               `def:"-1" desc:"PID you want to profile. -1 to profile the whole system" mapstructure:"pid"`
	Cpu             int               `def:"-1" desc:"Number of cpu you want to profile. -1 to profile the whole system" mapstructure:"cpu"`
	Dport           int               `def:"-1" desc:"Dport you want profile." mapstructure:"dport"`
	Sport           int               `def:"-1" desc:"Sport you want profile." mapstructure:"sport"`
	Delay           int               `def:"100" desc:"User take packet delay(ms)." mapstructure:"delay"`
	ExitTime        int               `def:"2" desc:"time of days the profiling to exit, default 2 days" mapstructure:"exitTime"`
	SymbolCacheSize int               `def:"256" desc:"max size of symbols cache" mapstructure:"symbol-cache-size"`
	SLS             string            `def:"unuser" desc:"producer/consumer/produceraw data to/from SLS" mapstructure:"sls"`
	Endpoint        string            `def:"endpoint" desc:"SLS Endpoint" mapstructure:"endpoint"`
	AKID            string            `def:"akid" desc:"SLS AccessKeyID" mapstructure:"akid"`
	AKSE            string            `def:"akse" desc:"SLS AccessKeySecret" mapstructure:"akse"`
	Project         string            `def:"akid" desc:"SLS Project" mapstructure:"project"`
	Logstore        string            `def:"logstore" desc:"SLS Logstore" mapstructure:"logstore"`
	Encrypt         string            `def:"base64" desc:"Encryte ak/sk" mapstructure:"encrypt"`
}
