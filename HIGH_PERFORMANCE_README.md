# High-Performance Mode for Raspberry Pi 5 with 8GB RAM

## Overview
This implementation adds a specialized high-performance mode optimized for 24/7 operation on Raspberry Pi 5 with 8GB RAM, capable of running 800 concurrent workers without thermal throttling or memory issues.

## Features

### 🚀 High-Performance Configuration
- **800 concurrent workers** for maximum throughput
- **2000 batch size** optimized for high-performance scanning
- **1ms request delay** for maximum speed
- **10s timeout** for stability under load
- **800 max concurrent IP scans**

### 🧠 Intelligent Memory Management
- **Memory usage monitoring** with 80% threshold
- **Automatic garbage collection** when memory pressure detected
- **Memory optimized for 8GB RAM** (6GB usable, 2GB reserved for system)
- **Connection pooling** to reduce memory overhead

### 🌡️ Thermal Monitoring & Dynamic Throttling
- **Real-time CPU temperature monitoring** (Linux/Raspberry Pi)
- **Dynamic throttling** based on temperature thresholds
- **Preemptive throttling** at 60°C for high-performance mode
- **Critical throttling** at 70°C standard threshold
- **Automatic performance adjustment** based on system conditions

### 🔗 Connection Pooling & Resource Management
- **Connection pool** with 1000 total connections
- **Per-worker connection limits** (5 connections per worker)
- **Connection reuse** and keep-alive optimization
- **Automatic cleanup** of stale connections
- **Connection timeout** and retry logic

### 📊 Comprehensive Monitoring & Logging
- **Real-time system health monitoring**
- **Detailed metrics** every 30 seconds
- **Resource usage tracking** (CPU, memory, goroutines)
- **Success/error rate monitoring**
- **Throttle level indicators**
- **Connection pool statistics**

### 🔧 Error Handling & Recovery
- **Circuit breaker pattern** for resilience
- **Automatic retry logic** with exponential backoff
- **Graceful degradation** under system pressure
- **Error rate monitoring** with automatic throttling

## Usage

### Command Line Options

```bash
# Enable high-performance mode with 800 workers
./recon-scanner --high-performance

# Enable detailed logging for monitoring
./recon-scanner --detailed-logging

# Force specific configuration profile
./recon-scanner --config highperformance

# Combine options
./recon-scanner --high-performance --detailed-logging
```

### Configuration Profiles

1. **Auto Mode** (default): Time-based switching between conservation and full-power
2. **Conservation Mode**: Low resource usage for daytime operation
3. **Full Power Mode**: Medium performance for nighttime operation
4. **High Performance Mode**: Maximum performance with 800 workers

### High-Performance Mode Activation

```bash
# Method 1: Command line flag
./recon-scanner --high-performance

# Method 2: Configuration profile
./recon-scanner --config highperformance

# Method 3: Both with detailed logging
./recon-scanner --high-performance --detailed-logging
```

## System Requirements

### Minimum Requirements
- **Raspberry Pi 5** with 8GB RAM
- **Adequate cooling** (heatsink + fan recommended)
- **Fast storage** (SSD recommended for database)
- **Stable network connection**

### Recommended Setup
- **Active cooling** (fan + heatsink)
- **External SSD** for database storage
- **Gigabit ethernet** connection
- **Power supply** with adequate capacity

## Performance Monitoring

### Real-Time Health Monitoring
When detailed logging is enabled, the system provides real-time health reports:

```
📊 === SYSTEM HEALTH REPORT ===
🌡️  CPU Temperature: 45.2°C
🧠 Memory Usage: 67.3% (4.2 GB)
🔄 Goroutines: 850
✅ Success Rate: 98.7% (12450/12616)
⚡ Throttle Level: 0%
🚀 Current Mode: 🚀 HIGH PERFORMANCE
🔗 Connection Pool: Active
⏰ Last Updated: 16:45:30
===============================
```

### Throttling Levels
- **0%**: Normal operation
- **25%**: Light throttling (high memory usage)
- **50%**: Medium throttling (high temperature)
- **75%**: Heavy throttling (critical memory usage)
- **90%**: Maximum throttling (critical temperature)

## Safety Features

### Thermal Protection
- **Preemptive throttling** at 60°C
- **Progressive throttling** as temperature rises
- **Automatic performance reduction** to prevent overheating
- **Emergency shutdown** if temperature exceeds critical levels

### Memory Protection
- **Memory usage monitoring** with 80% threshold
- **Automatic garbage collection** when usage is high
- **Connection pool limits** to prevent memory exhaustion
- **Batch size adjustment** based on available memory

### Error Handling
- **Circuit breaker** prevents cascade failures
- **Automatic retry** with exponential backoff
- **Error rate monitoring** with automatic throttling
- **Graceful degradation** under system pressure

## Configuration Parameters

### High-Performance Settings
```go
HighPerformance: PerformanceProfile{
    BatchSize:       2000,                    // Optimized batch size
    WorkerCount:     800,                     // Maximum concurrent workers
    RequestDelay:    time.Millisecond * 1,    // Minimal delay
    Timeout:         time.Second * 10,        // Stability timeout
    MaxConcurrentIP: 800,                     // Concurrent IP scans
}
```

### Resource Management
```go
MaxMemoryUsage:          6 * 1024 * 1024 * 1024, // 6GB usable
MemoryPressureThreshold: 0.8,                     // 80% threshold
ConnectionPoolSize:      1000,                    // Total connections
MaxConnectionsPerWorker: 5,                       // Per-worker limit
```

### Monitoring Configuration
```go
MetricsInterval:     time.Second * 30,  // Health report interval
HealthCheckInterval: time.Second * 10,  // System check interval
DetailedLogging:     true,              // Enable detailed output
```

## Testing

### Test Script
Run the included test script to validate high-performance mode:

```bash
go run test_high_performance.go
```

This will:
1. Initialize high-performance configuration
2. Test database connection
3. Test scanner initialization
4. Run a small scan with monitoring
5. Display system health metrics

### Performance Validation
The test should show:
- ✅ 800 workers configured
- ✅ Connection pool active
- ✅ Real-time monitoring working
- ✅ Thermal monitoring functional
- ✅ Memory management active

## Troubleshooting

### High CPU Temperature
- Check cooling system (fan/heatsink)
- Verify adequate airflow
- Consider reducing worker count
- Enable detailed logging to monitor throttling

### Memory Issues
- Monitor memory usage with detailed logging
- Check for memory leaks in long-running scans
- Consider reducing batch size
- Ensure adequate swap space

### Connection Issues
- Check network connectivity
- Monitor connection pool statistics
- Verify timeout settings
- Check for firewall restrictions

### Performance Issues
- Enable detailed logging for diagnostics
- Monitor system health reports
- Check throttle levels
- Verify system resources

## Best Practices

### 24/7 Operation
1. **Monitor system health** regularly
2. **Ensure adequate cooling** at all times
3. **Use external storage** for database
4. **Implement monitoring alerts**
5. **Regular system maintenance**

### Resource Management
1. **Start with lower worker counts** and scale up
2. **Monitor memory usage** continuously
3. **Watch CPU temperature** especially in summer
4. **Use connection pooling** for efficiency
5. **Enable detailed logging** for troubleshooting

### Security Considerations
1. **Rate limiting** to avoid being blocked
2. **Respect robots.txt** and terms of service
3. **Use proper user agents**
4. **Implement delays** between requests
5. **Monitor for abuse reports**

## Future Enhancements

### Planned Features
- **GPU acceleration** for DNS resolution
- **Distributed scanning** across multiple Pi devices
- **Machine learning** for optimal resource allocation
- **Web interface** for monitoring and control
- **Docker containerization** for easy deployment

### Performance Optimizations
- **Custom DNS resolver** with caching
- **Optimized network stack** configuration
- **Memory-mapped database** for faster access
- **Async I/O** for better concurrency
- **SIMD optimizations** for data processing

## License

This high-performance implementation is provided under the same license as the original web2ip project.