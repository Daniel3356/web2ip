package pool

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/recon-scanner/internal/config"
)

type Connection struct {
	conn        net.Conn
	lastUsed    time.Time
	inUse       bool
	host        string
	port        int
}

type ConnectionPool struct {
	config      *config.Config
	pools       map[string]*hostPool
	mutex       sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

type hostPool struct {
	connections []*Connection
	mutex       sync.Mutex
	host        string
	maxConns    int
	created     int
}

func NewConnectionPool(cfg *config.Config) *ConnectionPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	pool := &ConnectionPool{
		config: cfg,
		pools:  make(map[string]*hostPool),
		ctx:    ctx,
		cancel: cancel,
	}
	
	// Start cleanup routine
	go pool.cleanup()
	
	return pool
}

func (cp *ConnectionPool) GetConnection(host string, port int) (net.Conn, error) {
	hostKey := host // We could include port in the key if needed
	
	cp.mutex.RLock()
	pool, exists := cp.pools[hostKey]
	cp.mutex.RUnlock()
	
	if !exists {
		cp.mutex.Lock()
		// Double-check pattern
		if pool, exists = cp.pools[hostKey]; !exists {
			pool = &hostPool{
				host:     host,
				maxConns: cp.config.MaxConnectionsPerHost,
			}
			cp.pools[hostKey] = pool
		}
		cp.mutex.Unlock()
	}
	
	return pool.getConnection(port, cp.config.ConnectionTimeout)
}

func (cp *ConnectionPool) ReturnConnection(conn net.Conn, host string) {
	hostKey := host
	
	cp.mutex.RLock()
	pool, exists := cp.pools[hostKey]
	cp.mutex.RUnlock()
	
	if !exists {
		conn.Close()
		return
	}
	
	pool.returnConnection(conn)
}

func (cp *ConnectionPool) Close() {
	cp.cancel()
	
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	
	for _, pool := range cp.pools {
		pool.closeAll()
	}
	cp.pools = make(map[string]*hostPool)
}

func (cp *ConnectionPool) cleanup() {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cp.cleanupStaleConnections()
		case <-cp.ctx.Done():
			return
		}
	}
}

func (cp *ConnectionPool) cleanupStaleConnections() {
	cp.mutex.RLock()
	pools := make([]*hostPool, 0, len(cp.pools))
	for _, pool := range cp.pools {
		pools = append(pools, pool)
	}
	cp.mutex.RUnlock()
	
	for _, pool := range pools {
		pool.cleanup(cp.config.KeepAlive)
	}
}

func (hp *hostPool) getConnection(port int, timeout time.Duration) (net.Conn, error) {
	hp.mutex.Lock()
	defer hp.mutex.Unlock()
	
	// Try to find an available connection
	for i, conn := range hp.connections {
		if !conn.inUse && conn.port == port {
			// Check if connection is still valid
			if time.Since(conn.lastUsed) < time.Minute*5 {
				// Test connection
				if hp.testConnection(conn.conn) {
					conn.inUse = true
					conn.lastUsed = time.Now()
					return conn.conn, nil
				}
			}
			
			// Connection is stale, remove it
			conn.conn.Close()
			hp.connections = append(hp.connections[:i], hp.connections[i+1:]...)
			hp.created--
			break
		}
	}
	
	// Create new connection if pool not full
	if hp.created < hp.maxConns {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(hp.host, strconv.Itoa(port)), timeout)
		if err != nil {
			return nil, err
		}
		
		poolConn := &Connection{
			conn:     conn,
			lastUsed: time.Now(),
			inUse:    true,
			host:     hp.host,
			port:     port,
		}
		
		hp.connections = append(hp.connections, poolConn)
		hp.created++
		
		return conn, nil
	}
	
	// Pool is full, create temporary connection
	return net.DialTimeout("tcp", net.JoinHostPort(hp.host, strconv.Itoa(port)), timeout)
}

func (hp *hostPool) returnConnection(conn net.Conn) {
	hp.mutex.Lock()
	defer hp.mutex.Unlock()
	
	for _, poolConn := range hp.connections {
		if poolConn.conn == conn {
			poolConn.inUse = false
			poolConn.lastUsed = time.Now()
			return
		}
	}
	
	// Connection not from pool, close it
	conn.Close()
}

func (hp *hostPool) testConnection(conn net.Conn) bool {
	// Simple connection test
	conn.SetDeadline(time.Now().Add(time.Second))
	defer conn.SetDeadline(time.Time{})
	
	// Try to write/read a small amount of data
	_, err := conn.Write([]byte{})
	return err == nil
}

func (hp *hostPool) cleanup(maxAge time.Duration) {
	hp.mutex.Lock()
	defer hp.mutex.Unlock()
	
	var activeConnections []*Connection
	
	for _, conn := range hp.connections {
		if conn.inUse || time.Since(conn.lastUsed) < maxAge {
			activeConnections = append(activeConnections, conn)
		} else {
			conn.conn.Close()
			hp.created--
		}
	}
	
	hp.connections = activeConnections
}

func (hp *hostPool) closeAll() {
	hp.mutex.Lock()
	defer hp.mutex.Unlock()
	
	for _, conn := range hp.connections {
		conn.conn.Close()
	}
	hp.connections = nil
	hp.created = 0
}

func (cp *ConnectionPool) GetStats() map[string]interface{} {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	
	stats := make(map[string]interface{})
	totalPools := len(cp.pools)
	totalConnections := 0
	totalActive := 0
	
	for host, pool := range cp.pools {
		pool.mutex.Lock()
		hostStats := map[string]interface{}{
			"host":        host,
			"total":       len(pool.connections),
			"active":      0,
			"idle":        0,
			"max":         pool.maxConns,
		}
		
		for _, conn := range pool.connections {
			if conn.inUse {
				hostStats["active"] = hostStats["active"].(int) + 1
				totalActive++
			} else {
				hostStats["idle"] = hostStats["idle"].(int) + 1
			}
		}
		
		totalConnections += len(pool.connections)
		stats[fmt.Sprintf("host_%s", host)] = hostStats
		pool.mutex.Unlock()
	}
	
	stats["summary"] = map[string]interface{}{
		"total_pools":       totalPools,
		"total_connections": totalConnections,
		"total_active":      totalActive,
		"total_idle":        totalConnections - totalActive,
	}
	
	return stats
}