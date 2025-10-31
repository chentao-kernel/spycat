package metrics

import (
	"testing"
	"time"
)

func TestMetricTypes(t *testing.T) {
	tests := []struct {
		name     string
		metricType MetricType
		expected string
	}{
		{"Counter", Counter, "counter"},
		{"Gauge", Gauge, "gauge"},
		{"Histogram", Histogram, "histogram"},
		{"Summary", Summary, "summary"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.metricType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.metricType)
			}
		})
	}
}

func TestMetricCreation(t *testing.T) {
	now := time.Now()
	metric := Metric{
		Name:      "test_metric",
		Type:      Gauge,
		Value:     42.5,
		Timestamp: now,
		Labels: map[string]string{
			"host": "localhost",
			"env":  "test",
		},
	}

	if metric.Name != "test_metric" {
		t.Errorf("Expected name 'test_metric', got '%s'", metric.Name)
	}
	if metric.Type != Gauge {
		t.Errorf("Expected type Gauge, got %v", metric.Type)
	}
	if metric.Value != 42.5 {
		t.Errorf("Expected value 42.5, got %f", metric.Value)
	}
	if !metric.Timestamp.Equal(now) {
		t.Errorf("Timestamp mismatch")
	}
	if len(metric.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(metric.Labels))
	}
}

func TestCPUMetrics(t *testing.T) {
	now := time.Now()
	cpuMetrics := CPUMetrics{
		Timestamp:     now,
		UserPercent:   25.5,
		SystemPercent: 15.2,
		IdlePercent:   59.3,
		IOWaitPercent: 0.0,
		Load1:         1.2,
		Load5:         1.5,
		Load15:        1.3,
		ContextSwitch: 1000,
		Interrupts:    500,
	}

	if !cpuMetrics.Timestamp.Equal(now) {
		t.Error("CPU metrics timestamp mismatch")
	}
	if cpuMetrics.UserPercent != 25.5 {
		t.Errorf("Expected UserPercent 25.5, got %f", cpuMetrics.UserPercent)
	}
	if cpuMetrics.SystemPercent != 15.2 {
		t.Errorf("Expected SystemPercent 15.2, got %f", cpuMetrics.SystemPercent)
	}
	if cpuMetrics.IdlePercent != 59.3 {
		t.Errorf("Expected IdlePercent 59.3, got %f", cpuMetrics.IdlePercent)
	}
	if cpuMetrics.Load1 != 1.2 {
		t.Errorf("Expected Load1 1.2, got %f", cpuMetrics.Load1)
	}
	if cpuMetrics.ContextSwitch != 1000 {
		t.Errorf("Expected ContextSwitch 1000, got %d", cpuMetrics.ContextSwitch)
	}
}

func TestMemoryMetrics(t *testing.T) {
	now := time.Now()
	memMetrics := MemoryMetrics{
		Timestamp:       now,
		Total:          16 * 1024 * 1024 * 1024, // 16GB
		Used:           8 * 1024 * 1024 * 1024,  // 8GB
		Free:           4 * 1024 * 1024 * 1024,  // 4GB
		Available:      6 * 1024 * 1024 * 1024,  // 6GB
		SwapTotal:      4 * 1024 * 1024 * 1024,  // 4GB
		SwapUsed:       1 * 1024 * 1024 * 1024,  // 1GB
		SwapFree:       3 * 1024 * 1024 * 1024,  // 3GB
		PageCache:      2 * 1024 * 1024 * 1024,  // 2GB
		Buffers:        512 * 1024 * 1024,       // 512MB
		PageFaults:     1000,
		MajorPageFaults: 10,
	}

	if !memMetrics.Timestamp.Equal(now) {
		t.Error("Memory metrics timestamp mismatch")
	}
	if memMetrics.Total != 16*1024*1024*1024 {
		t.Errorf("Total memory mismatch")
	}
	if memMetrics.Used != 8*1024*1024*1024 {
		t.Errorf("Used memory mismatch")
	}
	if memMetrics.PageFaults != 1000 {
		t.Errorf("Expected PageFaults 1000, got %d", memMetrics.PageFaults)
	}
	if memMetrics.MajorPageFaults != 10 {
		t.Errorf("Expected MajorPageFaults 10, got %d", memMetrics.MajorPageFaults)
	}
}

func TestIOMetrics(t *testing.T) {
	now := time.Now()
	ioMetrics := IOMetrics{
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
		t.Errorf("Expected ReadIOPS 150.5, got %f", ioMetrics.ReadIOPS)
	}
	if ioMetrics.WriteIOPS != 200.3 {
		t.Errorf("Expected WriteIOPS 200.3, got %f", ioMetrics.WriteIOPS)
	}
	if ioMetrics.ReadBytes != 1024*1024 {
		t.Errorf("ReadBytes mismatch")
	}
	if ioMetrics.Utilization != 45.7 {
		t.Errorf("Expected Utilization 45.7, got %f", ioMetrics.Utilization)
	}
}

func TestNetworkMetrics(t *testing.T) {
	now := time.Now()
	netMetrics := NetworkMetrics{
		Timestamp:     now,
		Interface:     "eth0",
		RxBytes:       1024 * 1024,  // 1MB
		TxBytes:       512 * 1024,   // 512KB
		RxPackets:     1000,
		TxPackets:     800,
		RxErrors:      5,
		TxErrors:      2,
		RxDropped:     3,
		TxDropped:     1,
		Connections:   150,
		TCPEstablished: 120,
	}

	if !netMetrics.Timestamp.Equal(now) {
		t.Error("Network metrics timestamp mismatch")
	}
	if netMetrics.Interface != "eth0" {
		t.Errorf("Expected interface 'eth0', got '%s'", netMetrics.Interface)
	}
	if netMetrics.RxBytes != 1024*1024 {
		t.Errorf("RxBytes mismatch")
	}
	if netMetrics.TxBytes != 512*1024 {
		t.Errorf("TxBytes mismatch")
	}
	if netMetrics.Connections != 150 {
		t.Errorf("Expected Connections 150, got %d", netMetrics.Connections)
	}
	if netMetrics.TCPEstablished != 120 {
		t.Errorf("Expected TCPEstablished 120, got %d", netMetrics.TCPEstablished)
	}
}

func TestSystemMetrics(t *testing.T) {
	now := time.Now()
	sysMetrics := SystemMetrics{
		Timestamp:    now,
		Uptime:       86400, // 1 day
		Processes:    250,
		RunningProcs: 5,
		BlockedProcs: 2,
	}

	if !sysMetrics.Timestamp.Equal(now) {
		t.Error("System metrics timestamp mismatch")
	}
	if sysMetrics.Uptime != 86400 {
		t.Errorf("Expected Uptime 86400, got %d", sysMetrics.Uptime)
	}
	if sysMetrics.Processes != 250 {
		t.Errorf("Expected Processes 250, got %d", sysMetrics.Processes)
	}
	if sysMetrics.RunningProcs != 5 {
		t.Errorf("Expected RunningProcs 5, got %d", sysMetrics.RunningProcs)
	}
	if sysMetrics.BlockedProcs != 2 {
		t.Errorf("Expected BlockedProcs 2, got %d", sysMetrics.BlockedProcs)
	}
}
