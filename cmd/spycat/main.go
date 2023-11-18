package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/chentao-kernel/spycat/pkg/app"
	"github.com/chentao-kernel/spycat/pkg/log"
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
	app.Start()
	//go uprobe.NewBpfSession("uprobe", &core.SessionConfig{}).Start()
	waitSignal(sigCh)
}
