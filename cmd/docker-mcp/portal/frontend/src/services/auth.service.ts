'use client';

import type { ApiResponse, TokenResponse, User } from '@/types';
import type {
  UserSession,
  AuditLogEntry,
  ApiResponseData,
  MSALInstance,
  MSALLogoutRequest,
} from '@/types/global';

/**
 * Authentication Service
 * Handles communication with the backend authentication API
 */
class AuthService {
  private baseUrl: string;
  private defaultHeaders: Record<string, string>;

  constructor() {
    this.baseUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
    this.defaultHeaders = {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    };
  }

  /**
   * Create authenticated headers with token
   */
  private getAuthHeaders(token: string): Record<string, string> {
    return {
      ...this.defaultHeaders,
      Authorization: `Bearer ${token}`,
    };
  }

  /**
   * Make authenticated API request with error handling
   */
  private async makeRequest<T>(
    endpoint: string,
    options: {
      method?: string;
      headers?: Record<string, string>;
      body?: string;
      signal?: AbortSignal;
    } = {},
    token?: string
  ): Promise<ApiResponse<T>> {
    try {
      const url = `${this.baseUrl}${endpoint}`;
      const headers = token ? this.getAuthHeaders(token) : this.defaultHeaders;

      const config: RequestInit = {
        method: options.method || 'GET',
        headers,
        body: options.body,
        signal: options.signal,
      };

      // Add request timeout
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 10000); // 10 second timeout
      config.signal = controller.signal;

      const response = await fetch(url, config);
      clearTimeout(timeoutId);

      // Handle different response types
      let data: ApiResponseData;
      const contentType = response.headers.get('content-type');

      if (contentType && contentType.includes('application/json')) {
        data = await response.json();
      } else {
        data = await response.text();
      }

      if (!response.ok) {
        // Handle HTTP errors
        let errorMessage = `HTTP ${response.status}: ${response.statusText}`;

        if (typeof data === 'object' && data !== null) {
          const errorData = data as Record<string, unknown>;
          errorMessage =
            typeof errorData.error === 'string'
              ? errorData.error
              : typeof errorData.message === 'string'
                ? errorData.message
                : errorMessage;
        } else if (typeof data === 'string') {
          errorMessage = data || errorMessage;
        }

        return {
          success: false,
          error: errorMessage,
          data: undefined,
        };
      }

      return {
        success: true,
        data: data as T,
        error: undefined,
      };
    } catch (error: unknown) {
      const typedError = error as Error;
      console.error('API request failed:', error);

      let errorMessage = 'Network error occurred';

      if (typedError.name === 'AbortError') {
        errorMessage = 'Request timeout';
      } else if (typedError.message) {
        errorMessage = typedError.message;
      }

      return {
        success: false,
        error: errorMessage,
        data: undefined,
      };
    }
  }

  /**
   * Exchange Azure AD token for backend JWT token
   */
  async exchangeToken(azureToken: string): Promise<TokenResponse | null> {
    try {
      const response = await this.makeRequest<TokenResponse>(
        '/api/auth/token',
        {
          method: 'POST',
          body: JSON.stringify({
            azureToken,
            provider: 'azure-ad',
          }),
        }
      );

      if (response.success && response.data) {
        return response.data;
      }

      console.error('Token exchange failed:', response.error);
      return null;
    } catch (error) {
      console.error('Token exchange error:', error);
      return null;
    }
  }

  /**
   * Get user profile from backend
   */
  async getUserProfile(token: string): Promise<User | null> {
    try {
      const response = await this.makeRequest<User>(
        '/api/auth/profile',
        {
          method: 'GET',
        },
        token
      );

      if (response.success && response.data) {
        return response.data;
      }

      console.error('Get user profile failed:', response.error);
      return null;
    } catch (error) {
      console.error('Get user profile error:', error);
      return null;
    }
  }

  /**
   * Validate token with backend
   */
  async validateToken(token: string): Promise<boolean> {
    try {
      const response = await this.makeRequest(
        '/api/auth/validate',
        {
          method: 'POST',
        },
        token
      );

      return response.success;
    } catch (error) {
      console.error('Token validation error:', error);
      return false;
    }
  }

  /**
   * Refresh JWT token
   */
  async refreshToken(refreshToken: string): Promise<TokenResponse | null> {
    try {
      const response = await this.makeRequest<TokenResponse>(
        '/api/auth/refresh',
        {
          method: 'POST',
          body: JSON.stringify({ refreshToken }),
        }
      );

      if (response.success && response.data) {
        return response.data;
      }

      console.error('Token refresh failed:', response.error);
      return null;
    } catch (error) {
      console.error('Token refresh error:', error);
      return null;
    }
  }

  /**
   * Logout from backend (invalidate session)
   */
  async logout(token: string): Promise<boolean> {
    try {
      const response = await this.makeRequest(
        '/api/auth/logout',
        {
          method: 'POST',
        },
        token
      );

      return response.success;
    } catch (error) {
      console.error('Logout error:', error);
      return false;
    }
  }

  /**
   * Update user profile
   */
  async updateProfile(
    token: string,
    updates: Partial<User>
  ): Promise<User | null> {
    try {
      const response = await this.makeRequest<User>(
        '/api/auth/profile',
        {
          method: 'PUT',
          body: JSON.stringify(updates),
        },
        token
      );

      if (response.success && response.data) {
        return response.data;
      }

      console.error('Update profile failed:', response.error);
      return null;
    } catch (error) {
      console.error('Update profile error:', error);
      return null;
    }
  }

  /**
   * Change user password (if applicable)
   */
  async changePassword(
    token: string,
    currentPassword: string,
    newPassword: string
  ): Promise<boolean> {
    try {
      const response = await this.makeRequest(
        '/api/auth/change-password',
        {
          method: 'POST',
          body: JSON.stringify({
            currentPassword,
            newPassword,
          }),
        },
        token
      );

      return response.success;
    } catch (error) {
      console.error('Change password error:', error);
      return false;
    }
  }

  /**
   * Get user sessions
   */
  async getUserSessions(token: string): Promise<UserSession[] | null> {
    try {
      const response = await this.makeRequest<UserSession[]>(
        '/api/auth/sessions',
        {
          method: 'GET',
        },
        token
      );

      if (response.success && response.data) {
        return response.data;
      }

      console.error('Get user sessions failed:', response.error);
      return null;
    } catch (error) {
      console.error('Get user sessions error:', error);
      return null;
    }
  }

  /**
   * Revoke user session
   */
  async revokeSession(token: string, sessionId: string): Promise<boolean> {
    try {
      const response = await this.makeRequest(
        `/api/auth/sessions/${sessionId}`,
        {
          method: 'DELETE',
        },
        token
      );

      return response.success;
    } catch (error) {
      console.error('Revoke session error:', error);
      return false;
    }
  }

  /**
   * Check if user has specific role
   */
  async checkRole(token: string, role: string): Promise<boolean> {
    try {
      const response = await this.makeRequest(
        `/api/auth/roles/${role}`,
        {
          method: 'GET',
        },
        token
      );

      return response.success;
    } catch (error) {
      console.error('Check role error:', error);
      return false;
    }
  }

  /**
   * Check if user has specific permission
   */
  async checkPermission(token: string, permission: string): Promise<boolean> {
    try {
      const response = await this.makeRequest(
        `/api/auth/permissions/${permission}`,
        {
          method: 'GET',
        },
        token
      );

      return response.success;
    } catch (error) {
      console.error('Check permission error:', error);
      return false;
    }
  }

  /**
   * Get user audit log
   */
  async getAuditLog(
    token: string,
    options?: {
      page?: number;
      limit?: number;
      startDate?: string;
      endDate?: string;
    }
  ): Promise<AuditLogEntry[] | null> {
    try {
      const params = new URLSearchParams();

      if (options?.page) params.append('page', options.page.toString());
      if (options?.limit) params.append('limit', options.limit.toString());
      if (options?.startDate) params.append('startDate', options.startDate);
      if (options?.endDate) params.append('endDate', options.endDate);

      const queryString = params.toString();
      const endpoint = `/api/auth/audit${queryString ? `?${queryString}` : ''}`;

      const response = await this.makeRequest<AuditLogEntry[]>(
        endpoint,
        {
          method: 'GET',
        },
        token
      );

      if (response.success && response.data) {
        return response.data;
      }

      console.error('Get audit log failed:', response.error);
      return null;
    } catch (error) {
      console.error('Get audit log error:', error);
      return null;
    }
  }
}

/**
 * Token Management Utility
 */
class TokenManager {
  private static readonly TOKEN_KEY = 'mcp_portal_token';
  private static readonly REFRESH_TOKEN_KEY = 'mcp_portal_refresh_token';
  private static readonly TOKEN_EXPIRY_KEY = 'mcp_portal_token_expiry';

  /**
   * Store tokens in localStorage
   */
  static storeTokens(tokenResponse: TokenResponse): void {
    if (typeof window === 'undefined') return;

    try {
      localStorage.setItem(this.TOKEN_KEY, tokenResponse.accessToken);
      localStorage.setItem(this.REFRESH_TOKEN_KEY, tokenResponse.refreshToken);

      // Calculate expiry time
      const expiryTime = Date.now() + tokenResponse.expiresIn * 1000;
      localStorage.setItem(this.TOKEN_EXPIRY_KEY, expiryTime.toString());
    } catch (error) {
      console.error('Failed to store tokens:', error);
    }
  }

  /**
   * Get access token from localStorage
   */
  static getAccessToken(): string | null {
    if (typeof window === 'undefined') return null;

    try {
      return localStorage.getItem(this.TOKEN_KEY);
    } catch {
      return null;
    }
  }

  /**
   * Get refresh token from localStorage
   */
  static getRefreshToken(): string | null {
    if (typeof window === 'undefined') return null;

    try {
      return localStorage.getItem(this.REFRESH_TOKEN_KEY);
    } catch {
      return null;
    }
  }

  /**
   * Check if token is expired
   */
  static isTokenExpired(): boolean {
    if (typeof window === 'undefined') return true;

    try {
      const expiryTime = localStorage.getItem(this.TOKEN_EXPIRY_KEY);
      if (!expiryTime) return true;

      return Date.now() > parseInt(expiryTime);
    } catch {
      return true;
    }
  }

  /**
   * Clear all tokens from localStorage
   */
  static clearTokens(): void {
    if (typeof window === 'undefined') return;

    try {
      localStorage.removeItem(this.TOKEN_KEY);
      localStorage.removeItem(this.REFRESH_TOKEN_KEY);
      localStorage.removeItem(this.TOKEN_EXPIRY_KEY);
    } catch {
      // Silently fail
    }
  }

  /**
   * Get time until token expiry in milliseconds
   */
  static getTimeUntilExpiry(): number {
    if (typeof window === 'undefined') return 0;

    try {
      const expiryTime = localStorage.getItem(this.TOKEN_EXPIRY_KEY);
      if (!expiryTime) return 0;

      const timeRemaining = parseInt(expiryTime) - Date.now();
      return Math.max(0, timeRemaining);
    } catch {
      return 0;
    }
  }
}

/**
 * Enhanced Auth Service with MSAL integration
 * Provides methods needed by api-client.ts
 */
class EnhancedAuthService extends AuthService {
  private msalInstance: MSALInstance | null = null;

  /**
   * Initialize MSAL instance (lazy loaded to avoid SSR issues)
   */
  private async getMsalInstance(): Promise<MSALInstance | null> {
    if (this.msalInstance) {
      return this.msalInstance;
    }

    if (typeof window === 'undefined') {
      return null; // SSR fallback
    }

    try {
      const { PublicClientApplication } = await import('@azure/msal-browser');
      const { msalConfig } = await import('@/config/msal.config');

      // Create instance and properly type cast to our interface
      const msalApp = new PublicClientApplication(msalConfig);
      await msalApp.initialize();

      // Type-safe assignment with proper interface mapping
      this.msalInstance = msalApp as unknown as MSALInstance;
      return this.msalInstance;
    } catch (error) {
      console.error('Failed to initialize MSAL:', error);
      return null;
    }
  }

  /**
   * Get access token from localStorage or MSAL
   */
  async getAccessToken(): Promise<string | null> {
    try {
      // First try localStorage (backend JWT token)
      const storedToken = TokenManager.getAccessToken();
      if (storedToken && !TokenManager.isTokenExpired()) {
        return storedToken;
      }

      // If no valid token, try to get from MSAL silently
      const msalInstance = await this.getMsalInstance();
      if (!msalInstance) {
        return null;
      }

      const accounts = msalInstance.getAllAccounts();
      if (accounts.length === 0) {
        return null;
      }

      const { silentRequest } = await import('@/config/msal.config');
      const silentResult = await msalInstance.acquireTokenSilent({
        ...silentRequest,
        account: accounts[0],
      });

      if (silentResult.accessToken) {
        // Exchange Azure token for backend JWT
        const tokenResponse = await this.exchangeToken(
          silentResult.accessToken
        );
        if (tokenResponse) {
          TokenManager.storeTokens(tokenResponse);
          return tokenResponse.accessToken;
        }
      }

      return null;
    } catch (error) {
      console.error('Failed to get access token:', error);
      return null;
    }
  }

  /**
   * Sign out from both MSAL and clear local tokens
   */
  async signOut(): Promise<void> {
    try {
      // Clear local tokens first
      TokenManager.clearTokens();

      // Sign out from MSAL if available
      const msalInstance = await this.getMsalInstance();
      if (msalInstance) {
        const accounts = msalInstance.getAllAccounts();
        if (accounts.length > 0) {
          // Create proper logout request object
          const logoutRequest: MSALLogoutRequest = {
            account: accounts[0],
            postLogoutRedirectUri: window.location.origin,
          };
          await msalInstance.logoutRedirect(logoutRequest);
        }
      }
    } catch (error) {
      console.error('Sign out error:', error);
      // Even if MSAL signout fails, ensure local tokens are cleared
      TokenManager.clearTokens();
    }
  }

  /**
   * Check if user is authenticated
   */
  async isAuthenticated(): Promise<boolean> {
    try {
      const token = await this.getAccessToken();
      return !!token;
    } catch {
      return false;
    }
  }

  /**
   * Get current user info
   */
  async getCurrentUser(): Promise<User | null> {
    try {
      const token = await this.getAccessToken();
      if (!token) {
        return null;
      }

      return await this.getUserProfile(token);
    } catch (error) {
      console.error('Failed to get current user:', error);
      return null;
    }
  }

  /**
   * Initialize authentication (call on app startup)
   */
  async initialize(): Promise<boolean> {
    try {
      const msalInstance = await this.getMsalInstance();
      if (!msalInstance) {
        return false;
      }

      // Handle redirect response
      const response = await msalInstance.handleRedirectPromise();
      if (response) {
        // Successfully authenticated, exchange token
        const tokenResponse = await this.exchangeToken(response.accessToken);
        if (tokenResponse) {
          TokenManager.storeTokens(tokenResponse);
          return true;
        }
      }

      // Check if already authenticated
      return await this.isAuthenticated();
    } catch (error) {
      console.error('Auth initialization error:', error);
      return false;
    }
  }

  /**
   * Trigger interactive login
   */
  async login(): Promise<boolean> {
    try {
      const msalInstance = await this.getMsalInstance();
      if (!msalInstance) {
        throw new Error('MSAL not initialized');
      }

      const { loginRequest } = await import('@/config/msal.config');
      await msalInstance.loginRedirect(loginRequest);

      return true; // Redirect will happen, return true for now
    } catch (error) {
      // Use console.error for this case since this is a critical auth error
      console.error('Login error:', error);
      return false;
    }
  }
}

// Export singleton instance
export const authService = new EnhancedAuthService();
export { TokenManager };
