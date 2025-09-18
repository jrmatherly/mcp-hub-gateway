'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { RealtimeWebSocketClient } from '@/lib/realtime-client';
import { serverKeys } from '@/hooks/api/use-servers';
import type {
  UseWebSocketReturn,
  RealtimeConfig,
  RealtimeEvent,
  EventFilter,
  ConnectionState,
  ChannelType,
  WebSocketMessage,
} from '@/types/realtime';
import type { Server } from '@/types/api';

/**
 * Hook for WebSocket real-time communication
 */
export function useWebSocket(
  config: Partial<RealtimeConfig> = {}
): UseWebSocketReturn {
  const queryClient = useQueryClient();
  const clientRef = useRef<RealtimeWebSocketClient | null>(null);
  const [connectionState, setConnectionState] = useState<ConnectionState>({
    status: 'disconnected',
    reconnectAttempts: 0,
  });
  const [lastEvent, setLastEvent] = useState<RealtimeEvent | undefined>();
  const eventHandlersRef = useRef(
    new Map<RealtimeEvent['type'], Set<(event: RealtimeEvent) => void>>()
  );

  // Handle server status updates
  const handleServerStatusUpdate = useCallback(
    (event: RealtimeEvent) => {
      if (event.type !== 'server_status_changed') return;

      const { server_name, status, health_status, error_message } = event.data;

      // Update servers list cache optimistically
      queryClient.setQueryData<Server[]>(
        serverKeys.lists(),
        old =>
          old?.map(server =>
            server.name === server_name
              ? {
                  ...server,
                  status,
                  health_status,
                  error_message,
                  last_updated: event.timestamp,
                }
              : server
          ) || []
      );

      // Update server details cache if it exists
      queryClient.setQueryData(
        serverKeys.detail(server_name),
        (old: Server | undefined) =>
          old
            ? {
                ...old,
                status,
                health_status,
                error_message,
                last_updated: event.timestamp,
              }
            : undefined
      );

      // Show toast for significant status changes
      if (status === 'error' && error_message) {
        toast.error(`Server ${server_name} error: ${error_message}`);
      } else if (status === 'running') {
        toast.success(`Server ${server_name} is now running`);
      } else if (status === 'stopped') {
        toast.info(`Server ${server_name} has stopped`);
      }
    },
    [queryClient]
  );

  // Handle gateway status updates
  const handleGatewayStatusUpdate = useCallback((event: RealtimeEvent) => {
    if (event.type !== 'gateway_status_changed') return;

    // Could update gateway status cache here
    // For now, just show notifications for significant changes
    const gatewayData = event.data;

    if (!gatewayData.running) {
      toast.warning('Gateway has stopped');
    }
  }, []);

  // Handle server metrics updates
  const handleServerMetricsUpdate = useCallback(
    (event: RealtimeEvent) => {
      if (event.type !== 'server_metrics') return;

      const { server_name, cpu_usage, memory_usage, memory_limit, cpu_limit } =
        event.data;

      // Update server caches with new metrics
      queryClient.setQueryData<Server[]>(
        serverKeys.lists(),
        old =>
          old?.map(server =>
            server.name === server_name
              ? {
                  ...server,
                  resources: {
                    ...server.resources,
                    cpu_usage,
                    memory_usage,
                    memory_limit,
                    cpu_limit,
                  },
                }
              : server
          ) || []
      );

      // Warn on high resource usage
      if (cpu_usage && cpu_usage > 90) {
        toast.warning(`Server ${server_name} CPU usage is high: ${cpu_usage}%`);
      }
      if (memory_usage && memory_usage > 90) {
        toast.warning(
          `Server ${server_name} memory usage is high: ${memory_usage}%`
        );
      }
    },
    [queryClient]
  );

  // Handle server log events
  const handleServerLogEvent = useCallback((event: RealtimeEvent) => {
    if (event.type !== 'server_log') return;

    const { server_name, level, message } = event.data;

    // Show toast for error level logs
    if (level === 'error') {
      toast.error(`${server_name}: ${message}`);
    }
  }, []);

  // Handle system error events
  const handleSystemErrorEvent = useCallback((event: RealtimeEvent) => {
    if (event.type !== 'system_error') return;

    const { error_type, message } = event.data;
    toast.error(`System Error (${error_type}): ${message}`);
  }, []);

  // Initialize client
  useEffect(() => {
    clientRef.current = new RealtimeWebSocketClient(config);

    // Set up state listener
    const unsubscribeState =
      clientRef.current.addStateListener(setConnectionState);

    // Set up event listeners for cache invalidation
    const unsubscribeServerStatus = clientRef.current.addEventListener(
      'server_status_changed',
      event => {
        handleServerStatusUpdate(event);
        setLastEvent(event);
      }
    );

    const unsubscribeGatewayStatus = clientRef.current.addEventListener(
      'gateway_status_changed',
      event => {
        handleGatewayStatusUpdate(event);
        setLastEvent(event);
      }
    );

    const unsubscribeServerMetrics = clientRef.current.addEventListener(
      'server_metrics',
      event => {
        handleServerMetricsUpdate(event);
        setLastEvent(event);
      }
    );

    const unsubscribeServerLog = clientRef.current.addEventListener(
      'server_log',
      event => {
        handleServerLogEvent(event);
        setLastEvent(event);
      }
    );

    const unsubscribeSystemError = clientRef.current.addEventListener(
      'system_error',
      event => {
        handleSystemErrorEvent(event);
        setLastEvent(event);
      }
    );

    // Clean up on unmount
    return () => {
      unsubscribeState();
      unsubscribeServerStatus();
      unsubscribeGatewayStatus();
      unsubscribeServerMetrics();
      unsubscribeServerLog();
      unsubscribeSystemError();

      if (clientRef.current) {
        clientRef.current.disconnect();
        clientRef.current = null;
      }
    };
  }, [
    config,
    handleServerStatusUpdate,
    handleGatewayStatusUpdate,
    handleServerMetricsUpdate,
    handleServerLogEvent,
    handleSystemErrorEvent,
  ]);

  // Connection actions
  const connect = useCallback(() => {
    if (clientRef.current) {
      clientRef.current.connect();
    }
  }, []);

  const disconnect = useCallback(() => {
    if (clientRef.current) {
      clientRef.current.disconnect();
    }
  }, []);

  const subscribe = useCallback((channels: string[], filter?: EventFilter) => {
    if (clientRef.current) {
      clientRef.current.subscribe(channels, filter);
    }
  }, []);

  const unsubscribe = useCallback((channels?: string[]) => {
    if (clientRef.current) {
      clientRef.current.unsubscribe(channels);
    }
  }, []);

  const sendMessage = useCallback((message: WebSocketMessage) => {
    if (clientRef.current) {
      clientRef.current.sendMessage(message);
    }
  }, []);

  // Event handler management
  const addEventListener = useCallback(
    (
      type: RealtimeEvent['type'],
      handler: (event: RealtimeEvent) => void
    ): (() => void) => {
      // Add to local handlers map
      if (!eventHandlersRef.current.has(type)) {
        eventHandlersRef.current.set(type, new Set());
      }
      const handlers = eventHandlersRef.current.get(type);
      if (handlers) {
        handlers.add(handler);
      }

      // Add to client
      const unsubscribeFromClient = clientRef.current?.addEventListener(
        type,
        handler
      );

      // Return combined unsubscribe function
      return () => {
        const handlers = eventHandlersRef.current.get(type);
        if (handlers) {
          handlers.delete(handler);
          if (handlers.size === 0) {
            eventHandlersRef.current.delete(type);
          }
        }

        if (unsubscribeFromClient) {
          unsubscribeFromClient();
        }
      };
    },
    []
  );

  const isConnected = connectionState.status === 'connected';

  return {
    connectionState,
    isConnected,
    lastEvent,
    connect,
    disconnect,
    subscribe,
    unsubscribe,
    sendMessage,
    addEventListener,
  };
}

/**
 * Hook for subscribing to specific channels with automatic connection management
 */
export function useWebSocketSubscription(
  channels: ChannelType[],
  config?: Partial<RealtimeConfig>
) {
  const websocket = useWebSocket(config);

  useEffect(() => {
    if (websocket.isConnected) {
      websocket.subscribe(channels);
    }
  }, [websocket.isConnected, channels, websocket]);

  useEffect(() => {
    // Auto-connect when hook is used
    websocket.connect();

    // Cleanup on unmount
    return () => {
      websocket.disconnect();
    };
  }, [websocket]);

  return websocket;
}

/**
 * Hook for server-specific WebSocket events
 */
export function useServerWebSocket(serverName: string) {
  const channels: ChannelType[] = [`server:${serverName}`, 'servers'];
  const filter: EventFilter = {
    server_names: [serverName],
  };

  return useWebSocketSubscription(channels, { filter });
}

/**
 * Hook for gateway-specific WebSocket events
 */
export function useGatewayWebSocket() {
  const channels: ChannelType[] = ['gateway', 'system'];

  return useWebSocketSubscription(channels);
}

/**
 * Hook for log-specific WebSocket events
 */
export function useLogWebSocket(serverNames?: string[]) {
  const channels: ChannelType[] = ['logs'];
  const filter: EventFilter = {
    event_types: ['server_log', 'system_error'],
    ...(serverNames && { server_names: serverNames }),
  };

  return useWebSocketSubscription(channels, { filter });
}
