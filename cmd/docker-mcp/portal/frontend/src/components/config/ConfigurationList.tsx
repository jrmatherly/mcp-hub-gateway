'use client';

import {
  CheckCircle,
  Clock,
  FileText,
  GitCompare,
  MoreHorizontal,
  Settings,
  Shield,
  Zap,
} from 'lucide-react';
import React, { useState } from 'react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import type { MCPConfig } from '@/types/api';

interface ConfigurationListProps {
  currentConfig?: MCPConfig;
  isLoading?: boolean;
  onConfigSelect?: (config: MCPConfig) => void;
  onDiffView?: (config1: MCPConfig, config2: MCPConfig) => void;
  className?: string;
}

interface ConfigurationItem {
  id: string;
  name: string;
  description: string;
  type: 'current' | 'backup' | 'template' | 'imported';
  lastModified: string;
  serverCount: number;
  hasSecrets: boolean;
  status: 'active' | 'inactive' | 'draft';
  config: MCPConfig;
}

// Mock configurations for demonstration
const mockConfigurations: ConfigurationItem[] = [
  {
    id: 'current',
    name: 'Current Configuration',
    description: 'Your active MCP server configuration',
    type: 'current',
    lastModified: new Date().toISOString(),
    serverCount: 5,
    hasSecrets: true,
    status: 'active',
    config: {},
  },
  {
    id: 'backup-1',
    name: 'Backup - 2024-01-15',
    description: 'Automatic backup created before last update',
    type: 'backup',
    lastModified: '2024-01-15T10:30:00Z',
    serverCount: 4,
    hasSecrets: true,
    status: 'inactive',
    config: {},
  },
  {
    id: 'template-dev',
    name: 'Development Template',
    description: 'Template for development environment setup',
    type: 'template',
    lastModified: '2024-01-10T15:45:00Z',
    serverCount: 8,
    hasSecrets: false,
    status: 'draft',
    config: {},
  },
];

const getTypeIcon = (type: ConfigurationItem['type']) => {
  switch (type) {
    case 'current':
      return <CheckCircle className="h-4 w-4 text-green-500" />;
    case 'backup':
      return <Shield className="h-4 w-4 text-blue-500" />;
    case 'template':
      return <FileText className="h-4 w-4 text-purple-500" />;
    case 'imported':
      return <Zap className="h-4 w-4 text-orange-500" />;
    default:
      return <FileText className="h-4 w-4 text-gray-500" />;
  }
};

const getTypeBadgeVariant = (type: ConfigurationItem['type']) => {
  switch (type) {
    case 'current':
      return 'success' as const;
    case 'backup':
      return 'secondary' as const;
    case 'template':
      return 'outline' as const;
    case 'imported':
      return 'warning' as const;
    default:
      return 'secondary' as const;
  }
};

const getStatusBadgeVariant = (status: ConfigurationItem['status']) => {
  switch (status) {
    case 'active':
      return 'success' as const;
    case 'inactive':
      return 'secondary' as const;
    case 'draft':
      return 'outline' as const;
    default:
      return 'secondary' as const;
  }
};

const formatDate = (dateString: string) => {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  const diffDays = Math.floor(diffHours / 24);

  if (diffDays === 0) {
    if (diffHours === 0) {
      return 'Just now';
    }
    return `${diffHours} hours ago`;
  } else if (diffDays < 7) {
    return `${diffDays} days ago`;
  } else {
    return date.toLocaleDateString();
  }
};

function ConfigurationCard({
  item,
  onSelect,
  onCompare,
  isComparison = false,
}: {
  item: ConfigurationItem;
  onSelect?: () => void;
  onCompare?: () => void;
  isComparison?: boolean;
}) {
  return (
    <Card
      className={cn(
        'transition-all duration-200 hover:shadow-md cursor-pointer',
        isComparison && 'ring-2 ring-primary ring-offset-2'
      )}
      onClick={onSelect}
    >
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="flex items-start space-x-3 flex-1 min-w-0">
            {getTypeIcon(item.type)}
            <div className="flex-1 min-w-0">
              <CardTitle className="text-lg font-semibold truncate">
                {item.name}
              </CardTitle>
              <CardDescription className="mt-1 line-clamp-2">
                {item.description}
              </CardDescription>
            </div>
          </div>

          <div className="flex items-center space-x-2 ml-3">
            <Badge variant={getTypeBadgeVariant(item.type)}>{item.type}</Badge>
            <Badge variant={getStatusBadgeVariant(item.status)}>
              {item.status}
            </Badge>
          </div>
        </div>
      </CardHeader>

      <CardContent className="pt-0">
        <div className="space-y-3">
          {/* Configuration metadata */}
          <div className="grid grid-cols-2 gap-3 text-sm">
            <div className="flex items-center space-x-1 text-muted-foreground">
              <Settings className="h-3 w-3" />
              <span>{item.serverCount} servers</span>
            </div>

            <div className="flex items-center space-x-1 text-muted-foreground">
              <Clock className="h-3 w-3" />
              <span>{formatDate(item.lastModified)}</span>
            </div>

            {item.hasSecrets && (
              <div className="flex items-center space-x-1 text-muted-foreground">
                <Shield className="h-3 w-3" />
                <span>Contains secrets</span>
              </div>
            )}
          </div>

          {/* Actions */}
          <div className="flex items-center justify-between pt-2 border-t">
            <Button
              variant="ghost"
              size="sm"
              onClick={e => {
                e.stopPropagation();
                onSelect?.();
              }}
            >
              <Settings className="h-4 w-4 mr-2" />
              Edit
            </Button>

            <div className="flex items-center space-x-1">
              {onCompare && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={e => {
                    e.stopPropagation();
                    onCompare();
                  }}
                >
                  <GitCompare className="h-4 w-4 mr-2" />
                  Compare
                </Button>
              )}

              <Button variant="ghost" size="sm">
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function ConfigurationListSkeleton() {
  return (
    <div className="space-y-4">
      {Array.from({ length: 3 }).map((_, index) => (
        <Card key={index}>
          <CardHeader className="pb-3">
            <div className="flex items-start justify-between">
              <div className="flex items-start space-x-3 flex-1">
                <Skeleton className="h-4 w-4 rounded-full" />
                <div className="space-y-2 flex-1">
                  <Skeleton className="h-5 w-48" />
                  <Skeleton className="h-4 w-64" />
                </div>
              </div>
              <div className="flex space-x-2">
                <Skeleton className="h-5 w-16" />
                <Skeleton className="h-5 w-16" />
              </div>
            </div>
          </CardHeader>
          <CardContent className="pt-0">
            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <Skeleton className="h-4 w-20" />
                <Skeleton className="h-4 w-24" />
              </div>
              <div className="flex justify-between items-center pt-2 border-t">
                <Skeleton className="h-8 w-16" />
                <div className="flex space-x-1">
                  <Skeleton className="h-8 w-20" />
                  <Skeleton className="h-8 w-8" />
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

export function ConfigurationList({
  currentConfig,
  isLoading = false,
  onConfigSelect,
  onDiffView,
  className,
}: ConfigurationListProps) {
  const [searchTerm, setSearchTerm] = useState('');
  const [filterType, setFilterType] = useState<string>('all');
  const [comparisonMode, setComparisonMode] = useState(false);
  const [selectedForComparison, setSelectedForComparison] = useState<
    ConfigurationItem[]
  >([]);

  // Update mock current config with real data
  const configurations = mockConfigurations.map(config =>
    config.id === 'current'
      ? { ...config, config: currentConfig || {} }
      : config
  );

  const filteredConfigurations = configurations.filter(config => {
    const matchesSearch =
      config.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      config.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesType = filterType === 'all' || config.type === filterType;
    return matchesSearch && matchesType;
  });

  const handleConfigSelect = (item: ConfigurationItem) => {
    if (comparisonMode) {
      if (selectedForComparison.includes(item)) {
        setSelectedForComparison(prev => prev.filter(c => c.id !== item.id));
      } else if (selectedForComparison.length < 2) {
        setSelectedForComparison(prev => [...prev, item]);
      }
    } else {
      onConfigSelect?.(item.config);
    }
  };

  const handleCompare = () => {
    if (selectedForComparison.length === 2) {
      onDiffView?.(
        selectedForComparison[0].config,
        selectedForComparison[1].config
      );
      setComparisonMode(false);
      setSelectedForComparison([]);
    }
  };

  if (isLoading) {
    return (
      <div className={className}>
        <ConfigurationListSkeleton />
      </div>
    );
  }

  return (
    <div className={cn('space-y-6', className)}>
      {/* Filters and Search */}
      <Card>
        <CardHeader>
          <CardTitle>Configuration Library</CardTitle>
          <CardDescription>
            Manage your configuration versions, backups, and templates
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center space-x-4">
            <div className="flex-1">
              <Input
                placeholder="Search configurations..."
                value={searchTerm}
                onChange={e => setSearchTerm(e.target.value)}
                className="max-w-md"
              />
            </div>

            <Select value={filterType} onValueChange={setFilterType}>
              <SelectTrigger className="w-40">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Types</SelectItem>
                <SelectItem value="current">Current</SelectItem>
                <SelectItem value="backup">Backups</SelectItem>
                <SelectItem value="template">Templates</SelectItem>
                <SelectItem value="imported">Imported</SelectItem>
              </SelectContent>
            </Select>

            <Button
              variant={comparisonMode ? 'default' : 'outline'}
              onClick={() => {
                setComparisonMode(!comparisonMode);
                setSelectedForComparison([]);
              }}
            >
              <GitCompare className="h-4 w-4 mr-2" />
              Compare
            </Button>

            {comparisonMode && selectedForComparison.length === 2 && (
              <Button onClick={handleCompare}>Compare Selected</Button>
            )}
          </div>

          {comparisonMode && (
            <div className="mt-4 p-3 bg-blue-50 dark:bg-blue-900/20 rounded-md">
              <p className="text-sm text-blue-600 dark:text-blue-400">
                Comparison mode: Select 2 configurations to compare. Selected:{' '}
                {selectedForComparison.length}/2
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Configuration List */}
      <div className="space-y-4">
        {filteredConfigurations.length === 0 ? (
          <Card>
            <CardContent className="pt-6">
              <div className="text-center py-8">
                <FileText className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                <h3 className="text-lg font-medium text-foreground mb-2">
                  No configurations found
                </h3>
                <p className="text-sm text-muted-foreground">
                  {searchTerm || filterType !== 'all'
                    ? 'Try adjusting your search or filter criteria.'
                    : 'Start by creating your first configuration.'}
                </p>
              </div>
            </CardContent>
          </Card>
        ) : (
          filteredConfigurations.map(item => (
            <ConfigurationCard
              key={item.id}
              item={item}
              onSelect={() => handleConfigSelect(item)}
              onCompare={
                comparisonMode ? () => handleConfigSelect(item) : undefined
              }
              isComparison={selectedForComparison.some(c => c.id === item.id)}
            />
          ))
        )}
      </div>
    </div>
  );
}
