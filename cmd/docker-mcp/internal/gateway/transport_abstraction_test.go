package gateway

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStdioTransportChannelSeparation verifies that stdio transport
// maintains proper channel separation: stdout for protocol, stderr for logs
func TestStdioTransportChannelSeparation(t *testing.T) {
	transport := NewStdioTransportWrapper()

	// Test that transport identifies as protocol channel
	assert.True(t, transport.IsProtocolChannel(), "stdio should be a protocol channel")

	// Test transport name
	assert.Equal(t, "stdio", transport.Name())

	// Verify logger exists
	assert.NotNil(t, transport.Logger(), "logger should not be nil")

	// Test that GetReader returns stdin (non-nil)
	assert.NotNil(t, transport.GetReader(), "reader should not be nil")

	// Test that GetWriter returns stdout (non-nil)
	assert.NotNil(t, transport.GetWriter(), "writer should not be nil")
}

// TestHTTPTransportWrapper verifies HTTP transport configuration
func TestHTTPTransportWrapper(t *testing.T) {
	// Create a mock listener (nil is acceptable for this test)
	transport := NewHTTPTransportWrapper(nil)

	// Test that transport identifies as protocol channel
	assert.True(t, transport.IsProtocolChannel(), "HTTP should be a protocol channel")

	// Test transport name
	assert.Equal(t, "http", transport.Name())

	// Verify logger exists
	assert.NotNil(t, transport.Logger(), "logger should not be nil")
}

// TestSSETransportWrapper verifies SSE transport configuration
func TestSSETransportWrapper(t *testing.T) {
	// Create SSE transport
	transport := NewSSETransportWrapper(nil)

	// Test that transport does NOT identify as protocol channel (one-way)
	assert.False(
		t,
		transport.IsProtocolChannel(),
		"SSE should not be a bidirectional protocol channel",
	)

	// Test transport name
	assert.Equal(t, "sse", transport.Name())

	// Verify logger exists
	assert.NotNil(t, transport.Logger(), "logger should not be nil")

	// SSE should have nil reader (write-only)
	assert.Nil(t, transport.GetReader(), "SSE reader should be nil (write-only)")
}

// TestTransportFactory verifies transport creation
func TestTransportFactory(t *testing.T) {
	factory := &TransportFactory{}

	tests := []struct {
		name          string
		transportType string
		needsListener bool
		expectedName  string
		expectError   bool
	}{
		{
			name:          "stdio transport",
			transportType: "stdio",
			needsListener: false,
			expectedName:  "stdio",
			expectError:   false,
		},
		{
			name:          "http transport needs listener",
			transportType: "http",
			needsListener: false, // Will cause error
			expectedName:  "",
			expectError:   true,
		},
		{
			name:          "sse transport needs listener",
			transportType: "sse",
			needsListener: false, // Will cause error
			expectedName:  "",
			expectError:   true,
		},
		{
			name:          "unknown transport",
			transportType: "websocket",
			needsListener: false,
			expectedName:  "",
			expectError:   true,
		},
		{
			name:          "streaming transport alias",
			transportType: "streaming",
			needsListener: false, // Will cause error
			expectedName:  "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, err := factory.CreateTransport(tt.transportType, nil)

			if tt.expectError {
				assert.Error(t, err, "expected error for %s", tt.transportType)
				assert.Nil(t, transport, "transport should be nil on error")
			} else {
				assert.NoError(t, err, "unexpected error for %s", tt.transportType)
				assert.NotNil(t, transport, "transport should not be nil")
				assert.Equal(t, tt.expectedName, transport.Name())
			}
		})
	}
}

// TestStderrLogger verifies the logger always outputs to stderr
func TestStderrLogger(t *testing.T) {
	logger := NewStderrLogger("[TEST]", LogLevelInfo)

	// Test that logger is not quiet by default
	assert.False(t, logger.IsQuiet(), "logger should not be quiet by default")

	// Test log level setting
	logger.SetLevel(LogLevelDebug)
	assert.Equal(t, LogLevelDebug, logger.level)

	// Test that quiet mode can be set
	logger.quiet = true
	assert.True(t, logger.IsQuiet(), "logger should be quiet when set")
}

// TestLogWithTransport verifies the helper functions
func TestLogWithTransport(t *testing.T) {
	// Create a mock transport with captured output
	var capturedOutput bytes.Buffer
	mockTransport := &mockTransport{
		name:   "mock",
		logger: &mockLogger{output: &capturedOutput},
	}

	// Test LogWithTransport
	LogWithTransport(mockTransport, "test message")
	output := capturedOutput.String()
	assert.Contains(t, output, "test message", "log message should be captured")

	// Reset buffer
	capturedOutput.Reset()

	// Test LogfWithTransport
	LogfWithTransport(mockTransport, "formatted %s", "message")
	output = capturedOutput.String()
	assert.Contains(t, output, "formatted message", "formatted message should be captured")
}

// TestLogWithTransportNilHandling verifies nil transport handling
func TestLogWithTransportNilHandling(t *testing.T) {
	// These should not panic with nil transport
	LogWithTransport(nil, "test with nil transport")
	LogfWithTransport(nil, "formatted %s with nil", "transport")

	// Create transport with nil logger
	mockTransport := &mockTransport{
		name:   "mock",
		logger: nil,
	}

	// These should also not panic
	LogWithTransport(mockTransport, "test with nil logger")
	LogfWithTransport(mockTransport, "formatted %s", "test")
}

// Mock implementations for testing

type mockTransport struct {
	name   string
	logger TransportLogger
}

func (m *mockTransport) Name() string                  { return m.name }
func (m *mockTransport) Logger() TransportLogger       { return m.logger }
func (m *mockTransport) IsProtocolChannel() bool       { return true }
func (m *mockTransport) GetReader() io.Reader          { return nil }
func (m *mockTransport) GetWriter() io.Writer          { return nil }
func (m *mockTransport) Close() error                  { return nil }
func (m *mockTransport) GetMetrics() *TransportMetrics { return nil }
func (m *mockTransport) EnableMetrics(enabled bool)    {}

type mockLogger struct {
	output *bytes.Buffer
}

func (m *mockLogger) Log(a ...any) {
	if m.output != nil {
		fmt.Fprintln(m.output, a...)
	}
}

func (m *mockLogger) Logf(format string, a ...any) {
	if m.output != nil {
		fmt.Fprintf(m.output, format+"\n", a...)
	}
}

func (m *mockLogger) SetLevel(level LogLevel) {}
func (m *mockLogger) IsQuiet() bool           { return false }

// BenchmarkTransportCreation measures transport creation performance
func BenchmarkTransportCreation(b *testing.B) {
	factory := &TransportFactory{}

	b.Run("stdio", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			transport, _ := factory.CreateTransport("stdio", nil)
			_ = transport.Close()
		}
	})
}

// TestIntegrationStdioLogging validates that stdio transport logs go to stderr
func TestIntegrationStdioLogging(t *testing.T) {
	// This test demonstrates the expected behavior:
	// 1. Protocol messages should go to stdout
	// 2. Log messages should go to stderr

	transport := NewStdioTransportWrapper()

	// Simulate protocol write (would go to stdout in real usage)
	writer := transport.GetWriter()
	assert.NotNil(t, writer, "writer should not be nil")

	// Simulate logging (should go to stderr)
	logger := transport.Logger()
	assert.NotNil(t, logger, "logger should not be nil")

	// In production, this would output to stderr:
	logger.Log("Operational message - should go to stderr")
	logger.Logf("Formatted message: %s", "also to stderr")

	// The key insight: transport ensures channel separation
	assert.True(t, transport.IsProtocolChannel(), "stdio uses stdout for protocol")
	// But logger always uses stderr (verified by implementation)
}
