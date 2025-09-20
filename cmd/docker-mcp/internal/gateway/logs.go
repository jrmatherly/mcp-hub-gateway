package gateway

import (
	"fmt"
	"os"
	"strings"
)

// quietMode can be set to true to suppress log output for specific scenarios
// NOTE: This is NOT used for stdio mode - logs always go to stderr which doesn't interfere with JSON-RPC on stdout
var quietMode bool

func log(a ...any) {
	if !quietMode {
		_, _ = fmt.Fprintln(os.Stderr, a...)
	}
}

func logf(format string, a ...any) {
	if !quietMode {
		if !strings.HasSuffix(format, "\n") {
			format += "\n"
		}
		_, _ = fmt.Fprintf(os.Stderr, format, a...)
	}
}
