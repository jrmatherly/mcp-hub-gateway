// Package realtime provides mock implementations for testing.
package realtime

import (
	"context"

	"github.com/google/uuid"
)

// MockConnectionManager is a mock implementation of ConnectionManager for testing
type MockConnectionManager struct {
	connections map[string]map[uuid.UUID]*Connection
	subscribers map[string][]ChannelSubscriber
}

// NewMockConnectionManager creates a new mock connection manager
func NewMockConnectionManager() *MockConnectionManager {
	return &MockConnectionManager{
		connections: make(map[string]map[uuid.UUID]*Connection),
		subscribers: make(map[string][]ChannelSubscriber),
	}
}

func (m *MockConnectionManager) AddConnection(
	ctx context.Context,
	userID string,
	conn *Connection,
) error {
	if m.connections[userID] == nil {
		m.connections[userID] = make(map[uuid.UUID]*Connection)
	}
	m.connections[userID][conn.ID] = conn
	return nil
}

func (m *MockConnectionManager) RemoveConnection(userID string, connectionID uuid.UUID) error {
	if userConnections, exists := m.connections[userID]; exists {
		delete(userConnections, connectionID)
		if len(userConnections) == 0 {
			delete(m.connections, userID)
		}
	}
	return nil
}

func (m *MockConnectionManager) GetConnections(userID string) []*Connection {
	if userConnections, exists := m.connections[userID]; exists {
		connections := make([]*Connection, 0, len(userConnections))
		for _, conn := range userConnections {
			connections = append(connections, conn)
		}
		return connections
	}
	return nil
}

func (m *MockConnectionManager) GetConnection(
	userID string,
	connectionID uuid.UUID,
) (*Connection, bool) {
	if userConnections, exists := m.connections[userID]; exists {
		conn, exists := userConnections[connectionID]
		return conn, exists
	}
	return nil, false
}

func (m *MockConnectionManager) BroadcastToUser(userID string, event Event) error {
	return nil // Mock implementation
}

func (m *MockConnectionManager) BroadcastToAll(event Event) error {
	return nil // Mock implementation
}

func (m *MockConnectionManager) BroadcastToChannel(channel string, event Event) error {
	return nil // Mock implementation
}

func (m *MockConnectionManager) Subscribe(
	userID string,
	connectionID uuid.UUID,
	channel string,
) error {
	subscriber := ChannelSubscriber{
		UserID:       userID,
		ConnectionID: connectionID,
	}
	m.subscribers[channel] = append(m.subscribers[channel], subscriber)
	return nil
}

func (m *MockConnectionManager) Unsubscribe(
	userID string,
	connectionID uuid.UUID,
	channel string,
) error {
	if subscribers, exists := m.subscribers[channel]; exists {
		for i, subscriber := range subscribers {
			if subscriber.UserID == userID && subscriber.ConnectionID == connectionID {
				m.subscribers[channel] = append(subscribers[:i], subscribers[i+1:]...)
				break
			}
		}
	}
	return nil
}

func (m *MockConnectionManager) GetChannelSubscribers(channel string) []ChannelSubscriber {
	if subscribers, exists := m.subscribers[channel]; exists {
		result := make([]ChannelSubscriber, len(subscribers))
		copy(result, subscribers)
		return result
	}
	return nil
}

func (m *MockConnectionManager) StreamEvent(ctx context.Context, event Event) error {
	return nil // Mock implementation
}

func (m *MockConnectionManager) StreamToConnection(
	ctx context.Context,
	connectionID uuid.UUID,
	event Event,
) error {
	return nil // Mock implementation
}

func (m *MockConnectionManager) PingConnections(ctx context.Context) error {
	return nil // Mock implementation
}

func (m *MockConnectionManager) GetConnectionStats() ConnectionStats {
	return ConnectionStats{} // Mock implementation
}

func (m *MockConnectionManager) CleanupStaleConnections(ctx context.Context) error {
	return nil // Mock implementation
}

func (m *MockConnectionManager) HandleWebSocket(c any) {
	// Mock implementation - would normally upgrade HTTP connection to WebSocket
}

func (m *MockConnectionManager) HandleSSE(c any) {
	// Mock implementation - would normally set up Server-Sent Events connection
}

func (m *MockConnectionManager) Stop() {
	// Mock implementation - would normally stop connection manager and cleanup resources
}
