'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { RealtimeSSEClient } from '@/lib/realtime-client';
import { serverKeys } from '@/hooks/api/use-servers';
import type {
  UseSSEReturn,
  RealtimeConfig,
  RealtimeEvent,
  EventFilter,
  ConnectionState,
  ChannelType,
} from '@/types/realtime';
import type { Server } from '@/types/api';

/**
 * Hook for Server-Sent Events real-time communication
 */
export function useSSE(config: Partial<RealtimeConfig> = {}): UseSSEReturn {
  const queryClient = useQueryClient();
  const clientRef = useRef<RealtimeSSEClient | null>(null);
  const [connectionState, setConnectionState] = useState<ConnectionState>({
    status: 'disconnected',
    reconnectAttempts: 0,
  });
  const [lastEvent, setLastEvent] = useState<RealtimeEvent | undefined>();
  const [events, setEvents] = useState<RealtimeEvent[]>([]);
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

    const gatewayData = event.data;

    if (!gatewayData.running) {
      toast.warning('Gateway has stopped');
    } else if (gatewayData.running) {
      toast.success('Gateway is now running');
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

      // Warn on high resource usage (but not too frequently)
      if (cpu_usage && cpu_usage > 90) {
        const lastCpuWarning = localStorage.getItem(
          `cpu_warning_${server_name}`
        );
        const now = Date.now();
        if (!lastCpuWarning || now - parseInt(lastCpuWarning) > 60000) {
          // 1 minute cooldown
          toast.warning(
            `Server ${server_name} CPU usage is high: ${cpu_usage}%`
          );
          localStorage.setItem(`cpu_warning_${server_name}`, now.toString());
        }
      }

      if (memory_usage && memory_usage > 90) {
        const lastMemoryWarning = localStorage.getItem(
          `memory_warning_${server_name}`
        );
        const now = Date.now();
        if (!lastMemoryWarning || now - parseInt(lastMemoryWarning) > 60000) {
          // 1 minute cooldown
          toast.warning(
            `Server ${server_name} memory usage is high: ${memory_usage}%`
          );
          localStorage.setItem(`memory_warning_${server_name}`, now.toString());
        }
      }
    },
    [queryClient]
  );

  // Handle server log events
  const handleServerLogEvent = useCallback((event: RealtimeEvent) => {
    if (event.type !== 'server_log') return;

    const { server_name, level, message } = event.data;

    // Show toast for error level logs only (with rate limiting)
    if (level === 'error') {
      const errorKey = `error_${server_name}_${Date.now()}`;
      const lastErrorTime = localStorage.getItem(`last_error_${server_name}`);
      const now = Date.now();

      if (!lastErrorTime || now - parseInt(lastErrorTime) > 30000) {
        // 30 second cooldown
        toast.error(`${server_name}: ${message}`, {
          id: errorKey,
          duration: 5000,
        });
        localStorage.setItem(`last_error_${server_name}`, now.toString());
      }
    }
  }, []);

  // Handle system error events
  const handleSystemErrorEvent = useCallback((event: RealtimeEvent) => {
    if (event.type !== 'system_error') return;

    const { error_type, message } = event.data;
    toast.error(`System Error (${error_type}): ${message}`, {
      duration: 7000, // Show longer for system errors
    });
  }, []);

  // Handle system health events
  const handleSystemHealthEvent = useCallback((event: RealtimeEvent) => {
    if (event.type !== 'system_health') return;

    const { cpu_usage, memory_usage, disk_usage, docker_status } = event.data;

    // Warn on critical system resource usage
    if (cpu_usage > 95) {
      toast.error(`Critical: System CPU usage at ${cpu_usage}%`);
    } else if (cpu_usage > 85) {
      const lastWarning = localStorage.getItem('system_cpu_warning');
      const now = Date.now();
      if (!lastWarning || now - parseInt(lastWarning) > 120000) {
        // 2 minute cooldown
        toast.warning(`High system CPU usage: ${cpu_usage}%`);
        localStorage.setItem('system_cpu_warning', now.toString());
      }
    }

    if (memory_usage > 95) {
      toast.error(`Critical: System memory usage at ${memory_usage}%`);
    }

    if (disk_usage > 95) {
      toast.error(`Critical: System disk usage at ${disk_usage}%`);
    }

    if (docker_status === 'error' || docker_status === 'stopped') {
      toast.error('Docker service is not running');
    }
  }, []);

  // Handle gateway connection events
  const handleGatewayConnectionEvent = useCallback((event: RealtimeEvent) => {
    if (event.type !== 'gateway_connection') return;

    const { action, client_count } = event.data;

    // Only show connection info in debug mode
    if (process.env.NODE_ENV === 'development') {
      if (action === 'connected') {
        toast.info(`Client connected (${client_count} total)`);
      } else if (action === 'disconnected') {
        toast.info(`Client disconnected (${client_count} total)`);
      }
    }
  }, []);

  // Initialize client
  useEffect(() => {
    clientRef.current = new RealtimeSSEClient(config);

    // Set up state listener
    const unsubscribeState =
      clientRef.current.addStateListener(setConnectionState);

    // Set up event listeners for cache invalidation and state updates
    const unsubscribeServerStatus = clientRef.current.addEventListener(
      'server_status_changed',
      event => {
        handleServerStatusUpdate(event);
        setLastEvent(event);
        setEvents(prev => [...prev, event].slice(-1000)); // Keep last 1000 events
      }
    );

    const unsubscribeGatewayStatus = clientRef.current.addEventListener(
      'gateway_status_changed',
      event => {
        handleGatewayStatusUpdate(event);
        setLastEvent(event);
        setEvents(prev => [...prev, event].slice(-1000));
      }
    );

    const unsubscribeServerMetrics = clientRef.current.addEventListener(
      'server_metrics',
      event => {
        handleServerMetricsUpdate(event);
        setLastEvent(event);
        setEvents(prev => [...prev, event].slice(-1000));
      }
    );

    const unsubscribeServerLog = clientRef.current.addEventListener(
      'server_log',
      event => {
        handleServerLogEvent(event);
        setLastEvent(event);
        setEvents(prev => [...prev, event].slice(-1000));
      }
    );

    const unsubscribeSystemError = clientRef.current.addEventListener(
      'system_error',
      event => {
        handleSystemErrorEvent(event);
        setLastEvent(event);
        setEvents(prev => [...prev, event].slice(-1000));
      }
    );

    const unsubscribeSystemHealth = clientRef.current.addEventListener(
      'system_health',
      event => {
        handleSystemHealthEvent(event);
        setLastEvent(event);
        setEvents(prev => [...prev, event].slice(-1000));
      }
    );

    const unsubscribeGatewayConnection = clientRef.current.addEventListener(
      'gateway_connection',
      event => {
        handleGatewayConnectionEvent(event);
        setLastEvent(event);
        setEvents(prev => [...prev, event].slice(-1000));
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
      unsubscribeSystemHealth();
      unsubscribeGatewayConnection();

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
    handleSystemHealthEvent,
    handleGatewayConnectionEvent,
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

  const clearEvents = useCallback(() => {
    setEvents([]);
    if (clientRef.current) {
      clientRef.current.clearEvents();
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
    events,
    connect,
    disconnect,
    clearEvents,
    addEventListener,
  };
}

/**
 * Hook for subscribing to specific channels with automatic connection management
 */
export function useSSESubscription(
  channels: ChannelType[],
  config?: Partial<RealtimeConfig>
) {
  const sse = useSSE({
    ...config,
    subscribeTo: channels,
  });

  useEffect(() => {
    // Auto-connect when hook is used
    sse.connect();

    // Cleanup on unmount
    return () => {
      sse.disconnect();
    };
  }, [sse]);

  return sse;
}

/**
 * Hook for server-specific SSE events
 */
export function useServerSSE(serverName: string) {
  const channels: ChannelType[] = [`server:${serverName}`, 'servers'];
  const filter: EventFilter = {
    server_names: [serverName],
  };

  return useSSESubscription(channels, { filter });
}

/**
 * Hook for gateway-specific SSE events
 */
export function useGatewaySSE() {
  const channels: ChannelType[] = ['gateway', 'system'];

  return useSSESubscription(channels);
}

/**
 * Hook for log-specific SSE events with filtering
 */
export function useLogSSE(
  serverNames?: string[],
  minLevel: 'debug' | 'info' | 'warn' | 'error' = 'info'
) {
  const channels: ChannelType[] = ['logs'];
  const filter: EventFilter = {
    event_types: ['server_log', 'system_error'],
    min_level: minLevel,
    ...(serverNames && { server_names: serverNames }),
  };

  return useSSESubscription(channels, { filter });
}

/**
 * Hook for metrics-specific SSE events
 */
export function useMetricsSSE(serverNames?: string[]) {
  const channels: ChannelType[] = ['metrics'];
  const filter: EventFilter = {
    event_types: ['server_metrics', 'system_health'],
    ...(serverNames && { server_names: serverNames }),
  };

  return useSSESubscription(channels, { filter });
}

/**
 * Hook that provides both recent events and real-time updates
 */
export function useEventHistory(maxEvents = 500) {
  const sse = useSSE();
  const [filteredEvents, setFilteredEvents] = useState<RealtimeEvent[]>([]);

  // Filter and limit events
  useEffect(() => {
    const recent = sse.events.slice(-maxEvents);
    setFilteredEvents(recent);
  }, [sse.events, maxEvents]);

  return {
    ...sse,
    events: filteredEvents,
    allEvents: sse.events,
  };
}
