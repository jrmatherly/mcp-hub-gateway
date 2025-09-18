'use client';

import React from 'react';
import { Skeleton } from '@/components/ui/skeleton';
import type { Server } from '@/types/api';
import { ServerCard } from './ServerCard';

interface ServerGridProps {
  servers: Server[];
  isLoading?: boolean;
  selectedServers?: Set<string>;
  onServerSelect?: (_serverName: string, _selected: boolean) => void;
  onServerInspect?: (_serverName: string) => void;
  className?: string;
}

function ServerCardSkeleton() {
  return (
    <div className="rounded-lg border bg-card p-6 space-y-4">
      <div className="flex items-start justify-between">
        <div className="space-y-2 flex-1">
          <Skeleton className="h-6 w-32" />
          <Skeleton className="h-4 w-48" />
        </div>
        <div className="flex items-center space-x-2">
          <Skeleton className="h-4 w-4 rounded-full" />
          <Skeleton className="h-6 w-16 rounded-full" />
        </div>
      </div>

      <div className="space-y-3">
        <div className="grid grid-cols-2 gap-3">
          <Skeleton className="h-4 w-24" />
          <Skeleton className="h-4 w-16" />
          <Skeleton className="h-4 w-20" />
          <Skeleton className="h-4 w-12" />
        </div>

        <div className="flex flex-wrap gap-1">
          <Skeleton className="h-5 w-12 rounded-full" />
          <Skeleton className="h-5 w-16 rounded-full" />
          <Skeleton className="h-5 w-10 rounded-full" />
        </div>
      </div>

      <div className="flex items-center justify-between pt-2">
        <div className="flex items-center space-x-2">
          <Skeleton className="h-6 w-11 rounded-full" />
          <Skeleton className="h-4 w-16" />
        </div>
        <Skeleton className="h-9 w-16" />
      </div>
    </div>
  );
}

export function ServerGrid({
  servers,
  isLoading = false,
  selectedServers = new Set(),
  onServerSelect,
  onServerInspect,
  className,
}: ServerGridProps) {
  if (isLoading) {
    return (
      <div
        className={`grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6 ${className || ''}`}
      >
        {Array.from({ length: 8 }).map((_, index) => (
          <ServerCardSkeleton key={index} />
        ))}
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
              <title>Server not found</title>
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
      className={`grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6 ${className || ''}`}
    >
      {servers.map(server => (
        <ServerCard
          key={server.name}
          server={server}
          isSelected={selectedServers.has(server.name)}
          onSelect={onServerSelect}
          onInspect={onServerInspect}
        />
      ))}
    </div>
  );
}
