package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

type DomainResult struct {
	Domain       string
	ARecords     []string
	AAAARecords  []string
	MXRecords    []string
	CNAMERecords []string
	NSRecords    []string
	TXTRecords   []string
	ProcessedAt  time.Time
}

type IPResult struct {
	IP           string
	PTRRecord    string
	OpenPorts    []int
	ProcessedAt  time.Time
}

type PortResult struct {
	IP          string
	Port        int
	IsOpen      bool
	Banner      string
	Service     string
	ProcessedAt time.Time
}

type Progress struct {
	Phase       string
	BatchIndex  int
	ItemIndex   int
	CompletedAt time.Time
}

func New(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	d := &Database{db: db}
	
	if err := d.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return d, nil
}

func (d *Database) createTables() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS domains (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			domain TEXT UNIQUE NOT NULL,
			a_records TEXT,
			aaaa_records TEXT,
			mx_records TEXT,
			cname_records TEXT,
			ns_records TEXT,
			txt_records TEXT,
			processed_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS ips (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ip TEXT UNIQUE NOT NULL,
			ptr_record TEXT,
			processed_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS ports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ip TEXT NOT NULL,
			port INTEGER NOT NULL,
			is_open BOOLEAN NOT NULL,
			banner TEXT,
			service TEXT,
			processed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(ip, port)
		)`,
		`CREATE TABLE IF NOT EXISTS progress (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			phase TEXT NOT NULL,
			batch_index INTEGER NOT NULL,
			item_index INTEGER NOT NULL,
			completed_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS ssl_certificates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ip TEXT NOT NULL,
			port INTEGER NOT NULL,
			common_name TEXT,
			subject_alt_names TEXT,
			issuer TEXT,
			valid_from DATETIME,
			valid_to DATETIME,
			processed_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, table := range tables {
		if _, err := d.db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

func (d *Database) SaveDomain(result *DomainResult) error {
	query := `INSERT OR REPLACE INTO domains 
		(domain, a_records, aaaa_records, mx_records, cname_records, ns_records, txt_records, processed_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	
	_, err := d.db.Exec(query, 
		result.Domain,
		joinStrings(result.ARecords),
		joinStrings(result.AAAARecords),
		joinStrings(result.MXRecords),
		joinStrings(result.CNAMERecords),
		joinStrings(result.NSRecords),
		joinStrings(result.TXTRecords),
		result.ProcessedAt,
	)
	
	return err
}

func (d *Database) SaveIP(result *IPResult) error {
	query := `INSERT OR REPLACE INTO ips (ip, ptr_record, processed_at) VALUES (?, ?, ?)`
	_, err := d.db.Exec(query, result.IP, result.PTRRecord, result.ProcessedAt)
	return err
}

func (d *Database) SavePort(result *PortResult) error {
	query := `INSERT OR REPLACE INTO ports (ip, port, is_open, banner, service, processed_at) 
		VALUES (?, ?, ?, ?, ?, ?)`
	_, err := d.db.Exec(query, 
		result.IP, 
		result.Port, 
		result.IsOpen, 
		result.Banner, 
		result.Service, 
		result.ProcessedAt,
	)
	return err
}

func (d *Database) SaveProgress(progress *Progress) error {
	query := `INSERT INTO progress (phase, batch_index, item_index, completed_at) VALUES (?, ?, ?, ?)`
	_, err := d.db.Exec(query, progress.Phase, progress.BatchIndex, progress.ItemIndex, progress.CompletedAt)
	return err
}

func (d *Database) GetLastProgress(phase string) (*Progress, error) {
	query := `SELECT phase, batch_index, item_index, completed_at FROM progress 
		WHERE phase = ? ORDER BY completed_at DESC LIMIT 1`
	
	var p Progress
	err := d.db.QueryRow(query, phase).Scan(&p.Phase, &p.BatchIndex, &p.ItemIndex, &p.CompletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No previous progress
		}
		return nil, err
	}
	
	return &p, nil
}

func (d *Database) GetProcessedDomains() (map[string]bool, error) {
	query := `SELECT domain FROM domains`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	processed := make(map[string]bool)
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			return nil, err
		}
		processed[domain] = true
	}

	return processed, nil
}

func (d *Database) GetAllIPsFromDomains() ([]string, error) {
	query := `SELECT a_records, aaaa_records, mx_records FROM domains WHERE a_records != '' OR aaaa_records != '' OR mx_records != ''`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allIPs []string
	ipMap := make(map[string]bool)

	for rows.Next() {
		var aRecords, aaaaRecords, mxRecords sql.NullString
		if err := rows.Scan(&aRecords, &aaaaRecords, &mxRecords); err != nil {
			return nil, err
		}

		// Parse A records
		if aRecords.Valid && aRecords.String != "" {
			ips := strings.Split(aRecords.String, ",")
			for _, ip := range ips {
				ip = strings.TrimSpace(ip)
				if ip != "" && !ipMap[ip] {
					ipMap[ip] = true
					allIPs = append(allIPs, ip)
				}
			}
		}

		// Parse AAAA records
		if aaaaRecords.Valid && aaaaRecords.String != "" {
			ips := strings.Split(aaaaRecords.String, ",")
			for _, ip := range ips {
				ip = strings.TrimSpace(ip)
				if ip != "" && !ipMap[ip] {
					ipMap[ip] = true
					allIPs = append(allIPs, ip)
				}
			}
		}

		// For MX records, we would need to resolve them to IPs
		// For now, let's skip MX IPs to keep it simple
	}

	return allIPs, nil
}

func (d *Database) GetUniqueIPs() ([]string, error) {
	query := `SELECT DISTINCT ip FROM ips`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ips []string
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, err
		}
		ips = append(ips, ip)
	}

	return ips, nil
}

func (d *Database) IsPortScanned(ip string, port int) (bool, error) {
	query := `SELECT COUNT(*) FROM ports WHERE ip = ? AND port = ?`
	var count int
	err := d.db.QueryRow(query, ip, port).Scan(&count)
	return count > 0, err
}

func (d *Database) Close() error {
	return d.db.Close()
}

func joinStrings(slice []string) string {
	if len(slice) == 0 {
		return ""
	}
	return strings.Join(slice, ",")
}