package scanner

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/recon-scanner/internal/config"
	"github.com/recon-scanner/internal/database"
	"github.com/recon-scanner/internal/dns"
	"github.com/recon-scanner/internal/portscanner"
	"github.com/recon-scanner/internal/scheduler"
)

type Scanner struct {
	config      *config.Config
	db          *database.Database
	dns         *dns.Resolver
	portScanner *portscanner.Scanner
	scheduler   *scheduler.Scheduler
}

func New(cfg *config.Config, db *database.Database) *Scanner {
	return &Scanner{
		config:      cfg,
		db:          db,
		dns:         dns.New(cfg),
		portScanner: portscanner.New(cfg),
		scheduler:   scheduler.New(cfg),
	}
}

func (s *Scanner) Close() {
	if s.portScanner != nil {
		s.portScanner.Close()
	}
	if s.scheduler != nil {
		s.scheduler.Stop()
	}
}

func (s *Scanner) Run(domains []string) error {
	// Start the scheduler
	s.scheduler.Start()
	defer s.scheduler.Stop()
	
	// Log initial status
	s.logCurrentStatus()
	
	// Start health monitoring in high-performance mode
	if s.config.EnableHighPerformanceMode {
		go s.healthMonitoringLoop()
	}

	fmt.Println("📋 Phase 1: DNS Resolution")
	if err := s.resolveDNS(domains); err != nil {
		return fmt.Errorf("DNS resolution failed: %w", err)
	}

	fmt.Println("🔍 Phase 2: Extracting unique IPs and reverse lookup")
	uniqueIPs, err := s.extractAndProcessIPs()
	if err != nil {
		return fmt.Errorf("IP extraction failed: %w", err)
	}

	fmt.Printf("Found %d unique IPs\n", len(uniqueIPs))

	fmt.Println("🔌 Phase 3: Port Scanning")
	if err := s.scanPorts(uniqueIPs); err != nil {
		return fmt.Errorf("port scanning failed: %w", err)
	}

	return nil
}

func (s *Scanner) logCurrentStatus() {
	mode := s.config.GetModeString()
	profile := s.config.GetCurrentProfile()
	
	location, _ := time.LoadLocation(s.config.Timezone)
	now := time.Now().In(location)
	
	fmt.Printf("\n🏁 SCANNER STARTING at %s\n", now.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("Current Mode: %s\n", mode)
	fmt.Printf("Workers: %d | Batch Size: %d | Delay: %v\n", 
		profile.WorkerCount, profile.BatchSize, profile.RequestDelay)
	
	timeUntilChange := s.config.GetTimeUntilModeChange()
	fmt.Printf("Time until mode change: %v\n\n", timeUntilChange)
}

func (s *Scanner) resolveDNS(domains []string) error {
	// Wait for optimal time if intensive operation
	s.scheduler.WaitForOptimalTime("DNS resolution")
	
	// Check for existing progress
	progress, err := s.db.GetLastProgress("dns_resolution")
	if err != nil {
		log.Printf("Error checking DNS progress: %v", err)
	}

	startIndex := 0
	if progress != nil {
		startIndex = progress.ItemIndex
		fmt.Printf("Resuming DNS resolution from index %d\n", startIndex)
	}

	// Get already processed domains
	processed, err := s.db.GetProcessedDomains()
	if err != nil {
		return fmt.Errorf("failed to get processed domains: %w", err)
	}

	// Filter out already processed domains
	var remainingDomains []string
	for i := startIndex; i < len(domains); i++ {
		if !processed[domains[i]] {
			remainingDomains = append(remainingDomains, domains[i])
		}
	}

	fmt.Printf("Processing %d remaining domains\n", len(remainingDomains))

	// Process in batches with dynamic sizing based on current mode
	profile := s.config.GetCurrentProfile()
	batchSize := profile.BatchSize
	totalBatches := (len(remainingDomains) + batchSize - 1) / batchSize

	for batchIndex := 0; batchIndex < totalBatches; batchIndex++ {
		// Check if mode changed and update batch processing accordingly
		currentProfile := s.config.GetCurrentProfile()
		if currentProfile.BatchSize != profile.BatchSize {
			profile = currentProfile
			fmt.Printf("🔄 Performance mode changed, adapting batch processing\n")
		}
		
		start := batchIndex * batchSize
		end := start + batchSize
		if end > len(remainingDomains) {
			end = len(remainingDomains)
		}

		batch := remainingDomains[start:end]
		mode := s.config.GetModeString()
		fmt.Printf("%s Processing DNS batch %d/%d (%d domains)\n", 
			mode, batchIndex+1, totalBatches, len(batch))

		if err := s.processDNSBatch(batch); err != nil {
			log.Printf("Error processing DNS batch %d: %v", batchIndex, err)
			continue
		}

		// Save progress
		progress := &database.Progress{
			Phase:       "dns_resolution",
			BatchIndex:  batchIndex,
			ItemIndex:   start + len(batch),
			CompletedAt: time.Now(),
		}
		s.db.SaveProgress(progress)

		fmt.Printf("Completed DNS batch %d/%d\n", batchIndex+1, totalBatches)
		
		// Add inter-batch delay during conservation mode
		if s.scheduler.ShouldThrottle() {
			time.Sleep(time.Second * 2)
		}
	}

	return nil
}

func (s *Scanner) processDNSBatch(domains []string) error {
	profile := s.config.GetCurrentProfile()
	
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, profile.WorkerCount)
	
	for _, domain := range domains {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			result, err := s.dns.ResolveDomain(d)
			if err != nil {
				log.Printf("Failed to resolve %s: %v", d, err)
				return
			}

			if err := s.db.SaveDomain(result); err != nil {
				log.Printf("Failed to save domain %s: %v", d, err)
			}

			// Use adaptive delay based on current mode and system state
			delay := s.scheduler.GetAdaptiveDelay(profile.RequestDelay)
			time.Sleep(delay)
		}(domain)
	}

	wg.Wait()
	return nil
}

func (s *Scanner) extractAndProcessIPs() ([]string, error) {
	// Get all unique IPs from domains
	uniqueIPsMap := make(map[string]bool)
	
	// Query database for all A and AAAA records
	rows, err := s.db.GetAllIPsFromDomains()
	if err != nil {
		return nil, fmt.Errorf("failed to get IPs from domains: %w", err)
	}

	for _, ip := range rows {
		if ip != "" {
			uniqueIPsMap[ip] = true
		}
	}

	// Convert map to slice
	var uniqueIPs []string
	for ip := range uniqueIPsMap {
		uniqueIPs = append(uniqueIPs, ip)
	}

	// Process reverse DNS lookups for new IPs
	if err := s.processReverseDNS(uniqueIPs); err != nil {
		log.Printf("Error processing reverse DNS: %v", err)
	}

	return uniqueIPs, nil
}

func (s *Scanner) processReverseDNS(ips []string) error {
	fmt.Printf("🔄 Processing reverse DNS for %d IPs\n", len(ips))
	
	profile := s.config.GetCurrentProfile()
	
	// Limit concurrent IPs during conservation mode
	maxConcurrent := profile.MaxConcurrentIP
	if len(ips) > maxConcurrent && s.scheduler.ShouldThrottle() {
		fmt.Printf("⚠️ Conservation mode: limiting to %d concurrent IP lookups\n", maxConcurrent)
	}
	
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, profile.WorkerCount)

	for _, ip := range ips {
		// Throttle during conservation mode
		if s.scheduler.ShouldThrottle() && len(ips) > maxConcurrent {
			time.Sleep(time.Millisecond * 50)
		}
		
		wg.Add(1)
		go func(targetIP string) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			ptrRecord, _ := s.dns.ReverseLookup(targetIP)
			
			ipResult := &database.IPResult{
				IP:          targetIP,
				PTRRecord:   ptrRecord,
				ProcessedAt: time.Now(),
			}

			if err := s.db.SaveIP(ipResult); err != nil {
				log.Printf("Failed to save IP %s: %v", targetIP, err)
			}

			delay := s.scheduler.GetAdaptiveDelay(profile.RequestDelay)
			time.Sleep(delay)
		}(ip)
	}

	wg.Wait()
	return nil
}

func (s *Scanner) scanPorts(ips []string) error {
	ports := s.config.AllPorts()
	
	for _, port := range ports {
		mode := s.config.GetModeString()
		fmt.Printf("%s Scanning port %d on %d IPs\n", mode, port, len(ips))
		
		if err := s.scanPortOnIPs(ips, port); err != nil {
			log.Printf("Error scanning port %d: %v", port, err)
			continue
		}
		
		// Longer pause between ports during conservation mode
		if s.scheduler.ShouldThrottle() {
			time.Sleep(time.Second * 5)
		}
	}

	return nil
}

func (s *Scanner) scanPortOnIPs(ips []string, port int) error {
	// Filter IPs that haven't been scanned for this port
	var unscannedIPs []string
	for _, ip := range ips {
		scanned, err := s.db.IsPortScanned(ip, port)
		if err != nil {
			log.Printf("Error checking port scan status for %s:%d: %v", ip, port, err)
			continue
		}
		if !scanned {
			unscannedIPs = append(unscannedIPs, ip)
		}
	}

	if len(unscannedIPs) == 0 {
		fmt.Printf("Port %d already scanned on all IPs\n", port)
		return nil
	}

	fmt.Printf("Scanning port %d on %d unscanned IPs\n", port, len(unscannedIPs))

	// Process in batches with dynamic sizing
	profile := s.config.GetCurrentProfile()
	batchSize := profile.BatchSize
	totalBatches := (len(unscannedIPs) + batchSize - 1) / batchSize

	for batchIndex := 0; batchIndex < totalBatches; batchIndex++ {
		start := batchIndex * batchSize
		end := start + batchSize
		if end > len(unscannedIPs) {
			end = len(unscannedIPs)
		}

		batch := unscannedIPs[start:end]
		mode := s.config.GetModeString()
		fmt.Printf("%s Scanning port %d - batch %d/%d (%d IPs)\n", 
			mode, port, batchIndex+1, totalBatches, len(batch))

		if err := s.scanPortBatch(batch, port); err != nil {
			log.Printf("Error scanning port %d batch %d: %v", port, batchIndex, err)
			continue
		}

		// Save progress
		progress := &database.Progress{
			Phase:       fmt.Sprintf("port_scan_%d", port),
			BatchIndex:  batchIndex,
			ItemIndex:   end,
			CompletedAt: time.Now(),
		}
		s.db.SaveProgress(progress)
	}

	return nil
}

func (s *Scanner) healthMonitoringLoop() {
	ticker := time.NewTicker(s.config.MetricsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			s.logSystemHealth()
		}
	}
}

func (s *Scanner) logSystemHealth() {
	if !s.config.DetailedLogging {
		return
	}
	
	metrics := s.scheduler.GetSystemMetrics()
	throttleLevel := s.scheduler.GetThrottleLevel()
	
	fmt.Printf("\n📊 === SYSTEM HEALTH REPORT ===\n")
	fmt.Printf("🌡️  CPU Temperature: %.1f°C\n", metrics.CPUTemperature)
	fmt.Printf("🧠 Memory Usage: %.1f%% (%.1f MB)\n", metrics.MemoryPercent*100, float64(metrics.MemoryUsage)/1024/1024)
	fmt.Printf("🔄 Goroutines: %d\n", metrics.GoroutineCount)
	fmt.Printf("✅ Success Rate: %.1f%% (%d/%d)\n", 
		float64(metrics.SuccessCount)/float64(metrics.SuccessCount+metrics.ErrorCount)*100,
		metrics.SuccessCount, metrics.SuccessCount+metrics.ErrorCount)
	fmt.Printf("⚡ Throttle Level: %d%%\n", throttleLevel)
	fmt.Printf("🚀 Current Mode: %s\n", s.config.GetModeString())
	
	// Add connection pool statistics if available
	if s.portScanner != nil && s.config.EnableHighPerformanceMode {
		// Here we would add connection pool stats if we had access to them
		// For now, we'll just show that high-performance mode is active
		fmt.Printf("🔗 Connection Pool: Active\n")
	}
	
	fmt.Printf("⏰ Last Updated: %s\n", metrics.LastUpdateTime.Format("15:04:05"))
	fmt.Printf("===============================\n\n")
}

func (s *Scanner) scanPortBatch(ips []string, port int) error {
	profile := s.config.GetCurrentProfile()
	
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, profile.WorkerCount)

	for _, ip := range ips {
		wg.Add(1)
		go func(targetIP string) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			result, err := s.portScanner.ScanPort(targetIP, port)
			if err != nil {
				log.Printf("Failed to scan %s:%d: %v", targetIP, port, err)
				s.scheduler.RecordError()
				return
			}

			s.scheduler.RecordSuccess()
			
			if err := s.db.SavePort(result); err != nil {
				log.Printf("Failed to save port result %s:%d: %v", targetIP, port, err)
				s.scheduler.RecordError()
				return
			}

			delay := s.scheduler.GetAdaptiveDelay(profile.RequestDelay)
			time.Sleep(delay)
		}(ip)
	}

	wg.Wait()
	return nil
}