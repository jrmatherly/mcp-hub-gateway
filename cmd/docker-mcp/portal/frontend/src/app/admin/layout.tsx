'use client';

import { Suspense } from 'react';
import {
  Activity,
  AlertTriangle,
  Cog,
  Shield,
  Users,
  FileText,
  BarChart3,
  Settings,
  BookOpen,
} from 'lucide-react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';

import { cn } from '@/lib/utils';
import { useAuth, usePermissions } from '@/contexts/AuthContext';

interface AdminLayoutProps {
  children: React.ReactNode;
}

// Admin navigation items
const adminNavItems = [
  {
    title: 'Overview',
    href: '/admin',
    icon: BarChart3,
    description: 'System overview and statistics',
    permissions: ['admin'],
  },
  {
    title: 'User Management',
    href: '/admin/users',
    icon: Users,
    description: 'Manage users, roles, and permissions',
    permissions: ['admin', 'super_admin'],
  },
  {
    title: 'Catalog Management',
    href: '/admin/catalogs',
    icon: BookOpen,
    description: 'Manage admin base catalogs and server configurations',
    permissions: ['admin', 'super_admin'],
  },
  {
    title: 'System Health',
    href: '/admin/health',
    icon: Activity,
    description: 'Real-time system monitoring',
    permissions: ['admin', 'super_admin', 'system_admin'],
  },
  {
    title: 'Audit Logs',
    href: '/admin/audit',
    icon: FileText,
    description: 'Security and operation logs',
    permissions: ['admin', 'super_admin'],
  },
  {
    title: 'System Settings',
    href: '/admin/settings',
    icon: Settings,
    description: 'Global system configuration',
    permissions: ['super_admin', 'system_admin'],
  },
  {
    title: 'Security',
    href: '/admin/security',
    icon: Shield,
    description: 'Security monitoring and controls',
    permissions: ['super_admin', 'system_admin'],
  },
];

// Admin loading component
function AdminLoading() {
  return (
    <div className="flex items-center justify-center min-h-[400px]">
      <div className="flex flex-col items-center gap-4">
        <div className="loading-spinner w-8 h-8 border-4"></div>
        <p className="text-sm text-muted-foreground">Loading admin panel...</p>
      </div>
    </div>
  );
}

// Access denied component
function AccessDenied() {
  return (
    <div className="flex items-center justify-center min-h-[400px]">
      <div className="text-center">
        <AlertTriangle className="h-16 w-16 text-amber-500 mx-auto mb-4" />
        <h2 className="text-2xl font-semibold text-foreground mb-2">
          Access Denied
        </h2>
        <p className="text-muted-foreground mb-6 max-w-md">
          You don't have permission to access the admin panel. Contact your
          system administrator if you believe this is an error.
        </p>
        <Link
          href="/dashboard"
          className="inline-flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors"
        >
          Return to Dashboard
        </Link>
      </div>
    </div>
  );
}

export default function AdminLayout({ children }: AdminLayoutProps) {
  const { user, isLoading } = useAuth();
  const { hasAnyRole } = usePermissions();
  const pathname = usePathname();

  // Show loading state
  if (isLoading) {
    return <AdminLoading />;
  }

  // Check if user has admin access
  const hasAdminAccess = hasAnyRole(['admin', 'super_admin', 'system_admin']);

  if (!hasAdminAccess) {
    return <AccessDenied />;
  }

  // Filter nav items based on user permissions
  const filteredNavItems = adminNavItems.filter(item =>
    hasAnyRole(item.permissions)
  );

  return (
    <div className="flex min-h-screen bg-background">
      {/* Admin Sidebar */}
      <aside className="w-64 border-r bg-card shadow-sm">
        <div className="p-6">
          {/* Header */}
          <div className="mb-6">
            <div className="flex items-center gap-2 mb-2">
              <Cog className="h-6 w-6 text-primary" />
              <h2 className="text-xl font-bold text-foreground">Admin Panel</h2>
            </div>
            <p className="text-sm text-muted-foreground">
              System Administration
            </p>
          </div>

          {/* User Info */}
          <div className="mb-6 p-3 bg-muted/50 rounded-lg">
            <div className="flex items-center gap-3">
              <div className="h-8 w-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-medium">
                {user?.name?.charAt(0) || user?.email?.charAt(0) || 'A'}
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-foreground truncate">
                  {user?.name || user?.email}
                </p>
                <p className="text-xs text-muted-foreground">
                  {user?.roles?.[0] || 'admin'}
                </p>
              </div>
            </div>
          </div>

          {/* Navigation */}
          <nav className="space-y-1">
            {filteredNavItems.map(item => {
              const isActive = pathname === item.href;
              const Icon = item.icon;

              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={cn(
                    'flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-primary text-primary-foreground'
                      : 'text-muted-foreground hover:text-foreground hover:bg-muted'
                  )}
                  title={item.description}
                >
                  <Icon className="h-4 w-4" />
                  <span className="truncate">{item.title}</span>
                </Link>
              );
            })}
          </nav>

          {/* Quick Actions */}
          <div className="mt-8 pt-6 border-t">
            <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-3">
              Quick Actions
            </h3>
            <div className="space-y-1">
              <Link
                href="/dashboard"
                className="flex items-center gap-2 px-2 py-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                ← Return to Dashboard
              </Link>
              <Link
                href="/admin/logs?level=error"
                className="flex items-center gap-2 px-2 py-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                View Error Logs
              </Link>
              <Link
                href="/admin/health"
                className="flex items-center gap-2 px-2 py-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                System Status
              </Link>
            </div>
          </div>
        </div>
      </aside>

      {/* Main Content Area */}
      <main className="flex-1">
        {/* Header */}
        <header className="border-b bg-card px-6 py-4">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold text-foreground">
                Administration
              </h1>
              <p className="text-sm text-muted-foreground">
                Manage system settings, users, and monitor health
              </p>
            </div>

            {/* System Status Indicator */}
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <div className="h-2 w-2 rounded-full bg-green-500"></div>
                <span className="text-sm text-muted-foreground">
                  System Online
                </span>
              </div>

              {/* Emergency Actions */}
              {hasAnyRole(['super_admin', 'system_admin']) && (
                <div className="flex items-center gap-2">
                  <button
                    className="px-3 py-1 text-xs bg-amber-100 text-amber-800 rounded-md hover:bg-amber-200 transition-colors"
                    title="Enable Maintenance Mode"
                  >
                    Maintenance
                  </button>
                  <button
                    className="px-3 py-1 text-xs bg-red-100 text-red-800 rounded-md hover:bg-red-200 transition-colors"
                    title="Emergency Actions"
                  >
                    Emergency
                  </button>
                </div>
              )}
            </div>
          </div>
        </header>

        {/* Page Content */}
        <div className="p-6">
          <Suspense fallback={<AdminLoading />}>{children}</Suspense>
        </div>
      </main>
    </div>
  );
}
