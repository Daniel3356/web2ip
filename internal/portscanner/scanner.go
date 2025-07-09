package portscanner

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/recon-scanner/internal/config"
	"github.com/recon-scanner/internal/database"
	"github.com/recon-scanner/internal/pool"
)

type Scanner struct {
	config         *config.Config
	connectionPool *pool.ConnectionPool
}

func New(cfg *config.Config) *Scanner {
	var connectionPool *pool.ConnectionPool
	if cfg.EnableHighPerformanceMode {
		connectionPool = pool.NewConnectionPool(cfg)
	}
	
	return &Scanner{
		config:         cfg,
		connectionPool: connectionPool,
	}
}

func (s *Scanner) Close() {
	if s.connectionPool != nil {
		s.connectionPool.Close()
	}
}

func (s *Scanner) ScanPort(ip string, port int) (*database.PortResult, error) {
	result := &database.PortResult{
		IP:          ip,
		Port:        port,
		IsOpen:      false,
		ProcessedAt: time.Now(),
	}

	// Try with connection pool first if available
	if s.connectionPool != nil {
		return s.scanPortWithPool(result, ip, port)
	}
	
	// Fall back to direct connection
	return s.scanPortDirect(result, ip, port)
}

func (s *Scanner) scanPortWithPool(result *database.PortResult, ip string, port int) (*database.PortResult, error) {
	conn, err := s.connectionPool.GetConnection(ip, port)
	if err != nil {
		return result, nil // Port is closed, not an error
	}
	defer s.connectionPool.ReturnConnection(conn)

	result.IsOpen = true

	// Try to grab banner
	banner, service := s.grabBannerFromPooledConn(conn, port)
	result.Banner = banner
	result.Service = service

	return result, nil
}

func (s *Scanner) scanPortDirect(result *database.PortResult, ip string, port int) (*database.PortResult, error) {
	profile := s.config.GetCurrentProfile()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), profile.Timeout)
	if err != nil {
		return result, nil // Port is closed, not an error
	}
	defer conn.Close()

	result.IsOpen = true

	// Try to grab banner
	banner, service := s.grabBanner(conn, port)
	result.Banner = banner
	result.Service = service

	return result, nil
}

func (s *Scanner) grabBannerFromPooledConn(conn *pool.PooledConnection, port int) (string, string) {
	conn.SetReadDeadline(time.Now().Add(time.Second * 3))

	// For some services, we need to send a request first
	switch port {
	case 80, 8080, 3000, 8888, 5000, 8081:
		conn.Write([]byte("GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"))
	case 443, 8443:
		return "", "HTTPS"
	case 25, 587:
		// SMTP services usually send a greeting
	case 21:
		// FTP services usually send a greeting
	case 22:
		// SSH services usually send a greeting
	}

	// Read response
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return "", s.identifyServiceByPort(port)
	}

	banner := strings.TrimSpace(string(buffer[:n]))
	service := s.identifyService(banner, port)

	return banner, service
}

func (s *Scanner) grabBanner(conn net.Conn, port int) (string, string) {
	conn.SetReadDeadline(time.Now().Add(time.Second * 3))

	// For some services, we need to send a request first
	switch port {
	case 80, 8080, 3000, 8888, 5000, 8081:
		conn.Write([]byte("GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"))
	case 443, 8443:
		return "", "HTTPS"
	case 25, 587:
		// SMTP services usually send a greeting
	case 21:
		// FTP services usually send a greeting
	case 22:
		// SSH services usually send a greeting
	}

	// Read response
	scanner := bufio.NewScanner(conn)
	var lines []string
	maxLines := 3

	for scanner.Scan() && len(lines) < maxLines {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	banner := strings.Join(lines, "\n")
	service := s.identifyService(banner, port)

	return banner, service
}

func (s *Scanner) identifyService(banner string, port int) string {
	bannerLower := strings.ToLower(banner)

	// Port-based identification
	switch port {
	case 80, 8080, 3000, 8888, 5000, 8081:
		if strings.Contains(bannerLower, "http") {
			return "HTTP"
		}
	case 443, 8443:
		return "HTTPS"
	case 22:
		if strings.Contains(bannerLower, "ssh") {
			return "SSH"
		}
	case 21:
		if strings.Contains(bannerLower, "ftp") {
			return "FTP"
		}
	case 25, 587, 465:
		if strings.Contains(bannerLower, "smtp") {
			return "SMTP"
		}
	case 110, 995:
		if strings.Contains(bannerLower, "pop") {
			return "POP3"
		}
	case 143, 993:
		if strings.Contains(bannerLower, "imap") {
			return "IMAP"
		}
	case 3306:
		if strings.Contains(bannerLower, "mysql") {
			return "MySQL"
		}
	case 5432:
		if strings.Contains(bannerLower, "postgresql") {
			return "PostgreSQL"
		}
	case 6379:
		if strings.Contains(bannerLower, "redis") {
			return "Redis"
		}
	case 27017:
		if strings.Contains(bannerLower, "mongodb") {
			return "MongoDB"
		}
	}

	// Banner-based identification
	if strings.Contains(bannerLower, "apache") {
		return "Apache"
	}
	if strings.Contains(bannerLower, "nginx") {
		return "Nginx"
	}
	if strings.Contains(bannerLower, "microsoft") {
		return "Microsoft IIS"
	}

	return s.identifyServiceByPort(port)
}

func (s *Scanner) identifyServiceByPort(port int) string {
	switch port {
	case 80, 8080, 3000, 8888, 5000, 8081:
		return "HTTP"
	case 443, 8443:
		return "HTTPS"
	case 22:
		return "SSH"
	case 21:
		return "FTP"
	case 25, 587, 465:
		return "SMTP"
	case 110, 995:
		return "POP3"
	case 143, 993:
		return "IMAP"
	case 3306:
		return "MySQL"
	case 5432:
		return "PostgreSQL"
	case 6379:
		return "Redis"
	case 27017:
		return "MongoDB"
	case 139, 445:
		return "SMB"
	case 161:
		return "SNMP"
	case 3389:
		return "RDP"
	case 23:
		return "Telnet"
	case 1521:
		return "Oracle"
	case 1433:
		return "SQL Server"
	default:
		return "Unknown"
	}
}