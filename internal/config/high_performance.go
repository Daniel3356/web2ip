package config

import (
	"runtime"
	"time"
)

type HighPerformanceConfig struct {
	// Worker configuration
	MaxWorkers        int
	MinWorkers        int
	WorkerScaleStep   int
	BatchSize         int
	MaxBatchSize      int
	MinBatchSize      int
	
	// Memory management
	MaxMemoryUsage    int64 // 6GB of 8GB RAM
	GCThreshold       int64 // 4GB trigger for GC
	MemoryCheckInterval time.Duration
	
	// Thermal management
	MaxCPUTemp        float64
	ThrottleTemp      float64
	CooldownTemp      float64
	TempCheckInterval time.Duration
	
	// Network configuration
	MaxConnections    int
	ConnectionTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	KeepAlive         time.Duration
	
	// Performance tuning
	RequestDelay      time.Duration
	RetryAttempts     int
	BackoffMultiplier float64
	
	// Monitoring
	MetricsInterval   time.Duration
	LogLevel          string
	HealthCheckInterval time.Duration
	
	// File paths
	CSVFile          string
	DatabasePath     string
	LogFile          string
	
	// Port scanning
	WebPorts         []int
	InfraPorts       []int
	MailPorts        []int
	DatabasePorts    []int
	
	// System limits
	MaxOpenFiles     int
	MaxCPUUsage      float64
	LoadAvgThreshold float64
}

func NewHighPerformanceConfig() *HighPerformanceConfig {
	return &HighPerformanceConfig{
		// Worker configuration for 800 workers
		MaxWorkers:      800,
		MinWorkers:      50,
		WorkerScaleStep: 25,
		BatchSize:       1000,
		MaxBatchSize:    5000,
		MinBatchSize:    100,
		
		// Memory management for 8GB RAM
		MaxMemoryUsage:      6 * 1024 * 1024 * 1024, // 6GB
		GCThreshold:         4 * 1024 * 1024 * 1024, // 4GB
		MemoryCheckInterval: 30 * time.Second,
		
		// Thermal management for Raspberry Pi 5
		MaxCPUTemp:        75.0,
		ThrottleTemp:      70.0,
		CooldownTemp:      65.0,
		TempCheckInterval: 10 * time.Second,
		
		// Network configuration
		MaxConnections:    1000,
		ConnectionTimeout: 10 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		KeepAlive:         30 * time.Second,
		
		// Performance tuning
		RequestDelay:      1 * time.Millisecond,
		RetryAttempts:     3,
		BackoffMultiplier: 2.0,
		
		// Monitoring
		MetricsInterval:     60 * time.Second,
		LogLevel:           "INFO",
		HealthCheckInterval: 30 * time.Second,
		
		// File paths
		CSVFile:      "top10milliondomains.csv",
		DatabasePath: "recon_results.db",
		LogFile:      "high_performance_recon.log",
		
		// Port scanning
		WebPorts:      []int{80, 443, 3000, 8080, 8888, 8443, 5000, 9000},
		InfraPorts:    []int{21, 22, 23, 139, 161, 445, 3389, 5985, 5986},
		MailPorts:     []int{25, 465, 587, 110, 995, 143, 993, 2525},
		DatabasePorts: []int{3306, 5432, 6379, 27017, 1521, 1433, 5984, 9200},
		
		// System limits
		MaxOpenFiles:     65536,
		MaxCPUUsage:      85.0,
		LoadAvgThreshold: float64(runtime.NumCPU()) * 0.8,
	}
}