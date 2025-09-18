/**
 * Sentry Edge Runtime Configuration
 * This file configures Sentry for Edge runtime (middleware, edge routes)
 */

import * as Sentry from '@sentry/nextjs';

const SENTRY_DSN = process.env.SENTRY_DSN || process.env.NEXT_PUBLIC_SENTRY_DSN;
const ENV = process.env.APP_ENV || process.env.NODE_ENV || 'development';

Sentry.init({
  dsn: SENTRY_DSN,
  environment: ENV,

  // Performance Monitoring
  tracesSampleRate: ENV === 'production' ? 0.1 : 1.0,

  // Release Tracking
  release: process.env.APP_VERSION || process.env.NEXT_PUBLIC_APP_VERSION,

  // Privacy & Security
  sendDefaultPii: false,

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

    return event;
  },

  // Error filtering for edge runtime
  ignoreErrors: ['ECONNREFUSED', 'ETIMEDOUT', 'ENOTFOUND', 'fetch failed'],

  // Transaction filtering
  ignoreTransactions: ['/api/health', '/api/metrics', '/_next/static'],
});
