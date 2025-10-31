package io

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chentao-kernel/spycat/pkg/monitor/metrics"
)

// IOCollector
type IOCollector struct {
	lastStats map[string]*DiskStats
}

// DiskStats
type DiskStats struct {
	Timestamp    time.Time
	Device       string
	ReadIO       uint64
	ReadMerges   uint64
	ReadSectors  uint64
	ReadTicks    uint64
	WriteIO      uint64
	WriteMerges  uint64
	WriteSectors uint64
	WriteTicks   uint64
	InFlight     uint64
	IOTicks      uint64
	TimeInQueue  uint64
}

func NewIOCollector() *IOCollector {
	return &IOCollector{
		lastStats: make(map[string]*DiskStats),
	}
}

func (i *IOCollector) Collect() ([]*metrics.IOMetrics, error) {
	currentStats, err := i.readDiskStats()
	if err != nil {
		return nil, err
	}

	var ioMetrics []*metrics.IOMetrics

	for device, current := range currentStats {
		if last, exists := i.lastStats[device]; exists {
			// get the increment
			timeDiff := current.Timestamp.Sub(last.Timestamp).Seconds()
			if timeDiff <= 0 {
				continue
			}

			readIOPS := float64(current.ReadIO-last.ReadIO) / timeDiff
			writeIOPS := float64(current.WriteIO-last.WriteIO) / timeDiff
			readBytes := (current.ReadSectors - last.ReadSectors) * 512 // convert sector to bytes
			writeBytes := (current.WriteSectors - last.WriteSectors) * 512

			// calculate the latency (convert ticks to milliseconds)
			readLatency := float64(current.ReadTicks-last.ReadTicks) / 1000.0
			writeLatency := float64(current.WriteTicks-last.WriteTicks) / 1000.0

			// calculate usage
			utilization := float64(current.IOTicks-last.IOTicks) / 10.0 // convert ticks to percentages

			// queue depth
			queueDepth := uint32(current.InFlight)

			ioMetric := &metrics.IOMetrics{
				Timestamp:    current.Timestamp,
				Device:       device,
				ReadIOPS:     readIOPS,
				WriteIOPS:    writeIOPS,
				ReadBytes:    readBytes,
				WriteBytes:   writeBytes,
				ReadLatency:  readLatency,
				WriteLatency: writeLatency,
				QueueDepth:   queueDepth,
				Utilization:  utilization,
			}

			ioMetrics = append(ioMetrics, ioMetric)
		}

		// update last stat
		i.lastStats[device] = current
	}

	return ioMetrics, nil
}

// readDiskStats
func (i *IOCollector) readDiskStats() (map[string]*DiskStats, error) {
	file, err := os.Open("/proc/diskstats")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/diskstats: %v", err)
	}
	defer file.Close()

	stats := make(map[string]*DiskStats)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 14 {
			continue
		}

		// skip partitions and focus only on physical devices
		device := fields[2]
		if strings.HasPrefix(device, "loop") || strings.HasPrefix(device, "ram") {
			continue
		}

		// parse disk statistics fields
		readIO, _ := strconv.ParseUint(fields[3], 10, 64)
		readMerges, _ := strconv.ParseUint(fields[4], 10, 64)
		readSectors, _ := strconv.ParseUint(fields[5], 10, 64)
		readTicks, _ := strconv.ParseUint(fields[6], 10, 64)
		writeIO, _ := strconv.ParseUint(fields[7], 10, 64)
		writeMerges, _ := strconv.ParseUint(fields[8], 10, 64)
		writeSectors, _ := strconv.ParseUint(fields[9], 10, 64)
		writeTicks, _ := strconv.ParseUint(fields[10], 10, 64)
		inFlight, _ := strconv.ParseUint(fields[11], 10, 64)
		ioTicks, _ := strconv.ParseUint(fields[12], 10, 64)
		timeInQueue, _ := strconv.ParseUint(fields[13], 10, 64)

		stats[device] = &DiskStats{
			Timestamp:    time.Now(),
			Device:       device,
			ReadIO:       readIO,
			ReadMerges:   readMerges,
			ReadSectors:  readSectors,
			ReadTicks:    readTicks,
			WriteIO:      writeIO,
			WriteMerges:  writeMerges,
			WriteSectors: writeSectors,
			WriteTicks:   writeTicks,
			InFlight:     inFlight,
			IOTicks:      ioTicks,
			TimeInQueue:  timeInQueue,
		}
	}

	return stats, nil
}

func (i *IOCollector) GetDeviceList() ([]string, error) {
	stats, err := i.readDiskStats()
	if err != nil {
		return nil, err
	}

	var devices []string
	for device := range stats {
		devices = append(devices, device)
	}

	return devices, nil
}

func (i *IOCollector) GetTotalIO(ioMetrics []*metrics.IOMetrics) *metrics.IOMetrics {
	if len(ioMetrics) == 0 {
		return nil
	}

	total := &metrics.IOMetrics{
		Timestamp: time.Now(),
		Device:    "total",
	}

	var totalReadLatency, totalWriteLatency, totalUtilization float64
	count := len(ioMetrics)

	for _, metric := range ioMetrics {
		total.ReadIOPS += metric.ReadIOPS
		total.WriteIOPS += metric.WriteIOPS
		total.ReadBytes += metric.ReadBytes
		total.WriteBytes += metric.WriteBytes
		totalReadLatency += metric.ReadLatency
		totalWriteLatency += metric.WriteLatency
		total.QueueDepth += metric.QueueDepth
		totalUtilization += metric.Utilization
	}

	// calculate the average value
	if count > 0 {
		total.ReadLatency = totalReadLatency / float64(count)
		total.WriteLatency = totalWriteLatency / float64(count)
		total.Utilization = totalUtilization / float64(count)
	}

	return total
}
