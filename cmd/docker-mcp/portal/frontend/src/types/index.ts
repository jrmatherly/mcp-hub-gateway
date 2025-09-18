// =================================================================
// MCP Portal Frontend Type Definitions
// =================================================================

// ----------------------------------------
// Base Types
// ----------------------------------------
export interface BaseEntity {
  id: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface ApiResponse<T = unknown> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

export interface PaginatedResponse<T> extends ApiResponse<T[]> {
  pagination?: {
    page: number;
    limit: number;
    total: number;
    pages: number;
  };
}

// ----------------------------------------
// Authentication Types
// ----------------------------------------
export interface User extends BaseEntity {
  email: string;
  name: string;
  picture?: string;
  roles: string[];
  tenantId: string;
}

export interface AuthState {
  isAuthenticated: boolean;
  user: User | null;
  token: string | null;
  isLoading: boolean;
  error: string | null;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface TokenResponse {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  user: User;
}

// ----------------------------------------
// MCP Server Types
// ----------------------------------------
// Note: These enums are defined for future use in server management
export enum ServerStatus {
  ENABLED = 'enabled',
  DISABLED = 'disabled',
  RUNNING = 'running',
  STOPPED = 'stopped',
  ERROR = 'error',
  UNKNOWN = 'unknown',
}

export enum ServerHealth {
  HEALTHY = 'healthy',
  UNHEALTHY = 'unhealthy',
  DEGRADED = 'degraded',
  UNKNOWN = 'unknown',
}

export interface ServerConfig {
  name: string;
  command?: string;
  args?: string[];
  env?: Record<string, string>;
  image?: string;
  version?: string;
  description?: string;
  category?: string;
  tags?: string[];
  dependencies?: string[];
  ports?: number[];
  volumes?: string[];
  networks?: string[];
  healthCheck?: {
    enabled: boolean;
    interval: number;
    timeout: number;
    retries: number;
    path?: string;
  };
  resources?: {
    memory?: string;
    cpu?: string;
  };
}

export interface Server extends BaseEntity {
  name: string;
  status: ServerStatus;
  health: ServerHealth;
  config: ServerConfig;
  containerId?: string;
  lastStarted?: Date;
  lastStopped?: Date;
  errorMessage?: string;
  metrics?: {
    cpuUsage: number;
    memoryUsage: number;
    networkIn: number;
    networkOut: number;
  };
  logs?: LogEntry[];
}

export interface ServerStats {
  total: number;
  enabled: number;
  disabled: number;
  running: number;
  stopped: number;
  error: number;
}

// ----------------------------------------
// Catalog Types
// ----------------------------------------
export interface CatalogServer {
  name: string;
  description: string;
  category: string;
  tags: string[];
  version: string;
  author: string;
  repository?: string;
  documentation?: string;
  license?: string;
  config: ServerConfig;
  dependencies?: string[];
  screenshots?: string[];
  featured?: boolean;
  verified?: boolean;
  downloads?: number;
  rating?: number;
}

export interface Catalog {
  name: string;
  description: string;
  url: string;
  version: string;
  servers: CatalogServer[];
  lastUpdated: Date;
}

// ----------------------------------------
// Configuration Types
// ----------------------------------------
export interface UserConfig {
  userId: string;
  theme: 'light' | 'dark' | 'system';
  language: string;
  timezone: string;
  notifications: {
    enabled: boolean;
    email: boolean;
    push: boolean;
    serverStatus: boolean;
    systemUpdates: boolean;
  };
  dashboard: {
    layout: 'grid' | 'list';
    pageSize: number;
    defaultSort: string;
    showMetrics: boolean;
  };
  advanced: {
    enableDebugMode: boolean;
    autoRefresh: boolean;
    refreshInterval: number;
  };
}

// ----------------------------------------
// Real-time Types
// ----------------------------------------

export enum EventType {
  SERVER_STATUS_CHANGED = 'server_status_changed',
  SERVER_HEALTH_CHANGED = 'server_health_changed',
  SERVER_LOGS = 'server_logs',
  CONTAINER_CREATED = 'container_created',
  CONTAINER_STARTED = 'container_started',
  CONTAINER_STOPPED = 'container_stopped',
  CONTAINER_REMOVED = 'container_removed',
  SYSTEM_ALERT = 'system_alert',
}

export interface RealtimeEvent {
  type: EventType;
  timestamp: Date;
  serverId?: string;
  data: unknown;
}

export interface ServerStatusEvent {
  serverId: string;
  status: ServerStatus;
  previousStatus: ServerStatus;
  timestamp: Date;
}

export interface ServerHealthEvent {
  serverId: string;
  health: ServerHealth;
  previousHealth: ServerHealth;
  timestamp: Date;
  metrics?: Server['metrics'];
}

export interface LogEntry {
  id: string;
  timestamp: Date;
  level: 'debug' | 'info' | 'warn' | 'error';
  message: string;
  source: string;
  metadata?: Record<string, unknown>;
}

// ----------------------------------------
// Bulk Operations Types
// ----------------------------------------

export enum BulkOperationType {
  ENABLE = 'enable',
  DISABLE = 'disable',
  START = 'start',
  STOP = 'stop',
  DELETE = 'delete',
  UPDATE_CONFIG = 'update_config',
}

export interface BulkOperation {
  id: string;
  type: BulkOperationType;
  serverIds: string[];
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  progress: number;
  startedAt?: Date;
  completedAt?: Date;
  results: BulkOperationResult[];
  options?: Record<string, unknown>;
}

export interface BulkOperationResult {
  serverId: string;
  serverName: string;
  success: boolean;
  error?: string;
  timestamp: Date;
}

// ----------------------------------------
// UI Component Types
// ----------------------------------------
export interface Toast {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title: string;
  message?: string;
  duration?: number;
  dismissible?: boolean;
  actions?: ToastAction[];
}

export interface ToastAction {
  label: string;
  action: () => void;
}

export interface TableColumn<T> {
  key: keyof T;
  label: string;
  sortable?: boolean;
  width?: string;
  align?: 'left' | 'center' | 'right';
  render?: (_value: T[keyof T], _row: T) => React.ReactNode;
}

export interface FilterOption {
  key: string;
  label: string;
  value: string;
  count?: number;
}

export interface SortOption {
  key: string;
  label: string;
  direction: 'asc' | 'desc';
}

// ----------------------------------------
// Hook Types
// ----------------------------------------
export interface UseApiOptions {
  enabled?: boolean;
  refetchInterval?: number;
  retry?: number;
  onSuccess?: (_data: unknown) => void;
  onError?: (_error: Error) => void;
}

export interface UseWebSocketOptions {
  url: string;
  protocols?: string[];
  onOpen?: () => void;
  onClose?: () => void;
  onError?: (_error: Event) => void;
  onMessage?: (_message: MessageEvent) => void;
  reconnect?: boolean;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
}

// ----------------------------------------
// Store Types
// ----------------------------------------
export interface AppState {
  theme: 'light' | 'dark' | 'system';
  sidebarOpen: boolean;
  loading: boolean;
  error: string | null;
  notifications: Toast[];
}

export interface ServerStore {
  servers: Server[];
  selectedServers: string[];
  filters: {
    status: ServerStatus[];
    category: string[];
    search: string;
  };
  sort: SortOption;
  view: 'grid' | 'list';
  loading: boolean;
  error: string | null;
}

// ----------------------------------------
// Form Types
// ----------------------------------------
export interface ServerFormData {
  name: string;
  image: string;
  version: string;
  description: string;
  category: string;
  tags: string[];
  config: {
    command?: string;
    args: string[];
    env: Array<{ key: string; value: string }>;
    ports: Array<{ host: number; container: number }>;
    volumes: Array<{ host: string; container: string }>;
  };
  resources: {
    memory: string;
    cpu: string;
  };
  healthCheck: {
    enabled: boolean;
    path: string;
    interval: number;
    timeout: number;
    retries: number;
  };
}

export interface ValidationError {
  field: string;
  message: string;
}

export interface FormState<T> {
  data: T;
  errors: ValidationError[];
  isValid: boolean;
  isDirty: boolean;
  isSubmitting: boolean;
}

// ----------------------------------------
// Utility Types
// ----------------------------------------
export type DeepPartial<T> = {
  [P in keyof T]?: T[P] extends object ? DeepPartial<T[P]> : T[P];
};

export type Omit<T, K extends keyof T> = Pick<T, Exclude<keyof T, K>>;

export type Optional<T, K extends keyof T> = Omit<T, K> & Partial<Pick<T, K>>;

export type WithId<T> = T & { id: string };

export type WithTimestamps<T> = T & {
  createdAt: Date;
  updatedAt: Date;
};

// ----------------------------------------
// Next.js Types
// ----------------------------------------
export interface PageProps {
  params: { [key: string]: string | string[] };
  searchParams: { [key: string]: string | string[] | undefined };
}

export interface LayoutProps {
  children: React.ReactNode;
  params?: { [key: string]: string | string[] };
}

// ----------------------------------------
// Error Types
// ----------------------------------------
export class AppError extends Error {
  constructor(
    message: string,
    public code?: string,
    public status?: number
  ) {
    super(message);
    this.name = 'AppError';
    // Note: code and status are stored as public properties for external use
    void code; // Suppress unused variable warning
    void status; // Suppress unused variable warning
  }
}

export interface ErrorBoundaryState {
  hasError: boolean;
  error?: Error;
  errorInfo?: React.ErrorInfo;
}

// ----------------------------------------
// Enhanced Real-time Types
// ----------------------------------------
// Re-export enhanced realtime types for WebSocket and SSE
export * from './realtime';

// Re-export API types for convenience
export * from './api';
