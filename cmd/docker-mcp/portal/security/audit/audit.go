package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// EventType represents different types of audit events
type EventType string

const (
	EventTypeCommandExecution EventType = "command_execution"
	EventTypeCommandSuccess   EventType = "command_success"
	EventTypeCommandFailure   EventType = "command_failure"
	EventTypeSecurityAlert    EventType = "security_alert"
	EventTypeRateLimitHit     EventType = "rate_limit_hit"
	EventTypeAccessDenied     EventType = "access_denied"
	EventTypeAuthentication   EventType = "authentication"
	EventTypeConfiguration    EventType = "configuration_change"
	EventTypeDataAccess       EventType = "data_access"
	EventTypeExecution        EventType = "execution"
	EventTypeBulkOperation    EventType = "bulk_operation"
	EventTypeWarning          EventType = "warning"
)

// Action represents audit action types for structured logging
type Action string

const (
	ActionCreate  Action = "create"
	ActionRead    Action = "read"
	ActionUpdate  Action = "update"
	ActionDelete  Action = "delete"
	ActionExecute Action = "execute"
	ActionImport  Action = "import"
	ActionExport  Action = "export"
	ActionLogin   Action = "login"
	ActionLogout  Action = "logout"
)

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	ID        uuid.UUID              `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    uuid.UUID              `json:"user_id"`
	EventType EventType              `json:"event_type"`
	Command   string                 `json:"command,omitempty"`
	Arguments []string               `json:"arguments,omitempty"`
	Result    string                 `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Severity  string                 `json:"severity"`
	SessionID string                 `json:"session_id,omitempty"`
}

// Logger interface for audit logging
type Logger interface {
	LogCommand(ctx context.Context, userID uuid.UUID, command string, args []string) uuid.UUID
	LogCommandResult(
		ctx context.Context,
		auditID uuid.UUID,
		result string,
		err error,
		duration time.Duration,
	)
	LogSecurityEvent(
		ctx context.Context,
		userID uuid.UUID,
		event EventType,
		details map[string]interface{},
	)
	LogAccessDenied(ctx context.Context, userID uuid.UUID, resource string, reason string)
	LogRateLimitExceeded(ctx context.Context, userID uuid.UUID, command string)
	GetLogs(ctx context.Context, userID uuid.UUID, limit int) ([]AuditEntry, error)

	// Log provides a general-purpose audit logging method
	Log(
		ctx context.Context,
		action Action,
		resource string,
		resourceID string,
		userID string,
		metadata map[string]interface{},
	) error
}

// AuditLogger implements the Logger interface
type AuditLogger struct {
	storage Storage
}

// Storage interface for audit log persistence
type Storage interface {
	Store(ctx context.Context, entry AuditEntry) error
	Retrieve(ctx context.Context, userID uuid.UUID, limit int) ([]AuditEntry, error)
	RetrieveByID(ctx context.Context, id uuid.UUID) (*AuditEntry, error)
}

// NewLogger creates a new audit logger
func NewLogger(storage Storage) *AuditLogger {
	return &AuditLogger{
		storage: storage,
	}
}

// LogCommand logs the start of a command execution
func (l *AuditLogger) LogCommand(
	ctx context.Context,
	userID uuid.UUID,
	command string,
	args []string,
) uuid.UUID {
	auditID := uuid.New()
	entry := AuditEntry{
		ID:        auditID,
		Timestamp: time.Now().UTC(),
		UserID:    userID,
		EventType: EventTypeCommandExecution,
		Command:   command,
		Arguments: args,
		Severity:  "info",
	}

	// Extract metadata from context if available
	if sessionID, ok := ctx.Value("session_id").(string); ok {
		entry.SessionID = sessionID
	}
	if ipAddress, ok := ctx.Value("ip_address").(string); ok {
		entry.IPAddress = ipAddress
	}
	if userAgent, ok := ctx.Value("user_agent").(string); ok {
		entry.UserAgent = userAgent
	}

	// Store asynchronously to avoid blocking
	go func() {
		if err := l.storage.Store(context.Background(), entry); err != nil {
			// Log storage error (would typically go to a fallback logger)
			fmt.Printf("Failed to store audit log: %v\n", err)
		}
	}()

	return auditID
}

// LogCommandResult logs the result of a command execution
func (l *AuditLogger) LogCommandResult(
	ctx context.Context,
	auditID uuid.UUID,
	result string,
	err error,
	duration time.Duration,
) {
	eventType := EventTypeCommandSuccess
	severity := "info"
	var errorMsg string

	if err != nil {
		eventType = EventTypeCommandFailure
		severity = "error"
		errorMsg = err.Error()
	}

	entry := AuditEntry{
		ID:        auditID,
		Timestamp: time.Now().UTC(),
		EventType: eventType,
		Result:    result,
		Error:     errorMsg,
		Duration:  duration,
		Severity:  severity,
	}

	// Store asynchronously
	go func() {
		if err := l.storage.Store(context.Background(), entry); err != nil {
			fmt.Printf("Failed to store audit result: %v\n", err)
		}
	}()
}

// LogSecurityEvent logs a security-related event
func (l *AuditLogger) LogSecurityEvent(
	ctx context.Context,
	userID uuid.UUID,
	event EventType,
	details map[string]interface{},
) {
	entry := AuditEntry{
		ID:        uuid.New(),
		Timestamp: time.Now().UTC(),
		UserID:    userID,
		EventType: event,
		Metadata:  details,
		Severity:  "warning",
	}

	// For security alerts, use critical severity
	if event == EventTypeSecurityAlert {
		entry.Severity = "critical"
	}

	// Store asynchronously with higher priority
	go func() {
		if err := l.storage.Store(context.Background(), entry); err != nil {
			fmt.Printf("Failed to store security event: %v\n", err)
		}
	}()
}

// LogAccessDenied logs an access denied event
func (l *AuditLogger) LogAccessDenied(
	ctx context.Context,
	userID uuid.UUID,
	resource string,
	reason string,
) {
	l.LogSecurityEvent(ctx, userID, EventTypeAccessDenied, map[string]interface{}{
		"resource": resource,
		"reason":   reason,
	})
}

// LogRateLimitExceeded logs a rate limit exceeded event
func (l *AuditLogger) LogRateLimitExceeded(ctx context.Context, userID uuid.UUID, command string) {
	l.LogSecurityEvent(ctx, userID, EventTypeRateLimitHit, map[string]interface{}{
		"command": command,
	})
}

// GetLogs retrieves audit logs for a user
func (l *AuditLogger) GetLogs(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
) ([]AuditEntry, error) {
	return l.storage.Retrieve(ctx, userID, limit)
}

// Log provides a general-purpose audit logging method
func (l *AuditLogger) Log(
	ctx context.Context,
	action Action,
	resource string,
	resourceID string,
	userID string,
	metadata map[string]interface{},
) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	entry := AuditEntry{
		ID:        uuid.New(),
		Timestamp: time.Now().UTC(),
		UserID:    userUUID,
		EventType: EventTypeConfiguration, // Default event type for general logging
		Metadata: map[string]interface{}{
			"action":      string(action),
			"resource":    resource,
			"resource_id": resourceID,
		},
		Severity: "info",
	}

	// Add provided metadata
	if metadata != nil {
		for k, v := range metadata {
			entry.Metadata[k] = v
		}
	}

	// Extract context metadata
	if sessionID, ok := ctx.Value("session_id").(string); ok {
		entry.SessionID = sessionID
	}
	if ipAddress, ok := ctx.Value("ip_address").(string); ok {
		entry.IPAddress = ipAddress
	}
	if userAgent, ok := ctx.Value("user_agent").(string); ok {
		entry.UserAgent = userAgent
	}

	// Store asynchronously
	go func() {
		if err := l.storage.Store(context.Background(), entry); err != nil {
			fmt.Printf("Failed to store audit log: %v\n", err)
		}
	}()

	return nil
}

// MemoryStorage provides in-memory storage for testing
type MemoryStorage struct {
	entries []AuditEntry
}

// NewMemoryStorage creates a new in-memory storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		entries: make([]AuditEntry, 0),
	}
}

// Store saves an entry in memory
func (m *MemoryStorage) Store(ctx context.Context, entry AuditEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}

// Retrieve gets entries from memory
func (m *MemoryStorage) Retrieve(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
) ([]AuditEntry, error) {
	var result []AuditEntry
	count := 0

	// Iterate in reverse for most recent first
	for i := len(m.entries) - 1; i >= 0 && count < limit; i-- {
		if m.entries[i].UserID == userID {
			result = append(result, m.entries[i])
			count++
		}
	}

	return result, nil
}

// RetrieveByID gets a specific entry by ID
func (m *MemoryStorage) RetrieveByID(ctx context.Context, id uuid.UUID) (*AuditEntry, error) {
	for _, entry := range m.entries {
		if entry.ID == id {
			return &entry, nil
		}
	}
	return nil, fmt.Errorf("audit entry not found: %s", id)
}

// ToJSON converts an audit entry to JSON
func (e *AuditEntry) ToJSON() (string, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return "", fmt.Errorf("failed to marshal audit entry: %w", err)
	}
	return string(data), nil
}
