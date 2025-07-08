package dns

import (
	"net"
	"strings"
	"time"

	"github.com/recon-scanner/internal/config"
	"github.com/recon-scanner/internal/database"
)

type Resolver struct {
	config *config.Config
}

func New(cfg *config.Config) *Resolver {
	return &Resolver{config: cfg}
}

func (r *Resolver) ResolveDomain(domain string) (*database.DomainResult, error) {
	result := &database.DomainResult{
		Domain:      domain,
		ProcessedAt: time.Now(),
	}

	// A and AAAA records
	if ips, err := net.LookupIP(domain); err == nil {
		for _, ip := range ips {
			if ip.To4() != nil {
				result.ARecords = append(result.ARecords, ip.String())
			} else {
				result.AAAARecords = append(result.AAAARecords, ip.String())
			}
		}
	}

	// MX records
	if mxRecords, err := net.LookupMX(domain); err == nil {
		for _, mx := range mxRecords {
			result.MXRecords = append(result.MXRecords, mx.Host)
		}
	}

	// CNAME record
	if cname, err := net.LookupCNAME(domain); err == nil {
		if cname != domain+"." { // Only add if it's actually a CNAME
			result.CNAMERecords = append(result.CNAMERecords, strings.TrimSuffix(cname, "."))
		}
	}

	// NS records
	if nsRecords, err := net.LookupNS(domain); err == nil {
		for _, ns := range nsRecords {
			result.NSRecords = append(result.NSRecords, strings.TrimSuffix(ns.Host, "."))
		}
	}

	// TXT records
	if txtRecords, err := net.LookupTXT(domain); err == nil {
		result.TXTRecords = txtRecords
	}

	return result, nil
}

func (r *Resolver) ReverseLookup(ip string) (string, error) {
	names, err := net.LookupAddr(ip)
	if err != nil {
		return "", err
	}

	if len(names) > 0 {
		return strings.TrimSuffix(names[0], "."), nil
	}

	return "", nil
}