'use client';

import {
  LayoutGrid,
  LayoutList,
  Plus,
  RefreshCw,
  Settings,
} from 'lucide-react';
import React, { Suspense, useMemo, useState } from 'react';
import { ServerBulkActions } from '@/components/dashboard/ServerBulkActions';
import { ServerFilters } from '@/components/dashboard/ServerFilters';
import { ServerGrid } from '@/components/dashboard/ServerGrid';
import { ServerList } from '@/components/dashboard/ServerList';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { useGatewayStatus, useRefreshServers, useServers } from '@/hooks/api';
import { cn } from '@/lib/utils';
import type { ServerFilters as IServerFilters } from '@/types/api';

type ViewMode = 'grid' | 'list';

function ServersDashboardContent() {
  // State management
  const [viewMode, setViewMode] = useState<ViewMode>('grid');
  const [filters, setFilters] = useState<IServerFilters>({});
  const [selectedServers, setSelectedServers] = useState<Set<string>>(
    new Set()
  );

  // Data fetching
  const { data: servers = [], isLoading, error } = useServers(filters);
  const { data: gatewayStatus } = useGatewayStatus();
  const refreshServers = useRefreshServers();

  // Derived data
  const availableTags = useMemo(() => {
    const tagSet = new Set<string>();
    servers.forEach(server => {
      server.tags?.forEach(tag => tagSet.add(tag));
    });
    return Array.from(tagSet).sort();
  }, [servers]);

  const serverStats = useMemo(() => {
    const total = servers.length;
    const running = servers.filter(s => s.status === 'running').length;
    const enabled = servers.filter(s => s.enabled).length;
    const healthy = servers.filter(s => s.health_status === 'healthy').length;
    const errors = servers.filter(s => s.status === 'error').length;

    return { total, running, enabled, healthy, errors };
  }, [servers]);

  // Event handlers
  const handleServerSelect = (serverName: string, selected: boolean) => {
    setSelectedServers(prev => {
      const newSet = new Set(prev);
      if (selected) {
        newSet.add(serverName);
      } else {
        newSet.delete(serverName);
      }
      return newSet;
    });
  };

  const handleSelectAll = () => {
    setSelectedServers(new Set(servers.map(s => s.name)));
  };

  const handleClearSelection = () => {
    setSelectedServers(new Set());
  };

  const handleServerInspect = (serverName: string) => {
    // Navigate to server details page
    window.location.href = `/dashboard/servers/${encodeURIComponent(serverName)}`;
  };

  const handleRefresh = () => {
    refreshServers.mutate();
  };

  const handleAddServer = () => {
    // Navigate to add server page or open modal
    window.location.href = '/dashboard/catalog';
  };

  return (
    <div className="flex flex-col space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between space-y-4 sm:space-y-0">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">MCP Servers</h1>
          <p className="text-muted-foreground">
            Manage and monitor your Model Context Protocol servers
          </p>
        </div>

        <div className="flex items-center space-x-2">
          <Button
            variant="outline"
            onClick={handleRefresh}
            disabled={refreshServers.isPending}
            className="flex items-center space-x-2"
          >
            <RefreshCw
              className={cn(
                'h-4 w-4',
                refreshServers.isPending && 'animate-spin'
              )}
            />
            <span>Refresh</span>
          </Button>

          <Button
            onClick={handleAddServer}
            className="flex items-center space-x-2"
          >
            <Plus className="h-4 w-4" />
            <span>Add Server</span>
          </Button>

          <Button variant="outline" size="icon">
            <Settings className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Gateway Status Banner */}
      {gatewayStatus && (
        <Card
          className={cn(
            'border-l-4',
            gatewayStatus.running
              ? 'border-l-green-500 bg-green-50 dark:bg-green-900/20'
              : 'border-l-yellow-500 bg-yellow-50 dark:bg-yellow-900/20'
          )}
        >
          <CardContent className="flex items-center justify-between py-3">
            <div className="flex items-center space-x-3">
              <Badge variant={gatewayStatus.running ? 'success' : 'warning'}>
                Gateway {gatewayStatus.running ? 'Running' : 'Stopped'}
              </Badge>
              {gatewayStatus.running && (
                <div className="text-sm text-muted-foreground">
                  Port {gatewayStatus.port} •{' '}
                  {gatewayStatus.active_servers?.length || 0} active servers
                </div>
              )}
            </div>
            {!gatewayStatus.running && <Button size="sm">Start Gateway</Button>}
          </CardContent>
        </Card>
      )}

      {/* Statistics Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
        <Card>
          <CardContent className="flex items-center justify-between py-4">
            <div>
              <p className="text-sm font-medium text-muted-foreground">Total</p>
              <p className="text-2xl font-bold">{serverStats.total}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="flex items-center justify-between py-4">
            <div>
              <p className="text-sm font-medium text-muted-foreground">
                Running
              </p>
              <p className="text-2xl font-bold text-green-600">
                {serverStats.running}
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="flex items-center justify-between py-4">
            <div>
              <p className="text-sm font-medium text-muted-foreground">
                Enabled
              </p>
              <p className="text-2xl font-bold text-blue-600">
                {serverStats.enabled}
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="flex items-center justify-between py-4">
            <div>
              <p className="text-sm font-medium text-muted-foreground">
                Healthy
              </p>
              <p className="text-2xl font-bold text-emerald-600">
                {serverStats.healthy}
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="flex items-center justify-between py-4">
            <div>
              <p className="text-sm font-medium text-muted-foreground">
                Errors
              </p>
              <p className="text-2xl font-bold text-red-600">
                {serverStats.errors}
              </p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Filters */}
      <ServerFilters
        filters={filters}
        onFiltersChange={setFilters}
        serverCount={servers.length}
        filteredCount={servers.length}
        availableTags={availableTags}
      />

      {/* Bulk Actions */}
      {selectedServers.size > 0 && (
        <ServerBulkActions
          selectedServers={selectedServers}
          onClearSelection={handleClearSelection}
          onSelectAll={handleSelectAll}
          totalServers={servers.length}
        />
      )}

      {/* View Mode Toggle and Actions */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-2">
          <Button
            variant={viewMode === 'grid' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setViewMode('grid')}
            className="flex items-center space-x-1"
          >
            <LayoutGrid className="h-4 w-4" />
            <span>Grid</span>
          </Button>
          <Button
            variant={viewMode === 'list' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setViewMode('list')}
            className="flex items-center space-x-1"
          >
            <LayoutList className="h-4 w-4" />
            <span>List</span>
          </Button>
        </div>

        {servers.length > 0 && (
          <div className="text-sm text-muted-foreground">
            {selectedServers.size > 0 && (
              <span>{selectedServers.size} selected • </span>
            )}
            {servers.length} server{servers.length !== 1 ? 's' : ''}
          </div>
        )}
      </div>

      {/* Error State */}
      {error && (
        <Card className="border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-900/20">
          <CardContent className="py-4">
            <div className="flex items-center space-x-2">
              <div className="text-red-600 dark:text-red-400">
                <h3 className="font-medium">Error loading servers</h3>
                <p className="text-sm mt-1">
                  {error instanceof Error
                    ? error.message
                    : 'An unexpected error occurred'}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Server List/Grid */}
      {viewMode === 'grid' ? (
        <ServerGrid
          servers={servers}
          isLoading={isLoading}
          selectedServers={selectedServers}
          onServerSelect={handleServerSelect}
          onServerInspect={handleServerInspect}
        />
      ) : (
        <ServerList
          servers={servers}
          isLoading={isLoading}
          selectedServers={selectedServers}
          onServerSelect={handleServerSelect}
          onServerInspect={handleServerInspect}
        />
      )}
    </div>
  );
}

export default function ServersDashboardPage() {
  return (
    <Suspense
      fallback={
        <div className="flex items-center justify-center min-h-screen">
          <div className="text-center space-y-4">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto"></div>
            <p className="text-muted-foreground">Loading servers...</p>
          </div>
        </div>
      }
    >
      <ServersDashboardContent />
    </Suspense>
  );
}
