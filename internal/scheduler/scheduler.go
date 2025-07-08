package scheduler

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/recon-scanner/internal/config"
)

type Scheduler struct {
	config          *config.Config
	currentMode     config.PerformanceMode
	modeChangeTimer *time.Timer
	ctx             context.Context
	cancel          context.CancelFunc
}

func New(cfg *config.Config) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	
	scheduler := &Scheduler{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}
	
	scheduler.updateCurrentMode()
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
	if s.config.IsFullPowerTime() {
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
	if !s.IsFullPowerMode() {
		return profile.RequestDelay
	}
	
	// During full power mode, potentially reduce delay based on system load
	temp := s.getCPUTemperature()
	if temp > float64(s.config.ThermalThrottleTemp-5) { // Preemptive throttling
		return profile.RequestDelay * 2
	}
	
	return profile.RequestDelay
}