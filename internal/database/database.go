package database

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DomainResult struct {
	Domain            string
	ARecords          []string
	AAAARecords       []string
	CNAMERecords      []string
	MXRecords         []string
	NSRecords         []string
	TXTRecords        []string
	ProcessedAt       time.Time
	DNSDuration       time.Duration
	PortScanDuration  time.Duration
	ReverseDuration   time.Duration
}

type Database struct {
	db *sql.DB
}

func New(path string) (*Database, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for concurrency
	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		log.Printf("Warning: could not enable WAL mode: %v", err)
	}

	// Create the domains table with per-phase duration columns
	createStmt := `
	CREATE TABLE IF NOT EXISTS domains (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		domain TEXT UNIQUE,
		a_records TEXT,
		aaaa_records TEXT,
		cname_records TEXT,
		mx_records TEXT,
		ns_records TEXT,
		txt_records TEXT,
		processed_at TEXT,
		dns_duration INTEGER,
		portscan_duration INTEGER,
		reverse_duration INTEGER
	);`
	if _, err := db.Exec(createStmt); err != nil {
		db.Close()
		return nil, err
	}

	return &Database{db: db}, nil
}

func (d *Database) SaveDomain(res *DomainResult) error {
	stmt := `
	INSERT OR REPLACE INTO domains (
		domain, a_records, aaaa_records, cname_records, mx_records, ns_records, txt_records, processed_at, dns_duration, portscan_duration, reverse_duration
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	_, err := d.db.Exec(
		stmt,
		res.Domain,
		joinStrings(res.ARecords),
		joinStrings(res.AAAARecords),
		joinStrings(res.CNAMERecords),
		joinStrings(res.MXRecords),
		joinStrings(res.NSRecords),
		joinStrings(res.TXTRecords),
		res.ProcessedAt.Format(time.RFC3339),
		int64(res.DNSDuration.Milliseconds()),
		int64(res.PortScanDuration.Milliseconds()),
		int64(res.ReverseDuration.Milliseconds()),
	)
	return err
}

func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func joinStrings(vals []string) string {
	result := ""
	for i, v := range vals {
		if i > 0 {
			result += ","
		}
		result += v
	}
	return result
}
