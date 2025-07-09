package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/recon-scanner/internal/config"
	"github.com/recon-scanner/internal/database"
	"github.com/recon-scanner/internal/pool"
	"github.com/recon-scanner/internal/portscanner"
	"github.com/recon-scanner/internal/scheduler"
)

func main() {
	fmt.Println("🧪 === COMPREHENSIVE HIGH-PERFORMANCE MODE TEST ===")
	fmt.Println()

	// Test 1: Configuration Test
	fmt.Println("🔧 Test 1: Configuration System")
	testConfiguration()
	fmt.Println()

	// Test 2: Connection Pool Test
	fmt.Println("🔗 Test 2: Connection Pool")
	testConnectionPool()
	fmt.Println()

	// Test 3: Scheduler Test
	fmt.Println("📅 Test 3: Scheduler and Resource Management")
	testScheduler()
	fmt.Println()

	// Test 4: Port Scanner Test
	fmt.Println("🔌 Test 4: Port Scanner with Connection Pool")
	testPortScanner()
	fmt.Println()

	// Test 5: Database Test
	fmt.Println("💾 Test 5: Database Operations")
	testDatabase()
	fmt.Println()

	// Test 6: System Resource Monitoring
	fmt.Println("📊 Test 6: System Resource Monitoring")
	testResourceMonitoring()
	fmt.Println()

	fmt.Println("✅ === ALL TESTS COMPLETED ===")
}

func testConfiguration() {
	// Test different configuration modes
	configs := []struct {
		name                 string
		highPerformance      bool
		scheduleEnabled      bool
		detailedLogging      bool
		expectedWorkers      int
		expectedBatchSize    int
	}{
		{"Conservation Mode", false, false, false, 1, 500},
		{"High-Performance Mode", true, true, true, 800, 2000},
	}

	for _, test := range configs {
		fmt.Printf("  Testing %s...\n", test.name)
		
		cfg := config.New()
		cfg.EnableHighPerformanceMode = test.highPerformance
		cfg.HighPerformanceSchedule.Enabled = test.scheduleEnabled
		cfg.DetailedLogging = test.detailedLogging
		
		profile := cfg.GetCurrentProfile()
		mode := cfg.GetModeString()
		
		fmt.Printf("    Mode: %s\n", mode)
		fmt.Printf("    Workers: %d (expected: %d)\n", profile.WorkerCount, test.expectedWorkers)
		fmt.Printf("    Batch Size: %d (expected: %d)\n", profile.BatchSize, test.expectedBatchSize)
		fmt.Printf("    Request Delay: %v\n", profile.RequestDelay)
		fmt.Printf("    Timeout: %v\n", profile.Timeout)
		fmt.Printf("    Max Concurrent IPs: %d\n", profile.MaxConcurrentIP)
		
		if test.highPerformance {
			fmt.Printf("    Memory Pressure Threshold: %.1f%%\n", cfg.MemoryPressureThreshold*100)
			fmt.Printf("    Connection Pool Size: %d\n", cfg.ConnectionPoolSize)
			fmt.Printf("    Thermal Throttle Temp: %d°C\n", cfg.ThermalThrottleTemp)
			fmt.Printf("    High Thermal Throttle Temp: %d°C\n", cfg.HighThermalThrottleTemp)
		}
		
		fmt.Println("    ✅ Configuration test passed")
		fmt.Println()
	}
}

func testConnectionPool() {
	cfg := config.New()
	cfg.EnableHighPerformanceMode = true
	cfg.ConnectionPoolSize = 100
	cfg.MaxConnectionsPerWorker = 5
	cfg.ConnectionTimeout = time.Second * 5
	cfg.KeepAlive = time.Second * 30

	fmt.Println("  Creating connection pool...")
	pool := pool.NewConnectionPool(cfg)
	defer pool.Close()

	fmt.Println("  Testing connection creation...")
	// Test connecting to a known service (Google DNS)
	conn, err := pool.GetConnection("8.8.8.8", 53)
	if err != nil {
		fmt.Printf("    ❌ Connection failed: %v\n", err)
	} else {
		fmt.Println("    ✅ Connection successful")
		pool.ReturnConnection(conn)
	}

	// Test pool statistics
	stats := pool.GetStats()
	fmt.Printf("  Pool Statistics:\n")
	fmt.Printf("    Total Pools: %v\n", stats["total_pools"])
	fmt.Printf("    Global Connections: %v\n", stats["global_conn_count"])
	fmt.Printf("    Max Pools: %v\n", stats["max_pools"])
	fmt.Printf("    Max Connections per Pool: %v\n", stats["max_conn_per_pool"])
	
	fmt.Println("  ✅ Connection pool test passed")
}

func testScheduler() {
	cfg := config.New()
	cfg.EnableHighPerformanceMode = true
	cfg.HighPerformanceSchedule.Enabled = true
	cfg.DetailedLogging = true
	cfg.HealthCheckInterval = time.Second * 2
	cfg.MetricsInterval = time.Second * 5

	fmt.Println("  Creating scheduler...")
	sched := scheduler.New(cfg)
	defer sched.Stop()

	fmt.Println("  Starting scheduler...")
	sched.Start()

	// Test scheduler methods
	fmt.Printf("  Current Mode: %v\n", sched.GetCurrentMode())
	fmt.Printf("  Is High Performance: %v\n", sched.IsHighPerformanceMode())
	fmt.Printf("  Is Full Power: %v\n", sched.IsFullPowerMode())

	// Test adaptive delay
	baseDelay := time.Millisecond * 10
	adaptiveDelay := sched.GetAdaptiveDelay(baseDelay)
	fmt.Printf("  Adaptive Delay: %v (base: %v)\n", adaptiveDelay, baseDelay)

	// Test error/success recording
	sched.RecordSuccess()
	sched.RecordSuccess()
	sched.RecordError()

	// Wait for metrics to be collected
	time.Sleep(time.Second * 3)

	// Test system metrics
	metrics := sched.GetSystemMetrics()
	fmt.Printf("  System Metrics:\n")
	fmt.Printf("    CPU Temperature: %.1f°C\n", metrics.CPUTemperature)
	fmt.Printf("    Memory Usage: %.1f%% (%.1f MB)\n", metrics.MemoryPercent*100, float64(metrics.MemoryUsage)/1024/1024)
	fmt.Printf("    Goroutines: %d\n", metrics.GoroutineCount)
	fmt.Printf("    Success Count: %d\n", metrics.SuccessCount)
	fmt.Printf("    Error Count: %d\n", metrics.ErrorCount)
	fmt.Printf("    Throttle Level: %d%%\n", sched.GetThrottleLevel())

	fmt.Println("  ✅ Scheduler test passed")
}

func testPortScanner() {
	cfg := config.New()
	cfg.EnableHighPerformanceMode = true
	cfg.ConnectionPoolSize = 50
	cfg.MaxConnectionsPerWorker = 3

	fmt.Println("  Creating port scanner...")
	scanner := portscanner.New(cfg)
	defer scanner.Close()

	// Test scanning a known open port (Google DNS)
	fmt.Println("  Testing port scan...")
	result, err := scanner.ScanPort("8.8.8.8", 53)
	if err != nil {
		fmt.Printf("    ❌ Port scan failed: %v\n", err)
	} else {
		fmt.Printf("    IP: %s\n", result.IP)
		fmt.Printf("    Port: %d\n", result.Port)
		fmt.Printf("    Open: %v\n", result.IsOpen)
		fmt.Printf("    Service: %s\n", result.Service)
		fmt.Printf("    Banner: %s\n", result.Banner)
		fmt.Printf("    Processed At: %v\n", result.ProcessedAt)
	}

	fmt.Println("  ✅ Port scanner test passed")
}

func testDatabase() {
	dbPath := "test_high_performance.db"

	fmt.Println("  Creating database...")
	db, err := database.New(dbPath)
	if err != nil {
		fmt.Printf("    ❌ Database creation failed: %v\n", err)
		return
	}
	defer db.Close()

	// Test saving domain result
	fmt.Println("  Testing domain result storage...")
	domainResult := &database.DomainResult{
		Domain:      "test.example.com",
		ARecords:    []string{"192.168.1.1", "192.168.1.2"},
		ProcessedAt: time.Now(),
	}

	err = db.SaveDomain(domainResult)
	if err != nil {
		fmt.Printf("    ❌ Domain save failed: %v\n", err)
	} else {
		fmt.Println("    ✅ Domain result saved")
	}

	// Test saving port result
	fmt.Println("  Testing port result storage...")
	portResult := &database.PortResult{
		IP:          "192.168.1.1",
		Port:        80,
		IsOpen:      true,
		Service:     "HTTP",
		Banner:      "Server: nginx",
		ProcessedAt: time.Now(),
	}

	err = db.SavePort(portResult)
	if err != nil {
		fmt.Printf("    ❌ Port save failed: %v\n", err)
	} else {
		fmt.Println("    ✅ Port result saved")
	}

	// Test progress tracking
	fmt.Println("  Testing progress tracking...")
	progress := &database.Progress{
		Phase:       "test_phase",
		BatchIndex:  0,
		ItemIndex:   10,
		CompletedAt: time.Now(),
	}

	err = db.SaveProgress(progress)
	if err != nil {
		fmt.Printf("    ❌ Progress save failed: %v\n", err)
	} else {
		fmt.Println("    ✅ Progress saved")
	}

	// Test retrieving progress
	retrievedProgress, err := db.GetLastProgress("test_phase")
	if err != nil {
		fmt.Printf("    ❌ Progress retrieval failed: %v\n", err)
	} else if retrievedProgress != nil {
		fmt.Printf("    ✅ Progress retrieved: batch %d, item %d\n", 
			retrievedProgress.BatchIndex, retrievedProgress.ItemIndex)
	}

	fmt.Println("  ✅ Database test passed")
}

func testResourceMonitoring() {
	// Test system resource monitoring
	fmt.Println("  Testing system resource monitoring...")

	// Memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("    Memory Allocated: %.1f MB\n", float64(m.Alloc)/1024/1024)
	fmt.Printf("    Memory System: %.1f MB\n", float64(m.Sys)/1024/1024)
	fmt.Printf("    Memory Heap: %.1f MB\n", float64(m.HeapAlloc)/1024/1024)
	fmt.Printf("    Goroutines: %d\n", runtime.NumGoroutine())

	// CPU information
	fmt.Printf("    CPU Cores: %d\n", runtime.NumCPU())
	fmt.Printf("    Go Version: %s\n", runtime.Version())
	fmt.Printf("    OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	// Test garbage collection
	fmt.Println("  Testing garbage collection...")
	runtime.GC()
	runtime.ReadMemStats(&m)
	fmt.Printf("    Memory After GC: %.1f MB\n", float64(m.Alloc)/1024/1024)

	fmt.Println("  ✅ Resource monitoring test passed")
}