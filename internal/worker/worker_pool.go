package worker

import (
	"fmt"
	"log"
	"time"

	"github.com/recon-scanner/internal/database"
	"github.com/recon-scanner/internal/dns"
)

type Task struct {
	ID       string
	Type     string
	Data     interface{}
	Priority int
	Retry    int
}

type Result struct {
	TaskID  string
	Success bool
	Data    *database.DomainResult
	Error   error
}

type WorkerPool struct {
	resolver *dns.Resolver
	db       *database.Database
}

func (wp *WorkerPool) processDomainTask(task Task) Result {
	domain, ok := task.Data.(string)
	if !ok {
		return Result{
			TaskID:  task.ID,
			Success: false,
			Error:   fmt.Errorf("invalid domain data type"),
		}
	}

	result := &database.DomainResult{
		Domain:      domain,
		ProcessedAt: time.Now(),
	}

	// DNS Phase
	startDNS := time.Now()
	dnsResult, err := wp.resolver.ResolveDomain(domain)
	result.DNSDuration = time.Since(startDNS)
	if err != nil {
		return Result{
			TaskID:  task.ID,
			Success: false,
			Error:   err,
		}
	}
	result.ARecords = dnsResult.ARecords
	result.AAAARecords = dnsResult.AAAARecords
	result.CNAMERecords = dnsResult.CNAMERecords
	result.MXRecords = dnsResult.MXRecords
	result.NSRecords = dnsResult.NSRecords
	result.TXTRecords = dnsResult.TXTRecords

	// PortScan Phase (stub example, replace with real logic if needed)
	startPort := time.Now()
	time.Sleep(10 * time.Millisecond)
	result.PortScanDuration = time.Since(startPort)

	// ReverseLookup Phase (stub example, replace with real logic if needed)
	startReverse := time.Now()
	time.Sleep(5 * time.Millisecond)
	result.ReverseDuration = time.Since(startReverse)

	if err := wp.db.SaveDomain(result); err != nil {
		log.Printf("Error saving domain result: %v", err)
		return Result{
			TaskID:  task.ID,
			Success: false,
			Error:   err,
		}
	}

	return Result{
		TaskID:  task.ID,
		Success: true,
		Data:    result,
	}
}
