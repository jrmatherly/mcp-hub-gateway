import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { configApi } from '@/lib/api-client';
import type { MCPConfig } from '@/types/api';

// Configuration query key
const CONFIG_QUERY_KEY = ['config'] as const;

// Get current configuration
export function useConfig() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: CONFIG_QUERY_KEY,
    queryFn: configApi.get,
    staleTime: 1000 * 60 * 5, // 5 minutes
    retry: 2,
  });

  // Update configuration
  const updateConfig = useMutation({
    mutationFn: (config: Partial<MCPConfig>) => configApi.update(config),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: CONFIG_QUERY_KEY });
    },
  });

  // Export configuration
  const exportConfig = useMutation({
    mutationFn: async ({ format }: { format: 'json' | 'yaml' }) => {
      const config = await configApi.get();
      if (format === 'json') {
        return JSON.stringify(config, null, 2);
      } else {
        // For now, return JSON format. In a real implementation,
        // you'd convert to YAML using a library like js-yaml
        return JSON.stringify(config, null, 2);
      }
    },
  });

  // Import configuration
  const importConfig = useMutation({
    mutationFn: async ({
      data,
      format,
    }: {
      data: string;
      format: 'json' | 'yaml';
    }) => {
      let parsedConfig: MCPConfig;

      if (format === 'json') {
        parsedConfig = JSON.parse(data);
      } else {
        // For now, assume JSON format. In a real implementation,
        // you'd parse YAML using a library like js-yaml
        parsedConfig = JSON.parse(data);
      }

      return configApi.update(parsedConfig);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: CONFIG_QUERY_KEY });
    },
  });

  return {
    ...query,
    updateConfig,
    exportConfig,
    importConfig,
  };
}

// Get configuration history (mock implementation)
export function useConfigHistory() {
  return useQuery({
    queryKey: ['config', 'history'],
    queryFn: async () => {
      // Mock implementation - in real app, this would call an API
      return [
        {
          id: 'current',
          name: 'Current Configuration',
          timestamp: new Date().toISOString(),
          config: await configApi.get(),
        },
        {
          id: 'backup-1',
          name: 'Backup - 2024-01-15',
          timestamp: '2024-01-15T10:30:00Z',
          config: await configApi.get(),
        },
      ];
    },
    staleTime: 1000 * 60 * 10, // 10 minutes
  });
}

// Configuration validation
export function useConfigValidation() {
  const validateConfig = useMutation({
    mutationFn: async (config: MCPConfig) => {
      // Mock validation - in real app, this would call a validation API
      const errors = [];

      // Basic validation rules
      if (
        !config.gateway?.port ||
        config.gateway.port < 1 ||
        config.gateway.port > 65535
      ) {
        errors.push('Gateway port must be between 1 and 65535');
      }

      if (config.gateway?.timeout && config.gateway.timeout < 1000) {
        errors.push('Gateway timeout must be at least 1000ms');
      }

      // Validate servers
      if (config.servers) {
        for (const [serverName, serverConfig] of Object.entries(
          config.servers
        )) {
          if (!serverConfig.image) {
            errors.push(`Server ${serverName} must have an image specified`);
          }

          if (
            serverConfig.port &&
            (serverConfig.port < 1 || serverConfig.port > 65535)
          ) {
            errors.push(
              `Server ${serverName} port must be between 1 and 65535`
            );
          }
        }
      }

      return {
        valid: errors.length === 0,
        errors,
      };
    },
  });

  return { validateConfig };
}
