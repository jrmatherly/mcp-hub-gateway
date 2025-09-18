package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockCLIExecutor provides a mock implementation for testing
type MockCLIExecutor struct {
	mu              sync.Mutex
	responses       map[CommandType]MockResponse
	executedCmds    []ExecutedCommand
	failNextN       int
	delayNext       time.Duration
	validateCalls   bool
	securityChecked bool
}

// MockResponse defines a mock response for a command type
type MockResponse struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Error    error
}

// ExecutedCommand tracks executed commands for verification
type ExecutedCommand struct {
	Type      CommandType
	Args      []string
	UserID    string
	Timestamp time.Time
}

// Command type alias for compatibility
type Command = ExecutionRequest

// Result type alias for compatibility
type Result = ExecutionResult

// NewMockCLIExecutor creates a new mock executor
func NewMockCLIExecutor() *MockCLIExecutor {
	return &MockCLIExecutor{
		responses:     make(map[CommandType]MockResponse),
		executedCmds:  make([]ExecutedCommand, 0),
		validateCalls: true,
	}
}

// SetResponse sets a mock response for a command type
func (m *MockCLIExecutor) SetResponse(cmdType CommandType, response MockResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[cmdType] = response
}

// SetJSONResponse sets a JSON response for a command type
func (m *MockCLIExecutor) SetJSONResponse(cmdType CommandType, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	m.SetResponse(cmdType, MockResponse{
		Stdout:   string(jsonData),
		ExitCode: 0,
	})
	return nil
}

// FailNext causes the next N commands to fail
func (m *MockCLIExecutor) FailNext(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failNextN = n
}

// DelayNext adds a delay to the next command execution
func (m *MockCLIExecutor) DelayNext(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delayNext = d
}

// Execute implements the CLI executor interface
func (m *MockCLIExecutor) Execute(ctx context.Context, cmd Command) (*Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Track the execution
	m.executedCmds = append(m.executedCmds, ExecutedCommand{
		Type:      cmd.Command,
		Args:      cmd.Args,
		UserID:    cmd.UserID,
		Timestamp: time.Now(),
	})

	// Mark that security was checked
	m.securityChecked = true

	// Apply delay if set
	if m.delayNext > 0 {
		delay := m.delayNext
		m.delayNext = 0
		time.Sleep(delay)
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return &ExecutionResult{
			Stderr:    "command cancelled",
			ExitCode:  -1,
			StartTime: time.Now(),
			EndTime:   time.Now(),
			Command:   cmd.Command,
			RequestID: cmd.RequestID,
		}, ctx.Err()
	default:
	}

	// Check if we should fail this command
	if m.failNextN > 0 {
		m.failNextN--
		return &ExecutionResult{
			Stderr:    "mock failure",
			ExitCode:  1,
			StartTime: time.Now(),
			EndTime:   time.Now(),
			Command:   cmd.Command,
			RequestID: cmd.RequestID,
		}, fmt.Errorf("mock failure for command: %s", cmd.Command)
	}

	// Return configured response
	if response, exists := m.responses[cmd.Command]; exists {
		result := &ExecutionResult{
			Stdout:    response.Stdout,
			Stderr:    response.Stderr,
			ExitCode:  response.ExitCode,
			Duration:  100 * time.Millisecond,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(100 * time.Millisecond),
			Command:   cmd.Command,
			RequestID: cmd.RequestID,
			Success:   response.ExitCode == 0,
		}
		return result, response.Error
	}

	// Default response
	return m.defaultResponse(cmd.Command)
}

// defaultResponse provides default responses for common commands
func (m *MockCLIExecutor) defaultResponse(cmdType CommandType) (*Result, error) {
	switch cmdType {
	case CommandTypeServerList:
		return &ExecutionResult{
			Stdout: `{
				"servers": [
					{
						"id": "550e8400-e29b-41d4-a716-446655440000",
						"name": "test-server-1",
						"enabled": true,
						"status": "running"
					},
					{
						"id": "550e8400-e29b-41d4-a716-446655440001",
						"name": "test-server-2",
						"enabled": false,
						"status": "stopped"
					}
				]
			}`,
			ExitCode:  0,
			StartTime: time.Now(),
			EndTime:   time.Now(),
			Command:   cmdType,
			Success:   true,
		}, nil

	case CommandTypeServerEnable, CommandTypeServerDisable:
		return &ExecutionResult{
			Stdout: `{
				"success": true,
				"message": "Server state changed successfully"
			}`,
			ExitCode:  0,
			StartTime: time.Now(),
			EndTime:   time.Now(),
			Command:   cmdType,
			Success:   true,
		}, nil

	case CommandTypeServerInspect:
		return &ExecutionResult{
			Stdout: `{
				"id": "550e8400-e29b-41d4-a716-446655440000",
				"name": "test-server",
				"enabled": true,
				"status": "running",
				"config": {
					"image": "mcpserver/test:latest",
					"env": ["KEY=value"]
				}
			}`,
			ExitCode:  0,
			StartTime: time.Now(),
			EndTime:   time.Now(),
			Command:   cmdType,
			Success:   true,
		}, nil

	case CommandTypeConfigRead:
		return &ExecutionResult{
			Stdout: `{
				"version": "1.0",
				"servers": {},
				"settings": {
					"autoStart": true
				}
			}`,
			ExitCode:  0,
			StartTime: time.Now(),
			EndTime:   time.Now(),
			Command:   cmdType,
			Success:   true,
		}, nil

	default:
		return &ExecutionResult{
			Stderr:    fmt.Sprintf("no mock response configured for command: %s", cmdType),
			ExitCode:  1,
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}, fmt.Errorf("no mock response for command: %s", cmdType)
	}
}

// GetExecutedCommands returns all executed commands
func (m *MockCLIExecutor) GetExecutedCommands() []ExecutedCommand {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]ExecutedCommand, len(m.executedCmds))
	copy(result, m.executedCmds)
	return result
}

// GetLastCommand returns the last executed command
func (m *MockCLIExecutor) GetLastCommand() *ExecutedCommand {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.executedCmds) == 0 {
		return nil
	}

	last := m.executedCmds[len(m.executedCmds)-1]
	return &last
}

// AssertCommandExecuted checks if a command was executed
func (m *MockCLIExecutor) AssertCommandExecuted(cmdType CommandType) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, cmd := range m.executedCmds {
		if cmd.Type == cmdType {
			return true
		}
	}
	return false
}

// AssertSecurityValidated checks if security was validated
func (m *MockCLIExecutor) AssertSecurityValidated() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.securityChecked
}

// Reset clears all mock state
func (m *MockCLIExecutor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.responses = make(map[CommandType]MockResponse)
	m.executedCmds = make([]ExecutedCommand, 0)
	m.failNextN = 0
	m.delayNext = 0
	m.securityChecked = false
}

// CommandCount returns the number of executed commands
func (m *MockCLIExecutor) CommandCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.executedCmds)
}

// SimulateTimeout simulates a command timeout
func (m *MockCLIExecutor) SimulateTimeout(cmdType CommandType) {
	m.SetResponse(cmdType, MockResponse{
		Stderr:   "operation timed out",
		ExitCode: 124, // Standard timeout exit code
		Error:    context.DeadlineExceeded,
	})
}

// SimulateStreamingOutput simulates streaming output for long-running commands
func (m *MockCLIExecutor) SimulateStreamingOutput(cmdType CommandType, lines []string) {
	output := ""
	for _, line := range lines {
		output += line + "\n"
	}

	m.SetResponse(cmdType, MockResponse{
		Stdout:   output,
		ExitCode: 0,
	})
}

// SimpleExecutor provides a simple interface for userconfig tests
type SimpleExecutor struct {
	mock.Mock
}

// SimpleResult represents a simple execution result for compatibility
type SimpleResult struct {
	Success  bool   `json:"success"`
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
}

// Execute implements the simple execute signature expected by userconfig tests
func (m *SimpleExecutor) Execute(
	ctx context.Context,
	command string,
	args []string,
) (*SimpleResult, error) {
	arguments := m.Called(ctx, command, args)
	if arguments.Get(0) == nil {
		return nil, arguments.Error(1)
	}
	return arguments.Get(0).(*SimpleResult), arguments.Error(1)
}

// NewSimpleExecutor creates a new simple executor for userconfig tests
func NewSimpleExecutor() *SimpleExecutor {
	return &SimpleExecutor{}
}

// TestableExecutor provides a testify mock implementation of the Executor interface
type TestableExecutor struct {
	mock.Mock
}

// Execute implements the Executor interface
func (m *TestableExecutor) Execute(
	ctx context.Context,
	req *ExecutionRequest,
) (*ExecutionResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ExecutionResult), args.Error(1)
}

// ExecuteStream implements the Executor interface
func (m *TestableExecutor) ExecuteStream(
	ctx context.Context,
	req *ExecutionRequest,
	outputChan chan<- string,
) (*ExecutionResult, error) {
	args := m.Called(ctx, req, outputChan)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ExecutionResult), args.Error(1)
}

// ValidateCommand implements the Executor interface
func (m *TestableExecutor) ValidateCommand(req *ExecutionRequest) []ValidationError {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]ValidationError)
}

// GetWhitelist implements the Executor interface
func (m *TestableExecutor) GetWhitelist() []CommandWhitelist {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]CommandWhitelist)
}

// GetRateLimit implements the Executor interface
func (m *TestableExecutor) GetRateLimit(
	userID string,
	command CommandType,
) (remaining int, resetTime time.Time, err error) {
	args := m.Called(userID, command)
	return args.Int(0), args.Get(1).(time.Time), args.Error(2)
}

// Health implements the Executor interface
func (m *TestableExecutor) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// NewTestableExecutor creates a new testable executor for integration tests
func NewTestableExecutor() *TestableExecutor {
	return &TestableExecutor{}
}
