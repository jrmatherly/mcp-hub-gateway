'use client';

import {
  type AuthenticationResult,
  type EventMessage,
  EventType,
  PublicClientApplication,
} from '@azure/msal-browser';
import { MsalProvider } from '@azure/msal-react';
import React, { useEffect, useState } from 'react';
import {
  msalConfig,
  msalInstanceEvents,
  validateMsalConfig,
} from '@/config/msal.config';
import { authLogger } from '@/lib/logger';
import { Button } from '@/components/ui/button';

/**
 * MSAL Instance - Singleton pattern for browser authentication
 */
let msalInstance: PublicClientApplication | null = null;

/**
 * Initialize MSAL instance with proper error handling
 */
function initializeMsal(): PublicClientApplication | null {
  try {
    // Validate configuration before initialization
    const validation = validateMsalConfig();
    if (!validation.isValid) {
      authLogger.error(
        'MSAL Configuration Error:',
        validation.errors.join(', ')
      );
      console.error('MSAL Configuration Errors:', validation.errors);
      return null;
    }

    // Create singleton instance
    if (!msalInstance) {
      msalInstance = new PublicClientApplication(msalConfig);

      // Set up event callbacks
      msalInstance.addEventCallback((event: EventMessage) => {
        handleMsalEvent(event);
      });
    }

    return msalInstance;
  } catch (error) {
    authLogger.error('Failed to initialize MSAL:', error);
    return null;
  }
}

/**
 * Handle MSAL events for logging and state management
 */
function handleMsalEvent(event: EventMessage): void {
  if (!event || !event.eventType) return;

  switch (event.eventType) {
    case EventType.LOGIN_SUCCESS:
      if (event.payload && 'account' in event.payload) {
        const payload = event.payload as AuthenticationResult;
        authLogger.info('Login successful', {
          username: payload.account?.username,
        });

        // Custom event for application-level handling
        window.dispatchEvent(
          new CustomEvent(msalInstanceEvents.LOGIN_SUCCESS, {
            detail: payload,
          })
        );
      }
      break;

    case EventType.LOGIN_FAILURE:
      authLogger.error('Login failed', event.error);
      window.dispatchEvent(
        new CustomEvent(msalInstanceEvents.LOGIN_FAILURE, {
          detail: event.error,
        })
      );
      break;

    case EventType.LOGOUT_SUCCESS:
      authLogger.info('Logout successful');

      // Clear application state
      if (typeof window !== 'undefined') {
        try {
          localStorage.removeItem('mcp_portal_user');
          sessionStorage.clear();
        } catch {
          // Silently handle storage errors
        }
      }

      window.dispatchEvent(new CustomEvent(msalInstanceEvents.LOGOUT_SUCCESS));
      break;

    case EventType.LOGOUT_FAILURE:
      authLogger.error('Logout failed', event.error);
      window.dispatchEvent(
        new CustomEvent(msalInstanceEvents.LOGOUT_FAILURE, {
          detail: event.error,
        })
      );
      break;

    case EventType.ACQUIRE_TOKEN_SUCCESS:
      if (event.payload && 'account' in event.payload) {
        const payload = event.payload as AuthenticationResult;
        authLogger.info('Token acquired', {
          username: payload.account?.username,
        });

        window.dispatchEvent(
          new CustomEvent(msalInstanceEvents.ACQUIRE_TOKEN_SUCCESS, {
            detail: payload,
          })
        );
      }
      break;

    case EventType.ACQUIRE_TOKEN_FAILURE:
      authLogger.error('Token acquisition failed', event.error);
      window.dispatchEvent(
        new CustomEvent(msalInstanceEvents.ACQUIRE_TOKEN_FAILURE, {
          detail: event.error,
        })
      );
      break;

    case EventType.SSO_SILENT_SUCCESS:
      if (event.payload && 'account' in event.payload) {
        const payload = event.payload as AuthenticationResult;
        authLogger.info('SSO silent success', {
          username: payload.account?.username,
        });

        window.dispatchEvent(
          new CustomEvent(msalInstanceEvents.SSO_SILENT_SUCCESS, {
            detail: payload,
          })
        );
      }
      break;

    case EventType.SSO_SILENT_FAILURE:
      // Silent failures are expected and should not be logged as errors
      authLogger.debug('SSO silent failure', event.error);
      window.dispatchEvent(
        new CustomEvent(msalInstanceEvents.SSO_SILENT_FAILURE, {
          detail: event.error,
        })
      );
      break;

    default:
      // Log other events in development
      if (process.env.NODE_ENV === 'development') {
        authLogger.debug('MSAL Event', {
          eventType: event.eventType,
          payload: event.payload,
        });
      }
  }
}

/**
 * Auth Provider Props
 */
interface AuthProviderProps {
  children: React.ReactNode;
}

/**
 * Auth Provider Error Boundary State
 */
interface AuthProviderState {
  hasError: boolean;
  error?: Error;
  isInitialized: boolean;
}

/**
 * Auth Provider Error Fallback Component
 */
function AuthErrorFallback({ error }: { error?: Error }) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="max-w-md w-full p-6">
        <div className="text-center space-y-4">
          <div className="inline-flex items-center justify-center w-12 h-12 bg-error-100 text-error-600 rounded-full dark:bg-error-900/20 dark:text-error-400">
            <svg
              className="w-6 h-6"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z"
              />
            </svg>
          </div>

          <div>
            <h1 className="text-xl font-semibold text-foreground mb-2">
              Authentication Error
            </h1>
            <p className="text-muted-foreground text-sm mb-4">
              Unable to initialize the authentication system. Please check your
              configuration and try again.
            </p>

            {error && process.env.NODE_ENV === 'development' && (
              <details className="text-left bg-muted p-3 rounded text-xs">
                <summary className="cursor-pointer font-medium mb-2">
                  Error Details
                </summary>
                <pre className="whitespace-pre-wrap break-all">
                  {error.message}\n{error.stack}
                </pre>
              </details>
            )}
          </div>

          <div className="space-y-2">
            <Button onClick={() => window.location.reload()} className="w-full">
              Retry
            </Button>

            <a
              href="mailto:support@mcp-portal.com?subject=Authentication Error"
              className="block text-sm text-primary hover:underline"
            >
              Contact Support
            </a>
          </div>
        </div>
      </div>
    </div>
  );
}

/**
 * Auth Provider Component with Error Boundary
 */
class AuthProviderClass extends React.Component<
  AuthProviderProps,
  AuthProviderState
> {
  private msalInstance: PublicClientApplication | null = null;

  constructor(props: AuthProviderProps) {
    super(props);
    this.state = { hasError: false, isInitialized: false };
  }

  static getDerivedStateFromError(error: Error): AuthProviderState {
    return {
      hasError: true,
      error,
      isInitialized: false,
    };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    authLogger.error('Auth Provider Error', error, { errorInfo });

    // Report to error tracking service in production
    if (process.env.NODE_ENV === 'production') {
      // TODO: Integrate with error tracking service (e.g., Sentry)
    }
  }

  componentDidMount() {
    this.initializeAuth();
  }

  private async initializeAuth() {
    try {
      this.msalInstance = initializeMsal();

      if (!this.msalInstance) {
        throw new Error('Failed to initialize MSAL instance');
      }

      // Initialize MSAL instance
      await this.msalInstance.initialize();

      // Handle redirect promise
      await this.msalInstance.handleRedirectPromise();

      // Mark as initialized and trigger re-render
      this.setState({ hasError: false, isInitialized: true });
    } catch (error) {
      authLogger.error('Auth initialization error', error);
      this.setState({
        hasError: true,
        error: error as Error,
        isInitialized: false,
      });
    }
  }

  render() {
    if (this.state.hasError) {
      return <AuthErrorFallback error={this.state.error} />;
    }

    if (!this.state.isInitialized || !this.msalInstance) {
      // Loading state while MSAL initializes
      return (
        <div className="min-h-screen flex items-center justify-center bg-background">
          <div className="text-center space-y-4">
            <div className="inline-flex items-center justify-center w-8 h-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            </div>
            <p className="text-muted-foreground text-sm">
              Initializing authentication...
            </p>
          </div>
        </div>
      );
    }

    return (
      <MsalProvider instance={this.msalInstance}>
        {this.props.children}
      </MsalProvider>
    );
  }
}

/**
 * Functional Auth Provider Export
 */
export default function AuthProvider({ children }: AuthProviderProps) {
  return <AuthProviderClass>{children}</AuthProviderClass>;
}

/**
 * Export MSAL instance for use in other parts of the application
 */
export function getMsalInstance(): PublicClientApplication | null {
  return msalInstance;
}

/**
 * Hook to check if MSAL is ready
 */
export function useMsalReady(): boolean {
  const [isReady, setIsReady] = useState(false);

  useEffect(() => {
    const checkMsal = () => {
      setIsReady(msalInstance !== null);
    };

    checkMsal();

    // Check periodically until MSAL is ready
    const interval = setInterval(checkMsal, 100);

    return () => clearInterval(interval);
  }, []);

  return isReady;
}

/**
 * Performance monitoring hook for auth operations
 */
export function useAuthPerformanceMonitoring() {
  useEffect(() => {
    const performanceObserver = new PerformanceObserver(list => {
      const entries = list.getEntries();
      entries.forEach(entry => {
        if (entry.name.includes('msal') || entry.name.includes('auth')) {
          authLogger.debug('Auth Performance', {
            operation: entry.name,
            duration: `${entry.duration}ms`,
          });
        }
      });
    });

    try {
      performanceObserver.observe({ entryTypes: ['measure', 'navigation'] });
    } catch {
      // Performance Observer not supported
    }

    return () => {
      try {
        performanceObserver.disconnect();
      } catch {
        // Ignore cleanup errors
      }
    };
  }, []);
}
