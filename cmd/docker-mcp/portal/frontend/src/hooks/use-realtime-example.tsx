// Example usage of WebSocket and SSE hooks
// This file shows how to use the real-time hooks in components

import React, { useCallback, useEffect, useState } from 'react';
import {
  useWebSocket,
  useSSE,
  useServerWebSocket,
  useLogSSE,
} from '@/hooks/api';
import type { RealtimeEvent } from '@/types/realtime';

/**
 * Example component showing WebSocket usage
 */
export function WebSocketExample() {
  const websocket = useWebSocket({
    autoReconnect: true,
    subscribeTo: ['servers', 'gateway'],
  });

  const [messages, setMessages] = useState<RealtimeEvent[]>([]);

  const handleServerStatusChange = useCallback((event: RealtimeEvent) => {
    // eslint-disable-next-line no-console
    console.log('Server status changed:', event);
    setMessages(prev => [...prev, event].slice(-10)); // Keep last 10 messages
  }, []);

  useEffect(() => {
    // Connect when component mounts
    websocket.connect();

    // Subscribe to server status changes
    const unsubscribe = websocket.addEventListener(
      'server_status_changed',
      handleServerStatusChange
    );

    return () => {
      unsubscribe();
      websocket.disconnect();
    };
  }, [websocket, handleServerStatusChange]);

  return (
    <div className="p-4">
      <h3 className="text-lg font-semibold mb-4">WebSocket Connection</h3>

      <div className="mb-4">
        <span
          className={`inline-block px-2 py-1 rounded text-sm ${
            websocket.isConnected
              ? 'bg-green-100 text-green-800'
              : 'bg-red-100 text-red-800'
          }`}
        >
          {websocket.connectionState.status}
        </span>

        {websocket.connectionState.reconnectAttempts > 0 && (
          <span className="ml-2 text-sm text-gray-600">
            Reconnect attempts: {websocket.connectionState.reconnectAttempts}
          </span>
        )}
      </div>

      <div className="space-y-2">
        <h4 className="font-medium">Recent Events:</h4>
        {messages.length === 0 ? (
          <p className="text-gray-500">No events received</p>
        ) : (
          messages.map((event, index) => (
            <div key={index} className="p-2 bg-gray-50 rounded text-sm">
              <div className="font-mono text-xs text-gray-500">
                {event.timestamp}
              </div>
              <div className="font-medium">{event.type}</div>
              <div className="text-gray-700">{JSON.stringify(event.data)}</div>
            </div>
          ))
        )}
      </div>

      <div className="mt-4 space-x-2">
        <button
          onClick={() => websocket.connect()}
          disabled={websocket.isConnected}
          className="px-3 py-1 bg-blue-500 text-white rounded disabled:opacity-50"
        >
          Connect
        </button>
        <button
          onClick={() => websocket.disconnect()}
          disabled={!websocket.isConnected}
          className="px-3 py-1 bg-red-500 text-white rounded disabled:opacity-50"
        >
          Disconnect
        </button>
      </div>
    </div>
  );
}

/**
 * Example component showing SSE usage
 */
export function SSEExample() {
  const sse = useSSE({
    autoReconnect: true,
    subscribeTo: ['servers', 'logs'],
  });

  useEffect(() => {
    // Connect when component mounts
    sse.connect();

    return () => {
      sse.disconnect();
    };
  }, [sse]);

  return (
    <div className="p-4">
      <h3 className="text-lg font-semibold mb-4">Server-Sent Events</h3>

      <div className="mb-4">
        <span
          className={`inline-block px-2 py-1 rounded text-sm ${
            sse.isConnected
              ? 'bg-green-100 text-green-800'
              : 'bg-red-100 text-red-800'
          }`}
        >
          {sse.connectionState.status}
        </span>

        <span className="ml-4 text-sm text-gray-600">
          Events received: {sse.events.length}
        </span>
      </div>

      <div className="space-y-2 max-h-64 overflow-y-auto">
        <h4 className="font-medium">Event Stream:</h4>
        {sse.events.length === 0 ? (
          <p className="text-gray-500">No events received</p>
        ) : (
          sse.events.slice(-20).map((event, index) => (
            <div key={index} className="p-2 bg-gray-50 rounded text-sm">
              <div className="flex justify-between items-start">
                <div className="font-mono text-xs text-gray-500">
                  {event.timestamp}
                </div>
                <div className="text-xs text-gray-400">{event.source}</div>
              </div>
              <div className="font-medium">{event.type}</div>
              <div className="text-gray-700 text-xs">
                {JSON.stringify(event.data)}
              </div>
            </div>
          ))
        )}
      </div>

      <div className="mt-4 space-x-2">
        <button
          onClick={() => sse.connect()}
          disabled={sse.isConnected}
          className="px-3 py-1 bg-blue-500 text-white rounded disabled:opacity-50"
        >
          Connect
        </button>
        <button
          onClick={() => sse.disconnect()}
          disabled={!sse.isConnected}
          className="px-3 py-1 bg-red-500 text-white rounded disabled:opacity-50"
        >
          Disconnect
        </button>
        <button
          onClick={() => sse.clearEvents()}
          className="px-3 py-1 bg-gray-500 text-white rounded"
        >
          Clear Events
        </button>
      </div>
    </div>
  );
}

/**
 * Example showing server-specific real-time updates
 */
export function ServerRealtimeExample({ serverName }: { serverName: string }) {
  const serverWS = useServerWebSocket(serverName);
  const [serverEvents, setServerEvents] = useState<RealtimeEvent[]>([]);

  const handleServerEvent = useCallback(
    (event: RealtimeEvent) => {
      if (
        event.type === 'server_status_changed' &&
        event.data.server_name === serverName
      ) {
        setServerEvents(prev => [...prev, event].slice(-5)); // Keep last 5 events
      }
    },
    [serverName]
  );

  useEffect(() => {
    const unsubscribe = serverWS.addEventListener(
      'server_status_changed',
      handleServerEvent
    );

    return unsubscribe;
  }, [serverWS, handleServerEvent]);

  return (
    <div className="p-4 border rounded-lg">
      <h4 className="font-semibold mb-2">Server: {serverName}</h4>

      <div className="mb-2">
        <span
          className={`inline-block px-2 py-1 rounded text-xs ${
            serverWS.isConnected
              ? 'bg-green-100 text-green-800'
              : 'bg-red-100 text-red-800'
          }`}
        >
          {serverWS.connectionState.status}
        </span>
      </div>

      <div className="space-y-1">
        <h5 className="text-sm font-medium">Recent Events:</h5>
        {serverEvents.length === 0 ? (
          <p className="text-xs text-gray-500">No events for this server</p>
        ) : (
          serverEvents.map((event, index) => (
            <div key={index} className="p-1 bg-blue-50 rounded text-xs">
              <span className="font-mono text-gray-500">{event.timestamp}</span>
              <span className="ml-2">{event.type}</span>
              {event.type === 'server_status_changed' && event.data.status && (
                <span className="ml-2 font-medium">{event.data.status}</span>
              )}
            </div>
          ))
        )}
      </div>
    </div>
  );
}

/**
 * Example showing log monitoring
 */
export function LogMonitorExample() {
  const logSSE = useLogSSE(undefined, 'warn'); // Only warnings and errors
  const [recentLogs, setRecentLogs] = useState<RealtimeEvent[]>([]);

  const handleLogEvent = useCallback((event: RealtimeEvent) => {
    setRecentLogs(prev => [...prev, event].slice(-10)); // Keep last 10 logs
  }, []);

  useEffect(() => {
    const unsubscribe = logSSE.addEventListener('server_log', handleLogEvent);

    return unsubscribe;
  }, [logSSE, handleLogEvent]);

  return (
    <div className="p-4">
      <h3 className="text-lg font-semibold mb-4">Log Monitor (Warn/Error)</h3>

      <div className="mb-4">
        <span
          className={`inline-block px-2 py-1 rounded text-sm ${
            logSSE.isConnected
              ? 'bg-green-100 text-green-800'
              : 'bg-red-100 text-red-800'
          }`}
        >
          {logSSE.connectionState.status}
        </span>
      </div>

      <div className="space-y-2 max-h-48 overflow-y-auto">
        {recentLogs.length === 0 ? (
          <p className="text-gray-500">No recent warnings or errors</p>
        ) : (
          recentLogs.map((event, index) => (
            <div
              key={index}
              className={`p-2 rounded text-sm ${
                event.type === 'server_log' && event.data.level === 'error'
                  ? 'bg-red-50 border-l-4 border-red-400'
                  : 'bg-yellow-50 border-l-4 border-yellow-400'
              }`}
            >
              <div className="flex justify-between items-start">
                <div className="font-mono text-xs text-gray-500">
                  {event.timestamp}
                </div>
                <div
                  className={`text-xs font-medium ${
                    event.type === 'server_log' && event.data.level === 'error'
                      ? 'text-red-600'
                      : 'text-yellow-600'
                  }`}
                >
                  {event.type === 'server_log' &&
                    event.data.level?.toUpperCase()}
                </div>
              </div>
              <div className="font-medium">
                {event.type === 'server_log' && event.data.server_name}
              </div>
              <div className="text-gray-700">
                {event.type === 'server_log' && event.data.message}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

/**
 * Complete example combining all real-time features
 */
export function RealtimeDashboard() {
  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Real-time Dashboard</h2>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <WebSocketExample />
        <SSEExample />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <ServerRealtimeExample serverName="sequential-thinking" />
        <ServerRealtimeExample serverName="brave-search" />
        <ServerRealtimeExample serverName="filesystem" />
      </div>

      <LogMonitorExample />
    </div>
  );
}
