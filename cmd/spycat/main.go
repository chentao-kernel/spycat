package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "net/http/pprof"

	"github.com/chentao-kernel/spycat/pkg/app"
	"github.com/chentao-kernel/spycat/pkg/config"
	"github.com/chentao-kernel/spycat/pkg/log"
)

var (
	version     string
	commitId    string
	releaseTime string
	goVersion   string
	author      string
)

func waitSignal(sigCh chan os.Signal) {
	select {
	case sig := <-sigCh:
		log.Loger.Info("Received signal and exit:%d", sig)
		os.Exit(-1)
	}
}

func main() {
	// init config
	err := config.ConfigInit()
	if err != nil {
		fmt.Printf("%v", err)
	}
	// init log
	log.LogInit(config.YamlConfigGlobal.Log.Path, log.LevelTransform(config.YamlConfigGlobal.Log.Level))

	sigCh := make(chan os.Signal, 1)
	if os.Getenv("ENABLE_SPYCAT_PPROF") == "true" {
		go func() {
			log.Loger.Info("Starting pprof server on :6060")
			if err := http.ListenAndServe(":6060", nil); err != nil {
				log.Loger.Error("pprof server failed: %v", err)
			}
		}()
	}

	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	// run with yaml
	if config.ConfigWithYaml() {
		err := config.RunWithYaml()
		if err != nil {
			log.Loger.Error("run with yaml failed:%v\n", err)
		}
	} else {
		// run with cmd
		cmd := app.NewCmd()
		app.SubCmdInit(cmd)
		// app.Start()
		// go uprobe.NewBpfSession("uprobe", &core.SessionConfig{}).Start()
		cmd.RootCmd.Execute()
	}
	// waitSignal(sigCh)
}

func init() {
	app.Version = version
	app.CommitId = commitId
	app.ReleaseTime = releaseTime
	app.GoVersion = goVersion
	app.Auhtor = author
}
