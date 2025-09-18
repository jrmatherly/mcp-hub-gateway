package realtime

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ConnectionManager manages real-time connections for the portal
type ConnectionManager interface {
	// Connection management
	AddConnection(ctx context.Context, userID string, conn *Connection) error
	RemoveConnection(userID string, connectionID uuid.UUID) error
	GetConnections(userID string) []*Connection
	GetConnection(userID string, connectionID uuid.UUID) (*Connection, bool)

	// Broadcasting
	BroadcastToUser(userID string, event Event) error
	BroadcastToAll(event Event) error
	BroadcastToChannel(channel string, event Event) error

	// Channel management
	Subscribe(userID string, connectionID uuid.UUID, channel string) error
	Unsubscribe(userID string, connectionID uuid.UUID, channel string) error
	GetChannelSubscribers(channel string) []ChannelSubscriber

	// Event streaming
	StreamEvent(ctx context.Context, event Event) error
	StreamToConnection(ctx context.Context, connectionID uuid.UUID, event Event) error

	// Health monitoring
	PingConnections(ctx context.Context) error
	GetConnectionStats() ConnectionStats
	CleanupStaleConnections(ctx context.Context) error

	// HTTP handlers
	HandleWebSocket(c any) // Using any to avoid importing gin here
	HandleSSE(c any)       // Using any to avoid importing gin here

	// Lifecycle
	Stop() // Stops the connection manager and cleans up resources
}

// Connection represents a WebSocket or SSE connection
type Connection struct {
	ID           uuid.UUID          `json:"id"`
	UserID       string             `json:"user_id"`
	Type         ConnectionType     `json:"type"`
	WebSocket    any                `json:"-"` // Will be replaced with proper WebSocket type later
	SSEWriter    SSEWriter          `json:"-"`
	Channels     map[string]bool    `json:"channels"`
	Metadata     map[string]any     `json:"metadata"`
	ConnectedAt  time.Time          `json:"connected_at"`
	LastPingAt   time.Time          `json:"last_ping_at"`
	LastActivity time.Time          `json:"last_activity"`
	IsActive     bool               `json:"is_active"`
	Context      context.Context    `json:"-"`
	Cancel       context.CancelFunc `json:"-"`
}

// ConnectionType represents the type of real-time connection
type ConnectionType string

const (
	ConnectionTypeWebSocket ConnectionType = "websocket"
	ConnectionTypeSSE       ConnectionType = "sse"
)

// Event represents a real-time event to be sent to clients
type Event struct {
	ID        uuid.UUID      `json:"id"`
	Type      EventType      `json:"type"`
	Channel   string         `json:"channel,omitempty"`
	UserID    string         `json:"user_id,omitempty"`
	Data      any            `json:"data"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	TTL       time.Duration  `json:"ttl,omitempty"`
}

// EventType represents the type of event being sent
type EventType string

const (
	// Server events
	EventTypeServerEnabled      EventType = "server.enabled"
	EventTypeServerDisabled     EventType = "server.disabled"
	EventTypeServerStarted      EventType = "server.started"
	EventTypeServerStopped      EventType = "server.stopped"
	EventTypeServerRestarted    EventType = "server.restarted"
	EventTypeServerError        EventType = "server.error"
	EventTypeServerStatusUpdate EventType = "server.status_update"

	// Gateway events
	EventTypeGatewayStarted      EventType = "gateway.started"
	EventTypeGatewayStopped      EventType = "gateway.stopped"
	EventTypeGatewayRestarted    EventType = "gateway.restarted"
	EventTypeGatewayError        EventType = "gateway.error"
	EventTypeGatewayHealthUpdate EventType = "gateway.health_update"

	// Catalog events
	EventTypeCatalogSynced   EventType = "catalog.synced"
	EventTypeCatalogImported EventType = "catalog.imported"
	EventTypeCatalogUpdated  EventType = "catalog.updated"
	EventTypeCatalogError    EventType = "catalog.error"

	// Configuration events
	EventTypeConfigUpdated EventType = "config.updated"
	EventTypeConfigApplied EventType = "config.applied"
	EventTypeConfigError   EventType = "config.error"

	// System events
	EventTypeSystemAlert       EventType = "system.alert"
	EventTypeSystemMaintenance EventType = "system.maintenance"
	EventTypeLogEntry          EventType = "system.log"

	// User events
	EventTypeUserConnected    EventType = "user.connected"
	EventTypeUserDisconnected EventType = "user.disconnected"
	EventTypeNotification     EventType = "user.notification"

	// Connection events
	EventTypePing         EventType = "connection.ping"
	EventTypePong         EventType = "connection.pong"
	EventTypeConnected    EventType = "connection.connected"
	EventTypeDisconnected EventType = "connection.disconnected"
	EventTypeError        EventType = "connection.error"
)

// ChannelSubscriber represents a user subscribed to a channel
type ChannelSubscriber struct {
	UserID       string    `json:"user_id"`
	ConnectionID uuid.UUID `json:"connection_id"`
	SubscribedAt time.Time `json:"subscribed_at"`
}

// SSEWriter interface for Server-Sent Events writing
type SSEWriter interface {
	WriteEvent(event Event) error
	WriteData(data []byte) error
	WriteComment(comment string) error
	Flush() error
	Close() error
}

// ConnectionStats represents connection statistics
type ConnectionStats struct {
	TotalConnections     int            `json:"total_connections"`
	ActiveConnections    int            `json:"active_connections"`
	WebSocketConnections int            `json:"websocket_connections"`
	SSEConnections       int            `json:"sse_connections"`
	ConnectionsByUser    map[string]int `json:"connections_by_user"`
	ChannelSubscriptions map[string]int `json:"channel_subscriptions"`
	AverageConnectionAge time.Duration  `json:"average_connection_age"`
	EventsProcessed      int64          `json:"events_processed"`
	ErrorsCount          int64          `json:"errors_count"`
	LastUpdated          time.Time      `json:"last_updated"`
}

// Channel constants for predefined channels
const (
	ChannelServers    = "servers"
	ChannelGateway    = "gateway"
	ChannelCatalogs   = "catalogs"
	ChannelConfig     = "config"
	ChannelSystem     = "system"
	ChannelLogs       = "logs"
	ChannelUserPrefix = "user:"
)

// GetUserChannel returns the user-specific channel name
func GetUserChannel(userID string) string {
	return ChannelUserPrefix + userID
}

// WebSocketMessage represents a message received from WebSocket clients
type WebSocketMessage struct {
	Type      string         `json:"type"`
	Channel   string         `json:"channel,omitempty"`
	Data      any            `json:"data,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// WebSocketResponse represents a response sent to WebSocket clients
type WebSocketResponse struct {
	Type      string         `json:"type"`
	Success   bool           `json:"success"`
	Data      any            `json:"data,omitempty"`
	Error     string         `json:"error,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// MessageType represents WebSocket message types
type MessageType string

const (
	MessageTypeSubscribe   MessageType = "subscribe"
	MessageTypeUnsubscribe MessageType = "unsubscribe"
	MessageTypePing        MessageType = "ping"
	MessageTypePong        MessageType = "pong"
	MessageTypeCommand     MessageType = "command"
	MessageTypeResponse    MessageType = "response"
	MessageTypeEvent       MessageType = "event"
	MessageTypeError       MessageType = "error"
)

// EventFilter represents filtering options for events
type EventFilter struct {
	Types     []EventType `json:"types,omitempty"`
	Channels  []string    `json:"channels,omitempty"`
	UserID    string      `json:"user_id,omitempty"`
	StartTime *time.Time  `json:"start_time,omitempty"`
	EndTime   *time.Time  `json:"end_time,omitempty"`
	Limit     int         `json:"limit,omitempty"`
}

// ConnectionConfig represents configuration for real-time connections
type ConnectionConfig struct {
	MaxConnections        int           `json:"max_connections"`
	MaxConnectionsPerUser int           `json:"max_connections_per_user"`
	PingInterval          time.Duration `json:"ping_interval"`
	PongTimeout           time.Duration `json:"pong_timeout"`
	WriteTimeout          time.Duration `json:"write_timeout"`
	ReadTimeout           time.Duration `json:"read_timeout"`
	MaxMessageSize        int64         `json:"max_message_size"`
	AllowedOrigins        []string      `json:"allowed_origins"`
	BufferSize            int           `json:"buffer_size"`
	EnableCompression     bool          `json:"enable_compression"`
	CleanupInterval       time.Duration `json:"cleanup_interval"`
}

// DefaultConnectionConfig returns default connection configuration
func DefaultConnectionConfig() ConnectionConfig {
	return ConnectionConfig{
		MaxConnections:        1000,
		MaxConnectionsPerUser: 10,
		PingInterval:          30 * time.Second,
		PongTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		ReadTimeout:           60 * time.Second,
		MaxMessageSize:        1024 * 1024, // 1MB
		AllowedOrigins:        []string{"*"},
		BufferSize:            256,
		EnableCompression:     true,
		CleanupInterval:       5 * time.Minute,
	}
}
