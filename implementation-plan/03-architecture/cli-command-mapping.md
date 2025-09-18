# CLI Command Mapping

## Overview

This document defines the complete mapping between web portal UI actions and underlying CLI commands, including input validation, output parsing strategies, and error handling for each operation.

## Command Categories

### Server Management Commands

#### List Servers

**UI Action**: Dashboard server list, search and filtering
**CLI Command**: `docker mcp server list`

```yaml
mapping:
  endpoint: GET /api/v1/servers
  cli_command: docker mcp server list --format json
  parameters:
    catalog_type:
      cli_arg: --catalog
      validation: enum[predefined,custom,all]
      default: all
    category:
      cli_arg: --category
      validation: string, max_length=50
    search:
      cli_arg: --search
      validation: string, max_length=100
  output_format: json
  parser_type: JSONParser
  timeout: 5s
```

**Example CLI Output**:

```json
{
  "servers": [
    {
      "id": "github",
      "name": "github",
      "display_name": "GitHub",
      "image": "docker/mcp-github:latest",
      "catalog_type": "predefined",
      "category": "development",
      "status": "available"
    }
  ]
}
```

#### Get Server Details

**UI Action**: Server detail view, configuration panel
**CLI Command**: `docker mcp server inspect`

```yaml
mapping:
  endpoint: GET /api/v1/servers/{id}
  cli_command: docker mcp server inspect {server_id} --format json
  parameters:
    server_id:
      cli_arg: positional[0]
      validation: regex=^[a-zA-Z0-9\-_]+$
      required: true
  output_format: json
  parser_type: JSONParser
  timeout: 3s
```

#### Enable Server

**UI Action**: Enable toggle, bulk enable
**CLI Command**: `docker mcp server enable`

```yaml
mapping:
  endpoint: POST /api/v1/servers/{id}/enable
  cli_command: docker mcp server enable {server_id} --user {user_id}
  parameters:
    server_id:
      cli_arg: positional[0]
      validation: regex=^[a-zA-Z0-9\-_]+$
      required: true
    user_id:
      cli_arg: --user
      validation: uuid
      source: jwt_token
    config:
      cli_arg: --config-file
      validation: json_object
      transform: write_temp_file
    auto_start:
      cli_arg: --auto-start
      validation: boolean
      default: true
  output_format: json
  parser_type: JSONParser
  timeout: 30s
  async: true
```

**Example CLI Output**:

```json
{
  "success": true,
  "server_id": "github",
  "container_id": "abc123def456",
  "status": "enabling",
  "estimated_time": 15
}
```

#### Disable Server

**UI Action**: Disable toggle, bulk disable
**CLI Command**: `docker mcp server disable`

```yaml
mapping:
  endpoint: POST /api/v1/servers/{id}/disable
  cli_command: docker mcp server disable {server_id} --user {user_id}
  parameters:
    server_id:
      cli_arg: positional[0]
      validation: regex=^[a-zA-Z0-9\-_]+$
      required: true
    user_id:
      cli_arg: --user
      validation: uuid
      source: jwt_token
    force:
      cli_arg: --force
      validation: boolean
      default: false
  output_format: json
  parser_type: JSONParser
  timeout: 15s
  async: true
```

#### Restart Server

**UI Action**: Restart button
**CLI Command**: `docker mcp server restart`

```yaml
mapping:
  endpoint: POST /api/v1/servers/{id}/restart
  cli_command: docker mcp server restart {server_id} --user {user_id}
  parameters:
    server_id:
      cli_arg: positional[0]
      validation: regex=^[a-zA-Z0-9\-_]+$
      required: true
    user_id:
      cli_arg: --user
      validation: uuid
      source: jwt_token
  output_format: json
  parser_type: JSONParser
  timeout: 30s
  async: true
```

### Log Management Commands

#### Get Server Logs

**UI Action**: Log viewer, download logs
**CLI Command**: `docker mcp server logs`

```yaml
mapping:
  endpoint: GET /api/v1/servers/{id}/logs
  cli_command: docker mcp server logs {server_id} --user {user_id}
  parameters:
    server_id:
      cli_arg: positional[0]
      validation: regex=^[a-zA-Z0-9\-_]+$
      required: true
    user_id:
      cli_arg: --user
      validation: uuid
      source: jwt_token
    lines:
      cli_arg: --lines
      validation: int, min=1, max=10000
      default: 100
    since:
      cli_arg: --since
      validation: rfc3339_datetime
    follow:
      cli_arg: --follow
      validation: boolean
      default: false
  output_format: text
  parser_type: LogParser
  timeout: 60s
  streaming: true
```

**Example CLI Output**:

```text
2024-01-15T10:00:00Z INFO  Server started successfully
2024-01-15T10:00:01Z DEBUG Connected to MCP protocol
2024-01-15T10:00:02Z INFO  Ready to accept requests
```

### Bulk Operations

#### Bulk Server Operations

**UI Action**: Multi-select operations
**CLI Command**: `docker mcp server bulk`

```yaml
mapping:
  endpoint: POST /api/v1/servers/bulk
  cli_command: docker mcp server bulk {operation}
  parameters:
    operation:
      cli_arg: positional[0]
      validation: enum[enable,disable,restart]
      required: true
    server_ids:
      cli_arg: --servers
      validation: array[string], max_items=50
      required: true
    user_id:
      cli_arg: --user
      validation: uuid
      source: jwt_token
    config:
      cli_arg: --config-file
      validation: json_object
      transform: write_temp_file
  output_format: json
  parser_type: JSONParser
  timeout: 300s
  async: true
  streaming: true
```

**Example CLI Output**:

```json
{
  "operation_id": "bulk_123456",
  "total": 5,
  "completed": 0,
  "failed": 0,
  "status": "starting",
  "results": []
}
```

### Configuration Management

#### Get User Configuration

**UI Action**: Settings page, export configuration
**CLI Command**: `docker mcp config get`

```yaml
mapping:
  endpoint: GET /api/v1/config
  cli_command: docker mcp config get --user {user_id} --format json
  parameters:
    user_id:
      cli_arg: --user
      validation: uuid
      source: jwt_token
  output_format: json
  parser_type: JSONParser
  timeout: 5s
```

#### Update User Configuration

**UI Action**: Settings update, import configuration
**CLI Command**: `docker mcp config set`

```yaml
mapping:
  endpoint: PUT /api/v1/config
  cli_command: docker mcp config set --user {user_id} --config-file {config_file}
  parameters:
    user_id:
      cli_arg: --user
      validation: uuid
      source: jwt_token
    config:
      cli_arg: --config-file
      validation: json_object
      transform: write_temp_file
      required: true
  output_format: json
  parser_type: JSONParser
  timeout: 10s
```

#### Export Configuration

**UI Action**: Export button, backup creation
**CLI Command**: `docker mcp config export`

```yaml
mapping:
  endpoint: GET /api/v1/config/export
  cli_command: docker mcp config export --user {user_id} --format {format}
  parameters:
    user_id:
      cli_arg: --user
      validation: uuid
      source: jwt_token
    format:
      cli_arg: --format
      validation: enum[json,yaml]
      default: json
  output_format: text
  parser_type: RawParser
  timeout: 10s
```

### Custom Server Management

#### Add Custom Server

**UI Action**: Add custom server form
**CLI Command**: `docker mcp catalog add`

```yaml
mapping:
  endpoint: POST /api/v1/catalog/custom
  cli_command: docker mcp catalog add --user {user_id} --definition-file {definition_file}
  parameters:
    user_id:
      cli_arg: --user
      validation: uuid
      source: jwt_token
    definition:
      cli_arg: --definition-file
      validation: json_object
      transform: write_temp_file
      required: true
  output_format: json
  parser_type: JSONParser
  timeout: 15s
```

#### Update Custom Server

**UI Action**: Edit custom server
**CLI Command**: `docker mcp catalog update`

```yaml
mapping:
  endpoint: PUT /api/v1/catalog/custom/{id}
  cli_command: docker mcp catalog update {server_id} --user {user_id} --definition-file {definition_file}
  parameters:
    server_id:
      cli_arg: positional[0]
      validation: regex=^[a-zA-Z0-9\-_]+$
      required: true
    user_id:
      cli_arg: --user
      validation: uuid
      source: jwt_token
    definition:
      cli_arg: --definition-file
      validation: json_object
      transform: write_temp_file
      required: true
  output_format: json
  parser_type: JSONParser
  timeout: 15s
```

#### Remove Custom Server

**UI Action**: Delete custom server
**CLI Command**: `docker mcp catalog remove`

```yaml
mapping:
  endpoint: DELETE /api/v1/catalog/custom/{id}
  cli_command: docker mcp catalog remove {server_id} --user {user_id}
  parameters:
    server_id:
      cli_arg: positional[0]
      validation: regex=^[a-zA-Z0-9\-_]+$
      required: true
    user_id:
      cli_arg: --user
      validation: uuid
      source: jwt_token
  output_format: json
  parser_type: JSONParser
  timeout: 10s
```

## Input Validation Framework

### Validation Types

```go
type ValidationType string

const (
    ValidationString    ValidationType = "string"
    ValidationInt       ValidationType = "int"
    ValidationFloat     ValidationType = "float"
    ValidationBoolean   ValidationType = "boolean"
    ValidationUUID      ValidationType = "uuid"
    ValidationEnum      ValidationType = "enum"
    ValidationRegex     ValidationType = "regex"
    ValidationJSON      ValidationType = "json_object"
    ValidationArray     ValidationType = "array"
    ValidationDateTime  ValidationType = "rfc3339_datetime"
)

type ValidationRule struct {
    Type        ValidationType  `yaml:"validation"`
    Required    bool           `yaml:"required"`
    MinLength   *int          `yaml:"min_length,omitempty"`
    MaxLength   *int          `yaml:"max_length,omitempty"`
    Min         *float64      `yaml:"min,omitempty"`
    Max         *float64      `yaml:"max,omitempty"`
    Pattern     *string       `yaml:"pattern,omitempty"`
    Enum        []string      `yaml:"enum,omitempty"`
    MaxItems    *int          `yaml:"max_items,omitempty"`
}
```

### Parameter Sources

```go
type ParameterSource string

const (
    SourceRequestBody   ParameterSource = "request_body"
    SourceQueryParam    ParameterSource = "query_param"
    SourcePathParam     ParameterSource = "path_param"
    SourceJWTToken     ParameterSource = "jwt_token"
    SourceHeader       ParameterSource = "header"
    SourceEnvironment  ParameterSource = "environment"
)
```

## Output Parser Specifications

### JSON Parser

Handles structured JSON output from CLI commands.

```go
type JSONParser struct{}

func (jp *JSONParser) Parse(output []byte, cmd *Command) (*ParsedResult, error) {
    var data interface{}
    if err := json.Unmarshal(output, &data); err != nil {
        return nil, fmt.Errorf("invalid JSON output: %w", err)
    }

    return &ParsedResult{
        Success: true,
        Data:    data,
    }, nil
}
```

### Table Parser

Parses tabular text output into structured data.

```go
type TableParser struct {
    HeaderLine    int
    Separator     string
    TrimWhitespace bool
}

func (tp *TableParser) Parse(output []byte, cmd *Command) (*ParsedResult, error) {
    lines := strings.Split(string(output), "\n")

    if len(lines) < tp.HeaderLine+1 {
        return nil, fmt.Errorf("insufficient lines for table parsing")
    }

    headers := strings.Split(lines[tp.HeaderLine], tp.Separator)
    rows := make([]map[string]string, 0)

    for i := tp.HeaderLine + 1; i < len(lines); i++ {
        if strings.TrimSpace(lines[i]) == "" {
            continue
        }

        columns := strings.Split(lines[i], tp.Separator)
        row := make(map[string]string)

        for j, col := range columns {
            if j < len(headers) {
                key := strings.TrimSpace(headers[j])
                value := strings.TrimSpace(col)
                row[key] = value
            }
        }
        rows = append(rows, row)
    }

    return &ParsedResult{
        Success: true,
        Data:    rows,
    }, nil
}
```

### Log Parser

Handles streaming log output with timestamp and level parsing.

```go
type LogParser struct {
    TimestampRegex *regexp.Regexp
    LevelRegex     *regexp.Regexp
}

func (lp *LogParser) ParseStream(reader io.Reader, callback StreamCallback) error {
    scanner := bufio.NewScanner(reader)

    for scanner.Scan() {
        line := scanner.Text()

        logEntry := &LogEntry{
            Raw:       line,
            Timestamp: lp.extractTimestamp(line),
            Level:     lp.extractLevel(line),
            Message:   lp.extractMessage(line),
        }

        if err := callback(logEntry); err != nil {
            return err
        }
    }

    return scanner.Err()
}
```

## Error Code Mapping

### CLI Exit Codes to HTTP Status Codes

```go
var exitCodeMapping = map[int]HTTPError{
    0: {Status: 200, Code: "SUCCESS"},
    1: {Status: 500, Code: "GENERAL_ERROR"},
    2: {Status: 400, Code: "INVALID_USAGE"},
    3: {Status: 404, Code: "RESOURCE_NOT_FOUND"},
    4: {Status: 403, Code: "ACCESS_DENIED"},
    5: {Status: 409, Code: "RESOURCE_CONFLICT"},
    6: {Status: 422, Code: "VALIDATION_ERROR"},
    7: {Status: 503, Code: "SERVICE_UNAVAILABLE"},
    8: {Status: 408, Code: "TIMEOUT"},
    9: {Status: 429, Code: "RATE_LIMITED"},
}
```

### Error Message Parsing

```go
type CLIErrorParser struct {
    patterns map[string]*regexp.Regexp
}

func (cep *CLIErrorParser) ParseError(output string, exitCode int) *ParsedError {
    // Extract structured error information from CLI output
    for errorType, pattern := range cep.patterns {
        if matches := pattern.FindStringSubmatch(output); len(matches) > 1 {
            return &ParsedError{
                Type:    errorType,
                Message: matches[1],
                Details: cep.extractDetails(matches),
            }
        }
    }

    return &ParsedError{
        Type:    "UNKNOWN_ERROR",
        Message: output,
        Details: map[string]interface{}{"exit_code": exitCode},
    }
}
```

## Streaming Implementation

### WebSocket Event Mapping

```yaml
cli_events:
  - cli_output: "Server enabling..."
    ws_event:
      type: "SERVER_STATE_CHANGE"
      data:
        server_id: "{server_id}"
        status: "enabling"
        progress: 25

  - cli_output: "Container started: {container_id}"
    ws_event:
      type: "SERVER_STATE_CHANGE"
      data:
        server_id: "{server_id}"
        status: "running"
        container_id: "{container_id}"
        progress: 100

  - cli_output: "Bulk operation progress: {completed}/{total}"
    ws_event:
      type: "BULK_OPERATION_PROGRESS"
      data:
        operation_id: "{operation_id}"
        completed: "{completed}"
        total: "{total}"
        progress: "{percentage}"
```

### Stream Processing Pipeline

```go
type StreamProcessor struct {
    parsers   map[string]StreamParser
    filters   []StreamFilter
    enhancers []StreamEnhancer
}

func (sp *StreamProcessor) ProcessLine(line string, context *StreamContext) (*StreamEvent, error) {
    // 1. Parse line content
    parsed, err := sp.parseLine(line, context)
    if err != nil {
        return nil, err
    }

    // 2. Apply filters
    if !sp.shouldProcess(parsed, context) {
        return nil, nil
    }

    // 3. Enhance with additional data
    enhanced := sp.enhanceEvent(parsed, context)

    // 4. Convert to WebSocket event
    return sp.toWebSocketEvent(enhanced), nil
}
```

## Testing and Validation

### Command Mapping Tests

```go
func TestServerListMapping(t *testing.T) {
    mapping := getCommandMapping("GET", "/api/v1/servers")

    assert.Equal(t, "docker mcp server list --format json", mapping.CLICommand)
    assert.Equal(t, "JSONParser", mapping.ParserType)
    assert.Equal(t, 5*time.Second, mapping.Timeout)
}

func TestParameterValidation(t *testing.T) {
    validator := NewParameterValidator()

    // Valid UUID parameter
    err := validator.Validate("user_id", "550e8400-e29b-41d4-a716-446655440000", ValidationUUID)
    assert.NoError(t, err)

    // Invalid enum parameter
    err = validator.Validate("format", "invalid", ValidationEnum, []string{"json", "yaml"})
    assert.Error(t, err)
}
```

### Integration Tests

```go
func TestRealCLIMapping(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    bridge := setupRealCLIBridge()

    // Test server list command
    result, err := bridge.ExecuteMapping("GET", "/api/v1/servers", map[string]interface{}{
        "user_id": testUserID,
    })

    require.NoError(t, err)
    assert.True(t, result.Success)
    assert.IsType(t, []interface{}{}, result.Data)
}
```

This command mapping ensures secure, validated, and well-tested integration between the web portal and CLI while maintaining full functionality and adding comprehensive error handling and streaming capabilities.

## Advanced Command Mapping Features

### Dynamic Parameter Injection

```go
type ParameterInjector struct {
    contextExtractors map[string]ContextExtractor
    sanitizers        map[string]SanitizerFunc
    validators        map[string]ValidatorFunc
}

func (pi *ParameterInjector) InjectParameters(cmd *Command, context *RequestContext) error {
    for paramName, spec := range cmd.Mapping.Parameters {
        value, err := pi.extractParameter(paramName, spec, context)
        if err != nil {
            return fmt.Errorf("failed to extract parameter %s: %w", paramName, err)
        }

        if spec.Sanitizer != nil {
            value = spec.Sanitizer(value)
        }

        if err := pi.validateParameter(paramName, value, spec); err != nil {
            return fmt.Errorf("parameter validation failed for %s: %w", paramName, err)
        }

        cmd.Parameters[paramName] = value
    }

    return nil
}
```

### Temporary File Management

```go
type TempFileManager struct {
    baseDir     string
    cleanup     *time.Ticker
    maxAge      time.Duration
    maxSize     int64
    activeFiles sync.Map
}

func (tfm *TempFileManager) CreateTempFile(content interface{}, format string) (string, error) {
    filename := fmt.Sprintf("mcp_temp_%s_%s.%s",
        uuid.New().String()[:8],
        time.Now().Format("20060102_150405"),
        format)

    filepath := path.Join(tfm.baseDir, filename)

    var data []byte
    switch format {
    case "json":
        var err error
        data, err = json.MarshalIndent(content, "", "  ")
        if err != nil {
            return "", err
        }
    case "yaml":
        var err error
        data, err = yaml.Marshal(content)
        if err != nil {
            return "", err
        }
    default:
        data = []byte(fmt.Sprintf("%v", content))
    }

    if err := os.WriteFile(filepath, data, 0600); err != nil {
        return "", err
    }

    tfm.trackFile(filename)
    return filepath, nil
}

func (tfm *TempFileManager) CleanupExpiredFiles() {
    cutoff := time.Now().Add(-tfm.maxAge)

    tfm.activeFiles.Range(func(key, value interface{}) bool {
        filename := key.(string)
        info := value.(*FileInfo)

        if info.CreatedAt.Before(cutoff) {
            filepath := path.Join(tfm.baseDir, filename)
            os.Remove(filepath)
            tfm.activeFiles.Delete(filename)
        }

        return true
    })
}
```

### Command Template System

```yaml
# Command templates for complex operations
command_templates:
  server_enable_with_config:
    template: "docker mcp server enable {{.server_id}} --user {{.user_id}} {{if .config_file}}--config-file {{.config_file}}{{end}} {{if .auto_start}}--auto-start{{end}}"
    parameters:
      - name: server_id
        type: string
        required: true
        validation: "^[a-zA-Z0-9\\-_]+$"
      - name: user_id
        type: uuid
        required: true
        source: jwt_token
      - name: config_file
        type: file
        required: false
        transform: temp_file_json
      - name: auto_start
        type: boolean
        default: true

  bulk_server_operation:
    template: 'docker mcp server bulk {{.operation}} --servers {{join .server_ids ","}} --user {{.user_id}} {{if .config_file}}--config-file {{.config_file}}{{end}}'
    parameters:
      - name: operation
        type: enum
        values: [enable, disable, restart]
        required: true
      - name: server_ids
        type: array
        item_type: string
        max_items: 50
        required: true
      - name: user_id
        type: uuid
        source: jwt_token
        required: true
      - name: config_file
        type: file
        transform: temp_file_json
```

### Real-time Progress Tracking

```go
type ProgressTracker struct {
    operations sync.Map // map[operationID]*OperationProgress
    subscribers sync.Map // map[operationID][]chan ProgressEvent
    eventBus   *EventBus
}

type OperationProgress struct {
    ID          string                 `json:"id"`
    Type        string                 `json:"type"`
    Status      OperationStatus       `json:"status"`
    Progress    float64               `json:"progress"`
    Message     string                `json:"message"`
    StartTime   time.Time             `json:"start_time"`
    UpdateTime  time.Time             `json:"update_time"`
    Metadata    map[string]interface{} `json:"metadata"`
    Steps       []ProgressStep        `json:"steps"`
}

type ProgressStep struct {
    Name        string          `json:"name"`
    Status      StepStatus      `json:"status"`
    Progress    float64         `json:"progress"`
    Message     string          `json:"message"`
    StartTime   *time.Time      `json:"start_time,omitempty"`
    EndTime     *time.Time      `json:"end_time,omitempty"`
    Error       *string         `json:"error,omitempty"`
}

func (pt *ProgressTracker) UpdateProgress(operationID string, update ProgressUpdate) {
    if progress, exists := pt.operations.Load(operationID); exists {
        op := progress.(*OperationProgress)

        op.Progress = update.Progress
        op.Message = update.Message
        op.UpdateTime = time.Now()

        if update.StepUpdate != nil {
            pt.updateStep(op, update.StepUpdate)
        }

        // Broadcast to subscribers
        pt.broadcastUpdate(operationID, op)
    }
}
```

## Security Enhancements

### Command Signing and Verification

```go
type CommandSigner struct {
    privateKey *rsa.PrivateKey
    publicKey  *rsa.PublicKey
}

func (cs *CommandSigner) SignCommand(cmd *Command) (string, error) {
    payload := cs.createPayload(cmd)
    hash := sha256.Sum256(payload)

    signature, err := rsa.SignPKCS1v15(rand.Reader, cs.privateKey, crypto.SHA256, hash[:])
    if err != nil {
        return "", err
    }

    return base64.StdEncoding.EncodeToString(signature), nil
}

func (cs *CommandSigner) VerifyCommand(cmd *Command, signature string) error {
    sig, err := base64.StdEncoding.DecodeString(signature)
    if err != nil {
        return err
    }

    payload := cs.createPayload(cmd)
    hash := sha256.Sum256(payload)

    return rsa.VerifyPKCS1v15(cs.publicKey, crypto.SHA256, hash[:], sig)
}
```

### Audit Trail Enhancement

```go
type DetailedCommandAudit struct {
    *CommandAudit
    RequestHeaders    map[string]string  `json:"request_headers"`
    CLIVersion       string             `json:"cli_version"`
    PortalVersion    string             `json:"portal_version"`
    ResourceUsage    *ResourceUsage     `json:"resource_usage"`
    NetworkActivity  *NetworkActivity   `json:"network_activity"`
    SecurityEvents   []SecurityEvent    `json:"security_events"`
}

type ResourceUsage struct {
    PeakMemoryMB     float64       `json:"peak_memory_mb"`
    CPUTimeSeconds   float64       `json:"cpu_time_seconds"`
    DiskIOBytes      int64         `json:"disk_io_bytes"`
    NetworkIOBytes   int64         `json:"network_io_bytes"`
    ExecutionTimeMs  int64         `json:"execution_time_ms"`
}

type SecurityEvent struct {
    Type        SecurityEventType `json:"type"`
    Severity    SecuritySeverity  `json:"severity"`
    Message     string           `json:"message"`
    Timestamp   time.Time        `json:"timestamp"`
    Details     map[string]interface{} `json:"details"`
}
```

## Testing Framework Extension

### Command Mapping Integration Tests

```go
func TestCommandMappingIntegration(t *testing.T) {
    tests := []struct {
        name        string
        endpoint    string
        method      string
        request     interface{}
        expectedCLI string
        mockOutput  string
        expectedResponse interface{}
    }{
        {
            name:     "Server Enable with Config",
            endpoint: "/api/v1/servers/github/enable",
            method:   "POST",
            request: map[string]interface{}{
                "config": map[string]interface{}{
                    "auto_start": true,
                    "restart_policy": "always",
                },
            },
            expectedCLI: "docker mcp server enable github --user test-user --config-file /tmp/config_*.json --auto-start",
            mockOutput: `{"success": true, "container_id": "abc123", "status": "enabling"}`,
            expectedResponse: map[string]interface{}{
                "message": "Server enabling in progress",
                "operation_id": "op_*",
                "estimated_time": 30,
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Execute test with mock CLI executor
            result := executeCommandMappingTest(t, tt)
            assert.Equal(t, tt.expectedResponse, result)
        })
    }
}
```

This comprehensive command mapping framework provides robust, secure, and efficient integration between the web portal and the MCP Gateway CLI, ensuring reliable operation at scale while maintaining security and auditability.
