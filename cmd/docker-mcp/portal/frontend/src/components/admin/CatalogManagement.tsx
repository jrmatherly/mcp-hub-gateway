'use client';

import { useState } from 'react';
import {
  Edit,
  Trash2,
  Download,
  Eye,
  Globe,
  Lock,
  Star,
  StarOff,
  Calendar,
  Package,
  ExternalLink,
} from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';

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
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';

import {
  useUpdateAdminCatalog,
  useDeleteAdminCatalog,
  useExportCatalog,
} from '@/hooks/api/use-catalog';
import type { AdminCatalog, UpdateCatalogRequest } from '@/types/catalog';

interface CatalogManagementProps {
  catalogs: AdminCatalog[];
  isLoading?: boolean;
}

export function CatalogManagement({
  catalogs,
  isLoading,
}: CatalogManagementProps) {
  const [selectedCatalog, setSelectedCatalog] = useState<AdminCatalog | null>(
    null
  );
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [isDetailsDialogOpen, setIsDetailsDialogOpen] = useState(false);

  const updateCatalogMutation = useUpdateAdminCatalog();
  const deleteCatalogMutation = useDeleteAdminCatalog();
  const exportCatalogMutation = useExportCatalog();

  const handleEdit = (catalog: AdminCatalog) => {
    setSelectedCatalog(catalog);
    // This would open the edit dialog (passed from parent)
  };

  const handleDelete = (catalog: AdminCatalog) => {
    setSelectedCatalog(catalog);
    setIsDeleteDialogOpen(true);
  };

  const handleToggleDefault = async (catalog: AdminCatalog) => {
    // TODO: Implement toggle default functionality
    // Note: This would need to be implemented on the backend
    // For now, we'll use the is_public flag as an example
    const request: UpdateCatalogRequest = {
      is_public: catalog.is_public,
    };

    try {
      await updateCatalogMutation.mutateAsync({
        id: catalog.id,
        request,
      });
    } catch (error) {
      console.error('Failed to toggle default:', error);
    }
  };

  const handleTogglePublic = async (catalog: AdminCatalog) => {
    const request: UpdateCatalogRequest = {
      is_public: !catalog.is_public,
    };

    try {
      await updateCatalogMutation.mutateAsync({
        id: catalog.id,
        request,
      });
    } catch (error) {
      console.error('Failed to toggle public:', error);
    }
  };

  const handleViewDetails = (catalog: AdminCatalog) => {
    setSelectedCatalog(catalog);
    setIsDetailsDialogOpen(true);
  };

  const handleExport = async (
    catalog: AdminCatalog,
    format: 'json' | 'yaml'
  ) => {
    try {
      const data = await exportCatalogMutation.mutateAsync({ format });

      // Create download link
      const blob = new Blob([data], {
        type: format === 'json' ? 'application/json' : 'application/x-yaml',
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${catalog.name}.${format}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (error) {
      console.error('Failed to export catalog:', error);
    }
  };

  const confirmDelete = async () => {
    if (!selectedCatalog) return;

    try {
      await deleteCatalogMutation.mutateAsync(selectedCatalog.id);
      setIsDeleteDialogOpen(false);
      setSelectedCatalog(null);
    } catch (error) {
      console.error('Failed to delete catalog:', error);
    }
  };

  if (isLoading) {
    return <CatalogTableSkeleton />;
  }

  if (catalogs.length === 0) {
    return <EmptyCatalogState />;
  }

  return (
    <TooltipProvider>
      <Card>
        <CardHeader>
          <CardTitle>Admin Base Catalogs</CardTitle>
          <CardDescription>
            Manage catalogs that are inherited by all users
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Servers</TableHead>
                <TableHead>Visibility</TableHead>
                <TableHead>Updated</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {catalogs.map(catalog => (
                <TableRow key={catalog.id}>
                  <TableCell>
                    <div className="flex flex-col gap-1">
                      <div className="flex items-center gap-2">
                        <span className="font-medium">
                          {catalog.display_name || catalog.name}
                        </span>
                        {catalog.is_default && (
                          <Tooltip>
                            <TooltipTrigger>
                              <Star className="h-4 w-4 text-amber-500 fill-current" />
                            </TooltipTrigger>
                            <TooltipContent>Default catalog</TooltipContent>
                          </Tooltip>
                        )}
                      </div>
                      {catalog.description && (
                        <span className="text-sm text-muted-foreground">
                          {catalog.description}
                        </span>
                      )}
                      {catalog.tags && catalog.tags.length > 0 && (
                        <div className="flex gap-1 flex-wrap">
                          {catalog.tags.slice(0, 3).map(tag => (
                            <Badge
                              key={tag}
                              variant="outline"
                              className="text-xs"
                            >
                              {tag}
                            </Badge>
                          ))}
                          {catalog.tags.length > 3 && (
                            <Badge variant="outline" className="text-xs">
                              +{catalog.tags.length - 3}
                            </Badge>
                          )}
                        </div>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="secondary">{catalog.type}</Badge>
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        catalog.status === 'active'
                          ? 'default'
                          : catalog.status === 'deprecated'
                            ? 'destructive'
                            : 'secondary'
                      }
                    >
                      {catalog.status}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Package className="h-4 w-4 text-muted-foreground" />
                      <span>{catalog.server_count}</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      {catalog.is_public ? (
                        <Globe className="h-4 w-4 text-green-600" />
                      ) : (
                        <Lock className="h-4 w-4 text-muted-foreground" />
                      )}
                      <span className="text-sm">
                        {catalog.is_public ? 'Public' : 'Private'}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1 text-sm text-muted-foreground">
                      <Calendar className="h-4 w-4" />
                      <span>
                        {formatDistanceToNow(new Date(catalog.updated_at), {
                          addSuffix: true,
                        })}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell className="text-right">
                    <CatalogActions
                      catalog={catalog}
                      onEdit={handleEdit}
                      onDelete={handleDelete}
                      onToggleDefault={handleToggleDefault}
                      onTogglePublic={handleTogglePublic}
                      onViewDetails={handleViewDetails}
                      onExport={handleExport}
                      isLoading={
                        updateCatalogMutation.isPending ||
                        deleteCatalogMutation.isPending ||
                        exportCatalogMutation.isPending
                      }
                    />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Delete Confirmation Dialog */}
      <Dialog open={isDeleteDialogOpen} onOpenChange={setIsDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Catalog</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete the catalog "
              {selectedCatalog?.display_name || selectedCatalog?.name}"? This
              action cannot be undone and will affect all users who inherit this
              catalog.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setIsDeleteDialogOpen(false)}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={confirmDelete}
              disabled={deleteCatalogMutation.isPending}
            >
              {deleteCatalogMutation.isPending ? 'Deleting...' : 'Delete'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Catalog Details Dialog */}
      <Dialog open={isDetailsDialogOpen} onOpenChange={setIsDetailsDialogOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Catalog Details</DialogTitle>
          </DialogHeader>
          {selectedCatalog && <CatalogDetails catalog={selectedCatalog} />}
        </DialogContent>
      </Dialog>
    </TooltipProvider>
  );
}

// Catalog actions dropdown component
interface CatalogActionsProps {
  catalog: AdminCatalog;
  onEdit: (catalog: AdminCatalog) => void;
  onDelete: (catalog: AdminCatalog) => void;
  onToggleDefault: (catalog: AdminCatalog) => void;
  onTogglePublic: (catalog: AdminCatalog) => void;
  onViewDetails: (catalog: AdminCatalog) => void;
  onExport: (catalog: AdminCatalog, format: 'json' | 'yaml') => void;
  isLoading?: boolean;
}

function CatalogActions({
  catalog,
  onEdit,
  onDelete,
  onToggleDefault,
  onTogglePublic,
  onViewDetails,
  onExport,
  isLoading,
}: CatalogActionsProps) {
  return (
    <div className="flex items-center gap-1">
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onViewDetails(catalog)}
          >
            <Eye className="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>View details</TooltipContent>
      </Tooltip>

      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onEdit(catalog)}
            disabled={isLoading}
          >
            <Edit className="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>Edit catalog</TooltipContent>
      </Tooltip>

      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onTogglePublic(catalog)}
            disabled={isLoading}
          >
            {catalog.is_public ? (
              <Lock className="h-4 w-4" />
            ) : (
              <Globe className="h-4 w-4" />
            )}
          </Button>
        </TooltipTrigger>
        <TooltipContent>
          {catalog.is_public ? 'Make private' : 'Make public'}
        </TooltipContent>
      </Tooltip>

      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onToggleDefault(catalog)}
            disabled={isLoading}
          >
            {catalog.is_default ? (
              <StarOff className="h-4 w-4" />
            ) : (
              <Star className="h-4 w-4" />
            )}
          </Button>
        </TooltipTrigger>
        <TooltipContent>
          {catalog.is_default ? 'Remove from default' : 'Set as default'}
        </TooltipContent>
      </Tooltip>

      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onExport(catalog, 'json')}
            disabled={isLoading}
          >
            <Download className="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>Export catalog</TooltipContent>
      </Tooltip>

      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onDelete(catalog)}
            disabled={isLoading}
            className="text-red-600 hover:text-red-700"
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>Delete catalog</TooltipContent>
      </Tooltip>
    </div>
  );
}

// Catalog details component
function CatalogDetails({ catalog }: { catalog: AdminCatalog }) {
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="text-sm font-medium text-muted-foreground">
            Name
          </label>
          <p className="font-medium">{catalog.name}</p>
        </div>
        <div>
          <label className="text-sm font-medium text-muted-foreground">
            Display Name
          </label>
          <p className="font-medium">{catalog.display_name || catalog.name}</p>
        </div>
        <div>
          <label className="text-sm font-medium text-muted-foreground">
            Type
          </label>
          <Badge variant="secondary">{catalog.type}</Badge>
        </div>
        <div>
          <label className="text-sm font-medium text-muted-foreground">
            Status
          </label>
          <Badge
            variant={
              catalog.status === 'active'
                ? 'default'
                : catalog.status === 'deprecated'
                  ? 'destructive'
                  : 'secondary'
            }
          >
            {catalog.status}
          </Badge>
        </div>
        <div>
          <label className="text-sm font-medium text-muted-foreground">
            Version
          </label>
          <p>{catalog.version || 'N/A'}</p>
        </div>
        <div>
          <label className="text-sm font-medium text-muted-foreground">
            Server Count
          </label>
          <p>{catalog.server_count}</p>
        </div>
      </div>

      {catalog.description && (
        <div>
          <label className="text-sm font-medium text-muted-foreground">
            Description
          </label>
          <p className="text-sm">{catalog.description}</p>
        </div>
      )}

      {catalog.tags && catalog.tags.length > 0 && (
        <div>
          <label className="text-sm font-medium text-muted-foreground">
            Tags
          </label>
          <div className="flex gap-1 flex-wrap mt-1">
            {catalog.tags.map(tag => (
              <Badge key={tag} variant="outline">
                {tag}
              </Badge>
            ))}
          </div>
        </div>
      )}

      {(catalog.homepage || catalog.repository) && (
        <div>
          <label className="text-sm font-medium text-muted-foreground">
            Links
          </label>
          <div className="flex gap-2 mt-1">
            {catalog.homepage && (
              <a
                href={catalog.homepage}
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm text-blue-600 hover:text-blue-700 flex items-center gap-1"
              >
                Homepage <ExternalLink className="h-3 w-3" />
              </a>
            )}
            {catalog.repository && (
              <a
                href={catalog.repository}
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm text-blue-600 hover:text-blue-700 flex items-center gap-1"
              >
                Repository <ExternalLink className="h-3 w-3" />
              </a>
            )}
          </div>
        </div>
      )}

      <div className="grid grid-cols-2 gap-4 text-sm text-muted-foreground">
        <div>
          <label className="font-medium">Created</label>
          <p>{new Date(catalog.created_at).toLocaleDateString()}</p>
        </div>
        <div>
          <label className="font-medium">Updated</label>
          <p>{new Date(catalog.updated_at).toLocaleDateString()}</p>
        </div>
      </div>
    </div>
  );
}

// Loading skeleton
function CatalogTableSkeleton() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Admin Base Catalogs</CardTitle>
        <CardDescription>
          Manage catalogs that are inherited by all users
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {Array.from({ length: 5 }).map((_, i) => (
            <div
              key={i}
              className="flex items-center justify-between p-4 border rounded"
            >
              <div className="flex items-center gap-4">
                <Skeleton className="h-10 w-10 rounded" />
                <div className="space-y-2">
                  <Skeleton className="h-4 w-32" />
                  <Skeleton className="h-3 w-48" />
                </div>
              </div>
              <div className="flex gap-2">
                <Skeleton className="h-8 w-8" />
                <Skeleton className="h-8 w-8" />
                <Skeleton className="h-8 w-8" />
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

// Empty state
function EmptyCatalogState() {
  return (
    <Card>
      <CardContent className="flex flex-col items-center justify-center py-12">
        <Package className="h-16 w-16 text-muted-foreground mb-4" />
        <h3 className="text-lg font-medium text-foreground mb-2">
          No Admin Catalogs
        </h3>
        <p className="text-muted-foreground text-center max-w-md mb-6">
          Admin base catalogs provide default server configurations that are
          inherited by all users. Create your first catalog to get started.
        </p>
        <Button>
          <Package className="h-4 w-4 mr-2" />
          Create First Catalog
        </Button>
      </CardContent>
    </Card>
  );
}
