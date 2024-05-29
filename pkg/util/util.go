package util

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/chentao-kernel/spycat/pkg/log"
)

const cpuOnline = "/sys/devices/system/cpu/online"
const DEBUGFS = "/sys/kernel/debug/tracing"
const TRACEFS = "/sys/kernel/tracing"
const KPROBELIST = "/sys/kernel/debug/kprobes/blacklist"

func PrintStructFields(s interface{}) {
	v := reflect.ValueOf(s)
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := t.Field(i).Name
		fieldValue := field.Interface()

		fmt.Printf("%s: %v\n", fieldName, fieldValue)
	}
}

// Get returns a slice with the online CPUs, for example `[0, 2, 3]`
func CpuGet() ([]uint, error) {
	buf, err := os.ReadFile(cpuOnline)
	if err != nil {
		return nil, err
	}
	return ReadCPURange(string(buf))
}

// loosely based on https://github.com/iovisor/bcc/blob/v0.3.0/src/python/bcc/utils.py#L15
func ReadCPURange(cpuRangeStr string) ([]uint, error) {
	var cpus []uint
	cpuRangeStr = strings.Trim(cpuRangeStr, "\n ")
	for _, cpuRange := range strings.Split(cpuRangeStr, ",") {
		rangeOp := strings.SplitN(cpuRange, "-", 2)
		first, err := strconv.ParseUint(rangeOp[0], 10, 32)
		if err != nil {
			return nil, err
		}
		if len(rangeOp) == 1 {
			cpus = append(cpus, uint(first))
			continue
		}
		last, err := strconv.ParseUint(rangeOp[1], 10, 32)
		if err != nil {
			return nil, err
		}
		for n := first; n <= last; n++ {
			cpus = append(cpus, uint(n))
		}
	}
	return cpus, nil
}

type Units string

func (u Units) String() string {
	return string(u)
}

const (
	SamplesUnits         Units = "samples"
	ObjectsUnits         Units = "objects"
	GoroutinesUnits      Units = "goroutines"
	BytesUnits           Units = "bytes"
	LockNanosecondsUnits Units = "lock_nanoseconds"
	LockSamplesUnits     Units = "lock_samples"
)

type AggregationType string

const (
	AverageAggregationType AggregationType = "average"
	SumAggregationType     AggregationType = "sum"
)

func (a AggregationType) String() string {
	return string(a)
}

type Metadata struct {
	SpyName         string
	SampleRate      uint32
	Units           Units
	AggregationType AggregationType
}

func PrintStack() {
	buf := make([]byte, 1024)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		buf = make([]byte, 2*len(buf))
	}
	fmt.Printf("Stack trace:\n%s\n", buf)
}

func TracePointExists(category string, event string) bool {
	path := filepath.Join(DEBUGFS, "events", category, event, "format")
	_, err := os.Stat(path)

	return err == nil
}

func readKprobeBlacklist() ([]string, error) {
	file, err := os.Open(KPROBELIST)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var blacklist []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, " ")
		if len(parts) == 2 {
			blacklist = append(blacklist, parts[1])
		}
	}
	return blacklist, nil
}

func checkKallsyms(name string) bool {
	file, err := os.Open("/proc/kallsyms")
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// TODO: record kernel syms in global variable
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, " ")
		if len(parts) >= 3 && parts[2] == name {
			return true
		}
	}
	if err := scanner.Err(); err != nil {
		return false
	}
	return false
}

func KprobeExists(name string) bool {
	blacklist, err := readKprobeBlacklist()
	if err != nil {
		log.Loger.Error("failed to read kprobe list:%v\n", err)
	} else {
		for _, sym := range blacklist {
			if sym == name {
				return false
			}
		}
	}
	return checkKallsyms(name)
}
