# Docker Desktop-Independent Multi-User Catalog System Design

**Date**: September 18, 2025
**Project**: MCP Gateway & Portal
**Version**: 1.0
**Status**: Ready for Implementation

## Executive Summary

This design document specifies a Docker Desktop-independent implementation of the multi-user catalog management system for the MCP Portal. The design leverages file-based catalog management, environment variable configuration, and Docker Engine features to provide enterprise-grade multi-user support without requiring Docker Desktop.

### Key Design Principles

1. **No Docker Desktop Dependencies**: Works with standard Docker Engine
2. **File-Based Configuration**: Uses YAML files instead of Docker Desktop APIs
3. **Environment Variable Secrets**: No reliance on Docker secrets API
4. **Backward Compatibility**: Minimal changes to existing deployment
5. **Security First**: Maintains isolation between users

## System Architecture

### High-Level Architecture

```
┌────────────────────────────────────────────────────┐
│                   MCP Portal                       │
│  ┌────────────────────────────────────────────┐    │
│  │         Catalog Management Service         │    │
│  └────────────────┬───────────────────────────┘    │
│                   │                                │
│  ┌────────────────▼───────────────────────────┐    │
│  │      File-Based Catalog Generator          │    │
│  └────────────────┬───────────────────────────┘    │
│                   │                                │
│  ┌────────────────▼───────────────────────────┐    │
│  │    User Container Orchestrator             │    │
│  └───────────────┬────────────────────────────┘    │
└──────────────────│─────────────────────────────────┘
                   │
        ┌──────────▼──────────────────────┐
        │     Docker Engine (No Desktop)  │
        │  ┌────────────────────────┐     │
        │  │ User Gateway Container │     │
        │  │   - Catalog Files      │     │
        │  │   - Env Variables      │     │
        │  │   - Isolated Network   │     │
        │  └────────────────────────┘     │
        └─────────────────────────────────┘
```

### File System Layout

```
/app/data/                             # Portal data directory
├── catalogs/                          # Catalog storage
│   ├── base/                          # Admin-controlled base catalogs
│   │   ├── docker-official.yaml       # Docker's official catalog
│   │   ├── organization.yaml          # Organization-wide catalog
│   │   └── team-{team-id}.yaml        # Team-specific catalogs
│   │
│   ├── users/                         # User-specific data
│   │   └── {user-id}/
│   │       ├── custom.yaml            # User's custom catalog
│   │       ├── overrides.yaml         # User overrides for base catalogs
│   │       ├── merged.yaml            # Generated merged catalog
│   │       └── secrets.env            # Encrypted user secrets
│   │
│   ├── templates/                     # Catalog templates
│   │   └── default-user.yaml          # Template for new users
│   │
│   └── cache/                         # Generated cache files
│       └── {user-id}-{hash}.yaml      # Cached merged catalogs
│
├── containers/                        # Container management
│   └── {user-id}/
│       ├── gateway.pid                # Gateway process ID
│       ├── port.lock                  # Allocated port number
│       └── logs/                      # User-specific logs
│
└── audit/                             # Audit logs
    └── catalog-operations.log         # All catalog modifications
```

## Component Design

### 1. File-Based Catalog Manager

```go
// cmd/docker-mcp/portal/catalog/file_manager.go
package catalog

import (
    "path/filepath"
    "gopkg.in/yaml.v3"
)

type FileCatalogManager struct {
    baseDir    string
    logger     *slog.Logger
    mutex      sync.RWMutex
    watcher    *fsnotify.Watcher
}

type CatalogFile struct {
    Path         string
    Type         CatalogType // base, user, overlay
    Priority     int
    LastModified time.Time
    Checksum     string
}

func (fcm *FileCatalogManager) GenerateUserCatalog(
    ctx context.Context,
    userID string,
) (*MergedCatalog, error) {
    fcm.mutex.RLock()
    defer fcm.mutex.RUnlock()

    // 1. Load base catalogs
    baseCatalogs, err := fcm.loadBaseCatalogs(ctx)
    if err != nil {
        return nil, fmt.Errorf("loading base catalogs: %w", err)
    }

    // 2. Load user-specific catalogs
    userCatalogs, err := fcm.loadUserCatalogs(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("loading user catalogs: %w", err)
    }

    // 3. Apply merge strategy
    merged := &MergedCatalog{
        UserID:    userID,
        Timestamp: time.Now(),
        Registry:  make(map[string]ServerConfig),
    }

    // Merge in precedence order: base -> team -> user
    for _, catalog := range append(baseCatalogs, userCatalogs...) {
        if err := fcm.mergeCatalog(merged, catalog); err != nil {
            return nil, fmt.Errorf("merging catalog %s: %w", catalog.Path, err)
        }
    }

    // 4. Write merged catalog to disk
    outputPath := filepath.Join(fcm.baseDir, "users", userID, "merged.yaml")
    if err := fcm.writeCatalog(outputPath, merged); err != nil {
        return nil, fmt.Errorf("writing merged catalog: %w", err)
    }

    // 5. Update cache
    fcm.updateCache(userID, merged)

    return merged, nil
}
```

### 2. User Container Orchestrator

```go
// cmd/docker-mcp/portal/docker/user_orchestrator.go
package docker

type UserOrchestrator struct {
    client      *client.Client
    portManager *PortManager
    network     *NetworkManager
    catalogMgr  *catalog.FileCatalogManager
}

type UserGateway struct {
    UserID      string
    ContainerID string
    Port        int
    Network     string
    Status      string
    StartedAt   time.Time
}

func (uo *UserOrchestrator) CreateUserGateway(
    ctx context.Context,
    userID string,
) (*UserGateway, error) {
    // 1. Generate user catalog
    catalog, err := uo.catalogMgr.GenerateUserCatalog(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("generating catalog: %w", err)
    }

    // 2. Allocate port
    port, err := uo.portManager.AllocatePort(userID)
    if err != nil {
        return nil, fmt.Errorf("allocating port: %w", err)
    }

    // 3. Create user network
    networkName := fmt.Sprintf("mcp-net-%s", userID)
    if err := uo.network.CreateUserNetwork(ctx, networkName); err != nil {
        uo.portManager.ReleasePort(port)
        return nil, fmt.Errorf("creating network: %w", err)
    }

    // 4. Prepare environment
    env := uo.buildEnvironment(userID, catalog)

    // 5. Create container
    containerName := fmt.Sprintf("mcp-gateway-%s", userID)

    config := &container.Config{
        Image: "mcp-gateway:latest",
        Env:   env,
        ExposedPorts: nat.PortSet{
            nat.Port(fmt.Sprintf("%d/tcp", port)): {},
        },
        Labels: map[string]string{
            "mcp.user_id": userID,
            "mcp.type":    "gateway",
        },
    }

    hostConfig := &container.HostConfig{
        PortBindings: nat.PortMap{
            nat.Port("8080/tcp"): []nat.PortBinding{
                {HostPort: strconv.Itoa(port)},
            },
        },
        NetworkMode: container.NetworkMode(networkName),
        Mounts: []mount.Mount{
            {
                Type:     mount.TypeBind,
                Source:   filepath.Join(uo.catalogMgr.BaseDir(), "users", userID),
                Target:   "/config",
                ReadOnly: true,
            },
            {
                Type:   mount.TypeVolume,
                Source: fmt.Sprintf("mcp-data-%s", userID),
                Target: "/data",
            },
        },
        Resources: container.Resources{
            Memory:   536870912,  // 512MB
            CPUQuota: 50000,      // 50% of one CPU
        },
        SecurityOpt: []string{
            "no-new-privileges:true",
        },
    }

    resp, err := uo.client.ContainerCreate(
        ctx, config, hostConfig, nil, nil, containerName,
    )
    if err != nil {
        uo.cleanup(ctx, userID, port, networkName)
        return nil, fmt.Errorf("creating container: %w", err)
    }

    // 6. Start container
    if err := uo.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
        uo.cleanup(ctx, userID, port, networkName)
        return nil, fmt.Errorf("starting container: %w", err)
    }

    return &UserGateway{
        UserID:      userID,
        ContainerID: resp.ID,
        Port:        port,
        Network:     networkName,
        Status:      "running",
        StartedAt:   time.Now(),
    }, nil
}
```

### 3. Dynamic Port Allocator

```go
// cmd/docker-mcp/portal/docker/port_manager.go
package docker

type PortManager struct {
    minPort     int
    maxPort     int
    allocated   map[int]string // port -> userID
    userPorts   map[string]int  // userID -> port
    mutex       sync.Mutex
    persistence PersistenceLayer
}

func NewPortManager(minPort, maxPort int, persistence PersistenceLayer) *PortManager {
    pm := &PortManager{
        minPort:     minPort,
        maxPort:     maxPort,
        allocated:   make(map[int]string),
        userPorts:   make(map[string]int),
        persistence: persistence,
    }

    // Restore state from persistence
    pm.restoreState()

    return pm
}

func (pm *PortManager) AllocatePort(userID string) (int, error) {
    pm.mutex.Lock()
    defer pm.mutex.Unlock()

    // Check if user already has a port
    if port, exists := pm.userPorts[userID]; exists {
        return port, nil
    }

    // Find next available port
    for port := pm.minPort; port <= pm.maxPort; port++ {
        if _, allocated := pm.allocated[port]; !allocated {
            // Check if port is actually available
            if pm.isPortAvailable(port) {
                pm.allocated[port] = userID
                pm.userPorts[userID] = port
                pm.persistState()
                return port, nil
            }
        }
    }

    return 0, fmt.Errorf("no available ports in range %d-%d", pm.minPort, pm.maxPort)
}

func (pm *PortManager) isPortAvailable(port int) bool {
    listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return false
    }
    listener.Close()
    return true
}
```

### 4. Secret Management Without Docker Desktop

```go
// cmd/docker-mcp/portal/secrets/env_manager.go
package secrets

type EnvSecretManager struct {
    encryptor   *crypto.AES256GCM
    baseDir     string
    auditLogger *audit.Logger
}

type UserSecrets struct {
    UserID      string
    Secrets     map[string]string
    LastUpdated time.Time
    Version     int
}

func (esm *EnvSecretManager) SaveUserSecrets(
    ctx context.Context,
    userID string,
    secrets map[string]string,
) error {
    // 1. Encrypt secrets
    encrypted := make(map[string]string)
    for key, value := range secrets {
        encValue, err := esm.encryptor.Encrypt([]byte(value))
        if err != nil {
            return fmt.Errorf("encrypting secret %s: %w", key, err)
        }
        encrypted[key] = base64.StdEncoding.EncodeToString(encValue)
    }

    // 2. Generate .env file content
    var envContent strings.Builder
    envContent.WriteString("# Generated: " + time.Now().Format(time.RFC3339) + "\n")
    envContent.WriteString("# User: " + userID + "\n\n")

    for key, value := range encrypted {
        envContent.WriteString(fmt.Sprintf("%s=%s\n", key, value))
    }

    // 3. Write to user directory
    secretPath := filepath.Join(esm.baseDir, "users", userID, "secrets.env")
    if err := os.WriteFile(secretPath, []byte(envContent.String()), 0600); err != nil {
        return fmt.Errorf("writing secrets file: %w", err)
    }

    // 4. Set file permissions (owner read/write only)
    if err := os.Chmod(secretPath, 0600); err != nil {
        return fmt.Errorf("setting permissions: %w", err)
    }

    // 5. Audit log
    esm.auditLogger.Log(audit.Event{
        Type:      "secrets_updated",
        UserID:    userID,
        Timestamp: time.Now(),
        Details:   map[string]interface{}{"keys": len(secrets)},
    })

    return nil
}

func (esm *EnvSecretManager) LoadUserSecrets(
    ctx context.Context,
    userID string,
) (*UserSecrets, error) {
    secretPath := filepath.Join(esm.baseDir, "users", userID, "secrets.env")

    // Read encrypted file
    data, err := os.ReadFile(secretPath)
    if err != nil {
        if os.IsNotExist(err) {
            return &UserSecrets{
                UserID:  userID,
                Secrets: make(map[string]string),
            }, nil
        }
        return nil, fmt.Errorf("reading secrets: %w", err)
    }

    // Parse and decrypt
    secrets := make(map[string]string)
    scanner := bufio.NewScanner(strings.NewReader(string(data)))

    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "#") || line == "" {
            continue
        }

        parts := strings.SplitN(line, "=", 2)
        if len(parts) == 2 {
            encValue, _ := base64.StdEncoding.DecodeString(parts[1])
            decValue, err := esm.encryptor.Decrypt(encValue)
            if err != nil {
                return nil, fmt.Errorf("decrypting secret %s: %w", parts[0], err)
            }
            secrets[parts[0]] = string(decValue)
        }
    }

    return &UserSecrets{
        UserID:      userID,
        Secrets:     secrets,
        LastUpdated: time.Now(),
    }, nil
}
```

## API Specifications

### Multi-User Catalog Management API

```yaml
openapi: 3.0.0
info:
  title: MCP Portal Multi-User Catalog API
  version: 1.0.0

paths:
  /api/v1/catalogs/base:
    get:
      summary: List base catalogs (admin-controlled)
      security:
        - bearerAuth: []
      responses:
        "200":
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/BaseCatalog"

    post:
      summary: Create new base catalog (admin only)
      security:
        - bearerAuth: []
      requestBody:
        content:
          multipart/form-data:
            schema:
              type: object
              properties:
                name:
                  type: string
                file:
                  type: string
                  format: binary
                priority:
                  type: integer
      responses:
        "201":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/BaseCatalog"

  /api/v1/users/{userId}/catalogs:
    get:
      summary: Get user's catalog configuration
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: string
        - name: include
          in: query
          schema:
            type: string
            enum: [base, custom, merged, all]
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/UserCatalogConfig"

    put:
      summary: Update user's custom catalog
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: string
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/CustomCatalog"
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/UserCatalogConfig"

  /api/v1/users/{userId}/gateway:
    post:
      summary: Create user's gateway container
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: string
      responses:
        "201":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/UserGateway"

    get:
      summary: Get user's gateway status
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/UserGateway"

    delete:
      summary: Stop user's gateway container
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: string
      responses:
        "204":
          description: Gateway stopped successfully

components:
  schemas:
    BaseCatalog:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
        path:
          type: string
        priority:
          type: integer
        serverCount:
          type: integer
        lastModified:
          type: string
          format: date-time

    UserCatalogConfig:
      type: object
      properties:
        userId:
          type: string
        baseCatalogs:
          type: array
          items:
            $ref: "#/components/schemas/BaseCatalog"
        customCatalog:
          $ref: "#/components/schemas/CustomCatalog"
        mergedCatalog:
          type: object
          properties:
            path:
              type: string
            serverCount:
              type: integer
            generatedAt:
              type: string
              format: date-time

    CustomCatalog:
      type: object
      properties:
        servers:
          type: array
          items:
            $ref: "#/components/schemas/ServerConfig"
        overrides:
          type: object
          additionalProperties:
            type: boolean

    ServerConfig:
      type: object
      properties:
        name:
          type: string
        image:
          type: string
        enabled:
          type: boolean
        config:
          type: object

    UserGateway:
      type: object
      properties:
        userId:
          type: string
        containerId:
          type: string
        port:
          type: integer
        status:
          type: string
          enum: [starting, running, stopping, stopped, error]
        startedAt:
          type: string
          format: date-time
        endpoint:
          type: string
          example: "http://localhost:20001"
```

## Docker Compose Configuration

### Enhanced docker-compose.mcp-portal.yml

```yaml
version: "3.8"

services:
  portal:
    build:
      context: .
      dockerfile: Dockerfile.mcp-portal
    image: mcp-portal:latest
    container_name: mcp-portal
    restart: unless-stopped
    ports:
      - "3000:3000" # Frontend
      - "8080:8080" # Backend API
    environment:
      # Multi-User Configuration
      MULTI_USER_ENABLED: "true"
      CATALOG_MODE: "file"
      CATALOG_BASE_DIR: "/app/data/catalogs"

      # Port Allocation Range
      PORT_RANGE_MIN: "20000"
      PORT_RANGE_MAX: "29999"

      # Resource Limits
      MAX_USERS_PER_INSTANCE: "100"
      USER_CONTAINER_MEMORY_LIMIT: "536870912" # 512MB
      USER_CONTAINER_CPU_QUOTA: "50000" # 50% of one CPU

      # Existing configuration...
      MCP_PORTAL_HOST: 0.0.0.0
      MCP_PORTAL_PORT: 8080

    volumes:
      # Docker socket (read-only for security)
      - /var/run/docker.sock:/var/run/docker.sock:ro

      # Multi-user data directories
      - portal-catalogs:/app/data/catalogs
      - portal-containers:/app/data/containers
      - portal-audit:/app/data/audit

      # User gateway volumes (dynamically created)
      - type: volume
        source: user-gateways
        target: /app/gateways
        volume:
          nocopy: true

    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy

    networks:
      - mcp-network
      - user-networks # Bridge to user-specific networks

    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/api/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s

  # Catalog File Watcher Service (Optional)
  catalog-watcher:
    build:
      context: .
      dockerfile: Dockerfile.catalog-watcher
    image: mcp-catalog-watcher:latest
    container_name: mcp-catalog-watcher
    restart: unless-stopped
    environment:
      CATALOG_DIR: "/catalogs"
      REDIS_URL: "redis://redis:6379"
    volumes:
      - portal-catalogs:/catalogs:ro
    depends_on:
      - redis
    networks:
      - mcp-network

volumes:
  portal-catalogs:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: ${CATALOGS_PATH:-/opt/mcp-portal/catalogs}

  portal-containers:
    driver: local

  portal-audit:
    driver: local

  user-gateways:
    driver: local

networks:
  mcp-network:
    driver: bridge

  user-networks:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
```

## Deployment Workflow

### Initial Setup

```bash
#!/bin/bash
# setup-multiuser.sh

# 1. Create directory structure
sudo mkdir -p /opt/mcp-portal/catalogs/{base,users,templates,cache}
sudo mkdir -p /opt/mcp-portal/containers
sudo mkdir -p /opt/mcp-portal/audit

# 2. Set permissions
sudo chown -R 1001:1001 /opt/mcp-portal
sudo chmod 700 /opt/mcp-portal/catalogs/users

# 3. Copy base catalogs
cp catalogs/docker-official.yaml /opt/mcp-portal/catalogs/base/
cp catalogs/organization.yaml /opt/mcp-portal/catalogs/base/

# 4. Start services
docker-compose -f docker-compose.mcp-portal.yml up -d

# 5. Initialize database
docker-compose exec portal /app/backend/docker-mcp portal migrate

# 6. Create admin user
docker-compose exec portal /app/backend/docker-mcp portal admin create \
  --email admin@example.com \
  --name "Admin User"
```

### User Onboarding

```bash
#!/bin/bash
# onboard-user.sh

USER_ID=$1
USER_EMAIL=$2

# 1. Create user directory
mkdir -p /opt/mcp-portal/catalogs/users/${USER_ID}

# 2. Copy template catalog
cp /opt/mcp-portal/catalogs/templates/default-user.yaml \
   /opt/mcp-portal/catalogs/users/${USER_ID}/custom.yaml

# 3. Create empty overrides
echo "# User overrides for ${USER_EMAIL}" > \
   /opt/mcp-portal/catalogs/users/${USER_ID}/overrides.yaml

# 4. Generate initial merged catalog via API
curl -X POST http://localhost:8080/api/v1/users/${USER_ID}/catalogs/generate \
  -H "Authorization: Bearer ${ADMIN_TOKEN}"

# 5. Create user gateway
curl -X POST http://localhost:8080/api/v1/users/${USER_ID}/gateway \
  -H "Authorization: Bearer ${ADMIN_TOKEN}"

echo "User ${USER_EMAIL} onboarded successfully"
echo "Gateway available at: http://localhost:$(cat /opt/mcp-portal/containers/${USER_ID}/port.lock)"
```

### Catalog Update Workflow

```bash
#!/bin/bash
# update-base-catalog.sh

CATALOG_NAME=$1
CATALOG_FILE=$2

# 1. Backup existing catalog
cp /opt/mcp-portal/catalogs/base/${CATALOG_NAME}.yaml \
   /opt/mcp-portal/catalogs/base/${CATALOG_NAME}.yaml.bak

# 2. Update catalog
cp ${CATALOG_FILE} /opt/mcp-portal/catalogs/base/${CATALOG_NAME}.yaml

# 3. Trigger regeneration for all users
curl -X POST http://localhost:8080/api/v1/admin/catalogs/regenerate \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -d '{"catalog": "'${CATALOG_NAME}'"}'

# 4. Monitor regeneration
watch -n 1 'curl -s http://localhost:8080/api/v1/admin/catalogs/regenerate/status'
```

## Security Considerations

### File System Security

```yaml
security_measures:
  file_permissions:
    base_catalogs: "0644" # World-readable (public)
    user_directories: "0700" # Owner only
    secret_files: "0600" # Owner read/write only
    audit_logs: "0640" # Owner write, group read

  directory_structure:
    user_isolation: "/app/data/catalogs/users/{user-id}/"
    no_traversal: "Validate all paths, reject ../ patterns"
    chroot_environments: "Consider for additional isolation"
```

### Container Isolation

```go
// Security configuration for user containers
securityConfig := container.SecurityConfig{
    User: "1000:1000",  // Non-root user
    ReadOnlyRootFilesystem: true,
    NoNewPrivileges: true,
    Capabilities: []string{
        "drop=ALL",
        "add=NET_BIND_SERVICE",
    },
    SeccompProfile: "default",
    AppArmorProfile: "docker-default",
}
```

### Network Isolation

```yaml
network_security:
  user_networks:
    naming: "mcp-net-{user-id}"
    isolation: "bridge driver with custom subnet"
    communication: "No inter-user network communication"

  port_allocation:
    range: "20000-29999"
    assignment: "One port per user"
    firewall: "Restrict to authenticated users only"
```

## Migration Strategy

### Phase 1: Infrastructure Setup (Weeks 1-2)

1. Deploy file system structure
2. Implement FileCatalogManager
3. Add PortManager component
4. Update Docker Compose configuration

### Phase 2: Core Multi-User Features (Weeks 3-4)

1. Implement UserOrchestrator
2. Add secret management without Docker Desktop
3. Create catalog generation pipeline
4. Implement user gateway creation

### Phase 3: Admin Interface (Weeks 5-6)

1. Build admin catalog management UI
2. Add user management interface
3. Implement bulk operations
4. Create monitoring dashboard

### Phase 4: Production Hardening (Weeks 7-8)

1. Performance optimization
2. Security audit
3. Load testing
4. Documentation and training

### Rollback Plan

```bash
#!/bin/bash
# rollback-multiuser.sh

# 1. Stop all user containers
docker ps --filter "label=mcp.type=gateway" -q | xargs docker stop

# 2. Restore single-user mode
docker-compose exec portal sh -c "
  echo 'MULTI_USER_ENABLED=false' >> /app/.env
"

# 3. Restart portal
docker-compose restart portal

# 4. Backup multi-user data
tar -czf multiuser-backup-$(date +%Y%m%d).tar.gz /opt/mcp-portal/

echo "Rollback complete. Single-user mode restored."
```

## Testing Framework

### Unit Tests

```go
func TestFileCatalogManager(t *testing.T) {
    tests := []struct {
        name     string
        userID   string
        catalogs []CatalogFile
        expected *MergedCatalog
    }{
        {
            name:   "merge_base_and_user",
            userID: "test-user-1",
            catalogs: []CatalogFile{
                {Path: "base/docker.yaml", Type: BaseType, Priority: 1},
                {Path: "users/test-user-1/custom.yaml", Type: UserType, Priority: 10},
            },
            expected: &MergedCatalog{
                ServerCount: 15,
                Sources:     2,
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mgr := NewFileCatalogManager(t.TempDir())
            result, err := mgr.GenerateUserCatalog(context.Background(), tt.userID)
            require.NoError(t, err)
            assert.Equal(t, tt.expected.ServerCount, len(result.Registry))
        })
    }
}
```

### Integration Tests

```go
func TestMultiUserIsolation(t *testing.T) {
    // Create test environment
    env := setupTestEnvironment(t)
    defer env.Cleanup()

    // Create two users
    user1 := env.CreateUser("user1")
    user2 := env.CreateUser("user2")

    // Create gateways for both users
    gw1, err := env.Orchestrator.CreateUserGateway(context.Background(), user1.ID)
    require.NoError(t, err)

    gw2, err := env.Orchestrator.CreateUserGateway(context.Background(), user2.ID)
    require.NoError(t, err)

    // Verify isolation
    assert.NotEqual(t, gw1.Port, gw2.Port)
    assert.NotEqual(t, gw1.Network, gw2.Network)

    // Verify catalog independence
    user1Catalog := env.GetUserCatalog(user1.ID)
    user2Catalog := env.GetUserCatalog(user2.ID)
    assert.NotEqual(t, user1Catalog, user2Catalog)

    // Verify network isolation
    canCommunicate := env.TestNetworkCommunication(gw1.ContainerID, gw2.ContainerID)
    assert.False(t, canCommunicate)
}
```

### Performance Tests

```go
func BenchmarkCatalogGeneration(b *testing.B) {
    mgr := NewFileCatalogManager("testdata/catalogs")

    b.Run("single_user", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            _, err := mgr.GenerateUserCatalog(context.Background(), "user1")
            if err != nil {
                b.Fatal(err)
            }
        }
    })

    b.Run("concurrent_users", func(b *testing.B) {
        b.RunParallel(func(pb *testing.PB) {
            userID := fmt.Sprintf("user-%d", rand.Intn(100))
            for pb.Next() {
                _, err := mgr.GenerateUserCatalog(context.Background(), userID)
                if err != nil {
                    b.Fatal(err)
                }
            }
        })
    })
}

// Performance targets:
// - Single user catalog generation: <50ms
// - Concurrent generation (100 users): <100ms average
// - Container creation: <2s
// - Port allocation: <10ms
```

## Monitoring and Observability

### Metrics

```yaml
prometheus_metrics:
  catalog_operations:
    - catalog_generation_duration_seconds
    - catalog_merge_operations_total
    - catalog_cache_hit_ratio

  user_gateways:
    - active_user_gateways
    - gateway_creation_duration_seconds
    - gateway_port_allocation_time
    - gateway_memory_usage_bytes

  file_system:
    - catalog_directory_size_bytes
    - user_directory_count
    - file_operation_errors_total
```

### Logging

```json
{
  "timestamp": "2025-09-18T10:30:00Z",
  "level": "INFO",
  "component": "UserOrchestrator",
  "user_id": "user-123",
  "action": "create_gateway",
  "details": {
    "port": 20001,
    "container_id": "abc123",
    "duration_ms": 1543,
    "catalog_servers": 12
  }
}
```

### Alerts

```yaml
alerting_rules:
  - alert: HighCatalogGenerationLatency
    expr: catalog_generation_duration_seconds > 0.2
    for: 5m
    annotations:
      summary: "Catalog generation taking >200ms"

  - alert: PortExhaustion
    expr: (port_range_used / port_range_total) > 0.9
    for: 1m
    annotations:
      summary: "90% of port range exhausted"

  - alert: UserGatewayFailures
    expr: rate(gateway_creation_errors[5m]) > 0.1
    for: 5m
    annotations:
      summary: "Gateway creation failure rate >10%"
```

## Conclusion

This design provides a comprehensive, Docker Desktop-independent solution for multi-user catalog management in the MCP Portal. Key benefits include:

1. **No Docker Desktop Required**: Works with standard Docker Engine
2. **File-Based Simplicity**: Easy to understand, debug, and manage
3. **Strong Isolation**: Users cannot access each other's data
4. **Backward Compatibility**: Minimal changes to existing deployment
5. **Production Ready**: Includes monitoring, security, and scaling considerations

The design leverages proven patterns from the MCP Gateway examples while maintaining the Portal's existing security and architecture standards. Implementation can begin immediately with Phase 1, providing incremental value while building toward full multi-user support.

---

**Document Status**: Approved for Implementation
**Next Steps**: Begin Phase 1 infrastructure setup
