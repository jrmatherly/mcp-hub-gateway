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
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/cache"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/executor"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/audit"
)

// containerService implements the ContainerService interface
type containerService struct {
	repo     ContainerRepository
	executor executor.Executor
	audit    audit.Logger
	cache    cache.Cache
	mu       sync.RWMutex

	// State management
	stateTracker map[string]*containerStateTracker
}

// containerStateTracker tracks the state of a container for monitoring
type containerStateTracker struct {
	ContainerID  string
	LastState    ContainerState
	LastCheck    time.Time
	HealthStatus ContainerHealthStatus
	StatsHistory []ContainerStats
	mu           sync.RWMutex
}

// CreateContainerService creates a new container service instance
func CreateContainerService(
	repo ContainerRepository,
	exec executor.Executor,
	auditLogger audit.Logger,
	cacheStore cache.Cache,
) *containerService {
	return &containerService{
		repo:         repo,
		executor:     exec,
		audit:        auditLogger,
		cache:        cacheStore,
		stateTracker: make(map[string]*containerStateTracker),
	}
}

// CreateContainer creates a new Docker container
func (s *containerService) CreateContainer(
	ctx context.Context,
	userID string,
	req *ContainerCreateRequest,
) (*Container, error) {
	// Validate request
	if errs := s.validateCreateContainerRequest(req); len(errs) > 0 {
		return nil, fmt.Errorf("validation failed: %v", errs)
	}

	// Parse user ID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Build docker run command arguments
	args := s.buildDockerRunArgs(req)

	// Execute docker create command
	cliReq := &executor.ExecutionRequest{
		Command:    "docker.create",
		Args:       args,
		UserID:     userID,
		RequestID:  uuid.New().String(),
		Timeout:    60 * time.Second,
		JSONOutput: true,
	}

	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeCommandFailure, map[string]any{
			"command": "container.create",
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("failed to create container via Docker CLI: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("Docker command failed: %s", result.Stderr)
	}

	// Parse container ID from output
	containerID := strings.TrimSpace(result.Stdout)
	if containerID == "" {
		return nil, fmt.Errorf("failed to get container ID from Docker output")
	}

	// Create container entity
	container := &Container{
		ID:            containerID,
		Name:          req.Name,
		Image:         req.Image,
		State:         ContainerStateCreated,
		Status:        "Created",
		HealthStatus:  HealthStatusNone,
		CreatedAt:     time.Now().UTC(),
		Command:       req.Command,
		Args:          req.Args,
		Environment:   req.Environment,
		WorkingDir:    req.WorkingDir,
		Ports:         req.Ports,
		Mounts:        req.Mounts,
		CPUShares:     req.CPUShares,
		Memory:        req.Memory,
		MemorySwap:    req.MemorySwap,
		CPUQuota:      req.CPUQuota,
		CPUPeriod:     req.CPUPeriod,
		RestartPolicy: req.RestartPolicy,
		MaxRetries:    req.MaxRetries,
		Labels:        req.Labels,
		ServerID:      req.ServerID,
		CatalogID:     req.CatalogID,
		UserID:        uid,
		IsManaged:     true,
	}

	// Store in database
	if err := s.repo.CreateContainer(ctx, userID, container); err != nil {
		// If database storage fails, try to remove the created container
		s.removeDockerContainer(ctx, userID, containerID, true)
		return nil, fmt.Errorf("failed to store container: %w", err)
	}

	// Start container if requested
	if req.AutoStart {
		if err := s.StartContainer(ctx, userID, containerID); err != nil {
			return container, fmt.Errorf("container created but failed to start: %w", err)
		}
	}

	// Start tracking container state
	s.startStateTracking(containerID)

	// Log success
	s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeConfiguration, map[string]any{
		"action":       "container.created",
		"container_id": containerID,
		"image":        req.Image,
		"name":         req.Name,
	})

	return container, nil
}

// StartContainer starts a Docker container
func (s *containerService) StartContainer(
	ctx context.Context,
	userID string,
	containerID string,
) error {
	// Validate container exists and get current state
	container, err := s.GetContainer(ctx, userID, containerID)
	if err != nil {
		return err
	}

	if container.State == ContainerStateRunning {
		return nil // Already running
	}

	// Execute docker start command
	cliReq := &executor.ExecutionRequest{
		Command:   "docker.start",
		Args:      []string{containerID},
		UserID:    userID,
		RequestID: uuid.New().String(),
		Timeout:   30 * time.Second,
	}

	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("Docker start failed: %s", result.Stderr)
	}

	// Update container state
	container.State = ContainerStateRunning

	if err := s.repo.UpdateContainerState(ctx, userID, containerID, ContainerStateRunning); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	// Invalidate cache
	s.invalidateContainerCache(userID, containerID)

	// Log action
	uid, _ := uuid.Parse(userID)
	s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeConfiguration, map[string]any{
		"action":       "container.started",
		"container_id": containerID,
	})

	return nil
}

// StopContainer stops a Docker container
func (s *containerService) StopContainer(
	ctx context.Context,
	userID string,
	containerID string,
	timeout time.Duration,
) error {
	// Validate container exists
	container, err := s.GetContainer(ctx, userID, containerID)
	if err != nil {
		return err
	}

	if container.State != ContainerStateRunning {
		return nil // Already stopped
	}

	// Build stop command with timeout
	args := []string{containerID}
	if timeout > 0 {
		args = append([]string{"--time", strconv.Itoa(int(timeout.Seconds()))}, args...)
	}

	// Execute docker stop command
	cliReq := &executor.ExecutionRequest{
		Command:   "docker.stop",
		Args:      args,
		UserID:    userID,
		RequestID: uuid.New().String(),
		Timeout:   timeout + 10*time.Second, // CLI timeout should be longer than stop timeout
	}

	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("Docker stop failed: %s", result.Stderr)
	}

	// Update container state
	if err := s.repo.UpdateContainerState(ctx, userID, containerID, ContainerStateExited); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	// Update in-memory state tracker
	s.updateStateTracker(containerID, ContainerStateExited, HealthStatusNone)

	// Invalidate cache
	s.invalidateContainerCache(userID, containerID)

	// Log action
	uid, _ := uuid.Parse(userID)
	s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeConfiguration, map[string]any{
		"action":       "container.stopped",
		"container_id": containerID,
		"timeout":      timeout.Seconds(),
	})

	return nil
}

// RestartContainer restarts a Docker container
func (s *containerService) RestartContainer(
	ctx context.Context,
	userID string,
	containerID string,
	timeout time.Duration,
) error {
	// Validate container exists
	_, err := s.GetContainer(ctx, userID, containerID)
	if err != nil {
		return err
	}

	// Build restart command with timeout
	args := []string{containerID}
	if timeout > 0 {
		args = append([]string{"--time", strconv.Itoa(int(timeout.Seconds()))}, args...)
	}

	// Execute docker restart command
	cliReq := &executor.ExecutionRequest{
		Command:   "docker.restart",
		Args:      args,
		UserID:    userID,
		RequestID: uuid.New().String(),
		Timeout:   timeout + 20*time.Second, // CLI timeout should be longer
	}

	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("Docker restart failed: %s", result.Stderr)
	}

	// Update container state
	if err := s.repo.UpdateContainerState(ctx, userID, containerID, ContainerStateRunning); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	// Update in-memory state tracker
	s.updateStateTracker(containerID, ContainerStateRunning, HealthStatusStarting)

	// Invalidate cache
	s.invalidateContainerCache(userID, containerID)

	// Log action
	uid, _ := uuid.Parse(userID)
	s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeConfiguration, map[string]any{
		"action":       "container.restarted",
		"container_id": containerID,
		"timeout":      timeout.Seconds(),
	})

	return nil
}

// PauseContainer pauses a Docker container
func (s *containerService) PauseContainer(
	ctx context.Context,
	userID string,
	containerID string,
) error {
	return s.performContainerAction(ctx, userID, containerID, "pause", ContainerStatePaused)
}

// UnpauseContainer unpauses a Docker container
func (s *containerService) UnpauseContainer(
	ctx context.Context,
	userID string,
	containerID string,
) error {
	return s.performContainerAction(ctx, userID, containerID, "unpause", ContainerStateRunning)
}

// KillContainer kills a Docker container
func (s *containerService) KillContainer(
	ctx context.Context,
	userID string,
	containerID string,
	signal string,
) error {
	// Validate container exists
	_, err := s.GetContainer(ctx, userID, containerID)
	if err != nil {
		return err
	}

	// Build kill command with signal
	args := []string{containerID}
	if signal != "" {
		args = append([]string{"--signal", signal}, args...)
	}

	// Execute docker kill command
	cliReq := &executor.ExecutionRequest{
		Command:   "docker.kill",
		Args:      args,
		UserID:    userID,
		RequestID: uuid.New().String(),
		Timeout:   30 * time.Second,
	}

	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		return fmt.Errorf("failed to kill container: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("Docker kill failed: %s", result.Stderr)
	}

	// Update container state
	if err := s.repo.UpdateContainerState(ctx, userID, containerID, ContainerStateExited); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	// Update in-memory state tracker
	s.updateStateTracker(containerID, ContainerStateExited, HealthStatusNone)

	// Invalidate cache
	s.invalidateContainerCache(userID, containerID)

	// Log action
	uid, _ := uuid.Parse(userID)
	s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeConfiguration, map[string]any{
		"action":       "container.killed",
		"container_id": containerID,
		"signal":       signal,
	})

	return nil
}

// RemoveContainer removes a Docker container
func (s *containerService) RemoveContainer(
	ctx context.Context,
	userID string,
	containerID string,
	force bool,
) error {
	// Validate container exists
	_, err := s.GetContainer(ctx, userID, containerID)
	if err != nil {
		return err
	}

	// Remove from Docker
	if err := s.removeDockerContainer(ctx, userID, containerID, force); err != nil {
		return err
	}

	// Remove from database
	if err := s.repo.DeleteContainer(ctx, userID, containerID); err != nil {
		return fmt.Errorf("failed to remove container from database: %w", err)
	}

	// Stop state tracking
	s.stopStateTracking(containerID)

	// Invalidate cache
	s.invalidateContainerCache(userID, containerID)

	// Log action
	uid, _ := uuid.Parse(userID)
	s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeConfiguration, map[string]any{
		"action":       "container.removed",
		"container_id": containerID,
		"force":        force,
	})

	return nil
}

// GetContainer retrieves a container by ID
func (s *containerService) GetContainer(
	ctx context.Context,
	userID string,
	containerID string,
) (*Container, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("container:%s:%s", userID, containerID)
	if data, err := s.cache.Get(ctx, cacheKey); err == nil && data != nil {
		var container Container
		if err := json.Unmarshal(data, &container); err == nil {
			return &container, nil
		}
	}

	// Get from repository
	container, err := s.repo.GetContainer(ctx, userID, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container: %w", err)
	}

	// Refresh container state from Docker
	if err := s.refreshContainerState(ctx, userID, container); err != nil {
		// Log error but don't fail the request
		uid, _ := uuid.Parse(userID)
		s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeWarning, map[string]any{
			"action":       "container.state_refresh_failed",
			"container_id": containerID,
			"error":        err.Error(),
		})
	}

	// Cache the result
	if data, err := json.Marshal(container); err == nil {
		s.cache.Set(ctx, cacheKey, data, 2*time.Minute)
	}

	return container, nil
}

// ListContainers lists containers with filtering
func (s *containerService) ListContainers(
	ctx context.Context,
	userID string,
	filter ContainerFilter,
) ([]*Container, int64, error) {
	// Check cache for list (only for simple filters)
	var cacheKey string
	if s.canCacheList(filter) {
		cacheKey = fmt.Sprintf("containers:%s:%v", userID, filter)
		if data, err := s.cache.Get(ctx, cacheKey); err == nil && data != nil {
			var cached struct {
				Containers []*Container `json:"containers"`
				Count      int64        `json:"count"`
			}
			if err := json.Unmarshal(data, &cached); err == nil {
				return cached.Containers, cached.Count, nil
			}
		}
	}

	// Get from repository
	containers, err := s.repo.ListContainers(ctx, userID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list containers: %w", err)
	}

	count, err := s.repo.CountContainers(ctx, userID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count containers: %w", err)
	}

	// Refresh states for running containers (async)
	go s.refreshContainersState(context.Background(), userID, containers)

	// Cache the results if cacheable
	if cacheKey != "" {
		cacheData := struct {
			Containers []*Container `json:"containers"`
			Count      int64        `json:"count"`
		}{
			Containers: containers,
			Count:      count,
		}
		if data, err := json.Marshal(cacheData); err == nil {
			s.cache.Set(ctx, cacheKey, data, 1*time.Minute)
		}
	}

	return containers, count, nil
}

// GetContainerStats retrieves runtime statistics for a container
func (s *containerService) GetContainerStats(
	ctx context.Context,
	userID string,
	containerID string,
) (*ContainerStats, error) {
	// Validate container exists
	container, err := s.GetContainer(ctx, userID, containerID)
	if err != nil {
		return nil, err
	}

	if container.State != ContainerStateRunning {
		return nil, fmt.Errorf("cannot get stats for non-running container")
	}

	// Execute docker stats command
	cliReq := &executor.ExecutionRequest{
		Command:    "docker.stats",
		Args:       []string{"--no-stream", "--format", "json", containerID},
		UserID:     userID,
		RequestID:  uuid.New().String(),
		Timeout:    15 * time.Second,
		JSONOutput: true,
	}

	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("Docker stats failed: %s", result.Stderr)
	}

	// Parse stats from JSON output
	var dockerStats map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &dockerStats); err != nil {
		return nil, fmt.Errorf("failed to parse Docker stats output: %w", err)
	}

	// Convert to our stats format
	stats := s.parseDockerStats(dockerStats)
	stats.Timestamp = time.Now().UTC()

	// Store stats in database
	if err := s.repo.UpdateContainerStats(ctx, userID, containerID, stats); err != nil {
		// Log error but don't fail the request
		uid, _ := uuid.Parse(userID)
		s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeWarning, map[string]any{
			"action":       "container.stats_store_failed",
			"container_id": containerID,
			"error":        err.Error(),
		})
	}

	return stats, nil
}

// Helper methods

// performContainerAction performs a simple container action (pause, unpause, etc.)
func (s *containerService) performContainerAction(
	ctx context.Context,
	userID string,
	containerID string,
	action string,
	newState ContainerState,
) error {
	// Validate container exists
	_, err := s.GetContainer(ctx, userID, containerID)
	if err != nil {
		return err
	}

	// Execute docker command
	cliReq := &executor.ExecutionRequest{
		Command:   executor.CommandType(fmt.Sprintf("docker.%s", action)),
		Args:      []string{containerID},
		UserID:    userID,
		RequestID: uuid.New().String(),
		Timeout:   30 * time.Second,
	}

	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		return fmt.Errorf("failed to %s container: %w", action, err)
	}

	if !result.Success {
		return fmt.Errorf("Docker %s failed: %s", action, result.Stderr)
	}

	// Update container state
	if err := s.repo.UpdateContainerState(ctx, userID, containerID, newState); err != nil {
		return fmt.Errorf("failed to update container state: %w", err)
	}

	// Update in-memory state tracker
	var healthStatus ContainerHealthStatus = HealthStatusNone
	if newState == ContainerStateRunning {
		healthStatus = HealthStatusStarting
	}
	s.updateStateTracker(containerID, newState, healthStatus)

	// Invalidate cache
	s.invalidateContainerCache(userID, containerID)

	// Log action
	uid, _ := uuid.Parse(userID)
	s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeConfiguration, map[string]any{
		"action":       fmt.Sprintf("container.%s", action),
		"container_id": containerID,
	})

	return nil
}

// removeDockerContainer removes a container from Docker
func (s *containerService) removeDockerContainer(
	ctx context.Context,
	userID string,
	containerID string,
	force bool,
) error {
	args := []string{containerID}
	if force {
		args = append([]string{"--force"}, args...)
	}

	cliReq := &executor.ExecutionRequest{
		Command:   "docker.rm",
		Args:      args,
		UserID:    userID,
		RequestID: uuid.New().String(),
		Timeout:   30 * time.Second,
	}

	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		return fmt.Errorf("failed to remove container from Docker: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("Docker remove failed: %s", result.Stderr)
	}

	return nil
}

// buildDockerRunArgs builds command arguments for docker create/run
func (s *containerService) buildDockerRunArgs(req *ContainerCreateRequest) []string {
	var args []string

	// Container name
	if req.Name != "" {
		args = append(args, "--name", req.Name)
	}

	// Environment variables
	for key, value := range req.Environment {
		args = append(args, "--env", fmt.Sprintf("%s=%s", key, value))
	}

	// Port mappings
	for _, port := range req.Ports {
		portMapping := fmt.Sprintf("%s:%s", port.HostPort, port.ContainerPort)
		if port.HostIP != "" {
			portMapping = fmt.Sprintf("%s:%s", port.HostIP, portMapping)
		}
		if port.Protocol != "" && port.Protocol != "tcp" {
			portMapping = fmt.Sprintf("%s/%s", portMapping, port.Protocol)
		}
		args = append(args, "--publish", portMapping)
	}

	// Volume mounts
	for _, mount := range req.Mounts {
		mountStr := fmt.Sprintf("%s:%s", mount.Source, mount.Destination)
		if mount.Mode != "" {
			mountStr = fmt.Sprintf("%s:%s", mountStr, mount.Mode)
		}
		args = append(args, "--volume", mountStr)
	}

	// Resource limits
	if req.Memory > 0 {
		args = append(args, "--memory", fmt.Sprintf("%d", req.Memory))
	}
	if req.CPUShares > 0 {
		args = append(args, "--cpu-shares", fmt.Sprintf("%d", req.CPUShares))
	}
	if req.CPUQuota > 0 && req.CPUPeriod > 0 {
		args = append(args, "--cpu-quota", fmt.Sprintf("%d", req.CPUQuota))
		args = append(args, "--cpu-period", fmt.Sprintf("%d", req.CPUPeriod))
	}

	// Restart policy
	if req.RestartPolicy != "" {
		restartStr := string(req.RestartPolicy)
		if req.RestartPolicy == RestartPolicyOnFailure && req.MaxRetries > 0 {
			restartStr = fmt.Sprintf("%s:%d", restartStr, req.MaxRetries)
		}
		args = append(args, "--restart", restartStr)
	}

	// Labels
	for key, value := range req.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}

	// Working directory
	if req.WorkingDir != "" {
		args = append(args, "--workdir", req.WorkingDir)
	}

	// Auto remove
	if req.AutoRemove {
		args = append(args, "--rm")
	}

	// Image
	args = append(args, req.Image)

	// Command and args
	if len(req.Command) > 0 {
		args = append(args, req.Command...)
	}
	if len(req.Args) > 0 {
		args = append(args, req.Args...)
	}

	return args
}

// refreshContainerState refreshes container state from Docker
func (s *containerService) refreshContainerState(
	ctx context.Context,
	userID string,
	container *Container,
) error {
	// Execute docker inspect command
	cliReq := &executor.ExecutionRequest{
		Command:    "docker.inspect",
		Args:       []string{"--format", "json", container.ID},
		UserID:     userID,
		RequestID:  uuid.New().String(),
		Timeout:    10 * time.Second,
		JSONOutput: true,
	}

	result, err := s.executor.Execute(ctx, cliReq)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("Docker inspect failed: %s", result.Stderr)
	}

	// Parse inspect output
	var inspectData []map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &inspectData); err != nil {
		return fmt.Errorf("failed to parse inspect output: %w", err)
	}

	if len(inspectData) == 0 {
		return fmt.Errorf("no container data in inspect output")
	}

	// Update container with fresh data
	s.updateContainerFromInspect(container, inspectData[0])

	return nil
}

// updateContainerFromInspect updates container fields from docker inspect output
func (s *containerService) updateContainerFromInspect(
	container *Container,
	inspectData map[string]any,
) {
	// Update state
	if stateData, ok := inspectData["State"].(map[string]any); ok {
		if status, ok := stateData["Status"].(string); ok {
			container.State = ContainerState(strings.ToLower(status))
			container.Status = status
		}
		if exitCode, ok := stateData["ExitCode"].(float64); ok {
			container.ExitCode = int(exitCode)
		}
		if startedAt, ok := stateData["StartedAt"].(string); ok && startedAt != "" {
			if t, err := time.Parse(time.RFC3339Nano, startedAt); err == nil {
				container.StartedAt = &t
			}
		}
		if finishedAt, ok := stateData["FinishedAt"].(string); ok && finishedAt != "" {
			if t, err := time.Parse(time.RFC3339Nano, finishedAt); err == nil {
				container.FinishedAt = &t
			}
		}

		// Update health status
		if health, ok := stateData["Health"].(map[string]any); ok {
			if status, ok := health["Status"].(string); ok {
				container.HealthStatus = ContainerHealthStatus(strings.ToLower(status))
			}
		}
	}

	// Update network info
	if networkSettings, ok := inspectData["NetworkSettings"].(map[string]any); ok {
		if ipAddress, ok := networkSettings["IPAddress"].(string); ok {
			container.IPAddress = ipAddress
		}
	}
}

// State tracking methods

// startStateTracking starts tracking state for a container
func (s *containerService) startStateTracking(containerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stateTracker[containerID] = &containerStateTracker{
		ContainerID:  containerID,
		LastState:    ContainerStateCreated,
		LastCheck:    time.Now(),
		HealthStatus: HealthStatusNone,
		StatsHistory: make([]ContainerStats, 0, 100), // Keep last 100 stats
	}
}

// stopStateTracking stops tracking state for a container
func (s *containerService) stopStateTracking(containerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.stateTracker, containerID)
}

// updateStateTracker updates the state tracker for a container
func (s *containerService) updateStateTracker(
	containerID string,
	state ContainerState,
	healthStatus ContainerHealthStatus,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if tracker, exists := s.stateTracker[containerID]; exists {
		tracker.mu.Lock()
		tracker.LastState = state
		tracker.LastCheck = time.Now()
		tracker.HealthStatus = healthStatus
		tracker.mu.Unlock()
	}
}

// Cache management methods

// invalidateContainerCache invalidates cache entries for a container
func (s *containerService) invalidateContainerCache(userID, containerID string) {
	ctx := context.Background()

	// Invalidate specific container cache
	cacheKey := fmt.Sprintf("container:%s:%s", userID, containerID)
	s.cache.Delete(ctx, cacheKey)

	// Invalidate container lists (we could be more selective here)
	// Note: Redis doesn't have a direct way to delete by pattern, so we'd need to track keys
	// For now, we'll just invalidate the most common list cache
	s.cache.Delete(ctx, fmt.Sprintf("containers:%s:%v", userID, ContainerFilter{}))
}

// canCacheList determines if a container list can be cached
func (s *containerService) canCacheList(filter ContainerFilter) bool {
	// Only cache simple filters to avoid memory issues
	return filter.Limit <= 50 &&
		len(filter.Names) == 0 &&
		len(filter.IDs) == 0 &&
		filter.CreatedBefore == nil &&
		filter.CreatedAfter == nil
}

// refreshContainersState refreshes state for multiple containers asynchronously
func (s *containerService) refreshContainersState(
	ctx context.Context,
	userID string,
	containers []*Container,
) {
	for _, container := range containers {
		if container.State == ContainerStateRunning || container.State == ContainerStatePaused {
			go func(c *Container) {
				if err := s.refreshContainerState(ctx, userID, c); err != nil {
					// Log error but don't fail
					uid, _ := uuid.Parse(userID)
					s.audit.LogSecurityEvent(ctx, uid, audit.EventTypeWarning, map[string]any{
						"action":       "container.async_refresh_failed",
						"container_id": c.ID,
						"error":        err.Error(),
					})
				}
			}(container)
		}
	}
}

// parseDockerStats parses Docker stats output into our format
func (s *containerService) parseDockerStats(dockerStats map[string]any) *ContainerStats {
	stats := &ContainerStats{}

	// Parse CPU usage
	if cpuUsage, ok := dockerStats["CPUPerc"].(string); ok {
		if val, err := strconv.ParseFloat(strings.TrimSuffix(cpuUsage, "%"), 64); err == nil {
			stats.CPUPercent = val
		}
	}

	// Parse memory usage
	if memUsage, ok := dockerStats["MemUsage"].(string); ok {
		parts := strings.Split(memUsage, " / ")
		if len(parts) == 2 {
			if val, err := parseMemoryValue(parts[0]); err == nil {
				stats.MemoryUsage = val
			}
			if val, err := parseMemoryValue(parts[1]); err == nil {
				stats.MemoryLimit = val
			}
		}
	}

	if memPerc, ok := dockerStats["MemPerc"].(string); ok {
		if val, err := strconv.ParseFloat(strings.TrimSuffix(memPerc, "%"), 64); err == nil {
			stats.MemoryPercent = val
		}
	}

	// Parse network I/O
	if netIO, ok := dockerStats["NetIO"].(string); ok {
		parts := strings.Split(netIO, " / ")
		if len(parts) == 2 {
			if val, err := parseMemoryValue(parts[0]); err == nil {
				stats.NetworkRx = val
			}
			if val, err := parseMemoryValue(parts[1]); err == nil {
				stats.NetworkTx = val
			}
		}
	}

	// Parse block I/O
	if blockIO, ok := dockerStats["BlockIO"].(string); ok {
		parts := strings.Split(blockIO, " / ")
		if len(parts) == 2 {
			if val, err := parseMemoryValue(parts[0]); err == nil {
				stats.BlockRead = val
			}
			if val, err := parseMemoryValue(parts[1]); err == nil {
				stats.BlockWrite = val
			}
		}
	}

	// Parse PIDs
	if pids, ok := dockerStats["PIDs"].(string); ok {
		if val, err := strconv.ParseUint(pids, 10, 64); err == nil {
			stats.PIDs = val
		}
	}

	return stats
}

// parseMemoryValue parses a memory value string (e.g., "1.5GiB") to bytes
func parseMemoryValue(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	// Define multipliers
	multipliers := map[string]uint64{
		"B":   1,
		"KiB": 1024,
		"MiB": 1024 * 1024,
		"GiB": 1024 * 1024 * 1024,
		"TiB": 1024 * 1024 * 1024 * 1024,
		"kB":  1000,
		"MB":  1000 * 1000,
		"GB":  1000 * 1000 * 1000,
		"TB":  1000 * 1000 * 1000 * 1000,
	}

	// Find the unit
	for unit, multiplier := range multipliers {
		if strings.HasSuffix(s, unit) {
			valueStr := strings.TrimSpace(strings.TrimSuffix(s, unit))
			if val, err := strconv.ParseFloat(valueStr, 64); err == nil {
				return uint64(val * float64(multiplier)), nil
			}
		}
	}

	// Try parsing as plain number (bytes)
	if val, err := strconv.ParseUint(s, 10, 64); err == nil {
		return val, nil
	}

	return 0, fmt.Errorf("unable to parse memory value: %s", s)
}

// Validation methods

// validateCreateContainerRequest validates a container creation request
func (s *containerService) validateCreateContainerRequest(
	req *ContainerCreateRequest,
) []ValidationError {
	var errors []ValidationError

	// Validate name
	if req.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "container name is required",
			Code:    "required",
		})
	} else if len(req.Name) > 255 {
		errors = append(errors, ValidationError{
			Field:   "name",
			Value:   req.Name,
			Message: "container name must be 255 characters or less",
			Code:    "max_length",
		})
	}

	// Validate image
	if req.Image == "" {
		errors = append(errors, ValidationError{
			Field:   "image",
			Message: "image is required",
			Code:    "required",
		})
	}

	// Validate port mappings
	for i, port := range req.Ports {
		if port.ContainerPort == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("ports[%d].container_port", i),
				Message: "container port is required",
				Code:    "required",
			})
		}
		if port.Protocol != "" && port.Protocol != "tcp" && port.Protocol != "udp" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("ports[%d].protocol", i),
				Value:   port.Protocol,
				Message: "protocol must be tcp or udp",
				Code:    "invalid_enum",
			})
		}
	}

	// Validate mounts
	for i, mount := range req.Mounts {
		if mount.Source == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("mounts[%d].source", i),
				Message: "mount source is required",
				Code:    "required",
			})
		}
		if mount.Destination == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("mounts[%d].destination", i),
				Message: "mount destination is required",
				Code:    "required",
			})
		}
		if mount.Type != "" && mount.Type != "bind" && mount.Type != "volume" &&
			mount.Type != "tmpfs" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("mounts[%d].type", i),
				Value:   mount.Type,
				Message: "mount type must be bind, volume, or tmpfs",
				Code:    "invalid_enum",
			})
		}
	}

	// Validate resource limits
	if req.Memory < 0 {
		errors = append(errors, ValidationError{
			Field:   "memory",
			Value:   fmt.Sprintf("%d", req.Memory),
			Message: "memory limit cannot be negative",
			Code:    "invalid_range",
		})
	}

	if req.CPUShares < 0 {
		errors = append(errors, ValidationError{
			Field:   "cpu_shares",
			Value:   fmt.Sprintf("%d", req.CPUShares),
			Message: "CPU shares cannot be negative",
			Code:    "invalid_range",
		})
	}

	return errors
}

// Ensure interface compliance
var _ ContainerService = (*containerService)(nil)
