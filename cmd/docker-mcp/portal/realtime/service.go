// Package realtime provides WebSocket and SSE real-time communication implementation.
package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/audit"
)

// connectionManagerService implements the ConnectionManager interface
type connectionManagerService struct {
	// Connection storage
	connections   map[string]map[uuid.UUID]*Connection // userID -> connectionID -> Connection
	connectionsMu sync.RWMutex

	// Channel subscriptions
	channelSubscribers map[string][]ChannelSubscriber // channel -> subscribers
	subscribersMu      sync.RWMutex

	// Configuration
	config ConnectionConfig

	// WebSocket upgrader
	upgrader websocket.Upgrader

	// Background workers
	stopChan chan struct{}
	wg       sync.WaitGroup

	// Statistics
	stats   ConnectionStats
	statsMu sync.RWMutex

	// Auditing
	auditor audit.Logger
}

// CreateConnectionManager creates a new connection manager service
func CreateConnectionManager(
	auditor audit.Logger,
	config ConnectionConfig,
) (ConnectionManager, error) {
	if auditor == nil {
		return nil, fmt.Errorf("auditor is required")
	}

	service := &connectionManagerService{
		connections:        make(map[string]map[uuid.UUID]*Connection),
		channelSubscribers: make(map[string][]ChannelSubscriber),
		config:             config,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from configured origins
				origin := r.Header.Get("Origin")
				if len(config.AllowedOrigins) == 0 {
					return true
				}
				for _, allowed := range config.AllowedOrigins {
					if allowed == "*" || allowed == origin {
						return true
					}
				}
				return false
			},
			EnableCompression: config.EnableCompression,
			ReadBufferSize:    config.BufferSize,
			WriteBufferSize:   config.BufferSize,
		},
		stopChan: make(chan struct{}),
		stats: ConnectionStats{
			ConnectionsByUser:    make(map[string]int),
			ChannelSubscriptions: make(map[string]int),
			LastUpdated:          time.Now(),
		},
		auditor: auditor,
	}

	// Start background workers
	service.startBackgroundWorkers()

	return service, nil
}

// AddConnection adds a new connection for a user
func (s *connectionManagerService) AddConnection(
	ctx context.Context,
	userID string,
	conn *Connection,
) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}
	if conn == nil {
		return fmt.Errorf("connection is required")
	}

	s.connectionsMu.Lock()
	defer s.connectionsMu.Unlock()

	// Initialize user connections map if not exists
	if s.connections[userID] == nil {
		s.connections[userID] = make(map[uuid.UUID]*Connection)
	}

	// Check connection limits
	if len(s.connections[userID]) >= s.config.MaxConnectionsPerUser {
		return fmt.Errorf("maximum connections per user exceeded")
	}

	totalConnections := s.getTotalConnections()
	if totalConnections >= s.config.MaxConnections {
		return fmt.Errorf("maximum total connections exceeded")
	}

	// Add connection
	conn.UserID = userID
	conn.ConnectedAt = time.Now()
	conn.LastPingAt = time.Now()
	conn.LastActivity = time.Now()
	conn.IsActive = true
	conn.Channels = make(map[string]bool)

	s.connections[userID][conn.ID] = conn

	// Update statistics
	s.updateConnectionStats(conn, true)

	// Audit log
	s.auditor.Log(
		ctx,
		audit.ActionCreate,
		"connection",
		conn.ID.String(),
		userID,
		map[string]any{
			"connection_type":   conn.Type,
			"total_connections": totalConnections + 1,
			"action":            "connection_added",
		},
	)

	// Start connection management for this connection
	go s.manageConnection(ctx, conn)

	return nil
}

// RemoveConnection removes a connection for a user
func (s *connectionManagerService) RemoveConnection(userID string, connectionID uuid.UUID) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}

	s.connectionsMu.Lock()
	defer s.connectionsMu.Unlock()

	userConnections, exists := s.connections[userID]
	if !exists {
		return fmt.Errorf("no connections found for user")
	}

	conn, exists := userConnections[connectionID]
	if !exists {
		return fmt.Errorf("connection not found")
	}

	// Mark as inactive and cancel context
	conn.IsActive = false
	if conn.Cancel != nil {
		conn.Cancel()
	}

	// Clean up subscriptions
	s.cleanupConnectionSubscriptions(userID, connectionID)

	// Update statistics
	s.updateConnectionStats(conn, false)

	// Remove connection
	delete(userConnections, connectionID)
	if len(userConnections) == 0 {
		delete(s.connections, userID)
	}

	// Audit log
	s.auditor.Log(
		context.Background(),
		audit.ActionDelete,
		"connection",
		connectionID.String(),
		userID,
		map[string]any{
			"connection_type": conn.Type,
			"action":          "connection_removed",
		},
	)

	return nil
}

// GetConnections gets all connections for a user
func (s *connectionManagerService) GetConnections(userID string) []*Connection {
	if userID == "" {
		return nil
	}

	s.connectionsMu.RLock()
	defer s.connectionsMu.RUnlock()

	userConnections, exists := s.connections[userID]
	if !exists {
		return nil
	}

	connections := make([]*Connection, 0, len(userConnections))
	for _, conn := range userConnections {
		if conn.IsActive {
			connections = append(connections, conn)
		}
	}

	return connections
}

// GetConnection gets a specific connection
func (s *connectionManagerService) GetConnection(
	userID string,
	connectionID uuid.UUID,
) (*Connection, bool) {
	if userID == "" {
		return nil, false
	}

	s.connectionsMu.RLock()
	defer s.connectionsMu.RUnlock()

	userConnections, exists := s.connections[userID]
	if !exists {
		return nil, false
	}

	conn, exists := userConnections[connectionID]
	if !exists || !conn.IsActive {
		return nil, false
	}

	return conn, true
}

// BroadcastToUser sends an event to all connections of a specific user
func (s *connectionManagerService) BroadcastToUser(userID string, event Event) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}

	connections := s.GetConnections(userID)
	if len(connections) == 0 {
		return nil // No connections to broadcast to
	}

	var lastError error
	successCount := 0

	for _, conn := range connections {
		if err := s.sendEventToConnection(conn, event); err != nil {
			lastError = err
		} else {
			successCount++
		}
	}

	// Update statistics
	s.statsMu.Lock()
	s.stats.EventsProcessed++
	if lastError != nil {
		s.stats.ErrorsCount++
	}
	s.statsMu.Unlock()

	if successCount == 0 && lastError != nil {
		return fmt.Errorf("failed to send to any connection: %w", lastError)
	}

	return nil
}

// BroadcastToAll sends an event to all active connections
func (s *connectionManagerService) BroadcastToAll(event Event) error {
	s.connectionsMu.RLock()
	defer s.connectionsMu.RUnlock()

	var lastError error
	successCount := 0
	totalConnections := 0

	for _, userConnections := range s.connections {
		for _, conn := range userConnections {
			if conn.IsActive {
				totalConnections++
				if err := s.sendEventToConnection(conn, event); err != nil {
					lastError = err
				} else {
					successCount++
				}
			}
		}
	}

	// Update statistics
	s.statsMu.Lock()
	s.stats.EventsProcessed += int64(totalConnections)
	if lastError != nil {
		s.stats.ErrorsCount++
	}
	s.statsMu.Unlock()

	if totalConnections > 0 && successCount == 0 && lastError != nil {
		return fmt.Errorf("failed to send to any connection: %w", lastError)
	}

	return nil
}

// BroadcastToChannel sends an event to all subscribers of a specific channel
func (s *connectionManagerService) BroadcastToChannel(channel string, event Event) error {
	if channel == "" {
		return fmt.Errorf("channel is required")
	}

	subscribers := s.GetChannelSubscribers(channel)
	if len(subscribers) == 0 {
		return nil // No subscribers
	}

	var lastError error
	successCount := 0

	for _, subscriber := range subscribers {
		conn, exists := s.GetConnection(subscriber.UserID, subscriber.ConnectionID)
		if !exists {
			continue
		}

		if err := s.sendEventToConnection(conn, event); err != nil {
			lastError = err
		} else {
			successCount++
		}
	}

	// Update statistics
	s.statsMu.Lock()
	s.stats.EventsProcessed++
	if lastError != nil {
		s.stats.ErrorsCount++
	}
	s.statsMu.Unlock()

	if successCount == 0 && lastError != nil {
		return fmt.Errorf("failed to send to any channel subscriber: %w", lastError)
	}

	return nil
}

// Subscribe subscribes a connection to a channel
func (s *connectionManagerService) Subscribe(
	userID string,
	connectionID uuid.UUID,
	channel string,
) error {
	if userID == "" || channel == "" {
		return fmt.Errorf("user ID and channel are required")
	}

	// Get connection
	conn, exists := s.GetConnection(userID, connectionID)
	if !exists {
		return fmt.Errorf("connection not found")
	}

	// Add to connection's channels
	s.connectionsMu.Lock()
	conn.Channels[channel] = true
	s.connectionsMu.Unlock()

	// Add to channel subscribers
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	subscriber := ChannelSubscriber{
		UserID:       userID,
		ConnectionID: connectionID,
		SubscribedAt: time.Now(),
	}

	s.channelSubscribers[channel] = append(s.channelSubscribers[channel], subscriber)

	// Update statistics
	s.statsMu.Lock()
	s.stats.ChannelSubscriptions[channel]++
	s.statsMu.Unlock()

	return nil
}

// Unsubscribe unsubscribes a connection from a channel
func (s *connectionManagerService) Unsubscribe(
	userID string,
	connectionID uuid.UUID,
	channel string,
) error {
	if userID == "" || channel == "" {
		return fmt.Errorf("user ID and channel are required")
	}

	// Remove from connection's channels
	conn, exists := s.GetConnection(userID, connectionID)
	if exists {
		s.connectionsMu.Lock()
		delete(conn.Channels, channel)
		s.connectionsMu.Unlock()
	}

	// Remove from channel subscribers
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	subscribers := s.channelSubscribers[channel]
	for i, subscriber := range subscribers {
		if subscriber.UserID == userID && subscriber.ConnectionID == connectionID {
			// Remove subscriber
			s.channelSubscribers[channel] = append(subscribers[:i], subscribers[i+1:]...)
			break
		}
	}

	// Update statistics
	s.statsMu.Lock()
	if s.stats.ChannelSubscriptions[channel] > 0 {
		s.stats.ChannelSubscriptions[channel]--
	}
	s.statsMu.Unlock()

	return nil
}

// GetChannelSubscribers gets all subscribers for a channel
func (s *connectionManagerService) GetChannelSubscribers(channel string) []ChannelSubscriber {
	if channel == "" {
		return nil
	}

	s.subscribersMu.RLock()
	defer s.subscribersMu.RUnlock()

	subscribers, exists := s.channelSubscribers[channel]
	if !exists {
		return nil
	}

	// Return a copy
	result := make([]ChannelSubscriber, len(subscribers))
	copy(result, subscribers)
	return result
}

// StreamEvent streams an event to appropriate connections
func (s *connectionManagerService) StreamEvent(ctx context.Context, event Event) error {
	// Determine where to send the event
	if event.UserID != "" {
		// Send to specific user
		return s.BroadcastToUser(event.UserID, event)
	} else if event.Channel != "" {
		// Send to channel subscribers
		return s.BroadcastToChannel(event.Channel, event)
	} else {
		// Send to all connections
		return s.BroadcastToAll(event)
	}
}

// StreamToConnection streams an event to a specific connection
func (s *connectionManagerService) StreamToConnection(
	ctx context.Context,
	connectionID uuid.UUID,
	event Event,
) error {
	s.connectionsMu.RLock()
	defer s.connectionsMu.RUnlock()

	// Find the connection
	for _, userConnections := range s.connections {
		for _, conn := range userConnections {
			if conn.ID == connectionID && conn.IsActive {
				return s.sendEventToConnection(conn, event)
			}
		}
	}

	return fmt.Errorf("connection not found")
}

// PingConnections sends ping to all active connections
func (s *connectionManagerService) PingConnections(ctx context.Context) error {
	s.connectionsMu.RLock()
	defer s.connectionsMu.RUnlock()

	pingEvent := Event{
		ID:        uuid.New(),
		Type:      EventTypePing,
		Data:      "ping",
		Timestamp: time.Now(),
	}

	var lastError error
	for _, userConnections := range s.connections {
		for _, conn := range userConnections {
			if conn.IsActive {
				if err := s.sendEventToConnection(conn, pingEvent); err != nil {
					lastError = err
				} else {
					conn.LastPingAt = time.Now()
				}
			}
		}
	}

	return lastError
}

// GetConnectionStats returns current connection statistics
func (s *connectionManagerService) GetConnectionStats() ConnectionStats {
	s.statsMu.RLock()
	defer s.statsMu.RUnlock()

	// Update current statistics
	s.updateCurrentStats()

	// Return a copy
	stats := s.stats
	return stats
}

// CleanupStaleConnections removes inactive and stale connections
func (s *connectionManagerService) CleanupStaleConnections(ctx context.Context) error {
	s.connectionsMu.Lock()
	defer s.connectionsMu.Unlock()

	staleThreshold := time.Now().Add(-s.config.PongTimeout * 3) // 3x pong timeout
	var removedCount int

	for userID, userConnections := range s.connections {
		for connectionID, conn := range userConnections {
			if !conn.IsActive || conn.LastActivity.Before(staleThreshold) {
				// Mark as inactive and clean up
				conn.IsActive = false
				if conn.Cancel != nil {
					conn.Cancel()
				}

				// Clean up subscriptions
				s.cleanupConnectionSubscriptions(userID, connectionID)

				// Update statistics
				s.updateConnectionStats(conn, false)

				// Remove connection
				delete(userConnections, connectionID)
				removedCount++
			}
		}

		// Clean up empty user connection maps
		if len(userConnections) == 0 {
			delete(s.connections, userID)
		}
	}

	if removedCount > 0 {
		s.auditor.Log(ctx, audit.ActionDelete, "stale_connections", "", "", map[string]any{
			"removed_count": removedCount,
			"action":        "cleanup_stale_connections",
		})
	}

	return nil
}

// HTTP handlers for WebSocket and SSE

// HandleWebSocket handles WebSocket connection requests
func (s *connectionManagerService) HandleWebSocket(c any) {
	ginCtx := c.(*gin.Context)
	userID := ginCtx.GetString("user_id") // Assume user ID is set by auth middleware
	if userID == "" {
		ginCtx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Upgrade to WebSocket
	ws, err := s.upgrader.Upgrade(ginCtx.Writer, ginCtx.Request, nil)
	if err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": "Failed to upgrade to WebSocket"})
		return
	}

	// Create connection
	ctx, cancel := context.WithCancel(ginCtx.Request.Context())
	conn := &Connection{
		ID:           uuid.New(),
		UserID:       userID,
		Type:         ConnectionTypeWebSocket,
		WebSocket:    ws,
		Channels:     make(map[string]bool),
		Metadata:     make(map[string]any),
		ConnectedAt:  time.Now(),
		LastPingAt:   time.Now(),
		LastActivity: time.Now(),
		IsActive:     true,
		Context:      ctx,
		Cancel:       cancel,
	}

	// Add connection
	if err := s.AddConnection(ctx, userID, conn); err != nil {
		cancel()
		ws.Close()
		ginCtx.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		return
	}

	// Send connection confirmed event
	welcomeEvent := Event{
		ID:   uuid.New(),
		Type: EventTypeConnected,
		Data: WebSocketResponse{
			Type:      "connected",
			Success:   true,
			Data:      conn.ID,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}
	s.sendEventToConnection(conn, welcomeEvent)
}

// HandleSSE handles Server-Sent Events connection requests
func (s *connectionManagerService) HandleSSE(c any) {
	ginCtx := c.(*gin.Context)
	userID := ginCtx.GetString("user_id") // Assume user ID is set by auth middleware
	if userID == "" {
		ginCtx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Set SSE headers
	ginCtx.Header("Content-Type", "text/event-stream")
	ginCtx.Header("Cache-Control", "no-cache")
	ginCtx.Header("Connection", "keep-alive")
	ginCtx.Header("Access-Control-Allow-Origin", "*")

	// Create SSE writer
	sseWriter := &sseWriter{
		writer:  ginCtx.Writer,
		flusher: ginCtx.Writer.(http.Flusher),
	}

	// Create connection
	ctx, cancel := context.WithCancel(ginCtx.Request.Context())
	conn := &Connection{
		ID:           uuid.New(),
		UserID:       userID,
		Type:         ConnectionTypeSSE,
		SSEWriter:    sseWriter,
		Channels:     make(map[string]bool),
		Metadata:     make(map[string]any),
		ConnectedAt:  time.Now(),
		LastPingAt:   time.Now(),
		LastActivity: time.Now(),
		IsActive:     true,
		Context:      ctx,
		Cancel:       cancel,
	}

	// Add connection
	if err := s.AddConnection(ctx, userID, conn); err != nil {
		cancel()
		ginCtx.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		return
	}

	// Send connection confirmed event
	welcomeEvent := Event{
		ID:   uuid.New(),
		Type: EventTypeConnected,
		Data: map[string]any{
			"connection_id": conn.ID,
			"type":          "sse",
			"connected_at":  conn.ConnectedAt,
		},
		Timestamp: time.Now(),
	}
	s.sendEventToConnection(conn, welcomeEvent)

	// Keep connection alive until context is cancelled
	<-ctx.Done()
	s.RemoveConnection(userID, conn.ID)
}

// Core helper methods

func (s *connectionManagerService) sendEventToConnection(conn *Connection, event Event) error {
	if !conn.IsActive {
		return fmt.Errorf("connection is not active")
	}

	// Update last activity
	conn.LastActivity = time.Now()

	switch conn.Type {
	case ConnectionTypeWebSocket:
		return s.sendWebSocketEvent(conn, event)
	case ConnectionTypeSSE:
		return s.sendSSEEvent(conn, event)
	default:
		return fmt.Errorf("unsupported connection type: %s", conn.Type)
	}
}

func (s *connectionManagerService) sendWebSocketEvent(conn *Connection, event Event) error {
	if conn.WebSocket == nil {
		return fmt.Errorf("WebSocket connection is nil")
	}

	ws := conn.WebSocket.(*websocket.Conn)

	// Set write deadline
	ws.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))

	// Send event as JSON
	return ws.WriteJSON(event)
}

func (s *connectionManagerService) sendSSEEvent(conn *Connection, event Event) error {
	if conn.SSEWriter == nil {
		return fmt.Errorf("SSE writer is nil")
	}

	// Convert event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Send as SSE event
	return conn.SSEWriter.WriteEvent(Event{
		ID:   event.ID,
		Type: event.Type,
		Data: string(data),
	})
}

func (s *connectionManagerService) manageConnection(ctx context.Context, conn *Connection) {
	defer func() {
		// Ensure connection is cleaned up
		s.RemoveConnection(conn.UserID, conn.ID)
	}()

	if conn.Type == ConnectionTypeWebSocket {
		s.manageWebSocketConnection(ctx, conn)
	} else if conn.Type == ConnectionTypeSSE {
		s.manageSSEConnection(ctx, conn)
	}
}

func (s *connectionManagerService) manageWebSocketConnection(
	ctx context.Context,
	conn *Connection,
) {
	if conn.WebSocket == nil {
		return
	}

	ws := conn.WebSocket.(*websocket.Conn)
	defer ws.Close()

	// Set up ping/pong handlers
	ws.SetPingHandler(func(data string) error {
		conn.LastPingAt = time.Now()
		conn.LastActivity = time.Now()
		return ws.WriteControl(
			websocket.PongMessage,
			[]byte(data),
			time.Now().Add(s.config.WriteTimeout),
		)
	})

	ws.SetPongHandler(func(data string) error {
		conn.LastActivity = time.Now()
		return nil
	})

	// Read messages from client
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Set read deadline
			ws.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))

			var message WebSocketMessage
			if err := ws.ReadJSON(&message); err != nil {
				if websocket.IsUnexpectedCloseError(
					err,
					websocket.CloseGoingAway,
					websocket.CloseAbnormalClosure,
				) {
					// Log unexpected close
					s.auditor.Log(
						ctx,
						audit.ActionDelete,
						"websocket",
						conn.ID.String(),
						conn.UserID,
						map[string]any{
							"action": "websocket_unexpected_close",
							"error":  err.Error(),
						},
					)
				}
				return
			}

			conn.LastActivity = time.Now()

			// Handle message
			s.handleWebSocketMessage(ctx, conn, message)
		}
	}
}

func (s *connectionManagerService) manageSSEConnection(ctx context.Context, _ *Connection) {
	// SSE connections are read-only, just wait for context cancellation
	<-ctx.Done()
}

func (s *connectionManagerService) handleWebSocketMessage(
	_ context.Context,
	conn *Connection,
	message WebSocketMessage,
) {
	response := WebSocketResponse{
		Type:      string(MessageTypeResponse),
		RequestID: message.RequestID,
		Timestamp: time.Now(),
	}

	switch MessageType(message.Type) {
	case MessageTypeSubscribe:
		if message.Channel != "" {
			if err := s.Subscribe(conn.UserID, conn.ID, message.Channel); err != nil {
				response.Success = false
				response.Error = err.Error()
			} else {
				response.Success = true
				response.Data = map[string]any{
					"channel": message.Channel,
					"action":  "subscribed",
				}
			}
		} else {
			response.Success = false
			response.Error = "Channel is required for subscription"
		}

	case MessageTypeUnsubscribe:
		if message.Channel != "" {
			if err := s.Unsubscribe(conn.UserID, conn.ID, message.Channel); err != nil {
				response.Success = false
				response.Error = err.Error()
			} else {
				response.Success = true
				response.Data = map[string]any{
					"channel": message.Channel,
					"action":  "unsubscribed",
				}
			}
		} else {
			response.Success = false
			response.Error = "Channel is required for unsubscription"
		}

	case MessageTypePing:
		response.Type = string(MessageTypePong)
		response.Success = true
		response.Data = "pong"

	default:
		response.Success = false
		response.Error = fmt.Sprintf("Unknown message type: %s", message.Type)
	}

	// Send response
	event := Event{
		ID:        uuid.New(),
		Type:      EventTypeConnected, // Use generic event type
		Data:      response,
		Timestamp: time.Now(),
	}
	s.sendEventToConnection(conn, event)
}

// Utility methods

func (s *connectionManagerService) getTotalConnections() int {
	total := 0
	for _, userConnections := range s.connections {
		for _, conn := range userConnections {
			if conn.IsActive {
				total++
			}
		}
	}
	return total
}

func (s *connectionManagerService) updateConnectionStats(conn *Connection, added bool) {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()

	if added {
		s.stats.TotalConnections++
		s.stats.ActiveConnections++
		s.stats.ConnectionsByUser[conn.UserID]++

		switch conn.Type {
		case ConnectionTypeWebSocket:
			s.stats.WebSocketConnections++
		case ConnectionTypeSSE:
			s.stats.SSEConnections++
		}
	} else {
		if s.stats.ActiveConnections > 0 {
			s.stats.ActiveConnections--
		}
		if s.stats.ConnectionsByUser[conn.UserID] > 0 {
			s.stats.ConnectionsByUser[conn.UserID]--
		}

		switch conn.Type {
		case ConnectionTypeWebSocket:
			if s.stats.WebSocketConnections > 0 {
				s.stats.WebSocketConnections--
			}
		case ConnectionTypeSSE:
			if s.stats.SSEConnections > 0 {
				s.stats.SSEConnections--
			}
		}
	}

	s.stats.LastUpdated = time.Now()
}

func (s *connectionManagerService) updateCurrentStats() {
	// Calculate average connection age
	s.connectionsMu.RLock()
	defer s.connectionsMu.RUnlock()

	if s.stats.ActiveConnections > 0 {
		var totalAge time.Duration
		activeCount := 0

		for _, userConnections := range s.connections {
			for _, conn := range userConnections {
				if conn.IsActive {
					totalAge += time.Since(conn.ConnectedAt)
					activeCount++
				}
			}
		}

		if activeCount > 0 {
			s.stats.AverageConnectionAge = totalAge / time.Duration(activeCount)
		}
	}
}

func (s *connectionManagerService) cleanupConnectionSubscriptions(
	userID string,
	connectionID uuid.UUID,
) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	// Remove from all channel subscriptions
	for channel, subscribers := range s.channelSubscribers {
		for i := len(subscribers) - 1; i >= 0; i-- {
			if subscribers[i].UserID == userID && subscribers[i].ConnectionID == connectionID {
				s.channelSubscribers[channel] = append(subscribers[:i], subscribers[i+1:]...)
				s.stats.ChannelSubscriptions[channel]--
			}
		}
	}
}

// Background workers

func (s *connectionManagerService) startBackgroundWorkers() {
	// Ping worker
	s.wg.Add(1)
	go s.pingWorker()

	// Cleanup worker
	s.wg.Add(1)
	go s.cleanupWorker()
}

func (s *connectionManagerService) pingWorker() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			ctx := context.Background()
			s.PingConnections(ctx)
		}
	}
}

func (s *connectionManagerService) cleanupWorker() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			ctx := context.Background()
			s.CleanupStaleConnections(ctx)
		}
	}
}

// Shutdown stops the connection manager and cleans up resources
func (s *connectionManagerService) Stop() {
	close(s.stopChan)
	s.wg.Wait()

	// Close all connections
	s.connectionsMu.Lock()
	defer s.connectionsMu.Unlock()

	for _, userConnections := range s.connections {
		for _, conn := range userConnections {
			if conn.IsActive && conn.Cancel != nil {
				conn.Cancel()
			}
		}
	}
}

// SSE Writer implementation

type sseWriter struct {
	writer  http.ResponseWriter
	flusher http.Flusher
}

func (w *sseWriter) WriteEvent(event Event) error {
	data, err := json.Marshal(event.Data)
	if err != nil {
		return err
	}

	if event.ID != uuid.Nil {
		fmt.Fprintf(w.writer, "id: %s\n", event.ID.String())
	}
	if event.Type != "" {
		fmt.Fprintf(w.writer, "event: %s\n", event.Type)
	}
	fmt.Fprintf(w.writer, "data: %s\n\n", string(data))

	w.flusher.Flush()
	return nil
}

func (w *sseWriter) WriteData(data []byte) error {
	fmt.Fprintf(w.writer, "data: %s\n\n", string(data))
	w.flusher.Flush()
	return nil
}

func (w *sseWriter) WriteComment(comment string) error {
	fmt.Fprintf(w.writer, ": %s\n", comment)
	w.flusher.Flush()
	return nil
}

func (w *sseWriter) Flush() error {
	w.flusher.Flush()
	return nil
}

func (w *sseWriter) Close() error {
	// HTTP connections are closed by the HTTP server
	return nil
}
