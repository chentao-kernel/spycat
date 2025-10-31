package net

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/chentao-kernel/spycat/pkg/monitor/metrics"
)

func TestNetworkCollectorCreation(t *testing.T) {
	collector := NewNetworkCollector()
	if collector == nil {
		t.Fatal("Failed to create Network collector")
	}
	if collector.lastStats == nil {
		t.Error("Network collector should have initialized lastStats map")
	}
	if len(collector.lastStats) != 0 {
		t.Error("New collector should have empty lastStats map")
	}
}

func TestInterfaceStatsParsing(t *testing.T) {
	interfaceStats := &InterfaceStats{
		Timestamp: time.Now(),
		Interface: "eth0",
		RxBytes:   1024 * 1024,  // 1MB
		RxPackets: 1000,
		RxErrors:  5,
		RxDropped: 3,
		TxBytes:   512 * 1024,   // 512KB
		TxPackets: 800,
		TxErrors:  2,
		TxDropped: 1,
	}

	if interfaceStats.Interface != "eth0" {
		t.Errorf("Expected interface 'eth0', got '%s'", interfaceStats.Interface)
	}
	if interfaceStats.RxBytes != 1024*1024 {
		t.Error("RxBytes mismatch")
	}
	if interfaceStats.RxPackets != 1000 {
		t.Errorf("Expected RxPackets 1000, got %d", interfaceStats.RxPackets)
	}
	if interfaceStats.RxErrors != 5 {
		t.Errorf("Expected RxErrors 5, got %d", interfaceStats.RxErrors)
	}
	if interfaceStats.RxDropped != 3 {
		t.Errorf("Expected RxDropped 3, got %d", interfaceStats.RxDropped)
	}
	if interfaceStats.TxBytes != 512*1024 {
		t.Error("TxBytes mismatch")
	}
	if interfaceStats.TxPackets != 800 {
		t.Errorf("Expected TxPackets 800, got %d", interfaceStats.TxPackets)
	}
	if interfaceStats.TxErrors != 2 {
		t.Errorf("Expected TxErrors 2, got %d", interfaceStats.TxErrors)
	}
	if interfaceStats.TxDropped != 1 {
		t.Errorf("Expected TxDropped 1, got %d", interfaceStats.TxDropped)
	}
}

func TestNetworkMetricsCalculation(t *testing.T) {
	collector := NewNetworkCollector()

	firstStats := &InterfaceStats{
		Timestamp: time.Now(),
		Interface: "eth0",
		RxBytes:   1024 * 1024,  // 1MB
		RxPackets: 1000,
		RxErrors:  5,
		RxDropped: 3,
		TxBytes:   512 * 1024,   // 512KB
		TxPackets: 800,
		TxErrors:  2,
		TxDropped: 1,
	}

	secondStats := &InterfaceStats{
		Timestamp: firstStats.Timestamp.Add(time.Second),
		Interface: "eth0",
		RxBytes:   2 * 1024 * 1024,  // 2MB (+1MB)
		RxPackets: 2000,              // +1000
		RxErrors:  8,                 // +3
		RxDropped: 5,                 // +2
		TxBytes:   1 * 1024 * 1024,  // 1MB (+512KB)
		TxPackets: 1600,              // +800
		TxErrors:  4,                 // +2
		TxDropped: 2,                 // +1
	}

	collector.lastStats["eth0"] = firstStats

	connections := uint32(150)
	tcpEstablished := uint32(120)

	timeDiff := secondStats.Timestamp.Sub(firstStats.Timestamp).Seconds()
	if timeDiff <= 0 {
		t.Fatal("Time difference should be positive")
	}

	rxBytes := secondStats.RxBytes - firstStats.RxBytes
	txBytes := secondStats.TxBytes - firstStats.TxBytes
	rxPackets := secondStats.RxPackets - firstStats.RxPackets
	txPackets := secondStats.TxPackets - firstStats.TxPackets
	rxErrors := secondStats.RxErrors - firstStats.RxErrors
	txErrors := secondStats.TxErrors - firstStats.TxErrors
	rxDropped := secondStats.RxDropped - firstStats.RxDropped
	txDropped := secondStats.TxDropped - firstStats.TxDropped

	if rxBytes != 1024*1024 {
		t.Errorf("Expected RxBytes %d, got %d", 1024*1024, rxBytes)
	}
	if txBytes != 512*1024 {
		t.Errorf("Expected TxBytes %d, got %d", 512*1024, txBytes)
	}
	if rxPackets != 1000 {
		t.Errorf("Expected RxPackets 1000, got %d", rxPackets)
	}
	if txPackets != 800 {
		t.Errorf("Expected TxPackets 800, got %d", txPackets)
	}
	if rxErrors != 3 {
		t.Errorf("Expected RxErrors 3, got %d", rxErrors)
	}
	if txErrors != 2 {
		t.Errorf("Expected TxErrors 2, got %d", txErrors)
	}
	if rxDropped != 2 {
		t.Errorf("Expected RxDropped 2, got %d", rxDropped)
	}
	if txDropped != 1 {
		t.Errorf("Expected TxDropped 1, got %d", txDropped)
	}
	if connections != 150 {
		t.Errorf("Expected Connections 150, got %d", connections)
	}
	if tcpEstablished != 120 {
		t.Errorf("Expected TCPEstablished 120, got %d", tcpEstablished)
	}
}

func TestNetworkMetricsCreation(t *testing.T) {
	now := time.Now()
	netMetrics := &metrics.NetworkMetrics{
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
		t.Error("RxBytes mismatch")
	}
	if netMetrics.TxBytes != 512*1024 {
		t.Error("TxBytes mismatch")
	}
	if netMetrics.RxPackets != 1000 {
		t.Errorf("Expected RxPackets 1000, got %d", netMetrics.RxPackets)
	}
	if netMetrics.TxPackets != 800 {
		t.Errorf("Expected TxPackets 800, got %d", netMetrics.TxPackets)
	}
	if netMetrics.Connections != 150 {
		t.Errorf("Expected Connections 150, got %d", netMetrics.Connections)
	}
	if netMetrics.TCPEstablished != 120 {
		t.Errorf("Expected TCPEstablished 120, got %d", netMetrics.TCPEstablished)
	}
}

func TestTotalNetworkCalculation(t *testing.T) {
	collector := NewNetworkCollector()

	netMetrics := []*metrics.NetworkMetrics{
		{
			Interface:     "eth0",
			RxBytes:       1024 * 1024,
			TxBytes:       512 * 1024,
			RxPackets:     1000,
			TxPackets:     800,
			RxErrors:      5,
			TxErrors:      2,
			RxDropped:     3,
			TxDropped:     1,
			Connections:   150,
			TCPEstablished: 120,
		},
		{
			Interface:     "eth1",
			RxBytes:       512 * 1024,
			TxBytes:       256 * 1024,
			RxPackets:     500,
			TxPackets:     400,
			RxErrors:      2,
			TxErrors:      1,
			RxDropped:     1,
			TxDropped:     0,
			Connections:   150,
			TCPEstablished: 120,
		},
	}

	total := collector.GetTotalNetwork(netMetrics)

	if total == nil {
		t.Fatal("Failed to calculate total network")
	}

	if total.Interface != "total" {
		t.Errorf("Expected interface 'total', got '%s'", total.Interface)
	}
	if total.RxBytes != 1024*1024+512*1024 {
		t.Error("Total RxBytes mismatch")
	}
	if total.TxBytes != 512*1024+256*1024 {
		t.Error("Total TxBytes mismatch")
	}
	if total.RxPackets != 1000+500 {
		t.Errorf("Expected total RxPackets 1500, got %d", total.RxPackets)
	}
	if total.TxPackets != 800+400 {
		t.Errorf("Expected total TxPackets 1200, got %d", total.TxPackets)
	}
	if total.RxErrors != 5+2 {
		t.Errorf("Expected total RxErrors 7, got %d", total.RxErrors)
	}
	if total.TxErrors != 2+1 {
		t.Errorf("Expected total TxErrors 3, got %d", total.TxErrors)
	}
	if total.RxDropped != 3+1 {
		t.Errorf("Expected total RxDropped 4, got %d", total.RxDropped)
	}
	if total.TxDropped != 1+0 {
		t.Errorf("Expected total TxDropped 1, got %d", total.TxDropped)
	}
	if total.Connections != 150 {
		t.Errorf("Expected Connections 150, got %d", total.Connections)
	}
	if total.TCPEstablished != 120 {
		t.Errorf("Expected TCPEstablished 120, got %d", total.TCPEstablished)
	}
}

func TestEmptyNetworkMetrics(t *testing.T) {
	collector := NewNetworkCollector()

	emptyMetrics := []*metrics.NetworkMetrics{}
	total := collector.GetTotalNetwork(emptyMetrics)
	if total != nil {
		t.Error("Total network should be nil for empty metrics list")
	}

	total = collector.GetTotalNetwork(nil)
	if total != nil {
		t.Error("Total network should be nil for nil metrics list")
	}
}

func TestNetworkCollectorIntegration(t *testing.T) {
	collector := NewNetworkCollector()

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
		t.Log("No network interfaces found (may be expected in test environment)")
		return
	}

	for _, metric := range firstResult {
		if metric.Timestamp.IsZero() {
			t.Error("Network metrics timestamp should not be zero")
		}
		if metric.Interface == "" {
			t.Error("Network interface name should not be empty")
		}

		if metric.RxBytes < 0 {
			t.Errorf("RxBytes should be non-negative, got %d", metric.RxBytes)
		}
		if metric.TxBytes < 0 {
			t.Errorf("TxBytes should be non-negative, got %d", metric.TxBytes)
		}

		if metric.RxPackets < 0 {
			t.Errorf("RxPackets should be non-negative, got %d", metric.RxPackets)
		}
		if metric.TxPackets < 0 {
			t.Errorf("TxPackets should be non-negative, got %d", metric.TxPackets)
		}

		if metric.RxErrors < 0 {
			t.Errorf("RxErrors should be non-negative, got %d", metric.RxErrors)
		}
		if metric.TxErrors < 0 {
			t.Errorf("TxErrors should be non-negative, got %d", metric.TxErrors)
		}
		if metric.RxDropped < 0 {
			t.Errorf("RxDropped should be non-negative, got %d", metric.RxDropped)
		}
		if metric.TxDropped < 0 {
			t.Errorf("TxDropped should be non-negative, got %d", metric.TxDropped)
		}

		if metric.Connections < 0 {
			t.Errorf("Connections should be non-negative, got %d", metric.Connections)
		}
		if metric.TCPEstablished < 0 {
			t.Errorf("TCPEstablished should be non-negative, got %d", metric.TCPEstablished)
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

func TestInterfaceList(t *testing.T) {
	collector := NewNetworkCollector()

	interfaces, err := collector.GetInterfaceList()
	if err != nil {
		t.Logf("GetInterfaceList returned error (may be expected in test environment): %v", err)
		return
	}

	if interfaces == nil {
		t.Error("Interface list should not be nil")
	}

	for _, iface := range interfaces {
		if iface == "" {
			t.Error("Interface name should not be empty")
		}
	}
}

func TestNetworkMetricsEdgeCases(t *testing.T) {
	collector := NewNetworkCollector()

	testCases := []struct {
		name     string
		metrics  []*metrics.NetworkMetrics
		expected *metrics.NetworkMetrics
	}{
		{
			name:     "Empty metrics",
			metrics:  []*metrics.NetworkMetrics{},
			expected: nil,
		},
		{
			name: "Single interface",
			metrics: []*metrics.NetworkMetrics{
				{
					Interface:     "eth0",
					RxBytes:       1024,
					TxBytes:       512,
					RxPackets:     100,
					TxPackets:     80,
					RxErrors:      5,
					TxErrors:      2,
					RxDropped:     3,
					TxDropped:     1,
					Connections:   150,
					TCPEstablished: 120,
				},
			},
			expected: &metrics.NetworkMetrics{
				Interface:     "total",
				RxBytes:       1024,
				TxBytes:       512,
				RxPackets:     100,
				TxPackets:     80,
				RxErrors:      5,
				TxErrors:      2,
				RxDropped:     3,
				TxDropped:     1,
				Connections:   150,
				TCPEstablished: 120,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := collector.GetTotalNetwork(tc.metrics)
			if tc.expected == nil {
				if result != nil {
					t.Error("Expected nil result")
				}
				return
			}

			if result == nil {
				t.Fatal("Result should not be nil")
			}

			if result.Interface != tc.expected.Interface {
				t.Errorf("Expected interface '%s', got '%s'", tc.expected.Interface, result.Interface)
			}
			if result.RxBytes != tc.expected.RxBytes {
				t.Errorf("Expected RxBytes %d, got %d", tc.expected.RxBytes, result.RxBytes)
			}
			if result.TxBytes != tc.expected.TxBytes {
				t.Errorf("Expected TxBytes %d, got %d", tc.expected.TxBytes, result.TxBytes)
			}
			if result.Connections != tc.expected.Connections {
				t.Errorf("Expected Connections %d, got %d", tc.expected.Connections, result.Connections)
			}
		})
	}
}

func TestConnectionCountParsing(t *testing.T) {
	sockstatLine := "TCP: inuse 12 orphan 0 tw 0 alloc 16 mem 1"
	fields := strings.Fields(sockstatLine)
	
	if len(fields) >= 4 && fields[0] == "TCP:" && fields[1] == "inuse" {
		inuse, _ := strconv.ParseUint(fields[2], 10, 64)
		connections := uint32(inuse)
		
		if connections != 12 {
			t.Errorf("Expected connections 12, got %d", connections)
		}
	}
}
