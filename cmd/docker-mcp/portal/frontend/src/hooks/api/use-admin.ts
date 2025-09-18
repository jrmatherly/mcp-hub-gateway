'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiClient } from '@/lib/api-client';
import { toast } from '@/lib/toast';
import type { User, LogEntry, ApiResponse } from '@/types';

// ===========================
// Types
// ===========================

export interface SystemStats {
  users: {
    total: number;
    active: number;
    admins: number;
    lastHour: number;
  };
  sessions: {
    active: number;
    total: number;
    avgDuration: number;
  };
  servers: {
    total: number;
    enabled: number;
    running: number;
    errors: number;
  };
  system: {
    uptime: number;
    cpuUsage: number;
    memoryUsage: number;
    diskUsage: number;
  };
  security: {
    failedLogins: number;
    blockedIPs: number;
    activeThreats: number;
  };
}

export interface AuditLog {
  id: string;
  timestamp: Date;
  userId: string;
  userName: string;
  action: string;
  resource: string;
  resourceId?: string;
  ipAddress: string;
  userAgent: string;
  success: boolean;
  errorMessage?: string;
  metadata?: Record<string, unknown>;
}

export interface SystemHealth {
  status: 'healthy' | 'warning' | 'critical';
  components: {
    database: 'healthy' | 'warning' | 'critical';
    redis: 'healthy' | 'warning' | 'critical';
    docker: 'healthy' | 'warning' | 'critical';
    auth: 'healthy' | 'warning' | 'critical';
  };
  metrics: {
    responseTime: number;
    errorRate: number;
    throughput: number;
  };
  lastCheck: Date;
}

export interface SystemConfig {
  maintenance: {
    enabled: boolean;
    message?: string;
    scheduledAt?: Date;
  };
  security: {
    loginAttempts: number;
    sessionTimeout: number;
    passwordPolicy: {
      minLength: number;
      requireSpecialChars: boolean;
      requireNumbers: boolean;
    };
  };
  features: {
    registration: boolean;
    guestAccess: boolean;
    realTimeUpdates: boolean;
    auditLogging: boolean;
  };
  limits: {
    maxUsers: number;
    maxServers: number;
    maxSessions: number;
  };
}

export interface UserManagementData extends User {
  lastActivity?: Date;
  sessionCount: number;
  serverCount: number;
  auditLogCount: number;
  active: boolean;
  role: string; // Primary role for display purposes
}

// ===========================
// System Stats
// ===========================

export function useSystemStats(options: { refetchInterval?: number } = {}) {
  return useQuery({
    queryKey: ['admin', 'stats'],
    queryFn: async (): Promise<SystemStats> => {
      const response =
        await apiClient.get<ApiResponse<SystemStats>>('/admin/stats');
      if (!response.data?.success || !response.data.data) {
        throw new Error(response.data?.error || 'Failed to fetch system stats');
      }
      return response.data.data;
    },
    refetchInterval: options.refetchInterval || 30000, // 30 seconds
    staleTime: 15000, // 15 seconds
  });
}

// ===========================
// System Health
// ===========================

export function useSystemHealth(options: { refetchInterval?: number } = {}) {
  return useQuery({
    queryKey: ['admin', 'health'],
    queryFn: async (): Promise<SystemHealth> => {
      const response =
        await apiClient.get<ApiResponse<SystemHealth>>('/admin/health');
      if (!response.data?.success || !response.data.data) {
        throw new Error(
          response.data?.error || 'Failed to fetch system health'
        );
      }
      return response.data.data;
    },
    refetchInterval: options.refetchInterval || 10000, // 10 seconds
    staleTime: 5000, // 5 seconds
  });
}

// ===========================
// User Management
// ===========================

export function useAdminUsers() {
  return useQuery({
    queryKey: ['admin', 'users'],
    queryFn: async (): Promise<UserManagementData[]> => {
      const response =
        await apiClient.get<ApiResponse<UserManagementData[]>>('/admin/users');
      if (!response.data?.success || !response.data.data) {
        throw new Error(response.data?.error || 'Failed to fetch users');
      }
      return response.data.data;
    },
    staleTime: 30000, // 30 seconds
  });
}

export function useUpdateUserRole() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ userId, role }: { userId: string; role: string }) => {
      const response = await apiClient.patch<ApiResponse<User>>(
        `/admin/users/${userId}/role`,
        { role }
      );

      if (!response.data?.success) {
        throw new Error(response.data?.error || 'Failed to update user role');
      }
      return response.data.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] });
      queryClient.invalidateQueries({ queryKey: ['admin', 'stats'] });
      toast.success('User role updated successfully');
    },
    onError: error => {
      toast.error('Failed to update user role', {
        description: error.message,
      });
    },
  });
}

export function useToggleUserStatus() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      userId,
      active,
    }: {
      userId: string;
      active: boolean;
    }) => {
      const response = await apiClient.patch<ApiResponse<User>>(
        `/admin/users/${userId}/status`,
        { active }
      );

      if (!response.data?.success) {
        throw new Error(response.data?.error || 'Failed to update user status');
      }
      return response.data.data;
    },
    onSuccess: (_, { active }) => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] });
      queryClient.invalidateQueries({ queryKey: ['admin', 'stats'] });
      toast.success(
        `User ${active ? 'activated' : 'deactivated'} successfully`
      );
    },
    onError: error => {
      toast.error('Failed to update user status', {
        description: error.message,
      });
    },
  });
}

export function useDeleteUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (userId: string) => {
      const response = await apiClient.delete<ApiResponse<void>>(
        `/admin/users/${userId}`
      );

      if (!response.data?.success) {
        throw new Error(response.data?.error || 'Failed to delete user');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] });
      queryClient.invalidateQueries({ queryKey: ['admin', 'stats'] });
      toast.success('User deleted successfully');
    },
    onError: error => {
      toast.error('Failed to delete user', {
        description: error.message,
      });
    },
  });
}

// ===========================
// Audit Logs
// ===========================

export function useAuditLogs(
  options: {
    page?: number;
    limit?: number;
    userId?: string;
    action?: string;
    dateFrom?: Date;
    dateTo?: Date;
  } = {}
) {
  return useQuery({
    queryKey: ['admin', 'audit-logs', options],
    queryFn: async (): Promise<{
      logs: AuditLog[];
      total: number;
      pages: number;
    }> => {
      const params = new URLSearchParams();
      if (options.page) params.append('page', options.page.toString());
      if (options.limit) params.append('limit', options.limit.toString());
      if (options.userId) params.append('userId', options.userId);
      if (options.action) params.append('action', options.action);
      if (options.dateFrom)
        params.append('dateFrom', options.dateFrom.toISOString());
      if (options.dateTo) params.append('dateTo', options.dateTo.toISOString());

      const response = await apiClient.get<
        ApiResponse<{
          logs: AuditLog[];
          total: number;
          pages: number;
        }>
      >(`/admin/audit-logs?${params.toString()}`);

      if (!response.data?.success || !response.data.data) {
        throw new Error(response.data?.error || 'Failed to fetch audit logs');
      }
      return response.data.data;
    },
    staleTime: 60000, // 1 minute
  });
}

// ===========================
// System Configuration
// ===========================

export function useSystemConfig() {
  return useQuery({
    queryKey: ['admin', 'config'],
    queryFn: async (): Promise<SystemConfig> => {
      const response =
        await apiClient.get<ApiResponse<SystemConfig>>('/admin/config');
      if (!response.data?.success || !response.data.data) {
        throw new Error(
          response.data?.error || 'Failed to fetch system config'
        );
      }
      return response.data.data;
    },
    staleTime: 300000, // 5 minutes
  });
}

export function useUpdateSystemConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (config: Partial<SystemConfig>) => {
      const response = await apiClient.patch<ApiResponse<SystemConfig>>(
        '/admin/config',
        config
      );

      if (!response.data?.success) {
        throw new Error(
          response.data?.error || 'Failed to update system config'
        );
      }
      return response.data.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'config'] });
      toast.success('System configuration updated successfully');
    },
    onError: error => {
      toast.error('Failed to update system configuration', {
        description: error.message,
      });
    },
  });
}

// ===========================
// Session Management
// ===========================

export function useUserSessions(userId?: string) {
  return useQuery({
    queryKey: ['admin', 'sessions', userId],
    queryFn: async (): Promise<unknown[]> => {
      const url = userId
        ? `/admin/sessions?userId=${userId}`
        : '/admin/sessions';
      const response = await apiClient.get<ApiResponse<unknown[]>>(url);
      if (!response.data?.success || !response.data.data) {
        throw new Error(response.data?.error || 'Failed to fetch sessions');
      }
      return response.data.data;
    },
    enabled: !!userId,
    staleTime: 30000, // 30 seconds
  });
}

export function useTerminateSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (sessionId: string) => {
      const response = await apiClient.delete<ApiResponse<void>>(
        `/admin/sessions/${sessionId}`
      );

      if (!response.data?.success) {
        throw new Error(response.data?.error || 'Failed to terminate session');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'sessions'] });
      queryClient.invalidateQueries({ queryKey: ['admin', 'stats'] });
      toast.success('Session terminated successfully');
    },
    onError: error => {
      toast.error('Failed to terminate session', {
        description: error.message,
      });
    },
  });
}

// ===========================
// System Operations
// ===========================

export function useSystemOperation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (
      operation: 'restart' | 'backup' | 'cleanup' | 'maintenance'
    ) => {
      const response = await apiClient.post<ApiResponse<{ message: string }>>(
        `/admin/system/${operation}`
      );

      if (!response.data?.success) {
        throw new Error(
          response.data?.error || `Failed to perform ${operation}`
        );
      }
      return response.data.data;
    },
    onSuccess: (data, operation) => {
      queryClient.invalidateQueries({ queryKey: ['admin'] });
      toast.success(`System ${operation} initiated successfully`, {
        description: data?.message,
      });
    },
    onError: (error, operation) => {
      toast.error(`Failed to perform ${operation}`, {
        description: error.message,
      });
    },
  });
}

// ===========================
// Real-time System Logs
// ===========================

export function useSystemLogs(
  options: {
    level?: 'debug' | 'info' | 'warn' | 'error';
    limit?: number;
    follow?: boolean;
  } = {}
) {
  return useQuery({
    queryKey: ['admin', 'logs', options],
    queryFn: async (): Promise<LogEntry[]> => {
      const params = new URLSearchParams();
      if (options.level) params.append('level', options.level);
      if (options.limit) params.append('limit', options.limit.toString());

      const response = await apiClient.get<ApiResponse<LogEntry[]>>(
        `/admin/logs?${params.toString()}`
      );

      if (!response.data?.success || !response.data.data) {
        throw new Error(response.data?.error || 'Failed to fetch system logs');
      }
      return response.data.data;
    },
    refetchInterval: options.follow ? 5000 : false, // 5 seconds if following
    staleTime: 1000, // 1 second
  });
}
