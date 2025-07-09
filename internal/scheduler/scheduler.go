package scheduler

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/recon-scanner/internal/config"
)

type Scheduler struct {
	config          *config.Config
	currentMode     config.PerformanceMode
	modeChangeTimer *time.Timer
	ctx             context.Context
	cancel          context.CancelFunc
	
	// High-performance mode management
	systemMetrics   *SystemMetrics
	resourceMonitor *ResourceMonitor
	throttleLevel   int  // 0-100, percentage of throttling
	errorRate       float64
	mutex           sync.RWMutex
}

type SystemMetrics struct {
	CPUTemperature  float64
	MemoryUsage     int64
	MemoryPercent   float64
	CPULoad         float64
	GoroutineCount  int
	ErrorCount      int64
	SuccessCount    int64
	LastUpdateTime  time.Time
	mutex           sync.RWMutex
}

type ResourceMonitor struct {
	config          *config.Config
	metrics         *SystemMetrics
	alertThresholds AlertThresholds
	circuitBreaker  *CircuitBreaker
}

type AlertThresholds struct {
	HighMemoryUsage   float64
	CriticalMemoryUsage float64
	HighTemperature   float64
	CriticalTemperature float64
	HighCPULoad       float64
	CriticalCPULoad   float64
	HighErrorRate     float64
	CriticalErrorRate float64
}

type CircuitBreaker struct {
	failureCount     int
	lastFailureTime  time.Time
	state            string // "closed", "open", "half-open"
	threshold        int
	timeout          time.Duration
	mutex            sync.RWMutex
}

func New(cfg *config.Config) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	
	systemMetrics := &SystemMetrics{}
	resourceMonitor := &ResourceMonitor{
		config:  cfg,
		metrics: systemMetrics,
		alertThresholds: AlertThresholds{
			HighMemoryUsage:     0.70,
			CriticalMemoryUsage: cfg.MemoryPressureThreshold,
			HighTemperature:     float64(cfg.HighThermalThrottleTemp),
			CriticalTemperature: float64(cfg.ThermalThrottleTemp),
			HighCPULoad:         0.80,
			CriticalCPULoad:     cfg.CpuLoadThreshold,
			HighErrorRate:       0.02,
			CriticalErrorRate:   cfg.ErrorRateThreshold,
		},
		circuitBreaker: &CircuitBreaker{
			threshold: 10,
			timeout:   time.Minute * 5,
			state:     "closed",
		},
	}
	
	scheduler := &Scheduler{
		config:          cfg,
		ctx:             ctx,
		cancel:          cancel,
		systemMetrics:   systemMetrics,
		resourceMonitor: resourceMonitor,
		throttleLevel:   0,
		errorRate:       0.0,
	}
	
	scheduler.updateCurrentMode()
	
	// Start resource monitoring if in high-performance mode
	if cfg.EnableHighPerformanceMode {
		go scheduler.monitorResources()
	}
	
	return scheduler
}

func (s *Scheduler) Start() {
	go s.run()
}

func (s *Scheduler) Stop() {
	if s.modeChangeTimer != nil {
		s.modeChangeTimer.Stop()
	}
	s.cancel()
}

func (s *Scheduler) GetCurrentMode() config.PerformanceMode {
	return s.currentMode
}

func (s *Scheduler) IsFullPowerMode() bool {
	return s.currentMode == config.FullPowerMode
}

func (s *Scheduler) run() {
	for {
		s.scheduleNextModeChange()
		
		select {
		case <-s.modeChangeTimer.C:
			s.updateCurrentMode()
			s.logModeChange()
			s.checkSystemResources()
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Scheduler) scheduleNextModeChange() {
	duration := s.config.GetTimeUntilModeChange()
	
	if s.modeChangeTimer != nil {
		s.modeChangeTimer.Stop()
	}
	
	s.modeChangeTimer = time.NewTimer(duration)
	
	nextChangeTime := time.Now().Add(duration)
	nextMode := "CONSERVATION"
	if !s.config.IsFullPowerTime() {
		nextMode = "FULL POWER"
	}
	
	log.Printf("Next mode change to %s scheduled for %s (in %v)", 
		nextMode, nextChangeTime.Format("2006-01-02 15:04:05 MST"), duration)
}

func (s *Scheduler) updateCurrentMode() {
	if s.config.EnableHighPerformanceMode && s.config.IsHighPerformanceTime() {
		s.currentMode = config.HighPerformanceMode
	} else if s.config.IsFullPowerTime() {
		s.currentMode = config.FullPowerMode
	} else {
		s.currentMode = config.ConservationMode
	}
}

func (s *Scheduler) logModeChange() {
	mode := s.config.GetModeString()
	profile := s.config.GetCurrentProfile()
	
	location, _ := time.LoadLocation(s.config.Timezone)
	now := time.Now().In(location)
	
	fmt.Printf("\nMODE CHANGE at %s\n", now.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("Current Mode: %s\n", mode)
	fmt.Printf("Workers: %d | Batch Size: %d | Delay: %v\n", 
		profile.WorkerCount, profile.BatchSize, profile.RequestDelay)
	
	log.Printf("Mode changed to %s - Workers: %d, Batch: %d, Delay: %v",
		mode, profile.WorkerCount, profile.BatchSize, profile.RequestDelay)
}

func (s *Scheduler) checkSystemResources() {
	// Check memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	memUsageMB := int64(m.Alloc / 1024 / 1024)
	maxMemoryMB := s.config.MaxMemoryUsage / 1024 / 1024
	
	log.Printf("System check - Memory: %dMB/%dMB, Goroutines: %d", 
		memUsageMB, maxMemoryMB, runtime.NumGoroutine())
	
	// Check CPU temperature (Linux specific)
	if temp := s.getCPUTemperature(); temp > 0 {
		log.Printf("CPU Temperature: %.1f°C", temp)
		
		if temp > float64(s.config.ThermalThrottleTemp) {
			log.Printf("WARNING: CPU temperature high (%.1f°C), consider thermal throttling", temp)
		}
	}
	
	// Force garbage collection if memory usage is high
	if memUsageMB > maxMemoryMB/2 {
		log.Printf("High memory usage detected, forcing garbage collection")
		runtime.GC()
	}
}

func (s *Scheduler) getCPUTemperature() float64 {
	// This will only work on Raspberry Pi (Linux)
	if runtime.GOOS != "linux" {
		return 0
	}
	
	// Read CPU temperature from Raspberry Pi thermal zone
	data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err != nil {
		return 0
	}
	
	tempStr := strings.TrimSpace(string(data))
	temp, err := strconv.Atoi(tempStr)
	if err != nil {
		return 0
	}
	
	// Convert from millidegrees to degrees Celsius
	return float64(temp) / 1000.0
}

func (s *Scheduler) WaitForOptimalTime(operation string) {
	if s.IsFullPowerMode() {
		return // Already in optimal time
	}
	
	timeUntilFullPower := s.config.GetTimeUntilModeChange()
	
	// Only wait if we are close to full power time (within 2 hours)
	if timeUntilFullPower <= 2*time.Hour {
		fmt.Printf("Waiting %v for full power mode to start %s\n", 
			timeUntilFullPower, operation)
		
		select {
		case <-time.After(timeUntilFullPower):
			fmt.Printf("Full power mode started, continuing with %s\n", operation)
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Scheduler) ShouldThrottle() bool {
	return !s.IsFullPowerMode()
}

func (s *Scheduler) GetAdaptiveDelay(baseDelay time.Duration) time.Duration {
	profile := s.config.GetCurrentProfile()
	
	// During conservation mode, use configured delay
	if !s.IsFullPowerMode() && s.currentMode != config.HighPerformanceMode {
		return profile.RequestDelay
	}
	
	// In high-performance mode, consider throttling
	if s.currentMode == config.HighPerformanceMode {
		s.mutex.RLock()
		throttleLevel := s.throttleLevel
		s.mutex.RUnlock()
		
		if throttleLevel > 0 {
			// Apply throttling by increasing delay
			multiplier := 1.0 + (float64(throttleLevel) / 100.0)
			return time.Duration(float64(profile.RequestDelay) * multiplier)
		}
	}
	
	// During full power mode, potentially reduce delay based on system load
	temp := s.getCPUTemperature()
	if temp > float64(s.config.ThermalThrottleTemp-5) { // Preemptive throttling
		return profile.RequestDelay * 2
	}
	
	return profile.RequestDelay
}

// monitorResources continuously monitors system resources in high-performance mode
func (s *Scheduler) monitorResources() {
	ticker := time.NewTicker(s.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			s.updateSystemMetrics()
			s.assessSystemHealth()
			s.adjustThrottling()
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Scheduler) updateSystemMetrics() {
	s.systemMetrics.mutex.Lock()
	defer s.systemMetrics.mutex.Unlock()
	
	// Update memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	s.systemMetrics.MemoryUsage = int64(m.Alloc)
	s.systemMetrics.MemoryPercent = float64(m.Alloc) / float64(s.config.MaxMemoryUsage)
	s.systemMetrics.GoroutineCount = runtime.NumGoroutine()
	s.systemMetrics.CPUTemperature = s.getCPUTemperature()
	s.systemMetrics.LastUpdateTime = time.Now()
	
	// Calculate error rate
	if s.systemMetrics.SuccessCount+s.systemMetrics.ErrorCount > 0 {
		s.errorRate = float64(s.systemMetrics.ErrorCount) / float64(s.systemMetrics.SuccessCount+s.systemMetrics.ErrorCount)
	}
}

func (s *Scheduler) assessSystemHealth() {
	s.systemMetrics.mutex.RLock()
	defer s.systemMetrics.mutex.RUnlock()
	
	alerts := []string{}
	
	// Check memory pressure
	if s.systemMetrics.MemoryPercent > s.resourceMonitor.alertThresholds.CriticalMemoryUsage {
		alerts = append(alerts, fmt.Sprintf("CRITICAL: Memory usage %.1f%%", s.systemMetrics.MemoryPercent*100))
	} else if s.systemMetrics.MemoryPercent > s.resourceMonitor.alertThresholds.HighMemoryUsage {
		alerts = append(alerts, fmt.Sprintf("HIGH: Memory usage %.1f%%", s.systemMetrics.MemoryPercent*100))
	}
	
	// Check temperature
	if s.systemMetrics.CPUTemperature > s.resourceMonitor.alertThresholds.CriticalTemperature {
		alerts = append(alerts, fmt.Sprintf("CRITICAL: CPU temperature %.1f°C", s.systemMetrics.CPUTemperature))
	} else if s.systemMetrics.CPUTemperature > s.resourceMonitor.alertThresholds.HighTemperature {
		alerts = append(alerts, fmt.Sprintf("HIGH: CPU temperature %.1f°C", s.systemMetrics.CPUTemperature))
	}
	
	// Check error rate
	if s.errorRate > s.resourceMonitor.alertThresholds.CriticalErrorRate {
		alerts = append(alerts, fmt.Sprintf("CRITICAL: Error rate %.1f%%", s.errorRate*100))
	} else if s.errorRate > s.resourceMonitor.alertThresholds.HighErrorRate {
		alerts = append(alerts, fmt.Sprintf("HIGH: Error rate %.1f%%", s.errorRate*100))
	}
	
	// Check goroutine count
	if s.systemMetrics.GoroutineCount > 1000 {
		alerts = append(alerts, fmt.Sprintf("HIGH: Goroutines %d", s.systemMetrics.GoroutineCount))
	}
	
	// Log alerts
	for _, alert := range alerts {
		log.Printf("RESOURCE ALERT: %s", alert)
		if s.config.DetailedLogging {
			fmt.Printf("🚨 %s\n", alert)
		}
	}
}

func (s *Scheduler) adjustThrottling() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.systemMetrics.mutex.RLock()
	memPercent := s.systemMetrics.MemoryPercent
	temperature := s.systemMetrics.CPUTemperature
	s.systemMetrics.mutex.RUnlock()
	
	newThrottleLevel := 0
	
	// Calculate throttling based on memory pressure
	if memPercent > s.resourceMonitor.alertThresholds.CriticalMemoryUsage {
		newThrottleLevel = 75 // Heavy throttling
	} else if memPercent > s.resourceMonitor.alertThresholds.HighMemoryUsage {
		newThrottleLevel = 25 // Light throttling
	}
	
	// Calculate throttling based on temperature
	if temperature > s.resourceMonitor.alertThresholds.CriticalTemperature {
		newThrottleLevel = max(newThrottleLevel, 90) // Very heavy throttling
	} else if temperature > s.resourceMonitor.alertThresholds.HighTemperature {
		newThrottleLevel = max(newThrottleLevel, 50) // Medium throttling
	}
	
	// Calculate throttling based on error rate
	if s.errorRate > s.resourceMonitor.alertThresholds.CriticalErrorRate {
		newThrottleLevel = max(newThrottleLevel, 60) // Heavy throttling
	} else if s.errorRate > s.resourceMonitor.alertThresholds.HighErrorRate {
		newThrottleLevel = max(newThrottleLevel, 30) // Medium throttling
	}
	
	// Update throttle level
	if newThrottleLevel != s.throttleLevel {
		s.throttleLevel = newThrottleLevel
		
		if s.config.DetailedLogging {
			if newThrottleLevel > 0 {
				fmt.Printf("⚡ Throttling adjusted to %d%% (Memory: %.1f%%, Temp: %.1f°C, Errors: %.1f%%)\n", 
					newThrottleLevel, memPercent*100, temperature, s.errorRate*100)
			} else {
				fmt.Printf("✅ Throttling disabled - system running normally\n")
			}
		}
		
		log.Printf("Throttle level changed to %d%% (Memory: %.1f%%, Temp: %.1f°C, Errors: %.1f%%)", 
			newThrottleLevel, memPercent*100, temperature, s.errorRate*100)
	}
	
	// Force garbage collection if memory pressure is high
	if memPercent > s.resourceMonitor.alertThresholds.HighMemoryUsage {
		runtime.GC()
		if s.config.DetailedLogging {
			fmt.Printf("🗑️ Forced garbage collection due to memory pressure\n")
		}
	}
}

func (s *Scheduler) RecordError() {
	s.systemMetrics.mutex.Lock()
	s.systemMetrics.ErrorCount++
	s.systemMetrics.mutex.Unlock()
}

func (s *Scheduler) RecordSuccess() {
	s.systemMetrics.mutex.Lock()
	s.systemMetrics.SuccessCount++
	s.systemMetrics.mutex.Unlock()
}

func (s *Scheduler) GetSystemMetrics() *SystemMetrics {
	s.systemMetrics.mutex.RLock()
	defer s.systemMetrics.mutex.RUnlock()
	
	// Return a copy to prevent race conditions
	return &SystemMetrics{
		CPUTemperature:  s.systemMetrics.CPUTemperature,
		MemoryUsage:     s.systemMetrics.MemoryUsage,
		MemoryPercent:   s.systemMetrics.MemoryPercent,
		CPULoad:         s.systemMetrics.CPULoad,
		GoroutineCount:  s.systemMetrics.GoroutineCount,
		ErrorCount:      s.systemMetrics.ErrorCount,
		SuccessCount:    s.systemMetrics.SuccessCount,
		LastUpdateTime:  s.systemMetrics.LastUpdateTime,
	}
}

func (s *Scheduler) GetThrottleLevel() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.throttleLevel
}

func (s *Scheduler) IsHighPerformanceMode() bool {
	return s.currentMode == config.HighPerformanceMode
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}