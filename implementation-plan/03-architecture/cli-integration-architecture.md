# CLI Integration Architecture

## Executive Summary

The MCP Portal CLI integration provides a secure web interface that wraps the existing MCP Gateway CLI functionality. Key architectural decisions include:

- **CLI Bridge Service Pattern**: Portal acts as secure bridge, not reimplementing CLI functionality
- **Command Execution**: All server operations execute underlying CLI commands with validation
- **Output Parsing**: Structured parsing of CLI output into JSON responses
- **Stream Management**: Real-time updates from CLI operations via WebSocket
- **Security Layer**: Command validation and parameter sanitization before execution
- **Non-Docker Desktop Support**: Works with standalone Docker Engine via socket mounting

This approach ensures we leverage the mature, tested CLI codebase while providing modern web capabilities.

## Overview

The MCP Portal provides a web-based interface that wraps the existing MCP Gateway CLI, which is a mature Docker plugin. The portal executes CLI commands and parses their output to provide a seamless web experience while leveraging all existing CLI functionality.

## Architecture Components

### CLI Bridge Service

The CLI Bridge Service acts as the critical interface layer between the web portal and the underlying CLI commands.

```go
// CLIBridge coordinates command execution and output parsing
type CLIBridge struct {
    executor    *CommandExecutor
    parser      *OutputParser
    streamer    *StreamManager
    security    *SecurityManager
}

// Execute runs a CLI command with proper security and parsing
func (cb *CLIBridge) Execute(ctx context.Context, cmd *Command) (*Result, error) {
    // 1. Validate command and parameters
    // 2. Execute with timeout and resource limits
    // 3. Parse output in real-time
    // 4. Stream updates to WebSocket clients
    // 5. Return structured result
}
```

### Command Execution Strategy

#### Secure Execution Environment

```go
type CommandExecutor struct {
    sandboxConfig *SandboxConfig
    timeouts      map[string]time.Duration
    retryPolicy   *RetryPolicy
    resourceLimits *ResourceLimits
}

type SandboxConfig struct {
    AllowedCommands []string          // Whitelist of allowed CLI commands
    MaxConcurrency  int               // Max parallel executions
    TempDirectory   string            // Isolated temp directory
    Environment     map[string]string // Controlled environment variables
}
```

#### Command Categories and Timeouts

```go
var commandTimeouts = map[string]time.Duration{
    "server-list":    5 * time.Second,   // Quick operations
    "server-enable":  30 * time.Second,  // Container operations
    "server-disable": 15 * time.Second,  // Stop operations
    "bulk-enable":    300 * time.Second, // Bulk operations
    "logs":          60 * time.Second,   // Log retrieval
}
```

### Output Parser Framework

The output parser handles different CLI output formats and converts them to structured data for the web interface.

#### Parser Interface

```go
type OutputParser interface {
    Parse(output []byte, cmd *Command) (*ParsedResult, error)
    SupportsStreaming() bool
    ParseStream(reader io.Reader, callback StreamCallback) error
}

type ParsedResult struct {
    Success    bool                   `json:"success"`
    Data       interface{}           `json:"data,omitempty"`
    Error      *ParsedError          `json:"error,omitempty"`
    Warnings   []string              `json:"warnings,omitempty"`
    Metadata   map[string]interface{} `json:"metadata,omitempty"`
}
```

#### Format-Specific Parsers

```go
// JSONParser for structured CLI output
type JSONParser struct{}

// TableParser for tabular CLI output
type TableParser struct {
    columns []string
    separator string
}

// LogParser for streaming log output
type LogParser struct {
    timestampPattern *regexp.Regexp
    levelPattern     *regexp.Regexp
}
```

### Stream Management for Real-time Updates

#### Stream Coordinator

```go
type StreamManager struct {
    activeStreams sync.Map // map[streamID]*Stream
    wsClients     sync.Map // map[clientID]*WebSocketClient
    eventBus      *EventBus
}

type Stream struct {
    ID          string
    Command     *Command
    StartTime   time.Time
    Status      StreamStatus
    Buffer      *RingBuffer
    Subscribers []string // Client IDs
}

func (sm *StreamManager) StartStream(cmd *Command, clientID string) (*Stream, error) {
    stream := &Stream{
        ID:        uuid.New().String(),
        Command:   cmd,
        StartTime: time.Now(),
        Status:    StreamStatusStarting,
        Buffer:    NewRingBuffer(1000), // Keep last 1000 lines
    }

    go sm.executeWithStreaming(stream)
    return stream, nil
}
```

### Error Handling and Recovery

#### Error Classification

```go
type ErrorCategory int

const (
    ErrorCategoryCLI ErrorCategory = iota     // CLI command errors
    ErrorCategoryParsing                      // Output parsing errors
    ErrorCategoryTimeout                      // Timeout errors
    ErrorCategoryPermission                   // Permission/security errors
    ErrorCategoryResource                     // Resource limit errors
)

type CLIError struct {
    Category    ErrorCategory `json:"category"`
    Command     string        `json:"command"`
    ExitCode    int          `json:"exit_code"`
    Message     string        `json:"message"`
    Output      string        `json:"output,omitempty"`
    Suggestion  string        `json:"suggestion,omitempty"`
    Recoverable bool         `json:"recoverable"`
}
```

#### Retry Mechanisms

```go
type RetryPolicy struct {
    MaxAttempts   int
    InitialDelay  time.Duration
    MaxDelay      time.Duration
    BackoffFactor float64
    RetryableCategories []ErrorCategory
}

func (rp *RetryPolicy) ShouldRetry(err *CLIError, attempt int) bool {
    if attempt >= rp.MaxAttempts {
        return false
    }

    // Only retry specific categories
    for _, category := range rp.RetryableCategories {
        if err.Category == category {
            return true
        }
    }
    return false
}
```

## Security Considerations

### Command Injection Prevention

```go
type SecurityManager struct {
    allowedCommands map[string]*CommandSpec
    validator       *InputValidator
}

type CommandSpec struct {
    Name       string
    Args       []ArgumentSpec
    MaxRuntime time.Duration
    RequiredPermissions []string
}

type ArgumentSpec struct {
    Name        string
    Type        ArgumentType  // STRING, NUMBER, BOOLEAN, UUID
    Required    bool
    Validator   *regexp.Regexp
    MaxLength   int
}

func (sm *SecurityManager) ValidateCommand(cmd *Command) error {
    spec, exists := sm.allowedCommands[cmd.Name]
    if !exists {
        return ErrCommandNotAllowed
    }

    return sm.validateArguments(cmd.Args, spec.Args)
}
```

### Resource Limiting

```go
type ResourceLimits struct {
    MaxMemory        int64          // Maximum memory usage
    MaxCPU          float64         // Maximum CPU usage (cores)
    MaxExecutionTime time.Duration  // Per-command timeout
    MaxOutputSize   int64           // Maximum output buffer size
    MaxConcurrent   int             // Max parallel commands
}

func (rl *ResourceLimits) CreateContext(ctx context.Context) context.Context {
    // Create context with resource limits
    ctx, cancel := context.WithTimeout(ctx, rl.MaxExecutionTime)

    // Add memory and CPU constraints
    return withResourceLimits(ctx, rl, cancel)
}
```

### Audit Trail

```go
type CommandAudit struct {
    ID          string                 `json:"id"`
    UserID      string                 `json:"user_id"`
    Command     string                 `json:"command"`
    Args        map[string]interface{} `json:"args"`
    StartTime   time.Time              `json:"start_time"`
    EndTime     *time.Time             `json:"end_time,omitempty"`
    Success     bool                   `json:"success"`
    ExitCode    int                    `json:"exit_code"`
    OutputSize  int64                  `json:"output_size"`
    IPAddress   string                 `json:"ip_address"`
    UserAgent   string                 `json:"user_agent"`
}
```

## Integration Patterns

### Synchronous Operations

For quick operations that complete within a few seconds:

```go
func (h *ServerHandler) GetServerList(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()

    cmd := &Command{
        Name: "server-list",
        Args: map[string]interface{}{
            "format": "json",
            "user":   getUserFromContext(ctx),
        },
    }

    result, err := h.cliBridge.Execute(ctx, cmd)
    if err != nil {
        handleError(w, err)
        return
    }

    writeJSON(w, result.Data)
}
```

### Asynchronous Operations

For long-running operations like bulk enables:

```go
func (h *ServerHandler) BulkEnable(w http.ResponseWriter, r *http.Request) {
    var req BulkOperationRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", 400)
        return
    }

    // Start operation asynchronously
    operationID := uuid.New().String()
    go h.executeBulkOperation(operationID, req)

    // Return operation ID for tracking
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "operation_id": operationID,
        "status":      "started",
    })
}
```

### WebSocket Streaming

For real-time updates and log streaming:

```go
func (h *WSHandler) HandleLogStream(conn *websocket.Conn) {
    var req LogStreamRequest
    if err := conn.ReadJSON(&req); err != nil {
        return
    }

    stream, err := h.streamManager.StartLogStream(&Command{
        Name: "server-logs",
        Args: map[string]interface{}{
            "server_id": req.ServerID,
            "follow":    true,
        },
    }, conn.ID)

    if err != nil {
        conn.WriteJSON(ErrorMessage{Error: err.Error()})
        return
    }

    // Stream updates to WebSocket
    for update := range stream.Updates {
        if err := conn.WriteJSON(update); err != nil {
            break
        }
    }
}
```

## Performance Optimization

### Command Result Caching

```go
type CacheManager struct {
    redis  *redis.Client
    ttls   map[string]time.Duration
}

func (cm *CacheManager) GetCachedResult(cmd *Command) (*ParsedResult, bool) {
    key := cm.generateCacheKey(cmd)
    data, err := cm.redis.Get(context.Background(), key).Result()
    if err != nil {
        return nil, false
    }

    var result ParsedResult
    if err := json.Unmarshal([]byte(data), &result); err != nil {
        return nil, false
    }

    return &result, true
}
```

### Connection Pool Management

```go
type CLIPool struct {
    available chan *CLIClient
    busy      sync.Map
    maxSize   int
}

func (p *CLIPool) Acquire(ctx context.Context) (*CLIClient, error) {
    select {
    case client := <-p.available:
        return client, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // Create new client if under limit
        return p.createClient()
    }
}
```

## Testing Strategy

### Unit Testing

```go
func TestCLIBridge_Execute(t *testing.T) {
    bridge := &CLIBridge{
        executor: &MockExecutor{},
        parser:   &MockParser{},
    }

    cmd := &Command{Name: "server-list"}
    result, err := bridge.Execute(context.Background(), cmd)

    assert.NoError(t, err)
    assert.True(t, result.Success)
}
```

### Integration Testing

```go
func TestRealCLIIntegration(t *testing.T) {
    // Test with actual CLI binary
    bridge := NewCLIBridge(RealExecutor{})

    // Test server listing
    cmd := &Command{Name: "server-list"}
    result, err := bridge.Execute(context.Background(), cmd)

    require.NoError(t, err)
    assert.IsType(t, []Server{}, result.Data)
}
```

### Performance Testing

```go
func BenchmarkCommandExecution(b *testing.B) {
    bridge := setupBridge()
    cmd := &Command{Name: "server-list"}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := bridge.Execute(context.Background(), cmd)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Monitoring and Observability

### Metrics Collection

```go
var (
    commandExecutionTime = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "cli_command_execution_seconds",
            Help: "CLI command execution time",
        },
        []string{"command", "success"},
    )

    commandExecutionCount = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cli_command_executions_total",
            Help: "Total CLI command executions",
        },
        []string{"command", "success"},
    )
)
```

### Health Monitoring

```go
type HealthChecker struct {
    cliBridge *CLIBridge
}

func (hc *HealthChecker) CheckCLIHealth() error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    cmd := &Command{Name: "version"}
    _, err := hc.cliBridge.Execute(ctx, cmd)
    return err
}
```

## Deployment Considerations

The CLI Integration Architecture requires:

1. **CLI Binary Availability**: The MCP CLI must be installed and available in the container PATH
2. **Docker Socket Access**: Required for container management operations
3. **Resource Monitoring**: Track CLI execution resource usage
4. **Log Aggregation**: Collect both portal and CLI operation logs
5. **Security Scanning**: Regular scans of CLI binary and execution environment

This architecture ensures secure, performant, and maintainable integration between the web portal and the existing CLI while preserving all existing functionality and adding web-specific enhancements.

## Non-Docker Desktop Deployment Architecture

### Socket Mounting Strategy

```yaml
# docker-compose.yml
version: "3.8"
services:
  mcp-portal:
    image: docker/mcp-portal:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - portal_config:/app/config
      - portal_logs:/app/logs
    environment:
      - DOCKER_HOST=unix:///var/run/docker.sock
      - MCP_CLI_PATH=/usr/local/bin/docker-mcp
    user: "${UID:-1000}:${DOCKER_GID:-999}"
    networks:
      - mcp-network
    depends_on:
      - postgres
      - redis

networks:
  mcp-network:
    driver: bridge
    ipam:
      config:
        - subnet: 10.20.0.0/16
```

### Container Runtime Configuration

```bash
# Set up proper Docker socket permissions
sudo groupadd docker-mcp
sudo usermod -aG docker-mcp mcp-portal
sudo chown root:docker-mcp /var/run/docker.sock
sudo chmod 660 /var/run/docker.sock

# Create systemd service for auto-start
cat > /etc/systemd/system/mcp-portal.service << 'EOF'
[Unit]
Description=MCP Portal Service
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/mcp-portal
Environment=COMPOSE_PROJECT_NAME=mcp_portal
ExecStart=/usr/bin/docker compose up -d
ExecStop=/usr/bin/docker compose down
TimeoutStartSec=300

[Install]
WantedBy=multi-user.target
EOF
```

### Secret Management Strategy

```yaml
# Using Docker secrets (Swarm mode)
secrets:
  jwt_private_key:
    file: ./secrets/jwt_private_key.pem
  azure_client_secret:
    file: ./secrets/azure_client_secret.txt
  postgres_password:
    file: ./secrets/postgres_password.txt

services:
  mcp-portal:
    secrets:
      - jwt_private_key
      - azure_client_secret
      - postgres_password
    environment:
      - JWT_PRIVATE_KEY_FILE=/run/secrets/jwt_private_key
      - AZURE_CLIENT_SECRET_FILE=/run/secrets/azure_client_secret
      - POSTGRES_PASSWORD_FILE=/run/secrets/postgres_password
```

### Volume Management

```bash
# Create persistent volume structure
sudo mkdir -p /opt/mcp-portal/{data,config,logs,backups}
sudo chown -R 1000:1000 /opt/mcp-portal

# Set up volume backup script
cat > /usr/local/bin/backup-mcp-portal.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/opt/mcp-portal/backups/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

# Stop services
docker compose -f /opt/mcp-portal/docker-compose.yml stop

# Backup volumes
docker run --rm \
  -v mcp_portal_data:/source:ro \
  -v "$BACKUP_DIR":/backup \
  alpine:latest \
  tar czf /backup/portal_data.tar.gz -C /source .

docker run --rm \
  -v mcp_portal_config:/source:ro \
  -v "$BACKUP_DIR":/backup \
  alpine:latest \
  tar czf /backup/portal_config.tar.gz -C /source .

# Restart services
docker compose -f /opt/mcp-portal/docker-compose.yml start

echo "Backup completed: $BACKUP_DIR"
EOF

chmod +x /usr/local/bin/backup-mcp-portal.sh
```

### Network Security Configuration

```bash
# Configure iptables for container isolation
iptables -I DOCKER-USER -i mcp-bridge -o mcp-bridge -j ACCEPT
iptables -I DOCKER-USER -i mcp-bridge ! -o mcp-bridge -j ACCEPT
iptables -I DOCKER-USER -o mcp-bridge -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
iptables -I DOCKER-USER -o mcp-bridge -j DROP

# Save iptables rules
iptables-save > /etc/iptables/rules.v4
```

## CLI Error Handling and Recovery Patterns

### Error Classification and Response

```go
type CLIErrorHandler struct {
    patterns        map[string]*ErrorPattern
    retryPolicies   map[ErrorCategory]*RetryPolicy
    suggestions     map[string]string
}

type ErrorPattern struct {
    Regex       *regexp.Regexp
    Category    ErrorCategory
    Severity    ErrorSeverity
    Recoverable bool
    HTTPStatus  int
    UserMessage string
}

func (eh *CLIErrorHandler) HandleError(output string, exitCode int) *CLIResponse {
    for _, pattern := range eh.patterns {
        if pattern.Regex.MatchString(output) {
            return &CLIResponse{
                Success:     false,
                Error:       eh.createUserError(pattern, output),
                Recoverable: pattern.Recoverable,
                HTTPStatus:  pattern.HTTPStatus,
            }
        }
    }

    return eh.handleUnknownError(output, exitCode)
}
```

### Retry Mechanisms

```go
type RetryPolicy struct {
    MaxAttempts      int
    InitialDelay     time.Duration
    MaxDelay         time.Duration
    BackoffFactor    float64
    JitterEnabled    bool
    RetryableErrors  []ErrorCategory
}

func (rp *RetryPolicy) Execute(ctx context.Context, operation func() error) error {
    var lastError error

    for attempt := 1; attempt <= rp.MaxAttempts; attempt++ {
        if err := operation(); err == nil {
            return nil
        } else if !rp.shouldRetry(err, attempt) {
            return err
        }

        delay := rp.calculateDelay(attempt)
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(delay):
            // Continue to next attempt
        }
    }

    return fmt.Errorf("max retries exceeded: %w", lastError)
}
```

### Circuit Breaker Pattern

```go
type CLICircuitBreaker struct {
    state         CircuitState
    failures      int
    lastFailure   time.Time
    timeout       time.Duration
    threshold     int
    mu            sync.RWMutex
}

func (cb *CLICircuitBreaker) Execute(operation func() error) error {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    switch cb.state {
    case StateClosed:
        return cb.executeInClosed(operation)
    case StateOpen:
        return cb.executeInOpen(operation)
    case StateHalfOpen:
        return cb.executeInHalfOpen(operation)
    default:
        return errors.New("unknown circuit breaker state")
    }
}
```

## Performance Optimization Strategies

### Command Result Caching

```go
type CLICache struct {
    redis       *redis.Client
    defaultTTL  time.Duration
    cacheTTLs   map[string]time.Duration
    keyPrefix   string
}

func (cc *CLICache) GetOrExecute(ctx context.Context, cmd *Command, executor func() (*ParsedResult, error)) (*ParsedResult, error) {
    // Try cache first
    cacheKey := cc.generateKey(cmd)
    if cached, err := cc.get(ctx, cacheKey); err == nil {
        return cached, nil
    }

    // Execute command
    result, err := executor()
    if err != nil {
        return nil, err
    }

    // Cache successful results
    if result.Success {
        ttl := cc.getTTL(cmd.Name)
        cc.set(ctx, cacheKey, result, ttl)
    }

    return result, nil
}
```

### Connection Pooling for CLI Operations

```go
type CLIPool struct {
    available   chan *CLIExecutor
    busy        sync.Map
    maxSize     int
    timeout     time.Duration
    healthCheck func(*CLIExecutor) error
}

func (cp *CLIPool) Acquire(ctx context.Context) (*CLIExecutor, error) {
    select {
    case executor := <-cp.available:
        if err := cp.healthCheck(executor); err != nil {
            return cp.createNewExecutor()
        }
        return executor, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    case <-time.After(cp.timeout):
        return nil, errors.New("timeout acquiring CLI executor")
    }
}

func (cp *CLIPool) Release(executor *CLIExecutor) {
    select {
    case cp.available <- executor:
        // Successfully returned to pool
    default:
        // Pool is full, close the executor
        executor.Close()
    }
}
```

### Stream Buffer Management

```go
type CircularBuffer struct {
    data     [][]byte
    size     int
    head     int
    tail     int
    full     bool
    mu       sync.RWMutex
}

func (cb *CircularBuffer) Write(data []byte) {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    cb.data[cb.head] = make([]byte, len(data))
    copy(cb.data[cb.head], data)

    if cb.full {
        cb.tail = (cb.tail + 1) % cb.size
    }

    cb.head = (cb.head + 1) % cb.size
    cb.full = cb.head == cb.tail
}

func (cb *CircularBuffer) ReadAll() [][]byte {
    cb.mu.RLock()
    defer cb.mu.RUnlock()

    if !cb.full && cb.head == cb.tail {
        return nil
    }

    var result [][]byte
    start := cb.tail
    if cb.full {
        start = cb.head
    }

    for i := start; i != cb.head; i = (i + 1) % cb.size {
        result = append(result, cb.data[i])
    }

    return result
}
```

This comprehensive architecture provides a robust foundation for integrating the web portal with the existing CLI while maintaining security, performance, and reliability standards required for production deployment.
