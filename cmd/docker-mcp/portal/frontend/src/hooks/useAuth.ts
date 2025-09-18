'use client';

import type {
  AuthenticationResult,
  PopupRequest,
  RedirectRequest,
} from '@azure/msal-browser';
import { useIsAuthenticated, useMsal } from '@azure/msal-react';
import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';
import { loginRequest, msalErrorMessages } from '@/config/msal.config';
import {
  useAuth as useAuthContext,
  useRequireAuth as useRequireAuthContext,
} from '@/contexts/AuthContext';
import { authLogger } from '@/lib/logger';

/**
 * Enhanced authentication hook with additional utilities
 */
export function useAuth() {
  const authContext = useAuthContext();
  const { instance } = useMsal();
  const isAuthenticated = useIsAuthenticated();
  const router = useRouter();
  const [loginMode, setLoginMode] = useState<'redirect' | 'popup'>('redirect');

  /**
   * Login with popup window
   */
  const loginPopup =
    useCallback(async (): Promise<AuthenticationResult | null> => {
      try {
        const popupRequest: PopupRequest = {
          ...loginRequest,
          prompt: 'select_account',
        };

        const response = await instance.loginPopup(popupRequest);
        return response;
      } catch (error) {
        console.error('Popup login failed:', error);
        return null;
      }
    }, [instance]);

  /**
   * Login with redirect
   */
  const loginRedirect = useCallback(async (): Promise<void> => {
    try {
      const redirectRequest: RedirectRequest = {
        ...loginRequest,
        prompt: 'select_account',
      };

      await instance.loginRedirect(redirectRequest);
    } catch (error) {
      console.error('Redirect login failed:', error);
      throw error;
    }
  }, [instance]);

  /**
   * Smart login that chooses method based on context
   */
  const smartLogin = useCallback(async () => {
    // Use popup on mobile or when explicitly set
    if (loginMode === 'popup' || /Mobi|Android/i.test(navigator.userAgent)) {
      return loginPopup();
    } else {
      return loginRedirect();
    }
  }, [loginMode, loginPopup, loginRedirect]);

  /**
   * Enhanced login with error handling
   */
  const enhancedLogin = useCallback(
    async (options?: {
      mode?: 'popup' | 'redirect';
      redirectTo?: string;
      onSuccess?: (result: AuthenticationResult) => void;
      onError?: (error: unknown) => void;
    }) => {
      try {
        const mode = options?.mode || loginMode;
        let result: AuthenticationResult | null = null;

        if (mode === 'popup') {
          result = await loginPopup();
        } else {
          await loginRedirect();
          return; // Redirect doesn't return immediately
        }

        if (result) {
          options?.onSuccess?.(result);

          // Redirect after successful login
          if (options?.redirectTo) {
            router.push(options.redirectTo);
          }
        }

        return result;
      } catch (error: unknown) {
        const errorCode = (
          error as { errorCode?: keyof typeof msalErrorMessages }
        )?.errorCode;
        const errorMessage =
          (errorCode && msalErrorMessages[errorCode]) ||
          'Login failed. Please try again.';
        options?.onError?.(errorMessage);
        throw new Error(errorMessage);
      }
    },
    [loginMode, loginPopup, loginRedirect, router]
  );

  /**
   * Enhanced logout with cleanup
   */
  const enhancedLogout = useCallback(
    async (options?: { redirectTo?: string; clearStorage?: boolean }) => {
      try {
        // Custom cleanup before logout
        if (options?.clearStorage !== false) {
          // Clear application-specific data
          if (typeof window !== 'undefined') {
            try {
              localStorage.removeItem('mcp_portal_preferences');
              localStorage.removeItem('mcp_portal_cache');
              sessionStorage.clear();
            } catch {
              // Silently handle storage errors
            }
          }
        }

        // Use context logout (handles backend cleanup)
        await authContext.logout();

        // Custom redirect if specified
        if (options?.redirectTo) {
          router.push(options.redirectTo);
        }
      } catch (error) {
        console.error('Enhanced logout error:', error);
      }
    },
    [authContext, router]
  );

  return {
    // Original context methods
    ...authContext,

    // Enhanced methods
    loginPopup,
    loginRedirect,
    smartLogin,
    enhancedLogin,
    enhancedLogout,

    // Additional state
    isAuthenticatedMsal: isAuthenticated,
    loginMode,
    setLoginMode,
  };
}

/**
 * Hook for requiring authentication with custom options
 */
export function useRequireAuth(options?: {
  redirectTo?: string;
  requireRoles?: string[];
  requirePermissions?: string[];
  fallback?: React.ComponentType;
}) {
  const auth = useRequireAuthContext();
  const router = useRouter();
  const [hasAccess, setHasAccess] = useState<boolean | null>(null);

  useEffect(() => {
    if (auth.isLoading) {
      setHasAccess(null);
      return;
    }

    if (!auth.isAuthenticated) {
      setHasAccess(false);
      if (options?.redirectTo) {
        router.push(options.redirectTo);
      }
      return;
    }

    // Check role requirements
    if (options?.requireRoles && !auth.hasAnyRole(options.requireRoles)) {
      setHasAccess(false);
      router.push('/unauthorized');
      return;
    }

    // Check permission requirements
    if (options?.requirePermissions) {
      const hasAllPermissions = options.requirePermissions.every(permission =>
        auth.hasPermission(permission)
      );

      if (!hasAllPermissions) {
        setHasAccess(false);
        router.push('/unauthorized');
        return;
      }
    }

    setHasAccess(true);
  }, [auth, options, router]);

  return {
    ...auth,
    hasAccess,
    isReady: hasAccess !== null,
  };
}

/**
 * Hook for checking authentication status with loading states
 */
export function useAuthStatus() {
  const auth = useAuth();
  const [previousAuthState, setPreviousAuthState] = useState(
    auth.isAuthenticated
  );
  const [authTransition, setAuthTransition] = useState<
    'logging-in' | 'logging-out' | 'stable'
  >('stable');

  useEffect(() => {
    if (previousAuthState !== auth.isAuthenticated) {
      setAuthTransition(auth.isAuthenticated ? 'logging-in' : 'logging-out');

      // Reset transition state after a delay
      const timeout = setTimeout(() => {
        setAuthTransition('stable');
        setPreviousAuthState(auth.isAuthenticated);
      }, 500);

      return () => clearTimeout(timeout);
    }
  }, [auth.isAuthenticated, previousAuthState]);

  return {
    isAuthenticated: auth.isAuthenticated,
    isLoading: auth.isLoading,
    error: auth.error,
    user: auth.user,
    authTransition,
    isLoginInProgress: authTransition === 'logging-in',
    isLogoutInProgress: authTransition === 'logging-out',
    isStable: authTransition === 'stable',
  };
}

/**
 * Hook for handling authentication errors
 */
export function useAuthError() {
  const { error, clearError } = useAuth();
  const [displayError, setDisplayError] = useState<string | null>(null);
  const [errorHistory, setErrorHistory] = useState<string[]>([]);

  useEffect(() => {
    if (error) {
      setDisplayError(error);
      setErrorHistory(prev => [...prev, error].slice(-5)); // Keep last 5 errors
    }
  }, [error]);

  const dismissError = useCallback(() => {
    setDisplayError(null);
    clearError();
  }, [clearError]);

  const retryLastAction = useCallback(() => {
    dismissError();
    // Could implement retry logic based on error context
  }, [dismissError]);

  return {
    error: displayError,
    hasError: !!displayError,
    errorHistory,
    dismissError,
    retryLastAction,
  };
}

/**
 * Hook for authentication-related analytics and monitoring
 */
export function useAuthAnalytics() {
  const auth = useAuth();

  useEffect(() => {
    if (auth.isAuthenticated && auth.user) {
      // Track successful authentication
      authLogger.info('User authenticated', {
        userId: auth.user.id,
        email: auth.user.email,
        roles: auth.user.roles,
        timestamp: new Date().toISOString(),
      });

      // Could integrate with analytics service
      // analytics.track('user_authenticated', { ... });
    }
  }, [auth.isAuthenticated, auth.user]);

  useEffect(() => {
    if (auth.error) {
      // Track authentication errors
      console.error('Authentication error:', {
        error: auth.error,
        timestamp: new Date().toISOString(),
      });

      // Could integrate with error tracking service
      // errorTracking.captureException(auth.error);
    }
  }, [auth.error]);

  const trackUserAction = useCallback(
    (action: string, metadata?: Record<string, unknown>) => {
      if (auth.isAuthenticated && auth.user) {
        authLogger.info('User action', {
          userId: auth.user.id,
          action,
          metadata,
          timestamp: new Date().toISOString(),
        });

        // Could integrate with analytics service
        // analytics.track(action, { userId: auth.user.id, ...metadata });
      }
    },
    [auth.isAuthenticated, auth.user]
  );

  return {
    trackUserAction,
  };
}
