'use client';

import { ArrowLeft, Check, Minus, Plus, X } from 'lucide-react';
import React, { useMemo } from 'react';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { cn } from '@/lib/utils';
import type { MCPConfig } from '@/types/api';

interface ConfigurationDiffProps {
  config1: MCPConfig;
  config2: MCPConfig;
  onBack?: () => void;
  className?: string;
}

type DiffType = 'added' | 'removed' | 'modified' | 'unchanged';

interface DiffItem {
  path: string;
  type: DiffType;
  oldValue?: unknown;
  newValue?: unknown;
  description?: string;
}

function flattenObject(obj: unknown, prefix = ''): Record<string, unknown> {
  const flattened: Record<string, unknown> = {};

  if (!obj || typeof obj !== 'object' || Array.isArray(obj)) {
    return flattened;
  }

  const objRecord = obj as Record<string, unknown>;

  for (const key in objRecord) {
    if (Object.prototype.hasOwnProperty.call(objRecord, key)) {
      const path = prefix ? `${prefix}.${key}` : key;

      if (
        objRecord[key] !== null &&
        typeof objRecord[key] === 'object' &&
        !Array.isArray(objRecord[key])
      ) {
        Object.assign(flattened, flattenObject(objRecord[key], path));
      } else {
        flattened[path] = objRecord[key];
      }
    }
  }

  return flattened;
}

function computeDiff(config1: MCPConfig, config2: MCPConfig): DiffItem[] {
  const flat1 = flattenObject(config1);
  const flat2 = flattenObject(config2);
  const allKeys = new Set([...Object.keys(flat1), ...Object.keys(flat2)]);
  const diffs: DiffItem[] = [];

  for (const key of allKeys) {
    const value1 = flat1[key];
    const value2 = flat2[key];

    if (value1 === undefined && value2 !== undefined) {
      diffs.push({
        path: key,
        type: 'added',
        newValue: value2,
        description: getPathDescription(key),
      });
    } else if (value1 !== undefined && value2 === undefined) {
      diffs.push({
        path: key,
        type: 'removed',
        oldValue: value1,
        description: getPathDescription(key),
      });
    } else if (JSON.stringify(value1) !== JSON.stringify(value2)) {
      diffs.push({
        path: key,
        type: 'modified',
        oldValue: value1,
        newValue: value2,
        description: getPathDescription(key),
      });
    } else {
      diffs.push({
        path: key,
        type: 'unchanged',
        oldValue: value1,
        newValue: value2,
        description: getPathDescription(key),
      });
    }
  }

  return diffs.sort((a, b) => a.path.localeCompare(b.path));
}

function getPathDescription(path: string): string {
  const pathDescriptions: Record<string, string> = {
    'gateway.port': 'Gateway port number',
    'gateway.transport': 'Transport protocol (stdio/streaming)',
    'gateway.log_level': 'Logging level',
    'gateway.enable_cors': 'CORS enabled',
    'gateway.timeout': 'Request timeout',
    'catalog.default_enabled': 'Enable new servers by default',
    'catalog.auto_update': 'Auto-update catalog',
    'catalog.cache_ttl': 'Catalog cache TTL',
  };

  // Handle server-specific paths
  if (path.startsWith('servers.')) {
    const parts = path.split('.');
    if (parts.length >= 3) {
      const serverName = parts[1];
      const property = parts.slice(2).join('.');

      const propertyDescriptions: Record<string, string> = {
        enabled: 'Server enabled',
        image: 'Docker image',
        port: 'Port number',
        'resources.cpu_limit': 'CPU limit',
        'resources.memory_limit': 'Memory limit',
        'health_check.enabled': 'Health check enabled',
        'health_check.interval': 'Health check interval',
        'health_check.timeout': 'Health check timeout',
        'health_check.retries': 'Health check retries',
      };

      return propertyDescriptions[property] || `${serverName} ${property}`;
    }
  }

  return pathDescriptions[path] || path;
}

function DiffIcon({ type }: { type: DiffType }) {
  switch (type) {
    case 'added':
      return <Plus className="h-4 w-4 text-green-600" />;
    case 'removed':
      return <Minus className="h-4 w-4 text-red-600" />;
    case 'modified':
      return <X className="h-4 w-4 text-yellow-600" />;
    case 'unchanged':
      return <Check className="h-4 w-4 text-gray-400" />;
  }
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined) {
    return 'null';
  }
  if (typeof value === 'boolean') {
    return value ? 'true' : 'false';
  }
  if (typeof value === 'string') {
    return `"${value}"`;
  }
  return String(value);
}

function DiffRow({ item }: { item: DiffItem }) {
  return (
    <div
      className={cn(
        'flex items-center space-x-3 p-3 rounded-md border',
        item.type === 'added' &&
          'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800',
        item.type === 'removed' &&
          'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800',
        item.type === 'modified' &&
          'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800',
        item.type === 'unchanged' &&
          'bg-gray-50 dark:bg-gray-900/20 border-gray-200 dark:border-gray-800'
      )}
    >
      <DiffIcon type={item.type} />

      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between">
          <div className="font-medium text-sm">{item.path}</div>
          <div className="text-xs text-muted-foreground">
            {item.description}
          </div>
        </div>

        {item.type !== 'unchanged' && (
          <div className="mt-1 space-y-1">
            {item.oldValue !== undefined && (
              <div className="text-xs">
                <span className="text-red-600">- </span>
                <span className="font-mono">{formatValue(item.oldValue)}</span>
              </div>
            )}
            {item.newValue !== undefined && (
              <div className="text-xs">
                <span className="text-green-600">+ </span>
                <span className="font-mono">{formatValue(item.newValue)}</span>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function SummaryCard({
  title,
  count,
  color,
  icon,
}: {
  title: string;
  count: number;
  color: string;
  icon: React.ReactNode;
}) {
  return (
    <Card>
      <CardContent className="p-4">
        <div className="flex items-center space-x-3">
          <div className={cn('p-2 rounded-md', color)}>{icon}</div>
          <div>
            <div className="font-semibold text-lg">{count}</div>
            <div className="text-sm text-muted-foreground">{title}</div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function SideBySideView({
  config1,
  config2,
}: {
  config1: MCPConfig;
  config2: MCPConfig;
}) {
  return (
    <div className="grid grid-cols-2 gap-6">
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Configuration A</CardTitle>
        </CardHeader>
        <CardContent>
          <pre className="text-xs font-mono bg-muted p-4 rounded-md overflow-auto max-h-96">
            {JSON.stringify(config1, null, 2)}
          </pre>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Configuration B</CardTitle>
        </CardHeader>
        <CardContent>
          <pre className="text-xs font-mono bg-muted p-4 rounded-md overflow-auto max-h-96">
            {JSON.stringify(config2, null, 2)}
          </pre>
        </CardContent>
      </Card>
    </div>
  );
}

export function ConfigurationDiff({
  config1,
  config2,
  onBack,
  className,
}: ConfigurationDiffProps) {
  const diffs = useMemo(
    () => computeDiff(config1, config2),
    [config1, config2]
  );

  const summary = useMemo(() => {
    return diffs.reduce(
      (acc, diff) => {
        acc[diff.type]++;
        return acc;
      },
      { added: 0, removed: 0, modified: 0, unchanged: 0 }
    );
  }, [diffs]);

  const filteredDiffs = {
    all: diffs,
    changes: diffs.filter(d => d.type !== 'unchanged'),
    added: diffs.filter(d => d.type === 'added'),
    removed: diffs.filter(d => d.type === 'removed'),
    modified: diffs.filter(d => d.type === 'modified'),
  };

  return (
    <div className={cn('space-y-6', className)}>
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          {onBack && (
            <Button variant="ghost" size="sm" onClick={onBack}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
          )}
          <div>
            <h2 className="text-2xl font-bold">Configuration Comparison</h2>
            <p className="text-muted-foreground">
              Compare two configuration versions to see what changed
            </p>
          </div>
        </div>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-4 gap-4">
        <SummaryCard
          title="Added"
          count={summary.added}
          color="bg-green-100 dark:bg-green-900/20"
          icon={<Plus className="h-4 w-4 text-green-600" />}
        />
        <SummaryCard
          title="Removed"
          count={summary.removed}
          color="bg-red-100 dark:bg-red-900/20"
          icon={<Minus className="h-4 w-4 text-red-600" />}
        />
        <SummaryCard
          title="Modified"
          count={summary.modified}
          color="bg-yellow-100 dark:bg-yellow-900/20"
          icon={<X className="h-4 w-4 text-yellow-600" />}
        />
        <SummaryCard
          title="Unchanged"
          count={summary.unchanged}
          color="bg-gray-100 dark:bg-gray-900/20"
          icon={<Check className="h-4 w-4 text-gray-400" />}
        />
      </div>

      {/* Diff Details */}
      <Card>
        <CardHeader>
          <CardTitle>Detailed Changes</CardTitle>
          <CardDescription>
            Review the specific differences between configurations
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs defaultValue="changes">
            <TabsList>
              <TabsTrigger value="changes">
                Changes Only ({filteredDiffs.changes.length})
              </TabsTrigger>
              <TabsTrigger value="all">
                All Properties ({filteredDiffs.all.length})
              </TabsTrigger>
              <TabsTrigger value="side-by-side">Side by Side</TabsTrigger>
            </TabsList>

            <TabsContent value="changes" className="mt-6">
              <div className="space-y-3">
                {filteredDiffs.changes.length === 0 ? (
                  <div className="text-center py-8 text-muted-foreground">
                    <Check className="h-12 w-12 mx-auto mb-4" />
                    <h3 className="font-medium mb-2">No differences found</h3>
                    <p className="text-sm">The configurations are identical.</p>
                  </div>
                ) : (
                  filteredDiffs.changes.map((diff, index) => (
                    <DiffRow key={index} item={diff} />
                  ))
                )}
              </div>
            </TabsContent>

            <TabsContent value="all" className="mt-6">
              <div className="space-y-3">
                {filteredDiffs.all.map((diff, index) => (
                  <DiffRow key={index} item={diff} />
                ))}
              </div>
            </TabsContent>

            <TabsContent value="side-by-side" className="mt-6">
              <SideBySideView config1={config1} config2={config2} />
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </div>
  );
}
