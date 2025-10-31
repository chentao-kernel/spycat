package cpu

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chentao-kernel/spycat/pkg/monitor/metrics"
)

// CPUCollector CPU monitoring collector
type CPUCollector struct {
	lastStats *CPUStats
}

// CPUStats CPU statistics information
type CPUStats struct {
	Timestamp    time.Time
	User         uint64
	Nice         uint64
	System       uint64
	Idle         uint64
	IOWait       uint64
	IRQ          uint64
	SoftIRQ      uint64
	Steal        uint64
	Guest        uint64
	GuestNice    uint64
	ContextSwitches uint64
	Interrupts   uint64
}

// NewCPUCollector creates a new CPU collector
func NewCPUCollector() *CPUCollector {
	return &CPUCollector{}
}

// Collect collects CPU metrics
func (c *CPUCollector) Collect() (*metrics.CPUMetrics, error) {
	stats, err := c.readCPUStats()
	if err != nil {
		return nil, err
	}

	load, err := c.readLoadAvg()
	if err != nil {
		return nil, err
	}

	contextSwitches, interrupts, err := c.readContextStats()
	if err != nil {
		return nil, err
	}

	// Calculate CPU usage
	var cpuMetrics *metrics.CPUMetrics
	if c.lastStats != nil {
		cpuMetrics = c.calculateUsage(stats, load, contextSwitches, interrupts)
	}

	c.lastStats = stats
	return cpuMetrics, nil
}

// readCPUStats reads CPU statistics from /proc/stat
func (c *CPUCollector) readCPUStats() (*CPUStats, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/stat: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 8 {
				return nil, fmt.Errorf("invalid cpu stat line: %s", line)
			}

			stats := &CPUStats{
				Timestamp: time.Now(),
			}

			// Parse CPU time
			stats.User, _ = strconv.ParseUint(fields[1], 10, 64)
			stats.Nice, _ = strconv.ParseUint(fields[2], 10, 64)
			stats.System, _ = strconv.ParseUint(fields[3], 10, 64)
			stats.Idle, _ = strconv.ParseUint(fields[4], 10, 64)
			stats.IOWait, _ = strconv.ParseUint(fields[5], 10, 64)
			stats.IRQ, _ = strconv.ParseUint(fields[6], 10, 64)
			stats.SoftIRQ, _ = strconv.ParseUint(fields[7], 10, 64)

			if len(fields) > 8 {
				stats.Steal, _ = strconv.ParseUint(fields[8], 10, 64)
			}
			if len(fields) > 9 {
				stats.Guest, _ = strconv.ParseUint(fields[9], 10, 64)
			}
			if len(fields) > 10 {
				stats.GuestNice, _ = strconv.ParseUint(fields[10], 10, 64)
			}

			return stats, nil
		}
	}

	return nil, fmt.Errorf("cpu stats not found in /proc/stat")
}

// readLoadAvg reads system load average
func (c *CPUCollector) readLoadAvg() ([]float64, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/loadavg: %v", err)
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return nil, fmt.Errorf("invalid loadavg format")
	}

	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)

	return []float64{load1, load5, load15}, nil
}

// readContextStats reads context switches and interrupt statistics
func (c *CPUCollector) readContextStats() (uint64, uint64, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open /proc/stat: %v", err)
	}
	defer file.Close()

	var contextSwitches, interrupts uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ctxt ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				contextSwitches, _ = strconv.ParseUint(fields[1], 10, 64)
			}
		} else if strings.HasPrefix(line, "intr ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				interrupts, _ = strconv.ParseUint(fields[1], 10, 64)
			}
		}
	}

	return contextSwitches, interrupts, nil
}

// calculateUsage calculates CPU usage percentages
func (c *CPUCollector) calculateUsage(current *CPUStats, load []float64, contextSwitches, interrupts uint64) *metrics.CPUMetrics {
	prev := c.lastStats

	// Calculate total time difference
	prevTotal := prev.User + prev.Nice + prev.System + prev.Idle + prev.IOWait + prev.IRQ + prev.SoftIRQ + prev.Steal
	currentTotal := current.User + current.Nice + current.System + current.Idle + current.IOWait + current.IRQ + current.SoftIRQ + current.Steal
	totalDiff := float64(currentTotal - prevTotal)

	if totalDiff == 0 {
		return nil
	}

	// Calculate usage percentages for each state
	userPercent := float64(current.User-prev.User) / totalDiff * 100
	systemPercent := float64(current.System-prev.System) / totalDiff * 100
	idlePercent := float64(current.Idle-prev.Idle) / totalDiff * 100
	ioWaitPercent := float64(current.IOWait-prev.IOWait) / totalDiff * 100

	// Calculate context switch and interrupt deltas
	contextSwitchDiff := contextSwitches - c.lastStats.ContextSwitches
	interruptsDiff := interrupts - c.lastStats.Interrupts

	return &metrics.CPUMetrics{
		Timestamp:     current.Timestamp,
		UserPercent:   userPercent,
		SystemPercent: systemPercent,
		IdlePercent:   idlePercent,
		IOWaitPercent: ioWaitPercent,
		Load1:         load[0],
		Load5:         load[1],
		Load15:        load[2],
		ContextSwitch: contextSwitchDiff,
		Interrupts:    interruptsDiff,
	}
}
