// MSW API handlers for testing
import { http, HttpResponse } from 'msw';

// Mock API base URL
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export const handlers = [
  // Auth endpoints
  http.get(`${API_BASE}/api/auth/user`, () => {
    return HttpResponse.json({
      id: 'test-user-id',
      email: 'test@example.com',
      name: 'Test User',
      picture: 'https://example.com/avatar.jpg',
      roles: ['user'],
    });
  }),

  http.post(`${API_BASE}/api/auth/login`, () => {
    return HttpResponse.json({
      success: true,
      token: 'mock-jwt-token',
      user: {
        id: 'test-user-id',
        email: 'test@example.com',
        name: 'Test User',
      },
    });
  }),

  http.post(`${API_BASE}/api/auth/logout`, () => {
    return HttpResponse.json({ success: true });
  }),

  // MCP Server management endpoints
  http.get(`${API_BASE}/api/servers`, () => {
    return HttpResponse.json({
      servers: [
        {
          id: 'server-1',
          name: 'Test Server 1',
          description: 'A test MCP server',
          status: 'running',
          enabled: true,
          version: '1.0.0',
          health: 'healthy',
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
        {
          id: 'server-2',
          name: 'Test Server 2',
          description: 'Another test MCP server',
          status: 'stopped',
          enabled: false,
          version: '1.1.0',
          health: 'unhealthy',
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
      ],
    });
  }),

  http.get(`${API_BASE}/api/servers/:id`, ({ params }) => {
    const { id } = params;
    return HttpResponse.json({
      id,
      name: `Test Server ${id}`,
      description: `A test MCP server with ID ${id}`,
      status: 'running',
      enabled: true,
      version: '1.0.0',
      health: 'healthy',
      config: {
        command: 'node',
        args: ['server.js'],
        env: {},
      },
      metrics: {
        requests: 100,
        errors: 2,
        uptime: 3600,
      },
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    });
  }),

  http.post(`${API_BASE}/api/servers/:id/enable`, ({ params }) => {
    const { id } = params;
    return HttpResponse.json({
      success: true,
      message: `Server ${id} enabled successfully`,
      server: {
        id,
        status: 'starting',
        enabled: true,
      },
    });
  }),

  http.post(`${API_BASE}/api/servers/:id/disable`, ({ params }) => {
    const { id } = params;
    return HttpResponse.json({
      success: true,
      message: `Server ${id} disabled successfully`,
      server: {
        id,
        status: 'stopping',
        enabled: false,
      },
    });
  }),

  http.post(`${API_BASE}/api/servers/:id/restart`, ({ params }) => {
    const { id } = params;
    return HttpResponse.json({
      success: true,
      message: `Server ${id} restarting`,
      server: {
        id,
        status: 'restarting',
      },
    });
  }),

  // Catalog endpoints
  http.get(`${API_BASE}/api/catalog`, () => {
    return HttpResponse.json({
      servers: [
        {
          name: 'filesystem',
          description: 'File system operations',
          version: '1.0.0',
          author: 'MCP Team',
          category: 'utility',
          tags: ['files', 'filesystem'],
          repository: 'https://github.com/modelcontextprotocol/servers',
          documentation: 'https://docs.example.com/filesystem',
        },
        {
          name: 'weather',
          description: 'Weather information service',
          version: '2.1.0',
          author: 'Weather Corp',
          category: 'api',
          tags: ['weather', 'api'],
          repository: 'https://github.com/weather-corp/mcp-weather',
          documentation: 'https://docs.weather.com/mcp',
        },
      ],
    });
  }),

  // Configuration endpoints
  http.get(`${API_BASE}/api/config`, () => {
    return HttpResponse.json({
      gateway: {
        port: 8080,
        host: 'localhost',
        transport: 'stdio',
      },
      servers: {
        filesystem: {
          command: 'node',
          args: ['filesystem-server.js'],
          env: {},
        },
      },
    });
  }),

  http.put(`${API_BASE}/api/config`, async ({ request }) => {
    const config = await request.json();
    return HttpResponse.json({
      success: true,
      message: 'Configuration updated successfully',
      config,
    });
  }),

  // Health check endpoint
  http.get(`${API_BASE}/api/health`, () => {
    return HttpResponse.json({
      status: 'healthy',
      timestamp: new Date().toISOString(),
      version: '1.0.0',
      uptime: 3600,
    });
  }),

  // WebSocket endpoint (for testing purposes)
  http.get(`${API_BASE}/api/ws`, () => {
    return new HttpResponse(null, {
      status: 101,
      statusText: 'Switching Protocols',
    });
  }),

  // Error handling - catch unhandled requests
  http.all('*', ({ request }) => {
    console.warn(`Unhandled request: ${request.method} ${request.url}`);
    return new HttpResponse(null, { status: 404 });
  }),
];

// Utility functions for dynamic handlers
export const createServerHandler = (
  serverId: string,
  overrides: Record<string, unknown> = {}
) => {
  return http.get(`${API_BASE}/api/servers/${serverId}`, () => {
    return HttpResponse.json({
      id: serverId,
      name: `Server ${serverId}`,
      description: `Dynamic test server ${serverId}`,
      status: 'running',
      enabled: true,
      version: '1.0.0',
      health: 'healthy',
      ...overrides,
    });
  });
};

export const createErrorHandler = (
  endpoint: string,
  status: number = 500,
  message: string = 'Internal Server Error'
) => {
  return http.all(`${API_BASE}${endpoint}`, () => {
    return HttpResponse.json(
      {
        error: true,
        message,
        timestamp: new Date().toISOString(),
      },
      { status }
    );
  });
};
