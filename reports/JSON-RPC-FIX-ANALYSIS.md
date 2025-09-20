# JSON-RPC Integration Test Fix Analysis

## Root Cause Analysis

The integration tests are failing with "invalid character ' ' in numeric literal" during MCP client initialization. After comprehensive analysis, the issue is:

**The gateway outputs status messages to stdout that interfere with JSON-RPC protocol communication.**

## Problem Details

1. When the gateway runs in stdio mode (used by tests), it outputs non-JSON messages to stdout:

   - "- Those servers are enabled: time"
   - "Successfully reloaded configuration with 7 tools..."

2. The MCP client expects pure JSON-RPC messages on stdout, but receives these text messages first

3. The JSON parser tries to parse these text messages as JSON and fails with "invalid character ' ' in numeric literal"

## Evidence

Running the gateway with JSON-RPC input shows stdout pollution:

```bash
$ echo '{"jsonrpc":"2.0",...}' | docker mcp gateway run --servers=time 2>/dev/null
- Those servers are enabled: time
Successfully reloaded configuration with 7 tools, 0 prompts, 0 resources, 0 resource templates
```

These messages should go to stderr, not stdout.

## The Fix

The issue is that while most log messages use the `log()` function which outputs to stderr, some messages are somehow ending up on stdout. This appears to be happening during the reload configuration phase.

### Investigation Results

1. The `log()` function in `internal/gateway/logs.go` correctly outputs to stderr
2. Both problematic messages use the `log()` function
3. Yet they still appear on stdout when running the gateway

This suggests there might be output redirection happening somewhere in the code flow or these messages are being captured and re-output.

## Solution

We need to ensure that when running in stdio mode, NO non-JSON messages go to stdout. Options:

1. **Suppress all logs in stdio mode** - When Transport == "stdio", disable all log output
2. **Add a quiet mode** - Add a --quiet flag that suppresses non-error output
3. **Fix the output redirection** - Find where stdout is being incorrectly used

The cleanest solution is option 1 - suppress logs when in stdio mode since the protocol requires clean stdout.
