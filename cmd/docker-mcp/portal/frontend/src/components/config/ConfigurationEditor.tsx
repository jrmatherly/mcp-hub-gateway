'use client';

import { AlertCircle, Code, Eye, Save, Settings, Undo } from 'lucide-react';
import React, { useState, useEffect } from 'react';
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import type { MCPConfig, ServerConfig } from '@/types/api';

interface ConfigurationEditorProps {
  config?: MCPConfig;
  isLoading?: boolean;
  onChange?: (config: MCPConfig) => void;
  onSave?: (config: MCPConfig) => void;
  className?: string;
}

interface ValidationError {
  path: string;
  message: string;
  severity: 'error' | 'warning';
}

const DEFAULT_CONFIG: MCPConfig = {
  gateway: {
    port: 8080,
    transport: 'stdio',
    log_level: 'info',
    enable_cors: true,
    timeout: 30000,
  },
  servers: {},
  secrets: {},
  catalog: {
    default_enabled: true,
    auto_update: false,
    cache_ttl: 3600,
  },
};

function JsonEditor({
  value,
  onChange,
  onValidate,
  className,
}: {
  value: string;
  onChange: (value: string) => void;
  onValidate: (errors: ValidationError[]) => void;
  className?: string;
}) {
  const [localValue, setLocalValue] = useState(value);
  const [errors, setErrors] = useState<ValidationError[]>([]);

  useEffect(() => {
    setLocalValue(value);
  }, [value]);

  const handleChange = (newValue: string) => {
    setLocalValue(newValue);

    // Validate JSON
    try {
      JSON.parse(newValue);
      setErrors([]);
      onValidate([]);
      onChange(newValue);
    } catch (error) {
      const validationError: ValidationError = {
        path: 'root',
        message: error instanceof Error ? error.message : 'Invalid JSON',
        severity: 'error',
      };
      setErrors([validationError]);
      onValidate([validationError]);
    }
  };

  return (
    <div className={cn('space-y-2', className)}>
      <textarea
        value={localValue}
        onChange={e => handleChange(e.target.value)}
        className="w-full h-96 p-4 font-mono text-sm border rounded-md bg-background resize-none focus:outline-none focus:ring-2 focus:ring-ring"
        placeholder="Enter JSON configuration..."
      />

      {errors.length > 0 && (
        <div className="space-y-1">
          {errors.map((error, index) => (
            <div
              key={index}
              className="flex items-center space-x-2 text-sm text-red-600"
            >
              <AlertCircle className="h-3 w-3" />
              <span>{error.message}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function GatewaySettings({
  gateway,
  onChange,
}: {
  gateway: MCPConfig['gateway'];
  onChange: (gateway: MCPConfig['gateway']) => void;
}) {
  const updateGateway = <K extends keyof NonNullable<MCPConfig['gateway']>>(
    field: K,
    value: NonNullable<MCPConfig['gateway']>[K]
  ) => {
    onChange({
      ...gateway,
      [field]: value,
    });
  };

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="port">Port</Label>
          <Input
            id="port"
            type="number"
            value={gateway?.port || 8080}
            onChange={e => updateGateway('port', parseInt(e.target.value))}
            min={1}
            max={65535}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="transport">Transport</Label>
          <Select
            value={gateway?.transport || 'stdio'}
            onValueChange={value =>
              updateGateway('transport', value as 'stdio' | 'streaming')
            }
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="stdio">STDIO</SelectItem>
              <SelectItem value="streaming">Streaming</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label htmlFor="log_level">Log Level</Label>
          <Select
            value={gateway?.log_level || 'info'}
            onValueChange={value =>
              updateGateway(
                'log_level',
                value as 'debug' | 'info' | 'warn' | 'error'
              )
            }
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="debug">Debug</SelectItem>
              <SelectItem value="info">Info</SelectItem>
              <SelectItem value="warn">Warning</SelectItem>
              <SelectItem value="error">Error</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label htmlFor="timeout">Timeout (ms)</Label>
          <Input
            id="timeout"
            type="number"
            value={gateway?.timeout || 30000}
            onChange={e => updateGateway('timeout', parseInt(e.target.value))}
            min={1000}
            max={300000}
          />
        </div>
      </div>

      <div className="flex items-center space-x-2">
        <Switch
          id="enable_cors"
          checked={gateway?.enable_cors || false}
          onCheckedChange={checked => updateGateway('enable_cors', checked)}
        />
        <Label htmlFor="enable_cors">Enable CORS</Label>
      </div>
    </div>
  );
}

function ServerSettings({
  servers,
  onChange,
}: {
  servers: MCPConfig['servers'];
  onChange: (servers: MCPConfig['servers']) => void;
}) {
  const [selectedServer, setSelectedServer] = useState<string>('');

  const serverNames = Object.keys(servers || {});

  const updateServer = (serverName: string, config: ServerConfig) => {
    onChange({
      ...servers,
      [serverName]: config,
    });
  };

  const removeServer = (serverName: string) => {
    const newServers = { ...servers };
    delete newServers[serverName];
    onChange(newServers);
    if (selectedServer === serverName) {
      setSelectedServer('');
    }
  };

  const selectedServerConfig = selectedServer
    ? servers?.[selectedServer]
    : null;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">Server Configurations</h3>
        <Badge variant="secondary">{serverNames.length} configured</Badge>
      </div>

      <div className="grid grid-cols-3 gap-6">
        {/* Server List */}
        <div className="space-y-2">
          <Label>Servers</Label>
          <div className="border rounded-md max-h-64 overflow-y-auto">
            {serverNames.length === 0 ? (
              <div className="p-4 text-center text-muted-foreground text-sm">
                No servers configured
              </div>
            ) : (
              serverNames.map(serverName => (
                <div
                  key={serverName}
                  className={cn(
                    'p-3 border-b last:border-b-0 cursor-pointer hover:bg-muted/50',
                    selectedServer === serverName && 'bg-accent'
                  )}
                  onClick={() => setSelectedServer(serverName)}
                >
                  <div className="flex items-center justify-between">
                    <span className="font-medium">{serverName}</span>
                    <Badge
                      variant={
                        servers?.[serverName]?.enabled ? 'success' : 'secondary'
                      }
                      className="text-xs"
                    >
                      {servers?.[serverName]?.enabled ? 'Enabled' : 'Disabled'}
                    </Badge>
                  </div>
                  {servers?.[serverName]?.image && (
                    <div className="text-xs text-muted-foreground mt-1">
                      {servers[serverName].image}
                    </div>
                  )}
                </div>
              ))
            )}
          </div>
        </div>

        {/* Server Configuration */}
        <div className="col-span-2 space-y-4">
          {selectedServerConfig ? (
            <>
              <div className="flex items-center justify-between">
                <h4 className="font-medium">{selectedServer}</h4>
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => removeServer(selectedServer)}
                >
                  Remove Server
                </Button>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Image</Label>
                  <Input
                    value={selectedServerConfig.image || ''}
                    onChange={e =>
                      updateServer(selectedServer, {
                        ...selectedServerConfig,
                        image: e.target.value,
                      })
                    }
                    placeholder="docker/mcp-server:latest"
                  />
                </div>

                <div className="space-y-2">
                  <Label>Port</Label>
                  <Input
                    type="number"
                    value={selectedServerConfig.port || ''}
                    onChange={e =>
                      updateServer(selectedServer, {
                        ...selectedServerConfig,
                        port: parseInt(e.target.value),
                      })
                    }
                    placeholder="8080"
                  />
                </div>
              </div>

              <div className="flex items-center space-x-2">
                <Switch
                  checked={selectedServerConfig.enabled}
                  onCheckedChange={checked =>
                    updateServer(selectedServer, {
                      ...selectedServerConfig,
                      enabled: checked,
                    })
                  }
                />
                <Label>Enabled</Label>
              </div>

              {selectedServerConfig.resources && (
                <div className="space-y-2">
                  <Label>Resource Limits</Label>
                  <div className="grid grid-cols-2 gap-2">
                    <Input
                      value={selectedServerConfig.resources.cpu_limit || ''}
                      onChange={e =>
                        updateServer(selectedServer, {
                          ...selectedServerConfig,
                          resources: {
                            ...selectedServerConfig.resources,
                            cpu_limit: e.target.value,
                          },
                        })
                      }
                      placeholder="CPU limit (e.g., 0.5)"
                    />
                    <Input
                      value={selectedServerConfig.resources.memory_limit || ''}
                      onChange={e =>
                        updateServer(selectedServer, {
                          ...selectedServerConfig,
                          resources: {
                            ...selectedServerConfig.resources,
                            memory_limit: e.target.value,
                          },
                        })
                      }
                      placeholder="Memory limit (e.g., 512m)"
                    />
                  </div>
                </div>
              )}
            </>
          ) : (
            <div className="text-center text-muted-foreground py-8">
              <Settings className="h-8 w-8 mx-auto mb-2" />
              <p>Select a server to configure its settings</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function CatalogSettings({
  catalog,
  onChange,
}: {
  catalog: MCPConfig['catalog'];
  onChange: (catalog: MCPConfig['catalog']) => void;
}) {
  const updateCatalog = <K extends keyof NonNullable<MCPConfig['catalog']>>(
    field: K,
    value: NonNullable<MCPConfig['catalog']>[K]
  ) => {
    onChange({
      ...catalog,
      [field]: value,
    });
  };

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="cache_ttl">Cache TTL (seconds)</Label>
          <Input
            id="cache_ttl"
            type="number"
            value={catalog?.cache_ttl || 3600}
            onChange={e => updateCatalog('cache_ttl', parseInt(e.target.value))}
            min={60}
            max={86400}
          />
        </div>
      </div>

      <div className="space-y-4">
        <div className="flex items-center space-x-2">
          <Switch
            id="default_enabled"
            checked={catalog?.default_enabled || false}
            onCheckedChange={checked =>
              updateCatalog('default_enabled', checked)
            }
          />
          <Label htmlFor="default_enabled">Enable new servers by default</Label>
        </div>

        <div className="flex items-center space-x-2">
          <Switch
            id="auto_update"
            checked={catalog?.auto_update || false}
            onCheckedChange={checked => updateCatalog('auto_update', checked)}
          />
          <Label htmlFor="auto_update">Auto-update catalog</Label>
        </div>
      </div>
    </div>
  );
}

function ConfigurationEditorSkeleton() {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="space-y-2">
            <Skeleton className="h-6 w-48" />
            <Skeleton className="h-4 w-64" />
          </div>
          <div className="flex space-x-2">
            <Skeleton className="h-9 w-20" />
            <Skeleton className="h-9 w-24" />
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-64 w-full" />
        </div>
      </CardContent>
    </Card>
  );
}

export function ConfigurationEditor({
  config,
  isLoading = false,
  onChange,
  onSave,
  className,
}: ConfigurationEditorProps) {
  const [localConfig, setLocalConfig] = useState<MCPConfig>(
    config || DEFAULT_CONFIG
  );
  const [activeTab, setActiveTab] = useState('visual');
  const [jsonValue, setJsonValue] = useState('');
  const [validationErrors, setValidationErrors] = useState<ValidationError[]>(
    []
  );
  const [hasChanges, setHasChanges] = useState(false);

  useEffect(() => {
    if (config) {
      setLocalConfig(config);
      setJsonValue(JSON.stringify(config, null, 2));
      setHasChanges(false);
    }
  }, [config]);

  const handleConfigChange = (newConfig: MCPConfig) => {
    setLocalConfig(newConfig);
    setJsonValue(JSON.stringify(newConfig, null, 2));
    setHasChanges(true);
    onChange?.(newConfig);
  };

  const handleJsonChange = (jsonString: string) => {
    setJsonValue(jsonString);
    if (validationErrors.length === 0) {
      try {
        const parsed = JSON.parse(jsonString);
        setLocalConfig(parsed);
        setHasChanges(true);
        onChange?.(parsed);
      } catch {
        // JSON is invalid, don't update config
      }
    }
  };

  const handleSave = () => {
    if (validationErrors.length === 0) {
      onSave?.(localConfig);
      setHasChanges(false);
      toast.success('Configuration saved successfully');
    } else {
      toast.error('Please fix validation errors before saving');
    }
  };

  const handleReset = () => {
    if (config) {
      setLocalConfig(config);
      setJsonValue(JSON.stringify(config, null, 2));
      setHasChanges(false);
      onChange?.(config);
      toast.info('Changes reset to original configuration');
    }
  };

  if (isLoading) {
    return (
      <div className={className}>
        <ConfigurationEditorSkeleton />
      </div>
    );
  }

  return (
    <div className={cn('space-y-6', className)}>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Configuration Editor</CardTitle>
              <CardDescription>
                Edit your MCP server configuration using the visual editor or
                JSON mode
              </CardDescription>
            </div>

            <div className="flex items-center space-x-2">
              {validationErrors.length > 0 && (
                <Badge
                  variant="destructive"
                  className="flex items-center space-x-1"
                >
                  <AlertCircle className="h-3 w-3" />
                  <span>{validationErrors.length} errors</span>
                </Badge>
              )}

              {hasChanges && (
                <Badge
                  variant="warning"
                  className="flex items-center space-x-1"
                >
                  <Settings className="h-3 w-3" />
                  <span>Unsaved changes</span>
                </Badge>
              )}

              <Button
                variant="outline"
                onClick={handleReset}
                disabled={!hasChanges}
              >
                <Undo className="h-4 w-4 mr-2" />
                Reset
              </Button>

              <Button
                onClick={handleSave}
                disabled={validationErrors.length > 0 || !hasChanges}
              >
                <Save className="h-4 w-4 mr-2" />
                Save
              </Button>
            </div>
          </div>
        </CardHeader>

        <CardContent>
          <Tabs value={activeTab} onValueChange={setActiveTab}>
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger
                value="visual"
                className="flex items-center space-x-2"
              >
                <Eye className="h-4 w-4" />
                <span>Visual Editor</span>
              </TabsTrigger>
              <TabsTrigger value="json" className="flex items-center space-x-2">
                <Code className="h-4 w-4" />
                <span>JSON Editor</span>
              </TabsTrigger>
            </TabsList>

            <TabsContent value="visual" className="mt-6">
              <Tabs defaultValue="gateway">
                <TabsList>
                  <TabsTrigger value="gateway">Gateway</TabsTrigger>
                  <TabsTrigger value="servers">Servers</TabsTrigger>
                  <TabsTrigger value="catalog">Catalog</TabsTrigger>
                </TabsList>

                <TabsContent value="gateway" className="mt-6">
                  <GatewaySettings
                    gateway={localConfig.gateway}
                    onChange={gateway =>
                      handleConfigChange({ ...localConfig, gateway })
                    }
                  />
                </TabsContent>

                <TabsContent value="servers" className="mt-6">
                  <ServerSettings
                    servers={localConfig.servers}
                    onChange={servers =>
                      handleConfigChange({ ...localConfig, servers })
                    }
                  />
                </TabsContent>

                <TabsContent value="catalog" className="mt-6">
                  <CatalogSettings
                    catalog={localConfig.catalog}
                    onChange={catalog =>
                      handleConfigChange({ ...localConfig, catalog })
                    }
                  />
                </TabsContent>
              </Tabs>
            </TabsContent>

            <TabsContent value="json" className="mt-6">
              <JsonEditor
                value={jsonValue}
                onChange={handleJsonChange}
                onValidate={setValidationErrors}
              />
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </div>
  );
}
