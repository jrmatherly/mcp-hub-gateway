// Environment variable type definitions for env.mjs compatibility

import type { env } from '@/env.mjs';

// Type-safe environment variable access
export type Env = typeof env;

// Augment process.env with our validated environment variables
declare global {
  namespace NodeJS {
    interface ProcessEnv {
      // Server-side variables
      AZURE_TENANT_ID?: string;
      AZURE_CLIENT_ID?: string;
      AZURE_CLIENT_SECRET?: string;
      JWT_SECRET?: string;
      SESSION_COOKIE_NAME?: string;
      SESSION_COOKIE_SECURE?: string;
      SESSION_COOKIE_HTTPONLY?: string;
      SESSION_COOKIE_SAMESITE?: string;
      NODE_ENV?: 'development' | 'production' | 'test';
      DATABASE_URL?: string;
      REDIS_URL?: string;

      // Client-side variables (prefixed with NEXT_PUBLIC_)
      NEXT_PUBLIC_API_URL?: string;
      NEXT_PUBLIC_WS_URL?: string;
      NEXT_PUBLIC_AZURE_REDIRECT_URI?: string;
      NEXT_PUBLIC_AZURE_POST_LOGOUT_URI?: string;
      NEXT_PUBLIC_AZURE_AUTHORITY?: string;
      NEXT_PUBLIC_AZURE_SCOPES?: string;
      NEXT_PUBLIC_DEBUG?: string;
      NEXT_PUBLIC_API_TIMEOUT?: string;
      NEXT_PUBLIC_WS_RECONNECT_INTERVAL?: string;
      NEXT_PUBLIC_ENABLE_WEBSOCKET?: string;
      NEXT_PUBLIC_ENABLE_SSE?: string;
      NEXT_PUBLIC_ENABLE_ADMIN?: string;
      NEXT_PUBLIC_ENABLE_BULK_OPS?: string;
      NEXT_PUBLIC_TOKEN_STORAGE?: string;
      NEXT_PUBLIC_SESSION_TIMEOUT?: string;
      NEXT_PUBLIC_ENABLE_CSRF?: string;
      NEXT_PUBLIC_DEFAULT_THEME?: string;
      NEXT_PUBLIC_DEFAULT_PAGE_SIZE?: string;
      NEXT_PUBLIC_STATUS_REFRESH_INTERVAL?: string;
      NEXT_PUBLIC_SENTRY_DSN?: string;
      NEXT_PUBLIC_ENABLE_ERROR_REPORTING?: string;

      // Skip validation flag
      SKIP_ENV_VALIDATION?: string;
    }
  }
}

export {};
