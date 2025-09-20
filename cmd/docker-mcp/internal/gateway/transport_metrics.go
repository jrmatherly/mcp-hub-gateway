package gateway

import (
	"sync"
	"sync/atomic"
	"time"
)

// TransportMetrics tracks metrics for a transport
type TransportMetrics struct {
	// Connection metrics
	ConnectionsTotal   atomic.Int64
	ConnectionsActive  atomic.Int64
	ConnectionsFailed  atomic.Int64
	ConnectionDuration atomic.Int64 // Total duration in nanoseconds

	// Message metrics
	MessagesReceived atomic.Int64
	MessagesSent     atomic.Int64
	BytesReceived    atomic.Int64
	BytesSent        atomic.Int64

	// Error metrics
	ErrorsTotal     atomic.Int64
	ErrorsProtocol  atomic.Int64
	ErrorsTransport atomic.Int64

	// Performance metrics
	LastMessageTime     atomic.Int64 // Unix nano timestamp
	AverageResponseTime atomic.Int64 // Nanoseconds

	// Transport-specific metrics
	Custom map[string]*atomic.Int64
	mu     sync.RWMutex
}

// NewTransportMetrics creates a new metrics instance
func NewTransportMetrics() *TransportMetrics {
	return &TransportMetrics{
		Custom: make(map[string]*atomic.Int64),
	}
}

// RecordConnection records a new connection
func (m *TransportMetrics) RecordConnection() {
	m.ConnectionsTotal.Add(1)
	m.ConnectionsActive.Add(1)
}

// RecordDisconnection records a connection close
func (m *TransportMetrics) RecordDisconnection(duration time.Duration) {
	m.ConnectionsActive.Add(-1)
	m.ConnectionDuration.Add(int64(duration))
}

// RecordConnectionFailure records a failed connection attempt
func (m *TransportMetrics) RecordConnectionFailure() {
	m.ConnectionsFailed.Add(1)
}

// RecordMessage records message metrics
func (m *TransportMetrics) RecordMessage(sent bool, bytes int) {
	if sent {
		m.MessagesSent.Add(1)
		m.BytesSent.Add(int64(bytes))
	} else {
		m.MessagesReceived.Add(1)
		m.BytesReceived.Add(int64(bytes))
	}
	m.LastMessageTime.Store(time.Now().UnixNano())
}

// RecordError records an error
func (m *TransportMetrics) RecordError(errType string) {
	m.ErrorsTotal.Add(1)
	switch errType {
	case "protocol":
		m.ErrorsProtocol.Add(1)
	case "transport":
		m.ErrorsTransport.Add(1)
	}
}

// RecordResponseTime records response time for a request
func (m *TransportMetrics) RecordResponseTime(duration time.Duration) {
	// Simple moving average (could be improved with weighted average)
	current := m.AverageResponseTime.Load()
	count := m.MessagesReceived.Load()
	if count > 0 {
		newAvg := (current*(count-1) + int64(duration)) / count
		m.AverageResponseTime.Store(newAvg)
	}
}

// SetCustomMetric sets a custom transport-specific metric
func (m *TransportMetrics) SetCustomMetric(name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if metric, exists := m.Custom[name]; exists {
		metric.Store(value)
	} else {
		m.Custom[name] = &atomic.Int64{}
		m.Custom[name].Store(value)
	}
}

// GetCustomMetric gets a custom transport-specific metric
func (m *TransportMetrics) GetCustomMetric(name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if metric, exists := m.Custom[name]; exists {
		return metric.Load()
	}
	return 0
}

// GetSnapshot returns a snapshot of all metrics
func (m *TransportMetrics) GetSnapshot() map[string]int64 {
	snapshot := map[string]int64{
		"connections_total":   m.ConnectionsTotal.Load(),
		"connections_active":  m.ConnectionsActive.Load(),
		"connections_failed":  m.ConnectionsFailed.Load(),
		"connection_duration": m.ConnectionDuration.Load(),
		"messages_received":   m.MessagesReceived.Load(),
		"messages_sent":       m.MessagesSent.Load(),
		"bytes_received":      m.BytesReceived.Load(),
		"bytes_sent":          m.BytesSent.Load(),
		"errors_total":        m.ErrorsTotal.Load(),
		"errors_protocol":     m.ErrorsProtocol.Load(),
		"errors_transport":    m.ErrorsTransport.Load(),
		"last_message_time":   m.LastMessageTime.Load(),
		"avg_response_time":   m.AverageResponseTime.Load(),
	}

	// Add custom metrics
	m.mu.RLock()
	for name, metric := range m.Custom {
		snapshot["custom_"+name] = metric.Load()
	}
	m.mu.RUnlock()

	return snapshot
}

// MetricsCollector interface for transports that support metrics
type MetricsCollector interface {
	GetMetrics() *TransportMetrics
	EnableMetrics(enabled bool)
	IsMetricsEnabled() bool
}
