# Linux System Monitoring Module

This module implements monitoring metrics collection for various Linux system subsystems, including CPU, memory, I/O, and network subsystems.

## Directory Structure

```
pkg/monitor/
├── README.md           # This document
├── collector/          # Collector management
├── cpu/               # CPU monitoring
├── io/                # I/O monitoring
├── mem/               # Memory monitoring
├── net/               # Network monitoring
├── metrics/           # Metrics definitions
└── exporter/          # Metrics export
```

## Features

### 1. CPU Monitoring (cpu/)
- **CPU Usage**: User, system, idle, I/O wait, etc.
- **Load Average**: 1-minute, 5-minute, 15-minute load
- **Context Switches**: Context switches per second
- **Interrupt Statistics**: Interrupts per second
- **Key Metrics**:
  - UserPercent: User mode CPU usage
  - SystemPercent: System mode CPU usage
  - IdlePercent: Idle CPU percentage
  - IOWaitPercent: I/O wait percentage
  - Load1/Load5/Load15: Load averages
  - ContextSwitch: Context switch count
  - Interrupts: Interrupt count

### 2. Memory Monitoring (mem/)
- **Memory Usage**: Total, used, free, available memory
- **Swap Space**: Swap space usage
- **Page Cache**: Page cache and buffers
- **Page Faults**: Page faults and major page faults
- **Key Metrics**:
  - Total: Total memory
  - Used: Used memory
  - Free: Free memory
  - Available: Available memory
  - SwapTotal/SwapUsed/SwapFree: Swap space
  - PageCache: Page cache
  - Buffers: Buffers
  - PageFaults/MajorPageFaults: Page faults

### 3. I/O Monitoring (io/)
- **IOPS**: Read/write IOPS
- **Throughput**: Read/write bytes
- **Latency**: Read/write latency
- **Queue Depth**: Current queue depth
- **Utilization**: Disk utilization
- **Key Metrics**:
  - ReadIOPS/WriteIOPS: Read/write IOPS
  - ReadBytes/WriteBytes: Read/write bytes
  - ReadLatency/WriteLatency: Read/write latency
  - QueueDepth: Queue depth
  - Utilization: Utilization

### 4. Network Monitoring (net/)
- **Network Traffic**: Receive and transmit bytes
- **Packet Statistics**: Receive and transmit packets
- **Error Statistics**: Receive and transmit errors
- **Drop Statistics**: Receive and transmit drops
- **Connection Statistics**: TCP connections and established connections
- **Key Metrics**:
  - RxBytes/TxBytes: Receive/transmit bytes
  - RxPackets/TxPackets: Receive/transmit packets
  - RxErrors/TxErrors: Receive/transmit errors
  - RxDropped/TxDropped: Receive/transmit drops
  - Connections: Connection count
  - TCPEstablished: Established TCP connections

## Data Sources

### CPU Monitoring
- `/proc/stat` - CPU statistics
- `/proc/loadavg` - Load averages
- `/proc/vmstat` - Context switches and interrupts

### Memory Monitoring
- `/proc/meminfo` - Memory information
- `/proc/vmstat` - Page fault statistics

### I/O Monitoring
- `/proc/diskstats` - Disk statistics

### Network Monitoring
- `/proc/net/dev` - Network interface statistics
- `/proc/net/sockstat` - Socket statistics

## Usage

### Basic Usage

```go
import "github.com/chentao-kernel/spycat/pkg/monitor"

// Create collectors
cpuCollector := cpu.NewCPUCollector()
memCollector := mem.NewMemoryCollector()
ioCollector := io.NewIOCollector()
netCollector := net.NewNetworkCollector()

// Collect metrics
cpuMetrics, err := cpuCollector.Collect()
memMetrics, err := memCollector.Collect()
ioMetrics, err := ioCollector.Collect()
netMetrics, err := netCollector.Collect()
```

### Testing

All modules include comprehensive unit tests:

```bash
# Run all tests
cd pkg/monitor && go test ./... -v

# Run specific module tests
go test ./cpu -v
go test ./mem -v
go test ./io -v
go test ./net -v
```

## Test Coverage

- ✅ **CPU Monitoring**: Complete unit tests including calculation logic, parsing logic, and integration tests
- ✅ **Memory Monitoring**: Complete unit tests including memory usage calculation, swap space calculation, and page fault statistics
- ✅ **I/O Monitoring**: Complete unit tests including IOPS calculation, latency calculation, and total I/O statistics
- ✅ **Network Monitoring**: Complete unit tests including network traffic calculation, connection statistics, and total network statistics
- ✅ **Metrics Definitions**: Complete type definitions and creation tests

## Design Features

1. **Incremental Calculation**: Calculate delta metrics based on time differences to avoid absolute value issues
2. **Error Handling**: Comprehensive error handling and boundary condition checking
3. **Performance Optimization**: Avoid repeated system file reads, cache previous statistics
4. **Type Safety**: Use strong type definitions to prevent type errors
5. **Test Coverage**: High test coverage to ensure code quality

## Extensibility

The module design has good extensibility, making it easy to add new monitoring metrics or subsystems:

1. Add new metric types in the `metrics/` package
2. Create new collector implementing the `Collect()` method
3. Add corresponding unit tests

## Notes

- First collection may return nil (no baseline data for comparison)
- Some tests may return empty results in specific environments (e.g., no network interfaces)
- All timestamps use UTC time
- Memory units are unified to bytes, time units are unified to seconds
