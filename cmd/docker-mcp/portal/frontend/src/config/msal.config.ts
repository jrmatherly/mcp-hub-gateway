import {
  BrowserCacheLocation,
  type Configuration,
  LogLevel,
} from '@azure/msal-browser';
import { authLogger } from '@/lib/logger';

/**
 * MSAL Configuration for Azure AD B2C Integration
 * Supports both development and production environments
 */

// Environment variables with fallbacks
const CLIENT_ID = process.env.NEXT_PUBLIC_AZURE_CLIENT_ID || '';
const TENANT_ID = process.env.NEXT_PUBLIC_AZURE_TENANT_ID || '';
const REDIRECT_URI =
  process.env.NEXT_PUBLIC_REDIRECT_URI ||
  (typeof window !== 'undefined'
    ? `${window.location.origin}/auth/callback`
    : 'http://localhost:3000/auth/callback');
const POST_LOGOUT_REDIRECT_URI =
  process.env.NEXT_PUBLIC_POST_LOGOUT_REDIRECT_URI ||
  (typeof window !== 'undefined'
    ? window.location.origin
    : 'http://localhost:3000');

// Authority URL construction
const AUTHORITY = `https://login.microsoftonline.com/${TENANT_ID}`;

// Known authorities for B2C scenarios
const KNOWN_AUTHORITIES = [
  'login.microsoftonline.com',
  `${TENANT_ID}.b2clogin.com`,
];

/**
 * MSAL Configuration Object
 * Production-ready configuration with security best practices
 */
export const msalConfig: Configuration = {
  auth: {
    clientId: CLIENT_ID,
    authority: AUTHORITY,
    redirectUri: REDIRECT_URI,
    postLogoutRedirectUri: POST_LOGOUT_REDIRECT_URI,

    // Security settings
    navigateToLoginRequestUrl: false, // Prevent navigation loops
    knownAuthorities: KNOWN_AUTHORITIES,

    // Cloud discovery metadata
    cloudDiscoveryMetadata: '',
    authorityMetadata: '',
  },

  cache: {
    cacheLocation: BrowserCacheLocation.LocalStorage, // Store tokens in localStorage
    storeAuthStateInCookie: true, // Store auth state in cookies for IE11 compatibility

    // Security: Clear sensitive data from sessionStorage
    secureCookies: process.env.NODE_ENV === 'production',

    // Token cleanup settings
    claimsBasedCachingEnabled: true,
  },

  system: {
    loggerOptions: {
      loggerCallback: (
        level: LogLevel,
        message: string,
        containsPii: boolean
      ) => {
        if (containsPii) {
          return; // Don't log PII
        }

        switch (level) {
          case LogLevel.Error:
            console.error('MSAL Error:', message);
            break;
          case LogLevel.Warning:
            console.warn('MSAL Warning:', message);
            break;
          case LogLevel.Info:
            authLogger.info(`MSAL Info: ${message}`);
            break;
          case LogLevel.Verbose:
            authLogger.debug(`MSAL Verbose: ${message}`);
            break;
        }
      },
      piiLoggingEnabled: false, // Never log PII
      logLevel:
        process.env.NODE_ENV === 'development'
          ? LogLevel.Info
          : LogLevel.Warning,
    },

    // Network configuration - allowNativeBroker is not available in web environment

    // Window configuration for popups
    windowHashTimeout: 60000,
    iframeHashTimeout: 6000,

    // Token renewal configuration
    tokenRenewalOffsetSeconds: 300, // Renew tokens 5 minutes before expiry

    // AsyncPopups for better UX (Chrome 88+)
    asyncPopups: true,
  },
};

/**
 * Login Request Configuration
 * Defines scopes and behavior for interactive login
 */
export const loginRequest = {
  scopes: [
    'openid',
    'profile',
    'email',
    'User.Read', // Microsoft Graph basic profile
  ],

  // Optional: Request specific claims
  extraQueryParameters: {
    // domain_hint: 'organizations', // Uncomment to restrict to org accounts
  },

  // Prompt behavior
  prompt: 'select_account' as const, // Always show account selection

  // Force refresh
  forceRefresh: false,
};

/**
 * Silent Token Request Configuration
 * Used for background token renewal
 */
export const silentRequest = {
  scopes: ['openid', 'profile', 'email', 'User.Read'],
  forceRefresh: false,
};

/**
 * Graph API Request Configuration
 * For Microsoft Graph API calls
 */
export const graphRequest = {
  scopes: ['User.Read', 'User.ReadBasic.All'],
};

/**
 * Custom API Request Configuration
 * For calling the MCP Portal backend API
 */
export const apiRequest = {
  scopes: [
    `api://${CLIENT_ID}/access_as_user`, // Custom scope for backend API
  ],
};

/**
 * Protected Resource Map
 * Maps API endpoints to required scopes
 */
export const protectedResources = {
  graphMeEndpoint: 'https://graph.microsoft.com/v1.0/me',
  mcpPortalApi: {
    endpoint: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080',
    scopes: [`api://${CLIENT_ID}/access_as_user`],
  },
};

/**
 * MSAL Instance Events Configuration
 * Event handlers for authentication events
 */
export const msalInstanceEvents = {
  LOGIN_SUCCESS: 'msal:loginSuccess',
  LOGIN_FAILURE: 'msal:loginFailure',
  LOGOUT_SUCCESS: 'msal:logoutSuccess',
  LOGOUT_FAILURE: 'msal:logoutFailure',
  ACQUIRE_TOKEN_SUCCESS: 'msal:acquireTokenSuccess',
  ACQUIRE_TOKEN_FAILURE: 'msal:acquireTokenFailure',
  SSO_SILENT_SUCCESS: 'msal:ssoSilentSuccess',
  SSO_SILENT_FAILURE: 'msal:ssoSilentFailure',
};

/**
 * Error Messages for User-Friendly Display
 */
export const msalErrorMessages = {
  user_cancelled: 'Login was cancelled by the user.',
  consent_required:
    'Additional consent is required. Please contact your administrator.',
  interaction_required: 'User interaction is required to complete the request.',
  login_required: 'Please sign in to continue.',
  token_expired: 'Your session has expired. Please sign in again.',
  invalid_grant: 'Invalid authentication. Please sign in again.',
  network_error:
    'Network error occurred. Please check your connection and try again.',
  server_error: 'Server error occurred. Please try again later.',
  temporarily_unavailable:
    'Service is temporarily unavailable. Please try again later.',
};

/**
 * Configuration Validation
 * Validates required environment variables
 */
export function validateMsalConfig(): { isValid: boolean; errors: string[] } {
  const errors: string[] = [];

  if (!CLIENT_ID) {
    errors.push('NEXT_PUBLIC_AZURE_CLIENT_ID is required');
  }

  if (!TENANT_ID) {
    errors.push('NEXT_PUBLIC_AZURE_TENANT_ID is required');
  }

  return {
    isValid: errors.length === 0,
    errors,
  };
}

/**
 * Helper function to get environment-specific redirect URI
 */
export function getRedirectUri(): string {
  if (typeof window === 'undefined') {
    return 'http://localhost:3000/auth/callback';
  }

  const { protocol, host } = window.location;
  return `${protocol}//${host}/auth/callback`;
}

/**
 * Helper function to get current account hint
 */
export function getAccountHint(): string | undefined {
  if (typeof window === 'undefined') return undefined;

  try {
    const hint = localStorage.getItem('mcp_portal_account_hint');
    return hint || undefined;
  } catch {
    return undefined;
  }
}

/**
 * Helper function to set account hint
 */
export function setAccountHint(email: string): void {
  if (typeof window === 'undefined') return;

  try {
    localStorage.setItem('mcp_portal_account_hint', email);
  } catch {
    // Silently fail if localStorage is not available
  }
}

/**
 * Helper function to clear account hint
 */
export function clearAccountHint(): void {
  if (typeof window === 'undefined') return;

  try {
    localStorage.removeItem('mcp_portal_account_hint');
  } catch {
    // Silently fail if localStorage is not available
  }
}
