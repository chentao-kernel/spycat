package exporter

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/chentao-kernel/spycat/pkg/monitor/collector"
)

// ExporterType
type ExporterType string

const (
	ConsoleExporter ExporterType = "console"
	FileExporter    ExporterType = "file"
	JSONExporter    ExporterType = "json"
	PrometheusExporter ExporterType = "prometheus"
)

// Exporter interface
type Exporter interface {
	Export(metrics *collector.SystemMetrics) error
	ExportSummary(summary *collector.SystemSummary) error
	Type() ExporterType
}

// ConsoleExporterImpl
type ConsoleExporterImpl struct{}

func NewConsoleExporter() *ConsoleExporterImpl {
	return &ConsoleExporterImpl{}
}

// Export to console
func (e *ConsoleExporterImpl) Export(systemMetrics *collector.SystemMetrics) error {
	fmt.Printf("\n=== System Metrics at %s ===\n", systemMetrics.Timestamp.Format(time.RFC3339))

	if systemMetrics.CPUMetrics != nil {
		fmt.Printf("CPU: User=%.2f%%, System=%.2f%%, Idle=%.2f%%, IOWait=%.2f%%\n",
			systemMetrics.CPUMetrics.UserPercent,
			systemMetrics.CPUMetrics.SystemPercent,
			systemMetrics.CPUMetrics.IdlePercent,
			systemMetrics.CPUMetrics.IOWaitPercent)
		fmt.Printf("Load: 1min=%.2f, 5min=%.2f, 15min=%.2f\n",
			systemMetrics.CPUMetrics.Load1,
			systemMetrics.CPUMetrics.Load5,
			systemMetrics.CPUMetrics.Load15)
		fmt.Printf("Context Switches: %d, Interrupts: %d\n",
			systemMetrics.CPUMetrics.ContextSwitch,
			systemMetrics.CPUMetrics.Interrupts)
	}

	if systemMetrics.MemMetrics != nil {
		fmt.Printf("Memory: Used=%s, Free=%s, Available=%s\n",
			formatBytes(systemMetrics.MemMetrics.Used),
			formatBytes(systemMetrics.MemMetrics.Free),
			formatBytes(systemMetrics.MemMetrics.Available))
		fmt.Printf("Swap: Used=%s, Free=%s\n",
			formatBytes(systemMetrics.MemMetrics.SwapUsed),
			formatBytes(systemMetrics.MemMetrics.SwapFree))
		fmt.Printf("Page Cache: %s, Buffers: %s\n",
			formatBytes(systemMetrics.MemMetrics.PageCache),
			formatBytes(systemMetrics.MemMetrics.Buffers))
	}

	if len(systemMetrics.IOMetrics) > 0 {
		fmt.Println("IO Metrics:")
		for _, ioMetric := range systemMetrics.IOMetrics {
			fmt.Printf("  %s: ReadIOPS=%.2f, WriteIOPS=%.2f, ReadLatency=%.2fms, WriteLatency=%.2fms\n",
				ioMetric.Device,
				ioMetric.ReadIOPS,
				ioMetric.WriteIOPS,
				ioMetric.ReadLatency,
				ioMetric.WriteLatency)
		}
	}

	if len(systemMetrics.NetMetrics) > 0 {
		fmt.Println("Network Metrics:")
		for _, netMetric := range systemMetrics.NetMetrics {
			fmt.Printf("  %s: Rx=%s/s, Tx=%s/s, Errors=%d, Dropped=%d\n",
				netMetric.Interface,
				formatBytes(netMetric.RxBytes),
				formatBytes(netMetric.TxBytes),
				netMetric.RxErrors+netMetric.TxErrors,
				netMetric.RxDropped+netMetric.TxDropped)
		}
	}

	return nil
}

// ExportSummary to console
func (e *ConsoleExporterImpl) ExportSummary(summary *collector.SystemSummary) error {
	fmt.Printf("\n=== System Summary at %s ===\n", summary.Timestamp.Format(time.RFC3339))
	fmt.Printf("CPU Usage: %.2f%%\n", summary.CPUUsage)
	fmt.Printf("Memory Usage: %.2f%%\n", summary.MemoryUsage)
	fmt.Printf("Swap Usage: %.2f%%\n", summary.SwapUsage)
	fmt.Printf("Load Average: %.2f\n", summary.LoadAverage)
	fmt.Printf("IO Utilization: %.2f%%, Total IOPS: %.2f\n", summary.IOUtilization, summary.TotalIOPS)
	fmt.Printf("Network: Rx=%s/s, Tx=%s/s\n",
		formatBytes(summary.NetworkRxBytes),
		formatBytes(summary.NetworkTxBytes))
	return nil
}

func (e *ConsoleExporterImpl) Type() ExporterType {
	return ConsoleExporter
}

// FileExporterImpl
type FileExporterImpl struct {
	filePath string
}

func NewFileExporter(filePath string) *FileExporterImpl {
	return &FileExporterImpl{
		filePath: filePath,
	}
}

// Export to file
func (e *FileExporterImpl) Export(systemMetrics *collector.SystemMetrics) error {
	file, err := os.OpenFile(e.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	data, err := json.Marshal(systemMetrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %v", err)
	}

	_, err = file.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("failed to write to file: %v", err)
	}

	return nil
}

// ExportSummary to file
func (e *FileExporterImpl) ExportSummary(summary *collector.SystemSummary) error {
	file, err := os.OpenFile(e.filePath+".summary", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open summary file: %v", err)
	}
	defer file.Close()

	data, err := json.Marshal(summary)
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %v", err)
	}

	_, err = file.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("failed to write summary to file: %v", err)
	}

	return nil
}

func (e *FileExporterImpl) Type() ExporterType {
	return FileExporter
}

// JSONExporterImpl json
type JSONExporterImpl struct{}

func NewJSONExporter() *JSONExporterImpl {
	return &JSONExporterImpl{}
}

// Export to json
func (e *JSONExporterImpl) Export(systemMetrics *collector.SystemMetrics) error {
	data, err := json.MarshalIndent(systemMetrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %v", err)
	}

	fmt.Println(string(data))
	return nil
}

// ExportSummary to json
func (e *JSONExporterImpl) ExportSummary(summary *collector.SystemSummary) error {
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %v", err)
	}

	fmt.Println(string(data))
	return nil
}

func (e *JSONExporterImpl) Type() ExporterType {
	return JSONExporter
}

type PrometheusExporterImpl struct{}

func NewPrometheusExporter() *PrometheusExporterImpl {
	return &PrometheusExporterImpl{}
}

// Export Prometheus
func (e *PrometheusExporterImpl) Export(systemMetrics *collector.SystemMetrics) error {
	// TODO: implement export in Prometheus format
	fmt.Println("Prometheus exporter not yet implemented")
	return nil
}

func (e *PrometheusExporterImpl) ExportSummary(summary *collector.SystemSummary) error {
	// TODO: implement summary exporter Prometheus format
	fmt.Println("Prometheus summary exporter not yet implemented")
	return nil
}

func (e *PrometheusExporterImpl) Type() ExporterType {
	return PrometheusExporter
}

type ExporterFactory struct{}

func NewExporterFactory() *ExporterFactory {
	return &ExporterFactory{}
}

func (f *ExporterFactory) CreateExporter(exporterType ExporterType, config map[string]string) (Exporter, error) {
	switch exporterType {
	case ConsoleExporter:
		return NewConsoleExporter(), nil
	case FileExporter:
		filePath, ok := config["file_path"]
		if !ok {
			return nil, fmt.Errorf("file_path is required for file exporter")
		}
		return NewFileExporter(filePath), nil
	case JSONExporter:
		return NewJSONExporter(), nil
	case PrometheusExporter:
		return NewPrometheusExporter(), nil
	default:
		return nil, fmt.Errorf("unsupported exporter type: %s", exporterType)
	}
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
