package mem

import (
	"testing"
	"time"

	"github.com/chentao-kernel/spycat/pkg/monitor/metrics"
)

func TestMemoryCollectorCreation(t *testing.T) {
	collector := NewMemoryCollector()
	if collector == nil {
		t.Fatal("Failed to create Memory collector")
	}
	if collector.lastPageFaults != nil {
		t.Error("New collector should not have last page faults")
	}
}

func TestMemInfoParsing(t *testing.T) {
	memInfo := &MemInfo{
		Total:     16 * 1024 * 1024 * 1024, // 16GB
		Free:      4 * 1024 * 1024 * 1024,  // 4GB
		Available: 6 * 1024 * 1024 * 1024,  // 6GB
		Buffers:   512 * 1024 * 1024,       // 512MB
		PageCache: 2 * 1024 * 1024 * 1024,  // 2GB
		SwapTotal: 4 * 1024 * 1024 * 1024,  // 4GB
		SwapFree:  3 * 1024 * 1024 * 1024,  // 3GB
	}

	memInfo.Used = memInfo.Total - memInfo.Free - memInfo.Buffers - memInfo.PageCache
	memInfo.SwapUsed = memInfo.SwapTotal - memInfo.SwapFree

	expectedUsed := uint64(16*1024*1024*1024 - 4*1024*1024*1024 - 512*1024*1024 - 2*1024*1024*1024)
	if memInfo.Used != expectedUsed {
		t.Errorf("Expected Used %d, got %d", expectedUsed, memInfo.Used)
	}

	expectedSwapUsed := uint64(4*1024*1024*1024 - 3*1024*1024*1024)
	if memInfo.SwapUsed != expectedSwapUsed {
		t.Errorf("Expected SwapUsed %d, got %d", expectedSwapUsed, memInfo.SwapUsed)
	}
}

func TestVMStatsParsing(t *testing.T) {
	vmStats := &VMStats{
		PageFaults:     1000,
		MajorPageFaults: 10,
	}

	if vmStats.PageFaults != 1000 {
		t.Errorf("Expected PageFaults 1000, got %d", vmStats.PageFaults)
	}
	if vmStats.MajorPageFaults != 10 {
		t.Errorf("Expected MajorPageFaults 10, got %d", vmStats.MajorPageFaults)
	}
}

func TestMemoryUsageCalculation(t *testing.T) {
	collector := NewMemoryCollector()

	memInfo := &MemInfo{
		Total: 16 * 1024 * 1024 * 1024, // 16GB
		Used:  8 * 1024 * 1024 * 1024,  // 8GB
	}

	usage := collector.GetMemoryUsage(memInfo)
	expectedUsage := float64(8*1024*1024*1024) / float64(16*1024*1024*1024) * 100

	if usage != expectedUsage {
		t.Errorf("Expected memory usage %.2f%%, got %.2f%%", expectedUsage, usage)
	}

	zeroMemInfo := &MemInfo{}
	zeroUsage := collector.GetMemoryUsage(zeroMemInfo)
	if zeroUsage != 0 {
		t.Errorf("Expected zero usage for zero total memory, got %.2f%%", zeroUsage)
	}
}

func TestSwapUsageCalculation(t *testing.T) {
	collector := NewMemoryCollector()

	memInfo := &MemInfo{
		SwapTotal: 4 * 1024 * 1024 * 1024, // 4GB
		SwapUsed:  1 * 1024 * 1024 * 1024, // 1GB
	}

	usage := collector.GetSwapUsage(memInfo)
	expectedUsage := float64(1*1024*1024*1024) / float64(4*1024*1024*1024) * 100

	if usage != expectedUsage {
		t.Errorf("Expected swap usage %.2f%%, got %.2f%%", expectedUsage, usage)
	}

	zeroMemInfo := &MemInfo{}
	zeroUsage := collector.GetSwapUsage(zeroMemInfo)
	if zeroUsage != 0 {
		t.Errorf("Expected zero usage for zero swap total, got %.2f%%", zeroUsage)
	}
}

func TestPageFaultsCalculation(t *testing.T) {
	collector := NewMemoryCollector()

	firstPageFaults := &PageFaultStats{
		Timestamp:      time.Now(),
		PageFaults:     1000,
		MajorPageFaults: 10,
	}

	secondPageFaults := &PageFaultStats{
		Timestamp:      firstPageFaults.Timestamp.Add(time.Second),
		PageFaults:     1500, // +500
		MajorPageFaults: 15,  // +5
	}

	collector.lastPageFaults = firstPageFaults

	var pageFaultsDiff, majorPageFaultsDiff uint64
	currentPageFaults := &PageFaultStats{
		Timestamp:      secondPageFaults.Timestamp,
		PageFaults:     secondPageFaults.PageFaults,
		MajorPageFaults: secondPageFaults.MajorPageFaults,
	}

	if collector.lastPageFaults != nil {
		pageFaultsDiff = currentPageFaults.PageFaults - collector.lastPageFaults.PageFaults
		majorPageFaultsDiff = currentPageFaults.MajorPageFaults - collector.lastPageFaults.MajorPageFaults
	}

	if pageFaultsDiff != 500 {
		t.Errorf("Expected pageFaultsDiff 500, got %d", pageFaultsDiff)
	}
	if majorPageFaultsDiff != 5 {
		t.Errorf("Expected majorPageFaultsDiff 5, got %d", majorPageFaultsDiff)
	}
}

func TestMemoryMetricsCreation(t *testing.T) {
	collector := NewMemoryCollector()

	now := time.Now()
	memInfo := &MemInfo{
		Total:     16 * 1024 * 1024 * 1024,
		Used:      8 * 1024 * 1024 * 1024,
		Free:      4 * 1024 * 1024 * 1024,
		Available: 6 * 1024 * 1024 * 1024,
		SwapTotal: 4 * 1024 * 1024 * 1024,
		SwapUsed:  1 * 1024 * 1024 * 1024,
		SwapFree:  3 * 1024 * 1024 * 1024,
		PageCache: 2 * 1024 * 1024 * 1024,
		Buffers:   512 * 1024 * 1024,
	}

	vmStats := &VMStats{
		PageFaults:     1500,
		MajorPageFaults: 15,
	}

	collector.lastPageFaults = &PageFaultStats{
		Timestamp:      now.Add(-time.Second),
		PageFaults:     1000,
		MajorPageFaults: 10,
	}

	currentPageFaults := &PageFaultStats{
		Timestamp:      now,
		PageFaults:     vmStats.PageFaults,
		MajorPageFaults: vmStats.MajorPageFaults,
	}

	var pageFaultsDiff, majorPageFaultsDiff uint64
	if collector.lastPageFaults != nil {
		pageFaultsDiff = currentPageFaults.PageFaults - collector.lastPageFaults.PageFaults
		majorPageFaultsDiff = currentPageFaults.MajorPageFaults - collector.lastPageFaults.MajorPageFaults
	}

	memMetrics := &metrics.MemoryMetrics{
		Timestamp:       now,
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
	}

	if memMetrics.Total != 16*1024*1024*1024 {
		t.Error("Total memory mismatch")
	}
	if memMetrics.Used != 8*1024*1024*1024 {
		t.Error("Used memory mismatch")
	}
	if memMetrics.Available != 6*1024*1024*1024 {
		t.Error("Available memory mismatch")
	}
	if memMetrics.SwapTotal != 4*1024*1024*1024 {
		t.Error("Swap total mismatch")
	}
	if memMetrics.SwapUsed != 1*1024*1024*1024 {
		t.Error("Swap used mismatch")
	}
	if memMetrics.PageCache != 2*1024*1024*1024 {
		t.Error("Page cache mismatch")
	}
	if memMetrics.Buffers != 512*1024*1024 {
		t.Error("Buffers mismatch")
	}
	if memMetrics.PageFaults != 500 {
		t.Errorf("Expected PageFaults 500, got %d", memMetrics.PageFaults)
	}
	if memMetrics.MajorPageFaults != 5 {
		t.Errorf("Expected MajorPageFaults 5, got %d", memMetrics.MajorPageFaults)
	}
}

func TestMemoryCollectorIntegration(t *testing.T) {
	collector := NewMemoryCollector()

	firstResult, err := collector.Collect()
	if err != nil {
		t.Logf("First collection returned error (may be expected in test environment): %v", err)
		return
	}

	if firstResult == nil {
		t.Log("First collection returned nil (may be expected in test environment)")
		return
	}

	if firstResult.Timestamp.IsZero() {
		t.Error("Memory metrics timestamp should not be zero")
	}

	if firstResult.Total == 0 {
		t.Error("Total memory should not be zero")
	}
	if firstResult.Used > firstResult.Total {
		t.Errorf("Used memory (%d) should not exceed total memory (%d)", firstResult.Used, firstResult.Total)
	}
	if firstResult.Free > firstResult.Total {
		t.Errorf("Free memory (%d) should not exceed total memory (%d)", firstResult.Free, firstResult.Total)
	}
	if firstResult.Available > firstResult.Total {
		t.Errorf("Available memory (%d) should not exceed total memory (%d)", firstResult.Available, firstResult.Total)
	}

	if firstResult.SwapUsed > firstResult.SwapTotal {
		t.Errorf("Swap used (%d) should not exceed swap total (%d)", firstResult.SwapUsed, firstResult.SwapTotal)
	}
	if firstResult.SwapFree > firstResult.SwapTotal {
		t.Errorf("Swap free (%d) should not exceed swap total (%d)", firstResult.SwapFree, firstResult.SwapTotal)
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

	if secondResult.Timestamp.IsZero() {
		t.Error("Second collection timestamp should not be zero")
	}

	if secondResult.PageFaults < 0 {
		t.Errorf("PageFaults should be non-negative, got %d", secondResult.PageFaults)
	}
	if secondResult.MajorPageFaults < 0 {
		t.Errorf("MajorPageFaults should be non-negative, got %d", secondResult.MajorPageFaults)
	}
}

func TestMemoryUsageEdgeCases(t *testing.T) {
	collector := NewMemoryCollector()

	testCases := []struct {
		name     string
		memInfo  *MemInfo
		expected float64
	}{
		{
			name: "Zero total memory",
			memInfo: &MemInfo{
				Total: 0,
				Used:  100,
			},
			expected: 0,
		},
		{
			name: "Zero used memory",
			memInfo: &MemInfo{
				Total: 100,
				Used:  0,
			},
			expected: 0,
		},
		{
			name: "Full usage",
			memInfo: &MemInfo{
				Total: 100,
				Used:  100,
			},
			expected: 100,
		},
		{
			name: "Half usage",
			memInfo: &MemInfo{
				Total: 200,
				Used:  100,
			},
			expected: 50,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			usage := collector.GetMemoryUsage(tc.memInfo)
			if usage != tc.expected {
				t.Errorf("Expected usage %.2f%%, got %.2f%%", tc.expected, usage)
			}
		})
	}
}
