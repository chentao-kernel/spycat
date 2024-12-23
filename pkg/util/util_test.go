package util

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestKprobeExists(t *testing.T) {
	want := false
	ret := KprobeExists("finish_task_switch.isra.0")
	cmd := exec.Command("bash", "-c", "cat /proc/kallsyms | grep finish_task_switch")
	output, err := cmd.Output()
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, " ")
		if len(parts) == 3 {
			if parts[2] == "finish_task_switch.isra.0" {
				want = true
			}
		}
	}
	if ret != want {
		t.Errorf("sym:%s not exists\n", "finish_task_switch.isra.0")
	}
}

func TestTracePointExists(t *testing.T) {
	want := false
	ret := TracePointExists("sched", "sched_switch")

	if ret != want {
		t.Errorf("test tracepoint failed:%s\n", "sched_switch")
	}
}

func TestHostInfo(t *testing.T) {
	var machine_id string
	var ip string
	var err error
	var host string

	machine_id, err = HostMachineId()
	if err != nil {
		t.Errorf("test host machine id failed")
	}
	ip, err = HostIp()
	if err != nil {
		t.Error("test host ip failed")
	}
	host, err = HostName()
	if err != nil {
		t.Error("test host name failed")
	}
	fmt.Printf("host info, machine_id: %s, kernel: %s, ip:%s, hostname:%s\n", machine_id,
		HostKernelVersion(), ip, host)
}
