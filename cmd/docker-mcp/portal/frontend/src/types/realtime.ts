// Real-time communication types for WebSocket and SSE

import type { Server, GatewayStatus } from './api';

// Base event types
export interface BaseRealtimeEvent {
  id: string;
  timestamp: string;
  type: string;
  source: 'gateway' | 'server' | 'system';
}

// Server events
export interface ServerStatusEvent extends BaseRealtimeEvent {
  type: 'server_status_changed';
  source: 'server';
  data: {
    server_name: string;
    status: Server['status'];
    health_status?: Server['health_status'];
    container_id?: string;
    error_message?: string;
  };
}

export interface ServerLogEvent extends BaseRealtimeEvent {
  type: 'server_log';
  source: 'server';
  data: {
    server_name: string;
    level: 'debug' | 'info' | 'warn' | 'error';
    message: string;
    container_id?: string;
  };
}

export interface ServerMetricsEvent extends BaseRealtimeEvent {
  type: 'server_metrics';
  source: 'server';
  data: {
    server_name: string;
    cpu_usage?: number;
    memory_usage?: number;
    memory_limit?: string;
    cpu_limit?: string;
  };
}

// Gateway events
export interface GatewayStatusEvent extends BaseRealtimeEvent {
  type: 'gateway_status_changed';
  source: 'gateway';
  data: GatewayStatus;
}

export interface GatewayConnectionEvent extends BaseRealtimeEvent {
  type: 'gateway_connection';
  source: 'gateway';
  data: {
    action: 'connected' | 'disconnected';
    client_count: number;
  };
}

export interface GatewayRequestEvent extends BaseRealtimeEvent {
  type: 'gateway_request';
  source: 'gateway';
  data: {
    method: string;
    endpoint: string;
    status_code: number;
    response_time: number;
    user_id?: string;
  };
}

// System events
export interface SystemHealthEvent extends BaseRealtimeEvent {
  type: 'system_health';
  source: 'system';
  data: {
    cpu_usage: number;
    memory_usage: number;
    disk_usage: number;
    docker_status: 'running' | 'stopped' | 'error';
  };
}

export interface SystemErrorEvent extends BaseRealtimeEvent {
  type: 'system_error';
  source: 'system';
  data: {
    error_type: string;
    message: string;
    stack_trace?: string;
    context?: Record<string, unknown>;
  };
}

// Union type for all events
export type RealtimeEvent =
  | ServerStatusEvent
  | ServerLogEvent
  | ServerMetricsEvent
  | GatewayStatusEvent
  | GatewayConnectionEvent
  | GatewayRequestEvent
  | SystemHealthEvent
  | SystemErrorEvent;

// WebSocket specific types
export interface WebSocketMessage {
  action: 'subscribe' | 'unsubscribe' | 'ping' | 'pong';
  payload?: {
    channels?: string[];
    filter?: EventFilter;
  };
}

export interface WebSocketResponse {
  type: 'event' | 'ack' | 'error' | 'pong';
  event?: RealtimeEvent;
  error?: {
    code: string;
    message: string;
  };
}

// Event filtering
export interface EventFilter {
  event_types?: RealtimeEvent['type'][];
  sources?: RealtimeEvent['source'][];
  server_names?: string[];
  min_level?: 'debug' | 'info' | 'warn' | 'error';
}

// Connection configuration
export interface RealtimeConfig {
  baseUrl?: string;
  autoReconnect?: boolean;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
  heartbeatInterval?: number;
  subscribeTo?: string[];
  filter?: EventFilter;
}

// Connection state
export interface ConnectionState {
  status:
    | 'connecting'
    | 'connected'
    | 'disconnected'
    | 'error'
    | 'reconnecting';
  lastConnected?: Date;
  lastDisconnected?: Date;
  reconnectAttempts: number;
  error?: string;
}

// Hook return types
export interface UseWebSocketReturn {
  // Connection state
  connectionState: ConnectionState;
  isConnected: boolean;
  lastEvent?: RealtimeEvent;

  // Actions
  connect: () => void;
  disconnect: () => void;
  subscribe: (channels: string[], filter?: EventFilter) => void;
  unsubscribe: (channels?: string[]) => void;
  sendMessage: (message: WebSocketMessage) => void;

  // Event handlers
  addEventListener: (
    type: RealtimeEvent['type'],
    handler: (event: RealtimeEvent) => void
  ) => () => void;
}

export interface UseSSEReturn {
  // Connection state
  connectionState: ConnectionState;
  isConnected: boolean;
  lastEvent?: RealtimeEvent;
  events: RealtimeEvent[];

  // Actions
  connect: () => void;
  disconnect: () => void;
  clearEvents: () => void;

  // Event handlers
  addEventListener: (
    type: RealtimeEvent['type'],
    handler: (event: RealtimeEvent) => void
  ) => () => void;
}

// Channel types for subscription
export type ChannelType =
  | 'servers' // All server events
  | 'gateway' // Gateway status and metrics
  | 'system' // System health and errors
  | 'logs' // All log events
  | 'metrics' // All metrics events
  | `server:${string}` // Specific server events
  | `user:${string}`; // User-specific events

// Error types
export interface RealtimeError extends Error {
  code:
    | 'CONNECTION_FAILED'
    | 'AUTH_FAILED'
    | 'SUBSCRIPTION_FAILED'
    | 'PARSE_ERROR'
    | 'UNKNOWN';
  retryable: boolean;
}
