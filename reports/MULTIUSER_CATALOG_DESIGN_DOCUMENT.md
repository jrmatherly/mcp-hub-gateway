# Multi-User Catalog System Design Document

## Docker Desktop-Independent Architecture

**Document Version**: 1.0
**Date**: 2025-09-18
**Status**: Design Phase

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [System Architecture](#system-architecture)
3. [Core Components](#core-components)
4. [File-Based Catalog Management](#file-based-catalog-management)
5. [User Isolation & Multi-Tenancy](#user-isolation--multi-tenancy)
6. [Dynamic Port Allocation](#dynamic-port-allocation)
7. [Container Management](#container-management)
8. [Security Framework](#security-framework)
9. [Implementation Patterns](#implementation-patterns)
10. [Migration Strategy](#migration-strategy)
11. [API Specifications](#api-specifications)
12. [Deployment Architecture](#deployment-architecture)
13. [Testing & Validation](#testing--validation)

---

## Executive Summary

This design document outlines a Docker Desktop-independent multi-user catalog system for the MCP Portal that provides:

- **File-based catalog management** compatible with Docker Engine only
- **Per-user isolation** without requiring Docker Desktop features
- **Dynamic resource allocation** for containerized MCP servers
- **Environment variable-based configuration** for secrets management
- **Backward compatibility** with existing deployment architecture

### Key Design Principles

1. **Docker Engine Compatibility**: Works with Docker Engine without Desktop dependencies
2. **File-First Configuration**: Uses file system for catalog storage and distribution
3. **User Isolation**: Implements container-level isolation per user
4. **Minimal Changes**: Leverages existing Portal infrastructure with targeted enhancements
5. **Security by Design**: Comprehensive security model for multi-user environments

---

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      MCP Portal Frontend                        │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐    │
│  │  User Dashboard │ │ Catalog Manager │ │ Server Monitor  │    │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                                │
                    ┌───────────┼───────────┐
                    │    Portal Backend API │
                    └───────────┼───────────┘
                                │
    ┌───────────────────────────┼───────────────────────────┐
    │                File-Based Catalog System              │
    │  ┌─────────────────┐ ┌─────────────────┐ ┌──────────┐ │
    │  │   Base Catalog  │ │  User Catalogs  │ │ Overlays │ │
    │  │    (Admin)      │ │   (Per-User)    │ │ (Custom) │ │
    │  └─────────────────┘ └─────────────────┘ └──────────┘ │
    └───────────────────────────┼───────────────────────────┘
                                │
    ┌───────────────────────────┼───────────────────────────┐
    │            Container Orchestration Layer              │
    │  ┌─────────────────┐ ┌─────────────────┐ ┌──────────┐ │
    │  │ Port Allocator  │ │ Resource Mgr    │ │ Isolation│ │
    │  └─────────────────┘ └─────────────────┘ └──────────┘ │
    └───────────────────────────┼───────────────────────────┘
                                │
    ┌───────────────────────────┼───────────────────────────┐
    │              Docker Engine (No Desktop)               │
    │  ┌─────────────────┐ ┌─────────────────┐ ┌──────────┐ │
    │  │   User-1 Net    │ │   User-2 Net    │ │   ...    │ │
    │  │ ┌─────┐┌─────┐  │ │ ┌─────┐┌─────┐  │ │          │ │
    │  │ │Srv1 ││ Srv2│  │ │ │Srv3 ││ Srv4│  │ │          │ │
    │  │ └─────┘└─────┘  │ │ └─────┘└─────┘  │ │          │ │
    │  └─────────────────┘ └─────────────────┘ └──────────┘ │
    └───────────────────────────────────────────────────────┘
```

### Component Interaction Flow

```
[Admin] → Upload Base Catalog → [Catalog Manager] → Generate User Files
                                        ↓
[User] → Request Server List → [Portal API] → Merge Base + User Catalogs
                                        ↓
[Portal] → Enable Server → [Container Mgr] → Start with User-Specific Config
                                        ↓
[Gateway] → Load Merged Catalog → [Docker Engine] → Isolated Container
```

---

## Core Components

### 1. Catalog Management Service

**Purpose**: Manages catalog files and user-specific configurations

```go
type CatalogManager struct {
    baseDir        string
    userCatalogDir string
    overlayDir     string
    merger         *CatalogMerger
    fileWatcher    *fsnotify.Watcher
    cache          *CatalogCache
}

type CatalogFile struct {
    Version     string                 `json:"version"`
    Source      string                 `json:"source"`      // "base", "user", "overlay"
    UserID      *string               `json:"user_id,omitempty"`
    Servers     []ServerDefinition    `json:"servers"`
    Metadata    CatalogMetadata       `json:"metadata"`
    CreatedAt   time.Time            `json:"created_at"`
    UpdatedAt   time.Time            `json:"updated_at"`
}

type ServerDefinition struct {
    ID              string            `json:"id"`
    Name            string            `json:"name"`
    DisplayName     string            `json:"display_name"`
    Description     string            `json:"description"`
    Image           string            `json:"image"`
    Tag             string            `json:"tag"`
    Category        string            `json:"category"`
    Transport       string            `json:"transport"`
    Environment     map[string]string `json:"environment"`
    Volumes         []VolumeMount     `json:"volumes"`
    Ports           []PortMapping     `json:"ports"`
    Resources       ResourceLimits    `json:"resources"`
    HealthCheck     HealthCheck       `json:"health_check"`
    UserConfigurable bool             `json:"user_configurable"`
    RequiredSecrets []string          `json:"required_secrets"`
}
```

### 2. User Isolation Manager

**Purpose**: Provides container-level isolation between users

```go
type UserIsolationManager struct {
    portAllocator  *DynamicPortAllocator
    networkManager *UserNetworkManager
    resourceMgr    *ResourceManager
    containerMgr   *UserContainerManager
}

type UserContext struct {
    UserID          string            `json:"user_id"`
    NetworkName     string            `json:"network_name"`
    PortRange       PortRange         `json:"port_range"`
    ContainerPrefix string            `json:"container_prefix"`
    VolumePrefix    string            `json:"volume_prefix"`
    ResourceLimits  ResourceLimits    `json:"resource_limits"`
    Environment     map[string]string `json:"environment"`
}

type PortRange struct {
    Start int `json:"start"`
    End   int `json:"end"`
    Used  []int `json:"used"`
}
```

### 3. Environment-Based Secret Manager

**Purpose**: Manages secrets through environment variables and files

```go
type EnvSecretManager struct {
    secretsDir     string
    encryptionKey  []byte
    userSecrets    map[string]*UserSecrets
    templates      map[string]*SecretTemplate
}

type UserSecrets struct {
    UserID    string                 `json:"user_id"`
    Secrets   map[string]SecretValue `json:"secrets"`
    UpdatedAt time.Time             `json:"updated_at"`
}

type SecretValue struct {
    Value     string    `json:"value"`
    Encrypted bool      `json:"encrypted"`
    CreatedAt time.Time `json:"created_at"`
    ExpiresAt *time.Time `json:"expires_at,omitempty"`
}
```

---

## File-Based Catalog Management

### Directory Structure

```
/app/data/catalogs/
├── base/                           # Admin-managed base catalogs
│   ├── official-v1.2.3.json      # Official server catalog
│   ├── community-v1.1.0.json     # Community server catalog
│   └── enterprise-v2.0.0.json    # Enterprise server catalog
├── users/                         # User-specific catalogs
│   ├── {user-id}/
│   │   ├── catalog.json           # User's merged catalog
│   │   ├── enabled-servers.json   # User's enabled servers
│   │   ├── custom-servers.json    # User's custom server definitions
│   │   └── secrets.env            # User's encrypted secrets
│   └── {user-id}/...
├── overlays/                      # Overlay configurations
│   ├── development.json           # Development environment overlay
│   ├── staging.json               # Staging environment overlay
│   └── production.json            # Production environment overlay
└── cache/                         # Generated cache files
    ├── merged-{user-id}.json      # Cached merged catalogs
    └── validation-{checksum}.json # Validation results cache
```

### Base Catalog Management

**Admin Upload Process**:

```yaml
# Base catalog upload workflow
admin_upload: 1. Validate catalog format and security
  2. Version and backup existing catalogs
  3. Store new base catalog with metadata
  4. Trigger user catalog regeneration
  5. Notify users of catalog updates
  6. Update cache and invalidate old data

# Example base catalog structure
base_catalog:
  version: "1.2.3"
  source: "base"
  metadata:
    name: "Official MCP Servers"
    description: "Curated collection of official MCP servers"
    maintainer: "admin@company.com"
    checksum: "sha256:abc123..."
  servers:
    - id: "github"
      name: "github"
      display_name: "GitHub Integration"
      image: "mcp/github"
      tag: "v1.2.0"
      category: "development"
      user_configurable: true
      required_secrets: ["GITHUB_TOKEN"]
```

### User Catalog Generation

**Catalog Merging Algorithm**:

```go
func (cm *CatalogManager) GenerateUserCatalog(userID string) (*CatalogFile, error) {
    // 1. Load base catalogs
    baseCatalogs, err := cm.loadBaseCatalogs()
    if err != nil {
        return nil, err
    }

    // 2. Load user's custom servers
    userCustom, err := cm.loadUserCustomServers(userID)
    if err != nil {
        return nil, err
    }

    // 3. Load applicable overlays
    overlays, err := cm.loadOverlays(userID)
    if err != nil {
        return nil, err
    }

    // 4. Merge in priority order: base → overlays → user custom
    merged := &CatalogFile{
        Version:   cm.generateVersion(),
        Source:    "merged",
        UserID:    &userID,
        Servers:   make([]ServerDefinition, 0),
        Metadata:  cm.generateMetadata(userID),
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    // Merge servers with conflict resolution
    serverMap := make(map[string]ServerDefinition)

    // Add base servers
    for _, catalog := range baseCatalogs {
        for _, server := range catalog.Servers {
            serverMap[server.ID] = server
        }
    }

    // Apply overlays
    for _, overlay := range overlays {
        for _, server := range overlay.Servers {
            if existing, exists := serverMap[server.ID]; exists {
                serverMap[server.ID] = cm.mergeServerDefinitions(existing, server)
            } else {
                serverMap[server.ID] = server
            }
        }
    }

    // Add user custom servers
    for _, server := range userCustom.Servers {
        serverMap[server.ID] = server
    }

    // Convert map to slice
    for _, server := range serverMap {
        merged.Servers = append(merged.Servers, server)
    }

    // 5. Validate merged catalog
    if err := cm.validateCatalog(merged); err != nil {
        return nil, err
    }

    // 6. Cache result
    if err := cm.cacheCatalog(userID, merged); err != nil {
        return nil, err
    }

    return merged, nil
}
```

### File System Monitoring

**Catalog Change Detection**:

```go
type CatalogWatcher struct {
    watcher    *fsnotify.Watcher
    catalogMgr *CatalogManager
    eventChan  chan CatalogEvent
}

func (cw *CatalogWatcher) Start() error {
    go func() {
        for {
            select {
            case event, ok := <-cw.watcher.Events:
                if !ok {
                    return
                }

                if event.Op&fsnotify.Write == fsnotify.Write {
                    cw.handleCatalogUpdate(event.Name)
                }

            case err, ok := <-cw.watcher.Errors:
                if !ok {
                    return
                }
                log.Printf("Catalog watcher error: %v", err)
            }
        }
    }()

    return nil
}

func (cw *CatalogWatcher) handleCatalogUpdate(filename string) {
    // Determine if this is a base catalog update
    if strings.Contains(filename, "/base/") {
        // Regenerate all user catalogs
        cw.regenerateAllUserCatalogs()
    } else if strings.Contains(filename, "/users/") {
        // Extract user ID and regenerate specific user catalog
        userID := cw.extractUserIDFromPath(filename)
        cw.regenerateUserCatalog(userID)
    }
}
```

---

## User Isolation & Multi-Tenancy

### Container Naming Strategy

**Naming Convention**:

```yaml
container_naming:
  pattern: "mcp-{user-id}-{server-id}-{instance-id}"
  examples:
    - "mcp-user123-github-001"
    - "mcp-user456-slack-001"
    - "mcp-admin-monitoring-001"

  validation:
    user_id: "^[a-zA-Z0-9\\-]{1,32}$"
    server_id: "^[a-zA-Z0-9\\-_]{1,32}$"
    instance_id: "^[0-9]{3}$"

  network_naming:
    pattern: "mcp-net-{user-id}"
    examples:
      - "mcp-net-user123"
      - "mcp-net-admin"
```

### Network Isolation Implementation

**User-Specific Networks**:

```go
type UserNetworkManager struct {
    docker      *client.Client
    networks    sync.Map // userID -> *NetworkInfo
    ipamDriver  string
    subnetPool  *SubnetPool
}

type NetworkInfo struct {
    UserID      string    `json:"user_id"`
    NetworkID   string    `json:"network_id"`
    NetworkName string    `json:"network_name"`
    Subnet      string    `json:"subnet"`
    Gateway     string    `json:"gateway"`
    CreatedAt   time.Time `json:"created_at"`
}

func (unm *UserNetworkManager) CreateUserNetwork(userID string) (*NetworkInfo, error) {
    networkName := fmt.Sprintf("mcp-net-%s", userID)

    // Allocate subnet for user
    subnet, gateway, err := unm.subnetPool.AllocateSubnet(userID)
    if err != nil {
        return nil, err
    }

    // Create Docker network
    networkResp, err := unm.docker.NetworkCreate(context.Background(), networkName, types.NetworkCreate{
        Driver: "bridge",
        IPAM: &network.IPAM{
            Driver: unm.ipamDriver,
            Config: []network.IPAMConfig{
                {
                    Subnet:  subnet,
                    Gateway: gateway,
                },
            },
        },
        Options: map[string]string{
            "com.docker.network.bridge.name":          fmt.Sprintf("br-mcp-%s", userID[:8]),
            "com.docker.network.bridge.enable_icc":    "true",
            "com.docker.network.bridge.enable_ip_masquerade": "true",
        },
        Labels: map[string]string{
            "mcp.portal.user_id":     userID,
            "mcp.portal.network_type": "user_isolation",
            "mcp.portal.created_by":   "portal",
        },
    })

    if err != nil {
        unm.subnetPool.ReleaseSubnet(userID)
        return nil, err
    }

    networkInfo := &NetworkInfo{
        UserID:      userID,
        NetworkID:   networkResp.ID,
        NetworkName: networkName,
        Subnet:      subnet,
        Gateway:     gateway,
        CreatedAt:   time.Now(),
    }

    unm.networks.Store(userID, networkInfo)
    return networkInfo, nil
}
```

### Resource Isolation

**Per-User Resource Limits**:

```go
type ResourceManager struct {
    userLimits    map[string]*UserResourceLimits
    globalLimits  *GlobalResourceLimits
    monitor       *ResourceMonitor
}

type UserResourceLimits struct {
    MaxContainers int64  `json:"max_containers"`
    MaxMemoryMB   int64  `json:"max_memory_mb"`
    MaxCPUShares  int64  `json:"max_cpu_shares"`
    MaxDiskMB     int64  `json:"max_disk_mb"`
    MaxNetworkMbps int64 `json:"max_network_mbps"`
    MaxPorts      int    `json:"max_ports"`
}

func (rm *ResourceManager) ValidateResourceRequest(userID string, request *ResourceRequest) error {
    userLimits := rm.getUserLimits(userID)
    currentUsage := rm.monitor.GetCurrentUsage(userID)

    // Check container count limit
    if currentUsage.Containers+1 > userLimits.MaxContainers {
        return fmt.Errorf("container limit exceeded: %d/%d",
            currentUsage.Containers+1, userLimits.MaxContainers)
    }

    // Check memory limit
    if currentUsage.MemoryMB+request.MemoryMB > userLimits.MaxMemoryMB {
        return fmt.Errorf("memory limit exceeded: %dMB/%dMB",
            currentUsage.MemoryMB+request.MemoryMB, userLimits.MaxMemoryMB)
    }

    // Check CPU limit
    if currentUsage.CPUShares+request.CPUShares > userLimits.MaxCPUShares {
        return fmt.Errorf("CPU limit exceeded: %d/%d",
            currentUsage.CPUShares+request.CPUShares, userLimits.MaxCPUShares)
    }

    return nil
}
```

---

## Dynamic Port Allocation

### Port Range Management

**Port Allocation Strategy**:

```go
type DynamicPortAllocator struct {
    globalRange    PortRange       // 20000-29999 (10,000 ports)
    userRanges     map[string]PortRange
    allocations    sync.Map        // port -> allocation info
    portSize       int             // ports per user (default: 100)
    reservedPorts  map[int]bool    // system reserved ports
}

type PortAllocation struct {
    Port        int       `json:"port"`
    UserID      string    `json:"user_id"`
    ServerID    string    `json:"server_id"`
    ContainerID string    `json:"container_id"`
    AllocatedAt time.Time `json:"allocated_at"`
    LastUsed    time.Time `json:"last_used"`
}

func (dpa *DynamicPortAllocator) AllocatePortForUser(userID, serverID string) (int, error) {
    userRange, exists := dpa.userRanges[userID]
    if !exists {
        // Allocate new range for user
        var err error
        userRange, err = dpa.allocateUserRange(userID)
        if err != nil {
            return 0, err
        }
    }

    // Find available port in user's range
    for port := userRange.Start; port <= userRange.End; port++ {
        if !dpa.isPortAllocated(port) {
            allocation := &PortAllocation{
                Port:        port,
                UserID:      userID,
                ServerID:    serverID,
                AllocatedAt: time.Now(),
                LastUsed:    time.Now(),
            }

            dpa.allocations.Store(port, allocation)
            return port, nil
        }
    }

    return 0, fmt.Errorf("no available ports in user range %d-%d",
        userRange.Start, userRange.End)
}

func (dpa *DynamicPortAllocator) allocateUserRange(userID string) (PortRange, error) {
    // Calculate next available range
    nextStart := dpa.globalRange.Start
    for _, userRange := range dpa.userRanges {
        if userRange.End >= nextStart {
            nextStart = userRange.End + 1
        }
    }

    if nextStart+dpa.portSize > dpa.globalRange.End {
        return PortRange{}, fmt.Errorf("no available port ranges")
    }

    userRange := PortRange{
        Start: nextStart,
        End:   nextStart + dpa.portSize - 1,
        Used:  make([]int, 0),
    }

    dpa.userRanges[userID] = userRange
    return userRange, nil
}
```

### Health Check Port Management

**Dedicated Health Check Ports**:

```go
type HealthCheckManager struct {
    basePort       int              // 30000
    userHealthPorts map[string]int   // userID -> health check port
    portAllocator  *DynamicPortAllocator
}

func (hcm *HealthCheckManager) GetHealthCheckPort(userID string) (int, error) {
    if port, exists := hcm.userHealthPorts[userID]; exists {
        return port, nil
    }

    // Allocate dedicated health check port for user
    port := hcm.basePort + len(hcm.userHealthPorts)
    hcm.userHealthPorts[userID] = port

    return port, nil
}

func (hcm *HealthCheckManager) CreateHealthCheckConfig(userID string, servers []string) *HealthCheckConfig {
    port, _ := hcm.GetHealthCheckPort(userID)

    return &HealthCheckConfig{
        Port: port,
        Endpoints: hcm.generateHealthEndpoints(userID, servers),
        Interval:  30 * time.Second,
        Timeout:   5 * time.Second,
        Retries:   3,
    }
}
```

---

## Container Management

### User-Specific Container Lifecycle

**Container Creation with User Context**:

```go
type UserContainerManager struct {
    docker          *client.Client
    catalogMgr      *CatalogManager
    secretMgr       *EnvSecretManager
    networkMgr      *UserNetworkManager
    portAllocator   *DynamicPortAllocator
    resourceMgr     *ResourceManager
}

func (ucm *UserContainerManager) CreateServerContainer(ctx context.Context,
    userID, serverID string, config *ServerConfig) (*container.CreateResponse, error) {

    // 1. Get user context
    userCtx, err := ucm.getUserContext(userID)
    if err != nil {
        return nil, err
    }

    // 2. Load server definition from user's catalog
    serverDef, err := ucm.catalogMgr.GetServerDefinition(userID, serverID)
    if err != nil {
        return nil, err
    }

    // 3. Allocate resources
    port, err := ucm.portAllocator.AllocatePortForUser(userID, serverID)
    if err != nil {
        return nil, err
    }

    // 4. Prepare container configuration
    containerConfig := &container.Config{
        Image: fmt.Sprintf("%s:%s", serverDef.Image, serverDef.Tag),
        Env:   ucm.buildEnvironment(userCtx, serverDef, config),
        Labels: map[string]string{
            "mcp.portal.user_id":    userID,
            "mcp.portal.server_id":  serverID,
            "mcp.portal.managed_by": "portal",
            "mcp.portal.created_at": time.Now().Format(time.RFC3339),
        },
        ExposedPorts: nat.PortSet{
            nat.Port(fmt.Sprintf("%d/tcp", port)): {},
        },
        WorkingDir: "/app",
        User:       "1000:1000", // Non-root user
    }

    hostConfig := &container.HostConfig{
        NetworkMode: container.NetworkMode(userCtx.NetworkName),
        Resources: container.Resources{
            Memory:     serverDef.Resources.MemoryBytes,
            CPUShares:  serverDef.Resources.CPUShares,
            PidsLimit:  &serverDef.Resources.PidsLimit,
        },
        PortBindings: nat.PortMap{
            nat.Port(fmt.Sprintf("%d/tcp", port)): []nat.PortBinding{
                {HostPort: fmt.Sprintf("%d", port)},
            },
        },
        RestartPolicy: container.RestartPolicy{
            Name:              "unless-stopped",
            MaximumRetryCount: 3,
        },
        SecurityOpt: []string{
            "no-new-privileges:true",
            "seccomp:unconfined", // Required for some MCP servers
        },
        ReadonlyRootfs: false, // Some MCP servers need write access
        Tmpfs: map[string]string{
            "/tmp": "rw,noexec,nosuid,size=100m",
        },
    }

    // 5. Add volume mounts
    hostConfig.Binds = ucm.buildVolumeMounts(userCtx, serverDef, config)

    // 6. Create container
    containerName := fmt.Sprintf("mcp-%s-%s-001", userID, serverID)

    resp, err := ucm.docker.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName)
    if err != nil {
        ucm.portAllocator.ReleasePort(port)
        return nil, err
    }

    // 7. Connect to user network
    if err := ucm.connectToUserNetwork(ctx, resp.ID, userCtx); err != nil {
        ucm.docker.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
        ucm.portAllocator.ReleasePort(port)
        return nil, err
    }

    return &resp, nil
}
```

### Environment Variable Management

**User-Specific Environment Building**:

```go
func (ucm *UserContainerManager) buildEnvironment(userCtx *UserContext,
    serverDef *ServerDefinition, config *ServerConfig) []string {

    env := make([]string, 0)

    // 1. Base server environment
    for key, value := range serverDef.Environment {
        env = append(env, fmt.Sprintf("%s=%s", key, value))
    }

    // 2. User-specific environment
    for key, value := range userCtx.Environment {
        env = append(env, fmt.Sprintf("%s=%s", key, value))
    }

    // 3. Portal-specific environment
    env = append(env, []string{
        fmt.Sprintf("MCP_USER_ID=%s", userCtx.UserID),
        fmt.Sprintf("MCP_SERVER_ID=%s", serverDef.ID),
        fmt.Sprintf("MCP_PORTAL_VERSION=%s", ucm.getPortalVersion()),
        fmt.Sprintf("MCP_NETWORK_NAME=%s", userCtx.NetworkName),
        "MCP_TRANSPORT=stdio", // Default transport
    }...)

    // 4. User secrets (if configured)
    secrets, err := ucm.secretMgr.GetUserSecrets(userCtx.UserID)
    if err == nil {
        for _, secretName := range serverDef.RequiredSecrets {
            if secretValue, exists := secrets.Secrets[secretName]; exists {
                env = append(env, fmt.Sprintf("%s=%s", secretName, secretValue.Value))
            }
        }
    }

    // 5. User configuration overrides
    if config != nil {
        for key, value := range config.Environment {
            env = append(env, fmt.Sprintf("%s=%s", key, value))
        }
    }

    return env
}
```

### Volume Management

**User-Isolated Volume Mounts**:

```go
func (ucm *UserContainerManager) buildVolumeMounts(userCtx *UserContext,
    serverDef *ServerDefinition, config *ServerConfig) []string {

    binds := make([]string, 0)

    // 1. User data directory
    userDataDir := fmt.Sprintf("/app/data/users/%s", userCtx.UserID)
    os.MkdirAll(userDataDir, 0755)
    binds = append(binds, fmt.Sprintf("%s:/data:rw", userDataDir))

    // 2. User secrets directory (read-only)
    userSecretsDir := fmt.Sprintf("/app/data/secrets/%s", userCtx.UserID)
    if _, err := os.Stat(userSecretsDir); err == nil {
        binds = append(binds, fmt.Sprintf("%s:/secrets:ro", userSecretsDir))
    }

    // 3. User cache directory
    userCacheDir := fmt.Sprintf("/app/data/cache/%s", userCtx.UserID)
    os.MkdirAll(userCacheDir, 0755)
    binds = append(binds, fmt.Sprintf("%s:/cache:rw", userCacheDir))

    // 4. Server-specific volumes
    for _, volume := range serverDef.Volumes {
        hostPath := ucm.resolveUserPath(userCtx.UserID, volume.HostPath)
        containerPath := volume.ContainerPath
        mode := volume.Mode // "ro" or "rw"

        os.MkdirAll(hostPath, 0755)
        binds = append(binds, fmt.Sprintf("%s:%s:%s", hostPath, containerPath, mode))
    }

    // 5. Configuration-specific volumes
    if config != nil {
        for _, volume := range config.Volumes {
            hostPath := ucm.resolveUserPath(userCtx.UserID, volume.HostPath)
            binds = append(binds, fmt.Sprintf("%s:%s:%s",
                hostPath, volume.ContainerPath, volume.Mode))
        }
    }

    return binds
}

func (ucm *UserContainerManager) resolveUserPath(userID, path string) string {
    // Replace placeholders with user-specific paths
    replacements := map[string]string{
        "{USER_ID}":     userID,
        "{USER_DATA}":   fmt.Sprintf("/app/data/users/%s", userID),
        "{USER_CACHE}":  fmt.Sprintf("/app/data/cache/%s", userID),
        "{USER_CONFIG}": fmt.Sprintf("/app/data/config/%s", userID),
    }

    result := path
    for placeholder, value := range replacements {
        result = strings.ReplaceAll(result, placeholder, value)
    }

    // Ensure absolute path
    if !strings.HasPrefix(result, "/") {
        result = fmt.Sprintf("/app/data/users/%s/%s", userID, result)
    }

    return result
}
```

---

## Security Framework

### File Permission Management

**Secure File System Access**:

```go
type SecureFileManager struct {
    baseDir      string
    userMapping  map[string]int // userID -> UID
    groupMapping map[string]int // userID -> GID
    permissions  *PermissionMatrix
}

type PermissionMatrix struct {
    UserCatalogs    os.FileMode // 0644 - read-only for users
    UserSecrets     os.FileMode // 0600 - owner only
    UserData        os.FileMode // 0755 - full access for user
    BaseCatalogs    os.FileMode // 0444 - read-only for all
    SystemConfig    os.FileMode // 0640 - admin only
}

func (sfm *SecureFileManager) CreateUserDirectory(userID string) error {
    userDir := filepath.Join(sfm.baseDir, "users", userID)
    uid := sfm.getUserUID(userID)
    gid := sfm.getUserGID(userID)

    // Create directory structure
    dirs := []string{
        userDir,
        filepath.Join(userDir, "data"),
        filepath.Join(userDir, "cache"),
        filepath.Join(userDir, "config"),
        filepath.Join(userDir, "secrets"),
        filepath.Join(userDir, "logs"),
    }

    for _, dir := range dirs {
        if err := os.MkdirAll(dir, sfm.permissions.UserData); err != nil {
            return err
        }

        if err := os.Chown(dir, uid, gid); err != nil {
            return err
        }
    }

    // Set strict permissions on secrets directory
    secretsDir := filepath.Join(userDir, "secrets")
    if err := os.Chmod(secretsDir, sfm.permissions.UserSecrets); err != nil {
        return err
    }

    return nil
}

func (sfm *SecureFileManager) WriteUserCatalog(userID string, catalog *CatalogFile) error {
    catalogPath := filepath.Join(sfm.baseDir, "users", userID, "catalog.json")

    // Marshal catalog to JSON
    data, err := json.MarshalIndent(catalog, "", "  ")
    if err != nil {
        return err
    }

    // Write with secure permissions
    if err := os.WriteFile(catalogPath, data, sfm.permissions.UserCatalogs); err != nil {
        return err
    }

    // Set ownership
    uid := sfm.getUserUID(userID)
    gid := sfm.getUserGID(userID)
    return os.Chown(catalogPath, uid, gid)
}
```

### Secret Encryption

**AES-256-GCM Secret Encryption**:

```go
type SecretEncryption struct {
    masterKey []byte
    gcm       cipher.AEAD
}

func NewSecretEncryption(masterKey []byte) (*SecretEncryption, error) {
    if len(masterKey) != 32 {
        return nil, fmt.Errorf("master key must be 32 bytes")
    }

    block, err := aes.NewCipher(masterKey)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    return &SecretEncryption{
        masterKey: masterKey,
        gcm:       gcm,
    }, nil
}

func (se *SecretEncryption) EncryptSecret(userID, secretName, value string) (string, error) {
    // Generate unique nonce for this encryption
    nonce := make([]byte, se.gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }

    // Additional authenticated data includes user ID and secret name
    aad := []byte(fmt.Sprintf("%s:%s", userID, secretName))

    // Encrypt the secret value
    ciphertext := se.gcm.Seal(nonce, nonce, []byte(value), aad)

    // Encode as base64 for storage
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (se *SecretEncryption) DecryptSecret(userID, secretName, encryptedValue string) (string, error) {
    // Decode from base64
    ciphertext, err := base64.StdEncoding.DecodeString(encryptedValue)
    if err != nil {
        return "", err
    }

    // Extract nonce
    nonceSize := se.gcm.NonceSize()
    if len(ciphertext) < nonceSize {
        return "", fmt.Errorf("ciphertext too short")
    }

    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

    // Additional authenticated data
    aad := []byte(fmt.Sprintf("%s:%s", userID, secretName))

    // Decrypt
    plaintext, err := se.gcm.Open(nil, nonce, ciphertext, aad)
    if err != nil {
        return "", err
    }

    return string(plaintext), nil
}
```

### Container Security

**Security Constraints for User Containers**:

```go
type ContainerSecurityPolicy struct {
    AllowedImages      []string          `json:"allowed_images"`
    BlockedCapabilities []string         `json:"blocked_capabilities"`
    RequiredSecurityOpts []string        `json:"required_security_opts"`
    ResourceLimits     ResourceLimits    `json:"resource_limits"`
    NetworkRestrictions NetworkPolicy    `json:"network_restrictions"`
    VolumeRestrictions VolumePolicy      `json:"volume_restrictions"`
}

func (csp *ContainerSecurityPolicy) ValidateContainerConfig(config *container.Config,
    hostConfig *container.HostConfig) error {

    // 1. Validate image
    if !csp.isImageAllowed(config.Image) {
        return fmt.Errorf("image not allowed: %s", config.Image)
    }

    // 2. Ensure non-root user
    if config.User == "" || config.User == "root" || config.User == "0" {
        return fmt.Errorf("containers must run as non-root user")
    }

    // 3. Check security options
    if !csp.hasRequiredSecurityOpts(hostConfig.SecurityOpt) {
        return fmt.Errorf("missing required security options")
    }

    // 4. Validate resource limits
    if err := csp.validateResourceLimits(hostConfig.Resources); err != nil {
        return err
    }

    // 5. Check network configuration
    if err := csp.validateNetworkConfig(hostConfig.NetworkMode); err != nil {
        return err
    }

    // 6. Validate volume mounts
    if err := csp.validateVolumeMounts(hostConfig.Binds); err != nil {
        return err
    }

    return nil
}

func (csp *ContainerSecurityPolicy) isImageAllowed(image string) bool {
    for _, allowed := range csp.AllowedImages {
        if matched, _ := filepath.Match(allowed, image); matched {
            return true
        }
    }
    return false
}
```

### Audit Logging

**Comprehensive Audit Trail**:

```go
type SecurityAuditLogger struct {
    logger     *logrus.Logger
    auditFile  *os.File
    encryptor  *SecretEncryption
    formatter  *AuditFormatter
}

type AuditEvent struct {
    Timestamp   time.Time              `json:"timestamp"`
    EventType   string                 `json:"event_type"`
    UserID      string                 `json:"user_id"`
    Action      string                 `json:"action"`
    Resource    string                 `json:"resource"`
    Result      string                 `json:"result"`
    Details     map[string]interface{} `json:"details"`
    IPAddress   string                 `json:"ip_address"`
    UserAgent   string                 `json:"user_agent"`
    SessionID   string                 `json:"session_id"`
    Severity    string                 `json:"severity"`
}

func (sal *SecurityAuditLogger) LogCatalogAccess(userID, action, catalogName string,
    result string, details map[string]interface{}) {

    event := &AuditEvent{
        Timestamp: time.Now(),
        EventType: "CATALOG_ACCESS",
        UserID:    userID,
        Action:    action,
        Resource:  catalogName,
        Result:    result,
        Details:   details,
        Severity:  sal.calculateSeverity(action, result),
    }

    sal.writeAuditEvent(event)
}

func (sal *SecurityAuditLogger) LogContainerOperation(userID, containerID, operation string,
    result string, details map[string]interface{}) {

    event := &AuditEvent{
        Timestamp: time.Now(),
        EventType: "CONTAINER_OPERATION",
        UserID:    userID,
        Action:    operation,
        Resource:  containerID,
        Result:    result,
        Details:   details,
        Severity:  sal.calculateContainerSeverity(operation, result),
    }

    sal.writeAuditEvent(event)
}
```

---

## Implementation Patterns

### Configuration Template System

**Dynamic Configuration Generation**:

```yaml
# Server configuration template
server_template:
  name: "{{.ServerID}}"
  image: "{{.Image}}:{{.Tag}}"
  environment:
    MCP_USER_ID: "{{.UserID}}"
    MCP_SERVER_PORT: "{{.Port}}"
    MCP_TRANSPORT: "{{.Transport}}"
    {{range .Secrets}}
    {{.Name}}: "{{.Value}}"
    {{end}}
  volumes:
    - "{{.UserDataDir}}:/data:rw"
    - "{{.UserCacheDir}}:/cache:rw"
    {{range .CustomVolumes}}
    - "{{.HostPath}}:{{.ContainerPath}}:{{.Mode}}"
    {{end}}
  networks:
    - "{{.UserNetwork}}"
  resources:
    memory: "{{.MemoryLimit}}"
    cpu_shares: {{.CPUShares}}
    pids_limit: {{.PidsLimit}}
  labels:
    mcp.portal.user_id: "{{.UserID}}"
    mcp.portal.server_id: "{{.ServerID}}"
    mcp.portal.version: "{{.Version}}"
```

### CLI Integration Enhancements

**Multi-User CLI Command Generation**:

```go
type MultiUserCLIExecutor struct {
    baseCLI      string
    userContexts map[string]*UserContext
    cmdBuilder   *CommandBuilder
    validator    *ParameterValidator
}

func (muce *MultiUserCLIExecutor) ExecuteUserCommand(userID string,
    command *Command) (*CommandResult, error) {

    // 1. Get user context
    userCtx, exists := muce.userContexts[userID]
    if !exists {
        return nil, fmt.Errorf("user context not found: %s", userID)
    }

    // 2. Build user-specific command
    userCmd, err := muce.buildUserCommand(userCtx, command)
    if err != nil {
        return nil, err
    }

    // 3. Validate command parameters
    if err := muce.validator.ValidateCommand(userCmd); err != nil {
        return nil, err
    }

    // 4. Execute with user isolation
    result, err := muce.executeWithIsolation(userCtx, userCmd)
    if err != nil {
        return nil, err
    }

    // 5. Post-process result for user context
    return muce.processUserResult(userCtx, result), nil
}

func (muce *MultiUserCLIExecutor) buildUserCommand(userCtx *UserContext,
    command *Command) (*Command, error) {

    userCmd := &Command{
        BaseCommand: command.BaseCommand,
        Parameters:  make(map[string]interface{}),
        Flags:       make(map[string]interface{}),
        Environment: make(map[string]string),
    }

    // Add user-specific parameters
    userCmd.Parameters["user"] = userCtx.UserID
    userCmd.Flags["catalog-file"] = userCtx.CatalogPath
    userCmd.Flags["network"] = userCtx.NetworkName

    // Copy and enhance original parameters
    for key, value := range command.Parameters {
        userCmd.Parameters[key] = muce.resolveUserValue(userCtx, value)
    }

    // Add user environment
    for key, value := range userCtx.Environment {
        userCmd.Environment[key] = value
    }

    return userCmd, nil
}
```

### Catalog Synchronization

**Real-time Catalog Updates**:

```go
type CatalogSynchronizer struct {
    catalogMgr    *CatalogManager
    eventBus      *EventBus
    syncScheduler *cron.Cron
    subscribers   map[string][]chan CatalogEvent
}

type CatalogEvent struct {
    Type        CatalogEventType `json:"type"`
    UserID      *string         `json:"user_id,omitempty"`
    CatalogName string          `json:"catalog_name"`
    Version     string          `json:"version"`
    Timestamp   time.Time       `json:"timestamp"`
    Changes     []Change        `json:"changes"`
}

func (cs *CatalogSynchronizer) SynchronizeUserCatalog(userID string) error {
    // 1. Check for base catalog updates
    baseUpdates, err := cs.checkBaseUpdates(userID)
    if err != nil {
        return err
    }

    // 2. Regenerate user catalog if needed
    if len(baseUpdates) > 0 {
        catalog, err := cs.catalogMgr.GenerateUserCatalog(userID)
        if err != nil {
            return err
        }

        // 3. Notify user of changes
        event := &CatalogEvent{
            Type:        CatalogEventTypeUpdated,
            UserID:      &userID,
            CatalogName: "user_catalog",
            Version:     catalog.Version,
            Timestamp:   time.Now(),
            Changes:     baseUpdates,
        }

        cs.notifySubscribers(userID, event)
    }

    return nil
}

func (cs *CatalogSynchronizer) ScheduleSync() {
    // Sync all user catalogs every 5 minutes
    cs.syncScheduler.AddFunc("*/5 * * * *", func() {
        cs.syncAllUserCatalogs()
    })

    // Check for base catalog updates every minute
    cs.syncScheduler.AddFunc("* * * * *", func() {
        cs.checkBaseCatalogUpdates()
    })

    cs.syncScheduler.Start()
}
```

---

## Migration Strategy

### Phase 1: Foundation (Weeks 1-2)

**Infrastructure Setup**:

```yaml
phase_1_tasks:
  - name: "File System Structure"
    description: "Create catalog directory structure"
    components:
      - Base catalog directory (/app/data/catalogs/base/)
      - User catalog directories (/app/data/catalogs/users/)
      - Overlay directory (/app/data/catalogs/overlays/)
      - Cache directory (/app/data/catalogs/cache/)
    validation:
      - Directory permissions are correct
      - File watchers are functional
      - Basic CRUD operations work

  - name: "User Isolation Framework"
    description: "Implement basic user isolation"
    components:
      - User context management
      - Network creation/management
      - Port allocation system
      - Basic resource limits
    validation:
      - Users get isolated networks
      - Port ranges are properly allocated
      - Containers can't access other user resources

  - name: "Environment Secret Management"
    description: "Replace Docker Desktop secrets with env vars"
    components:
      - Secret encryption system
      - Environment file generation
      - Secret injection into containers
    validation:
      - Secrets are encrypted at rest
      - Only authorized users can access secrets
      - Secrets are properly injected
```

### Phase 2: Core Functionality (Weeks 3-4)

**Catalog System Implementation**:

```yaml
phase_2_tasks:
  - name: "Catalog Management System"
    description: "Implement file-based catalog system"
    components:
      - Base catalog upload/management
      - User catalog generation
      - Catalog merging algorithm
      - Validation framework
    validation:
      - Admin can upload base catalogs
      - User catalogs are generated correctly
      - Catalog changes trigger regeneration
      - Validation catches malformed catalogs

  - name: "Container Orchestration"
    description: "Multi-user container management"
    components:
      - User-specific container creation
      - Volume mounting with user isolation
      - Environment variable injection
      - Container lifecycle management
    validation:
      - Containers run in user networks
      - User data is isolated
      - Environment variables are correct
      - Container cleanup works properly
```

### Phase 3: Integration (Weeks 5-6)

**Portal Integration**:

```yaml
phase_3_tasks:
  - name: "API Integration"
    description: "Update Portal APIs for multi-user support"
    components:
      - User context in all API calls
      - Multi-user server management
      - Catalog API endpoints
      - Real-time updates
    validation:
      - API calls include user context
      - Users only see their resources
      - Real-time updates work correctly
      - Error handling is comprehensive

  - name: "Frontend Updates"
    description: "Update UI for multi-user features"
    components:
      - User-specific server lists
      - Catalog management interface
      - Resource usage displays
      - Multi-user admin interface
    validation:
      - UI shows correct user data
      - Admin interface works
      - Resource displays are accurate
      - User experience is intuitive
```

### Phase 4: Testing & Deployment (Weeks 7-8)

**Production Readiness**:

```yaml
phase_4_tasks:
  - name: "Security Hardening"
    description: "Implement production security measures"
    components:
      - Security policy enforcement
      - Audit logging
      - Penetration testing
      - Vulnerability assessment
    validation:
      - Security policies are enforced
      - Audit trail is complete
      - No security vulnerabilities found
      - Compliance requirements met

  - name: "Performance Testing"
    description: "Validate system performance"
    components:
      - Load testing with multiple users
      - Resource usage monitoring
      - Performance optimization
      - Scalability validation
    validation:
      - System handles expected load
      - Resource usage is within limits
      - Performance meets requirements
      - System scales horizontally
```

### Data Migration Plan

**Existing Data Preservation**:

```go
type DataMigrationManager struct {
    oldConfigPath string
    newCatalogPath string
    backupPath    string
    migrator      *ConfigMigrator
}

func (dmm *DataMigrationManager) MigrateExistingConfigurations() error {
    // 1. Backup existing configurations
    if err := dmm.backupExistingData(); err != nil {
        return err
    }

    // 2. Convert single-user catalogs to multi-user format
    if err := dmm.convertCatalogs(); err != nil {
        return err
    }

    // 3. Migrate user configurations
    if err := dmm.migrateUserConfigs(); err != nil {
        return err
    }

    // 4. Update container configurations
    if err := dmm.updateContainers(); err != nil {
        return err
    }

    // 5. Validate migration
    return dmm.validateMigration()
}

func (dmm *DataMigrationManager) convertCatalogs() error {
    // Convert existing catalog.json to base catalog
    oldCatalog, err := dmm.loadOldCatalog()
    if err != nil {
        return err
    }

    baseCatalog := &CatalogFile{
        Version:   "1.0.0",
        Source:    "base",
        Servers:   dmm.convertServerDefinitions(oldCatalog.Servers),
        Metadata:  dmm.generateBaseCatalogMetadata(),
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    return dmm.saveBaseCatalog(baseCatalog)
}
```

---

## API Specifications

### Multi-User Catalog API

**Enhanced API Endpoints**:

```yaml
# User-specific catalog endpoints
endpoints:
  - path: "/api/v1/users/{user_id}/catalog"
    method: "GET"
    description: "Get user's merged catalog"
    parameters:
      user_id:
        type: "uuid"
        required: true
        source: "path"
    response:
      type: "CatalogFile"
      description: "User's complete merged catalog"

  - path: "/api/v1/users/{user_id}/catalog/servers"
    method: "GET"
    description: "Get user's available servers"
    parameters:
      user_id:
        type: "uuid"
        required: true
        source: "path"
      category:
        type: "string"
        required: false
        source: "query"
      enabled_only:
        type: "boolean"
        default: false
        source: "query"

  - path: "/api/v1/users/{user_id}/servers/{server_id}/enable"
    method: "POST"
    description: "Enable server for user"
    parameters:
      user_id:
        type: "uuid"
        required: true
        source: "path"
      server_id:
        type: "string"
        required: true
        source: "path"
      config:
        type: "ServerConfig"
        required: false
        source: "body"

  - path: "/api/v1/users/{user_id}/secrets"
    method: "POST"
    description: "Set user secrets"
    parameters:
      user_id:
        type: "uuid"
        required: true
        source: "path"
      secrets:
        type: "map[string]string"
        required: true
        source: "body"
```

### Admin Catalog Management API

**Administrative Endpoints**:

```yaml
# Admin-only catalog management
admin_endpoints:
  - path: "/api/v1/admin/catalogs/base"
    method: "POST"
    description: "Upload new base catalog"
    auth_required: "admin"
    parameters:
      catalog:
        type: "CatalogFile"
        required: true
        source: "body"
      auto_deploy:
        type: "boolean"
        default: true
        source: "body"

  - path: "/api/v1/admin/catalogs/base/{catalog_id}"
    method: "PUT"
    description: "Update base catalog"
    auth_required: "admin"

  - path: "/api/v1/admin/users/{user_id}/catalogs/regenerate"
    method: "POST"
    description: "Force regenerate user catalog"
    auth_required: "admin"

  - path: "/api/v1/admin/system/migration"
    method: "POST"
    description: "Trigger system migration"
    auth_required: "admin"
```

### WebSocket Events for Multi-User

**Real-time Updates**:

```yaml
websocket_events:
  - event: "USER_CATALOG_UPDATED"
    description: "User's catalog was regenerated"
    data:
      user_id: "uuid"
      catalog_version: "string"
      changes: "array[Change]"
      timestamp: "datetime"

  - event: "SERVER_STATUS_CHANGED"
    description: "User's server status changed"
    data:
      user_id: "uuid"
      server_id: "string"
      old_status: "string"
      new_status: "string"
      container_id: "string"
      timestamp: "datetime"

  - event: "RESOURCE_USAGE_UPDATE"
    description: "User's resource usage updated"
    data:
      user_id: "uuid"
      resources:
        containers: "int"
        memory_mb: "int"
        cpu_shares: "int"
        ports_used: "int"
      limits:
        max_containers: "int"
        max_memory_mb: "int"
        max_cpu_shares: "int"
        max_ports: "int"
```

---

## Deployment Architecture

### Updated Docker Compose Configuration

**Multi-User Extensions**:

```yaml
# docker-compose.mcp-portal-multiuser.yml
services:
  portal:
    build:
      context: .
      dockerfile: Dockerfile.mcp-portal
    environment:
      # Multi-user specific configuration
      MCP_PORTAL_MULTIUSER_ENABLED: true
      MCP_PORTAL_USER_ISOLATION_ENABLED: true
      MCP_PORTAL_CATALOG_BASE_DIR: /app/data/catalogs
      MCP_PORTAL_USER_DATA_DIR: /app/data/users
      MCP_PORTAL_SECRET_ENCRYPTION_KEY: ${SECRET_ENCRYPTION_KEY}

      # Port allocation configuration
      MCP_PORTAL_PORT_RANGE_START: 20000
      MCP_PORTAL_PORT_RANGE_END: 29999
      MCP_PORTAL_PORTS_PER_USER: 100

      # Resource limits per user
      MCP_PORTAL_USER_MAX_CONTAINERS: 10
      MCP_PORTAL_USER_MAX_MEMORY_MB: 2048
      MCP_PORTAL_USER_MAX_CPU_SHARES: 1024

    volumes:
      # Docker socket for container management
      - /var/run/docker.sock:/var/run/docker.sock:ro
      # Multi-user data directories
      - portal-catalogs:/app/data/catalogs
      - portal-users:/app/data/users
      - portal-secrets:/app/data/secrets
      # Existing volumes
      - portal-data:/app/data
      - portal-logs:/app/logs

    # Additional capabilities for container management
    cap_add:
      - NET_ADMIN

    # Group access for Docker socket
    group_add:
      - ${DOCKER_GID:-999}

volumes:
  portal-catalogs:
    name: mcp-portal-catalogs
  portal-users:
    name: mcp-portal-users
  portal-secrets:
    name: mcp-portal-secrets
```

### Environment Configuration Updates

**Enhanced .env Template**:

```bash
# Multi-User Configuration
MCP_PORTAL_MULTIUSER_ENABLED=true
MCP_PORTAL_USER_ISOLATION_ENABLED=true
SECRET_ENCRYPTION_KEY=your-32-byte-encryption-key-here

# Port Allocation
MCP_PORTAL_PORT_RANGE_START=20000
MCP_PORTAL_PORT_RANGE_END=29999
MCP_PORTAL_PORTS_PER_USER=100

# Resource Limits (per user)
MCP_PORTAL_USER_MAX_CONTAINERS=10
MCP_PORTAL_USER_MAX_MEMORY_MB=2048
MCP_PORTAL_USER_MAX_CPU_SHARES=1024
MCP_PORTAL_USER_MAX_DISK_MB=5120

# Network Configuration
MCP_PORTAL_USER_NETWORK_SUBNET_BASE=172.20.0.0/16
MCP_PORTAL_USER_NETWORK_SUBNET_SIZE=24

# Security Configuration
MCP_PORTAL_CONTAINER_SECURITY_ENABLED=true
MCP_PORTAL_AUDIT_LOGGING_ENABLED=true
MCP_PORTAL_AUDIT_LOG_RETENTION_DAYS=90

# Docker Configuration
DOCKER_GID=999  # Docker group ID (adjust for your system)
```

### Dockerfile Enhancements

**Multi-User Support Additions**:

```dockerfile
# Add to existing Dockerfile.mcp-portal

# Install additional tools for multi-user management
RUN apk add --no-cache \
    iptables \
    bridge-utils \
    shadow \
    && rm -rf /var/cache/apk/*

# Create user directories
RUN mkdir -p /app/data/catalogs/base \
    /app/data/catalogs/users \
    /app/data/catalogs/overlays \
    /app/data/catalogs/cache \
    /app/data/users \
    /app/data/secrets \
    && chown -R portal:portal /app/data

# Copy multi-user initialization script
COPY --chown=portal:portal scripts/init-multiuser.sh /app/init-multiuser.sh
RUN chmod +x /app/init-multiuser.sh

# Update startup script to include multi-user initialization
COPY --chown=portal:portal <<'EOF' /app/start-multiuser.sh
#!/bin/sh
set -e

echo "Initializing multi-user environment..."
/app/init-multiuser.sh

echo "Starting MCP Portal with multi-user support..."
exec /app/start.sh
EOF

RUN chmod +x /app/start-multiuser.sh

# Use multi-user startup script
CMD ["/app/start-multiuser.sh"]
```

---

## Testing & Validation

### Multi-User Test Suite

**Integration Test Framework**:

```go
// Test suite for multi-user functionality
type MultiUserTestSuite struct {
    suite.Suite
    portal       *Portal
    testUsers    []*TestUser
    catalogMgr   *CatalogManager
    containerMgr *UserContainerManager
}

type TestUser struct {
    ID          string
    NetworkName string
    PortRange   PortRange
    Secrets     map[string]string
}

func (suite *MultiUserTestSuite) TestUserIsolation() {
    // Create two test users
    user1 := suite.createTestUser("test-user-1")
    user2 := suite.createTestUser("test-user-2")

    // Enable same server for both users
    server1, err := suite.portal.EnableServer(user1.ID, "github", nil)
    suite.NoError(err)

    server2, err := suite.portal.EnableServer(user2.ID, "github", nil)
    suite.NoError(err)

    // Verify containers are isolated
    suite.NotEqual(server1.ContainerID, server2.ContainerID)
    suite.NotEqual(server1.Port, server2.Port)
    suite.NotEqual(server1.NetworkName, server2.NetworkName)

    // Verify network isolation
    suite.verifyNetworkIsolation(user1, user2)
}

func (suite *MultiUserTestSuite) TestCatalogMerging() {
    // Upload base catalog
    baseCatalog := suite.createBaseCatalog()
    err := suite.catalogMgr.UploadBaseCatalog(baseCatalog)
    suite.NoError(err)

    // Create user with custom servers
    userID := "test-catalog-user"
    customServers := suite.createCustomServers()
    err = suite.catalogMgr.AddUserCustomServers(userID, customServers)
    suite.NoError(err)

    // Generate user catalog
    userCatalog, err := suite.catalogMgr.GenerateUserCatalog(userID)
    suite.NoError(err)

    // Verify merging
    suite.True(len(userCatalog.Servers) > len(baseCatalog.Servers))
    suite.verifyCustomServersPresent(userCatalog, customServers)
}

func (suite *MultiUserTestSuite) TestResourceLimits() {
    userID := "test-resource-user"

    // Set low resource limits for testing
    limits := &UserResourceLimits{
        MaxContainers: 2,
        MaxMemoryMB:   512,
        MaxCPUShares:  512,
        MaxPorts:      10,
    }
    suite.portal.SetUserResourceLimits(userID, limits)

    // Try to exceed container limit
    suite.enableServersUntilLimit(userID, 3)

    // Verify limit enforcement
    containers, err := suite.portal.GetUserContainers(userID)
    suite.NoError(err)
    suite.LessOrEqual(len(containers), 2)
}

func (suite *MultiUserTestSuite) TestSecretEncryption() {
    userID := "test-secret-user"

    // Set user secrets
    secrets := map[string]string{
        "GITHUB_TOKEN": "ghp_test_token_123",
        "API_KEY":      "test_api_key_456",
    }

    err := suite.portal.SetUserSecrets(userID, secrets)
    suite.NoError(err)

    // Verify secrets are encrypted at rest
    secretsFile := fmt.Sprintf("/app/data/secrets/%s/secrets.env", userID)
    content, err := ioutil.ReadFile(secretsFile)
    suite.NoError(err)

    // Verify original values are not in file
    suite.NotContains(string(content), "ghp_test_token_123")
    suite.NotContains(string(content), "test_api_key_456")

    // Verify secrets can be decrypted
    retrievedSecrets, err := suite.portal.GetUserSecrets(userID)
    suite.NoError(err)
    suite.Equal(secrets["GITHUB_TOKEN"], retrievedSecrets["GITHUB_TOKEN"])
}
```

### Performance Test Scenarios

**Load Testing Framework**:

```go
type PerformanceTestSuite struct {
    suite.Suite
    portal     *Portal
    userCount  int
    serverCount int
    metrics    *PerformanceMetrics
}

func (suite *PerformanceTestSuite) TestConcurrentUserOperations() {
    // Test concurrent operations by multiple users
    userCount := 50
    operationsPerUser := 20

    var wg sync.WaitGroup
    wg.Add(userCount)

    startTime := time.Now()

    for i := 0; i < userCount; i++ {
        go func(userIndex int) {
            defer wg.Done()

            userID := fmt.Sprintf("perf-user-%d", userIndex)

            // Perform various operations
            suite.performUserOperations(userID, operationsPerUser)
        }(i)
    }

    wg.Wait()
    duration := time.Since(startTime)

    // Verify performance metrics
    suite.metrics.RecordTest("concurrent_users", duration, userCount*operationsPerUser)
    suite.Less(duration, 30*time.Second, "Operations should complete within 30 seconds")
}

func (suite *PerformanceTestSuite) TestCatalogRegenerationPerformance() {
    // Test catalog regeneration with many users
    userCount := 100

    // Create users
    for i := 0; i < userCount; i++ {
        userID := fmt.Sprintf("catalog-user-%d", i)
        suite.createTestUserWithCustomServers(userID)
    }

    // Upload new base catalog and measure regeneration time
    startTime := time.Now()

    newBaseCatalog := suite.createLargeBaseCatalog(500) // 500 servers
    err := suite.portal.UploadBaseCatalog(newBaseCatalog)
    suite.NoError(err)

    // Wait for all catalogs to regenerate
    suite.waitForCatalogRegeneration(userCount)

    duration := time.Since(startTime)

    // Should complete within 60 seconds for 100 users
    suite.Less(duration, 60*time.Second)

    // Verify all user catalogs are updated
    suite.verifyAllCatalogsUpdated(userCount)
}
```

### Security Test Scenarios

**Security Validation Tests**:

```go
type SecurityTestSuite struct {
    suite.Suite
    portal      *Portal
    testUser    *TestUser
    attacker    *TestUser
}

func (suite *SecurityTestSuite) TestUserDataIsolation() {
    // Create two users
    user1 := suite.createTestUser("security-user-1")
    user2 := suite.createTestUser("security-user-2")

    // User1 creates data
    data1 := "sensitive-user1-data"
    err := suite.portal.StoreUserData(user1.ID, "test-file.txt", data1)
    suite.NoError(err)

    // User2 tries to access User1's data
    retrievedData, err := suite.portal.GetUserData(user2.ID, "test-file.txt")
    suite.Error(err, "User2 should not be able to access User1's data")
    suite.Empty(retrievedData)

    // Verify file system isolation
    user1Dir := fmt.Sprintf("/app/data/users/%s", user1.ID)
    user2Dir := fmt.Sprintf("/app/data/users/%s", user2.ID)

    // User2 should not have read access to User1's directory
    suite.verifyDirectoryIsolation(user1Dir, user2Dir)
}

func (suite *SecurityTestSuite) TestContainerNetworkIsolation() {
    // Create two users with servers
    user1 := suite.createTestUser("net-user-1")
    user2 := suite.createTestUser("net-user-2")

    container1, err := suite.portal.EnableServer(user1.ID, "test-server", nil)
    suite.NoError(err)

    container2, err := suite.portal.EnableServer(user2.ID, "test-server", nil)
    suite.NoError(err)

    // Verify containers cannot communicate
    suite.verifyNetworkIsolation(container1, container2)
}

func (suite *SecurityTestSuite) TestSecretAccess() {
    // Set secrets for user
    secrets := map[string]string{
        "SECRET_KEY": "super-secret-value",
    }

    err := suite.portal.SetUserSecrets(suite.testUser.ID, secrets)
    suite.NoError(err)

    // Verify other users cannot access secrets
    attackerSecrets, err := suite.portal.GetUserSecrets(suite.attacker.ID)
    suite.NoError(err)
    suite.NotContains(attackerSecrets, "SECRET_KEY")

    // Verify secrets are encrypted in filesystem
    suite.verifySecretsEncryption(suite.testUser.ID)
}
```

---

## Conclusion

This comprehensive design document provides a complete blueprint for implementing a Docker Desktop-independent multi-user catalog system for the MCP Portal. The design ensures:

### Key Benefits

1. **Docker Engine Compatibility**: Eliminates Docker Desktop dependency while maintaining full functionality
2. **User Isolation**: Provides secure multi-user environment with container-level isolation
3. **File-Based Configuration**: Uses robust file system approach for catalog management
4. **Minimal Disruption**: Leverages existing infrastructure with targeted enhancements
5. **Production Ready**: Includes comprehensive security, monitoring, and audit capabilities

### Implementation Readiness

The design includes:

- **Detailed technical specifications** with code examples
- **Complete security framework** with encryption and audit logging
- **Comprehensive testing strategy** covering functionality, performance, and security
- **Phased migration plan** to minimize risk and downtime
- **Production deployment configuration** ready for immediate use

### Next Steps

1. **Review and approve** the design with stakeholders
2. **Begin Phase 1 implementation** with infrastructure setup
3. **Implement comprehensive testing** throughout development
4. **Plan production migration** following the provided strategy
5. **Monitor and optimize** post-deployment performance

This design provides a solid foundation for scaling the MCP Portal to support multiple users while maintaining security, performance, and operational simplicity.
