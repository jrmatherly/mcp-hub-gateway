import { Activity, Container, Server, Settings } from 'lucide-react';
import type { Metadata } from 'next';
import { Suspense } from 'react';
import { RecentActivity } from '@/components/dashboard/RecentActivity';
import { QuickActions } from '@/components/dashboard/QuickActions';

export const metadata: Metadata = {
  title: 'Dashboard Overview',
  description: 'Overview of MCP servers, containers, and system status.',
};

// Loading component for cards
function StatsCardLoading() {
  return (
    <div className="card animate-pulse">
      <div className="card-content p-6">
        <div className="flex items-center justify-between">
          <div className="flex-1">
            <div className="h-4 bg-muted rounded w-24 mb-2"></div>
            <div className="h-8 bg-muted rounded w-16"></div>
          </div>
          <div className="h-12 w-12 bg-muted rounded-lg"></div>
        </div>
      </div>
    </div>
  );
}

// Stats card component
interface StatsCardProps {
  title: string;
  value: string | number;
  change?: string;
  icon: React.ComponentType<{ className?: string }>;
  color: string;
}

function StatsCard({
  title,
  value,
  change,
  icon: Icon,
  color,
}: StatsCardProps) {
  return (
    <div className="card hover:shadow-md transition-shadow">
      <div className="card-content p-6">
        <div className="flex items-center justify-between">
          <div className="flex-1">
            <p className="text-sm font-medium text-muted-foreground mb-1">
              {title}
            </p>
            <p className="text-2xl font-bold text-foreground">{value}</p>
            {change && (
              <p className="text-xs text-muted-foreground mt-1">{change}</p>
            )}
          </div>
          <div
            className={`inline-flex h-12 w-12 items-center justify-center rounded-lg ${color}`}
          >
            <Icon className="h-6 w-6 text-white" />
          </div>
        </div>
      </div>
    </div>
  );
}

export default function DashboardPage() {
  return (
    <div className="space-y-8">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-foreground">Dashboard</h1>
          <p className="text-muted-foreground mt-1">
            Monitor and manage your MCP servers
          </p>
        </div>
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <div className="h-2 w-2 bg-success-500 rounded-full animate-pulse"></div>
            <span>System Healthy</span>
          </div>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
        <Suspense fallback={<StatsCardLoading />}>
          <StatsCard
            title="Total Servers"
            value={12}
            change="+2 from last week"
            icon={Server}
            color="bg-docker-500"
          />
        </Suspense>

        <Suspense fallback={<StatsCardLoading />}>
          <StatsCard
            title="Running Containers"
            value={8}
            change="+1 from yesterday"
            icon={Container}
            color="bg-success-500"
          />
        </Suspense>

        <Suspense fallback={<StatsCardLoading />}>
          <StatsCard
            title="Active Configurations"
            value={15}
            change="Updated 3 hours ago"
            icon={Settings}
            color="bg-warning-500"
          />
        </Suspense>

        <Suspense fallback={<StatsCardLoading />}>
          <StatsCard
            title="System Uptime"
            value="99.9%"
            change="Last 30 days"
            icon={Activity}
            color="bg-error-500"
          />
        </Suspense>
      </div>

      {/* Main Content Grid */}
      <div className="grid gap-6 lg:grid-cols-2">
        <Suspense fallback={<div className="card animate-pulse h-64"></div>}>
          <RecentActivity />
        </Suspense>

        <Suspense fallback={<div className="card animate-pulse h-64"></div>}>
          <QuickActions />
        </Suspense>
      </div>
    </div>
  );
}
