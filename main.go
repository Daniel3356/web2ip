package main

import (
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
	"github.com/recon-scanner/internal/scanner"
	"github.com/recon-scanner/internal/utils"
)

func main() {
	// Set up logging with timestamps
	logFile, err := os.OpenFile("recon.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file:", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Log startup
	log.Printf("=== RECON SCANNER STARTING ===")
	log.Printf("Go Version: %s", runtime.Version())
	log.Printf("Architecture: %s/%s", runtime.GOOS, runtime.GOARCH)
	log.Printf("CPU Cores: %d", runtime.NumCPU())

	fmt.Println("🚀 Recon Scanner System - Raspberry Pi 5 Optimized")
	fmt.Printf("💻 Running on %s/%s with %d CPU cores\n", runtime.GOOS, runtime.GOARCH, runtime.NumCPU())

	// Initialize configuration
	cfg := config.New()
	
	// Display current time zone and schedule
	location, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.Printf("Warning: Could not load timezone %s, using UTC", cfg.Timezone)
		location = time.UTC
	}
	
	now := time.Now().In(location)
	fmt.Printf("🕐 Current time: %s\n", now.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("⚡ Full power window: %02d:%02d - %02d:%02d %s\n", 
		cfg.FullPowerStartHour, cfg.FullPowerStartMinute,
		cfg.FullPowerEndHour, cfg.FullPowerEndMinute,
		location.String())
	
	mode := cfg.GetModeString()
	fmt.Printf("🔋 Current mode: %s\n", mode)
	
	timeUntilChange := cfg.GetTimeUntilModeChange()
	fmt.Printf("⏰ Time until mode change: %v\n\n", timeUntilChange)

	// Initialize database
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Load domains from CSV
	domains, err := loadDomainsFromCSV(cfg.CSVFile)
	if err != nil {
		log.Fatal("Failed to load domains:", err)
	}

	fmt.Printf("📊 Loaded %d domains from CSV\n", len(domains))
	log.Printf("Loaded %d domains from %s", len(domains), cfg.CSVFile)

	// Set up graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-c
		fmt.Println("\n🛑 Received shutdown signal, gracefully stopping...")
		log.Printf("Received shutdown signal")
		
		// Give some time for graceful shutdown
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()

	// Initialize scanner
	scannerInstance := scanner.New(cfg, db)

	// Start the reconnaissance process
	fmt.Println("🎯 Starting reconnaissance process...")
	log.Printf("Starting reconnaissance with %d domains", len(domains))
	
	err = scannerInstance.Run(domains)
	if err != nil {
		log.Fatal("Scanner failed:", err)
	}

	fmt.Println("✅ Reconnaissance completed successfully!")
	log.Printf("=== RECON SCANNER COMPLETED ===")
}

func loadDomainsFromCSV(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	var domains []string
	// Skip header row
	for i := 1; i < len(records); i++ {
		if len(records[i]) >= 2 {
			domain := utils.CleanDomain(records[i][1])
			if domain != "" {
				domains = append(domains, domain)
			}
		}
	}

	return domains, nil
}