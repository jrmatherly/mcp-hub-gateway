'use client';

import { useState } from 'react';
import {
  AlertTriangle,
  Cog,
  Save,
  Shield,
  Server,
  Clock,
  RefreshCw,
  Database,
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
import { Switch } from '@/components/ui/switch';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  useSystemConfig,
  useUpdateSystemConfig,
  useSystemOperation,
  type SystemConfig,
} from '@/hooks/api/use-admin';

interface SettingSectionProps {
  title: string;
  description: string;
  icon: React.ElementType;
  children: React.ReactNode;
  badge?: {
    text: string;
    variant: 'default' | 'secondary' | 'destructive' | 'outline';
  };
}

function SettingSection({
  title,
  description,
  icon: Icon,
  children,
  badge,
}: SettingSectionProps) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <Icon className="h-5 w-5 text-primary" />
            <div>
              <CardTitle className="text-lg">{title}</CardTitle>
              <CardDescription>{description}</CardDescription>
            </div>
          </div>
          {badge && <Badge variant={badge.variant}>{badge.text}</Badge>}
        </div>
      </CardHeader>
      <CardContent>{children}</CardContent>
    </Card>
  );
}

interface ConfirmationDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  description: string;
  confirmText: string;
  variant: 'default' | 'destructive';
  isLoading?: boolean;
}

function ConfirmationDialog({
  isOpen,
  onClose,
  onConfirm,
  title,
  description,
  confirmText,
  variant,
  isLoading,
}: ConfirmationDialogProps) {
  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center space-x-2">
            <AlertTriangle className="h-5 w-5 text-amber-500" />
            <span>{title}</span>
          </DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button variant={variant} onClick={onConfirm} disabled={isLoading}>
            {isLoading ? 'Processing...' : confirmText}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function MaintenanceSettings({
  config,
  onUpdate,
}: {
  config: SystemConfig;
  onUpdate: (updates: Partial<SystemConfig>) => void;
}) {
  const [showMaintenanceDialog, setShowMaintenanceDialog] = useState(false);
  const [maintenanceMessage, setMaintenanceMessage] = useState(
    config.maintenance.message || ''
  );

  const handleToggleMaintenance = () => {
    if (!config.maintenance.enabled) {
      setShowMaintenanceDialog(true);
    } else {
      onUpdate({
        maintenance: {
          ...config.maintenance,
          enabled: false,
          message: undefined,
        },
      });
    }
  };

  const handleEnableMaintenance = () => {
    onUpdate({
      maintenance: {
        enabled: true,
        message:
          maintenanceMessage ||
          'System is under maintenance. Please try again later.',
        scheduledAt: new Date(),
      },
    });
    setShowMaintenanceDialog(false);
  };

  return (
    <SettingSection
      title="Maintenance Mode"
      description="Control system availability and maintenance windows"
      icon={Cog}
      badge={
        config.maintenance.enabled
          ? { text: 'Active', variant: 'destructive' }
          : undefined
      }
    >
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <Label className="text-sm font-medium">
              Enable Maintenance Mode
            </Label>
            <p className="text-xs text-muted-foreground">
              Temporarily disable system access for maintenance
            </p>
          </div>
          <Switch
            checked={config.maintenance.enabled}
            onCheckedChange={handleToggleMaintenance}
          />
        </div>

        {config.maintenance.enabled && (
          <div className="p-4 bg-amber-50 border border-amber-200 rounded-md">
            <div className="flex items-center space-x-2 mb-2">
              <Cog className="h-4 w-4 text-amber-600" />
              <span className="text-sm font-medium text-amber-800">
                Maintenance Mode Active
              </span>
            </div>
            <p className="text-sm text-amber-700">
              {config.maintenance.message}
            </p>
            {config.maintenance.scheduledAt && (
              <p className="text-xs text-amber-600 mt-1">
                Started:{' '}
                {new Date(config.maintenance.scheduledAt).toLocaleString()}
              </p>
            )}
          </div>
        )}
      </div>

      <Dialog
        open={showMaintenanceDialog}
        onOpenChange={setShowMaintenanceDialog}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Enable Maintenance Mode</DialogTitle>
            <DialogDescription>
              This will prevent users from accessing the system. Make sure to
              notify users beforehand.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div>
              <Label htmlFor="maintenance-message">Maintenance Message</Label>
              <Input
                id="maintenance-message"
                value={maintenanceMessage}
                onChange={e => setMaintenanceMessage(e.target.value)}
                placeholder="System is under maintenance..."
              />
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowMaintenanceDialog(false)}
            >
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleEnableMaintenance}>
              Enable Maintenance Mode
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </SettingSection>
  );
}

function SecuritySettings({
  config,
  onUpdate,
}: {
  config: SystemConfig;
  onUpdate: (updates: Partial<SystemConfig>) => void;
}) {
  return (
    <SettingSection
      title="Security Configuration"
      description="Authentication, authorization, and security policies"
      icon={Shield}
    >
      <div className="space-y-6">
        {/* Login Security */}
        <div className="space-y-4">
          <h4 className="text-sm font-semibold">Login Security</h4>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label htmlFor="login-attempts">Max Login Attempts</Label>
              <Input
                id="login-attempts"
                type="number"
                value={config.security.loginAttempts}
                onChange={e =>
                  onUpdate({
                    security: {
                      ...config.security,
                      loginAttempts: parseInt(e.target.value) || 5,
                    },
                  })
                }
                min="1"
                max="10"
              />
            </div>

            <div>
              <Label htmlFor="session-timeout">Session Timeout (minutes)</Label>
              <Input
                id="session-timeout"
                type="number"
                value={config.security.sessionTimeout}
                onChange={e =>
                  onUpdate({
                    security: {
                      ...config.security,
                      sessionTimeout: parseInt(e.target.value) || 60,
                    },
                  })
                }
                min="5"
                max="480"
              />
            </div>
          </div>
        </div>

        {/* Password Policy */}
        <div className="space-y-4">
          <h4 className="text-sm font-semibold">Password Policy</h4>

          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <div>
                <Label className="text-sm">Minimum Password Length</Label>
                <p className="text-xs text-muted-foreground">
                  Enforce minimum character count
                </p>
              </div>
              <Input
                type="number"
                value={config.security.passwordPolicy.minLength}
                onChange={e =>
                  onUpdate({
                    security: {
                      ...config.security,
                      passwordPolicy: {
                        ...config.security.passwordPolicy,
                        minLength: parseInt(e.target.value) || 8,
                      },
                    },
                  })
                }
                className="w-20"
                min="6"
                max="32"
              />
            </div>

            <div className="flex items-center justify-between">
              <div>
                <Label className="text-sm">Require Special Characters</Label>
                <p className="text-xs text-muted-foreground">
                  Passwords must contain special characters
                </p>
              </div>
              <Switch
                checked={config.security.passwordPolicy.requireSpecialChars}
                onCheckedChange={checked =>
                  onUpdate({
                    security: {
                      ...config.security,
                      passwordPolicy: {
                        ...config.security.passwordPolicy,
                        requireSpecialChars: checked,
                      },
                    },
                  })
                }
              />
            </div>

            <div className="flex items-center justify-between">
              <div>
                <Label className="text-sm">Require Numbers</Label>
                <p className="text-xs text-muted-foreground">
                  Passwords must contain numeric characters
                </p>
              </div>
              <Switch
                checked={config.security.passwordPolicy.requireNumbers}
                onCheckedChange={checked =>
                  onUpdate({
                    security: {
                      ...config.security,
                      passwordPolicy: {
                        ...config.security.passwordPolicy,
                        requireNumbers: checked,
                      },
                    },
                  })
                }
              />
            </div>
          </div>
        </div>
      </div>
    </SettingSection>
  );
}

function FeatureSettings({
  config,
  onUpdate,
}: {
  config: SystemConfig;
  onUpdate: (updates: Partial<SystemConfig>) => void;
}) {
  return (
    <SettingSection
      title="Feature Toggles"
      description="Enable or disable system features and capabilities"
      icon={Server}
    >
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <Label className="text-sm font-medium">User Registration</Label>
            <p className="text-xs text-muted-foreground">
              Allow new users to register accounts
            </p>
          </div>
          <Switch
            checked={config.features.registration}
            onCheckedChange={checked =>
              onUpdate({
                features: {
                  ...config.features,
                  registration: checked,
                },
              })
            }
          />
        </div>

        <div className="flex items-center justify-between">
          <div>
            <Label className="text-sm font-medium">Guest Access</Label>
            <p className="text-xs text-muted-foreground">
              Allow limited access without authentication
            </p>
          </div>
          <Switch
            checked={config.features.guestAccess}
            onCheckedChange={checked =>
              onUpdate({
                features: {
                  ...config.features,
                  guestAccess: checked,
                },
              })
            }
          />
        </div>

        <div className="flex items-center justify-between">
          <div>
            <Label className="text-sm font-medium">Real-time Updates</Label>
            <p className="text-xs text-muted-foreground">
              Enable WebSocket connections for live updates
            </p>
          </div>
          <Switch
            checked={config.features.realTimeUpdates}
            onCheckedChange={checked =>
              onUpdate({
                features: {
                  ...config.features,
                  realTimeUpdates: checked,
                },
              })
            }
          />
        </div>

        <div className="flex items-center justify-between">
          <div>
            <Label className="text-sm font-medium">Audit Logging</Label>
            <p className="text-xs text-muted-foreground">
              Log all user actions for security auditing
            </p>
          </div>
          <Switch
            checked={config.features.auditLogging}
            onCheckedChange={checked =>
              onUpdate({
                features: {
                  ...config.features,
                  auditLogging: checked,
                },
              })
            }
          />
        </div>
      </div>
    </SettingSection>
  );
}

function SystemLimits({
  config,
  onUpdate,
}: {
  config: SystemConfig;
  onUpdate: (updates: Partial<SystemConfig>) => void;
}) {
  return (
    <SettingSection
      title="System Limits"
      description="Configure resource limits and capacity constraints"
      icon={Database}
    >
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div>
          <Label htmlFor="max-users">Maximum Users</Label>
          <Input
            id="max-users"
            type="number"
            value={config.limits.maxUsers}
            onChange={e =>
              onUpdate({
                limits: {
                  ...config.limits,
                  maxUsers: parseInt(e.target.value) || 100,
                },
              })
            }
            min="1"
            max="10000"
          />
          <p className="text-xs text-muted-foreground mt-1">
            Total allowed user accounts
          </p>
        </div>

        <div>
          <Label htmlFor="max-servers">Maximum Servers</Label>
          <Input
            id="max-servers"
            type="number"
            value={config.limits.maxServers}
            onChange={e =>
              onUpdate({
                limits: {
                  ...config.limits,
                  maxServers: parseInt(e.target.value) || 50,
                },
              })
            }
            min="1"
            max="1000"
          />
          <p className="text-xs text-muted-foreground mt-1">
            Maximum MCP servers per user
          </p>
        </div>

        <div>
          <Label htmlFor="max-sessions">Maximum Concurrent Sessions</Label>
          <Input
            id="max-sessions"
            type="number"
            value={config.limits.maxSessions}
            onChange={e =>
              onUpdate({
                limits: {
                  ...config.limits,
                  maxSessions: parseInt(e.target.value) || 500,
                },
              })
            }
            min="10"
            max="5000"
          />
          <p className="text-xs text-muted-foreground mt-1">
            Concurrent user sessions
          </p>
        </div>
      </div>
    </SettingSection>
  );
}

function SystemOperations() {
  const [showBackupDialog, setShowBackupDialog] = useState(false);
  const [showRestartDialog, setShowRestartDialog] = useState(false);
  const [showCleanupDialog, setShowCleanupDialog] = useState(false);

  const systemOperation = useSystemOperation();

  const handleOperation = (operation: 'restart' | 'backup' | 'cleanup') => {
    systemOperation.mutate(operation, {
      onSuccess: () => {
        setShowBackupDialog(false);
        setShowRestartDialog(false);
        setShowCleanupDialog(false);
      },
    });
  };

  return (
    <SettingSection
      title="System Operations"
      description="Perform system maintenance and administrative tasks"
      icon={Cog}
    >
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card className="border-blue-200 bg-blue-50">
          <CardContent className="pt-6">
            <div className="flex items-center space-x-2 mb-2">
              <Database className="h-5 w-5 text-blue-600" />
              <h4 className="font-medium">System Backup</h4>
            </div>
            <p className="text-sm text-muted-foreground mb-4">
              Create a complete system backup including database and
              configurations
            </p>
            <Button
              variant="outline"
              onClick={() => setShowBackupDialog(true)}
              className="w-full"
            >
              Create Backup
            </Button>
          </CardContent>
        </Card>

        <Card className="border-amber-200 bg-amber-50">
          <CardContent className="pt-6">
            <div className="flex items-center space-x-2 mb-2">
              <RefreshCw className="h-5 w-5 text-amber-600" />
              <h4 className="font-medium">System Restart</h4>
            </div>
            <p className="text-sm text-muted-foreground mb-4">
              Restart all system services and containers
            </p>
            <Button
              variant="outline"
              onClick={() => setShowRestartDialog(true)}
              className="w-full"
            >
              Restart System
            </Button>
          </CardContent>
        </Card>

        <Card className="border-green-200 bg-green-50">
          <CardContent className="pt-6">
            <div className="flex items-center space-x-2 mb-2">
              <Clock className="h-5 w-5 text-green-600" />
              <h4 className="font-medium">System Cleanup</h4>
            </div>
            <p className="text-sm text-muted-foreground mb-4">
              Clean temporary files, logs, and unused resources
            </p>
            <Button
              variant="outline"
              onClick={() => setShowCleanupDialog(true)}
              className="w-full"
            >
              Run Cleanup
            </Button>
          </CardContent>
        </Card>
      </div>

      {/* Confirmation Dialogs */}
      <ConfirmationDialog
        isOpen={showBackupDialog}
        onClose={() => setShowBackupDialog(false)}
        onConfirm={() => handleOperation('backup')}
        title="Create System Backup"
        description="This will create a full system backup. The process may take several minutes."
        confirmText="Create Backup"
        variant="default"
        isLoading={systemOperation.isPending}
      />

      <ConfirmationDialog
        isOpen={showRestartDialog}
        onClose={() => setShowRestartDialog(false)}
        onConfirm={() => handleOperation('restart')}
        title="Restart System"
        description="This will restart all system services. Users will be temporarily disconnected."
        confirmText="Restart System"
        variant="destructive"
        isLoading={systemOperation.isPending}
      />

      <ConfirmationDialog
        isOpen={showCleanupDialog}
        onClose={() => setShowCleanupDialog(false)}
        onConfirm={() => handleOperation('cleanup')}
        title="Run System Cleanup"
        description="This will clean temporary files and optimize system performance."
        confirmText="Run Cleanup"
        variant="default"
        isLoading={systemOperation.isPending}
      />
    </SettingSection>
  );
}

function LoadingSkeleton() {
  return (
    <div className="space-y-6">
      {Array.from({ length: 4 }).map((_, i) => (
        <Card key={i}>
          <CardHeader>
            <div className="flex items-center space-x-2">
              <Skeleton className="h-5 w-5" />
              <div>
                <Skeleton className="h-5 w-32" />
                <Skeleton className="h-3 w-48 mt-1" />
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {Array.from({ length: 3 }).map((_, j) => (
                <div key={j} className="flex justify-between items-center">
                  <Skeleton className="h-4 w-40" />
                  <Skeleton className="h-6 w-12" />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

export default function SystemSettings() {
  const { data: config, isLoading, error, refetch } = useSystemConfig();
  const updateConfig = useUpdateSystemConfig();

  const [localConfig, setLocalConfig] = useState<SystemConfig | null>(null);
  const [hasChanges, setHasChanges] = useState(false);

  // Update local config when data loads
  if (config && !localConfig) {
    setLocalConfig(config);
  }

  const handleUpdate = (updates: Partial<SystemConfig>) => {
    if (!localConfig) return;

    const newConfig = {
      ...localConfig,
      ...updates,
    };

    setLocalConfig(newConfig);
    setHasChanges(true);
  };

  const handleSave = () => {
    if (!localConfig) return;

    updateConfig.mutate(localConfig, {
      onSuccess: () => {
        setHasChanges(false);
        refetch();
      },
    });
  };

  const handleReset = () => {
    if (config) {
      setLocalConfig(config);
      setHasChanges(false);
    }
  };

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (error || !config || !localConfig) {
    return (
      <Card>
        <CardContent className="pt-6">
          <div className="text-center">
            <AlertTriangle className="h-16 w-16 text-amber-500 mx-auto mb-4" />
            <h3 className="text-lg font-semibold mb-2">
              Failed to Load Settings
            </h3>
            <p className="text-muted-foreground mb-4">
              {error?.message || 'Unknown error occurred'}
            </p>
            <Button onClick={() => refetch()}>
              <RefreshCw className="h-4 w-4 mr-2" />
              Retry
            </Button>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">System Settings</h2>
          <p className="text-muted-foreground">
            Configure global system settings and preferences
          </p>
        </div>

        {hasChanges && (
          <div className="flex items-center space-x-2">
            <Badge variant="outline" className="text-amber-600 bg-amber-50">
              Unsaved Changes
            </Badge>
            <Button variant="outline" onClick={handleReset}>
              Reset
            </Button>
            <Button onClick={handleSave} disabled={updateConfig.isPending}>
              <Save className="h-4 w-4 mr-2" />
              {updateConfig.isPending ? 'Saving...' : 'Save Changes'}
            </Button>
          </div>
        )}
      </div>

      {/* Settings Tabs */}
      <Tabs defaultValue="general" className="space-y-6">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="general">General</TabsTrigger>
          <TabsTrigger value="security">Security</TabsTrigger>
          <TabsTrigger value="features">Features</TabsTrigger>
          <TabsTrigger value="operations">Operations</TabsTrigger>
        </TabsList>

        <TabsContent value="general" className="space-y-6">
          <MaintenanceSettings config={localConfig} onUpdate={handleUpdate} />
          <SystemLimits config={localConfig} onUpdate={handleUpdate} />
        </TabsContent>

        <TabsContent value="security" className="space-y-6">
          <SecuritySettings config={localConfig} onUpdate={handleUpdate} />
        </TabsContent>

        <TabsContent value="features" className="space-y-6">
          <FeatureSettings config={localConfig} onUpdate={handleUpdate} />
        </TabsContent>

        <TabsContent value="operations" className="space-y-6">
          <SystemOperations />
        </TabsContent>
      </Tabs>
    </div>
  );
}
