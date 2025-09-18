'use client';

import {
  AlertCircle,
  CheckCircle,
  Clock,
  Container,
  Loader2,
  MoreHorizontal,
  Play,
  Square,
} from 'lucide-react';
import React from 'react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Switch } from '@/components/ui/switch';
import { useServerToggle } from '@/hooks/api';
import { cn } from '@/lib/utils';
import type { Server } from '@/types/api';

interface ServerListProps {
  servers: Server[];
  isLoading?: boolean;
  selectedServers?: Set<string>;
  onServerSelect?: (_serverName: string, _selected: boolean) => void;
  onServerInspect?: (_serverName: string) => void;
  className?: string;
}

const StatusIcon = ({
  status,
  health_status,
}: {
  status: Server['status'];
  health_status?: Server['health_status'];
}) => {
  if (status === 'running') {
    if (health_status === 'healthy') {
      return <CheckCircle className="h-4 w-4 text-green-500" />;
    }
    if (health_status === 'starting') {
      return <Loader2 className="h-4 w-4 text-yellow-500 animate-spin" />;
    }
    if (health_status === 'unhealthy') {
      return <AlertCircle className="h-4 w-4 text-red-500" />;
    }
    return <Play className="h-4 w-4 text-green-500" />;
  }

  if (status === 'error') {
    return <AlertCircle className="h-4 w-4 text-red-500" />;
  }

  return <Square className="h-4 w-4 text-gray-400" />;
};

const getStatusBadgeVariant = (
  status: Server['status'],
  health_status?: Server['health_status']
) => {
  if (status === 'running') {
    if (health_status === 'healthy') return 'success';
    if (health_status === 'starting') return 'warning';
    if (health_status === 'unhealthy') return 'destructive';
    return 'success';
  }

  if (status === 'error') return 'destructive';
  return 'secondary';
};

const formatUptime = (lastStarted?: string) => {
  if (!lastStarted) return null;

  const startTime = new Date(lastStarted);
  const now = new Date();
  const uptimeMs = now.getTime() - startTime.getTime();

  const hours = Math.floor(uptimeMs / (1000 * 60 * 60));
  const minutes = Math.floor((uptimeMs % (1000 * 60 * 60)) / (1000 * 60));

  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  return `${minutes}m`;
};

interface ServerRowProps {
  server: Server;
  isSelected: boolean;
  onSelect?: (_serverName: string, _selected: boolean) => void;
  onInspect?: (_serverName: string) => void;
}

function ServerRow({
  server,
  isSelected,
  onSelect,
  onInspect,
}: ServerRowProps) {
  const serverToggle = useServerToggle();

  const handleToggleEnabled = async (enabled: boolean) => {
    try {
      await serverToggle.mutateAsync({ name: server.name, enabled });
    } catch {
      // Error handling is done in the hook
    }
  };

  const handleInspect = () => {
    onInspect?.(server.name);
  };

  const handleSelectChange = (checked: boolean) => {
    onSelect?.(server.name, checked);
  };

  const statusText =
    server.status === 'running'
      ? server.health_status || 'running'
      : server.status;

  return (
    <tr
      className={cn(
        'border-b transition-colors hover:bg-muted/50',
        isSelected && 'bg-accent'
      )}
    >
      {/* Selection */}
      {onSelect && (
        <td className="px-4 py-3">
          <input
            type="checkbox"
            checked={isSelected}
            onChange={e => handleSelectChange(e.target.checked)}
            className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
          />
        </td>
      )}

      {/* Name & Description */}
      <td className="px-4 py-3">
        <div className="flex flex-col">
          <div className="font-medium text-foreground">{server.name}</div>
          {server.description && (
            <div className="text-sm text-muted-foreground line-clamp-1 max-w-xs">
              {server.description}
            </div>
          )}
        </div>
      </td>

      {/* Status */}
      <td className="px-4 py-3">
        <div className="flex items-center space-x-2">
          <StatusIcon
            status={server.status}
            health_status={server.health_status}
          />
          <Badge
            variant={getStatusBadgeVariant(server.status, server.health_status)}
            className="text-xs"
          >
            {statusText}
          </Badge>
        </div>
      </td>

      {/* Image */}
      <td className="px-4 py-3">
        {server.image && (
          <div className="flex items-center space-x-1 text-sm text-muted-foreground">
            <Container className="h-3 w-3" />
            <span className="truncate max-w-32">{server.image}</span>
          </div>
        )}
      </td>

      {/* Port */}
      <td className="px-4 py-3 text-sm text-muted-foreground">
        {server.port || '-'}
      </td>

      {/* Uptime */}
      <td className="px-4 py-3 text-sm text-muted-foreground">
        {server.last_started && server.status === 'running' ? (
          <div className="flex items-center space-x-1">
            <Clock className="h-3 w-3" />
            <span>{formatUptime(server.last_started)}</span>
          </div>
        ) : (
          '-'
        )}
      </td>

      {/* Resources */}
      <td className="px-4 py-3">
        {server.resources &&
        (server.resources.cpu_usage !== undefined ||
          server.resources.memory_usage !== undefined) ? (
          <div className="text-xs space-y-1">
            {server.resources.cpu_usage !== undefined && (
              <div>CPU: {server.resources.cpu_usage.toFixed(1)}%</div>
            )}
            {server.resources.memory_usage !== undefined && (
              <div>Mem: {server.resources.memory_usage.toFixed(1)}%</div>
            )}
          </div>
        ) : (
          <span className="text-sm text-muted-foreground">-</span>
        )}
      </td>

      {/* Tags */}
      <td className="px-4 py-3">
        {server.tags && server.tags.length > 0 ? (
          <div className="flex flex-wrap gap-1 max-w-32">
            {server.tags.slice(0, 2).map(tag => (
              <Badge key={tag} variant="outline" className="text-xs">
                {tag}
              </Badge>
            ))}
            {server.tags.length > 2 && (
              <Badge variant="outline" className="text-xs">
                +{server.tags.length - 2}
              </Badge>
            )}
          </div>
        ) : (
          <span className="text-sm text-muted-foreground">-</span>
        )}
      </td>

      {/* Enabled Toggle */}
      <td className="px-4 py-3">
        <Switch
          checked={server.enabled}
          onCheckedChange={handleToggleEnabled}
          disabled={serverToggle.isPending}
          aria-label={`${server.enabled ? 'Disable' : 'Enable'} ${server.name}`}
        />
      </td>

      {/* Actions */}
      <td className="px-4 py-3">
        <div className="flex items-center space-x-2">
          <Button
            variant="outline"
            size="sm"
            onClick={handleInspect}
            disabled={!onInspect}
          >
            Inspect
          </Button>
          <Button variant="ghost" size="sm">
            <MoreHorizontal className="h-4 w-4" />
          </Button>
        </div>
      </td>
    </tr>
  );
}

function ServerRowSkeleton({ hasSelection }: { hasSelection: boolean }) {
  return (
    <tr className="border-b">
      {hasSelection && (
        <td className="px-4 py-3">
          <Skeleton className="h-4 w-4" />
        </td>
      )}
      <td className="px-4 py-3">
        <div className="space-y-2">
          <Skeleton className="h-4 w-32" />
          <Skeleton className="h-3 w-48" />
        </div>
      </td>
      <td className="px-4 py-3">
        <div className="flex items-center space-x-2">
          <Skeleton className="h-4 w-4 rounded-full" />
          <Skeleton className="h-5 w-16" />
        </div>
      </td>
      <td className="px-4 py-3">
        <Skeleton className="h-4 w-24" />
      </td>
      <td className="px-4 py-3">
        <Skeleton className="h-4 w-12" />
      </td>
      <td className="px-4 py-3">
        <Skeleton className="h-4 w-16" />
      </td>
      <td className="px-4 py-3">
        <Skeleton className="h-4 w-20" />
      </td>
      <td className="px-4 py-3">
        <Skeleton className="h-4 w-16" />
      </td>
      <td className="px-4 py-3">
        <Skeleton className="h-6 w-11" />
      </td>
      <td className="px-4 py-3">
        <div className="flex space-x-2">
          <Skeleton className="h-8 w-16" />
          <Skeleton className="h-8 w-8" />
        </div>
      </td>
    </tr>
  );
}

export function ServerList({
  servers,
  isLoading = false,
  selectedServers = new Set(),
  onServerSelect,
  onServerInspect,
  className,
}: ServerListProps) {
  const hasSelection = !!onServerSelect;

  if (isLoading) {
    return (
      <div
        className={`bg-background border rounded-lg overflow-hidden ${className || ''}`}
      >
        <table className="w-full">
          <thead className="bg-muted/50">
            <tr className="border-b">
              {hasSelection && <th className="px-4 py-3 text-left w-12"></th>}
              <th className="px-4 py-3 text-left font-medium">Server</th>
              <th className="px-4 py-3 text-left font-medium">Status</th>
              <th className="px-4 py-3 text-left font-medium">Image</th>
              <th className="px-4 py-3 text-left font-medium">Port</th>
              <th className="px-4 py-3 text-left font-medium">Uptime</th>
              <th className="px-4 py-3 text-left font-medium">Resources</th>
              <th className="px-4 py-3 text-left font-medium">Tags</th>
              <th className="px-4 py-3 text-left font-medium">Enabled</th>
              <th className="px-4 py-3 text-left font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {Array.from({ length: 5 }).map((_, index) => (
              <ServerRowSkeleton key={index} hasSelection={hasSelection} />
            ))}
          </tbody>
        </table>
      </div>
    );
  }

  if (servers.length === 0) {
    return (
      <div
        className={`flex flex-col items-center justify-center py-12 ${className || ''}`}
      >
        <div className="text-center space-y-3">
          <div className="mx-auto w-16 h-16 bg-muted rounded-full flex items-center justify-center">
            <svg
              className="w-8 h-8 text-muted-foreground"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0-1.125.504-1.125 1.125V11.25a9 9 0 00-9-9z"
              />
            </svg>
          </div>
          <h3 className="text-lg font-medium text-foreground">
            No servers found
          </h3>
          <p className="text-sm text-muted-foreground max-w-sm">
            No MCP servers match your current filters. Try adjusting your search
            criteria or adding servers from the catalog.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div
      className={`bg-background border rounded-lg overflow-hidden ${className || ''}`}
    >
      <table className="w-full">
        <thead className="bg-muted/50">
          <tr className="border-b">
            {hasSelection && (
              <th className="px-4 py-3 text-left w-12">
                <input
                  type="checkbox"
                  checked={
                    servers.length > 0 &&
                    selectedServers.size === servers.length
                  }
                  onChange={e => {
                    const shouldSelectAll = e.target.checked;
                    servers.forEach(server => {
                      onServerSelect?.(server.name, shouldSelectAll);
                    });
                  }}
                  className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
                />
              </th>
            )}
            <th className="px-4 py-3 text-left font-medium">Server</th>
            <th className="px-4 py-3 text-left font-medium">Status</th>
            <th className="px-4 py-3 text-left font-medium">Image</th>
            <th className="px-4 py-3 text-left font-medium">Port</th>
            <th className="px-4 py-3 text-left font-medium">Uptime</th>
            <th className="px-4 py-3 text-left font-medium">Resources</th>
            <th className="px-4 py-3 text-left font-medium">Tags</th>
            <th className="px-4 py-3 text-left font-medium">Enabled</th>
            <th className="px-4 py-3 text-left font-medium">Actions</th>
          </tr>
        </thead>
        <tbody>
          {servers.map(server => (
            <ServerRow
              key={server.name}
              server={server}
              isSelected={selectedServers.has(server.name)}
              onSelect={onServerSelect}
              onInspect={onServerInspect}
            />
          ))}
        </tbody>
      </table>
    </div>
  );
}
