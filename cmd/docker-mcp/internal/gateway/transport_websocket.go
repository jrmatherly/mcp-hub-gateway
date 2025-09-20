package gateway

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketTransportWrapper wraps WebSocket for bidirectional MCP communication
type WebSocketTransportWrapper struct {
	server         *http.Server
	listener       net.Listener
	logger         *StderrLogger
	upgrader       *websocket.Upgrader
	connections    map[string]*WebSocketConnection
	mu             sync.RWMutex
	metrics        *TransportMetrics
	metricsEnabled bool

	// Configuration
	config *WebSocketConfig

	// Channels for message passing
	incomingChan chan []byte
	outgoingChan chan []byte
	done         chan struct{}
}

// WebSocketConnection represents a single WebSocket connection
type WebSocketConnection struct {
	conn          *websocket.Conn
	id            string
	createdAt     time.Time
	lastMessageAt time.Time
	reader        *WebSocketReader
	writer        *WebSocketWriter
}

// WebSocketConfig configures the WebSocket transport
type WebSocketConfig struct {
	ReadBufferSize    int
	WriteBufferSize   int
	HandshakeTimeout  time.Duration
	PingInterval      time.Duration
	PongTimeout       time.Duration
	MaxMessageSize    int64
	EnableCompression bool
}

// DefaultWebSocketConfig returns default WebSocket configuration
func DefaultWebSocketConfig() *WebSocketConfig {
	return &WebSocketConfig{
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		HandshakeTimeout:  10 * time.Second,
		PingInterval:      30 * time.Second,
		PongTimeout:       60 * time.Second,
		MaxMessageSize:    10 * 1024 * 1024, // 10MB
		EnableCompression: true,
	}
}

// NewWebSocketTransportWrapper creates a new WebSocket transport wrapper
func NewWebSocketTransportWrapper(
	listener net.Listener,
	config *WebSocketConfig,
) *WebSocketTransportWrapper {
	if config == nil {
		config = DefaultWebSocketConfig()
	}

	upgrader := &websocket.Upgrader{
		ReadBufferSize:    config.ReadBufferSize,
		WriteBufferSize:   config.WriteBufferSize,
		HandshakeTimeout:  config.HandshakeTimeout,
		EnableCompression: config.EnableCompression,
		CheckOrigin: func(r *http.Request) bool {
			// In production, implement proper origin checking
			return true
		},
	}

	return &WebSocketTransportWrapper{
		listener:     listener,
		logger:       NewStderrLogger("[MCP-WebSocket]", LogLevelInfo),
		upgrader:     upgrader,
		connections:  make(map[string]*WebSocketConnection),
		config:       config,
		incomingChan: make(chan []byte, 100),
		outgoingChan: make(chan []byte, 100),
		done:         make(chan struct{}),
	}
}

// Name returns the transport name
func (t *WebSocketTransportWrapper) Name() string {
	return "websocket"
}

// Logger returns the transport logger
func (t *WebSocketTransportWrapper) Logger() TransportLogger {
	return t.logger
}

// IsProtocolChannel returns true as WebSocket handles protocol messages
func (t *WebSocketTransportWrapper) IsProtocolChannel() bool {
	return true
}

// GetReader returns a reader for WebSocket messages
func (t *WebSocketTransportWrapper) GetReader() io.Reader {
	return &WebSocketReader{transport: t}
}

// GetWriter returns a writer for WebSocket messages
func (t *WebSocketTransportWrapper) GetWriter() io.Writer {
	return &WebSocketWriter{transport: t}
}

// Close closes the WebSocket server and all connections
func (t *WebSocketTransportWrapper) Close() error {
	close(t.done)

	t.mu.Lock()
	defer t.mu.Unlock()

	// Close all connections
	for id, conn := range t.connections {
		conn.conn.Close()
		delete(t.connections, id)

		if t.metrics != nil {
			duration := time.Since(conn.createdAt)
			t.metrics.RecordDisconnection(duration)
		}
	}

	if t.server != nil {
		return t.server.Shutdown(context.Background())
	}

	return nil
}

// GetMetrics returns the metrics for this transport
func (t *WebSocketTransportWrapper) GetMetrics() *TransportMetrics {
	if !t.metricsEnabled {
		return nil
	}
	return t.metrics
}

// EnableMetrics enables or disables metrics collection
func (t *WebSocketTransportWrapper) EnableMetrics(enabled bool) {
	t.metricsEnabled = enabled
	if enabled && t.metrics == nil {
		t.metrics = NewTransportMetrics()
		// Set WebSocket-specific metrics
		t.metrics.SetCustomMetric("ws_connections", 0)
		t.metrics.SetCustomMetric("ws_messages_buffered", 0)
	}
}

// HandleWebSocket handles incoming WebSocket connections
func (t *WebSocketTransportWrapper) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := t.upgrader.Upgrade(w, r, nil)
	if err != nil {
		t.logger.Logf("Failed to upgrade connection: %v", err)
		if t.metrics != nil {
			t.metrics.RecordConnectionFailure()
		}
		return
	}

	// Configure connection
	conn.SetReadLimit(t.config.MaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(t.config.PongTimeout))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(t.config.PongTimeout))
		return nil
	})

	// Create connection wrapper
	wsConn := &WebSocketConnection{
		conn:          conn,
		id:            fmt.Sprintf("%s-%d", r.RemoteAddr, time.Now().UnixNano()),
		createdAt:     time.Now(),
		lastMessageAt: time.Now(),
		reader:        &WebSocketReader{transport: t},
		writer:        &WebSocketWriter{transport: t},
	}

	// Register connection
	t.mu.Lock()
	t.connections[wsConn.id] = wsConn
	t.mu.Unlock()

	if t.metrics != nil {
		t.metrics.RecordConnection()
		t.metrics.SetCustomMetric("ws_connections", int64(len(t.connections)))
	}

	// Start ping/pong handler
	go t.pingHandler(wsConn)

	// Start message handler
	go t.messageHandler(wsConn)
}

// pingHandler sends periodic pings to keep connection alive
func (t *WebSocketTransportWrapper) pingHandler(wsConn *WebSocketConnection) {
	ticker := time.NewTicker(t.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			wsConn.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := wsConn.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				t.logger.Logf("Ping failed for %s: %v", wsConn.id, err)
				t.removeConnection(wsConn.id)
				return
			}
		case <-t.done:
			return
		}
	}
}

// messageHandler handles incoming messages from a WebSocket connection
func (t *WebSocketTransportWrapper) messageHandler(wsConn *WebSocketConnection) {
	defer t.removeConnection(wsConn.id)

	for {
		messageType, message, err := wsConn.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				t.logger.Logf("WebSocket error from %s: %v", wsConn.id, err)
			}
			break
		}

		if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
			wsConn.lastMessageAt = time.Now()

			// Send to incoming channel
			select {
			case t.incomingChan <- message:
				if t.metrics != nil {
					t.metrics.RecordMessage(false, len(message))
				}
			default:
				t.logger.Log("Incoming message buffer full, dropping message")
				if t.metrics != nil {
					t.metrics.RecordError("buffer_overflow")
				}
			}
		}
	}
}

// removeConnection removes a connection from the pool
func (t *WebSocketTransportWrapper) removeConnection(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if conn, exists := t.connections[id]; exists {
		conn.conn.Close()
		delete(t.connections, id)

		if t.metrics != nil {
			duration := time.Since(conn.createdAt)
			t.metrics.RecordDisconnection(duration)
			t.metrics.SetCustomMetric("ws_connections", int64(len(t.connections)))
		}
	}
}

// Broadcast sends a message to all connected clients
func (t *WebSocketTransportWrapper) Broadcast(message []byte) error {
	t.mu.RLock()
	connections := make([]*WebSocketConnection, 0, len(t.connections))
	for _, conn := range t.connections {
		connections = append(connections, conn)
	}
	t.mu.RUnlock()

	var lastErr error
	successCount := 0

	for _, conn := range connections {
		conn.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			t.logger.Logf("Failed to send to %s: %v", conn.id, err)
			lastErr = err
			t.removeConnection(conn.id)
		} else {
			successCount++
			if t.metrics != nil {
				t.metrics.RecordMessage(true, len(message))
			}
		}
	}

	if successCount == 0 && lastErr != nil {
		return fmt.Errorf("broadcast failed to all connections: %w", lastErr)
	}

	return nil
}

// WebSocketReader implements io.Reader for WebSocket transport
type WebSocketReader struct {
	transport *WebSocketTransportWrapper
	buffer    []byte
	pos       int
}

// Read implements io.Reader
func (r *WebSocketReader) Read(p []byte) (n int, err error) {
	// If we have buffered data, return it
	if r.pos < len(r.buffer) {
		n = copy(p, r.buffer[r.pos:])
		r.pos += n
		return n, nil
	}

	// Wait for new message
	select {
	case message := <-r.transport.incomingChan:
		r.buffer = message
		r.pos = 0
		n = copy(p, r.buffer)
		r.pos += n
		return n, nil
	case <-r.transport.done:
		return 0, io.EOF
	}
}

// WebSocketWriter implements io.Writer for WebSocket transport
type WebSocketWriter struct {
	transport *WebSocketTransportWrapper
}

// Write implements io.Writer
func (w *WebSocketWriter) Write(p []byte) (n int, err error) {
	// Broadcast to all connections
	if err := w.transport.Broadcast(p); err != nil {
		return 0, err
	}
	return len(p), nil
}
