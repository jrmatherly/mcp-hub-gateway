// Server types
export interface Server {
  name: string;
  enabled: boolean;
  status: 'running' | 'stopped' | 'error' | 'unknown' | 'starting';
  container_id?: string;
  image?: string;
  port?: number;
  health_status?: 'healthy' | 'unhealthy' | 'starting';
  last_started?: string;
  last_stopped?: string;
  error_message?: string;
  description?: string;
  version?: string;
  tags?: string[];
  capabilities?: string[];
  resources?: {
    cpu_limit?: string;
    memory_limit?: string;
    cpu_usage?: number;
    memory_usage?: number;
  };
}

// Server details with additional metadata
export interface ServerDetails extends Server {
  logs?: string[];
  environment?: Record<string, string>;
  volumes?: string[];
  networks?: string[];
  created_at?: string;
  updated_at?: string;
}

// Gateway status
export interface GatewayStatus {
  running: boolean;
  port?: number;
  pid?: number;
  uptime?: number;
  version?: string;
  transport?: 'stdio' | 'streaming';
  active_servers?: string[];
  connections?: number;
  requests_handled?: number;
  errors?: number;
}

// Configuration
export interface MCPConfig {
  gateway?: {
    port?: number;
    transport?: 'stdio' | 'streaming';
    log_level?: 'debug' | 'info' | 'warn' | 'error';
    enable_cors?: boolean;
    timeout?: number;
  };
  servers?: Record<string, ServerConfig>;
  secrets?: Record<string, string>;
  catalog?: {
    default_enabled?: boolean;
    auto_update?: boolean;
    cache_ttl?: number;
  };
}

export interface ServerConfig {
  enabled: boolean;
  image?: string;
  port?: number;
  environment?: Record<string, string>;
  volumes?: string[];
  networks?: string[];
  resources?: {
    cpu_limit?: string;
    memory_limit?: string;
  };
  health_check?: {
    enabled: boolean;
    interval?: number;
    timeout?: number;
    retries?: number;
  };
}

// API response wrappers
export interface ApiResponse<T = unknown> {
  success: boolean;
  data?: T;
  message?: string;
  error?: string;
  timestamp?: string;
}

export interface PaginatedResponse<T> extends ApiResponse<T[]> {
  pagination?: {
    page: number;
    limit: number;
    total: number;
    total_pages: number;
  };
}

// API error types
export interface ApiError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
  timestamp?: string;
  request_id?: string;
}

// Request types
export interface ServerToggleRequest {
  enabled: boolean;
}

export interface ConfigUpdateRequest {
  config: Partial<MCPConfig>;
}

export interface GatewayStartRequest {
  port?: number;
  transport?: 'stdio' | 'streaming';
  servers?: string[];
}

// Bulk operations
export interface BulkServerOperation {
  server_names: string[];
  operation: 'enable' | 'disable' | 'restart' | 'delete';
}

export interface BulkOperationResult {
  success_count: number;
  error_count: number;
  results: Array<{
    server_name: string;
    success: boolean;
    error?: string;
  }>;
}

// Filter and search types
export interface ServerFilters {
  status?: Server['status'][];
  enabled?: boolean;
  health_status?: Server['health_status'][];
  tags?: string[];
  search?: string;
}

export interface ServerSortOptions {
  field: 'name' | 'status' | 'last_started' | 'enabled';
  direction: 'asc' | 'desc';
}

// Real-time update types
export interface ServerStatusUpdate {
  server_name: string;
  status: Server['status'];
  health_status?: Server['health_status'];
  timestamp: string;
}

export interface GatewayStatusUpdate {
  running: boolean;
  connections?: number;
  active_servers?: string[];
  timestamp: string;
}

// Catalog types
export interface CatalogServer {
  name: string;
  image: string;
  description: string;
  version: string;
  tags: string[];
  capabilities: string[];
  author?: string;
  repository?: string;
  documentation?: string;
  license?: string;
  default_config?: Partial<ServerConfig>;
}

export interface Catalog {
  name: string;
  description: string;
  version: string;
  servers: CatalogServer[];
  last_updated: string;
}
