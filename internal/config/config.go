package config

import (
	"runtime"
	"time"
)

type PerformanceMode int

const (
	ConservationMode PerformanceMode = iota
	FullPowerMode
)

type Config struct {
	// File paths
	CSVFile      string
	DatabasePath string
	
	// Time-based configuration
	Timezone             string
	FullPowerStartHour   int
	FullPowerStartMinute int
	FullPowerEndHour     int
	FullPowerEndMinute   int
	
	// Performance profiles
	FullPower     PerformanceProfile
	Conservation  PerformanceProfile
	
	// Port lists
	WebPorts      []int
	InfraPorts    []int
	MailPorts     []int
	DatabasePorts []int
	
	// Resumption
	CheckpointInterval time.Duration
	
	// Raspberry Pi specific
	ThermalThrottleTemp int
	MaxMemoryUsage      int64
}

type PerformanceProfile struct {
	BatchSize       int
	WorkerCount     int
	RequestDelay    time.Duration
	Timeout         time.Duration
	MaxConcurrentIP int
}

func New() *Config {
	// Raspberry Pi 5 has 4 cores (ARM Cortex-A76)
	cpuCores := runtime.NumCPU()
	
	return &Config{
		CSVFile:      "top10milliondomains.csv",
		DatabasePath: "recon_results.db",
		
		// Toronto timezone with full power from 1:37 AM to 6:30 AM
		Timezone:             "America/Toronto",
		FullPowerStartHour:   1,
		FullPowerStartMinute: 37,
		FullPowerEndHour:     6,
		FullPowerEndMinute:   30,
		
		// Full power profile (night time - 1:37 AM to 6:30 AM)
		FullPower: PerformanceProfile{
			BatchSize:       5000,                    // Reduced for Pi 5
			WorkerCount:     cpuCores * 3,          // 12 workers for 4 cores
			RequestDelay:    time.Millisecond * 5,   // Faster during night
			Timeout:         time.Second * 8,        // Longer timeout for stability
			MaxConcurrentIP: 200,                    // Concurrent IP scans
		},
		
		// Conservation profile (day time - 6:30 AM to 1:37 AM)
		Conservation: PerformanceProfile{
			BatchSize:       500,                     // Much smaller batches
			WorkerCount:     cpuCores / 2,           // 2 workers only
			RequestDelay:    time.Millisecond * 100, // Much slower during day
			Timeout:         time.Second * 3,        // Shorter timeout
			MaxConcurrentIP: 10,                     // Very limited concurrent scans
		},
		
		WebPorts:      []int{80, 443, 3000, 8080, 8888, 8443, 5000},
		InfraPorts:    []int{21, 22, 23, 139, 161, 3389},
		MailPorts:     []int{25, 465, 587, 110, 995, 143, 993},
		DatabasePorts: []int{3306, 5432, 6379, 27017, 1521, 1433},
		
		CheckpointInterval:  time.Minute * 3,
		ThermalThrottleTemp: 70, // Celsius - throttle if CPU gets too hot
		MaxMemoryUsage:      12 * 1024 * 1024 * 1024, // 12GB of 16GB available
	}
}

func (c *Config) GetCurrentProfile() PerformanceProfile {
	if c.IsFullPowerTime() {
		return c.FullPower
	}
	return c.Conservation
}

func (c *Config) IsFullPowerTime() bool {
	location, err := time.LoadLocation(c.Timezone)
	if err != nil {
		location = time.UTC
	}
	
	now := time.Now().In(location)
	
	// Create time objects for start and end times
	startTime := time.Date(now.Year(), now.Month(), now.Day(), 
		c.FullPowerStartHour, c.FullPowerStartMinute, 0, 0, location)
	endTime := time.Date(now.Year(), now.Month(), now.Day(), 
		c.FullPowerEndHour, c.FullPowerEndMinute, 0, 0, location)
	
	// Handle case where end time is next day (crosses midnight)
	if endTime.Before(startTime) {
		if now.After(startTime) {
			// Current time is after start time, so end time is tomorrow
			endTime = endTime.AddDate(0, 0, 1)
		} else {
			// Current time is before start time, so start time was yesterday
			startTime = startTime.AddDate(0, 0, -1)
		}
	}
	
	return now.After(startTime) && now.Before(endTime)
}

func (c *Config) GetTimeUntilModeChange() time.Duration {
	location, err := time.LoadLocation(c.Timezone)
	if err != nil {
		location = time.UTC
	}
	
	now := time.Now().In(location)
	
	if c.IsFullPowerTime() {
		// Calculate time until end of full power mode
		endTime := time.Date(now.Year(), now.Month(), now.Day(), 
			c.FullPowerEndHour, c.FullPowerEndMinute, 0, 0, location)
		
		if endTime.Before(now) {
			endTime = endTime.AddDate(0, 0, 1)
		}
		
		return endTime.Sub(now)
	} else {
		// Calculate time until start of full power mode
		startTime := time.Date(now.Year(), now.Month(), now.Day(), 
			c.FullPowerStartHour, c.FullPowerStartMinute, 0, 0, location)
		
		if startTime.Before(now) {
			startTime = startTime.AddDate(0, 0, 1)
		}
		
		return startTime.Sub(now)
	}
}

func (c *Config) AllPorts() []int {
	var allPorts []int
	allPorts = append(allPorts, c.WebPorts...)
	allPorts = append(allPorts, c.InfraPorts...)
	allPorts = append(allPorts, c.MailPorts...)
	allPorts = append(allPorts, c.DatabasePorts...)
	return allPorts
}

func (c *Config) GetModeString() string {
	if c.IsFullPowerTime() {
		return "🌙 FULL POWER"
	}
	return "☀️ CONSERVATION"
}