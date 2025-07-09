# Web2IP High-Performance Scanner

A high-performance network reconnaissance scanner optimized for 24/7 operation on Raspberry Pi 5 with 8GB RAM.

## Features

### Standard Operation Modes
- **Conservation Mode**: Low resource usage during day time (6:30 AM - 1:37 AM)
- **Full Power Mode**: Enhanced performance during night time (1:37 AM - 6:30 AM)

### High Performance Mode 🚀
- **800 concurrent workers** for maximum throughput
- **Intelligent memory management** optimized for 8GB RAM
- **Thermal monitoring** and dynamic throttling to prevent overheating
- **Connection pooling** for efficient resource management
- **24/7 continuous operation** without restarts
- **System health monitoring** and alerts
- **Graceful degradation** under system pressure
- **Dynamic batch sizing** based on system conditions

## Usage

### Basic Usage
```bash
./web2ip
```

### High Performance Mode
```bash
./web2ip -high-performance
```

### Help
```bash
./web2ip -help
```

## High Performance Mode Configuration

### Key Settings
- **Workers**: 800 concurrent workers
- **Memory Threshold**: 80% (triggers throttling)
- **Thermal Threshold**: 75°C (triggers throttling)
- **Error Rate Threshold**: 5% (triggers throttling)
- **Health Check Interval**: 30 seconds
- **Connection Pool Size**: 200 connections
- **Max Connections per Host**: 10

### Automatic Adjustments
The system automatically adjusts performance based on:
- **CPU Temperature**: Reduces batch size and increases delays when hot
- **Memory Usage**: Implements garbage collection and batch size reduction
- **System Load**: Monitors goroutine count and system resources
- **Network Conditions**: Adapts to connection timeouts and errors
- **Error Rates**: Throttles when error thresholds are exceeded

## System Requirements

### Raspberry Pi 5 Optimized
- **CPU**: ARM Cortex-A76 (4 cores)
- **RAM**: 8GB (configured for optimal usage)
- **OS**: Linux (thermal monitoring requires /sys/class/thermal/)
- **Storage**: SSD recommended for database operations

### Dependencies
- Go 1.21+
- SQLite3
- Linux thermal zone support (for temperature monitoring)

## Architecture

### Core Components
1. **Health Monitor** (`internal/monitor/health.go`)
   - Real-time system health tracking
   - Memory, CPU, and error rate monitoring
   - Alert generation and handling

2. **Connection Pool** (`internal/pool/connection.go`)
   - Efficient connection reuse
   - Per-host connection limits
   - Automatic cleanup of stale connections

3. **Scheduler** (`internal/scheduler/scheduler.go`)
   - Intelligent resource-aware scheduling
   - Dynamic throttling based on system health
   - Time-based and performance-based mode switching

4. **Scanner** (`internal/scanner/scanner.go`)
   - High-concurrency DNS resolution
   - Efficient port scanning with connection pooling
   - Error tracking and recovery

### Performance Profiles

| Mode | Workers | Batch Size | Delay | Memory Target |
|------|---------|------------|-------|---------------|
| Conservation | 2 | 500 | 100ms | < 50% |
| Full Power | 12 | 5000 | 5ms | < 75% |
| High Performance | 800 | 2000* | 1ms* | < 80% |

*Dynamically adjusted based on system health

## Monitoring and Alerts

### Health Metrics
- **Memory Usage**: Real-time RAM consumption tracking
- **CPU Temperature**: Thermal monitoring for Pi 5
- **Error Rate**: Network and database error tracking
- **Goroutine Count**: Concurrency health monitoring

### Alert Levels
- **INFO**: Normal operational alerts
- **WARNING**: Performance degradation alerts
- **CRITICAL**: System protection alerts (triggers automatic action)

### Automatic Actions
- **Memory Pressure**: Forces garbage collection
- **High Temperature**: Increases delays and reduces batch sizes
- **High Error Rate**: Implements exponential backoff

## Database Schema

The scanner uses SQLite for efficient data storage:
- Domain records with A/AAAA resolution
- IP address records with reverse DNS
- Port scan results with service detection
- Progress tracking for resumable operations

## Logging

### Log Levels
- **Application Log**: `recon.log` - Detailed operational logging
- **Console Output**: Real-time progress and statistics
- **Health Alerts**: System health and performance warnings

### Log Rotation
Logs are managed automatically with the following features:
- Timestamp prefixes for all entries
- Error rate tracking and reporting
- Performance metrics logging
- Health status updates

## 24/7 Operation

### Continuous Operation Features
- **Resumable Scanning**: Automatic checkpoint saving
- **Graceful Degradation**: Reduces performance under pressure
- **Error Recovery**: Automatic retry mechanisms
- **Resource Management**: Prevents system overload

### Recommended Setup
```bash
# Run in screen/tmux for persistent sessions
screen -S web2ip ./web2ip -high-performance

# Or as a systemd service
sudo systemctl enable web2ip-scanner
sudo systemctl start web2ip-scanner
```

## Performance Optimization

### Memory Management
- Connection pooling reduces memory fragmentation
- Dynamic batch sizing prevents memory spikes
- Automatic garbage collection on pressure
- Efficient data structures for large datasets

### Thermal Management
- Real-time temperature monitoring
- Dynamic throttling based on thermal zones
- Preemptive cooling through reduced activity
- Maintains performance within thermal limits

### Error Handling
- Exponential backoff for network errors
- Automatic retry mechanisms
- Error rate monitoring and alerting
- Graceful degradation on high error rates

## Building and Development

### Build
```bash
go build -o web2ip
```

### Test
```bash
go test ./...
```

### Development Mode
```bash
go run main.go -high-performance
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Implement changes with tests
4. Ensure all performance benchmarks pass
5. Submit pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues related to high performance mode:
1. Check system resources (memory, temperature)
2. Review logs for error patterns
3. Verify thermal monitoring is working
4. Check connection pool statistics

## Changelog

### v2.0.0 - High Performance Mode
- Added 800-worker high performance mode
- Implemented intelligent memory management
- Added thermal monitoring and throttling
- Implemented connection pooling
- Added system health monitoring
- Added graceful degradation mechanisms
- Enhanced error handling and recovery
download the full web
