'use client';

import {
  AlertCircle,
  CheckCircle,
  Clock,
  Container,
  Loader2,
  Play,
  Square,
} from 'lucide-react';
import type React from 'react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Switch } from '@/components/ui/switch';
import { useServerToggle } from '@/hooks/api';
import { cn } from '@/lib/utils';
import type { Server } from '@/types/api';

interface ServerCardProps {
  server: Server;
  onInspect?: (_serverName: string) => void;
  onSelect?: (_serverName: string, _selected: boolean) => void;
  isSelected?: boolean;
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

export function ServerCard({
  server,
  onInspect,
  onSelect,
  isSelected = false,
  className,
}: ServerCardProps) {
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

  const handleSelect = (event: React.MouseEvent) => {
    if (
      event.target === event.currentTarget ||
      (event.target as HTMLElement).closest('[data-card-selectable]')
    ) {
      onSelect?.(server.name, !isSelected);
    }
  };

  const statusText =
    server.status === 'running'
      ? server.health_status || 'running'
      : server.status;

  return (
    <Card
      className={cn(
        'relative transition-all duration-200 hover:shadow-md',
        isSelected && 'ring-2 ring-primary ring-offset-2',
        onSelect && 'cursor-pointer',
        className
      )}
      onClick={onSelect ? handleSelect : undefined}
      data-card-selectable
    >
      {/* Selection checkbox overlay */}
      {onSelect && (
        <div className="absolute top-3 left-3 z-10">
          <input
            type="checkbox"
            checked={isSelected}
            onChange={() => {}} // Handled by card click
            className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
          />
        </div>
      )}

      <CardHeader className={cn('pb-3', onSelect && 'pl-10')}>
        <div className="flex items-start justify-between">
          <div className="flex-1 min-w-0">
            <CardTitle className="text-lg font-semibold truncate">
              {server.name}
            </CardTitle>
            {server.description && (
              <CardDescription className="mt-1 line-clamp-2">
                {server.description}
              </CardDescription>
            )}
          </div>

          <div className="flex items-center space-x-2 ml-3">
            <StatusIcon
              status={server.status}
              health_status={server.health_status}
            />
            <Badge
              variant={getStatusBadgeVariant(
                server.status,
                server.health_status
              )}
            >
              {statusText}
            </Badge>
          </div>
        </div>
      </CardHeader>

      <CardContent className="pb-3">
        <div className="space-y-3">
          {/* Server metadata */}
          <div className="grid grid-cols-2 gap-3 text-sm">
            {server.image && (
              <div className="flex items-center space-x-1 text-muted-foreground">
                <Container className="h-3 w-3" />
                <span className="truncate">{server.image}</span>
              </div>
            )}

            {server.port && (
              <div className="text-muted-foreground">Port: {server.port}</div>
            )}

            {server.last_started && server.status === 'running' && (
              <div className="flex items-center space-x-1 text-muted-foreground">
                <Clock className="h-3 w-3" />
                <span>Up {formatUptime(server.last_started)}</span>
              </div>
            )}

            {server.version && (
              <div className="text-muted-foreground">v{server.version}</div>
            )}
          </div>

          {/* Resource usage */}
          {server.resources &&
            (server.resources.cpu_usage || server.resources.memory_usage) && (
              <div className="space-y-1">
                <div className="text-xs text-muted-foreground">
                  Resource Usage
                </div>
                <div className="grid grid-cols-2 gap-2 text-xs">
                  {server.resources.cpu_usage !== undefined && (
                    <div>CPU: {server.resources.cpu_usage.toFixed(1)}%</div>
                  )}
                  {server.resources.memory_usage !== undefined && (
                    <div>
                      Memory: {server.resources.memory_usage.toFixed(1)}%
                    </div>
                  )}
                </div>
              </div>
            )}

          {/* Tags */}
          {server.tags && server.tags.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {server.tags.slice(0, 3).map(tag => (
                <Badge key={tag} variant="outline" className="text-xs">
                  {tag}
                </Badge>
              ))}
              {server.tags.length > 3 && (
                <Badge variant="outline" className="text-xs">
                  +{server.tags.length - 3}
                </Badge>
              )}
            </div>
          )}

          {/* Error message */}
          {server.error_message && (
            <div className="p-2 bg-red-50 dark:bg-red-900/20 rounded-md">
              <p className="text-xs text-red-600 dark:text-red-400 line-clamp-2">
                {server.error_message}
              </p>
            </div>
          )}
        </div>
      </CardContent>

      <CardFooter className="pt-0 pb-4">
        <div className="flex items-center justify-between w-full">
          <div className="flex items-center space-x-2">
            <Switch
              checked={server.enabled}
              onCheckedChange={handleToggleEnabled}
              disabled={serverToggle.isPending}
              aria-label={`${server.enabled ? 'Disable' : 'Enable'} ${server.name}`}
            />
            <span className="text-sm text-muted-foreground">
              {server.enabled ? 'Enabled' : 'Disabled'}
            </span>
          </div>

          <Button
            variant="outline"
            size="sm"
            onClick={handleInspect}
            disabled={!onInspect}
          >
            Inspect
          </Button>
        </div>
      </CardFooter>
    </Card>
  );
}
