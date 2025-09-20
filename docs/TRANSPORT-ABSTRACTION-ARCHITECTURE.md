# Transport Abstraction Architecture

## Overview

The Transport Abstraction Pattern provides a clean, maintainable architecture for handling different transport mechanisms (stdio, HTTP, SSE) in the MCP Gateway while maintaining strict channel separation for stdio-based JSON-RPC protocols.

## Core Principle

```
All Transports:
├── Protocol Messages → Transport-specific channel (stdout for stdio, HTTP body, SSE data)
└── Operational Logs  → Always stderr (consistent across all transports)
```

## Architecture Design

### Transport Interface

```go
type MCPTransport interface {
    Name() string
    Logger() TransportLogger
    IsProtocolChannel() bool
    GetReader() io.Reader
    GetWriter() io.Writer
    Close() error
}
```

### Logger Interface

```go
type TransportLogger interface {
    Log(a ...any)
    Logf(format string, a ...any)
    SetLevel(level LogLevel)
    IsQuiet() bool
}
```

## Implementation Details

### File Structure

```
cmd/docker-mcp/internal/gateway/
├── transport.go                   # Existing transport implementations
├── transport_abstraction.go       # New abstraction layer
├── run.go                         # Updated to use transport abstraction
└── logs.go                        # Original logging (can be deprecated)
```

### Transport Implementations

#### 1. StdioTransportWrapper

**Purpose**: Handles stdio-based communication for JSON-RPC protocol

**Key Features**:

- Reader: stdin
- Writer: stdout (protocol messages only)
- Logger: stderr (all operational logs)
- Sacred channel separation maintained

```go
type StdioTransportWrapper struct {
    stdin  io.Reader    // Input from client
    stdout io.Writer    // JSON-RPC protocol only
    logger *StderrLogger // Always logs to stderr
}
```

#### 2. HTTPTransportWrapper

**Purpose**: Handles HTTP-based communication

**Key Features**:

- Reader: HTTP request body
- Writer: HTTP response body
- Logger: stderr (consistent with stdio)
- Protocol in HTTP body

```go
type HTTPTransportWrapper struct {
    server   *http.Server
    listener net.Listener
    logger   *StderrLogger // Always logs to stderr
    reader   io.Reader
    writer   io.Writer
}
```

#### 3. SSETransportWrapper

**Purpose**: Handles Server-Sent Events for one-way communication

**Key Features**:

- Reader: nil (write-only transport)
- Writer: SSE event stream
- Logger: stderr (consistent with other transports)
- Events only, no bidirectional protocol

```go
type SSETransportWrapper struct {
    server   *http.Server
    listener net.Listener
    logger   *StderrLogger // Always logs to stderr
    writer   io.Writer
}
```

## Benefits

### 1. Consistent Logging

All transports log to stderr, ensuring:

- Protocol purity (no stdout pollution in stdio mode)
- Consistent log location across all transports
- Easy log aggregation and monitoring

### 2. Flexibility

The abstraction allows:

- Easy addition of new transport types
- Transport-specific optimizations
- Testing with mock transports

### 3. Maintainability

- Single place to manage transport behavior
- Clear separation of concerns
- Transport-agnostic gateway logic

### 4. Compliance

- Full compliance with MCP/LSP specifications
- Proper channel separation for stdio protocols
- Industry-standard logging patterns

## Usage

### Gateway Integration

```go
func (g *Gateway) Run(ctx context.Context) error {
    // Create appropriate transport
    factory := &TransportFactory{}
    transport, err := factory.CreateTransport(g.Transport, listener)
    if err != nil {
        return err
    }
    g.transport = transport
    defer g.transport.Close()

    // Use transport for logging
    LogWithTransport(g.transport, "Starting gateway...")

    // Transport provides appropriate channels
    reader := g.transport.GetReader()
    writer := g.transport.GetWriter()
    // ...
}
```

### Logging with Transport

```go
// All logs go to stderr regardless of transport
LogWithTransport(transport, "Server started")
LogfWithTransport(transport, "Processing request: %s", requestID)
```

## Migration Path

### Phase 1: Transport Abstraction (Complete)

- ✅ Define transport interfaces
- ✅ Implement transport wrappers
- ✅ Create transport factory
- ✅ Integrate with gateway

### Phase 2: Full Integration (Complete)

- ✅ Updated logs.go to use transport abstraction with fallback
- ✅ Added SetGlobalTransport for gateway initialization
- ✅ Migrated all fmt.Fprintf calls in run.go to use log/logf
- ✅ Migrated all fmt.Fprintf calls in handlers.go to use log/logf
- ✅ Created integration tests verifying channel separation
- ✅ All 83 log calls now route through transport abstraction
- ✅ Verified STDIO protocol compliance with stderr logging

### Phase 3: Enhanced Features (Future)

- Add metrics per transport
- Implement transport-specific optimizations
- Add connection pooling for HTTP
- Support WebSocket transport

## Testing Strategy

### Unit Tests

```go
func TestStdioTransportChannelSeparation(t *testing.T) {
    transport := NewStdioTransportWrapper()

    // Protocol goes to stdout
    writer := transport.GetWriter()
    writer.Write([]byte(`{"jsonrpc":"2.0"}`))

    // Logs go to stderr
    transport.Logger().Log("Operational message")

    // Verify separation
    // stdout contains only JSON-RPC
    // stderr contains only logs
}
```

### Integration Tests

- Test each transport type
- Verify channel separation
- Validate protocol compliance
- Check log consistency

## Best Practices

### DO

- ✅ Always use transport logger for operational messages
- ✅ Keep protocol messages separate from logs
- ✅ Use transport factory for creating transports
- ✅ Close transports properly

### DON'T

- ❌ Write directly to stdout/stderr
- ❌ Mix protocol and log messages
- ❌ Assume transport type in gateway logic
- ❌ Create transports without factory

## Conclusion

The Transport Abstraction Pattern provides a robust, maintainable solution for handling multiple transport types while maintaining strict protocol compliance. By ensuring all operational logs go to stderr regardless of transport type, we achieve both protocol purity and operational visibility across all communication channels.
