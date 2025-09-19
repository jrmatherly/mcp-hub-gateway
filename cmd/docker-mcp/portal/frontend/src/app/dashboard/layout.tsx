import type { Metadata } from 'next';
import { Suspense } from 'react';
import { UserDropdown } from '@/components/dashboard/UserDropdown';
import { AuthGuard } from '@/components/auth/AuthGuard';

export const metadata: Metadata = {
  title: 'Dashboard',
  description: 'MCP Portal dashboard for managing servers and configurations.',
};

interface DashboardLayoutProps {
  children: React.ReactNode;
}

// Dashboard loading component
function DashboardLoading() {
  return (
    <div className="flex items-center justify-center min-h-[400px]">
      <div className="flex flex-col items-center gap-4">
        <div className="loading-spinner w-8 h-8 border-4"></div>
        <p className="text-sm text-muted-foreground">Loading dashboard...</p>
      </div>
    </div>
  );
}

export default function DashboardLayout({ children }: DashboardLayoutProps) {
  return (
    <AuthGuard requireAuth={true}>
      <div className="flex min-h-screen bg-background">
        {/* Sidebar placeholder - will be implemented with navigation */}
        <aside className="hidden lg:block w-64 border-r bg-card">
          <div className="p-6">
            <h2 className="text-lg font-semibold text-foreground mb-4">
              Navigation
            </h2>
            <nav className="space-y-2">
              <div className="text-sm text-muted-foreground">
                Navigation components will be implemented here
              </div>
            </nav>
          </div>
        </aside>

        {/* Main content area */}
        <main className="flex-1">
          {/* Header placeholder */}
          <header className="border-b bg-card px-6 py-4">
            <div className="flex items-center justify-between">
              <div>
                <h1 className="text-2xl font-bold text-foreground">
                  MCP Portal Dashboard
                </h1>
                <p className="text-sm text-muted-foreground">
                  Manage your Model Context Protocol servers
                </p>
              </div>
              <div className="flex items-center gap-4">
                <UserDropdown />
              </div>
            </div>
          </header>

          {/* Page content */}
          <div className="p-6">
            <Suspense fallback={<DashboardLoading />}>{children}</Suspense>
          </div>
        </main>
      </div>
    </AuthGuard>
  );
}
