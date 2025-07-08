package utils

import (
	"strings"
)

func CleanDomain(domain string) string {
	// Remove quotes if present
	domain = strings.Trim(domain, `"`)
	
	// Remove protocol if present
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	
	// Remove www prefix
	domain = strings.TrimPrefix(domain, "www.")
	
	// Remove trailing slash
	domain = strings.TrimSuffix(domain, "/")
	
	// Convert to lowercase
	domain = strings.ToLower(domain)
	
	return strings.TrimSpace(domain)
}