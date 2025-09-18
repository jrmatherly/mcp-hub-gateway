/**
 * Instrumentation file for Next.js 15
 * This file is loaded once when the server starts
 * Used for initializing monitoring and observability tools
 */

import * as Sentry from '@sentry/nextjs';

export async function register() {
  // Initialize Sentry for server-side
  if (process.env.NEXT_RUNTIME === 'nodejs') {
    await import('./sentry.server.config');
  }

  // Initialize Sentry for edge runtime
  if (process.env.NEXT_RUNTIME === 'edge') {
    await import('./sentry.edge.config');
  }
}

// Capture errors from nested React Server Components
export const onRequestError = Sentry.captureRequestError;
