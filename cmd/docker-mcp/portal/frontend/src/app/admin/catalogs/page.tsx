'use client';

import { useState } from 'react';
import {
  Plus,
  Upload,
  Search,
  RefreshCw,
  BookOpen,
  AlertCircle,
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
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';

import { CatalogManagement } from '@/components/admin/CatalogManagement';
import { CatalogEditor } from '@/components/admin/CatalogEditor';
import { CatalogImporter } from '@/components/admin/CatalogImporter';
import {
  useAdminCatalogs,
  useCreateAdminCatalog,
  useImportAdminCatalog,
} from '@/hooks/api/use-catalog';
import type {
  CatalogFilter,
  CatalogType,
  CatalogStatus,
  CreateCatalogRequest,
  UpdateCatalogRequest,
} from '@/types/catalog';

export default function AdminCatalogsPage() {
  // State
  const [searchQuery, setSearchQuery] = useState('');
  const [typeFilter, setTypeFilter] = useState<CatalogType[]>([]);
  const [statusFilter, setStatusFilter] = useState<CatalogStatus[]>([]);
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [isImportDialogOpen, setIsImportDialogOpen] = useState(false);

  // Build filter object
  const filter: CatalogFilter = {
    search: searchQuery || undefined,
    type: typeFilter.length > 0 ? typeFilter : undefined,
    status: statusFilter.length > 0 ? statusFilter : undefined,
    sort_by: 'updated_at',
    sort_order: 'desc',
  };

  // Hooks
  const {
    data: catalogs = [],
    isLoading,
    error,
    refetch,
  } = useAdminCatalogs(filter);

  const createCatalogMutation = useCreateAdminCatalog();
  const importCatalogMutation = useImportAdminCatalog();

  // Handlers
  const handleCreateCatalog = async (
    request: CreateCatalogRequest | UpdateCatalogRequest
  ) => {
    try {
      await createCatalogMutation.mutateAsync(request as CreateCatalogRequest);
      setIsCreateDialogOpen(false);
    } catch (error) {
      // Error is handled by the hook
      console.error('Failed to create catalog:', error);
    }
  };

  const handleImportCatalog = async (data: string, format: 'json' | 'yaml') => {
    try {
      await importCatalogMutation.mutateAsync({ data, format });
      setIsImportDialogOpen(false);
    } catch (error) {
      // Error is handled by the hook
      console.error('Failed to import catalog:', error);
    }
  };

  const handleClearFilters = () => {
    setSearchQuery('');
    setTypeFilter([]);
    setStatusFilter([]);
  };

  const hasActiveFilters =
    searchQuery || typeFilter.length > 0 || statusFilter.length > 0;

  // Loading state
  if (isLoading) {
    return <CatalogLoadingSkeleton />;
  }

  // Error state
  if (error) {
    return <CatalogErrorState error={error} onRetry={refetch} />;
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-foreground">
            Catalog Management
          </h1>
          <p className="text-muted-foreground">
            Manage admin-controlled base catalogs and server configurations
          </p>
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={() => refetch()}
            disabled={isLoading}
          >
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
          <Button variant="outline" onClick={() => setIsImportDialogOpen(true)}>
            <Upload className="h-4 w-4 mr-2" />
            Import
          </Button>
          <Button onClick={() => setIsCreateDialogOpen(true)}>
            <Plus className="h-4 w-4 mr-2" />
            New Catalog
          </Button>
        </div>
      </div>

      {/* Statistics Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Total Catalogs
            </CardTitle>
            <BookOpen className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{catalogs.length}</div>
            <p className="text-xs text-muted-foreground">Admin base catalogs</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Active Catalogs
            </CardTitle>
            <BookOpen className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {catalogs.filter(c => c.status === 'active').length}
            </div>
            <p className="text-xs text-muted-foreground">Currently active</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Servers</CardTitle>
            <BookOpen className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {catalogs.reduce((sum, c) => sum + c.server_count, 0)}
            </div>
            <p className="text-xs text-muted-foreground">Across all catalogs</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Public Catalogs
            </CardTitle>
            <BookOpen className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {catalogs.filter(c => c.is_public).length}
            </div>
            <p className="text-xs text-muted-foreground">
              Available to all users
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Filters */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Filters & Search</CardTitle>
          <CardDescription>
            Filter and search through admin catalogs
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col gap-4">
            {/* Search */}
            <div className="flex items-center gap-2">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Search catalogs..."
                  value={searchQuery}
                  onChange={e => setSearchQuery(e.target.value)}
                  className="pl-10"
                />
              </div>
              {hasActiveFilters && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleClearFilters}
                >
                  Clear Filters
                </Button>
              )}
            </div>

            {/* Type and Status Filters */}
            <div className="flex flex-wrap gap-2">
              <Select>
                <SelectTrigger className="w-48">
                  <SelectValue placeholder="Filter by type" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Types</SelectItem>
                  <SelectItem value="official">Official</SelectItem>
                  <SelectItem value="team">Team</SelectItem>
                  <SelectItem value="imported">Imported</SelectItem>
                  <SelectItem value="custom">Custom</SelectItem>
                </SelectContent>
              </Select>

              <Select>
                <SelectTrigger className="w-48">
                  <SelectValue placeholder="Filter by status" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Statuses</SelectItem>
                  <SelectItem value="active">Active</SelectItem>
                  <SelectItem value="deprecated">Deprecated</SelectItem>
                  <SelectItem value="experimental">Experimental</SelectItem>
                  <SelectItem value="archived">Archived</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Active Filters Display */}
            {hasActiveFilters && (
              <div className="flex flex-wrap gap-2">
                {searchQuery && (
                  <Badge variant="secondary" className="gap-1">
                    Search: {searchQuery}
                  </Badge>
                )}
                {typeFilter.map(type => (
                  <Badge key={type} variant="secondary" className="gap-1">
                    Type: {type}
                  </Badge>
                ))}
                {statusFilter.map(status => (
                  <Badge key={status} variant="secondary" className="gap-1">
                    Status: {status}
                  </Badge>
                ))}
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Catalog Management Table */}
      <CatalogManagement catalogs={catalogs} isLoading={isLoading} />

      {/* Dialogs */}
      <CatalogEditor
        isOpen={isCreateDialogOpen}
        onClose={() => setIsCreateDialogOpen(false)}
        onSave={handleCreateCatalog}
        isLoading={createCatalogMutation.isPending}
      />

      <CatalogImporter
        isOpen={isImportDialogOpen}
        onClose={() => setIsImportDialogOpen(false)}
        onImport={handleImportCatalog}
        isLoading={importCatalogMutation.isPending}
      />
    </div>
  );
}

// Loading skeleton component
function CatalogLoadingSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <Skeleton className="h-8 w-64 mb-2" />
          <Skeleton className="h-4 w-96" />
        </div>
        <div className="flex gap-2">
          <Skeleton className="h-10 w-24" />
          <Skeleton className="h-10 w-24" />
          <Skeleton className="h-10 w-32" />
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Card key={i}>
            <CardHeader className="pb-2">
              <Skeleton className="h-4 w-24" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-8 w-16 mb-1" />
              <Skeleton className="h-3 w-20" />
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <Skeleton className="h-6 w-32" />
          <Skeleton className="h-4 w-48" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-10 w-full mb-4" />
          <div className="flex gap-2">
            <Skeleton className="h-10 w-48" />
            <Skeleton className="h-10 w-48" />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="p-6">
          <div className="space-y-4">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-16 w-full" />
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// Error state component
function CatalogErrorState({
  error,
  onRetry,
}: {
  error: Error;
  onRetry: () => void;
}) {
  return (
    <div className="flex items-center justify-center min-h-[400px]">
      <div className="text-center max-w-md">
        <AlertCircle className="h-16 w-16 text-red-500 mx-auto mb-4" />
        <h2 className="text-2xl font-semibold text-foreground mb-2">
          Failed to Load Catalogs
        </h2>
        <p className="text-muted-foreground mb-6">
          {error.message ||
            'An unexpected error occurred while loading catalogs.'}
        </p>
        <div className="flex gap-2 justify-center">
          <Button onClick={onRetry}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Try Again
          </Button>
          <Button variant="outline" onClick={() => window.location.reload()}>
            Reload Page
          </Button>
        </div>
      </div>
    </div>
  );
}
