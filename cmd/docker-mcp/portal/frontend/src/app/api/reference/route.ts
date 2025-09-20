/**
 * Scalar API Documentation Route
 * Provides interactive API documentation for the MCP Portal
 */

import { ApiReference } from '@scalar/nextjs-api-reference';

// OpenAPI specification for MCP Portal API
const openApiSpec = {
  openapi: '3.1.0',
  info: {
    title: 'MCP Portal API',
    version: '1.0.0',
    description:
      'Model Context Protocol Portal - Management API for MCP servers',
    contact: {
      name: 'MCP Portal Team',
      email: 'support@matherly.net',
      url: process.env.NEXT_PUBLIC_SITE_URL || 'http://localhost:3000',
    },
    license: {
      name: 'MIT',
      url: 'https://opensource.org/licenses/MIT',
    },
  },
  servers: [
    {
      url: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080',
      description: 'MCP Portal Backend API',
    },
  ],
  paths: {
    '/api/v1/servers': {
      get: {
        summary: 'List all MCP servers',
        tags: ['Servers'],
        security: [{ bearerAuth: [] }],
        responses: {
          '200': {
            description: 'List of MCP servers',
            content: {
              'application/json': {
                schema: {
                  type: 'array',
                  items: {
                    $ref: '#/components/schemas/Server',
                  },
                },
              },
            },
          },
          '401': { $ref: '#/components/responses/Unauthorized' },
        },
      },
    },
    '/api/v1/servers/{id}': {
      get: {
        summary: 'Get server details',
        tags: ['Servers'],
        security: [{ bearerAuth: [] }],
        parameters: [
          {
            name: 'id',
            in: 'path',
            required: true,
            schema: { type: 'string' },
          },
        ],
        responses: {
          '200': {
            description: 'Server details',
            content: {
              'application/json': {
                schema: { $ref: '#/components/schemas/Server' },
              },
            },
          },
          '404': { $ref: '#/components/responses/NotFound' },
        },
      },
    },
    '/api/v1/servers/{id}/enable': {
      post: {
        summary: 'Enable a server',
        tags: ['Servers'],
        security: [{ bearerAuth: [] }],
        parameters: [
          {
            name: 'id',
            in: 'path',
            required: true,
            schema: { type: 'string' },
          },
        ],
        responses: {
          '200': {
            description: 'Server enabled successfully',
          },
          '404': { $ref: '#/components/responses/NotFound' },
        },
      },
    },
    '/api/v1/servers/{id}/disable': {
      post: {
        summary: 'Disable a server',
        tags: ['Servers'],
        security: [{ bearerAuth: [] }],
        parameters: [
          {
            name: 'id',
            in: 'path',
            required: true,
            schema: { type: 'string' },
          },
        ],
        responses: {
          '200': {
            description: 'Server disabled successfully',
          },
          '404': { $ref: '#/components/responses/NotFound' },
        },
      },
    },
    '/api/v1/catalog': {
      get: {
        summary: 'Get server catalog',
        tags: ['Catalog'],
        security: [{ bearerAuth: [] }],
        responses: {
          '200': {
            description: 'Server catalog',
            content: {
              'application/json': {
                schema: {
                  type: 'array',
                  items: {
                    $ref: '#/components/schemas/CatalogItem',
                  },
                },
              },
            },
          },
        },
      },
    },
    '/api/v1/config': {
      get: {
        summary: 'Get user configuration',
        tags: ['Configuration'],
        security: [{ bearerAuth: [] }],
        responses: {
          '200': {
            description: 'User configuration',
            content: {
              'application/json': {
                schema: { $ref: '#/components/schemas/UserConfig' },
              },
            },
          },
        },
      },
      put: {
        summary: 'Update user configuration',
        tags: ['Configuration'],
        security: [{ bearerAuth: [] }],
        requestBody: {
          required: true,
          content: {
            'application/json': {
              schema: { $ref: '#/components/schemas/UserConfig' },
            },
          },
        },
        responses: {
          '200': {
            description: 'Configuration updated',
          },
        },
      },
    },
  },
  components: {
    schemas: {
      Server: {
        type: 'object',
        properties: {
          id: { type: 'string' },
          name: { type: 'string' },
          description: { type: 'string' },
          status: {
            type: 'string',
            enum: ['enabled', 'disabled', 'error'],
          },
          config: { type: 'object' },
          createdAt: { type: 'string', format: 'date-time' },
          updatedAt: { type: 'string', format: 'date-time' },
        },
      },
      CatalogItem: {
        type: 'object',
        properties: {
          id: { type: 'string' },
          name: { type: 'string' },
          description: { type: 'string' },
          category: { type: 'string' },
          version: { type: 'string' },
          author: { type: 'string' },
          repository: { type: 'string' },
          tags: {
            type: 'array',
            items: { type: 'string' },
          },
        },
      },
      UserConfig: {
        type: 'object',
        properties: {
          theme: {
            type: 'string',
            enum: ['light', 'dark', 'system'],
          },
          notifications: { type: 'boolean' },
          autoUpdate: { type: 'boolean' },
          preferences: { type: 'object' },
        },
      },
    },
    securitySchemes: {
      bearerAuth: {
        type: 'http',
        scheme: 'bearer',
        bearerFormat: 'JWT',
      },
    },
    responses: {
      Unauthorized: {
        description: 'Unauthorized - Invalid or missing token',
        content: {
          'application/json': {
            schema: {
              type: 'object',
              properties: {
                error: { type: 'string' },
                message: { type: 'string' },
              },
            },
          },
        },
      },
      NotFound: {
        description: 'Resource not found',
        content: {
          'application/json': {
            schema: {
              type: 'object',
              properties: {
                error: { type: 'string' },
                message: { type: 'string' },
              },
            },
          },
        },
      },
    },
  },
};

const config = {
  spec: {
    content: openApiSpec,
  },
  theme: 'purple' as const,
  layout: 'modern' as const,
  darkMode: true,
  hideModels: false,
  searchHotKey: 'k' as const,
  customCss: `
    .scalar-app {
      font-family: var(--font-geist-sans);
    }
    .dark .scalar-app {
      background: #0a0a0a;
    }
  `,
  metaData: {
    title: 'MCP Portal API Documentation',
    description: 'Interactive API documentation for MCP Portal',
  },
};

export const GET = ApiReference(config);
