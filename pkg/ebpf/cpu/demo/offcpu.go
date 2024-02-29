package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	bpf "github.com/aquasecurity/libbpfgo"
	"golang.org/x/sys/unix"
)

var (
	bind, configFile string
)

type bpfEvent struct {
	ts    uint64
	pid   uint32
	cpu1  uint32
	cpu2  uint32
	sport uint16
	dport uint16
}

func showTrace(data []byte) {
	fmt.Printf("hello offcpu\n")
}

func attachProgs(bpfModule *bpf.Module) error {
	progIter := bpfModule.Iterator()
	for {
		prog := progIter.NextProgram()
		if prog == nil {
			break
		}
		if _, err := prog.AttachGeneric(); err != nil {
			return err
		}
		fmt.Printf("prog:%s attach success\n", prog.GetName())
	}
	return nil
}

func initArgsMap(bpfModule *bpf.Module) error {
	/*
		maps, err := bpfModule.GetMap("args_map")
		if err != nil {
			return fmt.Errorf("get map failed:%v", err)
		}
	*/
	return nil
}

func pollData(bpfModule *bpf.Module) {
	dataChan := make(chan []byte)
	lostChan := make(chan uint64)

	pb, err := bpfModule.InitPerfBuf("perf_map", dataChan, lostChan, 1)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	pb.Start()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer func() {
		pb.Stop()
		pb.Close()
		stop()
	}()

loop:
	for {
		select {
		case data := <-dataChan:
			showTrace(data)
		case e := <-lostChan:
			fmt.Printf("Events lost:%d\n", e)
		case <-ctx.Done():
			break loop
		}
	}
}

func main() {
	Start()
}

type BpfSession struct {
}

func Start() error {
	//var module *bpf.Module
	//var prog *bpf.BPFProg

	err := unix.Setrlimit(unix.RLIMIT_MEMLOCK, &unix.Rlimit{
		Cur: unix.RLIM_INFINITY,
		Max: unix.RLIM_INFINITY,
	})
	if err != nil {
		return err
	}

	args := bpf.NewModuleArgs{BPFObjBuff: offcpuBpf}

	module, err := bpf.NewModuleFromBufferArgs(args)
	if err != nil {
		fmt.Errorf("load module failed:%v", err)
		return err
	}

	err = module.BPFLoadObject()
	if err != nil {
		fmt.Errorf("load object failed:%v", err)
		return err
	}

	err = initArgsMap(module)
	if err != nil {
		fmt.Errorf("init args map failed:%v", err)
		return err
	}

	err = attachProgs(module)
	if err != nil {
		fmt.Errorf("attach program failed:%v", err)
	}

	pollData(module)
	return nil
}

func Stop() error {
	return nil
}

func ReadEvent() error {
	return nil
}

//go:embed offcpu.bpf.o
var offcpuBpf []byte
