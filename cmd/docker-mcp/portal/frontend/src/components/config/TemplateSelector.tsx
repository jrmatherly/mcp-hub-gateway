'use client';

import {
  Box,
  CheckCircle,
  Copy,
  FileText,
  Globe,
  Laptop,
  Play,
  Server,
  Settings,
  Shield,
  Zap,
} from 'lucide-react';
import React, { useState } from 'react';
import { toast } from 'sonner';
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
import { cn } from '@/lib/utils';
import type { MCPConfig } from '@/types/api';

interface TemplateSelectorProps {
  onTemplateApply?: (template: MCPConfig) => void;
  currentConfig?: MCPConfig;
  className?: string;
}

interface ConfigTemplate {
  id: string;
  name: string;
  description: string;
  category: 'development' | 'production' | 'testing' | 'minimal';
  tags: string[];
  icon: React.ReactNode;
  config: MCPConfig;
  popularity: number;
  author?: string;
  featured?: boolean;
}

const templates: ConfigTemplate[] = [
  {
    id: 'minimal',
    name: 'Minimal Setup',
    description: 'Basic configuration with essential servers only',
    category: 'minimal',
    tags: ['basic', 'starter', 'lightweight'],
    icon: <Box className="h-6 w-6" />,
    popularity: 95,
    author: 'MCP Team',
    featured: true,
    config: {
      gateway: {
        port: 8080,
        transport: 'stdio',
        log_level: 'info',
        enable_cors: true,
        timeout: 30000,
      },
      servers: {
        filesystem: {
          enabled: true,
          image: 'docker/mcp-filesystem:latest',
          resources: {
            cpu_limit: '0.5',
            memory_limit: '256m',
          },
          health_check: {
            enabled: true,
            interval: 30,
            timeout: 5,
            retries: 3,
          },
        },
      },
      catalog: {
        default_enabled: true,
        auto_update: false,
        cache_ttl: 3600,
      },
    },
  },
  {
    id: 'development',
    name: 'Development Environment',
    description: 'Full development setup with debugging and testing tools',
    category: 'development',
    tags: ['dev', 'debugging', 'testing', 'comprehensive'],
    icon: <Laptop className="h-6 w-6" />,
    popularity: 87,
    author: 'MCP Team',
    featured: true,
    config: {
      gateway: {
        port: 8080,
        transport: 'streaming',
        log_level: 'debug',
        enable_cors: true,
        timeout: 60000,
      },
      servers: {
        filesystem: {
          enabled: true,
          image: 'docker/mcp-filesystem:latest',
          resources: {
            cpu_limit: '1.0',
            memory_limit: '512m',
          },
        },
        git: {
          enabled: true,
          image: 'docker/mcp-git:latest',
          resources: {
            cpu_limit: '0.5',
            memory_limit: '256m',
          },
        },
        database: {
          enabled: true,
          image: 'docker/mcp-postgres:latest',
          port: 5432,
          resources: {
            cpu_limit: '1.0',
            memory_limit: '1g',
          },
        },
        memory: {
          enabled: true,
          image: 'docker/mcp-memory:latest',
          resources: {
            cpu_limit: '0.5',
            memory_limit: '256m',
          },
        },
      },
      catalog: {
        default_enabled: true,
        auto_update: true,
        cache_ttl: 1800,
      },
    },
  },
  {
    id: 'production',
    name: 'Production Ready',
    description: 'Optimized configuration for production deployments',
    category: 'production',
    tags: ['production', 'performance', 'reliable', 'secure'],
    icon: <Server className="h-6 w-6" />,
    popularity: 92,
    author: 'MCP Team',
    featured: true,
    config: {
      gateway: {
        port: 8080,
        transport: 'streaming',
        log_level: 'warn',
        enable_cors: false,
        timeout: 30000,
      },
      servers: {
        filesystem: {
          enabled: true,
          image: 'docker/mcp-filesystem:stable',
          resources: {
            cpu_limit: '2.0',
            memory_limit: '1g',
          },
          health_check: {
            enabled: true,
            interval: 15,
            timeout: 5,
            retries: 3,
          },
        },
        database: {
          enabled: true,
          image: 'docker/mcp-postgres:stable',
          port: 5432,
          resources: {
            cpu_limit: '4.0',
            memory_limit: '4g',
          },
          health_check: {
            enabled: true,
            interval: 30,
            timeout: 10,
            retries: 5,
          },
        },
      },
      catalog: {
        default_enabled: false,
        auto_update: false,
        cache_ttl: 7200,
      },
    },
  },
  {
    id: 'ai-workflow',
    name: 'AI Workflow',
    description:
      'Configuration optimized for AI and machine learning workflows',
    category: 'development',
    tags: ['ai', 'ml', 'workflow', 'specialized'],
    icon: <Zap className="h-6 w-6" />,
    popularity: 78,
    author: 'AI Team',
    config: {
      gateway: {
        port: 8080,
        transport: 'streaming',
        log_level: 'info',
        enable_cors: true,
        timeout: 120000,
      },
      servers: {
        memory: {
          enabled: true,
          image: 'docker/mcp-memory:latest',
          resources: {
            cpu_limit: '2.0',
            memory_limit: '2g',
          },
        },
        search: {
          enabled: true,
          image: 'docker/mcp-search:latest',
          resources: {
            cpu_limit: '1.0',
            memory_limit: '1g',
          },
        },
        filesystem: {
          enabled: true,
          image: 'docker/mcp-filesystem:latest',
          resources: {
            cpu_limit: '1.0',
            memory_limit: '512m',
          },
        },
        github: {
          enabled: true,
          image: 'docker/mcp-github:latest',
          resources: {
            cpu_limit: '0.5',
            memory_limit: '256m',
          },
        },
      },
      catalog: {
        default_enabled: true,
        auto_update: true,
        cache_ttl: 3600,
      },
    },
  },
  {
    id: 'web-development',
    name: 'Web Development',
    description:
      'Setup for modern web development with API and database access',
    category: 'development',
    tags: ['web', 'api', 'frontend', 'backend'],
    icon: <Globe className="h-6 w-6" />,
    popularity: 83,
    author: 'Web Team',
    config: {
      gateway: {
        port: 8080,
        transport: 'streaming',
        log_level: 'info',
        enable_cors: true,
        timeout: 45000,
      },
      servers: {
        filesystem: {
          enabled: true,
          image: 'docker/mcp-filesystem:latest',
          resources: {
            cpu_limit: '1.0',
            memory_limit: '512m',
          },
        },
        fetch: {
          enabled: true,
          image: 'docker/mcp-fetch:latest',
          resources: {
            cpu_limit: '0.5',
            memory_limit: '256m',
          },
        },
        database: {
          enabled: true,
          image: 'docker/mcp-postgres:latest',
          port: 5432,
          resources: {
            cpu_limit: '2.0',
            memory_limit: '2g',
          },
        },
        git: {
          enabled: true,
          image: 'docker/mcp-git:latest',
          resources: {
            cpu_limit: '0.5',
            memory_limit: '256m',
          },
        },
      },
      catalog: {
        default_enabled: true,
        auto_update: true,
        cache_ttl: 3600,
      },
    },
  },
  {
    id: 'testing',
    name: 'Testing Environment',
    description: 'Isolated testing setup with mock services and test data',
    category: 'testing',
    tags: ['testing', 'mock', 'isolated', 'ci'],
    icon: <CheckCircle className="h-6 w-6" />,
    popularity: 71,
    author: 'QA Team',
    config: {
      gateway: {
        port: 8080,
        transport: 'stdio',
        log_level: 'debug',
        enable_cors: true,
        timeout: 15000,
      },
      servers: {
        memory: {
          enabled: true,
          image: 'docker/mcp-memory:latest',
          resources: {
            cpu_limit: '0.5',
            memory_limit: '256m',
          },
        },
        filesystem: {
          enabled: true,
          image: 'docker/mcp-filesystem:latest',
          resources: {
            cpu_limit: '0.5',
            memory_limit: '256m',
          },
        },
      },
      catalog: {
        default_enabled: false,
        auto_update: false,
        cache_ttl: 1800,
      },
    },
  },
];

function getCategoryIcon(category: ConfigTemplate['category']) {
  switch (category) {
    case 'development':
      return <Laptop className="h-4 w-4" />;
    case 'production':
      return <Server className="h-4 w-4" />;
    case 'testing':
      return <CheckCircle className="h-4 w-4" />;
    case 'minimal':
      return <Box className="h-4 w-4" />;
    default:
      return <FileText className="h-4 w-4" />;
  }
}

function getCategoryColor(category: ConfigTemplate['category']) {
  switch (category) {
    case 'development':
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-300';
    case 'production':
      return 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-300';
    case 'testing':
      return 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-300';
    case 'minimal':
      return 'bg-gray-100 text-gray-800 dark:bg-gray-900/20 dark:text-gray-300';
    default:
      return 'bg-purple-100 text-purple-800 dark:bg-purple-900/20 dark:text-purple-300';
  }
}

function TemplateCard({
  template,
  onApply,
  onPreview,
  isApplied = false,
}: {
  template: ConfigTemplate;
  onApply: () => void;
  onPreview: () => void;
  isApplied?: boolean;
}) {
  const serverCount = Object.keys(template.config.servers || {}).length;

  return (
    <Card
      className={cn(
        'transition-all duration-200 hover:shadow-md',
        template.featured && 'ring-1 ring-primary/20',
        isApplied && 'ring-2 ring-green-500 ring-offset-2'
      )}
    >
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="flex items-start space-x-3 flex-1 min-w-0">
            <div className="p-2 bg-muted rounded-md">{template.icon}</div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center space-x-2">
                <CardTitle className="text-lg font-semibold truncate">
                  {template.name}
                </CardTitle>
                {template.featured && (
                  <Badge variant="secondary" className="text-xs">
                    Featured
                  </Badge>
                )}
                {isApplied && (
                  <Badge variant="success" className="text-xs">
                    Current
                  </Badge>
                )}
              </div>
              <CardDescription className="mt-1 line-clamp-2">
                {template.description}
              </CardDescription>
            </div>
          </div>

          <div className="flex flex-col items-end space-y-2">
            <Badge
              className={cn('text-xs', getCategoryColor(template.category))}
            >
              <div className="flex items-center space-x-1">
                {getCategoryIcon(template.category)}
                <span>{template.category}</span>
              </div>
            </Badge>
            <div className="text-xs text-muted-foreground">
              {template.popularity}% popular
            </div>
          </div>
        </div>
      </CardHeader>

      <CardContent className="pt-0">
        <div className="space-y-4">
          {/* Template metadata */}
          <div className="grid grid-cols-2 gap-3 text-sm">
            <div className="flex items-center space-x-1 text-muted-foreground">
              <Settings className="h-3 w-3" />
              <span>{serverCount} servers</span>
            </div>

            <div className="flex items-center space-x-1 text-muted-foreground">
              <Shield className="h-3 w-3" />
              <span>{template.config.gateway?.transport || 'stdio'}</span>
            </div>

            {template.author && (
              <div className="text-muted-foreground col-span-2">
                by {template.author}
              </div>
            )}
          </div>

          {/* Tags */}
          <div className="flex flex-wrap gap-1">
            {template.tags.slice(0, 3).map(tag => (
              <Badge key={tag} variant="outline" className="text-xs">
                {tag}
              </Badge>
            ))}
            {template.tags.length > 3 && (
              <Badge variant="outline" className="text-xs">
                +{template.tags.length - 3}
              </Badge>
            )}
          </div>

          {/* Actions */}
          <div className="flex items-center justify-between pt-2 border-t">
            <Button variant="ghost" size="sm" onClick={onPreview}>
              <FileText className="h-4 w-4 mr-2" />
              Preview
            </Button>

            <div className="flex items-center space-x-1">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  navigator.clipboard.writeText(
                    JSON.stringify(template.config, null, 2)
                  );
                  toast.success('Template copied to clipboard');
                }}
              >
                <Copy className="h-4 w-4" />
              </Button>

              <Button size="sm" onClick={onApply} disabled={isApplied}>
                <Play className="h-4 w-4 mr-2" />
                {isApplied ? 'Applied' : 'Apply'}
              </Button>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

export function TemplateSelector({
  onTemplateApply,
  currentConfig,
  className,
}: TemplateSelectorProps) {
  const [searchTerm, setSearchTerm] = useState('');
  const [categoryFilter, setCategoryFilter] = useState<string>('all');
  const [sortBy, setSortBy] = useState<'popularity' | 'name'>('popularity');
  const [previewTemplate, setPreviewTemplate] = useState<ConfigTemplate | null>(
    null
  );

  const filteredTemplates = templates
    .filter(template => {
      const matchesSearch =
        template.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        template.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
        template.tags.some(tag =>
          tag.toLowerCase().includes(searchTerm.toLowerCase())
        );
      const matchesCategory =
        categoryFilter === 'all' || template.category === categoryFilter;
      return matchesSearch && matchesCategory;
    })
    .sort((a, b) => {
      if (sortBy === 'popularity') {
        return b.popularity - a.popularity;
      }
      return a.name.localeCompare(b.name);
    });

  const handleTemplateApply = (template: ConfigTemplate) => {
    onTemplateApply?.(template.config);
    toast.success(`Applied "${template.name}" template`);
  };

  const isCurrentTemplate = (template: ConfigTemplate) => {
    if (!currentConfig) return false;
    return JSON.stringify(template.config) === JSON.stringify(currentConfig);
  };

  return (
    <div className={cn('space-y-6', className)}>
      {/* Header */}
      <Card>
        <CardHeader>
          <CardTitle>Configuration Templates</CardTitle>
          <CardDescription>
            Choose from pre-built configuration templates to quickly set up your
            MCP environment
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center space-x-4">
            <div className="flex-1">
              <Input
                placeholder="Search templates..."
                value={searchTerm}
                onChange={e => setSearchTerm(e.target.value)}
                className="max-w-md"
              />
            </div>

            <Select value={categoryFilter} onValueChange={setCategoryFilter}>
              <SelectTrigger className="w-40">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Categories</SelectItem>
                <SelectItem value="development">Development</SelectItem>
                <SelectItem value="production">Production</SelectItem>
                <SelectItem value="testing">Testing</SelectItem>
                <SelectItem value="minimal">Minimal</SelectItem>
              </SelectContent>
            </Select>

            <Select
              value={sortBy}
              onValueChange={(value: 'popularity' | 'name') => setSortBy(value)}
            >
              <SelectTrigger className="w-32">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="popularity">Popular</SelectItem>
                <SelectItem value="name">Name</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* Templates Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {filteredTemplates.length === 0 ? (
          <div className="col-span-full">
            <Card>
              <CardContent className="pt-6">
                <div className="text-center py-8">
                  <FileText className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                  <h3 className="text-lg font-medium text-foreground mb-2">
                    No templates found
                  </h3>
                  <p className="text-sm text-muted-foreground">
                    Try adjusting your search or filter criteria.
                  </p>
                </div>
              </CardContent>
            </Card>
          </div>
        ) : (
          filteredTemplates.map(template => (
            <TemplateCard
              key={template.id}
              template={template}
              onApply={() => handleTemplateApply(template)}
              onPreview={() => setPreviewTemplate(template)}
              isApplied={isCurrentTemplate(template)}
            />
          ))
        )}
      </div>

      {/* Preview Modal */}
      {previewTemplate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-background/80 backdrop-blur-sm">
          <Card className="w-full max-w-4xl max-h-[80vh] overflow-hidden">
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>{previewTemplate.name} Preview</CardTitle>
                  <CardDescription>
                    Review the template configuration before applying
                  </CardDescription>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setPreviewTemplate(null)}
                >
                  ×
                </Button>
              </div>
            </CardHeader>
            <CardContent className="overflow-auto">
              <pre className="text-xs font-mono bg-muted p-4 rounded-md overflow-auto">
                {JSON.stringify(previewTemplate.config, null, 2)}
              </pre>
              <div className="flex items-center justify-end space-x-2 mt-4 pt-4 border-t">
                <Button
                  variant="outline"
                  onClick={() => {
                    navigator.clipboard.writeText(
                      JSON.stringify(previewTemplate.config, null, 2)
                    );
                    toast.success('Template copied to clipboard');
                  }}
                >
                  <Copy className="h-4 w-4 mr-2" />
                  Copy
                </Button>
                <Button
                  onClick={() => {
                    handleTemplateApply(previewTemplate);
                    setPreviewTemplate(null);
                  }}
                >
                  <Play className="h-4 w-4 mr-2" />
                  Apply Template
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
