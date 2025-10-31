package net

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chentao-kernel/spycat/pkg/monitor/metrics"
)

// NetworkCollector
type NetworkCollector struct {
	lastStats map[string]*InterfaceStats
}

// InterfaceStats
type InterfaceStats struct {
	Timestamp time.Time
	Interface string
	RxBytes   uint64
	RxPackets uint64
	RxErrors  uint64
	RxDropped uint64
	TxBytes   uint64
	TxPackets uint64
	TxErrors  uint64
	TxDropped uint64
}

func NewNetworkCollector() *NetworkCollector {
	return &NetworkCollector{
		lastStats: make(map[string]*InterfaceStats),
	}
}

func (n *NetworkCollector) Collect() ([]*metrics.NetworkMetrics, error) {
	currentStats, err := n.readNetDev()
	if err != nil {
		return nil, err
	}

	connections, err := n.getConnectionCount()
	if err != nil {
		return nil, err
	}

	tcpEstablished, err := n.getTCPEstablishedCount()
	if err != nil {
		return nil, err
	}

	var netMetrics []*metrics.NetworkMetrics

	for iface, current := range currentStats {
		if last, exists := n.lastStats[iface]; exists {
			// calculate the increment
			timeDiff := current.Timestamp.Sub(last.Timestamp).Seconds()
			if timeDiff <= 0 {
				continue
			}

			// calculate the bandwidth (bytes/s)
			rxBytes := current.RxBytes - last.RxBytes
			txBytes := current.TxBytes - last.TxBytes

			// calculate the packet rate (packets/s)
			rxPackets := current.RxPackets - last.RxPackets
			txPackets := current.TxPackets - last.TxPackets

			// increment of errors and packet loss
			rxErrors := current.RxErrors - last.RxErrors
			txErrors := current.TxErrors - last.TxErrors
			rxDropped := current.RxDropped - last.RxDropped
			txDropped := current.TxDropped - last.TxDropped

			netMetric := &metrics.NetworkMetrics{
				Timestamp:     current.Timestamp,
				Interface:     iface,
				RxBytes:       rxBytes,
				TxBytes:       txBytes,
				RxPackets:     rxPackets,
				TxPackets:     txPackets,
				RxErrors:      rxErrors,
				TxErrors:      txErrors,
				RxDropped:     rxDropped,
				TxDropped:     txDropped,
				Connections:   connections,
				TCPEstablished: tcpEstablished,
			}

			netMetrics = append(netMetrics, netMetric)
		}

		// update last stat
		n.lastStats[iface] = current
	}

	return netMetrics, nil
}

func (n *NetworkCollector) readNetDev() (map[string]*InterfaceStats, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/net/dev: %v", err)
	}
	defer file.Close()

	stats := make(map[string]*InterfaceStats)
	scanner := bufio.NewScanner(file)

	// skip the first two lines (header lines)
	for i := 0; i < 2; i++ {
		scanner.Scan()
	}

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 17 {
			continue
		}

		// interface name (with the colon removed)
		iface := strings.TrimSuffix(fields[0], ":")

		// skip the loopback interface
		if iface == "lo" {
			continue
		}

		// receive statistics
		rxBytes, _ := strconv.ParseUint(fields[1], 10, 64)
		rxPackets, _ := strconv.ParseUint(fields[2], 10, 64)
		rxErrors, _ := strconv.ParseUint(fields[3], 10, 64)
		rxDropped, _ := strconv.ParseUint(fields[4], 10, 64)

		// send statistics
		txBytes, _ := strconv.ParseUint(fields[9], 10, 64)
		txPackets, _ := strconv.ParseUint(fields[10], 10, 64)
		txErrors, _ := strconv.ParseUint(fields[11], 10, 64)
		txDropped, _ := strconv.ParseUint(fields[12], 10, 64)

		stats[iface] = &InterfaceStats{
			Timestamp: time.Now(),
			Interface: iface,
			RxBytes:   rxBytes,
			RxPackets: rxPackets,
			RxErrors:  rxErrors,
			RxDropped: rxDropped,
			TxBytes:   txBytes,
			TxPackets: txPackets,
			TxErrors:  txErrors,
			TxDropped: txDropped,
		}
	}

	return stats, nil
}

func (n *NetworkCollector) getConnectionCount() (uint32, error) {
	file, err := os.Open("/proc/net/sockstat")
	if err != nil {
		return 0, fmt.Errorf("failed to open /proc/net/sockstat: %v", err)
	}
	defer file.Close()

	var connections uint32
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "TCP: inuse") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				// TCP: inuse 12 orphan 0 tw 0 alloc 16 mem 1
				inuse, _ := strconv.ParseUint(fields[2], 10, 64)
				connections = uint32(inuse)
				break
			}
		}
	}

	return connections, nil
}

func (n *NetworkCollector) getTCPEstablishedCount() (uint32, error) {
	file, err := os.Open("/proc/net/tcp")
	if err != nil {
		return 0, fmt.Errorf("failed to open /proc/net/tcp: %v", err)
	}
	defer file.Close()

	var established uint32
	scanner := bufio.NewScanner(file)

	// skip title line
	if scanner.Scan() {
	}

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			// check the connection status (the 4th field)
			// 01 = ESTABLISHED
			if fields[3] == "01" {
				established++
			}
		}
	}

	return established, nil
}

func (n *NetworkCollector) GetInterfaceList() ([]string, error) {
	stats, err := n.readNetDev()
	if err != nil {
		return nil, err
	}

	var interfaces []string
	for iface := range stats {
		interfaces = append(interfaces, iface)
	}

	return interfaces, nil
}

func (n *NetworkCollector) GetTotalNetwork(netMetrics []*metrics.NetworkMetrics) *metrics.NetworkMetrics {
	if len(netMetrics) == 0 {
		return nil
	}

	total := &metrics.NetworkMetrics{
		Timestamp: time.Now(),
		Interface: "total",
	}

	for _, metric := range netMetrics {
		total.RxBytes += metric.RxBytes
		total.TxBytes += metric.TxBytes
		total.RxPackets += metric.RxPackets
		total.TxPackets += metric.TxPackets
		total.RxErrors += metric.RxErrors
		total.TxErrors += metric.TxErrors
		total.RxDropped += metric.RxDropped
		total.TxDropped += metric.TxDropped
	}

	// use the number of connections from the first metric
	// (the number of connections is the same across all interfaces)
	if len(netMetrics) > 0 {
		total.Connections = netMetrics[0].Connections
		total.TCPEstablished = netMetrics[0].TCPEstablished
	}

	return total
}
