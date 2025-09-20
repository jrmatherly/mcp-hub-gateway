package gateway

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// ConnectionPool manages a pool of HTTP connections for improved performance
type ConnectionPool struct {
	// Configuration
	maxConnections     int
	maxIdleConnections int
	idleTimeout        time.Duration
	connectionTimeout  time.Duration

	// Connection tracking
	connections map[string]*PooledConnection
	activeCount int
	mu          sync.RWMutex

	// Metrics
	metrics *TransportMetrics

	// HTTP client with connection pooling
	httpClient *http.Client
	transport  *http.Transport

	// Cleanup
	done            chan struct{}
	cleanupInterval time.Duration
}

// PooledConnection represents a pooled connection
type PooledConnection struct {
	conn       net.Conn
	createdAt  time.Time
	lastUsedAt time.Time
	inUse      bool
	id         string
}

// ConnectionPoolConfig configures the connection pool
type ConnectionPoolConfig struct {
	MaxConnections     int
	MaxIdleConnections int
	IdleTimeout        time.Duration
	ConnectionTimeout  time.Duration
	CleanupInterval    time.Duration
}

// DefaultConnectionPoolConfig returns default configuration
func DefaultConnectionPoolConfig() *ConnectionPoolConfig {
	return &ConnectionPoolConfig{
		MaxConnections:     100,
		MaxIdleConnections: 50,
		IdleTimeout:        90 * time.Second,
		ConnectionTimeout:  30 * time.Second,
		CleanupInterval:    60 * time.Second,
	}
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config *ConnectionPoolConfig, metrics *TransportMetrics) *ConnectionPool {
	if config == nil {
		config = DefaultConnectionPoolConfig()
	}

	// Create HTTP transport with connection pooling
	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConnections,
		MaxIdleConnsPerHost: config.MaxIdleConnections / 2,
		MaxConnsPerHost:     config.MaxConnections,
		IdleConnTimeout:     config.IdleTimeout,
		DisableKeepAlives:   false,
		ForceAttemptHTTP2:   true,
		DialContext: (&net.Dialer{
			Timeout:   config.ConnectionTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	pool := &ConnectionPool{
		maxConnections:     config.MaxConnections,
		maxIdleConnections: config.MaxIdleConnections,
		idleTimeout:        config.IdleTimeout,
		connectionTimeout:  config.ConnectionTimeout,
		connections:        make(map[string]*PooledConnection),
		metrics:            metrics,
		transport:          transport,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   config.ConnectionTimeout,
		},
		done:            make(chan struct{}),
		cleanupInterval: config.CleanupInterval,
	}

	// Start cleanup goroutine
	go pool.cleanupLoop()

	return pool
}

// GetHTTPClient returns the pooled HTTP client
func (p *ConnectionPool) GetHTTPClient() *http.Client {
	return p.httpClient
}

// GetTransport returns the underlying HTTP transport
func (p *ConnectionPool) GetTransport() *http.Transport {
	return p.transport
}

// AcquireConnection gets a connection from the pool
func (p *ConnectionPool) AcquireConnection(ctx context.Context, addr string) (net.Conn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if we have an idle connection
	if pooledConn, exists := p.connections[addr]; exists && !pooledConn.inUse {
		pooledConn.inUse = true
		pooledConn.lastUsedAt = time.Now()
		p.activeCount++

		if p.metrics != nil {
			p.metrics.SetCustomMetric("pool_reuse", 1)
		}

		return pooledConn.conn, nil
	}

	// Check if we can create a new connection
	if p.activeCount >= p.maxConnections {
		if p.metrics != nil {
			p.metrics.RecordError("pool_exhausted")
		}
		return nil, fmt.Errorf("connection pool exhausted: %d/%d connections in use",
			p.activeCount, p.maxConnections)
	}

	// Create new connection
	dialer := &net.Dialer{
		Timeout:   p.connectionTimeout,
		KeepAlive: 30 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		if p.metrics != nil {
			p.metrics.RecordConnectionFailure()
		}
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	// Track the new connection
	pooledConn := &PooledConnection{
		conn:       conn,
		createdAt:  time.Now(),
		lastUsedAt: time.Now(),
		inUse:      true,
		id:         fmt.Sprintf("%s-%d", addr, time.Now().UnixNano()),
	}

	p.connections[pooledConn.id] = pooledConn
	p.activeCount++

	if p.metrics != nil {
		p.metrics.RecordConnection()
		p.metrics.SetCustomMetric("pool_size", int64(len(p.connections)))
		p.metrics.SetCustomMetric("pool_active", int64(p.activeCount))
	}

	return conn, nil
}

// ReleaseConnection returns a connection to the pool
func (p *ConnectionPool) ReleaseConnection(conn net.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Find the connection in our pool
	for _, pooledConn := range p.connections {
		if pooledConn.conn == conn {
			pooledConn.inUse = false
			pooledConn.lastUsedAt = time.Now()
			p.activeCount--

			if p.metrics != nil {
				p.metrics.SetCustomMetric("pool_active", int64(p.activeCount))
			}
			return
		}
	}
}

// cleanupLoop periodically cleans up idle connections
func (p *ConnectionPool) cleanupLoop() {
	ticker := time.NewTicker(p.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.cleanupIdleConnections()
		case <-p.done:
			return
		}
	}
}

// cleanupIdleConnections removes connections that have been idle too long
func (p *ConnectionPool) cleanupIdleConnections() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	toRemove := []string{}

	for id, pooledConn := range p.connections {
		if !pooledConn.inUse && now.Sub(pooledConn.lastUsedAt) > p.idleTimeout {
			toRemove = append(toRemove, id)
		}
	}

	// Remove idle connections
	for _, id := range toRemove {
		if pooledConn, exists := p.connections[id]; exists {
			pooledConn.conn.Close()
			delete(p.connections, id)

			if p.metrics != nil {
				duration := time.Since(pooledConn.createdAt)
				p.metrics.RecordDisconnection(duration)
				p.metrics.SetCustomMetric("pool_size", int64(len(p.connections)))
			}
		}
	}

	// Log cleanup stats
	if len(toRemove) > 0 && p.metrics != nil {
		p.metrics.SetCustomMetric("pool_cleaned", int64(len(toRemove)))
	}
}

// Stats returns current pool statistics
func (p *ConnectionPool) Stats() map[string]int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	idle := 0
	for _, conn := range p.connections {
		if !conn.inUse {
			idle++
		}
	}

	return map[string]int{
		"total":    len(p.connections),
		"active":   p.activeCount,
		"idle":     idle,
		"max":      p.maxConnections,
		"max_idle": p.maxIdleConnections,
	}
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() error {
	close(p.done)

	p.mu.Lock()
	defer p.mu.Unlock()

	// Close all connections
	for _, pooledConn := range p.connections {
		pooledConn.conn.Close()
	}

	// Clear the pool
	p.connections = make(map[string]*PooledConnection)
	p.activeCount = 0

	// Close the transport's idle connections
	p.transport.CloseIdleConnections()

	return nil
}
