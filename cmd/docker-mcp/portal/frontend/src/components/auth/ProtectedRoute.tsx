'use client';

import { AlertCircle, Loader2, RefreshCw, Shield } from 'lucide-react';
import { useRouter } from 'next/navigation';
import React, { useEffect, useState } from 'react';
import { useAuth, useAuthGuard } from '@/contexts/AuthContext';
import { Button } from '@/components/ui/button';

/**
 * Protected Route Props
 */
interface ProtectedRouteProps {
  children: React.ReactNode;
  roles?: string[];
  permissions?: string[];
  fallback?: React.ComponentType;
  redirectTo?: string;
  requireAll?: boolean; // Require all roles/permissions or just one
  showLoading?: boolean;
  showUnauthorized?: boolean;
}

/**
 * Loading Component
 */
function LoadingScreen({ message = 'Loading...' }: { message?: string }) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="text-center space-y-4">
        <div className="inline-flex items-center justify-center w-16 h-16">
          <Loader2 className="w-8 h-8 animate-spin text-primary" />
        </div>
        <div className="space-y-2">
          <p className="text-lg font-medium text-foreground">{message}</p>
          <p className="text-sm text-muted-foreground">
            Please wait while we verify your access...
          </p>
        </div>
      </div>
    </div>
  );
}

/**
 * Unauthorized Component
 */
function UnauthorizedScreen({
  onRetry,
  missingRoles,
  missingPermissions,
}: {
  onRetry?: () => void;
  missingRoles?: string[];
  missingPermissions?: string[];
}) {
  const router = useRouter();

  return (
    <div className="min-h-screen flex items-center justify-center bg-background p-4">
      <div className="max-w-md w-full text-center space-y-6">
        <div className="inline-flex items-center justify-center w-20 h-20 bg-error-100 text-error-600 rounded-full dark:bg-error-900/20 dark:text-error-400">
          <Shield className="w-10 h-10" />
        </div>

        <div className="space-y-2">
          <h1 className="text-2xl font-bold text-foreground">Access Denied</h1>
          <p className="text-muted-foreground">
            You don't have permission to access this resource.
          </p>

          {(missingRoles || missingPermissions) && (
            <div className="mt-4 p-4 bg-muted rounded-lg text-left">
              <p className="text-sm font-medium text-foreground mb-2">
                Required Access:
              </p>

              {missingRoles && missingRoles.length > 0 && (
                <div className="mb-2">
                  <p className="text-xs text-muted-foreground mb-1">Roles:</p>
                  <div className="flex flex-wrap gap-1">
                    {missingRoles.map(role => (
                      <span
                        key={role}
                        className="px-2 py-1 text-xs bg-primary/10 text-primary rounded"
                      >
                        {role}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {missingPermissions && missingPermissions.length > 0 && (
                <div>
                  <p className="text-xs text-muted-foreground mb-1">
                    Permissions:
                  </p>
                  <div className="flex flex-wrap gap-1">
                    {missingPermissions.map(permission => (
                      <span
                        key={permission}
                        className="px-2 py-1 text-xs bg-secondary/50 text-foreground rounded"
                      >
                        {permission}
                      </span>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        <div className="space-y-3">
          {onRetry && (
            <Button onClick={onRetry} className="w-full">
              <RefreshCw className="w-4 h-4 mr-2" />
              Retry Access
            </Button>
          )}

          <Button
            variant="secondary"
            onClick={() => router.push('/dashboard')}
            className="w-full"
          >
            Go to Dashboard
          </Button>

          <div className="flex items-center justify-center gap-4 text-sm">
            <button
              type="button"
              onClick={() => router.back()}
              className="text-muted-foreground hover:text-foreground transition-colors"
            >
              Go Back
            </button>

            <span className="text-muted-foreground">•</span>

            <a
              href="mailto:support@mcp-portal.com?subject=Access Request"
              className="text-primary hover:underline"
            >
              Request Access
            </a>
          </div>
        </div>
      </div>
    </div>
  );
}

/**
 * Error Screen Component
 */
function ErrorScreen({
  error,
  onRetry,
  onLogin,
}: {
  error: string;
  onRetry?: () => void;
  onLogin?: () => void;
}) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-background p-4">
      <div className="max-w-md w-full text-center space-y-6">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-error-100 text-error-600 rounded-full dark:bg-error-900/20 dark:text-error-400">
          <AlertCircle className="w-8 h-8" />
        </div>

        <div className="space-y-2">
          <h1 className="text-xl font-bold text-foreground">
            Authentication Error
          </h1>
          <p className="text-muted-foreground">
            {error || 'An error occurred while verifying your access.'}
          </p>
        </div>

        <div className="space-y-3">
          {onRetry && (
            <Button onClick={onRetry} className="w-full">
              <RefreshCw className="w-4 h-4 mr-2" />
              Try Again
            </Button>
          )}

          {onLogin && (
            <Button variant="secondary" onClick={onLogin} className="w-full">
              Sign In Again
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}

/**
 * Main Protected Route Component
 */
export default function ProtectedRoute({
  children,
  roles = [],
  permissions = [],
  fallback: Fallback,
  redirectTo,
  requireAll = false,
  showLoading = true,
  showUnauthorized = true,
}: ProtectedRouteProps) {
  const auth = useAuth();
  const authGuard = useAuthGuard();
  const router = useRouter();
  const [retryCount, setRetryCount] = useState(0);
  const [accessChecked, setAccessChecked] = useState(false);
  const [hasAccess, setHasAccess] = useState(false);
  const [missingRoles, setMissingRoles] = useState<string[]>([]);
  const [missingPermissions, setMissingPermissions] = useState<string[]>([]);

  /**
   * Check user access based on roles and permissions
   */
  const checkAccess = React.useCallback(() => {
    if (!auth.isAuthenticated || !auth.user) {
      setHasAccess(false);
      setAccessChecked(true);
      return;
    }

    // Check role requirements
    let hasRequiredRoles = true;
    const missing: string[] = [];

    if (roles.length > 0) {
      if (requireAll) {
        // User must have ALL roles
        hasRequiredRoles = roles.every(role => auth.hasRole(role));
        missing.push(...roles.filter(role => !auth.hasRole(role)));
      } else {
        // User must have ANY role
        hasRequiredRoles = roles.some(role => auth.hasRole(role));
        if (!hasRequiredRoles) {
          missing.push(...roles);
        }
      }
    }

    // Check permission requirements
    let hasRequiredPermissions = true;
    const missingPerms: string[] = [];

    if (permissions.length > 0) {
      if (requireAll) {
        // User must have ALL permissions
        hasRequiredPermissions = permissions.every(permission =>
          auth.hasPermission(permission)
        );
        missingPerms.push(
          ...permissions.filter(permission => !auth.hasPermission(permission))
        );
      } else {
        // User must have ANY permission
        hasRequiredPermissions = permissions.some(permission =>
          auth.hasPermission(permission)
        );
        if (!hasRequiredPermissions) {
          missingPerms.push(...permissions);
        }
      }
    }

    const access = hasRequiredRoles && hasRequiredPermissions;

    setHasAccess(access);
    setMissingRoles(missing);
    setMissingPermissions(missingPerms);
    setAccessChecked(true);
  }, [auth, roles, permissions, requireAll]);

  /**
   * Handle retry logic
   */
  const handleRetry = React.useCallback(async () => {
    setRetryCount(prev => prev + 1);
    setAccessChecked(false);

    try {
      // Try to refresh token
      await auth.refreshToken();

      // Re-check access
      setTimeout(checkAccess, 1000);
    } catch (error) {
      console.error('Retry failed:', error);
      setAccessChecked(true);
    }
  }, [auth, checkAccess]);

  /**
   * Handle login redirect
   */
  const handleLogin = React.useCallback(() => {
    auth.login();
  }, [auth]);

  /**
   * Effect to check access when auth state changes
   */
  useEffect(() => {
    if (!authGuard.isReady) return;

    checkAccess();
  }, [authGuard.isReady, checkAccess]);

  /**
   * Effect to handle redirects
   */
  useEffect(() => {
    if (!accessChecked) return;

    if (authGuard.shouldRedirectToLogin && redirectTo) {
      router.push(redirectTo);
      return;
    }

    if (!hasAccess && !showUnauthorized && redirectTo) {
      router.push(redirectTo);
    }
  }, [
    accessChecked,
    authGuard.shouldRedirectToLogin,
    hasAccess,
    showUnauthorized,
    redirectTo,
    router,
  ]);

  // Show custom fallback if provided
  if (Fallback && !hasAccess) {
    return <Fallback />;
  }

  // Show error screen
  if (authGuard.shouldShowError) {
    return (
      <ErrorScreen
        error={auth.error || 'Authentication error'}
        onRetry={retryCount < 3 ? handleRetry : undefined}
        onLogin={handleLogin}
      />
    );
  }

  // Show loading screen
  if (authGuard.shouldShowLoading || !accessChecked) {
    if (!showLoading) {
      return null;
    }

    let loadingMessage = 'Authenticating...';

    if (auth.isAuthenticated && !accessChecked) {
      loadingMessage = 'Checking permissions...';
    }

    return <LoadingScreen message={loadingMessage} />;
  }

  // Show unauthorized screen
  if (!hasAccess) {
    if (!showUnauthorized) {
      return null;
    }

    return (
      <UnauthorizedScreen
        onRetry={retryCount < 3 ? handleRetry : undefined}
        missingRoles={missingRoles}
        missingPermissions={missingPermissions}
      />
    );
  }

  // User has access, render children
  return <>{children}</>;
}

/**
 * Higher-order component for protecting routes
 */
export function withProtectedRoute<P extends object>(
  Component: React.ComponentType<P>,
  options?: Omit<ProtectedRouteProps, 'children'>
) {
  const ProtectedComponent = (props: P) => {
    return (
      <ProtectedRoute {...options}>
        <Component {...props} />
      </ProtectedRoute>
    );
  };

  ProtectedComponent.displayName = `withProtectedRoute(${Component.displayName || Component.name})`;

  return ProtectedComponent;
}

/**
 * Hook for using protected route logic in components
 */
export function useProtectedRoute(
  options: Omit<ProtectedRouteProps, 'children'> = {}
) {
  const auth = useAuth();
  const [hasAccess, setHasAccess] = useState<boolean | null>(null);

  const { roles = [], permissions = [], requireAll = false } = options;

  useEffect(() => {
    if (!auth.isAuthenticated || !auth.user) {
      setHasAccess(false);
      return;
    }

    // Check role requirements
    let hasRequiredRoles = true;
    if (roles.length > 0) {
      hasRequiredRoles = requireAll
        ? roles.every(role => auth.hasRole(role))
        : roles.some(role => auth.hasRole(role));
    }

    // Check permission requirements
    let hasRequiredPermissions = true;
    if (permissions.length > 0) {
      hasRequiredPermissions = requireAll
        ? permissions.every(permission => auth.hasPermission(permission))
        : permissions.some(permission => auth.hasPermission(permission));
    }

    setHasAccess(hasRequiredRoles && hasRequiredPermissions);
  }, [auth, roles, permissions, requireAll]);

  return {
    hasAccess,
    isAuthenticated: auth.isAuthenticated,
    isLoading: auth.isLoading || hasAccess === null,
    user: auth.user,
  };
}
