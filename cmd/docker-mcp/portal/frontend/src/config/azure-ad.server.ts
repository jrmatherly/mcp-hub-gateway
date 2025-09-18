/**
 * Azure AD Server-Side Configuration
 *
 * SECURITY: This file contains server-side only configuration.
 * These values should NEVER be exposed to the client/browser.
 *
 * This configuration is used for:
 * - Server-side token validation
 * - Client credentials flow for service-to-service auth
 * - Confidential client operations
 */

import {
  ConfidentialClientApplication,
  type Configuration,
  LogLevel,
} from '@azure/msal-node';
import { env } from '@/env.mjs';

/**
 * Azure AD Confidential Client Configuration
 * Used for server-side authentication operations
 */
const msalConfig: Configuration = {
  auth: {
    clientId: env.AZURE_CLIENT_ID || 'your-client-id',
    authority: `https://login.microsoftonline.com/${env.AZURE_TENANT_ID || 'common'}`,
    clientSecret: env.AZURE_CLIENT_SECRET || 'your-client-secret',
  },
  system: {
    loggerOptions: {
      loggerCallback(_loglevel, message, containsPii) {
        if (!containsPii && env.NODE_ENV === 'development') {
          // Use console.warn for development logging (allowed by ESLint)
          console.warn(`[MSAL Server] ${message}`);
        }
      },
      piiLoggingEnabled: false,
      logLevel:
        env.NODE_ENV === 'development' ? LogLevel.Verbose : LogLevel.Error,
    },
  },
  cache: {
    // Use in-memory cache for server-side operations
    cachePlugin: undefined,
  },
};

/**
 * Singleton instance of MSAL Confidential Client
 * Used for server-side authentication operations
 */
let msalInstance: ConfidentialClientApplication | null = null;

/**
 * Get or create MSAL Confidential Client instance
 */
export function getMsalInstance(): ConfidentialClientApplication {
  if (!msalInstance) {
    msalInstance = new ConfidentialClientApplication(msalConfig);
  }
  return msalInstance;
}

/**
 * Server-side authentication configuration
 */
export const serverAuthConfig = {
  // OAuth endpoints
  tokenEndpoint: `https://login.microsoftonline.com/${env.AZURE_TENANT_ID || 'common'}/oauth2/v2.0/token`,
  authorizeEndpoint: `https://login.microsoftonline.com/${env.AZURE_TENANT_ID || 'common'}/oauth2/v2.0/authorize`,

  // Application identifiers (server-side only)
  clientId: env.AZURE_CLIENT_ID || 'your-client-id',
  clientSecret: env.AZURE_CLIENT_SECRET || 'your-client-secret',
  tenantId: env.AZURE_TENANT_ID || 'common',

  // Redirect URIs
  redirectUri: env.NEXT_PUBLIC_AZURE_REDIRECT_URI,
  postLogoutRedirectUri: env.NEXT_PUBLIC_AZURE_POST_LOGOUT_URI,

  // Scopes for server-side operations
  scopes: {
    // Default scopes for user authentication
    userScopes: ['openid', 'profile', 'email', 'User.Read'],

    // Scopes for application (service-to-service) authentication
    apiScopes: ['https://graph.microsoft.com/.default'],
  },

  // Session configuration
  session: {
    secret: env.JWT_SECRET,
    cookieName: env.SESSION_COOKIE_NAME,
    cookieOptions: {
      httpOnly: env.SESSION_COOKIE_HTTPONLY,
      secure: env.SESSION_COOKIE_SECURE,
      sameSite: env.SESSION_COOKIE_SAMESITE,
      maxAge: 60 * 60 * 24, // 24 hours
    },
  },
};

/**
 * Graph API configuration for server-side calls
 */
export const graphConfig = {
  graphMeEndpoint: 'https://graph.microsoft.com/v1.0/me',
  graphMailEndpoint: 'https://graph.microsoft.com/v1.0/me/messages',
  graphGroupsEndpoint: 'https://graph.microsoft.com/v1.0/me/memberOf',
};

/**
 * Validate the server configuration
 * Should be called during application startup
 *
 * Note: Basic validation is now handled by T3 Env with Zod schemas.
 * This function performs additional runtime checks.
 */
export function validateServerConfig() {
  try {
    // Additional validation logic for production
    if (env.NODE_ENV === 'production') {
      if (env.JWT_SECRET === 'development-secret-change-in-production-please') {
        throw new Error(
          'JWT_SECRET must be set to a secure value in production'
        );
      }

      if (!env.SESSION_COOKIE_SECURE) {
        console.warn(
          'Warning: SESSION_COOKIE_SECURE should be true in production with HTTPS'
        );
      }
    }

    return true;
  } catch (error) {
    console.error('Server configuration validation failed:', error);
    return false;
  }
}
