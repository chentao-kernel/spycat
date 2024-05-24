package util

import (
	"bufio"
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
