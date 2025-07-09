package monitor

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

type SystemHealth struct {
	CPUTemperature   float64
	MemoryUsage      float64    // Percentage (0.0-1.0)
	MemoryUsageBytes int64      // Bytes
	GoroutineCount   int
	ErrorRate        float64    // Percentage (0.0-1.0)
	LastUpdate       time.Time
	IsHealthy        bool
	Alerts           []Alert
}

type Alert struct {
	Type      AlertType
	Message   string
	Timestamp time.Time
	Severity  AlertSeverity
}

type AlertType int

const (
	AlertMemory AlertType = iota
	AlertThermal
	AlertError
	AlertSystem
)

type AlertSeverity int

const (
	AlertInfo AlertSeverity = iota
	AlertWarning
	AlertCritical
)

type HealthMonitor struct {
	config       *config.Config
	health       *SystemHealth
	mutex        sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	alertChan    chan Alert
	errorCount   int64
	requestCount int64
	lastErrorRate float64
}

func NewHealthMonitor(cfg *config.Config) *HealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &HealthMonitor{
		config:    cfg,
		health:    &SystemHealth{},
		ctx:       ctx,
		cancel:    cancel,
		alertChan: make(chan Alert, 100),
	}
}

func (h *HealthMonitor) Start() {
	go h.monitor()
	go h.handleAlerts()
}

func (h *HealthMonitor) Stop() {
	h.cancel()
}

func (h *HealthMonitor) GetHealth() SystemHealth {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return *h.health
}

func (h *HealthMonitor) RecordError() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.errorCount++
}

func (h *HealthMonitor) RecordRequest() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.requestCount++
}

func (h *HealthMonitor) ShouldThrottle() bool {
	health := h.GetHealth()
	
	// Check memory usage
	if health.MemoryUsage > h.config.MemoryThrottleThreshold {
		return true
	}
	
	// Check temperature
	if health.CPUTemperature > h.config.ThermalThrottleThreshold {
		return true
	}
	
	// Check error rate
	if health.ErrorRate > h.config.ErrorRateThreshold {
		return true
	}
	
	return false
}

func (h *HealthMonitor) GetOptimalBatchSize(baseBatchSize int) int {
	health := h.GetHealth()
	profile := h.config.GetCurrentProfile()
	
	if !profile.DynamicBatchSizing {
		return baseBatchSize
	}
	
	// Start with base batch size
	batchSize := baseBatchSize
	
	// Adjust based on memory usage
	if profile.MemoryAwareBatching {
		if health.MemoryUsage > 0.7 {
			batchSize = int(float64(batchSize) * 0.5) // Reduce by 50%
		} else if health.MemoryUsage < 0.3 {
			batchSize = int(float64(batchSize) * 1.5) // Increase by 50%
		}
	}
	
	// Adjust based on temperature
	if profile.ThermalAwareBatching {
		if health.CPUTemperature > 70.0 {
			batchSize = int(float64(batchSize) * 0.6) // Reduce by 40%
		} else if health.CPUTemperature < 50.0 {
			batchSize = int(float64(batchSize) * 1.3) // Increase by 30%
		}
	}
	
	// Ensure within bounds
	if batchSize < profile.MinBatchSize {
		batchSize = profile.MinBatchSize
	}
	if batchSize > profile.MaxBatchSize {
		batchSize = profile.MaxBatchSize
	}
	
	return batchSize
}

func (h *HealthMonitor) monitor() {
	ticker := time.NewTicker(h.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			h.updateHealth()
		case <-h.ctx.Done():
			return
		}
	}
}

func (h *HealthMonitor) updateHealth() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	// Get memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	memoryUsageBytes := int64(m.Alloc)
	memoryUsage := float64(memoryUsageBytes) / float64(h.config.MaxMemoryUsage)
	
	// Get CPU temperature
	temperature := h.getCPUTemperature()
	
	// Calculate error rate
	errorRate := h.calculateErrorRate()
	
	// Check if system is healthy
	isHealthy := memoryUsage < h.config.MemoryThrottleThreshold &&
		temperature < h.config.ThermalThrottleThreshold &&
		errorRate < h.config.ErrorRateThreshold
	
	// Update health
	h.health.CPUTemperature = temperature
	h.health.MemoryUsage = memoryUsage
	h.health.MemoryUsageBytes = memoryUsageBytes
	h.health.GoroutineCount = runtime.NumGoroutine()
	h.health.ErrorRate = errorRate
	h.health.LastUpdate = time.Now()
	h.health.IsHealthy = isHealthy
	
	// Generate alerts
	h.checkAlerts()
	
	// Log health status
	log.Printf("Health: Memory %.1f%%, Temp %.1f°C, Errors %.1f%%, Goroutines %d, Healthy %v",
		memoryUsage*100, temperature, errorRate*100, h.health.GoroutineCount, isHealthy)
}

func (h *HealthMonitor) calculateErrorRate() float64 {
	if h.requestCount == 0 {
		return 0.0
	}
	
	errorRate := float64(h.errorCount) / float64(h.requestCount)
	h.lastErrorRate = errorRate
	
	// Reset counters periodically
	if h.requestCount > 10000 {
		h.errorCount = h.errorCount / 2
		h.requestCount = h.requestCount / 2
	}
	
	return errorRate
}

func (h *HealthMonitor) checkAlerts() {
	// Memory alerts
	if h.health.MemoryUsage > 0.95 {
		h.sendAlert(Alert{
			Type:      AlertMemory,
			Message:   fmt.Sprintf("Critical memory usage: %.1f%%", h.health.MemoryUsage*100),
			Timestamp: time.Now(),
			Severity:  AlertCritical,
		})
	} else if h.health.MemoryUsage > h.config.MemoryThrottleThreshold {
		h.sendAlert(Alert{
			Type:      AlertMemory,
			Message:   fmt.Sprintf("High memory usage: %.1f%%", h.health.MemoryUsage*100),
			Timestamp: time.Now(),
			Severity:  AlertWarning,
		})
	}
	
	// Thermal alerts
	if h.health.CPUTemperature > 80.0 {
		h.sendAlert(Alert{
			Type:      AlertThermal,
			Message:   fmt.Sprintf("Critical CPU temperature: %.1f°C", h.health.CPUTemperature),
			Timestamp: time.Now(),
			Severity:  AlertCritical,
		})
	} else if h.health.CPUTemperature > h.config.ThermalThrottleThreshold {
		h.sendAlert(Alert{
			Type:      AlertThermal,
			Message:   fmt.Sprintf("High CPU temperature: %.1f°C", h.health.CPUTemperature),
			Timestamp: time.Now(),
			Severity:  AlertWarning,
		})
	}
	
	// Error rate alerts
	if h.health.ErrorRate > 0.20 {
		h.sendAlert(Alert{
			Type:      AlertError,
			Message:   fmt.Sprintf("Critical error rate: %.1f%%", h.health.ErrorRate*100),
			Timestamp: time.Now(),
			Severity:  AlertCritical,
		})
	} else if h.health.ErrorRate > h.config.ErrorRateThreshold {
		h.sendAlert(Alert{
			Type:      AlertError,
			Message:   fmt.Sprintf("High error rate: %.1f%%", h.health.ErrorRate*100),
			Timestamp: time.Now(),
			Severity:  AlertWarning,
		})
	}
}

func (h *HealthMonitor) sendAlert(alert Alert) {
	select {
	case h.alertChan <- alert:
	default:
		// Alert channel is full, drop the alert
		log.Printf("Alert channel full, dropping alert: %s", alert.Message)
	}
}

func (h *HealthMonitor) handleAlerts() {
	for {
		select {
		case alert := <-h.alertChan:
			h.processAlert(alert)
		case <-h.ctx.Done():
			return
		}
	}
}

func (h *HealthMonitor) processAlert(alert Alert) {
	h.mutex.Lock()
	h.health.Alerts = append(h.health.Alerts, alert)
	
	// Keep only last 100 alerts
	if len(h.health.Alerts) > 100 {
		h.health.Alerts = h.health.Alerts[1:]
	}
	h.mutex.Unlock()
	
	// Log alert
	severity := "INFO"
	switch alert.Severity {
	case AlertWarning:
		severity = "WARNING"
	case AlertCritical:
		severity = "CRITICAL"
	}
	
	log.Printf("[%s] %s", severity, alert.Message)
	
	// Take action for critical alerts
	if alert.Severity == AlertCritical {
		switch alert.Type {
		case AlertMemory:
			log.Printf("Taking action for critical memory usage: forcing GC")
			runtime.GC()
		case AlertThermal:
			log.Printf("Taking action for critical temperature: requesting throttling")
			// The scheduler will check ShouldThrottle() and adjust accordingly
		}
	}
}

func (h *HealthMonitor) getCPUTemperature() float64 {
	// This will only work on Raspberry Pi (Linux)
	if runtime.GOOS != "linux" {
		return 0
	}
	
	// Try multiple thermal zones for better compatibility
	thermalZones := []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/class/thermal/thermal_zone1/temp",
		"/sys/devices/virtual/thermal/thermal_zone0/temp",
	}
	
	for _, zonePath := range thermalZones {
		if temp := h.readThermalZone(zonePath); temp > 0 {
			return temp
		}
	}
	
	return 0
}

func (h *HealthMonitor) readThermalZone(path string) float64 {
	data, err := os.ReadFile(path)
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