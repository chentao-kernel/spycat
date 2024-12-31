package util

import (
	"bufio"
	"fmt"
	"math"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/chentao-kernel/spycat/pkg/log"
)

var (
	eventDataTransform = strings.NewReplacer(" ", "_")
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

// copy from opentelemetry-go
func estimateBootTimeOffset() (bootTimeOffset int64, err error) {
	// The datapath is currently using ktime_get_boot_ns for the pcap timestamp,
	// which corresponds to CLOCK_BOOTTIME. To be able to convert the the
	// CLOCK_BOOTTIME to CLOCK_REALTIME (i.e. a unix timestamp).

	// There can be an arbitrary amount of time between the execution of
	// time.Now() and unix.ClockGettime() below, especially under scheduler
	// pressure during program startup. To reduce the error introduced by these
	// delays, we pin the current Go routine to its OS thread and measure the
	// clocks multiple times, taking only the smallest observed difference
	// between the two values (which implies the smallest possible delay
	// between the two snapshots).
	var minDiff int64 = 1<<63 - 1
	estimationRounds := 25
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for round := 0; round < estimationRounds; round++ {
		var bootTimespec unix.Timespec

		// Ideally we would use __vdso_clock_gettime for both clocks here,
		// to have as little overhead as possible.
		// time.Now() will actually use VDSO on Go 1.9+, but calling
		// unix.ClockGettime to obtain CLOCK_BOOTTIME is a regular system call
		// for now.
		unixTime := time.Now()
		err = unix.ClockGettime(unix.CLOCK_BOOTTIME, &bootTimespec)
		if err != nil {
			return 0, err
		}

		offset := unixTime.UnixNano() - bootTimespec.Nano()
		diff := offset
		if diff < 0 {
			diff = -diff
		}

		if diff < minDiff {
			minDiff = diff
			bootTimeOffset = offset
		}
	}

	return bootTimeOffset, nil
}

func bootTimeOffset() int64 {
	t, err := estimateBootTimeOffset()
	if err != nil {
		panic(err)
	}
	return t
}

func BootOffsetToTime(nsec uint64) time.Time {
	if nsec > math.MaxInt64 {
		nsec = math.MaxInt64
	}
	return time.Unix(0, bootTimeOffset()+int64(nsec))
}

func TimeToBootOffset(ts time.Time) uint64 {
	nsec := ts.UnixNano() - bootTimeOffset()
	if nsec < 0 {
		return 0
	}
	return uint64(nsec)
}

func HostUUID() (string, error) {
	// should use root tot read uuid
	data, err := os.ReadFile("/sys/class/dmi/id/product_uuid")
	if err != nil {
		return "", fmt.Errorf("read uuid failed:%v", err)
	}
	return strings.TrimSpace(string(data)), nil
}

func HostMachineId() (string, error) {
	data, err := os.ReadFile("/etc/machine-id")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func HostKernelVersion() string {
	data, err := os.ReadFile("/proc/version_signature")
	if err != nil {
		return ""
	}
	return eventDataTransform.Replace(string(data))
}

func HostName() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	return hostname, nil
}

func HostIp() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range interfaces {
		if strings.Contains(iface.Name, "eth0") {
			addrs, err := iface.Addrs()
			if err != nil {
				return "", err
			}
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
					return ipnet.IP.String(), nil
				}
			}
		}
	}
	return "", fmt.Errorf("eth0 addr not found")
}
