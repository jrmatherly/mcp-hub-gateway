package gateway

import (
	"fmt"
	"os"
	"strings"
)

// quietMode can be set to true to suppress log output for specific scenarios
// NOTE: This is NOT used for stdio mode - logs always go to stderr which doesn't interfere with JSON-RPC on stdout
var quietMode bool

// globalTransport holds the current transport for logging
// This is set during gateway initialization
var globalTransport MCPTransport

// SetGlobalTransport sets the global transport for logging
func SetGlobalTransport(t MCPTransport) {
	globalTransport = t
}

// log logs messages using the transport abstraction if available, otherwise falls back to stderr
func log(a ...any) {
	if quietMode {
		return
	}

	// Use transport logger if available
	if globalTransport != nil && globalTransport.Logger() != nil {
		globalTransport.Logger().Log(a...)
		return
	}

	// Fallback to stderr for backwards compatibility
	_, _ = fmt.Fprintln(os.Stderr, a...)
}

// logf logs formatted messages using the transport abstraction if available, otherwise falls back to stderr
func logf(format string, a ...any) {
	if quietMode {
		return
	}

	// Ensure format has newline
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}

	// Use transport logger if available
	if globalTransport != nil && globalTransport.Logger() != nil {
		globalTransport.Logger().Logf(format, a...)
		return
	}

	// Fallback to stderr for backwards compatibility
	_, _ = fmt.Fprintf(os.Stderr, format, a...)
}
