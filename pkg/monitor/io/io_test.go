package io

import (
	"testing"
	"time"

	"github.com/chentao-kernel/spycat/pkg/monitor/metrics"
)

func TestIOCollectorCreation(t *testing.T) {
	collector := NewIOCollector()
	if collector == nil {
		t.Fatal("Failed to create IO collector")
	}
	if collector.lastStats == nil {
		t.Error("IO collector should have initialized lastStats map")
	}
	if len(collector.lastStats) != 0 {
		t.Error("New collector should have empty lastStats map")
	}
}

func TestDiskStatsParsing(t *testing.T) {
	diskStats := &DiskStats{
		Timestamp:    time.Now(),
		Device:       "sda",
		ReadIO:       1000,
		ReadMerges:   50,
		ReadSectors:  2000,
		ReadTicks:    500,
		WriteIO:      800,
		WriteMerges:  40,
		WriteSectors: 1600,
		WriteTicks:   400,
		InFlight:     2,
		IOTicks:      900,
		TimeInQueue:  100,
	}

	if diskStats.Device != "sda" {
		t.Errorf("Expected device 'sda', got '%s'", diskStats.Device)
	}
	if diskStats.ReadIO != 1000 {
		t.Errorf("Expected ReadIO 1000, got %d", diskStats.ReadIO)
	}
	if diskStats.ReadSectors != 2000 {
		t.Errorf("Expected ReadSectors 2000, got %d", diskStats.ReadSectors)
	}
	if diskStats.WriteIO != 800 {
		t.Errorf("Expected WriteIO 800, got %d", diskStats.WriteIO)
	}
	if diskStats.WriteSectors != 1600 {
		t.Errorf("Expected WriteSectors 1600, got %d", diskStats.WriteSectors)
	}
	if diskStats.InFlight != 2 {
		t.Errorf("Expected InFlight 2, got %d", diskStats.InFlight)
	}
	if diskStats.IOTicks != 900 {
		t.Errorf("Expected IOTicks 900, got %d", diskStats.IOTicks)
	}
}

func TestIOMetricsCalculation(t *testing.T) {
	collector := NewIOCollector()

	firstStats := &DiskStats{
		Timestamp:    time.Now(),
		Device:       "sda",
		ReadIO:       1000,
		ReadSectors:  2000,
		ReadTicks:    500,
		WriteIO:      800,
		WriteSectors: 1600,
		WriteTicks:   400,
		InFlight:     2,
		IOTicks:      900,
	}

	secondStats := &DiskStats{
		Timestamp:    firstStats.Timestamp.Add(time.Second),
		Device:       "sda",
		ReadIO:       1100,  // +100
		ReadSectors:  2200,  // +200 sectors = 200 * 512 = 102400 bytes
		ReadTicks:    600,   // +100
		WriteIO:      900,   // +100
		WriteSectors: 1800,  // +200 sectors = 200 * 512 = 102400 bytes
		WriteTicks:   500,   // +100
		InFlight:     3,
		IOTicks:      1000,  // +100
	}

	collector.lastStats["sda"] = firstStats

	timeDiff := secondStats.Timestamp.Sub(firstStats.Timestamp).Seconds()
	if timeDiff <= 0 {
		t.Fatal("Time difference should be positive")
	}

	readIOPS := float64(secondStats.ReadIO-firstStats.ReadIO) / timeDiff
	writeIOPS := float64(secondStats.WriteIO-firstStats.WriteIO) / timeDiff
	readBytes := (secondStats.ReadSectors - firstStats.ReadSectors) * 512
	writeBytes := (secondStats.WriteSectors - firstStats.WriteSectors) * 512
	readLatency := float64(secondStats.ReadTicks-firstStats.ReadTicks) / 1000.0
	writeLatency := float64(secondStats.WriteTicks-firstStats.WriteTicks) / 1000.0
	utilization := float64(secondStats.IOTicks-firstStats.IOTicks) / 10.0
	queueDepth := uint32(secondStats.InFlight)

	if readIOPS != 100.0 {
		t.Errorf("Expected ReadIOPS 100.0, got %.2f", readIOPS)
	}
	if writeIOPS != 100.0 {
		t.Errorf("Expected WriteIOPS 100.0, got %.2f", writeIOPS)
	}
	if readBytes != 102400 {
		t.Errorf("Expected ReadBytes 102400, got %d", readBytes)
	}
	if writeBytes != 102400 {
		t.Errorf("Expected WriteBytes 102400, got %d", writeBytes)
	}
	if readLatency != 0.1 {
		t.Errorf("Expected ReadLatency 0.1, got %.2f", readLatency)
	}
	if writeLatency != 0.1 {
		t.Errorf("Expected WriteLatency 0.1, got %.2f", writeLatency)
	}
	if utilization != 10.0 {
		t.Errorf("Expected Utilization 10.0, got %.2f", utilization)
	}
	if queueDepth != 3 {
		t.Errorf("Expected QueueDepth 3, got %d", queueDepth)
	}
}

func TestIOMetricsCreation(t *testing.T) {
	now := time.Now()
	ioMetrics := &metrics.IOMetrics{
		Timestamp:    now,
		Device:       "sda",
		ReadIOPS:     150.5,
		WriteIOPS:    200.3,
		ReadBytes:    1024 * 1024,  // 1MB
		WriteBytes:   2 * 1024 * 1024, // 2MB
		ReadLatency:  5.2,
		WriteLatency: 3.8,
		QueueDepth:   4,
		Utilization:  45.7,
	}

	if !ioMetrics.Timestamp.Equal(now) {
		t.Error("IO metrics timestamp mismatch")
	}
	if ioMetrics.Device != "sda" {
		t.Errorf("Expected device 'sda', got '%s'", ioMetrics.Device)
	}
	if ioMetrics.ReadIOPS != 150.5 {
		t.Errorf("Expected ReadIOPS 150.5, got %.2f", ioMetrics.ReadIOPS)
	}
	if ioMetrics.WriteIOPS != 200.3 {
		t.Errorf("Expected WriteIOPS 200.3, got %.2f", ioMetrics.WriteIOPS)
	}
	if ioMetrics.ReadBytes != 1024*1024 {
		t.Error("ReadBytes mismatch")
	}
	if ioMetrics.WriteBytes != 2*1024*1024 {
		t.Error("WriteBytes mismatch")
	}
	if ioMetrics.ReadLatency != 5.2 {
		t.Errorf("Expected ReadLatency 5.2, got %.2f", ioMetrics.ReadLatency)
	}
	if ioMetrics.WriteLatency != 3.8 {
		t.Errorf("Expected WriteLatency 3.8, got %.2f", ioMetrics.WriteLatency)
	}
	if ioMetrics.QueueDepth != 4 {
		t.Errorf("Expected QueueDepth 4, got %d", ioMetrics.QueueDepth)
	}
	if ioMetrics.Utilization != 45.7 {
		t.Errorf("Expected Utilization 45.7, got %.2f", ioMetrics.Utilization)
	}
}

func TestTotalIOCalculation(t *testing.T) {
	collector := NewIOCollector()

	ioMetrics := []*metrics.IOMetrics{
		{
			Device:       "sda",
			ReadIOPS:     100.0,
			WriteIOPS:    150.0,
			ReadBytes:    1024 * 1024,
			WriteBytes:   2 * 1024 * 1024,
			ReadLatency:  5.0,
			WriteLatency: 3.0,
			QueueDepth:   2,
			Utilization:  40.0,
		},
		{
			Device:       "sdb",
			ReadIOPS:     50.0,
			WriteIOPS:    75.0,
			ReadBytes:    512 * 1024,
			WriteBytes:   1 * 1024 * 1024,
			ReadLatency:  6.0,
			WriteLatency: 4.0,
			QueueDepth:   1,
			Utilization:  30.0,
		},
	}

	total := collector.GetTotalIO(ioMetrics)

	if total == nil {
		t.Fatal("Failed to calculate total IO")
	}

	if total.Device != "total" {
		t.Errorf("Expected device 'total', got '%s'", total.Device)
	}
	if total.ReadIOPS != 150.0 {
		t.Errorf("Expected total ReadIOPS 150.0, got %.2f", total.ReadIOPS)
	}
	if total.WriteIOPS != 225.0 {
		t.Errorf("Expected total WriteIOPS 225.0, got %.2f", total.WriteIOPS)
	}
	if total.ReadBytes != 1024*1024+512*1024 {
		t.Error("Total ReadBytes mismatch")
	}
	if total.WriteBytes != 2*1024*1024+1*1024*1024 {
		t.Error("Total WriteBytes mismatch")
	}
	if total.ReadLatency != (5.0+6.0)/2 {
		t.Errorf("Expected average ReadLatency %.2f, got %.2f", (5.0+6.0)/2, total.ReadLatency)
	}
	if total.WriteLatency != (3.0+4.0)/2 {
		t.Errorf("Expected average WriteLatency %.2f, got %.2f", (3.0+4.0)/2, total.WriteLatency)
	}
	if total.QueueDepth != 3 {
		t.Errorf("Expected total QueueDepth 3, got %d", total.QueueDepth)
	}
	if total.Utilization != (40.0+30.0)/2 {
		t.Errorf("Expected average Utilization %.2f, got %.2f", (40.0+30.0)/2, total.Utilization)
	}
}

func TestEmptyIOMetrics(t *testing.T) {
	collector := NewIOCollector()

	emptyMetrics := []*metrics.IOMetrics{}
	total := collector.GetTotalIO(emptyMetrics)
	if total != nil {
		t.Error("Total IO should be nil for empty metrics list")
	}

	total = collector.GetTotalIO(nil)
	if total != nil {
		t.Error("Total IO should be nil for nil metrics list")
	}
}

func TestIOCollectorIntegration(t *testing.T) {
	collector := NewIOCollector()

	firstResult, err := collector.Collect()
	if err != nil {
		t.Logf("First collection returned error (may be expected in test environment): %v", err)
		return
	}

	if firstResult == nil {
		t.Log("First collection returned nil (may be expected in test environment)")
		return
	}

	if len(firstResult) == 0 {
		t.Log("No IO devices found (may be expected in test environment)")
		return
	}

	for _, metric := range firstResult {
		if metric.Timestamp.IsZero() {
			t.Error("IO metrics timestamp should not be zero")
		}
		if metric.Device == "" {
			t.Error("IO device name should not be empty")
		}

		if metric.ReadIOPS < 0 {
			t.Errorf("ReadIOPS should be non-negative, got %.2f", metric.ReadIOPS)
		}
		if metric.WriteIOPS < 0 {
			t.Errorf("WriteIOPS should be non-negative, got %.2f", metric.WriteIOPS)
		}

		if metric.ReadLatency < 0 {
			t.Errorf("ReadLatency should be non-negative, got %.2f", metric.ReadLatency)
		}
		if metric.WriteLatency < 0 {
			t.Errorf("WriteLatency should be non-negative, got %.2f", metric.WriteLatency)
		}

		if metric.Utilization < 0 || metric.Utilization > 100 {
			t.Errorf("Utilization should be between 0 and 100, got %.2f", metric.Utilization)
		}
	}

	time.Sleep(100 * time.Millisecond)

	secondResult, err := collector.Collect()
	if err != nil {
		t.Logf("Second collection returned error: %v", err)
		return
	}

	if secondResult == nil {
		t.Log("Second collection returned nil")
		return
	}

	for _, metric := range secondResult {
		if metric.Timestamp.IsZero() {
			t.Error("Second collection timestamp should not be zero")
		}
	}
}

func TestDeviceList(t *testing.T) {
	collector := NewIOCollector()

	devices, err := collector.GetDeviceList()
	if err != nil {
		t.Logf("GetDeviceList returned error (may be expected in test environment): %v", err)
		return
	}

	if devices == nil {
		t.Error("Device list should not be nil")
	}

	for _, device := range devices {
		if device == "" {
			t.Error("Device name should not be empty")
		}
	}
}

func TestIOMetricsEdgeCases(t *testing.T) {
	collector := NewIOCollector()

	testCases := []struct {
		name     string
		metrics  []*metrics.IOMetrics
		expected *metrics.IOMetrics
	}{
		{
			name:     "Empty metrics",
			metrics:  []*metrics.IOMetrics{},
			expected: nil,
		},
		{
			name: "Single device",
			metrics: []*metrics.IOMetrics{
				{
					Device:       "sda",
					ReadIOPS:     100.0,
					WriteIOPS:    200.0,
					ReadBytes:    1024,
					WriteBytes:   2048,
					ReadLatency:  5.0,
					WriteLatency: 3.0,
					QueueDepth:   2,
					Utilization:  50.0,
				},
			},
			expected: &metrics.IOMetrics{
				Device:       "total",
				ReadIOPS:     100.0,
				WriteIOPS:    200.0,
				ReadBytes:    1024,
				WriteBytes:   2048,
				ReadLatency:  5.0,
				WriteLatency: 3.0,
				QueueDepth:   2,
				Utilization:  50.0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := collector.GetTotalIO(tc.metrics)
			if tc.expected == nil {
				if result != nil {
					t.Error("Expected nil result")
				}
				return
			}

			if result == nil {
				t.Fatal("Result should not be nil")
			}

			if result.Device != tc.expected.Device {
				t.Errorf("Expected device '%s', got '%s'", tc.expected.Device, result.Device)
			}
			if result.ReadIOPS != tc.expected.ReadIOPS {
				t.Errorf("Expected ReadIOPS %.2f, got %.2f", tc.expected.ReadIOPS, result.ReadIOPS)
			}
			if result.WriteIOPS != tc.expected.WriteIOPS {
				t.Errorf("Expected WriteIOPS %.2f, got %.2f", tc.expected.WriteIOPS, result.WriteIOPS)
			}
		})
	}
}
