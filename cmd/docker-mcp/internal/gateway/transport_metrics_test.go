package gateway

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTransportMetrics verifies metrics collection
func TestTransportMetrics(t *testing.T) {
	metrics := NewTransportMetrics()

	// Test connection metrics
	metrics.RecordConnection()
	assert.Equal(t, int64(1), metrics.ConnectionsTotal.Load())
	assert.Equal(t, int64(1), metrics.ConnectionsActive.Load())

	metrics.RecordDisconnection(5 * time.Second)
	assert.Equal(t, int64(0), metrics.ConnectionsActive.Load())
	assert.Greater(t, metrics.ConnectionDuration.Load(), int64(0))

	// Test message metrics
	metrics.RecordMessage(true, 100)
	assert.Equal(t, int64(1), metrics.MessagesSent.Load())
	assert.Equal(t, int64(100), metrics.BytesSent.Load())

	metrics.RecordMessage(false, 200)
	assert.Equal(t, int64(1), metrics.MessagesReceived.Load())
	assert.Equal(t, int64(200), metrics.BytesReceived.Load())

	// Test error metrics
	metrics.RecordError("protocol")
	assert.Equal(t, int64(1), metrics.ErrorsTotal.Load())
	assert.Equal(t, int64(1), metrics.ErrorsProtocol.Load())

	// Test custom metrics
	metrics.SetCustomMetric("test_metric", 42)
	assert.Equal(t, int64(42), metrics.GetCustomMetric("test_metric"))

	// Test snapshot
	snapshot := metrics.GetSnapshot()
	assert.Equal(t, int64(1), snapshot["connections_total"])
	assert.Equal(t, int64(1), snapshot["messages_sent"])
	assert.Equal(t, int64(42), snapshot["custom_test_metric"])
}

// TestStdioTransportWithMetrics tests stdio transport with metrics enabled
func TestStdioTransportWithMetrics(t *testing.T) {
	transport := NewStdioTransportWrapper()

	// Initially metrics should be nil
	assert.Nil(t, transport.GetMetrics())

	// Enable metrics
	transport.EnableMetrics(true)
	assert.NotNil(t, transport.GetMetrics())

	// Should record initial connection
	metrics := transport.GetMetrics()
	assert.Equal(t, int64(1), metrics.ConnectionsTotal.Load())

	// Disable metrics
	transport.EnableMetrics(false)
	assert.Nil(t, transport.GetMetrics())
}

// TestHTTPTransportWithMetrics tests HTTP transport with metrics
func TestHTTPTransportWithMetrics(t *testing.T) {
	// Create a test listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	transport := NewHTTPTransportWrapper(listener)

	// Enable metrics
	transport.EnableMetrics(true)
	metrics := transport.GetMetrics()
	assert.NotNil(t, metrics)

	// Test that metrics are being tracked
	// Note: In a real test, we'd make HTTP requests and verify metrics
	assert.Equal(t, int64(0), metrics.ConnectionsTotal.Load())
}

// TestConnectionPool verifies connection pooling functionality
func TestConnectionPool(t *testing.T) {
	metrics := NewTransportMetrics()
	config := DefaultConnectionPoolConfig()
	config.MaxConnections = 2
	config.IdleTimeout = 100 * time.Millisecond

	pool := NewConnectionPool(config, metrics)
	defer pool.Close()

	// Test HTTP client is available
	client := pool.GetHTTPClient()
	assert.NotNil(t, client)

	// Test stats
	stats := pool.Stats()
	assert.Equal(t, 0, stats["total"])
	assert.Equal(t, 0, stats["active"])
	assert.Equal(t, 2, stats["max"])

	// Test connection acquisition (requires a test server)
	ctx := context.Background()

	// Start a test server
	testListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer testListener.Close()

	go func() {
		for {
			conn, err := testListener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	addr := testListener.Addr().String()

	// Acquire connection
	conn1, err := pool.AcquireConnection(ctx, addr)
	assert.NoError(t, err)
	assert.NotNil(t, conn1)

	// Check metrics
	assert.Equal(t, int64(1), metrics.ConnectionsTotal.Load())

	// Release connection
	pool.ReleaseConnection(conn1)

	// Verify pool stats
	stats = pool.Stats()
	assert.GreaterOrEqual(t, stats["total"], 0)
	assert.Equal(t, 0, stats["active"])
}

// TestWebSocketTransport tests WebSocket transport functionality
func TestWebSocketTransport(t *testing.T) {
	// Create a test listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	config := DefaultWebSocketConfig()
	transport := NewWebSocketTransportWrapper(listener, config)
	defer transport.Close()

	// Test basic properties
	assert.Equal(t, "websocket", transport.Name())
	assert.True(t, transport.IsProtocolChannel())
	assert.NotNil(t, transport.Logger())
	assert.NotNil(t, transport.GetReader())
	assert.NotNil(t, transport.GetWriter())

	// Test metrics
	transport.EnableMetrics(true)
	metrics := transport.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(0), metrics.GetCustomMetric("ws_connections"))

	// Test broadcast with no connections
	err = transport.Broadcast([]byte("test message"))
	assert.NoError(t, err) // Should not error with no connections
}

// TestMetricsIntegration tests metrics collection across all transports
func TestMetricsIntegration(t *testing.T) {
	factory := &TransportFactory{}

	// Test stdio transport metrics
	stdioTransport, err := factory.CreateTransport("stdio", nil)
	require.NoError(t, err)
	stdioTransport.EnableMetrics(true)
	assert.NotNil(t, stdioTransport.GetMetrics())

	// Test HTTP transport metrics
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	httpTransport, err := factory.CreateTransport("http", listener)
	require.NoError(t, err)
	httpTransport.EnableMetrics(true)
	assert.NotNil(t, httpTransport.GetMetrics())

	// Test SSE transport metrics
	sseTransport, err := factory.CreateTransport("sse", listener)
	require.NoError(t, err)
	sseTransport.EnableMetrics(true)
	assert.NotNil(t, sseTransport.GetMetrics())

	// Test WebSocket transport metrics
	wsTransport, err := factory.CreateTransport("websocket", listener)
	require.NoError(t, err)
	wsTransport.EnableMetrics(true)
	assert.NotNil(t, wsTransport.GetMetrics())

	// Clean up
	stdioTransport.Close()
	httpTransport.Close()
	sseTransport.Close()
	wsTransport.Close()
}
