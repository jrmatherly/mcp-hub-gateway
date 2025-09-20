package gateway

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
)

// TransportLogger defines the interface for transport-specific logging
type TransportLogger interface {
	Log(a ...any)
	Logf(format string, a ...any)
	SetLevel(level LogLevel)
	IsQuiet() bool
}

// LogLevel defines different logging severity levels
type LogLevel int

const (
	LogLevelError LogLevel = iota
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

// MCPTransport defines the abstraction interface for different transport mechanisms
// This allows the gateway to work with any transport type without knowing the details
type MCPTransport interface {
	// Name returns the name of the transport for debugging
	Name() string

	// Logger returns the logger for this transport
	Logger() TransportLogger

	// IsProtocolChannel returns true if this channel is used for protocol messages
	// This helps determine where to send different types of output
	IsProtocolChannel() bool

	// GetReader returns the reader for incoming data
	GetReader() io.Reader

	// GetWriter returns the writer for outgoing data
	GetWriter() io.Writer

	// Close closes the transport connection
	Close() error
}

// StderrLogger implements TransportLogger that always outputs to stderr
type StderrLogger struct {
	prefix   string
	level    LogLevel
	minLevel LogLevel
	quiet    bool
}

// NewStderrLogger creates a new logger that outputs to stderr
func NewStderrLogger(prefix string, minLevel LogLevel) *StderrLogger {
	return &StderrLogger{
		prefix:   prefix,
		level:    LogLevelInfo,
		minLevel: minLevel,
		quiet:    false,
	}
}

// Log logs a message to stderr
func (l *StderrLogger) Log(a ...any) {
	if l.quiet || l.level < l.minLevel {
		return
	}

	// Always output to stderr for all transports
	fmt.Fprintln(os.Stderr, a...)
}

// Logf logs a formatted message to stderr
func (l *StderrLogger) Logf(format string, a ...any) {
	if l.quiet || l.level < l.minLevel {
		return
	}

	message := fmt.Sprintf(format, a...)
	if message[len(message)-1] != '\n' {
		message += "\n"
	}
	// Always output to stderr for all transports
	fmt.Fprint(os.Stderr, message)
}

// SetLevel sets the current logging level
func (l *StderrLogger) SetLevel(level LogLevel) {
	l.level = level
}

// IsQuiet returns whether the logger is in quiet mode
func (l *StderrLogger) IsQuiet() bool {
	return l.quiet
}

// StdioTransportWrapper wraps stdio for MCP communication
type StdioTransportWrapper struct {
	stdin  io.Reader
	stdout io.Writer
	logger *StderrLogger
}

// NewStdioTransportWrapper creates a new stdio transport wrapper
func NewStdioTransportWrapper() *StdioTransportWrapper {
	return &StdioTransportWrapper{
		stdin:  os.Stdin,
		stdout: os.Stdout,
		// All logs go to stderr for stdio transport - this is the key principle
		logger: NewStderrLogger("[MCP]", LogLevelInfo),
	}
}

// Name returns the transport name
func (t *StdioTransportWrapper) Name() string {
	return "stdio"
}

// Logger returns the transport logger (writes to stderr)
func (t *StdioTransportWrapper) Logger() TransportLogger {
	return t.logger
}

// IsProtocolChannel returns true as stdio stdout is used for protocol
func (t *StdioTransportWrapper) IsProtocolChannel() bool {
	return true
}

// GetReader returns stdin for reading
func (t *StdioTransportWrapper) GetReader() io.Reader {
	return t.stdin
}

// GetWriter returns stdout for writing protocol messages
func (t *StdioTransportWrapper) GetWriter() io.Writer {
	return t.stdout
}

// Close closes the transport (no-op for stdio)
func (t *StdioTransportWrapper) Close() error {
	// stdio streams are managed by the OS
	return nil
}

// HTTPTransportWrapper wraps HTTP for MCP communication
type HTTPTransportWrapper struct {
	server   *http.Server
	listener net.Listener
	logger   *StderrLogger
	reader   io.Reader
	writer   io.Writer
}

// NewHTTPTransportWrapper creates a new HTTP transport wrapper
func NewHTTPTransportWrapper(listener net.Listener) *HTTPTransportWrapper {
	return &HTTPTransportWrapper{
		listener: listener,
		// HTTP transport logs also go to stderr for consistency
		logger: NewStderrLogger("[MCP-HTTP]", LogLevelInfo),
	}
}

// Name returns the transport name
func (t *HTTPTransportWrapper) Name() string {
	return "http"
}

// Logger returns the transport logger
func (t *HTTPTransportWrapper) Logger() TransportLogger {
	return t.logger
}

// IsProtocolChannel returns true as HTTP body is used for protocol
func (t *HTTPTransportWrapper) IsProtocolChannel() bool {
	return true
}

// GetReader returns the reader for incoming HTTP requests
func (t *HTTPTransportWrapper) GetReader() io.Reader {
	// In practice, this would be connected to the HTTP request body
	// For now, return a placeholder
	return t.reader
}

// GetWriter returns the writer for HTTP responses
func (t *HTTPTransportWrapper) GetWriter() io.Writer {
	// In practice, this would be connected to the HTTP response writer
	// For now, return a placeholder
	return t.writer
}

// Close closes the HTTP server
func (t *HTTPTransportWrapper) Close() error {
	if t.server != nil {
		return t.server.Shutdown(context.Background())
	}
	return nil
}

// SetServer sets the HTTP server for this transport
func (t *HTTPTransportWrapper) SetServer(server *http.Server) {
	t.server = server
}

// SSETransportWrapper wraps Server-Sent Events for MCP communication
type SSETransportWrapper struct {
	server   *http.Server
	listener net.Listener
	logger   *StderrLogger
	writer   io.Writer
}

// NewSSETransportWrapper creates a new SSE transport wrapper
func NewSSETransportWrapper(listener net.Listener) *SSETransportWrapper {
	return &SSETransportWrapper{
		listener: listener,
		// SSE transport logs also go to stderr
		logger: NewStderrLogger("[MCP-SSE]", LogLevelInfo),
	}
}

// Name returns the transport name
func (t *SSETransportWrapper) Name() string {
	return "sse"
}

// Logger returns the transport logger
func (t *SSETransportWrapper) Logger() TransportLogger {
	return t.logger
}

// IsProtocolChannel returns false as SSE is for events, not bidirectional protocol
func (t *SSETransportWrapper) IsProtocolChannel() bool {
	return false
}

// GetReader returns nil as SSE is write-only
func (t *SSETransportWrapper) GetReader() io.Reader {
	return nil
}

// GetWriter returns the writer for SSE events
func (t *SSETransportWrapper) GetWriter() io.Writer {
	return t.writer
}

// Close closes the SSE server
func (t *SSETransportWrapper) Close() error {
	if t.server != nil {
		return t.server.Shutdown(context.Background())
	}
	return nil
}

// SetServer sets the HTTP server for this transport
func (t *SSETransportWrapper) SetServer(server *http.Server) {
	t.server = server
}

// TransportFactory creates the appropriate transport based on configuration
type TransportFactory struct{}

// CreateTransport creates a transport wrapper based on the type
func (f *TransportFactory) CreateTransport(
	transportType string,
	listener net.Listener,
) (MCPTransport, error) {
	switch transportType {
	case "stdio":
		return NewStdioTransportWrapper(), nil

	case "http", "streaming", "streamable", "streamable-http":
		if listener == nil {
			return nil, fmt.Errorf("HTTP transport requires a listener")
		}
		return NewHTTPTransportWrapper(listener), nil

	case "sse":
		if listener == nil {
			return nil, fmt.Errorf("SSE transport requires a listener")
		}
		return NewSSETransportWrapper(listener), nil

	default:
		return nil, fmt.Errorf("unknown transport type: %s", transportType)
	}
}

// LogWithTransport logs a message using the transport's logger
// This ensures all logs go to the appropriate channel (stderr for all transports)
func LogWithTransport(transport MCPTransport, a ...any) {
	if transport != nil && transport.Logger() != nil {
		transport.Logger().Log(a...)
	} else {
		// Fallback to stderr if no transport is available
		fmt.Fprintln(os.Stderr, a...)
	}
}

// LogfWithTransport logs a formatted message using the transport's logger
func LogfWithTransport(transport MCPTransport, format string, a ...any) {
	if transport != nil && transport.Logger() != nil {
		transport.Logger().Logf(format, a...)
	} else {
		// Fallback to stderr if no transport is available
		fmt.Fprintf(os.Stderr, format+"\n", a...)
	}
}
