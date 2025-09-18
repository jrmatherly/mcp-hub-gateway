'use client';

import { authService } from '@/services/auth.service';
import { apiLogger } from '@/lib/logger';
import type {
  RealtimeConfig,
  RealtimeEvent,
  RealtimeError,
  WebSocketMessage,
  WebSocketResponse,
  EventFilter,
  ConnectionState,
} from '@/types/realtime';

// Default configuration
const DEFAULT_CONFIG: Required<RealtimeConfig> = {
  baseUrl: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080',
  autoReconnect: true,
  reconnectInterval: 5000,
  maxReconnectAttempts: 10,
  heartbeatInterval: 30000,
  subscribeTo: ['servers', 'gateway'],
  filter: {},
};

/**
 * WebSocket client for real-time communication
 */
export class RealtimeWebSocketClient {
  private ws: WebSocket | null = null;
  private config: Required<RealtimeConfig>;
  private state: ConnectionState;
  private eventHandlers = new Map<
    string,
    Set<(event: RealtimeEvent) => void>
  >();
  private stateHandlers = new Set<(state: ConnectionState) => void>();
  private reconnectTimer: NodeJS.Timeout | null = null;
  private heartbeatTimer: NodeJS.Timeout | null = null;
  private lastPongTime: number = 0;
  private subscribedChannels = new Set<string>();

  constructor(config: Partial<RealtimeConfig> = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config };
    this.state = {
      status: 'disconnected',
      reconnectAttempts: 0,
    };
  }

  /**
   * Connect to WebSocket server
   */
  async connect(): Promise<void> {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    try {
      this.updateState({ status: 'connecting' });

      const token = await authService.getAccessToken();
      if (!token) {
        throw this.createError(
          'AUTH_FAILED',
          'No authentication token available',
          false
        );
      }

      const wsUrl = this.config.baseUrl.replace(/^http/, 'ws') + '/api/v1/ws';
      const url = new URL(wsUrl);
      url.searchParams.set('token', token);

      this.ws = new WebSocket(url.toString());
      this.setupWebSocketHandlers();
    } catch (error) {
      const realtimeError =
        error instanceof Error && 'code' in error
          ? (error as RealtimeError)
          : this.createError(
              'CONNECTION_FAILED',
              `Failed to connect: ${error}`,
              true
            );

      this.handleError(realtimeError);
    }
  }

  /**
   * Disconnect from WebSocket server
   */
  disconnect(): void {
    this.clearTimers();
    this.subscribedChannels.clear();

    if (this.ws) {
      this.ws.close(1000, 'Client disconnect');
      this.ws = null;
    }

    this.updateState({ status: 'disconnected', reconnectAttempts: 0 });
  }

  /**
   * Subscribe to event channels
   */
  subscribe(channels: string[], filter?: EventFilter): void {
    if (!this.isConnected()) {
      apiLogger.warn('Cannot subscribe: WebSocket not connected');
      return;
    }

    const message: WebSocketMessage = {
      action: 'subscribe',
      payload: { channels, filter },
    };

    this.sendMessage(message);
    channels.forEach(channel => this.subscribedChannels.add(channel));
  }

  /**
   * Unsubscribe from event channels
   */
  unsubscribe(channels?: string[]): void {
    if (!this.isConnected()) {
      return;
    }

    const channelsToUnsubscribe =
      channels || Array.from(this.subscribedChannels);

    const message: WebSocketMessage = {
      action: 'unsubscribe',
      payload: { channels: channelsToUnsubscribe },
    };

    this.sendMessage(message);

    if (channels) {
      channels.forEach(channel => this.subscribedChannels.delete(channel));
    } else {
      this.subscribedChannels.clear();
    }
  }

  /**
   * Send message to WebSocket server
   */
  sendMessage(message: WebSocketMessage): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      apiLogger.warn('Cannot send message: WebSocket not connected', message);
      return;
    }

    try {
      this.ws.send(JSON.stringify(message));
    } catch (error) {
      apiLogger.error('Failed to send WebSocket message:', error);
    }
  }

  /**
   * Add event listener for specific event types
   */
  addEventListener(
    type: RealtimeEvent['type'],
    handler: (event: RealtimeEvent) => void
  ): () => void {
    if (!this.eventHandlers.has(type)) {
      this.eventHandlers.set(type, new Set());
    }

    const handlers = this.eventHandlers.get(type);
    if (handlers) {
      handlers.add(handler);
    }

    // Return unsubscribe function
    return () => {
      const handlers = this.eventHandlers.get(type);
      if (handlers) {
        handlers.delete(handler);
        if (handlers.size === 0) {
          this.eventHandlers.delete(type);
        }
      }
    };
  }

  /**
   * Add state change listener
   */
  addStateListener(handler: (state: ConnectionState) => void): () => void {
    this.stateHandlers.add(handler);

    // Return unsubscribe function
    return () => {
      this.stateHandlers.delete(handler);
    };
  }

  /**
   * Get current connection state
   */
  getState(): ConnectionState {
    return { ...this.state };
  }

  /**
   * Check if WebSocket is connected
   */
  isConnected(): boolean {
    return (
      this.ws?.readyState === WebSocket.OPEN &&
      this.state.status === 'connected'
    );
  }

  /**
   * Setup WebSocket event handlers
   */
  private setupWebSocketHandlers(): void {
    if (!this.ws) return;

    this.ws.onopen = () => {
      apiLogger.info('WebSocket connected');
      this.updateState({
        status: 'connected',
        lastConnected: new Date(),
        reconnectAttempts: 0,
        error: undefined,
      });

      this.startHeartbeat();

      // Re-subscribe to channels if reconnecting
      if (this.subscribedChannels.size > 0) {
        this.subscribe(Array.from(this.subscribedChannels), this.config.filter);
      } else {
        // Initial subscription
        this.subscribe(this.config.subscribeTo, this.config.filter);
      }
    };

    this.ws.onmessage = event => {
      try {
        const response: WebSocketResponse = JSON.parse(event.data);
        this.handleWebSocketMessage(response);
      } catch (error) {
        apiLogger.error('Failed to parse WebSocket message:', error);
      }
    };

    this.ws.onclose = event => {
      const wasConnected = this.state.status === 'connected';
      apiLogger.info(`WebSocket closed: ${event.code} ${event.reason}`);

      this.clearTimers();

      if (event.code !== 1000 && wasConnected && this.config.autoReconnect) {
        this.scheduleReconnect();
      } else {
        this.updateState({
          status: 'disconnected',
          lastDisconnected: new Date(),
        });
      }
    };

    this.ws.onerror = event => {
      apiLogger.error('WebSocket error:', event);
      const error = this.createError(
        'CONNECTION_FAILED',
        'WebSocket connection error',
        true
      );
      this.handleError(error);
    };
  }

  /**
   * Handle incoming WebSocket messages
   */
  private handleWebSocketMessage(response: WebSocketResponse): void {
    switch (response.type) {
      case 'event':
        if (response.event) {
          this.emitEvent(response.event);
        }
        break;

      case 'pong':
        this.lastPongTime = Date.now();
        break;

      case 'error':
        if (response.error) {
          const error = this.createError(
            'UNKNOWN',
            response.error.message,
            false
          );
          this.handleError(error);
        }
        break;

      case 'ack':
        // Handle acknowledgment if needed
        break;

      default:
        apiLogger.warn('Unknown WebSocket message type:', response);
    }
  }

  /**
   * Emit event to registered handlers
   */
  private emitEvent(event: RealtimeEvent): void {
    const handlers = this.eventHandlers.get(event.type);
    if (handlers) {
      handlers.forEach(handler => {
        try {
          handler(event);
        } catch (error) {
          apiLogger.error('Error in event handler:', error);
        }
      });
    }
  }

  /**
   * Update connection state and notify listeners
   */
  private updateState(updates: Partial<ConnectionState>): void {
    this.state = { ...this.state, ...updates };

    this.stateHandlers.forEach(handler => {
      try {
        handler(this.state);
      } catch (error) {
        apiLogger.error('Error in state handler:', error);
      }
    });
  }

  /**
   * Start heartbeat to keep connection alive
   */
  private startHeartbeat(): void {
    this.clearHeartbeat();

    this.heartbeatTimer = setInterval(() => {
      if (this.isConnected()) {
        this.sendMessage({ action: 'ping' });

        // Check if we received a pong recently
        const timeSinceLastPong = Date.now() - this.lastPongTime;
        if (timeSinceLastPong > this.config.heartbeatInterval * 2) {
          apiLogger.warn('Heartbeat timeout, closing connection');
          this.ws?.close(1006, 'Heartbeat timeout');
        }
      }
    }, this.config.heartbeatInterval);
  }

  /**
   * Schedule reconnection attempt
   */
  private scheduleReconnect(): void {
    if (this.state.reconnectAttempts >= this.config.maxReconnectAttempts) {
      const error = this.createError(
        'CONNECTION_FAILED',
        'Max reconnection attempts reached',
        false
      );
      this.handleError(error);
      return;
    }

    this.updateState({
      status: 'reconnecting',
      reconnectAttempts: this.state.reconnectAttempts + 1,
    });

    this.reconnectTimer = setTimeout(() => {
      apiLogger.info(
        `Reconnecting... (attempt ${this.state.reconnectAttempts})`
      );
      this.connect();
    }, this.config.reconnectInterval);
  }

  /**
   * Handle errors
   */
  private handleError(error: RealtimeError): void {
    apiLogger.error('WebSocket error:', error);

    this.updateState({
      status: 'error',
      error: error.message,
      lastDisconnected: new Date(),
    });

    if (error.retryable && this.config.autoReconnect) {
      this.scheduleReconnect();
    }
  }

  /**
   * Create a RealtimeError
   */
  private createError(
    code: RealtimeError['code'],
    message: string,
    retryable: boolean
  ): RealtimeError {
    const error = new Error(message) as RealtimeError;
    error.name = 'RealtimeError';
    error.code = code;
    error.retryable = retryable;
    return error;
  }

  /**
   * Clear all timers
   */
  private clearTimers(): void {
    this.clearReconnectTimer();
    this.clearHeartbeat();
  }

  /**
   * Clear reconnect timer
   */
  private clearReconnectTimer(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  /**
   * Clear heartbeat timer
   */
  private clearHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }
}

/**
 * Server-Sent Events client for real-time communication
 */
export class RealtimeSSEClient {
  private eventSource: EventSource | null = null;
  private config: Required<RealtimeConfig>;
  private state: ConnectionState;
  private eventHandlers = new Map<
    string,
    Set<(event: RealtimeEvent) => void>
  >();
  private stateHandlers = new Set<(state: ConnectionState) => void>();
  private reconnectTimer: NodeJS.Timeout | null = null;
  private events: RealtimeEvent[] = [];
  private maxEvents = 1000;

  constructor(config: Partial<RealtimeConfig> = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config };
    this.state = {
      status: 'disconnected',
      reconnectAttempts: 0,
    };
  }

  /**
   * Connect to SSE endpoint
   */
  async connect(): Promise<void> {
    if (this.eventSource?.readyState === EventSource.OPEN) {
      return;
    }

    try {
      this.updateState({ status: 'connecting' });

      const token = await authService.getAccessToken();
      if (!token) {
        throw this.createError(
          'AUTH_FAILED',
          'No authentication token available',
          false
        );
      }

      const sseUrl = this.config.baseUrl + '/api/v1/sse';
      const url = new URL(sseUrl);
      url.searchParams.set('token', token);

      // Add channel subscriptions
      if (this.config.subscribeTo.length > 0) {
        url.searchParams.set('channels', this.config.subscribeTo.join(','));
      }

      // Add filters
      if (this.config.filter.event_types?.length) {
        url.searchParams.set(
          'event_types',
          this.config.filter.event_types.join(',')
        );
      }
      if (this.config.filter.sources?.length) {
        url.searchParams.set('sources', this.config.filter.sources.join(','));
      }

      this.eventSource = new EventSource(url.toString());
      this.setupSSEHandlers();
    } catch (error) {
      const realtimeError =
        error instanceof Error && 'code' in error
          ? (error as RealtimeError)
          : this.createError(
              'CONNECTION_FAILED',
              `Failed to connect: ${error}`,
              true
            );

      this.handleError(realtimeError);
    }
  }

  /**
   * Disconnect from SSE
   */
  disconnect(): void {
    this.clearReconnectTimer();

    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }

    this.updateState({ status: 'disconnected', reconnectAttempts: 0 });
  }

  /**
   * Clear stored events
   */
  clearEvents(): void {
    this.events = [];
  }

  /**
   * Get all received events
   */
  getEvents(): RealtimeEvent[] {
    return [...this.events];
  }

  /**
   * Add event listener for specific event types
   */
  addEventListener(
    type: RealtimeEvent['type'],
    handler: (event: RealtimeEvent) => void
  ): () => void {
    if (!this.eventHandlers.has(type)) {
      this.eventHandlers.set(type, new Set());
    }

    const handlers = this.eventHandlers.get(type);
    if (handlers) {
      handlers.add(handler);
    }

    // Return unsubscribe function
    return () => {
      const handlers = this.eventHandlers.get(type);
      if (handlers) {
        handlers.delete(handler);
        if (handlers.size === 0) {
          this.eventHandlers.delete(type);
        }
      }
    };
  }

  /**
   * Add state change listener
   */
  addStateListener(handler: (state: ConnectionState) => void): () => void {
    this.stateHandlers.add(handler);

    // Return unsubscribe function
    return () => {
      this.stateHandlers.delete(handler);
    };
  }

  /**
   * Get current connection state
   */
  getState(): ConnectionState {
    return { ...this.state };
  }

  /**
   * Check if SSE is connected
   */
  isConnected(): boolean {
    return (
      this.eventSource?.readyState === EventSource.OPEN &&
      this.state.status === 'connected'
    );
  }

  /**
   * Setup SSE event handlers
   */
  private setupSSEHandlers(): void {
    if (!this.eventSource) return;

    this.eventSource.onopen = () => {
      apiLogger.info('SSE connected');
      this.updateState({
        status: 'connected',
        lastConnected: new Date(),
        reconnectAttempts: 0,
        error: undefined,
      });
    };

    this.eventSource.onmessage = event => {
      try {
        const realtimeEvent: RealtimeEvent = JSON.parse(event.data);
        this.handleEvent(realtimeEvent);
      } catch (error) {
        apiLogger.error('Failed to parse SSE message:', error);
      }
    };

    this.eventSource.onerror = () => {
      const wasConnected = this.state.status === 'connected';
      apiLogger.error('SSE connection error');

      if (wasConnected && this.config.autoReconnect) {
        this.scheduleReconnect();
      } else {
        const error = this.createError(
          'CONNECTION_FAILED',
          'SSE connection error',
          true
        );
        this.handleError(error);
      }
    };
  }

  /**
   * Handle incoming SSE events
   */
  private handleEvent(event: RealtimeEvent): void {
    // Store event in memory
    this.events.push(event);

    // Limit memory usage
    if (this.events.length > this.maxEvents) {
      this.events = this.events.slice(-this.maxEvents);
    }

    // Emit to handlers
    this.emitEvent(event);
  }

  /**
   * Emit event to registered handlers
   */
  private emitEvent(event: RealtimeEvent): void {
    const handlers = this.eventHandlers.get(event.type);
    if (handlers) {
      handlers.forEach(handler => {
        try {
          handler(event);
        } catch (error) {
          apiLogger.error('Error in event handler:', error);
        }
      });
    }
  }

  /**
   * Update connection state and notify listeners
   */
  private updateState(updates: Partial<ConnectionState>): void {
    this.state = { ...this.state, ...updates };

    this.stateHandlers.forEach(handler => {
      try {
        handler(this.state);
      } catch (error) {
        apiLogger.error('Error in state handler:', error);
      }
    });
  }

  /**
   * Schedule reconnection attempt
   */
  private scheduleReconnect(): void {
    if (this.state.reconnectAttempts >= this.config.maxReconnectAttempts) {
      const error = this.createError(
        'CONNECTION_FAILED',
        'Max reconnection attempts reached',
        false
      );
      this.handleError(error);
      return;
    }

    this.updateState({
      status: 'reconnecting',
      reconnectAttempts: this.state.reconnectAttempts + 1,
    });

    this.reconnectTimer = setTimeout(() => {
      apiLogger.info(
        `Reconnecting SSE... (attempt ${this.state.reconnectAttempts})`
      );
      this.connect();
    }, this.config.reconnectInterval);
  }

  /**
   * Handle errors
   */
  private handleError(error: RealtimeError): void {
    apiLogger.error('SSE error:', error);

    this.updateState({
      status: 'error',
      error: error.message,
      lastDisconnected: new Date(),
    });

    if (error.retryable && this.config.autoReconnect) {
      this.scheduleReconnect();
    }
  }

  /**
   * Create a RealtimeError
   */
  private createError(
    code: RealtimeError['code'],
    message: string,
    retryable: boolean
  ): RealtimeError {
    const error = new Error(message) as RealtimeError;
    error.name = 'RealtimeError';
    error.code = code;
    error.retryable = retryable;
    return error;
  }

  /**
   * Clear reconnect timer
   */
  private clearReconnectTimer(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }
}
