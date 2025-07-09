package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/recon-scanner/internal/config"
	"github.com/recon-scanner/internal/database"
	"github.com/recon-scanner/internal/scanner"
)

func main() {
	// Test high-performance configuration
	cfg := config.New()
	cfg.EnableHighPerformanceMode = true
	cfg.HighPerformanceSchedule.Enabled = true
	cfg.DetailedLogging = true
	
	fmt.Println("🚀 Testing High-Performance Configuration")
	fmt.Printf("📊 Configuration Profile: %s\n", cfg.GetModeString())
	
	profile := cfg.GetCurrentProfile()
	fmt.Printf("⚡ Workers: %d\n", profile.WorkerCount)
	fmt.Printf("📦 Batch Size: %d\n", profile.BatchSize)
	fmt.Printf("⏱️  Request Delay: %v\n", profile.RequestDelay)
	fmt.Printf("🔄 Max Concurrent IPs: %d\n", profile.MaxConcurrentIP)
	fmt.Printf("⏰ Timeout: %v\n", profile.Timeout)
	
	fmt.Printf("\n🔧 System Configuration:\n")
	fmt.Printf("🧠 Max Memory Usage: %.1f GB\n", float64(cfg.MaxMemoryUsage)/1024/1024/1024)
	fmt.Printf("🌡️  Thermal Throttle: %d°C\n", cfg.ThermalThrottleTemp)
	fmt.Printf("🌡️  High Thermal Throttle: %d°C\n", cfg.HighThermalThrottleTemp)
	fmt.Printf("💾 Memory Pressure Threshold: %.1f%%\n", cfg.MemoryPressureThreshold*100)
	fmt.Printf("🔗 Connection Pool Size: %d\n", cfg.ConnectionPoolSize)
	fmt.Printf("📊 Metrics Interval: %v\n", cfg.MetricsInterval)
	fmt.Printf("❤️  Health Check Interval: %v\n", cfg.HealthCheckInterval)
	
	fmt.Printf("\n🖥️  System Information:\n")
	fmt.Printf("🔧 Go Version: %s\n", runtime.Version())
	fmt.Printf("💻 OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("🔄 CPU Cores: %d\n", runtime.NumCPU())
	
	// Test database connection
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()
	
	fmt.Println("\n✅ Database connection successful")
	
	// Test scanner initialization
	scannerInstance := scanner.New(cfg, db)
	defer scannerInstance.Close()
	
	fmt.Println("✅ Scanner initialization successful")
	
	// Test with a small domain list
	testDomains := []string{
		"google.com",
		"github.com",
		"stackoverflow.com",
	}
	
	fmt.Printf("\n🎯 Testing with %d domains...\n", len(testDomains))
	
	startTime := time.Now()
	err = scannerInstance.Run(testDomains)
	duration := time.Since(startTime)
	
	if err != nil {
		log.Printf("Scanner test failed: %v", err)
	} else {
		fmt.Printf("✅ Scanner test completed in %v\n", duration)
	}
	
	fmt.Println("\n🎉 High-Performance Mode Test Complete!")
}