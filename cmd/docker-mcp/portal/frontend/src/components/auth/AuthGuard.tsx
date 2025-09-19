'use client';

import { useEffect, useState } from 'react';
import { useIsAuthenticated, useMsal } from '@azure/msal-react';
import { useRouter } from 'next/navigation';
import { InteractionStatus } from '@azure/msal-browser';

interface AuthGuardProps {
  children: React.ReactNode;
  requireAuth?: boolean;
  redirectTo?: string;
}

export function AuthGuard({
  children,
  requireAuth = true,
  redirectTo = '/',
}: AuthGuardProps) {
  const isAuthenticated = useIsAuthenticated();
  const { inProgress } = useMsal();
  const router = useRouter();
  const [isChecking, setIsChecking] = useState(true);

  useEffect(() => {
    // Wait for MSAL to finish initializing
    if (inProgress === InteractionStatus.None) {
      setIsChecking(false);

      // If authentication is required and user is not authenticated
      if (requireAuth && !isAuthenticated) {
        router.replace(redirectTo);
      }
    }
  }, [isAuthenticated, inProgress, requireAuth, redirectTo, router]);

  // Show loading while checking authentication
  if (isChecking || inProgress !== InteractionStatus.None) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center space-y-4">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="text-sm text-muted-foreground">
            Verifying authentication...
          </p>
        </div>
      </div>
    );
  }

  // If auth is required but user is not authenticated, don't render children
  if (requireAuth && !isAuthenticated) {
    return null;
  }

  return <>{children}</>;
}
