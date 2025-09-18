/**
 * Sentry Server Configuration
 * This file configures Sentry for server-side error tracking
 */

import * as Sentry from '@sentry/nextjs';

const SENTRY_DSN = process.env.SENTRY_DSN || process.env.NEXT_PUBLIC_SENTRY_DSN;
const ENV = process.env.APP_ENV || process.env.NODE_ENV || 'development';

Sentry.init({
  dsn: SENTRY_DSN,
  environment: ENV,

  // Performance Monitoring
  tracesSampleRate: ENV === 'production' ? 0.1 : 1.0,

  // Profiling (requires tracing)
  profilesSampleRate: ENV === 'production' ? 0.1 : 1.0,

  // Release Tracking
  release: process.env.APP_VERSION || process.env.NEXT_PUBLIC_APP_VERSION,

  // Server-specific integrations
  integrations: [
    // Captures console.error() calls
    Sentry.captureConsoleIntegration({
      levels: ['error', 'warn'],
    }),

    // HTTP request tracking
    Sentry.httpIntegration({
      breadcrumbs: true,
    }),
  ],

  // Privacy & Security
  sendDefaultPii: false,

  // Server-side specific options
  serverName: process.env.SERVER_NAME || 'mcp-portal',

  // Filtering
  beforeSend(event) {
    // Don't send events without a DSN
    if (!SENTRY_DSN) {
      return null;
    }

    // Filter development errors
    if (ENV === 'development') {
      if (event.level === 'log' || event.level === 'debug') {
        return null;
      }
    }

    // Sanitize sensitive data
    if (event.request) {
      // Remove auth headers
      if (event.request.headers) {
        delete event.request.headers['authorization'];
        delete event.request.headers['cookie'];
        delete event.request.headers['x-api-key'];
      }

      // Remove sensitive query params
      if (
        event.request.query_string &&
        typeof event.request.query_string === 'string'
      ) {
        event.request.query_string = event.request.query_string.replace(
          /token=[^&]*/gi,
          'token=***'
        );
      }
    }

    return event;
  },

  // Error filtering
  ignoreErrors: [
    // Ignore client-side errors on server
    'window is not defined',
    'document is not defined',

    // Ignore expected errors
    'ECONNREFUSED',
    'ETIMEDOUT',
    'ENOTFOUND',
  ],

  // Transaction filtering
  ignoreTransactions: [
    '/api/health',
    '/api/metrics',
    '/_next/static',
    '/favicon.ico',
  ],
});
