import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { configApi } from '@/lib/api-client';
import type { MCPConfig } from '@/types/api';

// Query keys
export const configKeys = {
  all: ['config'] as const,
  detail: () => [...configKeys.all, 'detail'] as const,
};

// Custom hook for config data
export function useConfig() {
  return useQuery({
    queryKey: configKeys.detail(),
    queryFn: configApi.get,
    staleTime: 2 * 60 * 1000, // 2 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    refetchOnWindowFocus: false, // Config doesn't change often
    retry: 2,
  });
}

// Custom hook for updating config
export function useUpdateConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (config: Partial<MCPConfig>) => {
      await configApi.update(config);
      return config;
    },
    onMutate: async newConfig => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: configKeys.detail() });

      // Snapshot previous value
      const previousConfig = queryClient.getQueryData<MCPConfig>(
        configKeys.detail()
      );

      // Optimistically update config
      if (previousConfig) {
        queryClient.setQueryData<MCPConfig>(configKeys.detail(), old => ({
          ...old,
          ...newConfig,
          // Deep merge for nested objects
          gateway: {
            ...old?.gateway,
            ...newConfig.gateway,
          },
          servers: {
            ...old?.servers,
            ...newConfig.servers,
          },
          secrets: {
            ...old?.secrets,
            ...newConfig.secrets,
          },
          catalog: {
            ...old?.catalog,
            ...newConfig.catalog,
          },
        }));
      }

      return { previousConfig };
    },
    onError: (error, variables, context) => {
      // Rollback optimistic update
      if (context?.previousConfig) {
        queryClient.setQueryData(configKeys.detail(), context.previousConfig);
      }

      console.error('Config update failed:', error);
      toast.error('Failed to update configuration');
    },
    onSuccess: () => {
      toast.success('Configuration updated successfully');
    },
    onSettled: () => {
      // Refetch config data after mutation
      queryClient.invalidateQueries({ queryKey: configKeys.detail() });

      // Also invalidate servers since config changes might affect server status
      queryClient.invalidateQueries({ queryKey: ['servers'] });
    },
  });
}

// Custom hook for updating gateway config specifically
export function useUpdateGatewayConfig() {
  const updateConfig = useUpdateConfig();

  return useMutation({
    mutationFn: async (gatewayConfig: NonNullable<MCPConfig['gateway']>) => {
      return updateConfig.mutateAsync({ gateway: gatewayConfig });
    },
    onSuccess: () => {
      toast.success('Gateway configuration updated');
    },
    onError: error => {
      console.error('Gateway config update failed:', error);
      toast.error('Failed to update gateway configuration');
    },
  });
}

// Custom hook for updating server config
export function useUpdateServerConfig() {
  const updateConfig = useUpdateConfig();

  return useMutation({
    mutationFn: async ({
      serverName,
      serverConfig,
    }: {
      serverName: string;
      serverConfig: NonNullable<MCPConfig['servers']>[string];
    }) => {
      return updateConfig.mutateAsync({
        servers: {
          [serverName]: serverConfig,
        },
      });
    },
    onSuccess: (_, { serverName }) => {
      toast.success(`Server ${serverName} configuration updated`);
    },
    onError: (error, { serverName }) => {
      console.error(`Server ${serverName} config update failed:`, error);
      toast.error(`Failed to update ${serverName} configuration`);
    },
  });
}

// Custom hook for batch updating server configs
export function useBatchUpdateServerConfigs() {
  const updateConfig = useUpdateConfig();

  return useMutation({
    mutationFn: async (serversConfig: NonNullable<MCPConfig['servers']>) => {
      return updateConfig.mutateAsync({ servers: serversConfig });
    },
    onSuccess: (_, serversConfig) => {
      const serverCount = Object.keys(serversConfig).length;
      toast.success(`Updated configuration for ${serverCount} server(s)`);
    },
    onError: error => {
      console.error('Batch server config update failed:', error);
      toast.error('Failed to update server configurations');
    },
  });
}

// Custom hook for resetting config
export function useResetConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      // Reset to default config
      const defaultConfig: MCPConfig = {
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

      await configApi.update(defaultConfig);
      return defaultConfig;
    },
    onSuccess: () => {
      toast.success('Configuration reset to defaults');
      // Invalidate all related queries
      queryClient.invalidateQueries({ queryKey: configKeys.all });
      queryClient.invalidateQueries({ queryKey: ['servers'] });
      queryClient.invalidateQueries({ queryKey: ['gateway'] });
    },
    onError: error => {
      console.error('Config reset failed:', error);
      toast.error('Failed to reset configuration');
    },
  });
}

// Custom hook for exporting config
export function useExportConfig() {
  const { data: config } = useConfig();

  return useMutation({
    mutationFn: async () => {
      if (!config) {
        throw new Error('No configuration data available');
      }

      // Create downloadable JSON file
      const configBlob = new Blob([JSON.stringify(config, null, 2)], {
        type: 'application/json',
      });

      const url = URL.createObjectURL(configBlob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `mcp-config-${new Date().toISOString().split('T')[0]}.json`;

      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);

      URL.revokeObjectURL(url);

      return config;
    },
    onSuccess: () => {
      toast.success('Configuration exported successfully');
    },
    onError: error => {
      console.error('Config export failed:', error);
      toast.error('Failed to export configuration');
    },
  });
}

// Custom hook for importing config
export function useImportConfig() {
  const updateConfig = useUpdateConfig();

  return useMutation({
    mutationFn: async (file: File) => {
      return new Promise<MCPConfig>((resolve, reject) => {
        const reader = new FileReader();

        reader.onload = async event => {
          try {
            const configData = JSON.parse(event.target?.result as string);
            await updateConfig.mutateAsync(configData);
            resolve(configData);
          } catch (_error) {
            reject(new Error('Invalid configuration file format'));
          }
        };

        reader.onerror = () => {
          reject(new Error('Failed to read configuration file'));
        };

        reader.readAsText(file);
      });
    },
    onSuccess: () => {
      toast.success('Configuration imported successfully');
    },
    onError: error => {
      console.error('Config import failed:', error);
      toast.error('Failed to import configuration');
    },
  });
}

// Helper hooks
export function useGatewayConfig() {
  const { data: config } = useConfig();
  return config?.gateway || {};
}

export function useServerConfigs() {
  const { data: config } = useConfig();
  return config?.servers || {};
}

export function useServerConfig(serverName: string) {
  const serverConfigs = useServerConfigs();
  return serverConfigs[serverName];
}

export function useCatalogConfig() {
  const { data: config } = useConfig();
  return config?.catalog || {};
}
