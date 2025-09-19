# Integration Test Troubleshooting Summary

## Issues Identified and Fixed

### 1. UserConfig Migration Direction Error ✅ FIXED

**Problem**: The `RunMigrationsSimple()` function didn't set the `Direction` field in `RunOptions`, causing "invalid migration direction: " error.

**Solution**: Added `Direction: DirectionUp` to the options struct in `/cmd/docker-mcp/portal/database/runner.go:340`

### 2. Elicit Container Cleanup Issue ✅ FIXED

**Problem**: The elicit container wasn't being cleaned up because the test panicked when trying to access an uninitialized client in the cleanup function.

**Solution**: Modified `/cmd/docker-mcp/long_lived_integration_test.go` to register cleanup only after successful initialization and added nil checks.

### 3. JSON-RPC Protocol Issue 🔄 IN PROGRESS

**Problem**: Tests are failing with "invalid character ' ' in numeric literal" during MCP client initialization. Further investigation shows this is actually a JSON-RPC version mismatch.

**Root Cause**: The MCP SDK v0.5.0 expects proper JSON-RPC 2.0 formatted messages with `"jsonrpc":"2.0"` field, but the test client or gateway may not be sending/receiving them correctly.

**Evidence**:

- Direct gateway invocation works with proper JSON-RPC 2.0 messages
- Tests using the MCP client library are failing during initialization
- Error changes to "invalid message version tag" when using stdio directly

## Remaining Work

### Container Cleanup

The elicit container issue appears to be related to the test framework not properly cleaning up when tests fail. The fix applied should help, but we need to ensure:

1. Containers are tagged properly for cleanup
2. Test framework cleanup handlers are registered correctly
3. Docker labels are used to identify test containers

### JSON-RPC Protocol Fix

Need to investigate:

1. How the MCP SDK v0.5.0 formats initialization messages
2. Whether there's a version mismatch between the gateway and SDK
3. If the stdio transport is properly handling JSON-RPC 2.0 format

## Commands for Testing

```bash
# Test userconfig migration (should work now)
go test -timeout 30s -v -count=1 ./cmd/docker-mcp/portal/userconfig -run 'TestIntegrationTestSuite'

# Test simple integration (works)
go test -timeout 10s -v -count=1 ./cmd/docker-mcp -run 'TestIntegrationVersion'

# Test that still fails (JSON-RPC issue)
go test -timeout 30s -v -count=1 ./cmd/docker-mcp -run 'TestIntegrationCallToolClickhouse'

# Clean up hanging containers
docker ps -a | grep elicit | awk '{print $1}' | xargs docker rm -f
```

## Next Steps

1. Check if the MCP SDK v0.5.0 has breaking changes in the wire protocol
2. Verify the gateway is using the correct JSON-RPC version
3. Update test initialization code to match new SDK requirements
4. Run full integration test suite to verify all fixes
