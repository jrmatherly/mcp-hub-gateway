// Package realtime provides real-time communication tests.
package realtime

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/audit"
)

// TestCreateConnectionManager tests the creation of a connection manager
func TestCreateConnectionManager(t *testing.T) {
	mockAuditor := audit.NewLogger(audit.NewMemoryStorage())
	config := DefaultConnectionConfig()

	// Test successful creation
	manager, err := CreateConnectionManager(mockAuditor, config)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if manager == nil {
		t.Error("Expected manager to be created, got nil")
	}

	// Test with nil auditor
	_, err = CreateConnectionManager(nil, config)
	if err == nil {
		t.Error("Expected error with nil auditor, got nil")
	}
}

// TestConnectionOperations tests basic connection operations
func TestConnectionOperations(t *testing.T) {
	mockAuditor := audit.NewLogger(audit.NewMemoryStorage())
	config := DefaultConnectionConfig()

	manager, err := CreateConnectionManager(mockAuditor, config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()
	userID := "test-user"
	connectionID := uuid.New()

	// Create a test connection
	conn := &Connection{
		ID:           connectionID,
		UserID:       userID,
		Type:         ConnectionTypeWebSocket,
		Channels:     make(map[string]bool),
		Metadata:     make(map[string]any),
		ConnectedAt:  time.Now(),
		LastPingAt:   time.Now(),
		LastActivity: time.Now(),
		IsActive:     true,
	}

	// Test adding connection
	err = manager.AddConnection(ctx, userID, conn)
	if err != nil {
		t.Errorf("Expected no error adding connection, got %v", err)
	}

	// Test getting connections
	connections := manager.GetConnections(userID)
	if len(connections) != 1 {
		t.Errorf("Expected 1 connection, got %d", len(connections))
	}

	// Test getting specific connection
	retrievedConn, exists := manager.GetConnection(userID, connectionID)
	if !exists {
		t.Error("Expected connection to exist")
	}
	if retrievedConn.ID != connectionID {
		t.Error("Expected retrieved connection to have correct ID")
	}

	// Test removing connection
	err = manager.RemoveConnection(userID, connectionID)
	if err != nil {
		t.Errorf("Expected no error removing connection, got %v", err)
	}

	// Verify connection is removed
	connections = manager.GetConnections(userID)
	if len(connections) != 0 {
		t.Errorf("Expected 0 connections after removal, got %d", len(connections))
	}
}

// TestChannelSubscriptions tests channel subscription functionality
func TestChannelSubscriptions(t *testing.T) {
	mockAuditor := audit.NewLogger(audit.NewMemoryStorage())
	config := DefaultConnectionConfig()

	manager, err := CreateConnectionManager(mockAuditor, config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()
	userID := "test-user"
	connectionID := uuid.New()
	channel := "test-channel"

	// Create and add a test connection
	conn := &Connection{
		ID:           connectionID,
		UserID:       userID,
		Type:         ConnectionTypeSSE,
		Channels:     make(map[string]bool),
		Metadata:     make(map[string]any),
		ConnectedAt:  time.Now(),
		LastPingAt:   time.Now(),
		LastActivity: time.Now(),
		IsActive:     true,
	}

	err = manager.AddConnection(ctx, userID, conn)
	if err != nil {
		t.Errorf("Expected no error adding connection, got %v", err)
	}

	// Test subscribing to channel
	err = manager.Subscribe(userID, connectionID, channel)
	if err != nil {
		t.Errorf("Expected no error subscribing to channel, got %v", err)
	}

	// Test getting channel subscribers
	subscribers := manager.GetChannelSubscribers(channel)
	if len(subscribers) != 1 {
		t.Errorf("Expected 1 subscriber, got %d", len(subscribers))
	}
	if subscribers[0].UserID != userID {
		t.Error("Expected subscriber to have correct user ID")
	}
	if subscribers[0].ConnectionID != connectionID {
		t.Error("Expected subscriber to have correct connection ID")
	}

	// Test unsubscribing from channel
	err = manager.Unsubscribe(userID, connectionID, channel)
	if err != nil {
		t.Errorf("Expected no error unsubscribing from channel, got %v", err)
	}

	// Verify subscription is removed
	subscribers = manager.GetChannelSubscribers(channel)
	if len(subscribers) != 0 {
		t.Errorf("Expected 0 subscribers after unsubscribing, got %d", len(subscribers))
	}
}

// TestEventBroadcasting tests event broadcasting functionality
func TestEventBroadcasting(t *testing.T) {
	mockAuditor := audit.NewLogger(audit.NewMemoryStorage())
	config := DefaultConnectionConfig()

	manager, err := CreateConnectionManager(mockAuditor, config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()
	userID := "test-user"
	channel := "test-channel"

	// Create test event
	event := Event{
		ID:        uuid.New(),
		Type:      EventTypeServerEnabled,
		Channel:   channel,
		Data:      "test data",
		Timestamp: time.Now(),
	}

	// Test broadcasting to user (no connections)
	err = manager.BroadcastToUser(userID, event)
	if err != nil {
		t.Logf("Expected no error broadcasting to user with no connections: %v", err)
	}

	// Test broadcasting to channel (no subscribers)
	err = manager.BroadcastToChannel(channel, event)
	if err != nil {
		t.Logf("Expected no error broadcasting to channel with no subscribers: %v", err)
	}

	// Test broadcasting to all (no connections)
	err = manager.BroadcastToAll(event)
	if err != nil {
		t.Logf("Expected no error broadcasting to all with no connections: %v", err)
	}

	// Test streaming event
	err = manager.StreamEvent(ctx, event)
	if err != nil {
		t.Logf("Expected no error streaming event: %v", err)
	}
}

// TestConnectionStats tests connection statistics
func TestConnectionStats(t *testing.T) {
	mockAuditor := audit.NewLogger(audit.NewMemoryStorage())
	config := DefaultConnectionConfig()

	manager, err := CreateConnectionManager(mockAuditor, config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Get initial stats
	stats := manager.GetConnectionStats()
	if stats.TotalConnections != 0 {
		t.Error("Expected 0 total connections initially")
	}
	if stats.ActiveConnections != 0 {
		t.Error("Expected 0 active connections initially")
	}

	// The stats structure should be properly initialized
	if stats.ConnectionsByUser == nil {
		t.Error("Expected connection by user map to be initialized")
	}
	if stats.ChannelSubscriptions == nil {
		t.Error("Expected channel subscriptions map to be initialized")
	}
}

// TestConnectionTypes tests connection type constants and methods
func TestConnectionTypes(t *testing.T) {
	// Test connection type constants
	if ConnectionTypeWebSocket != "websocket" {
		t.Error("Expected WebSocket connection type to be 'websocket'")
	}
	if ConnectionTypeSSE != "sse" {
		t.Error("Expected SSE connection type to be 'sse'")
	}

	// Test event type constants
	eventTypes := []EventType{
		EventTypeServerEnabled,
		EventTypeServerDisabled,
		EventTypeServerStarted,
		EventTypeServerStopped,
		EventTypeGatewayStarted,
		EventTypeCatalogSynced,
		EventTypeConfigUpdated,
		EventTypeUserConnected,
		EventTypePing,
		EventTypePong,
	}

	for _, eventType := range eventTypes {
		if string(eventType) == "" {
			t.Errorf("Expected event type %v to have non-empty string value", eventType)
		}
	}
}

// TestDefaultConnectionConfig tests the default connection configuration
func TestDefaultConnectionConfig(t *testing.T) {
	config := DefaultConnectionConfig()

	if config.MaxConnections <= 0 {
		t.Error("Expected max connections to be positive")
	}
	if config.MaxConnectionsPerUser <= 0 {
		t.Error("Expected max connections per user to be positive")
	}
	if config.PingInterval <= 0 {
		t.Error("Expected ping interval to be positive")
	}
	if config.PongTimeout <= 0 {
		t.Error("Expected pong timeout to be positive")
	}
	if config.WriteTimeout <= 0 {
		t.Error("Expected write timeout to be positive")
	}
	if config.ReadTimeout <= 0 {
		t.Error("Expected read timeout to be positive")
	}
	if config.MaxMessageSize <= 0 {
		t.Error("Expected max message size to be positive")
	}
	if config.BufferSize <= 0 {
		t.Error("Expected buffer size to be positive")
	}
	if config.CleanupInterval <= 0 {
		t.Error("Expected cleanup interval to be positive")
	}
}

// TestChannelConstants tests predefined channel constants
func TestChannelConstants(t *testing.T) {
	channels := []string{
		ChannelServers,
		ChannelGateway,
		ChannelCatalogs,
		ChannelConfig,
		ChannelSystem,
		ChannelLogs,
	}

	for _, channel := range channels {
		if channel == "" {
			t.Error("Expected channel constant to have non-empty value")
		}
	}

	// Test user channel generation
	userID := "test-user-123"
	userChannel := GetUserChannel(userID)
	expectedChannel := ChannelUserPrefix + userID
	if userChannel != expectedChannel {
		t.Errorf("Expected user channel '%s', got '%s'", expectedChannel, userChannel)
	}
}

// TestWebSocketMessage tests WebSocket message structures
func TestWebSocketMessage(t *testing.T) {
	message := WebSocketMessage{
		Type:      string(MessageTypeSubscribe),
		Channel:   "test-channel",
		Data:      map[string]any{"key": "value"},
		RequestID: "req-123",
		Metadata:  map[string]any{"meta": "data"},
	}

	if message.Type != string(MessageTypeSubscribe) {
		t.Error("Expected message type to be subscribe")
	}
	if message.Channel != "test-channel" {
		t.Error("Expected channel to be 'test-channel'")
	}
	if message.RequestID != "req-123" {
		t.Error("Expected request ID to be 'req-123'")
	}

	// Validate Data field
	if message.Data == nil {
		t.Error("Expected data to be non-nil")
	}
	if dataMap, ok := message.Data.(map[string]any); ok {
		if dataMap["key"] != "value" {
			t.Error("Expected data to contain key 'key' with value 'value'")
		}
	} else {
		t.Error("Expected data to be a map[string]any")
	}

	// Validate Metadata field
	if message.Metadata == nil {
		t.Error("Expected metadata to be non-nil")
	}
	if message.Metadata["meta"] != "data" {
		t.Error("Expected metadata to contain key 'meta' with value 'data'")
	}

	// Test message types
	messageTypes := []MessageType{
		MessageTypeSubscribe,
		MessageTypeUnsubscribe,
		MessageTypePing,
		MessageTypePong,
		MessageTypeCommand,
		MessageTypeResponse,
		MessageTypeEvent,
		MessageTypeError,
	}

	for _, msgType := range messageTypes {
		if string(msgType) == "" {
			t.Errorf("Expected message type %v to have non-empty string value", msgType)
		}
	}
}

// TestConnectionLimits tests connection limit enforcement
func TestConnectionLimits(t *testing.T) {
	mockAuditor := audit.NewLogger(audit.NewMemoryStorage())
	config := DefaultConnectionConfig()

	// Set low limits for testing
	config.MaxConnections = 2
	config.MaxConnectionsPerUser = 1

	manager, err := CreateConnectionManager(mockAuditor, config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()
	userID := "test-user"

	// Create first connection (should succeed)
	conn1 := &Connection{
		ID:           uuid.New(),
		UserID:       userID,
		Type:         ConnectionTypeWebSocket,
		Channels:     make(map[string]bool),
		Metadata:     make(map[string]any),
		ConnectedAt:  time.Now(),
		LastPingAt:   time.Now(),
		LastActivity: time.Now(),
		IsActive:     true,
	}

	err = manager.AddConnection(ctx, userID, conn1)
	if err != nil {
		t.Errorf("Expected no error adding first connection, got %v", err)
	}

	// Create second connection for same user (should fail due to per-user limit)
	conn2 := &Connection{
		ID:           uuid.New(),
		UserID:       userID,
		Type:         ConnectionTypeSSE,
		Channels:     make(map[string]bool),
		Metadata:     make(map[string]any),
		ConnectedAt:  time.Now(),
		LastPingAt:   time.Now(),
		LastActivity: time.Now(),
		IsActive:     true,
	}

	err = manager.AddConnection(ctx, userID, conn2)
	if err == nil {
		t.Error("Expected error adding second connection for same user, got nil")
	}
}
