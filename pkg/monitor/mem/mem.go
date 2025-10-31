package mem

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chentao-kernel/spycat/pkg/monitor/metrics"
)

// MemoryCollector memory monitoring collector
type MemoryCollector struct {
	lastPageFaults *PageFaultStats
}

// PageFaultStats page fault statistics
type PageFaultStats struct {
	Timestamp      time.Time
	PageFaults     uint64
	MajorPageFaults uint64
}

// NewMemoryCollector creates a new memory collector
func NewMemoryCollector() *MemoryCollector {
	return &MemoryCollector{}
}

// Collect collects memory metrics
func (m *MemoryCollector) Collect() (*metrics.MemoryMetrics, error) {
	memInfo, err := m.readMemInfo()
	if err != nil {
		return nil, err
	}

	vmStats, err := m.readVMStats()
	if err != nil {
		return nil, err
	}

	// Calculate page fault deltas
	var pageFaultsDiff, majorPageFaultsDiff uint64
	currentPageFaults := &PageFaultStats{
		Timestamp:      time.Now(),
		PageFaults:     vmStats.PageFaults,
		MajorPageFaults: vmStats.MajorPageFaults,
	}

	if m.lastPageFaults != nil {
		pageFaultsDiff = currentPageFaults.PageFaults - m.lastPageFaults.PageFaults
		majorPageFaultsDiff = currentPageFaults.MajorPageFaults - m.lastPageFaults.MajorPageFaults
	}

	m.lastPageFaults = currentPageFaults

	return &metrics.MemoryMetrics{
		Timestamp:       time.Now(),
		Total:           memInfo.Total,
		Used:            memInfo.Used,
		Free:            memInfo.Free,
		Available:       memInfo.Available,
		SwapTotal:       memInfo.SwapTotal,
		SwapUsed:        memInfo.SwapUsed,
		SwapFree:        memInfo.SwapFree,
		PageCache:       memInfo.PageCache,
		Buffers:         memInfo.Buffers,
		PageFaults:      pageFaultsDiff,
		MajorPageFaults: majorPageFaultsDiff,
	}, nil
}

// MemInfo memory information
type MemInfo struct {
	Total     uint64
	Used      uint64
	Free      uint64
	Available uint64
	Buffers   uint64
	PageCache uint64
	SwapTotal uint64
	SwapFree  uint64
	SwapUsed  uint64
}

// readMemInfo reads memory information from /proc/meminfo
func (m *MemoryCollector) readMemInfo() (*MemInfo, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/meminfo: %v", err)
	}
	defer file.Close()

	memInfo := &MemInfo{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		// Convert to bytes (meminfo uses KB)
		value *= 1024

		switch fields[0] {
		case "MemTotal:":
			memInfo.Total = value
		case "MemFree:":
			memInfo.Free = value
		case "MemAvailable:":
			memInfo.Available = value
		case "Buffers:":
			memInfo.Buffers = value
		case "Cached:":
			memInfo.PageCache = value
		case "SwapTotal:":
			memInfo.SwapTotal = value
		case "SwapFree:":
			memInfo.SwapFree = value
		}
	}

	// Calculate used memory and swap space
	memInfo.Used = memInfo.Total - memInfo.Free - memInfo.Buffers - memInfo.PageCache
	memInfo.SwapUsed = memInfo.SwapTotal - memInfo.SwapFree

	return memInfo, nil
}

// VMStats VM statistics information
type VMStats struct {
	PageFaults     uint64
	MajorPageFaults uint64
}

// readVMStats reads VM statistics from /proc/vmstat
func (m *MemoryCollector) readVMStats() (*VMStats, error) {
	file, err := os.Open("/proc/vmstat")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/vmstat: %v", err)
	}
	defer file.Close()

	vmStats := &VMStats{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}

		switch fields[0] {
		case "pgfault":
			vmStats.PageFaults = value
		case "pgmajfault":
			vmStats.MajorPageFaults = value
		}
	}

	return vmStats, nil
}

// GetMemoryUsage gets memory usage percentage
func (m *MemoryCollector) GetMemoryUsage(memInfo *MemInfo) float64 {
	if memInfo.Total == 0 {
		return 0
	}
	return float64(memInfo.Used) / float64(memInfo.Total) * 100
}

// GetSwapUsage gets swap space usage percentage
func (m *MemoryCollector) GetSwapUsage(memInfo *MemInfo) float64 {
	if memInfo.SwapTotal == 0 {
		return 0
	}
	return float64(memInfo.SwapUsed) / float64(memInfo.SwapTotal) * 100
}
