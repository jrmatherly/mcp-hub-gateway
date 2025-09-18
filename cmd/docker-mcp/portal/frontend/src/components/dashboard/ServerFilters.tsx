'use client';

import { Filter, Search, X } from 'lucide-react';
import React from 'react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { Server, ServerFilters as IServerFilters } from '@/types/api';

interface ServerFiltersProps {
  filters: IServerFilters;
  onFiltersChange: (_filters: IServerFilters) => void;
  serverCount?: number;
  filteredCount?: number;
  availableTags?: string[];
  className?: string;
}

const STATUS_OPTIONS: Array<{ value: Server['status']; label: string }> = [
  { value: 'running', label: 'Running' },
  { value: 'stopped', label: 'Stopped' },
  { value: 'error', label: 'Error' },
  { value: 'unknown', label: 'Unknown' },
];

const HEALTH_STATUS_OPTIONS: Array<{
  value: Server['health_status'];
  label: string;
}> = [
  { value: 'healthy', label: 'Healthy' },
  { value: 'unhealthy', label: 'Unhealthy' },
  { value: 'starting', label: 'Starting' },
];

export function ServerFilters({
  filters,
  onFiltersChange,
  serverCount = 0,
  filteredCount = 0,
  availableTags = [],
  className,
}: ServerFiltersProps) {
  const handleSearchChange = (search: string) => {
    onFiltersChange({ ...filters, search: search || undefined });
  };

  const handleStatusToggle = (status: Server['status']) => {
    const currentStatuses = filters.status || [];
    const newStatuses = currentStatuses.includes(status)
      ? currentStatuses.filter(s => s !== status)
      : [...currentStatuses, status];

    onFiltersChange({
      ...filters,
      status: newStatuses.length > 0 ? newStatuses : undefined,
    });
  };

  const handleHealthStatusToggle = (healthStatus: Server['health_status']) => {
    if (!healthStatus) return;

    const currentStatuses = filters.health_status || [];
    const newStatuses = currentStatuses.includes(healthStatus)
      ? currentStatuses.filter(s => s !== healthStatus)
      : [...currentStatuses, healthStatus];

    onFiltersChange({
      ...filters,
      health_status: newStatuses.length > 0 ? newStatuses : undefined,
    });
  };

  const handleTagToggle = (tag: string) => {
    const currentTags = filters.tags || [];
    const newTags = currentTags.includes(tag)
      ? currentTags.filter(t => t !== tag)
      : [...currentTags, tag];

    onFiltersChange({
      ...filters,
      tags: newTags.length > 0 ? newTags : undefined,
    });
  };

  const handleEnabledChange = (value: string) => {
    const enabled =
      value === 'enabled' ? true : value === 'disabled' ? false : undefined;
    onFiltersChange({ ...filters, enabled });
  };

  const clearAllFilters = () => {
    onFiltersChange({});
  };

  const hasActiveFilters = Boolean(
    filters.search ||
      filters.status?.length ||
      filters.health_status?.length ||
      filters.tags?.length ||
      filters.enabled !== undefined
  );

  const activeFilterCount = [
    filters.search ? 1 : 0,
    filters.status?.length || 0,
    filters.health_status?.length || 0,
    filters.tags?.length || 0,
    filters.enabled !== undefined ? 1 : 0,
  ].reduce((sum, count) => sum + count, 0);

  return (
    <div className={`space-y-4 ${className || ''}`}>
      {/* Search and primary filters */}
      <div className="flex flex-col sm:flex-row gap-4">
        {/* Search */}
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search servers..."
            value={filters.search || ''}
            onChange={e => handleSearchChange(e.target.value)}
            className="pl-10"
          />
        </div>

        {/* Enabled/Disabled filter */}
        <Select
          value={
            filters.enabled === undefined
              ? 'all'
              : filters.enabled
                ? 'enabled'
                : 'disabled'
          }
          onValueChange={handleEnabledChange}
        >
          <SelectTrigger className="w-full sm:w-48">
            <SelectValue placeholder="All servers" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All servers</SelectItem>
            <SelectItem value="enabled">Enabled only</SelectItem>
            <SelectItem value="disabled">Disabled only</SelectItem>
          </SelectContent>
        </Select>

        {/* Clear filters */}
        {hasActiveFilters && (
          <Button
            variant="outline"
            onClick={clearAllFilters}
            className="flex items-center space-x-2"
          >
            <X className="h-4 w-4" />
            <span>Clear</span>
            {activeFilterCount > 0 && (
              <Badge variant="secondary" className="ml-1">
                {activeFilterCount}
              </Badge>
            )}
          </Button>
        )}
      </div>

      {/* Advanced filters */}
      <div className="flex flex-wrap gap-3">
        {/* Status filters */}
        <div className="flex items-center space-x-2">
          <span className="text-sm font-medium text-muted-foreground">
            Status:
          </span>
          <div className="flex flex-wrap gap-1">
            {STATUS_OPTIONS.map(({ value, label }) => (
              <Badge
                key={value}
                variant={
                  filters.status?.includes(value) ? 'default' : 'outline'
                }
                className="cursor-pointer hover:bg-accent"
                onClick={() => handleStatusToggle(value)}
              >
                {label}
              </Badge>
            ))}
          </div>
        </div>

        {/* Health status filters */}
        <div className="flex items-center space-x-2">
          <span className="text-sm font-medium text-muted-foreground">
            Health:
          </span>
          <div className="flex flex-wrap gap-1">
            {HEALTH_STATUS_OPTIONS.map(({ value, label }) => (
              <Badge
                key={value}
                variant={
                  filters.health_status?.includes(value) ? 'default' : 'outline'
                }
                className="cursor-pointer hover:bg-accent"
                onClick={() => handleHealthStatusToggle(value)}
              >
                {label}
              </Badge>
            ))}
          </div>
        </div>

        {/* Tag filters */}
        {availableTags.length > 0 && (
          <div className="flex items-center space-x-2">
            <span className="text-sm font-medium text-muted-foreground">
              Tags:
            </span>
            <div className="flex flex-wrap gap-1">
              {availableTags.slice(0, 8).map(tag => (
                <Badge
                  key={tag}
                  variant={filters.tags?.includes(tag) ? 'default' : 'outline'}
                  className="cursor-pointer hover:bg-accent"
                  onClick={() => handleTagToggle(tag)}
                >
                  {tag}
                </Badge>
              ))}
              {availableTags.length > 8 && (
                <Badge variant="outline" className="cursor-pointer">
                  +{availableTags.length - 8} more
                </Badge>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Results summary */}
      {serverCount > 0 && (
        <div className="flex items-center justify-between text-sm text-muted-foreground">
          <div>
            Showing {filteredCount} of {serverCount} servers
            {hasActiveFilters && (
              <span className="ml-2">
                <Filter className="inline h-3 w-3 mr-1" />
                {activeFilterCount} filter{activeFilterCount !== 1 ? 's' : ''}{' '}
                applied
              </span>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
