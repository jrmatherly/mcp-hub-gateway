// Global type definitions for MCP Portal Frontend

declare global {
  namespace NodeJS {
    // Override default Timeout interface for better compatibility
    type Timeout = ReturnType<typeof setTimeout>;
  }

  interface Window {
    // Azure MSAL related globals
    msal?: {
      PublicClientApplication: unknown;
    };

    // Potential runtime globals
    __NEXT_DATA__?: {
      props: Record<string, unknown>;
      page: string;
      query: Record<string, unknown>;
      buildId: string;
      runtimeConfig?: Record<string, unknown>;
    };
  }
}

// Session and audit log types for auth service
export interface UserSession {
  id: string;
  userId: string;
  deviceId?: string;
  deviceType?: string;
  userAgent?: string;
  ipAddress?: string;
  location?: string;
  createdAt: string;
  lastActiveAt: string;
  expiresAt: string;
  isActive: boolean;
}

export interface AuditLogEntry {
  id: string;
  userId: string;
  action: string;
  resource?: string;
  resourceId?: string;
  metadata?: Record<string, unknown>;
  ipAddress?: string;
  userAgent?: string;
  timestamp: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
}

// Enhanced RequestInit interface with better typing
export interface TypedRequestInit
  extends Omit<globalThis.RequestInit, 'headers'> {
  headers?: Record<string, string> | Headers;
  timeout?: number;
  retries?: number;
  retryDelay?: number;
}

// Symbol hint type for better type safety
export type SymbolToPrimitiveHint = 'number' | 'string' | 'default';

// API response data that can be either JSON or text
export type ApiResponseData = Record<string, unknown> | string | null;

// MSAL related types for better typing - Updated to match @azure/msal-browser types
export interface MSALAccount {
  homeAccountId: string;
  localAccountId: string;
  environment: string;
  tenantId: string;
  username: string;
  name?: string;
}

export interface MSALTokenRequest {
  scopes: string[];
  account?: MSALAccount;
  authority?: string;
  correlationId?: string;
  forceRefresh?: boolean;
}

export interface MSALTokenResponse {
  accessToken: string;
  account: MSALAccount;
  authority: string;
  correlationId: string;
  expiresOn: Date | null; // Updated to match MSAL's AuthenticationResult type
  extExpiresOn?: Date;
  familyId?: string;
  fromCache: boolean;
  idToken: string;
  idTokenClaims: Record<string, unknown>;
  scopes: string[];
  tenantId: string;
  uniqueId: string;
}

// Updated to extend proper MSAL logout request interface
export interface MSALLogoutRequest {
  account?: MSALAccount;
  postLogoutRedirectUri?: string;
  authority?: string;
  correlationId?: string;
  idTokenHint?: string;
  logoutHint?: string;
  onRedirectNavigate?: (url: string) => boolean | void;
}

export interface MSALInstance {
  initialize(): Promise<void>;
  getAllAccounts(): MSALAccount[];
  acquireTokenSilent(_request: MSALTokenRequest): Promise<MSALTokenResponse>;
  loginRedirect(_request: MSALTokenRequest): Promise<void>;
  loginPopup(_request: MSALTokenRequest): Promise<MSALTokenResponse>;
  logoutRedirect(_request?: MSALLogoutRequest): Promise<void>; // Updated to use proper type
  handleRedirectPromise(): Promise<MSALTokenResponse | null>;
}

export {};
