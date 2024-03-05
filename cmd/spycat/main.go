package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/chentao-kernel/spycat/pkg/app"
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
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	cmd := app.NewCmd()
	app.SubCmdInit(cmd)
	//app.Start()
	//go uprobe.NewBpfSession("uprobe", &core.SessionConfig{}).Start()
	cmd.RootCmd.Execute()
	//waitSignal(sigCh)
}

func init() {
	app.Version = version
	app.CommitId = commitId
	app.ReleaseTime = releaseTime
	app.GoVersion = goVersion
	app.Auhtor = author
}
