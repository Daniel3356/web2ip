package pool

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/recon-scanner/internal/config"
)

// ConnectionPool manages a pool of network connections for high-performance scanning
type ConnectionPool struct {
	config       *config.Config
	pools        map[string]*HostPool // Map of target -> pool
	poolMutex    sync.RWMutex
	maxPools     int
	maxConnPerPool int
	globalConnCount int
	globalMutex     sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	cleanupTimer *time.Timer
}

// HostPool manages connections for a specific host
type HostPool struct {
	host        string
	connections chan *PooledConnection
	maxConns    int
	activeConns int
	lastUsed    time.Time
	mutex       sync.RWMutex
}

// PooledConnection wraps a network connection with pooling metadata
type PooledConnection struct {
	conn      net.Conn
	host      string
	port      int
	createdAt time.Time
	lastUsed  time.Time
	useCount  int
	pool      *HostPool
}

// NewConnectionPool creates a new connection pool with the specified configuration
func NewConnectionPool(cfg *config.Config) *ConnectionPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	pool := &ConnectionPool{
		config:         cfg,
		pools:          make(map[string]*HostPool),
		maxPools:       cfg.ConnectionPoolSize / 10, // Limit number of host pools
		maxConnPerPool: cfg.MaxConnectionsPerWorker,
		ctx:            ctx,
		cancel:         cancel,
	}
	
	// Start cleanup routine
	go pool.cleanupRoutine()
	
	return pool
}

// GetConnection retrieves a connection from the pool or creates a new one
func (cp *ConnectionPool) GetConnection(host string, port int) (*PooledConnection, error) {
	target := fmt.Sprintf("%s:%d", host, port)
	
	// Get or create host pool
	hostPool := cp.getOrCreateHostPool(target)
	if hostPool == nil {
		// Pool limit reached, create direct connection
		return cp.createDirectConnection(host, port)
	}
	
	// Try to get connection from pool
	select {
	case conn := <-hostPool.connections:
		if cp.isConnectionValid(conn) {
			conn.lastUsed = time.Now()
			conn.useCount++
			return conn, nil
		}
		// Connection is invalid, close it and create new one
		conn.Close()
		return cp.createPooledConnection(host, port, hostPool)
	default:
		// No available connections, create new one
		return cp.createPooledConnection(host, port, hostPool)
	}
}

// ReturnConnection returns a connection to the pool
func (cp *ConnectionPool) ReturnConnection(conn *PooledConnection) {
	if conn == nil || conn.pool == nil {
		return
	}
	
	if !cp.isConnectionValid(conn) {
		conn.Close()
		return
	}
	
	conn.lastUsed = time.Now()
	
	// Return to pool if there's space
	select {
	case conn.pool.connections <- conn:
		// Successfully returned to pool
	default:
		// Pool is full, close the connection
		conn.Close()
	}
}

// Close closes all connections in the pool
func (cp *ConnectionPool) Close() {
	cp.cancel()
	
	cp.poolMutex.Lock()
	defer cp.poolMutex.Unlock()
	
	for _, hostPool := range cp.pools {
		close(hostPool.connections)
		for conn := range hostPool.connections {
			conn.Close()
		}
	}
	
	cp.pools = make(map[string]*HostPool)
	
	if cp.cleanupTimer != nil {
		cp.cleanupTimer.Stop()
	}
}

// getOrCreateHostPool gets an existing host pool or creates a new one
func (cp *ConnectionPool) getOrCreateHostPool(target string) *HostPool {
	cp.poolMutex.RLock()
	hostPool, exists := cp.pools[target]
	cp.poolMutex.RUnlock()
	
	if exists {
		hostPool.mutex.Lock()
		hostPool.lastUsed = time.Now()
		hostPool.mutex.Unlock()
		return hostPool
	}
	
	// Create new host pool
	cp.poolMutex.Lock()
	defer cp.poolMutex.Unlock()
	
	// Check again in case another goroutine created it
	if hostPool, exists := cp.pools[target]; exists {
		return hostPool
	}
	
	// Check if we've reached the maximum number of pools
	if len(cp.pools) >= cp.maxPools {
		return nil
	}
	
	hostPool = &HostPool{
		host:        target,
		connections: make(chan *PooledConnection, cp.maxConnPerPool),
		maxConns:    cp.maxConnPerPool,
		lastUsed:    time.Now(),
	}
	
	cp.pools[target] = hostPool
	return hostPool
}

// createPooledConnection creates a new pooled connection
func (cp *ConnectionPool) createPooledConnection(host string, port int, hostPool *HostPool) (*PooledConnection, error) {
	// Check global connection limit
	cp.globalMutex.RLock()
	if cp.globalConnCount >= cp.config.ConnectionPoolSize {
		cp.globalMutex.RUnlock()
		return nil, fmt.Errorf("connection pool limit reached")
	}
	cp.globalMutex.RUnlock()
	
	// Create new connection
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), cp.config.ConnectionTimeout)
	if err != nil {
		return nil, err
	}
	
	// Configure connection
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(cp.config.KeepAlive)
	}
	
	cp.globalMutex.Lock()
	cp.globalConnCount++
	cp.globalMutex.Unlock()
	
	hostPool.mutex.Lock()
	hostPool.activeConns++
	hostPool.mutex.Unlock()
	
	return &PooledConnection{
		conn:      conn,
		host:      host,
		port:      port,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
		useCount:  1,
		pool:      hostPool,
	}, nil
}

// createDirectConnection creates a direct connection without pooling
func (cp *ConnectionPool) createDirectConnection(host string, port int) (*PooledConnection, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), cp.config.ConnectionTimeout)
	if err != nil {
		return nil, err
	}
	
	return &PooledConnection{
		conn:      conn,
		host:      host,
		port:      port,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
		useCount:  1,
	}, nil
}

// isConnectionValid checks if a connection is still valid
func (cp *ConnectionPool) isConnectionValid(conn *PooledConnection) bool {
	if conn == nil || conn.conn == nil {
		return false
	}
	
	// Check if connection is too old
	if time.Since(conn.createdAt) > cp.config.KeepAlive*2 {
		return false
	}
	
	// Check if connection has been used too many times
	if conn.useCount > 100 {
		return false
	}
	
	return true
}

// cleanupRoutine periodically cleans up old connections and pools
func (cp *ConnectionPool) cleanupRoutine() {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cp.cleanup()
		case <-cp.ctx.Done():
			return
		}
	}
}

// cleanup removes old connections and unused pools
func (cp *ConnectionPool) cleanup() {
	cp.poolMutex.Lock()
	defer cp.poolMutex.Unlock()
	
	cutoff := time.Now().Add(-cp.config.KeepAlive * 2)
	
	for target, hostPool := range cp.pools {
		hostPool.mutex.RLock()
		lastUsed := hostPool.lastUsed
		hostPool.mutex.RUnlock()
		
		if lastUsed.Before(cutoff) {
			// Remove old pool
			delete(cp.pools, target)
			
			// Close all connections in the pool
			close(hostPool.connections)
			for conn := range hostPool.connections {
				conn.Close()
			}
		} else {
			// Clean up old connections in the pool
			cp.cleanupHostPool(hostPool)
		}
	}
}

// cleanupHostPool removes old connections from a specific host pool
func (cp *ConnectionPool) cleanupHostPool(hostPool *HostPool) {
	cutoff := time.Now().Add(-cp.config.KeepAlive)
	
	// We need to drain and refill the channel to remove old connections
	var validConnections []*PooledConnection
	
	for {
		select {
		case conn := <-hostPool.connections:
			if conn.lastUsed.After(cutoff) && cp.isConnectionValid(conn) {
				validConnections = append(validConnections, conn)
			} else {
				conn.Close()
			}
		default:
			// No more connections in channel
			goto refill
		}
	}
	
refill:
	// Put valid connections back
	for _, conn := range validConnections {
		select {
		case hostPool.connections <- conn:
		default:
			// Channel is full, close excess connections
			conn.Close()
		}
	}
}

// Close closes a pooled connection
func (pc *PooledConnection) Close() error {
	if pc.conn == nil {
		return nil
	}
	
	// Decrease global connection count
	if pc.pool != nil {
		if cp := pc.pool; cp != nil {
			cp.mutex.Lock()
			cp.activeConns--
			cp.mutex.Unlock()
		}
	}
	
	return pc.conn.Close()
}

// Read reads data from the connection
func (pc *PooledConnection) Read(b []byte) (int, error) {
	if pc.conn == nil {
		return 0, fmt.Errorf("connection is nil")
	}
	return pc.conn.Read(b)
}

// Write writes data to the connection
func (pc *PooledConnection) Write(b []byte) (int, error) {
	if pc.conn == nil {
		return 0, fmt.Errorf("connection is nil")
	}
	return pc.conn.Write(b)
}

// SetReadDeadline sets the read deadline
func (pc *PooledConnection) SetReadDeadline(t time.Time) error {
	if pc.conn == nil {
		return fmt.Errorf("connection is nil")
	}
	return pc.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline
func (pc *PooledConnection) SetWriteDeadline(t time.Time) error {
	if pc.conn == nil {
		return fmt.Errorf("connection is nil")
	}
	return pc.conn.SetWriteDeadline(t)
}

// SetDeadline sets both read and write deadlines
func (pc *PooledConnection) SetDeadline(t time.Time) error {
	if pc.conn == nil {
		return fmt.Errorf("connection is nil")
	}
	return pc.conn.SetDeadline(t)
}

// LocalAddr returns the local address
func (pc *PooledConnection) LocalAddr() net.Addr {
	if pc.conn == nil {
		return nil
	}
	return pc.conn.LocalAddr()
}

// RemoteAddr returns the remote address
func (pc *PooledConnection) RemoteAddr() net.Addr {
	if pc.conn == nil {
		return nil
	}
	return pc.conn.RemoteAddr()
}

// GetStats returns connection pool statistics
func (cp *ConnectionPool) GetStats() map[string]interface{} {
	cp.poolMutex.RLock()
	cp.globalMutex.RLock()
	
	stats := map[string]interface{}{
		"total_pools":      len(cp.pools),
		"global_conn_count": cp.globalConnCount,
		"max_pools":        cp.maxPools,
		"max_conn_per_pool": cp.maxConnPerPool,
		"pools":            make(map[string]interface{}),
	}
	
	poolStats := stats["pools"].(map[string]interface{})
	for target, hostPool := range cp.pools {
		hostPool.mutex.RLock()
		poolStats[target] = map[string]interface{}{
			"active_conns": hostPool.activeConns,
			"max_conns":    hostPool.maxConns,
			"last_used":    hostPool.lastUsed,
		}
		hostPool.mutex.RUnlock()
	}
	
	cp.globalMutex.RUnlock()
	cp.poolMutex.RUnlock()
	
	return stats
}