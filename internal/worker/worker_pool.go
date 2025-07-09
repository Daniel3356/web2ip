package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
	
	"github.com/recon-scanner/internal/config"
	"github.com/recon-scanner/internal/database"
	"github.com/recon-scanner/internal/dns"
	"github.com/recon-scanner/internal/monitoring"
)

type WorkerPool struct {
	config       *config.HighPerformanceConfig
	monitor      *monitoring.SystemMonitor
	db           *database.Database
	resolver     *dns.Resolver
	workers      []*Worker
	taskChan     chan Task
	resultChan   chan Result
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	activeCount  int32
	processedCount int64
	errorCount   int64
	mu           sync.RWMutex
}

type Task struct {
	ID       string
	Type     string
	Data     interface{}
	Priority int
	Retry    int
}

type Result struct {
	TaskID    string
	Success   bool
	Data      interface{}
	Error     error
	Duration  time.Duration
	Worker    int
}

type Worker struct {
	id       int
	pool     *WorkerPool
	taskChan chan Task
	quit     chan bool
}

func NewWorkerPool(config *config.HighPerformanceConfig, monitor *monitoring.SystemMonitor, db *database.Database, resolver *dns.Resolver) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &WorkerPool{
		config:     config,
		monitor:    monitor,
		db:         db,
		resolver:   resolver,
		taskChan:   make(chan Task, config.MaxWorkers*2),
		resultChan: make(chan Result, config.MaxWorkers*2),
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (wp *WorkerPool) Start() {
	wp.scaleWorkers(wp.config.MinWorkers)
	go wp.scaleManager()
	go wp.resultHandler()
}

func (wp *WorkerPool) Stop() {
	wp.cancel()
	close(wp.taskChan)
	wp.wg.Wait()
}

func (wp *WorkerPool) SubmitTask(task Task) {
	select {
	case wp.taskChan <- task:
	case <-wp.ctx.Done():
		return
	}
}

func (wp *WorkerPool) scaleManager() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			wp.adjustWorkerCount()
		case <-wp.ctx.Done():
			return
		}
	}
}

func (wp *WorkerPool) adjustWorkerCount() {
	optimalCount := wp.monitor.GetOptimalWorkerCount()
	currentCount := int(atomic.LoadInt32(&wp.activeCount))
	
	if optimalCount > currentCount {
		toAdd := optimalCount - currentCount
		if toAdd > wp.config.WorkerScaleStep {
			toAdd = wp.config.WorkerScaleStep
		}
		wp.scaleWorkers(toAdd)
		log.Printf("Scaled up workers by %d, total: %d", toAdd, currentCount+toAdd)
	} else if optimalCount < currentCount {
		toRemove := currentCount - optimalCount
		if toRemove > wp.config.WorkerScaleStep {
			toRemove = wp.config.WorkerScaleStep
		}
		wp.scaleDownWorkers(toRemove)
		log.Printf("Scaled down workers by %d, total: %d", toRemove, currentCount-toRemove)
	}
}

func (wp *WorkerPool) scaleWorkers(count int) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	for i := 0; i < count; i++ {
		worker := &Worker{
			id:       len(wp.workers),
			pool:     wp,
			taskChan: wp.taskChan,
			quit:     make(chan bool),
		}
		
		wp.workers = append(wp.workers, worker)
		wp.wg.Add(1)
		go worker.start()
		atomic.AddInt32(&wp.activeCount, 1)
	}
}

func (wp *WorkerPool) scaleDownWorkers(count int) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	if count > len(wp.workers) {
		count = len(wp.workers)
	}
	
	for i := 0; i < count; i++ {
		if len(wp.workers) > 0 {
			worker := wp.workers[len(wp.workers)-1]
			wp.workers = wp.workers[:len(wp.workers)-1]
			worker.stop()
			atomic.AddInt32(&wp.activeCount, -1)
		}
	}
}

func (wp *WorkerPool) resultHandler() {
	for {
		select {
		case result := <-wp.resultChan:
			wp.handleResult(result)
		case <-wp.ctx.Done():
			return
		}
	}
}

func (wp *WorkerPool) handleResult(result Result) {
	atomic.AddInt64(&wp.processedCount, 1)
	
	if !result.Success {
		atomic.AddInt64(&wp.errorCount, 1)
	}
	
	// Save result to database
	if result.Success && result.Data != nil {
		if domainResult, ok := result.Data.(*database.DomainResult); ok {
			wp.db.SaveDomain(domainResult)
		}
	}
	
	// Update monitor stats
	processed := atomic.LoadInt64(&wp.processedCount)
	errors := atomic.LoadInt64(&wp.errorCount)
	errorRate := float64(errors) / float64(processed) * 100
	
	wp.monitor.UpdateStats(int(atomic.LoadInt32(&wp.activeCount)), processed, errorRate)
}

func (w *Worker) start() {
	defer w.pool.wg.Done()
	
	for {
		select {
		case task := <-w.taskChan:
			w.processTask(task)
		case <-w.quit:
			return
		case <-w.pool.ctx.Done():
			return
		}
	}
}

func (w *Worker) stop() {
	close(w.quit)
}

func (w *Worker) processTask(task Task) {
	start := time.Now()
	var result Result
	
	// Add delay if system is under pressure
	if w.pool.monitor.ShouldThrottle() {
		time.Sleep(w.pool.config.RequestDelay * 10)
	} else {
		time.Sleep(w.pool.config.RequestDelay)
	}
	
	// Process the task based on type
	switch task.Type {
	case "DNS":
		result = w.processDNSTask(task)
	case "PORT":
		result = w.processPortTask(task)
	case "REVERSE":
		result = w.processReverseTask(task)
	default:
		result = Result{
			TaskID:   task.ID,
			Success:  false,
			Error:    fmt.Errorf("unknown task type: %s", task.Type),
			Duration: time.Since(start),
			Worker:   w.id,
		}
	}
	
	result.Duration = time.Since(start)
	result.Worker = w.id
	
	select {
	case w.pool.resultChan <- result:
	case <-w.pool.ctx.Done():
		return
	}
}

func (w *Worker) processDNSTask(task Task) Result {
	domain, ok := task.Data.(string)
	if !ok {
		return Result{
			TaskID:  task.ID,
			Success: false,
			Error:   fmt.Errorf("invalid domain data type"),
		}
	}
	
	domainResult, err := w.pool.resolver.ResolveDomain(domain)
	if err != nil {
		return Result{
			TaskID:  task.ID,
			Success: false,
			Error:   err,
		}
	}
	
	return Result{
		TaskID:  task.ID,
		Success: true,
		Data:    domainResult,
	}
}

func (w *Worker) processPortTask(task Task) Result {
	// Port scanning logic would go here
	return Result{
		TaskID:  task.ID,
		Success: true,
		Data:    task.Data,
	}
}

func (w *Worker) processReverseTask(task Task) Result {
	// Reverse DNS lookup logic would go here
	return Result{
		TaskID:  task.ID,
		Success: true,
		Data:    task.Data,
	}
}