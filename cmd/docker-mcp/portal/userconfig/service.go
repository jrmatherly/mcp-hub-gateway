package userconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/cache"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/executor"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/audit"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/security/crypto"
)

// userConfigService implements UserConfigService interface
type userConfigService struct {
	repo       UserConfigRepository
	executor   executor.Executor
	audit      audit.Logger
	cache      cache.Cache
	encryption crypto.Encryption
	masterKey  []byte
	tenantID   string
}

// CreateUserConfigService creates a new UserConfigService instance
func CreateUserConfigService(
	repo UserConfigRepository,
	exec executor.Executor,
	auditLogger audit.Logger,
	cacheStore cache.Cache,
	encryption crypto.Encryption,
) (UserConfigService, error) {
	// Generate a master key for this service instance
	masterKey, err := encryption.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate master key: %w", err)
	}

	return &userConfigService{
		repo:       repo,
		executor:   exec,
		audit:      auditLogger,
		cache:      cacheStore,
		encryption: encryption,
		masterKey:  masterKey,
		tenantID:   "default", // TODO: Get from context or config
	}, nil
}

// CreateConfig creates a new user configuration
func (s *userConfigService) CreateConfig(
	ctx context.Context,
	userID string,
	req *CreateConfigRequest,
) (*UserConfig, error) {
	// Validate request
	if err := s.validateCreateConfigRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create new config
	config := &UserConfig{
		ID:          uuid.New(),
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Type:        req.Type,
		Status:      ConfigStatusDraft,
		OwnerID:     uuid.MustParse(userID),
		TenantID:    s.tenantID,
		IsDefault:   req.IsDefault,
		IsActive:    false, // New configs start inactive
		Version:     "1.0.0",
		Settings:    req.Settings,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Save to repository
	if err := s.repo.CreateConfig(ctx, userID, config); err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	// Log audit event
	userUUID, err := uuid.Parse(userID)
	if err == nil {
		s.audit.LogSecurityEvent(
			ctx,
			userUUID,
			audit.EventTypeConfiguration,
			map[string]interface{}{
				"action":      "create",
				"resource":    "user_config",
				"resource_id": config.ID.String(),
				"config_name": config.Name,
				"config_type": config.Type,
			},
		)
	}

	// Invalidate list cache
	if err := s.invalidateListCache(ctx, userID); err != nil {
		// Log but don't fail
		fmt.Printf("Failed to invalidate list cache: %v\n", err)
	}

	return config, nil
}

// GetConfig retrieves a user configuration by ID
func (s *userConfigService) GetConfig(
	ctx context.Context,
	userID string,
	id uuid.UUID,
) (*UserConfig, error) {
	cacheKey := s.getConfigCacheKey(userID, id)

	// Try cache first
	if data, err := s.cache.Get(ctx, cacheKey); err == nil {
		var config UserConfig
		if err := json.Unmarshal(data, &config); err == nil {
			return &config, nil
		}
	}

	// Cache miss - get from repository
	config, err := s.repo.GetConfig(ctx, userID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	// Cache the result
	if data, err := json.Marshal(config); err == nil {
		if err := s.cache.Set(ctx, cacheKey, data, 15*time.Minute); err != nil {
			// Log but don't fail
			fmt.Printf("Failed to cache config: %v\n", err)
		}
	}

	return config, nil
}

// UpdateConfig updates an existing user configuration
func (s *userConfigService) UpdateConfig(
	ctx context.Context,
	userID string,
	id uuid.UUID,
	req *UpdateConfigRequest,
) (*UserConfig, error) {
	// Get existing config
	existingConfig, err := s.repo.GetConfig(ctx, userID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing config: %w", err)
	}

	// Update fields
	if req.DisplayName != nil && *req.DisplayName != "" {
		existingConfig.DisplayName = *req.DisplayName
	}
	if req.Description != nil && *req.Description != "" {
		existingConfig.Description = *req.Description
	}
	if req.Settings != nil {
		existingConfig.Settings = req.Settings
	}
	if req.IsActive != nil {
		existingConfig.IsActive = *req.IsActive
	}
	if req.Status != nil {
		existingConfig.Status = *req.Status
	}

	existingConfig.UpdatedAt = time.Now().UTC()

	// Validate updated config
	if validationErrors := s.ValidateConfig(ctx, existingConfig); len(validationErrors) > 0 {
		return nil, fmt.Errorf("validation failed: %v", validationErrors)
	}

	// Save to repository
	if err := s.repo.UpdateConfig(ctx, userID, existingConfig); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	// Log audit event
	userUUID, err := uuid.Parse(userID)
	if err == nil {
		s.audit.LogSecurityEvent(
			ctx,
			userUUID,
			audit.EventTypeConfiguration,
			map[string]interface{}{
				"action":      "update",
				"resource":    "user_config",
				"resource_id": id.String(),
				"config_name": existingConfig.Name,
				"changes":     req,
			},
		)
	}

	// Invalidate cache
	if err := s.invalidateConfigCache(ctx, userID, id); err != nil {
		fmt.Printf("Failed to invalidate config cache: %v\n", err)
	}

	return existingConfig, nil
}

// DeleteConfig deletes a user configuration
func (s *userConfigService) DeleteConfig(ctx context.Context, userID string, id uuid.UUID) error {
	// Check if config exists
	_, err := s.repo.GetConfig(ctx, userID, id)
	if err != nil {
		return fmt.Errorf("config not found: %w", err)
	}

	// Delete from repository
	if err := s.repo.DeleteConfig(ctx, userID, id); err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}

	// Log audit event
	userUUID, err := uuid.Parse(userID)
	if err == nil {
		s.audit.LogSecurityEvent(
			ctx,
			userUUID,
			audit.EventTypeConfiguration,
			map[string]interface{}{
				"action":      "delete",
				"resource":    "user_config",
				"resource_id": id.String(),
			},
		)
	}

	// Invalidate cache
	if err := s.invalidateConfigCache(ctx, userID, id); err != nil {
		fmt.Printf("Failed to invalidate config cache: %v\n", err)
	}

	return nil
}

// ListConfigs lists user configurations with filtering and pagination
func (s *userConfigService) ListConfigs(
	ctx context.Context,
	userID string,
	filter ConfigFilter,
) ([]*UserConfig, int64, error) {
	// Get configs from repository
	configs, err := s.repo.ListConfigs(ctx, userID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list configs: %w", err)
	}

	// Get total count
	total, err := s.repo.CountConfigs(ctx, userID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count configs: %w", err)
	}

	return configs, total, nil
}

// EnableServer enables an MCP server via CLI
func (s *userConfigService) EnableServer(
	ctx context.Context,
	userID string,
	serverName string,
) error {
	if err := s.validateServerName(serverName); err != nil {
		return fmt.Errorf("invalid server name: %w", err)
	}

	// Execute CLI command
	req := &executor.ExecutionRequest{
		Command: executor.CommandTypeServerEnable,
		Args:    []string{serverName},
	}
	result, err := s.executor.Execute(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to execute enable command: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("enable command failed: %s", result.Stdout)
	}

	// Log audit event
	userUUID, err := uuid.Parse(userID)
	if err == nil {
		s.audit.LogCommand(ctx, userUUID, "server enable", []string{serverName})
	}

	return nil
}

// DisableServer disables an MCP server via CLI
func (s *userConfigService) DisableServer(
	ctx context.Context,
	userID string,
	serverName string,
) error {
	if err := s.validateServerName(serverName); err != nil {
		return fmt.Errorf("invalid server name: %w", err)
	}

	// Execute CLI command
	req := &executor.ExecutionRequest{
		Command: executor.CommandTypeServerDisable,
		Args:    []string{serverName},
	}
	result, err := s.executor.Execute(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to execute disable command: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("disable command failed: %s", result.Stdout)
	}

	// Log audit event
	userUUID, err := uuid.Parse(userID)
	if err == nil {
		s.audit.LogCommand(ctx, userUUID, "server disable", []string{serverName})
	}

	return nil
}

// ConfigureServer configures an MCP server
func (s *userConfigService) ConfigureServer(
	ctx context.Context,
	userID string,
	req *ServerConfigRequest,
) error {
	if err := s.validateServerConfigRequest(req); err != nil {
		return fmt.Errorf("invalid server config request: %w", err)
	}

	// Get current server config
	serverConfig, err := s.repo.GetServerConfig(ctx, userID, req.ServerName)
	if err != nil {
		// Create new server config if not found
		serverConfig = &ServerConfig{
			ServerName:  req.ServerName,
			ServerID:    req.ServerName, // Compatibility alias
			DisplayName: req.ServerName, // Default display name
			Status:      "configured",
		}
		// Apply config updates
		s.applyServerConfigUpdates(serverConfig, req.Config)
	} else {
		// Apply config updates to existing config
		s.applyServerConfigUpdates(serverConfig, req.Config)
		serverConfig.Status = "configured"
	}

	// Save server config
	if err := s.repo.SaveServerConfig(ctx, userID, serverConfig); err != nil {
		return fmt.Errorf("failed to save server config: %w", err)
	}

	// Execute configuration via CLI if needed
	if req.Action != "" {
		if err := s.executeServerAction(ctx, userID, req.ServerName, req.Action); err != nil {
			return fmt.Errorf("failed to execute server action: %w", err)
		}
	}

	return nil
}

// GetServerStatus gets the status of an MCP server
func (s *userConfigService) GetServerStatus(
	ctx context.Context,
	userID string,
	serverName string,
) (*ServerStatus, error) {
	if err := s.validateServerName(serverName); err != nil {
		return nil, fmt.Errorf("invalid server name: %w", err)
	}

	// Execute CLI command to get server status
	req := &executor.ExecutionRequest{
		Command: executor.CommandTypeServerInspect,
		Args:    []string{serverName, "--json"},
	}
	result, err := s.executor.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute inspect command: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("inspect command failed: %s", result.Stdout)
	}

	// Parse JSON output
	var status ServerStatus
	if err := json.Unmarshal([]byte(result.Stdout), &status); err != nil {
		return nil, fmt.Errorf("failed to parse server status: %w", err)
	}

	return &status, nil
}

// ImportConfig imports configuration from external data
func (s *userConfigService) ImportConfig(
	ctx context.Context,
	userID string,
	req *ConfigImportRequest,
) (*UserConfig, error) {
	if err := s.validateImportRequest(req); err != nil {
		return nil, fmt.Errorf("invalid import request: %w", err)
	}

	// Parse import data
	var importData map[string]any
	if err := json.Unmarshal(req.Data, &importData); err != nil {
		return nil, fmt.Errorf("failed to parse import data: %w", err)
	}

	// Encrypt sensitive keys if specified
	if len(req.EncryptKeys) > 0 {
		if err := s.encryptSensitiveData(importData, req.EncryptKeys); err != nil {
			return nil, fmt.Errorf("failed to encrypt sensitive data: %w", err)
		}
	}

	// Create config from import data
	config := &UserConfig{
		ID:          uuid.New(),
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Type:        ConfigTypePersonal, // Default for imports
		Status:      ConfigStatusDraft,
		OwnerID:     uuid.MustParse(userID),
		TenantID:    s.tenantID,
		IsDefault:   false,
		IsActive:    false,
		Version:     "1.0.0",
		Settings:    importData,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Save to repository
	if err := s.repo.CreateConfig(ctx, userID, config); err != nil {
		return nil, fmt.Errorf("failed to create imported config: %w", err)
	}

	// Log audit event
	userUUID, err := uuid.Parse(userID)
	if err == nil {
		s.audit.LogSecurityEvent(
			ctx,
			userUUID,
			audit.EventTypeConfiguration,
			map[string]interface{}{
				"action":      "import",
				"resource":    "user_config",
				"resource_id": config.ID.String(),
				"config_name": config.Name,
				"data_size":   len(req.Data),
				"merge_mode":  req.MergeMode,
			},
		)
	}

	return config, nil
}

// ExportConfig exports configuration data
func (s *userConfigService) ExportConfig(
	ctx context.Context,
	userID string,
	req *ConfigExportRequest,
) ([]byte, error) {
	configs := make([]*UserConfig, 0)

	if len(req.ConfigIDs) > 0 {
		// Export specific configs
		for _, id := range req.ConfigIDs {
			config, err := s.repo.GetConfig(ctx, userID, id)
			if err != nil {
				return nil, fmt.Errorf("failed to get config %s: %w", id, err)
			}
			configs = append(configs, config)
		}
	} else {
		// Export all configs
		allConfigs, err := s.repo.ListConfigs(ctx, userID, ConfigFilter{})
		if err != nil {
			return nil, fmt.Errorf("failed to list configs: %w", err)
		}
		configs = allConfigs
	}

	// Create export data
	exportData := map[string]any{
		"version":     "1.0",
		"exported_at": time.Now().UTC(),
		"user_id":     userID,
		"configs":     configs,
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal export data: %w", err)
	}

	// Log audit event
	userUUID, err := uuid.Parse(userID)
	if err == nil {
		s.audit.LogSecurityEvent(
			ctx,
			userUUID,
			audit.EventTypeConfiguration,
			map[string]interface{}{
				"action":       "export",
				"resource":     "user_config",
				"resource_id":  "bulk",
				"config_count": len(configs),
				"data_size":    len(data),
			},
		)
	}

	return data, nil
}

// ExecuteConfigCommand executes a configuration-related CLI command
func (s *userConfigService) ExecuteConfigCommand(
	ctx context.Context,
	userID string,
	command string,
	args []string,
) (*CLIResult, error) {
	// Validate command for security
	if err := s.validateCommand(command, args); err != nil {
		return nil, fmt.Errorf("command validation failed: %w", err)
	}

	// Execute command
	commandType, filteredArgs := s.mapToCommandType(command, args)
	req := &executor.ExecutionRequest{
		Command: commandType,
		Args:    filteredArgs,
	}
	result, err := s.executor.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("command execution failed: %w", err)
	}

	// Convert to CLIResult
	cliResult := &CLIResult{
		Success:    result.Success,
		Output:     result.Stdout,
		ErrorMsg:   result.Stderr,
		ExitCode:   result.ExitCode,
		ExecutedAt: time.Now().UTC(),
	}

	// Log audit event
	userUUID, err := uuid.Parse(userID)
	if err == nil {
		s.audit.LogCommand(ctx, userUUID, command, args)
	}

	return cliResult, nil
}

// ValidateConfig validates a user configuration
func (s *userConfigService) ValidateConfig(
	ctx context.Context,
	config *UserConfig,
) []ValidationError {
	var errors []ValidationError

	// Validate required fields
	if config.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "name is required",
		})
	}

	// Validate name format
	if !s.isValidConfigName(config.Name) {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "name must contain only alphanumeric characters, hyphens, and underscores",
		})
	}

	// Validate type
	if !s.isValidConfigType(string(config.Type)) {
		errors = append(errors, ValidationError{
			Field:   "type",
			Message: "invalid configuration type",
		})
	}

	// Validate settings
	if config.Settings == nil {
		errors = append(errors, ValidationError{
			Field:   "settings",
			Message: "settings cannot be nil",
		})
	}

	// Validate owner ID
	if config.OwnerID == uuid.Nil {
		errors = append(errors, ValidationError{
			Field:   "owner_id",
			Message: "owner_id is required",
		})
	}

	return errors
}

// Helper methods

func (s *userConfigService) validateCreateConfigRequest(req *CreateConfigRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.Type == "" {
		return fmt.Errorf("type is required")
	}
	if !s.isValidConfigType(string(req.Type)) {
		return fmt.Errorf("invalid configuration type: %s", req.Type)
	}
	return nil
}

func (s *userConfigService) validateServerName(serverName string) error {
	if serverName == "" {
		return fmt.Errorf("server name is required")
	}
	// Basic validation for server name format
	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9-_]*[a-zA-Z0-9]$`, serverName); !matched {
		return fmt.Errorf("invalid server name format")
	}
	return nil
}

func (s *userConfigService) validateServerConfigRequest(req *ServerConfigRequest) error {
	if err := s.validateServerName(req.ServerName); err != nil {
		return err
	}
	if req.Config == nil {
		return fmt.Errorf("config is required")
	}
	return nil
}

func (s *userConfigService) validateImportRequest(req *ConfigImportRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(req.Data) == 0 {
		return fmt.Errorf("data is required")
	}
	return nil
}

func (s *userConfigService) validateCommand(command string, args []string) error {
	// Whitelist allowed commands
	allowedCommands := map[string]bool{
		"docker": true,
	}

	if !allowedCommands[command] {
		return fmt.Errorf("command not allowed: %s", command)
	}

	// Validate command format (no shell injection)
	if strings.ContainsAny(command, ";|&$`()") {
		return fmt.Errorf("invalid command format")
	}

	// Validate arguments
	for _, arg := range args {
		if strings.ContainsAny(arg, ";|&$`()") {
			return fmt.Errorf("invalid argument format: %s", arg)
		}
	}

	return nil
}

func (s *userConfigService) isValidConfigName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9-_]*[a-zA-Z0-9]$`, name)
	return matched
}

func (s *userConfigService) isValidConfigType(configType string) bool {
	validTypes := map[string]bool{
		string(ConfigTypeDefault):     true,
		string(ConfigTypePersonal):    true,
		string(ConfigTypeTeam):        true,
		string(ConfigTypeEnvironment): true,
		string(ConfigTypeProject):     true,
	}
	return validTypes[configType]
}

func (s *userConfigService) executeServerAction(
	ctx context.Context,
	_ string,
	serverName string,
	action ServerConfigAction,
) error {
	var command []string

	switch action {
	case ServerActionEnable:
		command = []string{"docker", "mcp", "server", "enable", serverName}
	case ServerActionDisable:
		command = []string{"docker", "mcp", "server", "disable", serverName}
	case ServerActionRestart:
		command = []string{"docker", "mcp", "server", "restart", serverName}
	case ServerActionStart:
		command = []string{"docker", "mcp", "server", "start", serverName}
	case ServerActionStop:
		command = []string{"docker", "mcp", "server", "stop", serverName}
	default:
		return fmt.Errorf("unsupported server action: %s", action)
	}

	commandType, filteredArgs := s.mapToCommandType(command[0], command[1:])
	req := &executor.ExecutionRequest{
		Command: commandType,
		Args:    filteredArgs,
	}
	result, err := s.executor.Execute(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to execute server action: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("server action failed: %s", result.Stdout)
	}

	return nil
}

func (s *userConfigService) encryptSensitiveData(data map[string]any, encryptKeys []string) error {
	for _, key := range encryptKeys {
		if value, exists := data[key]; exists {
			if strValue, ok := value.(string); ok {
				encrypted, err := s.encryption.Encrypt([]byte(strValue), s.masterKey)
				if err != nil {
					return fmt.Errorf("failed to encrypt key %s: %w", key, err)
				}
				// Convert EncryptedData to Base64 for storage
				encryptedBase64, err := encrypted.ToBase64()
				if err != nil {
					return fmt.Errorf("failed to encode encrypted data for key %s: %w", key, err)
				}
				data[key] = encryptedBase64
			}
		}
	}
	return nil
}

func (s *userConfigService) getConfigCacheKey(userID string, configID uuid.UUID) string {
	return fmt.Sprintf("userconfig:%s:%s", userID, configID.String())
}

func (s *userConfigService) invalidateConfigCache(
	ctx context.Context,
	userID string,
	configID uuid.UUID,
) error {
	cacheKey := s.getConfigCacheKey(userID, configID)
	if err := s.cache.Delete(ctx, cacheKey); err != nil {
		return err
	}
	return s.invalidateListCache(ctx, userID)
}

func (s *userConfigService) invalidateListCache(ctx context.Context, userID string) error {
	// Pattern to match all list cache keys for this user
	pattern := fmt.Sprintf("userconfig:list:%s:*", userID)
	_, err := s.cache.DeletePattern(ctx, pattern)
	return err
}

// mapToCommandType maps a command and args to the appropriate CommandType
func (s *userConfigService) mapToCommandType(
	command string,
	args []string,
) (executor.CommandType, []string) {
	if command == "docker" && len(args) >= 2 && args[0] == "mcp" {
		switch args[1] {
		case "server":
			if len(args) >= 3 {
				switch args[2] {
				case "list":
					return executor.CommandTypeServerList, args[3:]
				case "enable":
					return executor.CommandTypeServerEnable, args[3:]
				case "disable":
					return executor.CommandTypeServerDisable, args[3:]
				case "inspect":
					return executor.CommandTypeServerInspect, args[3:]
				case "status":
					return executor.CommandTypeServerStatus, args[3:]
				}
			}
		case "catalog":
			if len(args) >= 3 {
				switch args[2] {
				case "init":
					return executor.CommandTypeCatalogInit, args[3:]
				case "list":
					return executor.CommandTypeCatalogList, args[3:]
				case "show":
					return executor.CommandTypeCatalogShow, args[3:]
				case "sync":
					return executor.CommandTypeCatalogSync, args[3:]
				}
			}
		case "config":
			if len(args) >= 3 {
				switch args[2] {
				case "read":
					return executor.CommandTypeConfigRead, args[3:]
				case "write":
					return executor.CommandTypeConfigWrite, args[3:]
				}
			}
		case "version":
			return executor.CommandTypeVersion, args[2:]
		case "health":
			return executor.CommandTypeHealth, args[2:]
		}
	}

	// Default fallback - this might need adjustment based on your needs
	return executor.CommandType(command), args
}

// applyServerConfigUpdates applies updates from UpdateServerConfigRequest to ServerConfig
func (s *userConfigService) applyServerConfigUpdates(
	config *ServerConfig,
	updates *UpdateServerConfigRequest,
) {
	if updates == nil {
		return
	}

	if updates.DisplayName != nil {
		config.DisplayName = *updates.DisplayName
	}
	if updates.IsEnabled != nil {
		config.IsEnabled = *updates.IsEnabled
	}
	if updates.Command != nil {
		config.Command = updates.Command
	}
	if updates.Environment != nil {
		config.Environment = updates.Environment
	}
	if updates.Args != nil {
		config.Args = updates.Args
	}
	if updates.WorkingDir != nil {
		config.WorkingDir = *updates.WorkingDir
	}
	if updates.Image != nil {
		config.Image = *updates.Image
	}
	if updates.Transport != nil {
		config.Transport = *updates.Transport
	}
	if updates.Settings != nil {
		config.Settings = updates.Settings
		// Also set the compatibility alias
		config.Config = updates.Settings
	}

	config.UpdatedAt = time.Now().UTC()
}
