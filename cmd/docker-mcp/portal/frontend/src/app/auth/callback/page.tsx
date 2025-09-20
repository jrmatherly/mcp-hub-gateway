'use client';

import { useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useMsal } from '@azure/msal-react';
import { Loader2, AlertCircle, CheckCircle } from 'lucide-react';
import { authLogger } from '@/lib/logger';

interface CallbackState {
  status: 'processing' | 'success' | 'error';
  message?: string;
  error?: string;
}

// OAuth error codes from Azure AD
const OAuthError = {
  USER_CANCELLED: 'user_cancelled',
  ACCESS_DENIED: 'access_denied',
  INVALID_REQUEST: 'invalid_request',
  INVALID_CLIENT: 'invalid_client',
  INVALID_GRANT: 'invalid_grant',
  UNSUPPORTED_GRANT_TYPE: 'unsupported_grant_type',
  CONSENT_REQUIRED: 'consent_required',
  INTERACTION_REQUIRED: 'interaction_required',
  LOGIN_REQUIRED: 'login_required',
} as const;

// User-friendly error messages
const getErrorMessage = (error: string): string => {
  switch (error) {
    case OAuthError.USER_CANCELLED:
      return 'Login was cancelled. Please try again.';
    case OAuthError.ACCESS_DENIED:
      return 'Access was denied. Please contact your administrator.';
    case OAuthError.CONSENT_REQUIRED:
      return 'Additional consent is required. Please contact your administrator.';
    case OAuthError.INTERACTION_REQUIRED:
    case OAuthError.LOGIN_REQUIRED:
      return 'Please sign in again to continue.';
    case OAuthError.INVALID_CLIENT:
      return 'Configuration error. Please contact support.';
    default:
      return `Authentication failed: ${error}`;
  }
};

export default function AuthCallbackPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { instance } = useMsal();

  const [state, setState] = useState<CallbackState>({
    status: 'processing',
  });

  useEffect(() => {
    const handleCallback = async () => {
      try {
        // Get URL parameters
        const code = searchParams.get('code');
        const stateParam = searchParams.get('state');
        const error = searchParams.get('error');
        const errorDescription = searchParams.get('error_description');

        // Handle OAuth errors from Azure AD
        if (error) {
          console.error('OAuth error from Azure AD:', error, errorDescription);
          setState({
            status: 'error',
            error: getErrorMessage(error),
          });

          // Log security audit event
          logAuthEvent('failure', { error, errorDescription });
          return;
        }

        // Validate required parameters
        if (!code || !stateParam) {
          console.error('Missing authorization code or state parameter');
          setState({
            status: 'error',
            error: 'Missing authorization code or state parameter',
          });

          logAuthEvent('failure', { reason: 'missing_parameters' });
          return;
        }

        // Validate state parameter for CSRF protection
        const expectedState = sessionStorage.getItem('oauth_state');
        if (!expectedState) {
          console.warn(
            'No stored state found - possible direct navigation to callback'
          );
          // Allow MSAL to handle this case
        } else if (stateParam !== expectedState) {
          console.error('State parameter mismatch - possible CSRF attack');
          setState({
            status: 'error',
            error: 'Security validation failed. Please try logging in again.',
          });

          logAuthEvent('failure', { reason: 'state_mismatch' });
          sessionStorage.removeItem('oauth_state');
          return;
        }

        // Handle MSAL redirect response
        const response = await instance.handleRedirectPromise();

        if (response) {
          // Successfully handled by MSAL
          authLogger.info('Authentication successful', {
            username: response.account?.username,
          });
          setState({
            status: 'success',
            message: 'Authentication successful! Redirecting to dashboard...',
          });

          // Clean up
          sessionStorage.removeItem('oauth_state');

          // Log success
          logAuthEvent('success', {
            username: response.account?.username,
            tenantId: response.account?.tenantId,
          });

          // Redirect to originally requested page or dashboard
          const returnUrl =
            sessionStorage.getItem('auth_return_url') || '/dashboard';
          sessionStorage.removeItem('auth_return_url');

          setTimeout(() => {
            router.push(returnUrl);
          }, 1500);
        } else {
          // Check if user is already authenticated
          const accounts = instance.getAllAccounts();
          if (accounts.length > 0) {
            authLogger.info('User already authenticated', {
              username: accounts[0].username,
            });
            setState({
              status: 'success',
              message: 'Welcome back! Redirecting to dashboard...',
            });

            // Clean up
            sessionStorage.removeItem('oauth_state');

            // Log success
            logAuthEvent('success', {
              username: accounts[0].username,
              tenantId: accounts[0].tenantId,
            });

            setTimeout(() => {
              router.push('/dashboard');
            }, 1500);
          } else {
            // Authentication failed for unknown reason
            console.error('No MSAL response and no authenticated accounts');
            setState({
              status: 'error',
              error: 'Authentication failed. Please try again.',
            });

            logAuthEvent('failure', { reason: 'no_valid_account' });
          }
        }
      } catch (error) {
        console.error('Callback handling error:', error);
        setState({
          status: 'error',
          error: 'An unexpected error occurred during authentication.',
        });

        logAuthEvent('failure', {
          reason: 'exception',
          error: error instanceof Error ? error.message : 'Unknown error',
        });
      }
    };

    handleCallback();
  }, [instance, router, searchParams]);

  // Auto-redirect to login on error after delay
  useEffect(() => {
    if (state.status === 'error') {
      const timer = setTimeout(() => {
        router.push('/auth/login');
      }, 5000);
      return () => clearTimeout(timer);
    }
  }, [state.status, router]);

  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="max-w-md w-full mx-4">
        <div className="bg-card rounded-lg shadow-lg p-8">
          <div className="text-center space-y-6">
            {/* Processing State */}
            {state.status === 'processing' && (
              <>
                <div className="relative">
                  <Loader2 className="h-16 w-16 animate-spin mx-auto text-primary" />
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className="h-12 w-12 rounded-full bg-primary/10 animate-ping" />
                  </div>
                </div>
                <div>
                  <h1 className="text-2xl font-semibold text-foreground mb-2">
                    Completing Sign-In
                  </h1>
                  <p className="text-muted-foreground">
                    Please wait while we verify your authentication...
                  </p>
                </div>
              </>
            )}

            {/* Success State */}
            {state.status === 'success' && (
              <>
                <div className="relative">
                  <CheckCircle className="h-16 w-16 mx-auto text-green-500" />
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className="h-12 w-12 rounded-full bg-green-500/20 animate-ping" />
                  </div>
                </div>
                <div>
                  <h1 className="text-2xl font-semibold text-foreground mb-2">
                    Sign-In Successful
                  </h1>
                  <p className="text-muted-foreground">{state.message}</p>
                  <div className="mt-4 flex justify-center">
                    <div className="flex space-x-1">
                      <div className="w-2 h-2 bg-primary rounded-full animate-bounce" />
                      <div className="w-2 h-2 bg-primary rounded-full animate-bounce delay-100" />
                      <div className="w-2 h-2 bg-primary rounded-full animate-bounce delay-200" />
                    </div>
                  </div>
                </div>
              </>
            )}

            {/* Error State */}
            {state.status === 'error' && (
              <>
                <div className="relative">
                  <AlertCircle className="h-16 w-16 mx-auto text-destructive" />
                </div>
                <div>
                  <h1 className="text-2xl font-semibold text-foreground mb-2">
                    Sign-In Failed
                  </h1>
                  <p className="text-muted-foreground mb-4">{state.error}</p>
                  <div className="pt-4 border-t border-border">
                    <p className="text-sm text-muted-foreground">
                      You will be redirected to the login page in 5 seconds...
                    </p>
                    <button
                      onClick={() => router.push('/auth/login')}
                      className="mt-4 text-sm text-primary hover:underline focus:outline-none focus:ring-2 focus:ring-primary rounded"
                    >
                      Return to login now
                    </button>
                  </div>
                </div>
              </>
            )}
          </div>
        </div>

        {/* Security Notice */}
        <p className="mt-4 text-center text-xs text-muted-foreground">
          This is a secure authentication page. If you are experiencing issues,
          please contact your system administrator.
        </p>
      </div>
    </div>
  );
}

// Audit logging function
async function logAuthEvent(
  event: 'success' | 'failure',
  details: Record<string, unknown>
) {
  try {
    // Log to console in development
    // Log audit event
    authLogger.debug(`[AUTH AUDIT] ${event.toUpperCase()}`, details);

    // Send audit event to backend
    await fetch('/api/v1/audit/auth', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        event_type: `oauth_callback_${event}`,
        timestamp: new Date().toISOString(),
        details: {
          ...details,
          user_agent: navigator.userAgent,
          referrer: document.referrer,
        },
      }),
    }).catch(error => {
      console.error('Failed to send audit log:', error);
    });
  } catch (error) {
    console.error('Audit logging failed:', error);
  }
}
