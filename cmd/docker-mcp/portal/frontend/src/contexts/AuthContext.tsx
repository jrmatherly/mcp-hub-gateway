'use client';

import { type AccountInfo, type SilentRequest } from '@azure/msal-browser';
import { useAccount, useMsal } from '@azure/msal-react';

import type React from 'react';
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useReducer,
} from 'react';

import {
  clearAccountHint,
  loginRequest,
  setAccountHint,
  silentRequest,
} from '@/config/msal.config';
import { authService } from '@/services/auth.service';
import type { AuthState, User } from '@/types';

/**
 * Auth Context State Management
 */
type AuthAction =
  | { type: 'SET_LOADING'; payload: boolean }
  | { type: 'SET_USER'; payload: User | null }
  | { type: 'SET_TOKEN'; payload: string | null }
  | { type: 'SET_ERROR'; payload: string | null }
  | { type: 'SET_AUTHENTICATED'; payload: boolean }
  | { type: 'RESET_AUTH' };

function authReducer(state: AuthState, action: AuthAction): AuthState {
  switch (action.type) {
    case 'SET_LOADING':
      return { ...state, isLoading: action.payload };

    case 'SET_USER':
      return {
        ...state,
        user: action.payload,
        isAuthenticated: action.payload !== null,
      };

    case 'SET_TOKEN':
      return { ...state, token: action.payload };

    case 'SET_ERROR':
      return { ...state, error: action.payload, isLoading: false };

    case 'SET_AUTHENTICATED':
      return { ...state, isAuthenticated: action.payload };

    case 'RESET_AUTH':
      return {
        isAuthenticated: false,
        user: null,
        token: null,
        isLoading: false,
        error: null,
      };

    default:
      return state;
  }
}

const initialAuthState: AuthState = {
  isAuthenticated: false,
  user: null,
  token: null,
  isLoading: true,
  error: null,
};

/**
 * Auth Context Interface
 */
interface AuthContextValue {
  // State
  isAuthenticated: boolean;
  user: User | null;
  token: string | null;
  isLoading: boolean;
  error: string | null;

  // Actions
  login: () => Promise<void>;
  logout: () => Promise<void>;
  refreshToken: () => Promise<string | null>;
  clearError: () => void;

  // Permissions
  hasRole: (role: string) => boolean;
  hasAnyRole: (roles: string[]) => boolean;
  hasPermission: (permission: string) => boolean;

  // Account info
  account: AccountInfo | null;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

/**
 * Auth Provider Props
 */
interface AuthProviderProps {
  children: React.ReactNode;
}

/**
 * Auth Context Provider Component
 */
export function AuthContextProvider({ children }: AuthProviderProps) {
  const { instance, accounts, inProgress } = useMsal();
  const account = useAccount(accounts[0] || {});
  const [state, dispatch] = useReducer(authReducer, initialAuthState);

  /**
   * Clear error state
   */
  const clearError = useCallback(() => {
    dispatch({ type: 'SET_ERROR', payload: null });
  }, []);

  /**
   * Extract user data from MSAL account and backend
   */
  const extractUserFromAccount = useCallback(
    async (account: AccountInfo, accessToken: string): Promise<User | null> => {
      try {
        // Get user profile from backend API
        const backendUser = await authService.getUserProfile(accessToken);

        if (backendUser) {
          return backendUser;
        }

        // Fallback to MSAL account data
        const user: User = {
          id: account.homeAccountId || account.localAccountId,
          email: account.username || '',
          name: account.name || account.username || '',
          picture: undefined, // Will be fetched from Microsoft Graph if needed
          roles: [], // Will be populated from backend
          tenantId: account.tenantId || '',
          createdAt: new Date(),
          updatedAt: new Date(),
        };

        return user;
      } catch (error) {
        console.error('Error extracting user data:', error);
        return null;
      }
    },
    []
  );

  /**
   * Acquire access token silently
   */
  const acquireTokenSilently = useCallback(
    async (account: AccountInfo): Promise<string | null> => {
      try {
        const request: SilentRequest = {
          ...silentRequest,
          account,
        };

        const response = await instance.acquireTokenSilent(request);
        return response.accessToken;
      } catch (error) {
        console.error('Silent token acquisition failed:', error);
        return null;
      }
    },
    [instance]
  );

  /**
   * Login with redirect
   */
  const login = useCallback(async () => {
    try {
      dispatch({ type: 'SET_LOADING', payload: true });
      dispatch({ type: 'SET_ERROR', payload: null });

      await instance.loginRedirect(loginRequest);
    } catch (error) {
      console.error('Login error:', error);
      dispatch({
        type: 'SET_ERROR',
        payload: 'Login failed. Please try again.',
      });
      dispatch({ type: 'SET_LOADING', payload: false });
    }
  }, [instance]);

  /**
   * Logout
   */
  const logout = useCallback(async () => {
    try {
      dispatch({ type: 'SET_LOADING', payload: true });

      // Clear local state
      dispatch({ type: 'RESET_AUTH' });

      // Clear account hint
      clearAccountHint();

      // Logout from backend
      if (state.token) {
        try {
          await authService.logout(state.token);
        } catch (error) {
          console.error('Backend logout error:', error);
        }
      }

      // Clear local storage
      if (typeof window !== 'undefined') {
        try {
          localStorage.removeItem('mcp_portal_user');
          localStorage.removeItem('mcp_portal_token');
          sessionStorage.clear();
        } catch {
          // Silently handle storage errors
        }
      }

      // Logout from MSAL
      const logoutRequest = {
        account: account,
        postLogoutRedirectUri: window.location.origin,
      };

      await instance.logoutRedirect(logoutRequest);
    } catch (error) {
      console.error('Logout error:', error);
      dispatch({
        type: 'SET_ERROR',
        payload: 'Logout failed. Please try again.',
      });
      dispatch({ type: 'SET_LOADING', payload: false });
    }
  }, [instance, account, state.token]);

  /**
   * Refresh access token
   */
  const refreshToken = useCallback(async (): Promise<string | null> => {
    if (!account) return null;

    try {
      const newToken = await acquireTokenSilently(account);

      if (newToken) {
        dispatch({ type: 'SET_TOKEN', payload: newToken });

        // Store in localStorage for persistence
        if (typeof window !== 'undefined') {
          try {
            localStorage.setItem('mcp_portal_token', newToken);
          } catch {
            // Silently handle storage errors
          }
        }

        return newToken;
      }

      return null;
    } catch (error) {
      console.error('Token refresh error:', error);
      return null;
    }
  }, [account, acquireTokenSilently]);

  /**
   * Check if user has specific role
   */
  const hasRole = useCallback(
    (role: string): boolean => {
      return state.user?.roles?.includes(role) ?? false;
    },
    [state.user]
  );

  /**
   * Check if user has any of the specified roles
   */
  const hasAnyRole = useCallback(
    (roles: string[]): boolean => {
      if (!state.user?.roles) return false;
      return roles.some(role => state.user?.roles?.includes(role) ?? false);
    },
    [state.user]
  );

  /**
   * Check if user has specific permission
   */
  const hasPermission = useCallback(
    (permission: string): boolean => {
      // TODO: Implement granular permissions system
      // For now, treat permissions as roles
      return hasRole(permission);
    },
    [hasRole]
  );

  /**
   * Initialize authentication state
   */
  useEffect(() => {
    const initializeAuth = async () => {
      try {
        if (inProgress !== 'none') {
          // Wait for MSAL to finish processing
          return;
        }

        if (!account) {
          // No active account, user is not authenticated
          dispatch({ type: 'RESET_AUTH' });
          return;
        }

        // Get access token
        const accessToken = await acquireTokenSilently(account);

        if (!accessToken) {
          // Token acquisition failed, user needs to login
          dispatch({ type: 'RESET_AUTH' });
          return;
        }

        // Extract user data
        const user = await extractUserFromAccount(account, accessToken);

        if (!user) {
          // User extraction failed
          dispatch({
            type: 'SET_ERROR',
            payload: 'Failed to load user profile.',
          });
          return;
        }

        // Set account hint for future logins
        if (account.username) {
          setAccountHint(account.username);
        }

        // Update state
        dispatch({ type: 'SET_USER', payload: user });
        dispatch({ type: 'SET_TOKEN', payload: accessToken });
        dispatch({ type: 'SET_AUTHENTICATED', payload: true });

        // Store in localStorage
        if (typeof window !== 'undefined') {
          try {
            localStorage.setItem('mcp_portal_user', JSON.stringify(user));
            localStorage.setItem('mcp_portal_token', accessToken);
          } catch {
            // Silently handle storage errors
          }
        }
      } catch (error) {
        console.error('Auth initialization error:', error);
        dispatch({
          type: 'SET_ERROR',
          payload: 'Authentication initialization failed.',
        });
      } finally {
        dispatch({ type: 'SET_LOADING', payload: false });
      }
    };

    initializeAuth();
  }, [account, inProgress, acquireTokenSilently, extractUserFromAccount]);

  /**
   * Set up token refresh interval
   */
  useEffect(() => {
    if (!state.isAuthenticated || !state.token) return;

    // Refresh token every 50 minutes (tokens expire in 1 hour)
    const refreshInterval = setInterval(
      async () => {
        try {
          await refreshToken();
        } catch (error) {
          console.error('Scheduled token refresh failed:', error);
        }
      },
      50 * 60 * 1000
    );

    return () => clearInterval(refreshInterval);
  }, [state.isAuthenticated, state.token, refreshToken]);

  /**
   * Handle page visibility change for token refresh
   */
  useEffect(() => {
    const handleVisibilityChange = async () => {
      if (document.visibilityState === 'visible' && state.isAuthenticated) {
        // Check and refresh token when page becomes visible
        try {
          await refreshToken();
        } catch (error) {
          console.error('Visibility token refresh failed:', error);
        }
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () =>
      document.removeEventListener('visibilitychange', handleVisibilityChange);
  }, [state.isAuthenticated, refreshToken]);

  /**
   * Context value
   */
  const contextValue: AuthContextValue = {
    // State
    isAuthenticated: state.isAuthenticated,
    user: state.user,
    token: state.token,
    isLoading: state.isLoading,
    error: state.error,

    // Actions
    login,
    logout,
    refreshToken,
    clearError,

    // Permissions
    hasRole,
    hasAnyRole,
    hasPermission,

    // Account
    account,
  };

  return (
    <AuthContext.Provider value={contextValue}>{children}</AuthContext.Provider>
  );
}

/**
 * Hook to use auth context
 */
export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);

  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }

  return context;
}

/**
 * Hook to require authentication
 */
export function useRequireAuth(): AuthContextValue {
  const auth = useAuth();

  useEffect(() => {
    if (!auth.isLoading && !auth.isAuthenticated) {
      // Redirect to login if not authenticated
      auth.login();
    }
  }, [auth]);

  return auth;
}

/**
 * Hook for role-based access control
 */
export function usePermissions() {
  const { hasRole, hasAnyRole, hasPermission, user } = useAuth();

  return {
    hasRole,
    hasAnyRole,
    hasPermission,
    isAdmin: hasRole('admin'),
    isUser: hasRole('user'),
    isModerator: hasRole('moderator'),
    roles: user?.roles || [],
  };
}

/**
 * Hook for conditional rendering based on authentication
 */
export function useAuthGuard() {
  const { isAuthenticated, isLoading, error } = useAuth();

  const shouldRender = isAuthenticated && !isLoading;
  const shouldShowLoading = isLoading && !error;
  const shouldShowError = !!error;
  const shouldRedirectToLogin = !isAuthenticated && !isLoading && !error;

  return {
    shouldRender,
    shouldShowLoading,
    shouldShowError,
    shouldRedirectToLogin,
    isReady: !isLoading,
  };
}

/**
 * Export auth context for direct usage if needed
 */
export { AuthContext };
