package gateway

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTransportLogIntegration verifies that logging uses transport abstraction
func TestTransportLogIntegration(t *testing.T) {
	// Create a mock transport with captured stderr
	var stderrBuffer bytes.Buffer
	mockTransport := &mockIntegrationTransport{
		name:         "test",
		stderrOutput: &stderrBuffer,
	}

	// Set as global transport
	SetGlobalTransport(mockTransport)
	defer func() {
		SetGlobalTransport(nil) // Reset after test
	}()

	// Test log function
	log("Test message")
	assert.Contains(t, stderrBuffer.String(), "Test message")

	// Clear buffer
	stderrBuffer.Reset()

	// Test logf function
	logf("Formatted %s", "message")
	assert.Contains(t, stderrBuffer.String(), "Formatted message")
}

// TestTransportChannelSeparation verifies stdio channel separation
func TestTransportChannelSeparation(t *testing.T) {
	// Create buffers for stdout and stderr
	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer

	// Create a mock stdio transport that captures both channels
	transport := &mockStdioTransport{
		stdoutBuffer: &stdoutBuffer,
		stderrBuffer: &stderrBuffer,
	}

	// Set as global transport
	SetGlobalTransport(transport)
	defer func() {
		SetGlobalTransport(nil)
	}()

	// Protocol message goes to stdout
	transport.GetWriter().Write([]byte(`{"jsonrpc":"2.0","method":"test"}`))

	// Log message should go to stderr via transport
	log("Operational log message")

	// Verify channel separation
	assert.Contains(t, stdoutBuffer.String(), "jsonrpc", "Protocol should be in stdout")
	assert.NotContains(t, stdoutBuffer.String(), "Operational", "Logs should not be in stdout")

	assert.Contains(t, stderrBuffer.String(), "Operational", "Logs should be in stderr")
	assert.NotContains(t, stderrBuffer.String(), "jsonrpc", "Protocol should not be in stderr")
}

// Mock stdio transport for testing channel separation
type mockStdioTransport struct {
	stdoutBuffer *bytes.Buffer
	stderrBuffer *bytes.Buffer
}

func (m *mockStdioTransport) Name() string { return "stdio" }
func (m *mockStdioTransport) Logger() TransportLogger {
	return &testStderrLogger{output: m.stderrBuffer}
}
func (m *mockStdioTransport) IsProtocolChannel() bool { return true }
func (m *mockStdioTransport) GetReader() io.Reader {
	return strings.NewReader("")
}

func (m *mockStdioTransport) GetWriter() io.Writer {
	return m.stdoutBuffer
}
func (m *mockStdioTransport) Close() error                  { return nil }
func (m *mockStdioTransport) GetMetrics() *TransportMetrics { return nil }
func (m *mockStdioTransport) EnableMetrics(enabled bool)    {}

// Mock transport for integration testing
type mockIntegrationTransport struct {
	name         string
	stderrOutput *bytes.Buffer
}

func (m *mockIntegrationTransport) Name() string { return m.name }
func (m *mockIntegrationTransport) Logger() TransportLogger {
	return &testStderrLogger{output: m.stderrOutput}
}
func (m *mockIntegrationTransport) IsProtocolChannel() bool       { return false }
func (m *mockIntegrationTransport) GetReader() io.Reader          { return nil }
func (m *mockIntegrationTransport) GetWriter() io.Writer          { return nil }
func (m *mockIntegrationTransport) Close() error                  { return nil }
func (m *mockIntegrationTransport) GetMetrics() *TransportMetrics { return nil }
func (m *mockIntegrationTransport) EnableMetrics(enabled bool)    {}

// Test logger that writes to a buffer
type testStderrLogger struct {
	output *bytes.Buffer
}

func (l *testStderrLogger) Log(a ...any) {
	fmt.Fprintln(l.output, a...)
}

func (l *testStderrLogger) Logf(format string, a ...any) {
	fmt.Fprintf(l.output, format+"\n", a...)
}

func (l *testStderrLogger) SetLevel(level LogLevel) {}
func (l *testStderrLogger) IsQuiet() bool           { return false }
