package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
	
	"github.com/recon-scanner/internal/config"
	"github.com/recon-scanner/internal/database"
	"github.com/recon-scanner/internal/dns"
	"github.com/recon-scanner/internal/monitoring"
	"github.com/recon-scanner/internal/worker"
)

func main() {
	// Set up high-performance configuration
	cfg := config.NewHighPerformanceConfig()
	
	// Set up logging
	logFile, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	
	// Set system limits
	setSystemLimits(cfg)
	
	// Initialize database
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	
	// Initialize DNS resolver with high-performance config
	resolver := dns.NewHighPerformance(cfg)
	
	// Initialize system monitor
	monitor := monitoring.NewSystemMonitor(cfg)
	monitor.Start()
	defer monitor.Stop()
	
	// Initialize worker pool
	pool := worker.NewWorkerPool(cfg, monitor, db, resolver)
	pool.Start()
	defer pool.Stop()
	
	// Print startup information
	printStartupInfo(cfg, monitor)
	
	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	setupGracefulShutdown(cancel)
	
	// Start processing
	startProcessing(ctx, cfg, pool, monitor)
	
	log.Println("High-performance scanner shutting down")
}

func setSystemLimits(cfg *config.HighPerformanceConfig) {
	// Set GOMAXPROCS to use all available cores
	runtime.GOMAXPROCS(runtime.NumCPU())
	
	// Set GC target percentage for better memory management
	runtime.GC()
	
	log.Printf("System configured for high performance: GOMAXPROCS=%d", runtime.GOMAXPROCS(0))
}

func printStartupInfo(cfg *config.HighPerformanceConfig, monitor *monitoring.SystemMonitor) {
	fmt.Printf("High-Performance Web2IP Scanner - Raspberry Pi 5 Optimized\n")
	fmt.Printf("Architecture: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("CPU Cores: %d\n", runtime.NumCPU())
	fmt.Printf("Max Workers: %d\n", cfg.MaxWorkers)
	fmt.Printf("Max Memory: %d MB\n", cfg.MaxMemoryUsage/1024/1024)
	fmt.Printf("Batch Size: %d\n", cfg.BatchSize)
	fmt.Printf("Started at: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	
	log.Printf("High-performance scanner started with %d max workers", cfg.MaxWorkers)
}

func setupGracefulShutdown(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-c
		fmt.Println("\nReceived shutdown signal, gracefully stopping...")
		log.Println("Received shutdown signal")
		cancel()
		time.Sleep(5 * time.Second)
		os.Exit(0)
	}()
}

func startProcessing(ctx context.Context, cfg *config.HighPerformanceConfig, pool *worker.WorkerPool, monitor *monitoring.SystemMonitor) {
	// Load domains from CSV
	domains, err := loadDomains(cfg.CSVFile)
	if err != nil {
		log.Fatalf("Failed to load domains: %v", err)
	}
	
	fmt.Printf("Loaded %d domains for processing\n", len(domains))
	log.Printf("Loaded %d domains from %s", len(domains), cfg.CSVFile)
	
	// Start metrics reporting
	go reportMetrics(ctx, monitor, cfg.MetricsInterval)
	
	// Process domains in batches
	processDomains(ctx, domains, pool, cfg, monitor)
}

func loadDomains(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file: %w", err)
	}
	
	var domains []string
	for _, record := range records {
		if len(record) > 0 {
			domains = append(domains, record[0])
		}
	}
	
	return domains, nil
}

func processDomains(ctx context.Context, domains []string, pool *worker.WorkerPool, cfg *config.HighPerformanceConfig, monitor *monitoring.SystemMonitor) {
	batchSize := cfg.BatchSize
	totalBatches := (len(domains) + batchSize - 1) / batchSize
	
	for i := 0; i < totalBatches; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		
		start := i * batchSize
		end := start + batchSize
		if end > len(domains) {
			end = len(domains)
		}
		
		batch := domains[start:end]
		
		// Adjust batch size based on system performance
		if monitor.ShouldThrottle() {
			batchSize = cfg.MinBatchSize
		} else {
			batchSize = cfg.BatchSize
		}
		
		// Submit DNS tasks
		for j, domain := range batch {
			task := worker.Task{
				ID:       fmt.Sprintf("dns_%d_%d", i, j),
				Type:     "DNS",
				Data:     domain,
				Priority: 1,
				Retry:    0,
			}
			pool.SubmitTask(task)
		}
		
		fmt.Printf("Submitted batch %d/%d (%d domains)\n", i+1, totalBatches, len(batch))
		
		// Add delay between batches if system is under pressure
		if monitor.ShouldThrottle() {
			time.Sleep(time.Second * 5)
		} else {
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func reportMetrics(ctx context.Context, monitor *monitoring.SystemMonitor, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			metrics := monitor.GetMetrics()
			fmt.Printf("Performance Report - CPU: %.1f°C, Memory: %.1f%%, Workers: %d, Processed: %d, Errors: %.2f%%\n",
				metrics.CPUTemp, metrics.MemoryPercent, metrics.ActiveWorkers, metrics.ProcessedItems, metrics.ErrorRate)
		case <-ctx.Done():
			return
		}
	}
}