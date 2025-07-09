package dns

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/recon-scanner/internal/config"
	"github.com/recon-scanner/internal/database"
)

type Resolver struct {
	config *config.HighPerformanceConfig
}

func NewHighPerformance(cfg *config.HighPerformanceConfig) *Resolver {
	return &Resolver{config: cfg}
}

func New(cfg *config.Config) *Resolver {
	// Convert regular config to high-performance config for compatibility
	hpConfig := &config.HighPerformanceConfig{
		ConnectionTimeout: 10 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		RetryAttempts:     3,
	}
	return &Resolver{config: hpConfig}
}

func (r *Resolver) ResolveDomain(domain string) (*database.DomainResult, error) {
	result := &database.DomainResult{
		Domain:      domain,
		ProcessedAt: time.Now(),
	}

	// Set timeouts based on configuration
	timeout := r.config.ConnectionTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	// Create a context with timeout for all DNS operations
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: timeout,
			}
			return d.DialContext(ctx, network, address)
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// A and AAAA records
	if ips, err := resolver.LookupIPAddr(ctx, domain); err == nil {
		for _, ip := range ips {
			if ip.IP.To4() != nil {
				result.ARecords = append(result.ARecords, ip.IP.String())
			} else {
				result.AAAARecords = append(result.AAAARecords, ip.IP.String())
			}
		}
	}

	// MX records
	if mxRecords, err := resolver.LookupMX(ctx, domain); err == nil {
		for _, mx := range mxRecords {
			result.MXRecords = append(result.MXRecords, mx.Host)
		}
	}

	// CNAME record
	if cname, err := resolver.LookupCNAME(ctx, domain); err == nil {
		if cname != domain+"." { // Only add if it is actually a CNAME
			result.CNAMERecords = append(result.CNAMERecords, strings.TrimSuffix(cname, "."))
		}
	}

	// NS records
	if nsRecords, err := resolver.LookupNS(ctx, domain); err == nil {
		for _, ns := range nsRecords {
			result.NSRecords = append(result.NSRecords, strings.TrimSuffix(ns.Host, "."))
		}
	}

	// TXT records
	if txtRecords, err := resolver.LookupTXT(ctx, domain); err == nil {
		result.TXTRecords = txtRecords
	}

	return result, nil
}

func (r *Resolver) ReverseLookup(ip string) (string, error) {
	timeout := r.config.ConnectionTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: timeout,
			}
			return d.DialContext(ctx, network, address)
		},
	}

	names, err := resolver.LookupAddr(ctx, ip)
	if err != nil {
		return "", err
	}

	if len(names) > 0 {
		return strings.TrimSuffix(names[0], "."), nil
	}

	return "", nil
}