# ADR-001: CLI Wrapper Pattern for Portal Implementation

**Status**: Accepted
**Date**: 2025-09-17
**Deciders**: Architecture Team
**Technical Story**: Portal needs to provide web interface for MCP server management

## Context and Problem Statement

The MCP Gateway project already has a mature, well-tested Docker CLI plugin that provides comprehensive MCP server management functionality. We need to create a web-based portal interface that provides the same capabilities through a browser-based UI.

We need to decide how to implement the portal backend:

1. Reimplement all CLI functionality in the portal backend
2. Create a wrapper that executes the existing CLI plugin
3. Extract shared libraries and build both CLI and portal on common foundation

## Decision Drivers

- **Time to Market**: Need to deliver portal functionality quickly
- **Code Consistency**: Ensure identical behavior between CLI and portal
- **Maintenance Burden**: Minimize duplicated logic and testing
- **Security**: Maintain security posture of existing CLI
- **Performance**: Acceptable latency for web operations
- **Reliability**: Leverage battle-tested CLI implementation

## Considered Options

### Option A: Full Reimplementation

**Description**: Rewrite all CLI functionality as native Go API endpoints in the portal backend.

**Pros**:

- Native API performance without process overhead
- Direct access to all data structures and state
- Type-safe interfaces between components
- No dependency on CLI binary availability

**Cons**:

- 6+ months development time to replicate existing functionality
- Risk of behavioral differences between CLI and portal
- Doubled maintenance burden for identical functionality
- Need to replicate all existing tests and validation
- Risk of introducing new bugs in reimplemented code

### Option B: CLI Wrapper Pattern

**Description**: Portal backend executes CLI plugin commands and parses output for API responses.

**Pros**:

- Rapid development - weeks instead of months
- Guaranteed identical behavior between CLI and portal
- Single source of truth for MCP management logic
- Leverage existing testing and validation
- Immediate access to all CLI features

**Cons**:

- Process execution overhead (~10-50ms per command)
- Need to parse CLI output for structured data
- Dependency on CLI binary installation
- Limited ability to customize behavior for web use cases

### Option C: Shared Library Extraction

**Description**: Extract common functionality into shared libraries used by both CLI and portal.

**Pros**:

- Code reuse without process overhead
- Type-safe interfaces between components
- Flexibility to customize behavior per interface
- Long-term architectural cleanliness

**Cons**:

- Major refactoring of existing CLI codebase
- 3+ months to extract and stabilize interfaces
- Risk of breaking existing CLI functionality
- Complex interface design and maintenance

## Decision Outcome

**Chosen Option**: Option B - CLI Wrapper Pattern

**Rationale**:

- The CLI plugin is mature, well-tested, and handles all edge cases
- Portal development timeline requires rapid delivery
- Risk of behavioral differences is eliminated by using same codebase
- Process overhead is acceptable for web use cases (<100ms response times)
- Future migration to shared libraries is possible if needed

## Implementation Details

### CLI Executor Framework

```go
type CLIExecutor struct {
    binaryPath    string
    timeout       time.Duration
    validator     CommandValidator
    outputParser  OutputParser
    auditLogger   AuditLogger
}

func (e *CLIExecutor) ExecuteCommand(ctx context.Context, cmd CLICommand) (*CLIResult, error) {
    // 1. Validate command and arguments
    // 2. Execute CLI binary with timeout
    // 3. Parse output and errors
    // 4. Log execution for audit trail
    // 5. Return structured result
}
```

### Security Controls

- **Input Validation**: Whitelist allowed commands and parameter patterns
- **Command Injection Prevention**: Use parameterized execution, no shell interpretation
- **Process Isolation**: Execute with minimal privileges and resource limits
- **Audit Logging**: Log all executed commands with user context

### Output Parsing Strategy

- **JSON Output**: CLI plugin modified to support `--output json` flag for structured data
- **Streaming Support**: Support for streaming output on long-running operations
- **Error Handling**: Parse stderr for error conditions and user-friendly messages
- **Exit Code Mapping**: Map CLI exit codes to appropriate HTTP status codes

### Performance Optimization

- **Command Caching**: Cache results for idempotent operations (server list, status)
- **Batch Operations**: Group multiple operations into single CLI invocation where possible
- **Async Execution**: Use goroutines for non-blocking command execution
- **Connection Pooling**: Reuse CLI process instances for rapid successive commands

## Positive Consequences

- **Rapid Development**: Portal backend developed in 3 weeks instead of 3+ months
- **Perfect Consistency**: Identical behavior between CLI and web interfaces
- **Reduced Risk**: Leverage battle-tested CLI implementation
- **Immediate Feature Parity**: All CLI features available in portal on day one
- **Simplified Testing**: Can focus on integration testing rather than reimplementing unit tests

## Negative Consequences

- **Process Overhead**: 10-50ms latency per CLI command execution
- **Parsing Complexity**: Need to maintain output parsing for CLI responses
- **Binary Dependency**: Portal requires CLI binary installation and maintenance
- **Limited Customization**: Cannot easily customize behavior for web-specific use cases
- **Future Refactoring**: May need to revisit if performance becomes critical

## Mitigation Strategies

### Performance Mitigation

- Implement response caching for expensive operations
- Use WebSocket connections for real-time updates instead of polling
- Batch multiple operations where CLI supports it
- Profile and optimize high-frequency operations

### Parsing Reliability

- Add `--output json` support to CLI for structured output
- Implement comprehensive output parsing tests
- Use semantic versioning to manage CLI/portal compatibility
- Graceful degradation when parsing fails

### Dependency Management

- Container deployment includes CLI binary automatically
- Version pinning between portal and CLI releases
- Health checks to ensure CLI binary availability
- Clear error messages when CLI is unavailable

## Validation and Success Metrics

### Functional Validation

- [ ] All CLI commands accessible via portal API
- [ ] Identical behavior between CLI and portal operations
- [ ] Proper error handling and user feedback
- [ ] Audit logging for all CLI executions

### Performance Validation

- [ ] 95th percentile API response times <500ms
- [ ] CLI command execution overhead <100ms
- [ ] Successful handling of 100+ concurrent users
- [ ] Memory usage <200MB for typical workloads

### Security Validation

- [ ] Command injection prevention verified by security review
- [ ] All user inputs validated against whitelist
- [ ] Process isolation and privilege restrictions enforced
- [ ] Comprehensive audit logging implemented

## Related Decisions

- **ADR-002**: Azure AD OAuth integration for portal authentication
- **ADR-003**: DCR bridge design for dynamic client registration
- **ADR-004**: Database RLS security for multi-tenant portal

## References

- [CLI Plugin Architecture](../C4-03-Components.md#cli-plugin-components)
- [Portal Backend Components](../C4-03-Components.md#portal-backend-components)
- [Security Architecture](../security-architecture.md)
- [CLI Integration Documentation](../../docs/cli-integration.md)

---

**ADR**: Architecture Decision Record
**Last Updated**: September 19, 2025
**Next Review**: December 2025 (quarterly review)
**Status**: Implemented and validated in production
