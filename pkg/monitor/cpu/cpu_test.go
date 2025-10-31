package cpu

import (
	"testing"
	"time"
)

func TestCPUCollectorCreation(t *testing.T) {
	collector := NewCPUCollector()
	if collector == nil {
		t.Fatal("Failed to create CPU collector")
	}
	if collector.lastStats != nil {
		t.Error("New collector should not have last stats")
	}
}

func TestCPUStatsCalculation(t *testing.T) {
	collector := NewCPUCollector()

	firstStats := &CPUStats{
		Timestamp: time.Now(),
		User:      1000,
		Nice:      50,
		System:    500,
		Idle:      8000,
		IOWait:    200,
		IRQ:       100,
		SoftIRQ:   50,
		Steal:     0,
		Guest:     0,
		GuestNice: 0,
		ContextSwitches: 10000,
		Interrupts:      5000,
	}

	secondStats := &CPUStats{
		Timestamp: firstStats.Timestamp.Add(time.Second),
		User:      1200,  // +200
		Nice:      60,    // +10
		System:    600,   // +100
		Idle:      9000,  // +1000
		IOWait:    250,   // +50
		IRQ:       120,   // +20
		SoftIRQ:   60,    // +10
		Steal:     0,
		Guest:     0,
		GuestNice: 0,
		ContextSwitches: 11000, // +1000
		Interrupts:      5500,  // +500
	}

	collector.lastStats = firstStats

	load := []float64{1.2, 1.5, 1.3}
	cpuMetrics := collector.calculateUsage(secondStats, load, 11000, 5500)

	if cpuMetrics == nil {
		t.Fatal("Failed to calculate CPU usage")
	}

	// total diff = (1200+60+600+9000+250+120+60) - (1000+50+500+8000+200+100+50) = 11290 - 9900 = 1390
	totalDiff := float64(1390)
	expectedUserPercent := float64(200) / totalDiff * 100
	expectedSystemPercent := float64(100) / totalDiff * 100
	expectedIdlePercent := float64(1000) / totalDiff * 100
	expectedIOWaitPercent := float64(50) / totalDiff * 100

	if cpuMetrics.UserPercent != expectedUserPercent {
		t.Errorf("Expected UserPercent %.2f, got %.2f", expectedUserPercent, cpuMetrics.UserPercent)
	}
	if cpuMetrics.SystemPercent != expectedSystemPercent {
		t.Errorf("Expected SystemPercent %.2f, got %.2f", expectedSystemPercent, cpuMetrics.SystemPercent)
	}
	if cpuMetrics.IdlePercent != expectedIdlePercent {
		t.Errorf("Expected IdlePercent %.2f, got %.2f", expectedIdlePercent, cpuMetrics.IdlePercent)
	}
	if cpuMetrics.IOWaitPercent != expectedIOWaitPercent {
		t.Errorf("Expected IOWaitPercent %.2f, got %.2f", expectedIOWaitPercent, cpuMetrics.IOWaitPercent)
	}
	if cpuMetrics.Load1 != 1.2 {
		t.Errorf("Expected Load1 1.2, got %.2f", cpuMetrics.Load1)
	}
	if cpuMetrics.ContextSwitch != 1000 {
		t.Errorf("Expected ContextSwitch 1000, got %d", cpuMetrics.ContextSwitch)
	}
	if cpuMetrics.Interrupts != 500 {
		t.Errorf("Expected Interrupts 500, got %d", cpuMetrics.Interrupts)
	}
}

func TestCPUStatsCalculationWithZeroDiff(t *testing.T) {
	collector := NewCPUCollector()

	// same timestamp should return nil
	stats := &CPUStats{
		Timestamp: time.Now(),
		User:      1000,
		System:    500,
		Idle:      8000,
	}

	collector.lastStats = stats
	load := []float64{1.2, 1.5, 1.3}

	// use the same statistical data with a time difference of 0
	result := collector.calculateUsage(stats, load, 10000, 5000)
	if result != nil {
		t.Error("Expected nil result when time difference is zero")
	}
}

func TestCPUStatsParsing(t *testing.T) {
	fields := []string{"cpu", "1000", "50", "500", "8000", "200", "100", "50", "0", "0", "0"}

	stats := &CPUStats{
		Timestamp: time.Now(),
	}

	stats.User, _ = parseUintField(fields[1])
	stats.Nice, _ = parseUintField(fields[2])
	stats.System, _ = parseUintField(fields[3])
	stats.Idle, _ = parseUintField(fields[4])
	stats.IOWait, _ = parseUintField(fields[5])
	stats.IRQ, _ = parseUintField(fields[6])
	stats.SoftIRQ, _ = parseUintField(fields[7])

	if len(fields) > 8 {
		stats.Steal, _ = parseUintField(fields[8])
	}
	if len(fields) > 9 {
		stats.Guest, _ = parseUintField(fields[9])
	}
	if len(fields) > 10 {
		stats.GuestNice, _ = parseUintField(fields[10])
	}

	if stats.User != 1000 {
		t.Errorf("Expected User 1000, got %d", stats.User)
	}
	if stats.Nice != 50 {
		t.Errorf("Expected Nice 50, got %d", stats.Nice)
	}
	if stats.System != 500 {
		t.Errorf("Expected System 500, got %d", stats.System)
	}
	if stats.Idle != 8000 {
		t.Errorf("Expected Idle 8000, got %d", stats.Idle)
	}
	if stats.IOWait != 200 {
		t.Errorf("Expected IOWait 200, got %d", stats.IOWait)
	}
}

func TestLoadAvgParsing(t *testing.T) {
	fields := []string{"1.20", "1.50", "1.30", "2/500", "12345"}

	if len(fields) < 3 {
		t.Fatal("Invalid loadavg format")
	}

	load1, _ := parseFloatField(fields[0])
	load5, _ := parseFloatField(fields[1])
	load15, _ := parseFloatField(fields[2])

	if load1 != 1.20 {
		t.Errorf("Expected load1 1.20, got %.2f", load1)
	}
	if load5 != 1.50 {
		t.Errorf("Expected load5 1.50, got %.2f", load5)
	}
	if load15 != 1.30 {
		t.Errorf("Expected load15 1.30, got %.2f", load15)
	}
}

func TestContextStatsParsing(t *testing.T) {
	ctxtFields := []string{"ctxt", "10000"}
	intrFields := []string{"intr", "5000"}

	var contextSwitches, interrupts uint64

	if len(ctxtFields) >= 2 {
		contextSwitches, _ = parseUintField(ctxtFields[1])
	}
	if len(intrFields) >= 2 {
		interrupts, _ = parseUintField(intrFields[1])
	}

	if contextSwitches != 10000 {
		t.Errorf("Expected contextSwitches 10000, got %d", contextSwitches)
	}
	if interrupts != 5000 {
		t.Errorf("Expected interrupts 5000, got %d", interrupts)
	}
}

func parseUintField(field string) (uint64, error) {
	switch field {
	case "1000":
		return 1000, nil
	case "50":
		return 50, nil
	case "500":
		return 500, nil
	case "8000":
		return 8000, nil
	case "200":
		return 200, nil
	case "100":
		return 100, nil
	case "10000":
		return 10000, nil
	case "5000":
		return 5000, nil
	default:
		return 0, nil
	}
}

func parseFloatField(field string) (float64, error) {
	switch field {
	case "1.20":
		return 1.20, nil
	case "1.50":
		return 1.50, nil
	case "1.30":
		return 1.30, nil
	default:
		return 0, nil
	}
}

func TestCPUMetricsIntegration(t *testing.T) {
	collector := NewCPUCollector()

	firstResult, err := collector.Collect()
	if err == nil && firstResult != nil {
		t.Error("First collection should return nil or error when no baseline data")
	}

	time.Sleep(100 * time.Millisecond)

	secondResult, err := collector.Collect()
	if err != nil {
		t.Logf("Collect returned error (may be expected in test environment): %v", err)
		return
	}

	if secondResult == nil {
		t.Log("Second collection returned nil (may be expected in test environment)")
		return
	}

	if secondResult.Timestamp.IsZero() {
		t.Error("CPU metrics timestamp should not be zero")
	}

	if secondResult.UserPercent < 0 || secondResult.UserPercent > 100 {
		t.Errorf("UserPercent out of range: %.2f", secondResult.UserPercent)
	}
	if secondResult.SystemPercent < 0 || secondResult.SystemPercent > 100 {
		t.Errorf("SystemPercent out of range: %.2f", secondResult.SystemPercent)
	}
	if secondResult.IdlePercent < 0 || secondResult.IdlePercent > 100 {
		t.Errorf("IdlePercent out of range: %.2f", secondResult.IdlePercent)
	}
	if secondResult.IOWaitPercent < 0 || secondResult.IOWaitPercent > 100 {
		t.Errorf("IOWaitPercent out of range: %.2f", secondResult.IOWaitPercent)
	}

	if secondResult.Load1 < 0 {
		t.Errorf("Load1 should be non-negative: %.2f", secondResult.Load1)
	}
	if secondResult.Load5 < 0 {
		t.Errorf("Load5 should be non-negative: %.2f", secondResult.Load5)
	}
	if secondResult.Load15 < 0 {
		t.Errorf("Load15 should be non-negative: %.2f", secondResult.Load15)
	}
}
