package collector

import (
	"time"

	"github.com/chentao-kernel/spycat/pkg/monitor/cpu"
	"github.com/chentao-kernel/spycat/pkg/monitor/io"
	"github.com/chentao-kernel/spycat/pkg/monitor/mem"
	"github.com/chentao-kernel/spycat/pkg/monitor/metrics"
	"github.com/chentao-kernel/spycat/pkg/monitor/net"
)

type SystemCollector struct {
	cpuCollector       *cpu.CPUCollector
	memCollector       *mem.MemoryCollector
	ioCollector        *io.IOCollector
	netCollector       *net.NetworkCollector
	collectionInterval time.Duration
}

type SystemMetrics struct {
	Timestamp  time.Time                 `json:"timestamp"`
	CPUMetrics *metrics.CPUMetrics       `json:"cpu_metrics,omitempty"`
	MemMetrics *metrics.MemoryMetrics    `json:"mem_metrics,omitempty"`
	IOMetrics  []*metrics.IOMetrics      `json:"io_metrics,omitempty"`
	NetMetrics []*metrics.NetworkMetrics `json:"net_metrics,omitempty"`
}

func NewSystemCollector(interval time.Duration) *SystemCollector {
	return &SystemCollector{
		cpuCollector:       cpu.NewCPUCollector(),
		memCollector:       mem.NewMemoryCollector(),
		ioCollector:        io.NewIOCollector(),
		netCollector:       net.NewNetworkCollector(),
		collectionInterval: interval,
	}
}

func (s *SystemCollector) Collect() (*SystemMetrics, error) {
	timestamp := time.Now()
	systemMetrics := &SystemMetrics{
		Timestamp: timestamp,
	}

	cpuMetrics, err := s.cpuCollector.Collect()
	if err != nil {
		return nil, err
	}
	systemMetrics.CPUMetrics = cpuMetrics

	memMetrics, err := s.memCollector.Collect()
	if err != nil {
		return nil, err
	}
	systemMetrics.MemMetrics = memMetrics

	ioMetrics, err := s.ioCollector.Collect()
	if err != nil {
		return nil, err
	}
	systemMetrics.IOMetrics = ioMetrics

	netMetrics, err := s.netCollector.Collect()
	if err != nil {
		return nil, err
	}
	systemMetrics.NetMetrics = netMetrics

	return systemMetrics, nil
}

func (s *SystemCollector) CollectCPU() (*metrics.CPUMetrics, error) {
	return s.cpuCollector.Collect()
}

func (s *SystemCollector) CollectMemory() (*metrics.MemoryMetrics, error) {
	return s.memCollector.Collect()
}

func (s *SystemCollector) CollectIO() ([]*metrics.IOMetrics, error) {
	return s.ioCollector.Collect()
}

func (s *SystemCollector) CollectNetwork() ([]*metrics.NetworkMetrics, error) {
	return s.netCollector.Collect()
}

func (s *SystemCollector) GetCollectionInterval() time.Duration {
	return s.collectionInterval
}

func (s *SystemCollector) SetCollectionInterval(interval time.Duration) {
	s.collectionInterval = interval
}

func (s *SystemCollector) GetTotalIO(ioMetrics []*metrics.IOMetrics) *metrics.IOMetrics {
	return s.ioCollector.GetTotalIO(ioMetrics)
}

func (s *SystemCollector) GetTotalNetwork(netMetrics []*metrics.NetworkMetrics) *metrics.NetworkMetrics {
	return s.netCollector.GetTotalNetwork(netMetrics)
}

func (s *SystemCollector) GetIODeviceList() ([]string, error) {
	return s.ioCollector.GetDeviceList()
}

func (s *SystemCollector) GetNetworkInterfaceList() ([]string, error) {
	return s.netCollector.GetInterfaceList()
}

func (s *SystemCollector) Summary(metrics *SystemMetrics) *SystemSummary {
	summary := &SystemSummary{
		Timestamp: metrics.Timestamp,
	}

	if metrics.CPUMetrics != nil {
		summary.CPUUsage = metrics.CPUMetrics.UserPercent + metrics.CPUMetrics.SystemPercent
		summary.LoadAverage = metrics.CPUMetrics.Load1
	}

	if metrics.MemMetrics != nil {
		summary.MemoryUsage = float64(metrics.MemMetrics.Used) / float64(metrics.MemMetrics.Total) * 100
		summary.SwapUsage = float64(metrics.MemMetrics.SwapUsed) / float64(metrics.MemMetrics.SwapTotal) * 100
	}

	if len(metrics.IOMetrics) > 0 {
		totalIO := s.GetTotalIO(metrics.IOMetrics)
		summary.IOUtilization = totalIO.Utilization
		summary.TotalIOPS = totalIO.ReadIOPS + totalIO.WriteIOPS
	}

	if len(metrics.NetMetrics) > 0 {
		totalNet := s.GetTotalNetwork(metrics.NetMetrics)
		summary.NetworkRxBytes = totalNet.RxBytes
		summary.NetworkTxBytes = totalNet.TxBytes
	}

	return summary
}

type SystemSummary struct {
	Timestamp      time.Time `json:"timestamp"`
	CPUUsage       float64   `json:"cpu_usage"`
	MemoryUsage    float64   `json:"memory_usage"`
	SwapUsage      float64   `json:"swap_usage"`
	LoadAverage    float64   `json:"load_average"`
	IOUtilization  float64   `json:"io_utilization"`
	TotalIOPS      float64   `json:"total_iops"`
	NetworkRxBytes uint64    `json:"network_rx_bytes"`
	NetworkTxBytes uint64    `json:"network_tx_bytes"`
}
