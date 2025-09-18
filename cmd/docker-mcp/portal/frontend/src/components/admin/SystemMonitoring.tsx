'use client';

import { useState, useEffect } from 'react';
import {
  Activity,
  AlertTriangle,
  CheckCircle,
  Cpu,
  Database,
  HardDrive,
  Network,
  RefreshCw,
  Server,
  Shield,
  Wifi,
  Zap,
} from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { useSystemHealth, useSystemStats } from '@/hooks/api/use-admin';
import { cn } from '@/lib/utils';

interface MetricCardProps {
  title: string;
  value: string | number;
  unit?: string;
  status: 'healthy' | 'warning' | 'critical';
  icon: React.ElementType;
  description?: string;
  trend?: {
    value: number;
    direction: 'up' | 'down' | 'neutral';
  };
}

function MetricCard({
  title,
  value,
  unit,
  status,
  icon: Icon,
  description,
  trend,
}: MetricCardProps) {
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy':
        return 'text-green-600 bg-green-50 border-green-200';
      case 'warning':
        return 'text-yellow-600 bg-yellow-50 border-yellow-200';
      case 'critical':
        return 'text-red-600 bg-red-50 border-red-200';
      default:
        return 'text-gray-600 bg-gray-50 border-gray-200';
    }
  };

  const getTrendColor = (direction: string) => {
    switch (direction) {
      case 'up':
        return 'text-green-600';
      case 'down':
        return 'text-red-600';
      default:
        return 'text-gray-600';
    }
  };

  return (
    <Card className={cn('border-2', getStatusColor(status))}>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
        <Icon className={cn('h-4 w-4', getStatusColor(status).split(' ')[0])} />
      </CardHeader>
      <CardContent>
        <div className="flex items-baseline space-x-1">
          <div className="text-2xl font-bold">
            {typeof value === 'number' ? value.toFixed(1) : value}
          </div>
          {unit && (
            <span className="text-sm text-muted-foreground">{unit}</span>
          )}
        </div>

        {trend && (
          <div className="flex items-center space-x-1 text-xs mt-1">
            <span className={getTrendColor(trend.direction)}>
              {trend.direction === 'up'
                ? '↑'
                : trend.direction === 'down'
                  ? '↓'
                  : '→'}
              {Math.abs(trend.value)}%
            </span>
            <span className="text-muted-foreground">from last check</span>
          </div>
        )}

        {description && (
          <p className="text-xs text-muted-foreground mt-1">{description}</p>
        )}

        <div className="mt-2">
          <Badge
            variant="outline"
            className={cn('text-xs', getStatusColor(status))}
          >
            {status === 'healthy' && <CheckCircle className="h-3 w-3 mr-1" />}
            {status === 'warning' && <AlertTriangle className="h-3 w-3 mr-1" />}
            {status === 'critical' && (
              <AlertTriangle className="h-3 w-3 mr-1" />
            )}
            <span className="capitalize">{status}</span>
          </Badge>
        </div>
      </CardContent>
    </Card>
  );
}

interface ComponentStatusProps {
  name: string;
  status: 'healthy' | 'warning' | 'critical' | 'unknown';
  description: string;
  lastCheck?: Date;
  responseTime?: number;
}

function ComponentStatus({
  name,
  status,
  description,
  lastCheck,
  responseTime,
}: ComponentStatusProps) {
  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'healthy':
        return <CheckCircle className="h-4 w-4 text-green-600" />;
      case 'warning':
        return <AlertTriangle className="h-4 w-4 text-yellow-600" />;
      case 'critical':
        return <AlertTriangle className="h-4 w-4 text-red-600" />;
      default:
        return <Activity className="h-4 w-4 text-gray-600" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy':
        return 'bg-green-50 border-green-200';
      case 'warning':
        return 'bg-yellow-50 border-yellow-200';
      case 'critical':
        return 'bg-red-50 border-red-200';
      default:
        return 'bg-gray-50 border-gray-200';
    }
  };

  return (
    <div className={cn('p-4 rounded-lg border', getStatusColor(status))}>
      <div className="flex items-start justify-between">
        <div className="flex items-center space-x-2">
          {getStatusIcon(status)}
          <div>
            <h4 className="font-medium">{name}</h4>
            <p className="text-sm text-muted-foreground">{description}</p>
          </div>
        </div>

        <div className="text-right text-xs text-muted-foreground">
          {responseTime && <div>{responseTime}ms</div>}
          {lastCheck && <div>{lastCheck.toLocaleTimeString()}</div>}
        </div>
      </div>
    </div>
  );
}

interface RealTimeMetricProps {
  title: string;
  value: number;
  max?: number;
  unit: string;
  icon: React.ElementType;
  className?: string;
}

function RealTimeMetric({
  title,
  value,
  max = 100,
  unit,
  icon: Icon,
  className,
}: RealTimeMetricProps) {
  const percentage = max > 0 ? (value / max) * 100 : 0;
  const getBarColor = (percentage: number) => {
    if (percentage > 85) return 'bg-red-500';
    if (percentage > 70) return 'bg-yellow-500';
    return 'bg-green-500';
  };

  return (
    <Card className={className}>
      <CardContent className="pt-6">
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center space-x-2">
            <Icon className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm font-medium">{title}</span>
          </div>
          <div className="text-sm font-mono">
            {value.toFixed(1)}
            {unit}
          </div>
        </div>

        <div className="w-full bg-gray-200 rounded-full h-2 mb-2">
          <div
            className={cn(
              'h-2 rounded-full transition-all duration-500',
              getBarColor(percentage)
            )}
            style={{ width: `${Math.min(percentage, 100)}%` }}
          />
        </div>

        <div className="flex justify-between text-xs text-muted-foreground">
          <span>0{unit}</span>
          <span>
            {max}
            {unit}
          </span>
        </div>
      </CardContent>
    </Card>
  );
}

function LoadingSkeleton() {
  return (
    <div className="space-y-6">
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Card key={i}>
            <CardHeader>
              <Skeleton className="h-4 w-24" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-8 w-16 mb-2" />
              <Skeleton className="h-4 w-20" />
            </CardContent>
          </Card>
        ))}
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        {Array.from({ length: 2 }).map((_, i) => (
          <Card key={i}>
            <CardHeader>
              <Skeleton className="h-5 w-32" />
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {Array.from({ length: 4 }).map((_, j) => (
                  <Skeleton key={j} className="h-16 w-full" />
                ))}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}

export default function SystemMonitoring() {
  const [refreshInterval, setRefreshInterval] = useState(10000); // 10 seconds
  const [isAutoRefresh, setIsAutoRefresh] = useState(true);

  const {
    data: health,
    isLoading: healthLoading,
    error: healthError,
    refetch: refetchHealth,
  } = useSystemHealth({
    refetchInterval: isAutoRefresh ? refreshInterval : undefined,
  });

  const {
    data: stats,
    isLoading: statsLoading,
    error: statsError,
    refetch: refetchStats,
  } = useSystemStats({
    refetchInterval: isAutoRefresh ? refreshInterval : undefined,
  });

  useEffect(() => {
    if (!isAutoRefresh) {
      // If auto-refresh is disabled, we can set up manual refresh
      return;
    }
  }, [isAutoRefresh, refreshInterval]);

  const handleManualRefresh = () => {
    refetchHealth();
    refetchStats();
  };

  const handleIntervalChange = (interval: number) => {
    setRefreshInterval(interval);
  };

  if (healthLoading || statsLoading) {
    return <LoadingSkeleton />;
  }

  if (healthError || statsError) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <AlertTriangle className="h-16 w-16 text-amber-500 mx-auto mb-4" />
          <h2 className="text-2xl font-semibold text-foreground mb-2">
            Monitoring Error
          </h2>
          <p className="text-muted-foreground mb-4">
            {healthError?.message ||
              statsError?.message ||
              'Failed to load monitoring data'}
          </p>
          <Button onClick={handleManualRefresh}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Retry
          </Button>
        </div>
      </div>
    );
  }

  const getSystemStatus = (): 'healthy' | 'warning' | 'critical' => {
    if (!health || !stats) return 'critical';

    const criticalComponents = Object.values(health.components).filter(
      status => status === 'critical'
    ).length;

    const warningComponents = Object.values(health.components).filter(
      status => status === 'warning'
    ).length;

    if (criticalComponents > 0) return 'critical';
    if (warningComponents > 0) return 'warning';
    return 'healthy';
  };

  return (
    <div className="space-y-6">
      {/* Header with controls */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">
            System Monitoring
          </h2>
          <p className="text-muted-foreground">
            Real-time system health and performance metrics
          </p>
        </div>

        <div className="flex items-center space-x-4">
          <div className="flex items-center space-x-2">
            <span className="text-sm">Auto-refresh:</span>
            <Button
              variant={isAutoRefresh ? 'default' : 'outline'}
              size="sm"
              onClick={() => setIsAutoRefresh(!isAutoRefresh)}
            >
              {isAutoRefresh ? 'ON' : 'OFF'}
            </Button>
          </div>

          <select
            value={refreshInterval}
            onChange={e => handleIntervalChange(Number(e.target.value))}
            className="text-sm border rounded px-2 py-1"
            disabled={!isAutoRefresh}
          >
            <option value={5000}>5s</option>
            <option value={10000}>10s</option>
            <option value={30000}>30s</option>
            <option value={60000}>60s</option>
          </select>

          <Button variant="outline" onClick={handleManualRefresh}>
            <RefreshCw className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* System Overview */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="System Status"
          value={getSystemStatus()}
          status={getSystemStatus()}
          icon={Activity}
          description="Overall system health"
        />

        <MetricCard
          title="Uptime"
          value={Math.floor((stats?.system.uptime || 0) / 86400)}
          unit="days"
          status="healthy"
          icon={Server}
          description="System availability"
        />

        <MetricCard
          title="Response Time"
          value={health?.metrics.responseTime ?? 0}
          unit="ms"
          status={
            health?.metrics.responseTime !== undefined
              ? health.metrics.responseTime > 1000
                ? 'critical'
                : health.metrics.responseTime > 500
                  ? 'warning'
                  : 'healthy'
              : 'healthy'
          }
          icon={Zap}
          description="Average API response time"
        />

        <MetricCard
          title="Error Rate"
          value={health?.metrics.errorRate ?? 0}
          unit="%"
          status={
            health?.metrics.errorRate !== undefined
              ? health.metrics.errorRate > 5
                ? 'critical'
                : health.metrics.errorRate > 1
                  ? 'warning'
                  : 'healthy'
              : 'healthy'
          }
          icon={AlertTriangle}
          description="Request error percentage"
        />
      </div>

      {/* Resource Usage */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <RealTimeMetric
          title="CPU Usage"
          value={stats?.system.cpuUsage || 0}
          unit="%"
          icon={Cpu}
        />

        <RealTimeMetric
          title="Memory Usage"
          value={stats?.system.memoryUsage || 0}
          unit="%"
          icon={Database}
        />

        <RealTimeMetric
          title="Disk Usage"
          value={stats?.system.diskUsage || 0}
          unit="%"
          icon={HardDrive}
        />

        <RealTimeMetric
          title="Network I/O"
          value={45} // Placeholder - would come from real metrics
          unit="%"
          icon={Network}
        />
      </div>

      {/* Component Health */}
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <CheckCircle className="h-5 w-5" />
              <span>Service Components</span>
            </CardTitle>
            <CardDescription>
              Health status of core system components
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <ComponentStatus
                name="Database"
                status={health?.components.database || 'unknown'}
                description="PostgreSQL with Row-Level Security"
                lastCheck={
                  health?.lastCheck ? new Date(health.lastCheck) : undefined
                }
                responseTime={12}
              />

              <ComponentStatus
                name="Redis Cache"
                status={health?.components.redis || 'unknown'}
                description="Session store and caching layer"
                lastCheck={
                  health?.lastCheck ? new Date(health.lastCheck) : undefined
                }
                responseTime={3}
              />

              <ComponentStatus
                name="Docker Engine"
                status={health?.components.docker || 'unknown'}
                description="Container runtime and orchestration"
                lastCheck={
                  health?.lastCheck ? new Date(health.lastCheck) : undefined
                }
                responseTime={25}
              />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <Shield className="h-5 w-5" />
              <span>Security & Authentication</span>
            </CardTitle>
            <CardDescription>
              Authentication and security service status
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <ComponentStatus
                name="Azure AD Authentication"
                status={health?.components.auth || 'unknown'}
                description="OAuth2 and JWT validation"
                lastCheck={
                  health?.lastCheck ? new Date(health.lastCheck) : undefined
                }
                responseTime={150}
              />

              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <div className="flex items-center space-x-2 mb-2">
                  <Wifi className="h-4 w-4 text-blue-600" />
                  <span className="font-medium">Security Metrics</span>
                </div>
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <span className="text-muted-foreground">
                      Failed Logins:
                    </span>
                    <span className="ml-2 font-medium">
                      {stats?.security.failedLogins || 0}
                    </span>
                  </div>
                  <div>
                    <span className="text-muted-foreground">Blocked IPs:</span>
                    <span className="ml-2 font-medium">
                      {stats?.security.blockedIPs || 0}
                    </span>
                  </div>
                  <div>
                    <span className="text-muted-foreground">
                      Active Sessions:
                    </span>
                    <span className="ml-2 font-medium">
                      {stats?.sessions.active || 0}
                    </span>
                  </div>
                  <div>
                    <span className="text-muted-foreground">
                      Active Threats:
                    </span>
                    <span className="ml-2 font-medium text-red-600">
                      {stats?.security.activeThreats || 0}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Performance Trends */}
      <Card>
        <CardHeader>
          <CardTitle>Performance Trends</CardTitle>
          <CardDescription>
            System performance over time (last 24 hours)
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="h-32 flex items-center justify-center text-muted-foreground">
            <div className="text-center">
              <Activity className="h-8 w-8 mx-auto mb-2" />
              <p>Performance charts will be implemented with real-time data</p>
              <p className="text-xs">Requires time-series data collection</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
