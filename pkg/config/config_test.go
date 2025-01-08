package config

import (
	conf "github.com/chentao-kernel/spycat/pkg/app/config"
	"testing"
)

func TestConfigInit(t *testing.T) {
	// init first for generate spycat.yaml
	err := ConfigInit()
	if err != nil {
		t.Errorf("test config init failed")
	}
	// init again for read spycat.yaml
	err = ConfigInit()
	if err != nil {
		t.Errorf("test config init failed")
	}
	syscall, ok := defaultConfig.Inputs[1].Config.(*conf.SYSCALL_YAML)
	if ok {
		if syscall.AppName != ToolsConfigGlobal.Syscall.AppName {
			t.Errorf("get yaml config failed,%s,%s", syscall.AppName, ToolsConfigGlobal.Syscall.AppName)
		}
	}
}
