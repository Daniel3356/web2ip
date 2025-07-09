package monitoring

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

type SystemMonitor struct {
	config       *config.HighPerformanceConfig
	metrics      *SystemMetrics
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	alertChannel chan Alert
}

type SystemMetrics struct {
	CPUTemp         float64
	MemoryUsage     int64
	MemoryPercent   float64
	LoadAvg         float64
	ActiveWorkers   int
	ProcessedItems  int64
	ErrorRate       float64
	NetworkErrors   int64
	LastUpdated     time.Time
}

type Alert struct {
	Type      string
	Level     string
	Message   string
	Timestamp time.Time
}

func NewSystemMonitor(config *config.HighPerformanceConfig) *SystemMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &SystemMonitor{
		config:       config,
		metrics:      &SystemMetrics{},
		ctx:          ctx,
		cancel:       cancel,
		alertChannel: make(chan Alert, 100),
	}
}

func (sm *SystemMonitor) Start() {
	go sm.monitorLoop()
	go sm.alertHandler()
}

func (sm *SystemMonitor) Stop() {
	sm.cancel()
}

func (sm *SystemMonitor) monitorLoop() {
	tempTicker := time.NewTicker(sm.config.TempCheckInterval)
	memoryTicker := time.NewTicker(sm.config.MemoryCheckInterval)
	healthTicker := time.NewTicker(sm.config.HealthCheckInterval)
	
	defer tempTicker.Stop()
	defer memoryTicker.Stop()
	defer healthTicker.Stop()
	
	for {
		select {
		case <-tempTicker.C:
			sm.updateCPUTemperature()
		case <-memoryTicker.C:
			sm.updateMemoryUsage()
		case <-healthTicker.C:
			sm.performHealthCheck()
		case <-sm.ctx.Done():
			return
		}
	}
}

func (sm *SystemMonitor) updateCPUTemperature() {
	temp := sm.getCPUTemperature()
	
	sm.mu.Lock()
	sm.metrics.CPUTemp = temp
	sm.metrics.LastUpdated = time.Now()
	sm.mu.Unlock()
	
	if temp > sm.config.MaxCPUTemp {
		sm.sendAlert(Alert{
			Type:      "THERMAL",
			Level:     "CRITICAL",
			Message:   fmt.Sprintf("CPU temperature critical: %.1f°C", temp),
			Timestamp: time.Now(),
		})
	} else if temp > sm.config.ThrottleTemp {
		sm.sendAlert(Alert{
			Type:      "THERMAL",
			Level:     "WARNING",
			Message:   fmt.Sprintf("CPU temperature high: %.1f°C", temp),
			Timestamp: time.Now(),
		})
	}
}

func (sm *SystemMonitor) updateMemoryUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	usage := int64(m.Alloc)
	percent := float64(usage) / float64(sm.config.MaxMemoryUsage) * 100
	
	sm.mu.Lock()
	sm.metrics.MemoryUsage = usage
	sm.metrics.MemoryPercent = percent
	sm.metrics.LastUpdated = time.Now()
	sm.mu.Unlock()
	
	if usage > sm.config.MaxMemoryUsage {
		sm.sendAlert(Alert{
			Type:      "MEMORY",
			Level:     "CRITICAL",
			Message:   fmt.Sprintf("Memory usage critical: %d MB", usage/1024/1024),
			Timestamp: time.Now(),
		})
		runtime.GC()
	} else if usage > sm.config.GCThreshold {
		sm.sendAlert(Alert{
			Type:      "MEMORY",
			Level:     "WARNING",
			Message:   fmt.Sprintf("Memory usage high: %d MB", usage/1024/1024),
			Timestamp: time.Now(),
		})
		runtime.GC()
	}
}

func (sm *SystemMonitor) performHealthCheck() {
	sm.mu.RLock()
	metrics := *sm.metrics
	sm.mu.RUnlock()
	
	log.Printf("System Health Check - CPU: %.1f°C, Memory: %.1f%%, Workers: %d, Processed: %d, Errors: %.2f%%",
		metrics.CPUTemp, metrics.MemoryPercent, metrics.ActiveWorkers, metrics.ProcessedItems, metrics.ErrorRate)
}

func (sm *SystemMonitor) getCPUTemperature() float64 {
	if runtime.GOOS != "linux" {
		// Return a simulated temperature for non-Linux systems
		return 45.0 + float64(time.Now().Second()%20)
	}
	
	data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err != nil {
		return 0
	}
	
	tempStr := strings.TrimSpace(string(data))
	temp, err := strconv.Atoi(tempStr)
	if err != nil {
		return 0
	}
	
	return float64(temp) / 1000.0
}

func (sm *SystemMonitor) sendAlert(alert Alert) {
	select {
	case sm.alertChannel <- alert:
	default:
		log.Printf("Alert channel full, dropping alert: %s", alert.Message)
	}
}

func (sm *SystemMonitor) alertHandler() {
	for {
		select {
		case alert := <-sm.alertChannel:
			log.Printf("[%s] %s: %s", alert.Level, alert.Type, alert.Message)
		case <-sm.ctx.Done():
			return
		}
	}
}

func (sm *SystemMonitor) GetMetrics() SystemMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return *sm.metrics
}

func (sm *SystemMonitor) ShouldThrottle() bool {
	sm.mu.RLock()
	temp := sm.metrics.CPUTemp
	memory := sm.metrics.MemoryPercent
	sm.mu.RUnlock()
	
	return temp > sm.config.ThrottleTemp || memory > 75.0
}

func (sm *SystemMonitor) GetOptimalWorkerCount() int {
	sm.mu.RLock()
	temp := sm.metrics.CPUTemp
	memory := sm.metrics.MemoryPercent
	sm.mu.RUnlock()
	
	if temp > sm.config.MaxCPUTemp || memory > 90.0 {
		return sm.config.MinWorkers
	}
	
	if temp > sm.config.ThrottleTemp || memory > 75.0 {
		return sm.config.MaxWorkers / 2
	}
	
	return sm.config.MaxWorkers
}

func (sm *SystemMonitor) UpdateStats(workers int, processed int64, errorRate float64) {
	sm.mu.Lock()
	sm.metrics.ActiveWorkers = workers
	sm.metrics.ProcessedItems = processed
	sm.metrics.ErrorRate = errorRate
	sm.metrics.LastUpdated = time.Now()
	sm.mu.Unlock()
}