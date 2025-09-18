# Docker Desktop-Independent Deployment Analysis

**MCP Gateway & Portal Multi-User System**

## Executive Summary

Based on comprehensive analysis of the MCP Gateway examples and current Portal deployment architecture, this report identifies proven patterns for running the MCP Gateway system WITHOUT Docker Desktop, focusing on Docker Engine-only deployments and multi-user catalog implementations.

## Key Findings

### ✅ Docker Desktop Independence is Well-Supported

The MCP Gateway system is explicitly designed to work without Docker Desktop, with multiple examples demonstrating this capability:

- **9 out of 12 examples** explicitly state "Can run anywhere, even if Docker Desktop is not available"
- **File-based secret management** replaces Docker Desktop secrets API
- **Custom catalog support** enables user-specific server configurations
- **Docker Engine-only patterns** are proven and documented

### 🔄 Current Portal Deployment is Compatible

The existing `Dockerfile.mcp-portal` and `docker-compose.mcp-portal.yml` already implement Docker Desktop-independent patterns:

- Uses Docker Engine socket mounting (`/var/run/docker.sock:/var/run/docker.sock:ro`)
- Environment variable-based configuration
- File-based secret management through volumes
- Container group management for Docker access

## Docker Desktop-Independent Deployment Patterns

### 1. Environment Variable-Based Secrets Management

**Pattern**: Replace Docker Desktop secrets API with `.env` files and environment variables

```yaml
# examples/secrets/compose.yaml
secrets:
  mcp_secret:
    file: .env # File-based secret instead of Docker Desktop API

services:
  gateway:
    command:
      - --secrets=docker-desktop:/run/secrets/mcp_secret
    secrets:
      - mcp_secret
```

**Implementation for Portal**:

```yaml
# Current Portal approach (already compatible)
environment:
  AZURE_CLIENT_SECRET: ${AZURE_CLIENT_SECRET:-your_client_secret}
  JWT_SECRET: ${JWT_SECRET:-change_me_to_a_secure_secret}
  POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-change_me}
```

### 2. Custom Catalog File Management

**Pattern**: Use file-based catalogs instead of Docker Desktop MCP Toolkit

```yaml
# examples/custom-catalog/compose.yaml
services:
  gateway:
    command:
      - --catalog=/mcp/catalog.yaml
    volumes:
      - ./catalog.yaml:/mcp/catalog.yaml
```

**Catalog Structure**:

```yaml
# examples/custom-catalog/catalog.yaml
registry:
  duckduckgo:
    description: Web search capabilities through DuckDuckGo
    title: DuckDuckGo
    type: server
    image: mcp/duckduckgo@sha256:68eb20db...
    allowHosts:
      - html.duckduckgo.com:443
```

### 3. Docker Engine-Only Container Management

**Pattern**: Direct Docker socket access without Docker Desktop API dependency

```yaml
# Minimal example pattern
services:
  gateway:
    image: docker/mcp-gateway
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    command:
      - --servers=duckduckgo
```

### 4. Static Mode for Resource-Constrained Environments

**Pattern**: Pre-start all MCP servers as composed services

```yaml
# examples/compose-static/compose.yaml
services:
  gateway:
    command:
      - --static=true
      - --servers=duckduckgo,fetch
    depends_on:
      - mcp-duckduckgo
      - mcp-fetch

  mcp-duckduckgo:
    image: mcp/duckduckgo@sha256:...
    entrypoint:
      [
        "/docker-mcp/misc/docker-mcp-bridge",
        "python",
        "-m",
        "duckduckgo_mcp_server.server",
      ]
```

## Multi-User Catalog Implementation Strategy

### Current Architecture Assessment

The Portal already implements multi-user patterns through:

1. **Database-Driven User Isolation**: PostgreSQL with Row-Level Security (RLS)
2. **User-Specific Configuration Storage**: Encrypted per-user configurations
3. **Container Lifecycle Management**: Per-user container management

### Recommended Multi-User Catalog Approach

#### Option 1: Database-Backed Dynamic Catalogs (Recommended)

**Implementation**: Extend existing Portal catalog service to generate user-specific catalog files

```go
// cmd/docker-mcp/portal/catalog/service.go (existing)
func (s *Service) GenerateUserCatalog(ctx context.Context, userID string) (*Catalog, error) {
    // Get user's enabled servers from database
    servers, err := s.repository.GetUserServers(ctx, userID)
    if err != nil {
        return nil, err
    }

    // Generate catalog file content
    catalog := &Catalog{
        Registry: make(map[string]ServerDefinition),
    }

    for _, server := range servers {
        catalog.Registry[server.Name] = ServerDefinition{
            Description: server.Description,
            Title:       server.Title,
            Type:        "server",
            Image:       server.Image,
            AllowHosts:  server.AllowHosts,
        }
    }

    return catalog, nil
}
```

**Volume Strategy**:

```yaml
# Portal deployment pattern
services:
  portal:
    volumes:
      - portal-catalogs:/app/catalogs # User-specific catalog storage
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

#### Option 2: Environment Variable Server Lists (Simpler)

**Implementation**: Use CLI `--servers` parameter with user-specific server lists

```bash
# Generated command per user
docker mcp gateway run \
    --servers=user1_server1,user1_server2 \
    --transport=sse \
    --port=8080
```

**Portal Integration**:

```go
// Generate user-specific server list
func (s *Service) GetUserServerList(ctx context.Context, userID string) (string, error) {
    servers, err := s.repository.GetEnabledServers(ctx, userID)
    if err != nil {
        return "", err
    }

    var serverNames []string
    for _, server := range servers {
        serverNames = append(serverNames, server.Name)
    }

    return strings.Join(serverNames, ","), nil
}
```

### Container Isolation Strategy

**User-Specific Container Naming**:

```go
// Docker service container management
func (s *DockerService) StartUserGateway(ctx context.Context, userID string, servers []string) error {
    containerName := fmt.Sprintf("mcp-gateway-%s", userID)

    config := &container.Config{
        Image: "docker/mcp-gateway",
        Cmd: []string{
            fmt.Sprintf("--servers=%s", strings.Join(servers, ",")),
            "--transport=sse",
            fmt.Sprintf("--port=%d", s.getUserPort(userID)),
        },
    }

    hostConfig := &container.HostConfig{
        Binds: []string{
            "/var/run/docker.sock:/var/run/docker.sock:ro",
        },
        NetworkMode: "mcp-portal-network",
    }

    return s.docker.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
}
```

## Compatibility Assessment with Current Portal

### ✅ Fully Compatible Components

1. **Docker Socket Access**: Portal already uses read-only socket mounting
2. **Environment Configuration**: All settings use environment variables
3. **Container Management**: Portal Docker service can manage MCP containers
4. **Network Isolation**: Portal uses custom Docker network for security

### ⚠️ Minor Modifications Required

1. **Catalog File Generation**: Add file-based catalog generation to existing catalog service
2. **User-Specific Ports**: Implement port allocation for multi-user gateways
3. **Container Naming**: Add user ID prefix to container names for isolation

### 🔧 Current Portal Strengths for Multi-User

```yaml
# Existing Portal docker-compose.mcp-portal.yml strengths:

# 1. Docker Engine-only compatibility
volumes:
  - /var/run/docker.sock:/var/run/docker.sock:ro

# 2. Environment-based secrets
environment:
  JWT_SECRET: ${JWT_SECRET:-change_me_to_a_secure_secret}
  POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-change_me}

# 3. Network isolation
networks:
  mcp-network:
    driver: bridge
    name: mcp-portal-network

# 4. User isolation through RLS database
postgres:
  # PostgreSQL with Row-Level Security for user isolation
```

## Specific Recommendations

### 1. Multi-User Catalog Implementation

**Recommended Approach**: Hybrid file-based + environment variable approach

```go
// Portal catalog service enhancement
type UserCatalogManager struct {
    repository *Repository
    docker     *DockerService
    catalogDir string
}

func (m *UserCatalogManager) CreateUserGateway(ctx context.Context, userID string) error {
    // 1. Generate user-specific catalog file
    catalog, err := m.generateUserCatalog(ctx, userID)
    if err != nil {
        return err
    }

    catalogPath := filepath.Join(m.catalogDir, fmt.Sprintf("%s.yaml", userID))
    if err := m.writeCatalogFile(catalogPath, catalog); err != nil {
        return err
    }

    // 2. Start user-specific gateway container
    return m.docker.StartUserGateway(ctx, userID, catalogPath)
}
```

### 2. Secret Management Without Docker Desktop

**Implementation**: Extend existing environment variable approach

```bash
# User-specific environment files
/app/data/users/{userID}/secrets.env
/app/data/users/{userID}/config.yaml
```

**Portal Integration**:

```go
// Secret management for user gateways
func (s *SecretService) GetUserSecrets(ctx context.Context, userID string) (map[string]string, error) {
    // Use existing crypto service for encrypted storage
    encryptedSecrets, err := s.repository.GetUserSecrets(ctx, userID)
    if err != nil {
        return nil, err
    }

    // Decrypt and return as environment variables
    return s.crypto.DecryptSecrets(encryptedSecrets)
}
```

### 3. Container Resource Management

**Implement Resource Limits**: Extend existing Docker service

```go
// Add to existing Portal DockerService
func (s *DockerService) CreateUserContainer(ctx context.Context, userID string, config ContainerConfig) error {
    hostConfig := &container.HostConfig{
        // Resource limits per user
        Memory:      512 * 1024 * 1024, // 512MB
        CPUShares:   512,                // Fair CPU sharing

        // Security constraints
        SecurityOpt: []string{"no-new-privileges"},
        ReadonlyRootfs: true,

        // Network isolation
        NetworkMode: "mcp-portal-network",

        // Docker socket (read-only)
        Binds: []string{
            "/var/run/docker.sock:/var/run/docker.sock:ro",
            fmt.Sprintf("user-%s-catalog:/mcp/catalog.yaml:ro", userID),
        },
    }

    return s.createContainer(ctx, config, hostConfig)
}
```

### 4. Port Management Strategy

**Dynamic Port Allocation**:

```go
// Port allocation service
type PortManager struct {
    startPort int
    endPort   int
    allocated map[string]int
    mutex     sync.RWMutex
}

func (pm *PortManager) AllocateUserPort(userID string) (int, error) {
    pm.mutex.Lock()
    defer pm.mutex.Unlock()

    if port, exists := pm.allocated[userID]; exists {
        return port, nil
    }

    for port := pm.startPort; port <= pm.endPort; port++ {
        if !pm.isPortAllocated(port) {
            pm.allocated[userID] = port
            return port, nil
        }
    }

    return 0, errors.New("no available ports")
}
```

## Implementation Priority

### Phase 1: Foundation (Immediate - 2 weeks)

1. **Extend Portal Catalog Service**: Add file-based catalog generation
2. **User Port Allocation**: Implement dynamic port management
3. **Container Naming**: Add user isolation to container management

### Phase 2: Multi-User Features (4 weeks)

1. **User-Specific Gateways**: Implement per-user gateway containers
2. **Secret Management**: Extend encrypted secret storage for MCP servers
3. **Resource Management**: Add container resource limits and monitoring

### Phase 3: Advanced Features (6 weeks)

1. **Static Mode Support**: Implement pre-composed server containers
2. **Remote MCP Support**: Add proxy support for remote MCP servers
3. **Performance Optimization**: Implement container pooling and reuse

## Security Considerations

### Docker Engine Security

- **Read-only socket access**: Prevents container privilege escalation
- **Network isolation**: User containers isolated in Portal network
- **Resource limits**: Prevent resource exhaustion attacks
- **No-new-privileges**: Container security constraints

### Multi-User Isolation

- **Database RLS**: User data isolation at database level
- **Container naming**: Prevent cross-user container access
- **Port allocation**: Isolated network access per user
- **Encrypted secrets**: User secrets encrypted at rest

## Conclusion

The MCP Gateway system is excellently positioned for Docker Desktop-independent operation, with the current Portal architecture already implementing most required patterns. The recommended multi-user catalog implementation leverages existing Portal capabilities while adding minimal complexity.

**Key Success Factors**:

1. **Proven Patterns**: All recommendations based on existing, tested examples
2. **Minimal Changes**: Leverage existing Portal infrastructure
3. **Security-First**: Maintain current RLS and encryption standards
4. **Scalable Design**: Support growth from single-user to enterprise deployments

The implementation can proceed incrementally, with each phase building on proven Docker Engine-only patterns demonstrated in the examples directory.
