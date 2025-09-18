import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { gatewayApi } from '@/lib/api-client';
import type { GatewayStartRequest, GatewayStatus } from '@/types/api';

// Query keys
export const gatewayKeys = {
  all: ['gateway'] as const,
  status: () => [...gatewayKeys.all, 'status'] as const,
};

// Custom hook for gateway status
export function useGatewayStatus() {
  return useQuery({
    queryKey: gatewayKeys.status(),
    queryFn: gatewayApi.getStatus,
    staleTime: 10 * 1000, // 10 seconds
    gcTime: 2 * 60 * 1000, // 2 minutes
    refetchInterval: 15 * 1000, // Refetch every 15 seconds
    refetchOnWindowFocus: true,
    retry: (failureCount, error: unknown) => {
      // Don't retry if gateway is not running (expected behavior)
      if (error && typeof error === 'object' && 'response' in error) {
        const axiosError = error as { response?: { status?: number } };
        if (axiosError.response?.status === 503) return false;
      }
      return failureCount < 2;
    },
  });
}

// Custom hook for starting gateway
export function useStartGateway() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (options?: GatewayStartRequest) => {
      await gatewayApi.start(options);
      return options;
    },
    onMutate: async options => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: gatewayKeys.status() });

      // Snapshot previous value
      const previousStatus = queryClient.getQueryData<GatewayStatus>(
        gatewayKeys.status()
      );

      // Optimistically update to starting state
      queryClient.setQueryData<GatewayStatus>(gatewayKeys.status(), old => ({
        ...old,
        running: true,
        port: options?.port || old?.port,
        transport: options?.transport || old?.transport || 'stdio',
        active_servers: options?.servers || old?.active_servers || [],
        uptime: 0,
        connections: 0,
        requests_handled: 0,
        errors: 0,
      }));

      return { previousStatus };
    },
    onError: (error, variables, context) => {
      // Rollback optimistic update
      if (context?.previousStatus) {
        queryClient.setQueryData(gatewayKeys.status(), context.previousStatus);
      }

      console.error('Gateway start failed:', error);
      toast.error('Failed to start gateway');
    },
    onSuccess: options => {
      const message = options?.port
        ? `Gateway started on port ${options.port}`
        : 'Gateway started successfully';
      toast.success(message);
    },
    onSettled: () => {
      // Refetch status after mutation
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: gatewayKeys.status() });
      }, 1000); // Give the gateway a moment to fully start
    },
  });
}

// Custom hook for stopping gateway
export function useStopGateway() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: gatewayApi.stop,
    onMutate: async () => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: gatewayKeys.status() });

      // Snapshot previous value
      const previousStatus = queryClient.getQueryData<GatewayStatus>(
        gatewayKeys.status()
      );

      // Optimistically update to stopped state
      queryClient.setQueryData<GatewayStatus>(gatewayKeys.status(), old => ({
        ...old,
        running: false,
        uptime: 0,
        connections: 0,
        active_servers: [],
      }));

      return { previousStatus };
    },
    onError: (error, variables, context) => {
      // Rollback optimistic update
      if (context?.previousStatus) {
        queryClient.setQueryData(gatewayKeys.status(), context.previousStatus);
      }

      console.error('Gateway stop failed:', error);
      toast.error('Failed to stop gateway');
    },
    onSuccess: () => {
      toast.success('Gateway stopped successfully');
    },
    onSettled: () => {
      // Refetch status after mutation
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: gatewayKeys.status() });
      }, 500);
    },
  });
}

// Custom hook for gateway toggle
export function useGatewayToggle() {
  return useMutation({
    mutationFn: async ({
      running,
      options,
    }: {
      running: boolean;
      options?: GatewayStartRequest;
    }) => {
      if (running) {
        await gatewayApi.start(options);
      } else {
        await gatewayApi.stop();
      }
      return { running, options };
    },
    onSuccess: ({ running, options }) => {
      if (running) {
        const message = options?.port
          ? `Gateway started on port ${options.port}`
          : 'Gateway started successfully';
        toast.success(message);
      } else {
        toast.success('Gateway stopped successfully');
      }
    },
    onError: (error, { running }) => {
      console.error(`Gateway ${running ? 'start' : 'stop'} failed:`, error);
      toast.error(`Failed to ${running ? 'start' : 'stop'} gateway`);
    },
  });
}

// Custom hook for refreshing gateway status
export function useRefreshGatewayStatus() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      await queryClient.invalidateQueries({ queryKey: gatewayKeys.status() });
      return gatewayApi.getStatus();
    },
    onSuccess: () => {
      toast.success('Gateway status refreshed');
    },
    onError: error => {
      console.error('Failed to refresh gateway status:', error);
      toast.error('Failed to refresh gateway status');
    },
  });
}

// Helper hook to check if gateway is healthy
export function useGatewayHealth() {
  const { data: status, isLoading, error } = useGatewayStatus();

  const isHealthy = status?.running && !error;
  const isStarting = status?.running && (status.uptime || 0) < 10; // Less than 10 seconds uptime
  const isUnhealthy = !isLoading && (!status?.running || !!error);

  return {
    isHealthy,
    isStarting,
    isUnhealthy,
    status,
    isLoading,
    error,
  };
}
