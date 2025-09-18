/**
 * Environment Variable Configuration with T3 Env and Zod
 *
 * This file validates all environment variables at build time and runtime.
 * It provides type-safe access to environment variables throughout the application.
 *
 * @see https://env.t3.gg/docs/introduction
 */

import { createEnv } from '@t3-oss/env-nextjs';
import { z } from 'zod';

export const env = createEnv({
  /**
   * Server-side environment variables (NOT exposed to client)
   * These are validated at build time and runtime on the server.
   */
  server: {
    // Azure AD Configuration (Server-side only)
    AZURE_TENANT_ID: z.string().optional(),
    AZURE_CLIENT_ID: z.string().optional(),
    AZURE_CLIENT_SECRET: z.string().optional(),
    //AZURE_TENANT_ID: z.string().min(1, 'Azure Tenant ID is required'),
    //AZURE_CLIENT_ID: z.string().min(1, 'Azure Client ID is required'),
    //AZURE_CLIENT_SECRET: z.string().min(1, 'Azure Client Secret is required'),

    // JWT and Session Configuration
    JWT_SECRET: z
      .string()
      .min(32, 'JWT Secret must be at least 32 characters')
      .default('development-secret-change-in-production-please'),

    // Session Cookie Settings
    SESSION_COOKIE_NAME: z.string().default('mcp-portal-session'),
    SESSION_COOKIE_SECURE: z
      .enum(['true', 'false'])
      .transform(val => val === 'true')
      .default('false'),
    SESSION_COOKIE_HTTPONLY: z
      .enum(['true', 'false'])
      .transform(val => val === 'true')
      .default('true'),
    SESSION_COOKIE_SAMESITE: z.enum(['strict', 'lax', 'none']).default('lax'),

    // Node Environment
    NODE_ENV: z
      .enum(['development', 'production', 'test'])
      .default('development'),

    // Database URL (if needed in future)
    DATABASE_URL: z.url().optional(),

    // Redis URL for sessions (if needed)
    REDIS_URL: z.url().optional(),
  },

  /**
   * Client-side environment variables (exposed to browser)
   * These MUST be prefixed with NEXT_PUBLIC_
   */
  client: {
    // API Configuration
    NEXT_PUBLIC_API_URL: z.url().default('http://localhost:8080'),
    NEXT_PUBLIC_WS_URL: z.url().default('ws://localhost:8080'),

    // Azure AD Public Configuration
    NEXT_PUBLIC_AZURE_REDIRECT_URI: z
      .url()
      .default('http://localhost:3000/auth/callback'),
    NEXT_PUBLIC_AZURE_POST_LOGOUT_URI: z.url().default('http://localhost:3000'),
    NEXT_PUBLIC_AZURE_AUTHORITY: z.url().optional(), // Optional because we construct it from tenant ID
    NEXT_PUBLIC_AZURE_SCOPES: z.string().default('openid profile User.Read'),

    // Development Configuration
    NEXT_PUBLIC_DEBUG: z
      .enum(['true', 'false'])
      .transform(val => val === 'true')
      .default('false'),
    NEXT_PUBLIC_API_TIMEOUT: z
      .string()
      .transform(val => parseInt(val, 10))
      .default('30000'),
    NEXT_PUBLIC_WS_RECONNECT_INTERVAL: z
      .string()
      .transform(val => parseInt(val, 10))
      .default('5000'),

    // Feature Flags
    NEXT_PUBLIC_ENABLE_WEBSOCKET: z
      .enum(['true', 'false'])
      .transform(val => val === 'true')
      .default('true'),
    NEXT_PUBLIC_ENABLE_SSE: z
      .enum(['true', 'false'])
      .transform(val => val === 'true')
      .default('true'),
    NEXT_PUBLIC_ENABLE_ADMIN: z
      .enum(['true', 'false'])
      .transform(val => val === 'true')
      .default('true'),
    NEXT_PUBLIC_ENABLE_BULK_OPS: z
      .enum(['true', 'false'])
      .transform(val => val === 'true')
      .default('true'),

    // Security Configuration
    NEXT_PUBLIC_TOKEN_STORAGE: z
      .enum(['localStorage', 'sessionStorage', 'memory'])
      .default('localStorage'),
    NEXT_PUBLIC_SESSION_TIMEOUT: z
      .string()
      .transform(val => parseInt(val, 10))
      .default('60'),
    NEXT_PUBLIC_ENABLE_CSRF: z
      .enum(['true', 'false'])
      .transform(val => val === 'true')
      .default('true'),

    // UI Configuration
    NEXT_PUBLIC_DEFAULT_THEME: z
      .enum(['light', 'dark', 'system'])
      .default('system'),
    NEXT_PUBLIC_DEFAULT_PAGE_SIZE: z
      .string()
      .transform(val => parseInt(val, 10))
      .default('20'),
    NEXT_PUBLIC_STATUS_REFRESH_INTERVAL: z
      .string()
      .transform(val => parseInt(val, 10))
      .default('10'),

    // Error Reporting (Optional)
    NEXT_PUBLIC_SENTRY_DSN: z.url().optional(),
    NEXT_PUBLIC_ENABLE_ERROR_REPORTING: z
      .enum(['true', 'false'])
      .transform(val => val === 'true')
      .default('false'),
  },

  /**
   * Runtime environment variables
   * These are available in both client and server but not validated at build time
   */
  runtimeEnv: {
    // Server
    AZURE_TENANT_ID: process.env.AZURE_TENANT_ID,
    AZURE_CLIENT_ID: process.env.AZURE_CLIENT_ID,
    AZURE_CLIENT_SECRET: process.env.AZURE_CLIENT_SECRET,
    JWT_SECRET: process.env.JWT_SECRET,
    SESSION_COOKIE_NAME: process.env.SESSION_COOKIE_NAME,
    SESSION_COOKIE_SECURE: process.env.SESSION_COOKIE_SECURE,
    SESSION_COOKIE_HTTPONLY: process.env.SESSION_COOKIE_HTTPONLY,
    SESSION_COOKIE_SAMESITE: process.env.SESSION_COOKIE_SAMESITE,
    NODE_ENV: process.env.NODE_ENV,
    DATABASE_URL: process.env.DATABASE_URL,
    REDIS_URL: process.env.REDIS_URL,

    // Client
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL,
    NEXT_PUBLIC_WS_URL: process.env.NEXT_PUBLIC_WS_URL,
    NEXT_PUBLIC_AZURE_REDIRECT_URI: process.env.NEXT_PUBLIC_AZURE_REDIRECT_URI,
    NEXT_PUBLIC_AZURE_POST_LOGOUT_URI:
      process.env.NEXT_PUBLIC_AZURE_POST_LOGOUT_URI,
    NEXT_PUBLIC_AZURE_AUTHORITY: process.env.NEXT_PUBLIC_AZURE_AUTHORITY,
    NEXT_PUBLIC_AZURE_SCOPES: process.env.NEXT_PUBLIC_AZURE_SCOPES,
    NEXT_PUBLIC_DEBUG: process.env.NEXT_PUBLIC_DEBUG,
    NEXT_PUBLIC_API_TIMEOUT: process.env.NEXT_PUBLIC_API_TIMEOUT,
    NEXT_PUBLIC_WS_RECONNECT_INTERVAL:
      process.env.NEXT_PUBLIC_WS_RECONNECT_INTERVAL,
    NEXT_PUBLIC_ENABLE_WEBSOCKET: process.env.NEXT_PUBLIC_ENABLE_WEBSOCKET,
    NEXT_PUBLIC_ENABLE_SSE: process.env.NEXT_PUBLIC_ENABLE_SSE,
    NEXT_PUBLIC_ENABLE_ADMIN: process.env.NEXT_PUBLIC_ENABLE_ADMIN,
    NEXT_PUBLIC_ENABLE_BULK_OPS: process.env.NEXT_PUBLIC_ENABLE_BULK_OPS,
    NEXT_PUBLIC_TOKEN_STORAGE: process.env.NEXT_PUBLIC_TOKEN_STORAGE,
    NEXT_PUBLIC_SESSION_TIMEOUT: process.env.NEXT_PUBLIC_SESSION_TIMEOUT,
    NEXT_PUBLIC_ENABLE_CSRF: process.env.NEXT_PUBLIC_ENABLE_CSRF,
    NEXT_PUBLIC_DEFAULT_THEME: process.env.NEXT_PUBLIC_DEFAULT_THEME,
    NEXT_PUBLIC_DEFAULT_PAGE_SIZE: process.env.NEXT_PUBLIC_DEFAULT_PAGE_SIZE,
    NEXT_PUBLIC_STATUS_REFRESH_INTERVAL:
      process.env.NEXT_PUBLIC_STATUS_REFRESH_INTERVAL,
    NEXT_PUBLIC_SENTRY_DSN: process.env.NEXT_PUBLIC_SENTRY_DSN,
    NEXT_PUBLIC_ENABLE_ERROR_REPORTING:
      process.env.NEXT_PUBLIC_ENABLE_ERROR_REPORTING,
  },

  /**
   * Skip validation in certain environments
   * Useful for Docker builds where env vars aren't available at build time
   */
  skipValidation: !!process.env.SKIP_ENV_VALIDATION,

  /**
   * Make it clear that empty strings are not valid
   */
  emptyStringAsUndefined: true,
});

/**
 * Type-safe environment variable access
 * Import this throughout your app instead of using process.env
 *
 * Note: Type annotations are handled in env.d.ts for .mjs compatibility
 */
