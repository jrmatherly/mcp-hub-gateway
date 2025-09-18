'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiClient } from '@/lib/api-client';
import { toast } from '@/lib/toast';
import type { ApiResponse } from '@/types';
import type {
  AdminCatalog,
  CatalogFilter,
  CatalogStats,
  CreateCatalogRequest,
  ResolvedCatalog,
  UpdateCatalogRequest,
  UserCatalogCustomizationRequest,
} from '@/types/catalog';

// ===========================
// Admin Catalog Hooks
// ===========================

export function useAdminCatalogs(filter?: CatalogFilter) {
  return useQuery({
    queryKey: ['admin', 'catalogs', 'base', filter],
    queryFn: async (): Promise<AdminCatalog[]> => {
      const params = new URLSearchParams();
      if (filter?.type?.length) {
        filter.type.forEach(t => params.append('type', t));
      }
      if (filter?.status?.length) {
        filter.status.forEach(s => params.append('status', s));
      }
      if (filter?.search) {
        params.append('search', filter.search);
      }
      if (filter?.is_public !== undefined) {
        params.append('is_public', filter.is_public.toString());
      }
      if (filter?.is_default !== undefined) {
        params.append('is_default', filter.is_default.toString());
      }
      if (filter?.limit) {
        params.append('limit', filter.limit.toString());
      }
      if (filter?.offset) {
        params.append('offset', filter.offset.toString());
      }
      if (filter?.sort_by) {
        params.append('sort_by', filter.sort_by);
      }
      if (filter?.sort_order) {
        params.append('sort_order', filter.sort_order);
      }

      const queryString = params.toString();
      const url = `/admin/catalogs/base${queryString ? `?${queryString}` : ''}`;

      const response = await apiClient.get<ApiResponse<AdminCatalog[]>>(url);
      if (!response.data?.success || !response.data.data) {
        throw new Error(
          response.data?.error || 'Failed to fetch admin catalogs'
        );
      }
      return response.data.data;
    },
    staleTime: 30000, // 30 seconds
  });
}

export function useCreateAdminCatalog() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: CreateCatalogRequest) => {
      const response = await apiClient.post<ApiResponse<AdminCatalog>>(
        '/admin/catalogs/base',
        request
      );
      if (!response.data?.success) {
        throw new Error(response.data?.error || 'Failed to create catalog');
      }
      return response.data.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'catalogs'] });
      toast.success('Admin catalog created successfully');
    },
    onError: error => {
      toast.error('Failed to create admin catalog', {
        description: error.message,
      });
    },
  });
}

export function useUpdateAdminCatalog() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      id,
      request,
    }: {
      id: string;
      request: UpdateCatalogRequest;
    }) => {
      const response = await apiClient.put<ApiResponse<AdminCatalog>>(
        `/admin/catalogs/base/${id}`,
        request
      );
      if (!response.data?.success) {
        throw new Error(response.data?.error || 'Failed to update catalog');
      }
      return response.data.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'catalogs'] });
      toast.success('Admin catalog updated successfully');
    },
    onError: error => {
      toast.error('Failed to update admin catalog', {
        description: error.message,
      });
    },
  });
}

export function useDeleteAdminCatalog() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      const response = await apiClient.delete<ApiResponse<void>>(
        `/admin/catalogs/base/${id}`
      );
      if (!response.data?.success) {
        throw new Error(response.data?.error || 'Failed to delete catalog');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'catalogs'] });
      toast.success('Admin catalog deleted successfully');
    },
    onError: error => {
      toast.error('Failed to delete admin catalog', {
        description: error.message,
      });
    },
  });
}

export function useImportAdminCatalog() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      data,
      format,
    }: {
      data: string;
      format: 'json' | 'yaml';
    }) => {
      const response = await apiClient.post<ApiResponse<AdminCatalog>>(
        `/admin/catalogs/import?format=${format}`,
        data,
        {
          headers: {
            'Content-Type':
              format === 'json' ? 'application/json' : 'application/x-yaml',
          },
        }
      );
      if (!response.data?.success) {
        throw new Error(response.data?.error || 'Failed to import catalog');
      }
      return response.data.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'catalogs'] });
      toast.success('Catalog imported successfully');
    },
    onError: error => {
      toast.error('Failed to import catalog', {
        description: error.message,
      });
    },
  });
}

// ===========================
// User Catalog Hooks
// ===========================

export function useResolvedCatalog() {
  return useQuery({
    queryKey: ['catalogs', 'resolved'],
    queryFn: async (): Promise<ResolvedCatalog> => {
      const response =
        await apiClient.get<ApiResponse<ResolvedCatalog>>('/catalogs/resolved');
      if (!response.data?.success || !response.data.data) {
        throw new Error(
          response.data?.error || 'Failed to fetch resolved catalog'
        );
      }
      return response.data.data;
    },
    staleTime: 60000, // 1 minute
  });
}

export function useCatalogStats() {
  return useQuery({
    queryKey: ['catalogs', 'stats'],
    queryFn: async (): Promise<CatalogStats> => {
      const response =
        await apiClient.get<ApiResponse<CatalogStats>>('/catalogs/stats');
      if (!response.data?.success || !response.data.data) {
        throw new Error(
          response.data?.error || 'Failed to fetch catalog stats'
        );
      }
      return response.data.data;
    },
    refetchInterval: 30000, // 30 seconds
    staleTime: 15000, // 15 seconds
  });
}

export function useUserCatalogCustomizations() {
  return useQuery({
    queryKey: ['catalogs', 'customizations'],
    queryFn: async (): Promise<UserCatalogCustomizationRequest> => {
      const response = await apiClient.get<
        ApiResponse<UserCatalogCustomizationRequest>
      >('/catalogs/customizations');
      if (!response.data?.success || !response.data.data) {
        throw new Error(
          response.data?.error || 'Failed to fetch catalog customizations'
        );
      }
      return response.data.data;
    },
    staleTime: 60000, // 1 minute
  });
}

export function useUpdateUserCatalogCustomizations() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: UserCatalogCustomizationRequest) => {
      const response = await apiClient.put<ApiResponse<void>>(
        '/catalogs/customizations',
        request
      );
      if (!response.data?.success) {
        throw new Error(
          response.data?.error || 'Failed to update catalog customizations'
        );
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['catalogs'] });
      toast.success('Catalog customizations updated successfully');
    },
    onError: error => {
      toast.error('Failed to update catalog customizations', {
        description: error.message,
      });
    },
  });
}

export function useExportCatalog() {
  return useMutation({
    mutationFn: async ({ format }: { format: 'json' | 'yaml' }) => {
      const response = await apiClient.get<ApiResponse<string>>(
        `/catalogs/export?format=${format}`
      );
      if (!response.data?.success || !response.data.data) {
        throw new Error(response.data?.error || 'Failed to export catalog');
      }
      return response.data.data;
    },
    onSuccess: () => {
      toast.success('Catalog exported successfully');
    },
    onError: error => {
      toast.error('Failed to export catalog', {
        description: error.message,
      });
    },
  });
}
