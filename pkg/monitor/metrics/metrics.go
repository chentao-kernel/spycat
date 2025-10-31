package metrics

import (
	"time"
)

type MetricType string

const (
	Counter   MetricType = "counter"
	Gauge     MetricType = "gauge"
	Histogram MetricType = "histogram"
	Summary   MetricType = "summary"
)

type Metric struct {
	Name      string            `json:"name"`
	Type      MetricType        `json:"type"`
	Value     float64           `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// CPUMetrics
type CPUMetrics struct {
	Timestamp     time.Time `json:"timestamp"`
	UserPercent   float64   `json:"user_percent"`    // User space CPU usage
	SystemPercent float64   `json:"system_percent"`  // Kernel space CPU usage
	IdlePercent   float64   `json:"idle_percent"`    // Idle CPU usage
	IOWaitPercent float64   `json:"io_wait_percent"` // I/O wait CPU usage
	Load1         float64   `json:"load_1"`          // 1-minute load average
	Load5         float64   `json:"load_5"`          // 5-minute load average
	Load15        float64   `json:"load_15"`         // 15-minute load average
	ContextSwitch uint64    `json:"context_switch"`  // Context switch count
	Interrupts    uint64    `json:"interrupts"`      // Interrupt count
}

type MemoryMetrics struct {
	Timestamp      time.Time `json:"timestamp"`
	Total          uint64    `json:"total"`           // Total memory (bytes)
	Used           uint64    `json:"used"`            // Used memory (bytes)
	Free           uint64    `json:"free"`            // Free memory (bytes)
	Available      uint64    `json:"available"`       // Available memory (bytes)
	SwapTotal      uint64    `json:"swap_total"`      // Total swap space (bytes)
	SwapUsed       uint64    `json:"swap_used"`       // Used swap space (bytes)
	SwapFree       uint64    `json:"swap_free"`       // Free swap space (bytes)
	PageCache      uint64    `json:"page_cache"`      // Page cache (bytes)
	Buffers        uint64    `json:"buffers"`         // Buffers (bytes)
	PageFaults     uint64    `json:"page_faults"`     // Page fault count
	MajorPageFaults uint64   `json:"major_page_faults"` // Major page fault count
}

type IOMetrics struct {
	Timestamp    time.Time `json:"timestamp"`
	Device       string    `json:"device"`        // Device name
	ReadIOPS     float64   `json:"read_iops"`     // Read IOPS
	WriteIOPS    float64   `json:"write_iops"`    // Write IOPS
	ReadBytes    uint64    `json:"read_bytes"`    // Read bytes
	WriteBytes   uint64    `json:"write_bytes"`   // Write bytes
	ReadLatency  float64   `json:"read_latency"`  // Read latency (ms)
	WriteLatency float64   `json:"write_latency"` // Write latency (ms)
	QueueDepth   uint32    `json:"queue_depth"`   // Queue depth
	Utilization  float64   `json:"utilization"`   // Utilization (%)
}

type NetworkMetrics struct {
	Timestamp     time.Time `json:"timestamp"`
	Interface     string    `json:"interface"`      // Network interface
	RxBytes       uint64    `json:"rx_bytes"`       // Received bytes
	TxBytes       uint64    `json:"tx_bytes"`       // Transmitted bytes
	RxPackets     uint64    `json:"rx_packets"`     // Received packets
	TxPackets     uint64    `json:"tx_packets"`     // Transmitted packets
	RxErrors      uint64    `json:"rx_errors"`      // Receive errors
	TxErrors      uint64    `json:"tx_errors"`      // Transmit errors
	RxDropped     uint64    `json:"rx_dropped"`     // Receive dropped packets
	TxDropped     uint64    `json:"tx_dropped"`     // Transmit dropped packets
	Connections   uint32    `json:"connections"`    // Connection count
	TCPEstablished uint32   `json:"tcp_established"` // TCP established connections
}

type SystemMetrics struct {
	Timestamp    time.Time `json:"timestamp"`
	Uptime       uint64    `json:"uptime"`        // System uptime (seconds)
	Processes    uint32    `json:"processes"`     // Process count
	RunningProcs uint32    `json:"running_procs"` // Running processes
	BlockedProcs uint32    `json:"blocked_procs"` // Blocked processes
}
