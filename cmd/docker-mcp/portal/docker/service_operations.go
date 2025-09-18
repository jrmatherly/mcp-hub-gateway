package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/executor"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/audit"
)

// GetContainerLogs retrieves container logs
func (s *containerService) GetContainerLogs(
	ctx context.Context,
	userID string,
	req *LogsRequest,
) ([]ContainerLogs, error) {
	// Validate container exists
	_, err := s.GetContainer(ctx, userID, req.ContainerID)
	if err != nil {
		return nil, err
	}

	// Build docker logs command arguments
	args := []string{"logs"}

	if req.Follow {
		args = append(args, "--follow")
	}
	if req.Tail != "" {
		args = append(args, "--tail", req.Tail)
	}
	if req.Since != nil {
		args = append(args, "--since", req.Since.Format(time.RFC3339))
	}
	if req.Until != nil {
		args = append(args, "--until", req.Until.Format(time.RFC3339))
	}
	if req.Timestamps {
		args = append(args, "--timestamps")
	}
	if req.Details {
		args = append(args, "--details")
	}

	args = append(args, req.ContainerID)

	// Execute docker logs command
	cliReq := &executor.ExecutionRequest{
		Command:      "docker.logs",
		Args:         args,
		UserID:       userID,
		RequestID:    uuid.New().String(),
		Timeout:      60 * time.Second,
		StreamOutput: req.Follow,
	}

	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("Docker logs failed: %s", result.Stderr)
	}

	// Parse logs output
	logs := s.parseContainerLogs(req.ContainerID, result.Stdout, req.Timestamps)

	// Log access
	uid, _ := uuid.Parse(userID)
	s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeDataAccess, map[string]any{
		"action":       "container.logs_accessed",
		"container_id": req.ContainerID,
		"follow":       req.Follow,
		"lines":        len(logs),
	})

	return logs, nil
}

// ExecInContainer executes a command in a container
func (s *containerService) ExecInContainer(
	ctx context.Context,
	userID string,
	req *ExecRequest,
) (*ExecResult, error) {
	// Validate container exists and is running
	container, err := s.GetContainer(ctx, userID, req.ContainerID)
	if err != nil {
		return nil, err
	}

	if container.State != ContainerStateRunning {
		return nil, ErrContainerNotRunning
	}

	// Build docker exec command arguments
	args := []string{"exec"}

	if req.User != "" {
		args = append(args, "--user", req.User)
	}
	if req.WorkingDir != "" {
		args = append(args, "--workdir", req.WorkingDir)
	}
	if req.Privileged {
		args = append(args, "--privileged")
	}
	if req.TTY {
		args = append(args, "--tty")
	}
	if req.AttachStdin {
		args = append(args, "--interactive")
	}

	// Add environment variables
	for _, env := range req.Environment {
		args = append(args, "--env", env)
	}

	args = append(args, req.ContainerID)
	args = append(args, req.Command...)

	// Execute docker exec command
	startTime := time.Now()
	cliReq := &executor.ExecutionRequest{
		Command:   "docker.exec",
		Args:      args,
		UserID:    userID,
		RequestID: uuid.New().String(),
		Timeout:   120 * time.Second, // Longer timeout for exec
	}

	result, err := s.executor.Execute(ctx, cliReq)
	duration := time.Since(startTime)

	execResult := &ExecResult{
		ExecID:      uuid.New().String(), // Generate a unique exec ID
		ContainerID: req.ContainerID,
		Command:     req.Command,
		Duration:    duration,
		Timestamp:   startTime,
	}

	if err != nil {
		execResult.ExitCode = -1
		execResult.Stderr = err.Error()
		return execResult, fmt.Errorf("failed to exec in container: %w", err)
	}

	execResult.ExitCode = result.ExitCode
	execResult.Stdout = result.Stdout
	execResult.Stderr = result.Stderr

	// Log execution
	uid, _ := uuid.Parse(userID)
	s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeExecution, map[string]any{
		"action":       "container.exec",
		"container_id": req.ContainerID,
		"command":      strings.Join(req.Command, " "),
		"exit_code":    execResult.ExitCode,
		"duration":     duration.Milliseconds(),
		"success":      result.Success,
	})

	return execResult, nil
}

// BulkContainerAction performs actions on multiple containers
func (s *containerService) BulkContainerAction(
	ctx context.Context,
	userID string,
	req *ContainerActionRequest,
) (*BulkActionResult, error) {
	startTime := time.Now()

	result := &BulkActionResult{
		Action:       req.Action,
		TotalCount:   len(req.ContainerIDs),
		SuccessCount: 0,
		FailureCount: 0,
		Results:      make([]ContainerActionResult, len(req.ContainerIDs)),
		Timestamp:    startTime,
	}

	// Validate action
	validActions := []ContainerAction{
		ActionStart,
		ActionStop,
		ActionRestart,
		ActionPause,
		ActionUnpause,
		ActionKill,
		ActionRemove,
		ActionUpdate,
	}
	isValidAction := false
	for _, action := range validActions {
		if req.Action == action {
			isValidAction = true
			break
		}
	}

	if !isValidAction {
		return nil, fmt.Errorf("invalid action: %s", req.Action)
	}

	// Process containers in parallel (with rate limiting)
	semaphore := make(chan struct{}, 5) // Limit to 5 concurrent operations
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, containerID := range req.ContainerIDs {
		wg.Add(1)
		go func(index int, id string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			actionResult := ContainerActionResult{
				ContainerID: id,
				Action:      req.Action,
				Timestamp:   time.Now(),
			}

			actionStart := time.Now()
			err := s.performSingleContainerAction(ctx, userID, id, req)
			actionResult.Duration = time.Since(actionStart)

			if err != nil {
				actionResult.Success = false
				actionResult.Error = err.Error()
				mu.Lock()
				result.FailureCount++
				mu.Unlock()
			} else {
				actionResult.Success = true
				mu.Lock()
				result.SuccessCount++
				mu.Unlock()
			}

			mu.Lock()
			result.Results[index] = actionResult
			mu.Unlock()
		}(i, containerID)
	}

	// Wait for all operations to complete
	wg.Wait()
	result.Duration = time.Since(startTime)

	// Log bulk action
	uid, _ := uuid.Parse(userID)
	s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeBulkOperation, map[string]any{
		"action":        fmt.Sprintf("container.bulk_%s", req.Action),
		"total_count":   result.TotalCount,
		"success_count": result.SuccessCount,
		"failure_count": result.FailureCount,
		"duration":      result.Duration.Milliseconds(),
	})

	return result, nil
}

// UpdateContainer updates container configuration
func (s *containerService) UpdateContainer(
	ctx context.Context,
	userID string,
	containerID string,
	req *ContainerUpdateRequest,
) error {
	// Validate container exists
	container, err := s.GetContainer(ctx, userID, containerID)
	if err != nil {
		return err
	}

	// Build docker update command arguments
	args := []string{"update"}

	if req.CPUShares != nil {
		args = append(args, "--cpu-shares", strconv.FormatInt(*req.CPUShares, 10))
	}
	if req.Memory != nil {
		args = append(args, "--memory", strconv.FormatInt(*req.Memory, 10))
	}
	if req.MemorySwap != nil {
		args = append(args, "--memory-swap", strconv.FormatInt(*req.MemorySwap, 10))
	}
	if req.CPUQuota != nil {
		args = append(args, "--cpu-quota", strconv.FormatInt(*req.CPUQuota, 10))
	}
	if req.CPUPeriod != nil {
		args = append(args, "--cpu-period", strconv.FormatInt(*req.CPUPeriod, 10))
	}
	if req.RestartPolicy != nil {
		restartStr := string(*req.RestartPolicy)
		if *req.RestartPolicy == RestartPolicyOnFailure && req.MaxRetries != nil &&
			*req.MaxRetries > 0 {
			restartStr = fmt.Sprintf("%s:%d", restartStr, *req.MaxRetries)
		}
		args = append(args, "--restart", restartStr)
	}

	args = append(args, containerID)

	// Execute docker update command
	cliReq := &executor.ExecutionRequest{
		Command:   "docker.update",
		Args:      args,
		UserID:    userID,
		RequestID: uuid.New().String(),
		Timeout:   30 * time.Second,
	}

	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		return fmt.Errorf("failed to update container: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("Docker update failed: %s", result.Stderr)
	}

	// Update container in database
	if req.CPUShares != nil {
		container.CPUShares = *req.CPUShares
	}
	if req.Memory != nil {
		container.Memory = *req.Memory
	}
	if req.MemorySwap != nil {
		container.MemorySwap = *req.MemorySwap
	}
	if req.CPUQuota != nil {
		container.CPUQuota = *req.CPUQuota
	}
	if req.CPUPeriod != nil {
		container.CPUPeriod = *req.CPUPeriod
	}
	if req.RestartPolicy != nil {
		container.RestartPolicy = *req.RestartPolicy
	}
	if req.MaxRetries != nil {
		container.MaxRetries = *req.MaxRetries
	}

	if err := s.repo.UpdateContainer(ctx, userID, container); err != nil {
		return fmt.Errorf("failed to update container in database: %w", err)
	}

	// Invalidate cache
	s.invalidateContainerCache(userID, containerID)

	// Log update
	uid, _ := uuid.Parse(userID)
	s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeConfiguration, map[string]any{
		"action":       "container.updated",
		"container_id": containerID,
	})

	return nil
}

// GetSystemInfo retrieves Docker system information
func (s *containerService) GetSystemInfo(ctx context.Context, userID string) (*SystemInfo, error) {
	// Check cache first
	cacheKey := "docker:system_info"
	if data, err := s.cache.Get(ctx, cacheKey); err == nil && data != nil {
		var info SystemInfo
		if err := json.Unmarshal(data, &info); err == nil {
			return &info, nil
		}
	}

	// Execute docker system info command
	cliReq := &executor.ExecutionRequest{
		Command:    "docker.info",
		Args:       []string{"--format", "json"},
		UserID:     userID,
		RequestID:  uuid.New().String(),
		Timeout:    30 * time.Second,
		JSONOutput: true,
	}

	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker system info: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("Docker info failed: %s", result.Stderr)
	}

	// Parse system info from JSON output
	var dockerInfo map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &dockerInfo); err != nil {
		return nil, fmt.Errorf("failed to parse Docker info output: %w", err)
	}

	// Convert to our format
	info := s.parseDockerSystemInfo(dockerInfo)
	info.ServerTime = time.Now().UTC()

	// Cache the result for 5 minutes
	if data, err := json.Marshal(info); err == nil {
		s.cache.Set(ctx, cacheKey, data, 5*time.Minute)
	}

	return info, nil
}

// HealthCheck checks Docker daemon health
func (s *containerService) HealthCheck(ctx context.Context) error {
	// Execute docker version command as health check
	cliReq := &executor.ExecutionRequest{
		Command:   "docker.version",
		Args:      []string{"--format", "json"},
		UserID:    "system",
		RequestID: uuid.New().String(),
		Timeout:   10 * time.Second,
	}

	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		return fmt.Errorf("Docker daemon not healthy: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("Docker daemon not responsive: %s", result.Stderr)
	}

	return nil
}

// GetContainerHealth checks the health status of a specific container
func (s *containerService) GetContainerHealth(
	ctx context.Context,
	userID string,
	containerID string,
) (*ContainerHealthStatus, error) {
	// Get container with current state
	container, err := s.GetContainer(ctx, userID, containerID)
	if err != nil {
		return nil, err
	}

	// If container is not running, health is none
	if container.State != ContainerStateRunning {
		status := HealthStatusNone
		return &status, nil
	}

	// Execute docker inspect to get health status
	cliReq := &executor.ExecutionRequest{
		Command:   "docker.inspect",
		Args:      []string{"--format", "{{.State.Health.Status}}", containerID},
		UserID:    userID,
		RequestID: uuid.New().String(),
		Timeout:   10 * time.Second,
	}

	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get container health: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("Docker inspect failed: %s", result.Stderr)
	}

	// Parse health status
	healthStr := strings.TrimSpace(result.Stdout)
	if healthStr == "" || healthStr == "<no value>" {
		status := HealthStatusNone
		return &status, nil
	}

	status := ContainerHealthStatus(strings.ToLower(healthStr))
	return &status, nil
}

// Helper methods for bulk operations and parsing

// performSingleContainerAction performs a single container action for bulk operations
func (s *containerService) performSingleContainerAction(
	ctx context.Context,
	userID string,
	containerID string,
	req *ContainerActionRequest,
) error {
	switch req.Action {
	case ActionStart:
		return s.StartContainer(ctx, userID, containerID)
	case ActionStop:
		timeout := 10 * time.Second
		if req.Timeout > 0 {
			timeout = req.Timeout
		}
		return s.StopContainer(ctx, userID, containerID, timeout)
	case ActionRestart:
		timeout := 10 * time.Second
		if req.Timeout > 0 {
			timeout = req.Timeout
		}
		return s.RestartContainer(ctx, userID, containerID, timeout)
	case ActionPause:
		return s.PauseContainer(ctx, userID, containerID)
	case ActionUnpause:
		return s.UnpauseContainer(ctx, userID, containerID)
	case ActionKill:
		signal := "SIGKILL"
		if req.Signal != "" {
			signal = req.Signal
		}
		return s.KillContainer(ctx, userID, containerID, signal)
	case ActionRemove:
		return s.RemoveContainer(ctx, userID, containerID, req.Force)
	case ActionUpdate:
		if req.UpdateConfig == nil {
			return fmt.Errorf("update config is required for update action")
		}
		return s.UpdateContainer(ctx, userID, containerID, req.UpdateConfig)
	default:
		return fmt.Errorf("unsupported action: %s", req.Action)
	}
}

// parseContainerLogs parses Docker logs output into structured format
func (s *containerService) parseContainerLogs(
	containerID, output string,
	hasTimestamps bool,
) []ContainerLogs {
	var logs []ContainerLogs
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		log := ContainerLogs{
			ContainerID: containerID,
			Stream:      "stdout", // Default to stdout
			Message:     line,
			Timestamp:   time.Now().UTC(),
		}

		// Parse timestamps if present
		if hasTimestamps && len(line) > 30 {
			// Docker timestamp format: 2023-01-01T12:00:00.000000000Z
			if timestampEnd := strings.Index(line, "Z "); timestampEnd > 0 {
				timestampStr := line[:timestampEnd+1]
				if t, err := time.Parse(time.RFC3339Nano, timestampStr); err == nil {
					log.Timestamp = t
					log.Message = line[timestampEnd+2:]
				}
			}
		}

		// Detect stderr vs stdout (basic heuristic)
		if strings.Contains(strings.ToLower(log.Message), "error") ||
			strings.Contains(strings.ToLower(log.Message), "warning") ||
			strings.Contains(strings.ToLower(log.Message), "fatal") {
			log.Stream = "stderr"
		}

		logs = append(logs, log)
	}

	return logs
}

// parseDockerSystemInfo parses Docker system info output into our format
func (s *containerService) parseDockerSystemInfo(dockerInfo map[string]any) *SystemInfo {
	info := &SystemInfo{}

	// Basic version info
	if serverVersion, ok := dockerInfo["ServerVersion"].(string); ok {
		info.DockerVersion = serverVersion
	}
	if apiVersion, ok := dockerInfo["ApiVersion"].(string); ok {
		info.APIVersion = apiVersion
	}
	if gitCommit, ok := dockerInfo["GitCommit"].(string); ok {
		info.GitCommit = gitCommit
	}
	if goVersion, ok := dockerInfo["GoVersion"].(string); ok {
		info.GoVersion = goVersion
	}
	if osType, ok := dockerInfo["OSType"].(string); ok {
		info.OS = osType
	}
	if arch, ok := dockerInfo["Architecture"].(string); ok {
		info.Arch = arch
	}
	if kernelVersion, ok := dockerInfo["KernelVersion"].(string); ok {
		info.KernelVersion = kernelVersion
	}

	// System resources
	if ncpu, ok := dockerInfo["NCPU"].(float64); ok {
		info.NCPU = int(ncpu)
	}
	if memTotal, ok := dockerInfo["MemTotal"].(float64); ok {
		info.MemTotal = int64(memTotal)
	}
	if dockerRootDir, ok := dockerInfo["DockerRootDir"].(string); ok {
		info.DockerRootDir = dockerRootDir
	}

	// Container counts
	if containers, ok := dockerInfo["Containers"].(float64); ok {
		info.ContainersRunning = int(containers)
	}
	if containersRunning, ok := dockerInfo["ContainersRunning"].(float64); ok {
		info.ContainersRunning = int(containersRunning)
	}
	if containersPaused, ok := dockerInfo["ContainersPaused"].(float64); ok {
		info.ContainersPaused = int(containersPaused)
	}
	if containersStopped, ok := dockerInfo["ContainersStopped"].(float64); ok {
		info.ContainersStopped = int(containersStopped)
	}
	if images, ok := dockerInfo["Images"].(float64); ok {
		info.Images = int(images)
	}

	// Storage driver
	if driver, ok := dockerInfo["Driver"].(string); ok {
		info.Driver = driver
	}
	if driverStatus, ok := dockerInfo["DriverStatus"].([]any); ok {
		info.DriverStatus = make(map[string]string)
		for _, statusPair := range driverStatus {
			if pair, ok := statusPair.([]any); ok && len(pair) == 2 {
				if key, ok := pair[0].(string); ok {
					if value, ok := pair[1].(string); ok {
						info.DriverStatus[key] = value
					}
				}
			}
		}
	}

	// Registry config
	if registryConfig, ok := dockerInfo["RegistryConfig"].(map[string]any); ok {
		info.RegistryConfig = registryConfig
	}

	return info
}
