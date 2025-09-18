/**
 * Sentry Client Configuration
 * This file configures Sentry for client-side error tracking
 */

import * as Sentry from '@sentry/nextjs';

const SENTRY_DSN = process.env.NEXT_PUBLIC_SENTRY_DSN;
const ENV = process.env.NEXT_PUBLIC_APP_ENV || 'development';

Sentry.init({
  dsn: SENTRY_DSN,
  environment: ENV,

  // Performance Monitoring
  tracesSampleRate: ENV === 'production' ? 0.1 : 1.0,

  // Session Replay
  replaysSessionSampleRate: ENV === 'production' ? 0.1 : 0.5,
  replaysOnErrorSampleRate: 1.0,

  // Release Tracking
  release: process.env.NEXT_PUBLIC_APP_VERSION,

  // Integrations
  integrations: [
    // Captures console.error() calls
    Sentry.captureConsoleIntegration({
      levels: ['error', 'warn'],
    }),

    // Session replay
    Sentry.replayIntegration({
      maskAllText: false,
      blockAllMedia: false,
      networkDetailAllowUrls: ['/api'],
    }),

    // User feedback widget
    Sentry.feedbackIntegration({
      colorScheme: 'auto',
      showBranding: false,
      themeDark: {
        background: '#1a1a1a',
        inputBackground: '#2a2a2a',
        submitBackground: '#0ea5e9',
      },
    }),
  ],

  // Privacy & Security
  sendDefaultPii: false,

  // Filtering
  beforeSend(event, hint) {
    // Filter out non-critical errors in development
    if (ENV === 'development') {
      if (event.level === 'log' || event.level === 'debug') {
        return null;
      }
    }

    // Don't send events without a DSN
    if (!SENTRY_DSN) {
      return null;
    }

    // Filter out specific errors
    const error = hint.originalException;
    if (error && error instanceof Error) {
      // Filter out network errors that are expected
      if (
        error.message?.includes('NetworkError') &&
        error.message?.includes('favicon')
      ) {
        return null;
      }

      // Filter out browser extension errors
      if (error.message?.includes('extension://')) {
        return null;
      }
    }

    return event;
  },

  // Error filtering
  ignoreErrors: [
    // Browser errors
    'ResizeObserver loop limit exceeded',
    'ResizeObserver loop completed with undelivered notifications',
    'Non-Error promise rejection captured',

    // Network errors
    'NetworkError',
    'Failed to fetch',

    // React errors we handle
    'Hydration failed',
    'There was an error while hydrating',

    // Third-party errors
    /extensions\//i,
    /^chrome:\/\//i,
    /^moz-extension:\/\//i,
  ],

  // Transaction filtering
  ignoreTransactions: ['/api/health', '/api/metrics', '/_next/static'],
});
