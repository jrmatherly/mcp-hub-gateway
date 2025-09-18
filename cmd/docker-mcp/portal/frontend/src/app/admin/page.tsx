'use client';

import {
  Activity,
  AlertTriangle,
  CheckCircle,
  Clock,
  Shield,
  TrendingUp,
  Users,
  Server,
  Database,
  Cpu,
  HardDrive,
} from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { useSystemStats, useSystemHealth } from '@/hooks/api/use-admin';
import { cn } from '@/lib/utils';

interface StatCardProps {
  title: string;
  value: string | number;
  change?: number;
  icon: React.ElementType;
  trend?: 'up' | 'down' | 'neutral';
  description?: string;
}

function StatCard({
  title,
  value,
  change,
  icon: Icon,
  trend,
  description,
}: StatCardProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{value}</div>
        {change !== undefined && (
          <div className="flex items-center space-x-1 text-xs">
            <TrendingUp
              className={cn(
                'h-3 w-3',
                trend === 'up'
                  ? 'text-green-600'
                  : trend === 'down'
                    ? 'text-red-600'
                    : 'text-muted-foreground'
              )}
            />
            <span
              className={cn(
                trend === 'up'
                  ? 'text-green-600'
                  : trend === 'down'
                    ? 'text-red-600'
                    : 'text-muted-foreground'
              )}
            >
              {change > 0 ? '+' : ''}
              {change}%
            </span>
            <span className="text-muted-foreground">from last hour</span>
          </div>
        )}
        {description && (
          <p className="text-xs text-muted-foreground mt-1">{description}</p>
        )}
      </CardContent>
    </Card>
  );
}

interface HealthStatusProps {
  status: 'healthy' | 'warning' | 'critical' | 'unknown';
  label: string;
  description?: string;
}

function HealthStatus({ status, label, description }: HealthStatusProps) {
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy':
        return 'text-green-600 bg-green-50';
      case 'warning':
        return 'text-yellow-600 bg-yellow-50';
      case 'critical':
        return 'text-red-600 bg-red-50';
      default:
        return 'text-gray-600 bg-gray-50';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'healthy':
        return <CheckCircle className="h-3 w-3" />;
      case 'warning':
        return <AlertTriangle className="h-3 w-3" />;
      case 'critical':
        return <AlertTriangle className="h-3 w-3" />;
      default:
        return <Activity className="h-3 w-3" />;
    }
  };

  return (
    <div className="flex items-center justify-between py-2">
      <div className="flex items-center space-x-2">
        <span className="text-sm font-medium">{label}</span>
        {description && (
          <span className="text-xs text-muted-foreground">({description})</span>
        )}
      </div>
      <Badge
        variant="outline"
        className={cn('text-xs', getStatusColor(status))}
      >
        {getStatusIcon(status)}
        <span className="ml-1 capitalize">{status}</span>
      </Badge>
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="space-y-6">
      {/* Stats Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 8 }).map((_, i) => (
          <Card key={i}>
            <CardHeader className="space-y-0 pb-2">
              <Skeleton className="h-4 w-24" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-8 w-16 mb-2" />
              <Skeleton className="h-3 w-32" />
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Health Status */}
      <div className="grid gap-4 md:grid-cols-2">
        {Array.from({ length: 2 }).map((_, i) => (
          <Card key={i}>
            <CardHeader>
              <Skeleton className="h-5 w-32" />
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                {Array.from({ length: 4 }).map((_, j) => (
                  <div key={j} className="flex justify-between">
                    <Skeleton className="h-4 w-20" />
                    <Skeleton className="h-4 w-16" />
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}

export default function AdminDashboard() {
  const {
    data: stats,
    isLoading: statsLoading,
    error: statsError,
  } = useSystemStats({ refetchInterval: 30000 });

  const {
    data: health,
    isLoading: healthLoading,
    error: healthError,
  } = useSystemHealth({ refetchInterval: 10000 });

  if (statsLoading || healthLoading) {
    return <LoadingSkeleton />;
  }

  if (statsError || healthError) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <AlertTriangle className="h-16 w-16 text-amber-500 mx-auto mb-4" />
          <h2 className="text-2xl font-semibold text-foreground mb-2">
            Failed to Load Data
          </h2>
          <p className="text-muted-foreground mb-4">
            {statsError?.message ||
              healthError?.message ||
              'Unknown error occurred'}
          </p>
          <button
            onClick={() => window.location.reload()}
            className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  const formatUptime = (seconds: number) => {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);

    if (days > 0) {
      return `${days}d ${hours}h ${minutes}m`;
    }
    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">System Overview</h1>
          <p className="text-muted-foreground">
            Real-time system metrics and health monitoring
          </p>
        </div>

        <div className="flex items-center space-x-2">
          <Badge variant="outline" className="text-green-600 bg-green-50">
            <Activity className="h-3 w-3 mr-1" />
            Live Data
          </Badge>
          <span className="text-xs text-muted-foreground">
            Last updated: {new Date().toLocaleTimeString()}
          </span>
        </div>
      </div>

      {/* Key Metrics Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Total Users"
          value={stats?.users.total || 0}
          change={stats?.users.lastHour}
          icon={Users}
          trend={
            stats?.users.lastHour && stats.users.lastHour > 0 ? 'up' : 'neutral'
          }
          description={`${stats?.users.active || 0} active`}
        />

        <StatCard
          title="Active Sessions"
          value={stats?.sessions.active || 0}
          icon={Shield}
          description={`Avg duration: ${Math.round((stats?.sessions.avgDuration || 0) / 60)}m`}
        />

        <StatCard
          title="MCP Servers"
          value={`${stats?.servers.running || 0}/${stats?.servers.total || 0}`}
          icon={Server}
          description={`${stats?.servers.enabled || 0} enabled`}
        />

        <StatCard
          title="System Uptime"
          value={formatUptime(stats?.system.uptime || 0)}
          icon={Clock}
          description="Since last restart"
        />

        <StatCard
          title="CPU Usage"
          value={`${stats?.system.cpuUsage?.toFixed(1) || 0}%`}
          icon={Cpu}
          trend={
            stats?.system.cpuUsage
              ? stats.system.cpuUsage > 80
                ? 'down'
                : stats.system.cpuUsage > 60
                  ? 'neutral'
                  : 'up'
              : 'neutral'
          }
        />

        <StatCard
          title="Memory Usage"
          value={`${stats?.system.memoryUsage?.toFixed(1) || 0}%`}
          icon={Database}
          trend={
            stats?.system.memoryUsage
              ? stats.system.memoryUsage > 85
                ? 'down'
                : stats.system.memoryUsage > 70
                  ? 'neutral'
                  : 'up'
              : 'neutral'
          }
        />

        <StatCard
          title="Disk Usage"
          value={`${stats?.system.diskUsage?.toFixed(1) || 0}%`}
          icon={HardDrive}
          trend={
            stats?.system.diskUsage
              ? stats.system.diskUsage > 90
                ? 'down'
                : stats.system.diskUsage > 75
                  ? 'neutral'
                  : 'up'
              : 'neutral'
          }
        />

        <StatCard
          title="Failed Logins"
          value={stats?.security.failedLogins || 0}
          icon={AlertTriangle}
          description={`${stats?.security.blockedIPs || 0} blocked IPs`}
        />
      </div>

      {/* System Health Status */}
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <Activity className="h-5 w-5" />
              <span>Component Health</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-1">
              <HealthStatus
                status={health?.components.database || 'unknown'}
                label="Database"
                description="PostgreSQL"
              />
              <HealthStatus
                status={health?.components.redis || 'unknown'}
                label="Redis Cache"
                description="Session store"
              />
              <HealthStatus
                status={health?.components.docker || 'unknown'}
                label="Docker Engine"
                description="Container runtime"
              />
              <HealthStatus
                status={health?.components.auth || 'unknown'}
                label="Authentication"
                description="Azure AD"
              />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <TrendingUp className="h-5 w-5" />
              <span>Performance Metrics</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">Response Time</span>
                <span className="text-sm text-muted-foreground">
                  {health?.metrics.responseTime?.toFixed(0) || 0}ms
                </span>
              </div>

              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">Error Rate</span>
                <span className="text-sm text-muted-foreground">
                  {health?.metrics.errorRate?.toFixed(2) || 0}%
                </span>
              </div>

              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">Throughput</span>
                <span className="text-sm text-muted-foreground">
                  {health?.metrics.throughput?.toFixed(0) || 0} req/s
                </span>
              </div>

              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">Last Health Check</span>
                <span className="text-sm text-muted-foreground">
                  {health?.lastCheck
                    ? new Date(health.lastCheck).toLocaleTimeString()
                    : 'Never'}
                </span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Quick Actions & Alerts */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Quick Actions</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            <button className="w-full text-left px-3 py-2 text-sm bg-muted hover:bg-muted/80 rounded-md transition-colors">
              View Error Logs
            </button>
            <button className="w-full text-left px-3 py-2 text-sm bg-muted hover:bg-muted/80 rounded-md transition-colors">
              Check Failed Logins
            </button>
            <button className="w-full text-left px-3 py-2 text-sm bg-muted hover:bg-muted/80 rounded-md transition-colors">
              System Backup
            </button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Recent Activity</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Last user login</span>
                <span>2 min ago</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Server enabled</span>
                <span>5 min ago</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Config updated</span>
                <span>12 min ago</span>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-sm">System Alerts</CardTitle>
          </CardHeader>
          <CardContent>
            {stats?.security.activeThreats &&
            stats.security.activeThreats > 0 ? (
              <div className="space-y-2">
                <div className="flex items-center space-x-2 text-red-600">
                  <AlertTriangle className="h-4 w-4" />
                  <span className="text-sm">
                    {stats.security.activeThreats} active threats detected
                  </span>
                </div>
              </div>
            ) : (
              <div className="flex items-center space-x-2 text-green-600">
                <CheckCircle className="h-4 w-4" />
                <span className="text-sm">No security alerts</span>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
