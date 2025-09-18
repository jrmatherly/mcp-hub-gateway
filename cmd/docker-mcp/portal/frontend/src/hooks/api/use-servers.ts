import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { serverApi } from '@/lib/api-client';
import type {
  BulkServerOperation,
  Server,
  ServerDetails,
  ServerFilters,
} from '@/types/api';

// Query keys
export const serverKeys = {
  all: ['servers'] as const,
  lists: () => [...serverKeys.all, 'list'] as const,
  list: (filters?: ServerFilters) => [...serverKeys.lists(), filters] as const,
  details: () => [...serverKeys.all, 'details'] as const,
  detail: (name: string) => [...serverKeys.details(), name] as const,
};

// Custom hook for servers list
export function useServers(filters?: ServerFilters) {
  return useQuery({
    queryKey: serverKeys.list(filters),
    queryFn: async () => {
      const servers = await serverApi.getServers();

      // Apply client-side filters if provided
      if (!filters) return servers;

      return servers.filter(server => {
        // Status filter
        if (filters.status && filters.status.length > 0) {
          if (!filters.status.includes(server.status)) return false;
        }

        // Enabled filter
        if (filters.enabled !== undefined) {
          if (server.enabled !== filters.enabled) return false;
        }

        // Health status filter
        if (filters.health_status && filters.health_status.length > 0) {
          if (
            !server.health_status ||
            !filters.health_status.includes(server.health_status)
          ) {
            return false;
          }
        }

        // Tags filter
        if (filters.tags && filters.tags.length > 0) {
          if (
            !server.tags ||
            !filters.tags.some(tag => server.tags?.includes(tag))
          ) {
            return false;
          }
        }

        // Search filter
        if (filters.search) {
          const searchLower = filters.search.toLowerCase();
          const searchableText = [
            server.name,
            server.description,
            ...(server.tags || []),
            ...(server.capabilities || []),
          ]
            .join(' ')
            .toLowerCase();

          if (!searchableText.includes(searchLower)) return false;
        }

        return true;
      });
    },
    staleTime: 30 * 1000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
    refetchOnWindowFocus: true,
    refetchInterval: 60 * 1000, // Refetch every minute
  });
}

// Custom hook for server details
export function useServerDetails(name: string, enabled = true) {
  return useQuery({
    queryKey: serverKeys.detail(name),
    queryFn: () => serverApi.getServerDetails(name),
    enabled: enabled && !!name,
    staleTime: 10 * 1000, // 10 seconds
    gcTime: 2 * 60 * 1000, // 2 minutes
    retry: (failureCount, error: unknown) => {
      // Don't retry on 404 errors
      if (error && typeof error === 'object' && 'response' in error) {
        const axiosError = error as { response?: { status?: number } };
        if (axiosError.response?.status === 404) return false;
      }
      return failureCount < 3;
    },
  });
}

// Custom hook for server toggle (enable/disable)
export function useServerToggle() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      name,
      enabled,
    }: {
      name: string;
      enabled: boolean;
    }) => {
      await serverApi.toggleServer(name, enabled);
      return { name, enabled };
    },
    onMutate: async ({ name, enabled }) => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: serverKeys.all });

      // Snapshot previous value
      const previousServers = queryClient.getQueryData<Server[]>(
        serverKeys.lists()
      );

      // Optimistically update server list
      if (previousServers) {
        queryClient.setQueryData<Server[]>(
          serverKeys.lists(),
          old =>
            old?.map(server =>
              server.name === name
                ? {
                    ...server,
                    enabled,
                    status: enabled
                      ? ('starting' as const)
                      : ('stopped' as const),
                  }
                : server
            ) || []
        );
      }

      // Optimistically update server details if it exists
      const serverDetail = queryClient.getQueryData<ServerDetails>(
        serverKeys.detail(name)
      );
      if (serverDetail) {
        queryClient.setQueryData<ServerDetails>(serverKeys.detail(name), {
          ...serverDetail,
          enabled,
          status: enabled ? ('starting' as const) : ('stopped' as const),
        });
      }

      return { previousServers };
    },
    onError: (error, variables, context) => {
      // Rollback optimistic update
      if (context?.previousServers) {
        queryClient.setQueryData(serverKeys.lists(), context.previousServers);
      }

      console.error('Server toggle failed:', error);
      toast.error(
        `Failed to ${variables.enabled ? 'enable' : 'disable'} server ${variables.name}`
      );
    },
    onSuccess: ({ name, enabled }) => {
      toast.success(
        `Server ${name} ${enabled ? 'enabled' : 'disabled'} successfully`
      );
    },
    onSettled: () => {
      // Refetch server data after mutation
      queryClient.invalidateQueries({ queryKey: serverKeys.all });
    },
  });
}

// Custom hook for bulk server operations
export function useBulkServerOperation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (operation: BulkServerOperation) =>
      serverApi.bulkOperation(operation),
    onMutate: async (operation: BulkServerOperation) => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: serverKeys.all });

      // Get current servers
      const previousServers = queryClient.getQueryData<Server[]>(
        serverKeys.lists()
      );

      // Optimistically update servers based on operation
      if (
        previousServers &&
        (operation.operation === 'enable' || operation.operation === 'disable')
      ) {
        const enabled = operation.operation === 'enable';
        queryClient.setQueryData<Server[]>(
          serverKeys.lists(),
          old =>
            old?.map(server =>
              operation.server_names.includes(server.name)
                ? {
                    ...server,
                    enabled,
                    status: enabled
                      ? ('starting' as const)
                      : ('stopped' as const),
                  }
                : server
            ) || []
        );
      }

      return { previousServers };
    },
    onError: (error, variables, context) => {
      // Rollback optimistic update
      if (context?.previousServers) {
        queryClient.setQueryData(serverKeys.lists(), context.previousServers);
      }

      console.error('Bulk operation failed:', error);
      toast.error(`Bulk ${variables.operation} operation failed`);
    },
    onSuccess: (result, variables) => {
      const { success_count, error_count, results } = result;

      if (error_count === 0) {
        toast.success(
          `Successfully ${variables.operation}d ${success_count} server(s)`
        );
      } else {
        toast.warning(
          `${variables.operation} completed with ${success_count} success(es) and ${error_count} error(s)`
        );

        // Show individual errors
        results
          .filter(r => !r.success)
          .forEach(r => {
            toast.error(`${r.server_name}: ${r.error || 'Unknown error'}`);
          });
      }
    },
    onSettled: () => {
      // Refetch server data after bulk operation
      queryClient.invalidateQueries({ queryKey: serverKeys.all });
    },
  });
}

// Custom hook for refreshing servers
export function useRefreshServers() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      await queryClient.invalidateQueries({ queryKey: serverKeys.all });
      return serverApi.getServers();
    },
    onSuccess: () => {
      toast.success('Server list refreshed');
    },
    onError: error => {
      console.error('Failed to refresh servers:', error);
      toast.error('Failed to refresh server list');
    },
  });
}

// Custom hook for server installation from catalog
export function useInstallServer() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      serverName: _serverName,
      catalogName: _catalogName,
    }: {
      serverName: string;
      catalogName?: string;
    }) => {
      // This would be implemented when catalog API is available
      // await catalogApi.installServer(serverName, catalogName);
      throw new Error('Server installation not yet implemented');
    },
    onSuccess: (_, { serverName }) => {
      toast.success(`Server ${serverName} installed successfully`);
      // Refresh servers list
      queryClient.invalidateQueries({ queryKey: serverKeys.all });
    },
    onError: (error, { serverName }) => {
      console.error('Server installation failed:', error);
      toast.error(`Failed to install server ${serverName}`);
    },
  });
}

// Helper hook to get server by name
export function useServer(name: string) {
  const { data: servers = [] } = useServers();
  return servers.find(server => server.name === name);
}
