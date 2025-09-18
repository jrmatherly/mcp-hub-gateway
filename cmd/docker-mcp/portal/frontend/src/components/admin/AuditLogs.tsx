'use client';

import { useState } from 'react';
import {
  Download,
  Eye,
  RefreshCw,
  Search,
  Shield,
  User,
  AlertCircle,
  CheckCircle,
  Info,
  Clock,
  Monitor,
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
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { useAuditLogs, type AuditLog } from '@/hooks/api/use-admin';

interface AuditLogDetailProps {
  log: AuditLog;
  isOpen: boolean;
  onClose: () => void;
}

function AuditLogDetail({ log, isOpen, onClose }: AuditLogDetailProps) {
  const formatMetadata = (metadata?: Record<string, unknown>) => {
    if (!metadata || Object.keys(metadata).length === 0) {
      return 'No additional metadata';
    }

    return Object.entries(metadata)
      .map(([key, value]) => `${key}: ${JSON.stringify(value)}`)
      .join('\n');
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center space-x-2">
            <Shield className="h-5 w-5" />
            <span>Audit Log Details</span>
          </DialogTitle>
          <DialogDescription>
            Detailed information about this audit event
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {/* Basic Information */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-sm font-medium text-muted-foreground">
                Timestamp
              </label>
              <p className="text-sm font-mono">
                {new Date(log.timestamp).toLocaleString()}
              </p>
            </div>

            <div>
              <label className="text-sm font-medium text-muted-foreground">
                Action
              </label>
              <p className="text-sm font-semibold">{log.action}</p>
            </div>

            <div>
              <label className="text-sm font-medium text-muted-foreground">
                User
              </label>
              <p className="text-sm">{log.userName || log.userId}</p>
            </div>

            <div>
              <label className="text-sm font-medium text-muted-foreground">
                Status
              </label>
              <Badge variant={log.success ? 'default' : 'destructive'}>
                {log.success ? 'Success' : 'Failed'}
              </Badge>
            </div>

            <div>
              <label className="text-sm font-medium text-muted-foreground">
                Resource
              </label>
              <p className="text-sm">{log.resource}</p>
            </div>

            {log.resourceId && (
              <div>
                <label className="text-sm font-medium text-muted-foreground">
                  Resource ID
                </label>
                <p className="text-sm font-mono">{log.resourceId}</p>
              </div>
            )}
          </div>

          {/* Network Information */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-sm font-medium text-muted-foreground">
                IP Address
              </label>
              <p className="text-sm font-mono">{log.ipAddress}</p>
            </div>

            <div>
              <label className="text-sm font-medium text-muted-foreground">
                User Agent
              </label>
              <p className="text-sm break-all">{log.userAgent}</p>
            </div>
          </div>

          {/* Error Information */}
          {!log.success && log.errorMessage && (
            <div>
              <label className="text-sm font-medium text-muted-foreground">
                Error Message
              </label>
              <div className="mt-1 p-3 bg-red-50 border border-red-200 rounded-md">
                <p className="text-sm text-red-800">{log.errorMessage}</p>
              </div>
            </div>
          )}

          {/* Metadata */}
          <div>
            <label className="text-sm font-medium text-muted-foreground">
              Additional Metadata
            </label>
            <div className="mt-1 p-3 bg-gray-50 border rounded-md">
              <pre className="text-xs whitespace-pre-wrap">
                {formatMetadata(log.metadata)}
              </pre>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

interface LogRowProps {
  log: AuditLog;
  onViewDetails: (log: AuditLog) => void;
}

function LogRow({ log, onViewDetails }: LogRowProps) {
  const getActionIcon = (action: string) => {
    if (action.includes('login') || action.includes('auth')) {
      return <Shield className="h-3 w-3" />;
    }
    if (action.includes('user')) {
      return <User className="h-3 w-3" />;
    }
    if (action.includes('server') || action.includes('config')) {
      return <Monitor className="h-3 w-3" />;
    }
    return <Info className="h-3 w-3" />;
  };

  const getActionBadgeVariant = (action: string, success: boolean) => {
    if (!success) return 'destructive';

    if (action.includes('delete') || action.includes('remove')) {
      return 'destructive';
    }
    if (action.includes('create') || action.includes('add')) {
      return 'default';
    }
    if (action.includes('update') || action.includes('modify')) {
      return 'secondary';
    }
    return 'outline';
  };

  return (
    <TableRow className="hover:bg-muted/50">
      <TableCell>
        <div className="flex items-center space-x-2 text-sm">
          <Clock className="h-3 w-3 text-muted-foreground" />
          <span className="font-mono">
            {new Date(log.timestamp).toLocaleDateString()}
          </span>
          <span className="font-mono text-muted-foreground">
            {new Date(log.timestamp).toLocaleTimeString()}
          </span>
        </div>
      </TableCell>

      <TableCell>
        <div className="flex items-center space-x-2">
          <div className="h-6 w-6 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-xs font-medium">
            {log.userName?.charAt(0) || log.userId.slice(0, 2)}
          </div>
          <span className="text-sm">{log.userName || log.userId}</span>
        </div>
      </TableCell>

      <TableCell>
        <Badge
          variant={getActionBadgeVariant(log.action, log.success)}
          className="text-xs"
        >
          {getActionIcon(log.action)}
          <span className="ml-1">{log.action}</span>
        </Badge>
      </TableCell>

      <TableCell>
        <div className="text-sm">
          <div className="font-medium">{log.resource}</div>
          {log.resourceId && (
            <div className="text-xs text-muted-foreground font-mono">
              {log.resourceId}
            </div>
          )}
        </div>
      </TableCell>

      <TableCell>
        <div className="flex items-center space-x-2">
          {log.success ? (
            <CheckCircle className="h-4 w-4 text-green-600" />
          ) : (
            <AlertCircle className="h-4 w-4 text-red-600" />
          )}
          <Badge variant={log.success ? 'default' : 'destructive'}>
            {log.success ? 'Success' : 'Failed'}
          </Badge>
        </div>
      </TableCell>

      <TableCell>
        <span className="text-sm font-mono text-muted-foreground">
          {log.ipAddress}
        </span>
      </TableCell>

      <TableCell>
        <Button variant="ghost" size="sm" onClick={() => onViewDetails(log)}>
          <Eye className="h-4 w-4" />
        </Button>
      </TableCell>
    </TableRow>
  );
}

interface FilterControlsProps {
  searchTerm: string;
  onSearchChange: (value: string) => void;
  actionFilter: string;
  onActionFilterChange: (value: string) => void;
  successFilter: string;
  onSuccessFilterChange: (value: string) => void;
  dateFrom: string;
  onDateFromChange: (value: string) => void;
  dateTo: string;
  onDateToChange: (value: string) => void;
  onRefresh: () => void;
  onExport: () => void;
}

function FilterControls({
  searchTerm,
  onSearchChange,
  actionFilter,
  onActionFilterChange,
  successFilter,
  onSuccessFilterChange,
  dateFrom,
  onDateFromChange,
  dateTo,
  onDateToChange,
  onRefresh,
  onExport,
}: FilterControlsProps) {
  return (
    <Card>
      <CardContent className="pt-6">
        <div className="space-y-4">
          {/* Search and Actions */}
          <div className="flex items-center space-x-4">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Search by user, action, or resource..."
                  value={searchTerm}
                  onChange={e => onSearchChange(e.target.value)}
                  className="pl-10"
                />
              </div>
            </div>

            <Button variant="outline" onClick={onRefresh}>
              <RefreshCw className="h-4 w-4 mr-2" />
              Refresh
            </Button>

            <Button variant="outline" onClick={onExport}>
              <Download className="h-4 w-4 mr-2" />
              Export
            </Button>
          </div>

          {/* Filters */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <Select value={actionFilter} onValueChange={onActionFilterChange}>
              <SelectTrigger>
                <SelectValue placeholder="All Actions" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Actions</SelectItem>
                <SelectItem value="login">Login</SelectItem>
                <SelectItem value="logout">Logout</SelectItem>
                <SelectItem value="user_create">User Create</SelectItem>
                <SelectItem value="user_update">User Update</SelectItem>
                <SelectItem value="user_delete">User Delete</SelectItem>
                <SelectItem value="server_enable">Server Enable</SelectItem>
                <SelectItem value="server_disable">Server Disable</SelectItem>
                <SelectItem value="config_update">Config Update</SelectItem>
              </SelectContent>
            </Select>

            <Select value={successFilter} onValueChange={onSuccessFilterChange}>
              <SelectTrigger>
                <SelectValue placeholder="All Results" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Results</SelectItem>
                <SelectItem value="success">Success Only</SelectItem>
                <SelectItem value="failed">Failed Only</SelectItem>
              </SelectContent>
            </Select>

            <div>
              <Input
                type="date"
                value={dateFrom}
                onChange={e => onDateFromChange(e.target.value)}
                placeholder="From Date"
              />
            </div>

            <div>
              <Input
                type="date"
                value={dateTo}
                onChange={e => onDateToChange(e.target.value)}
                placeholder="To Date"
              />
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function LoadingSkeleton() {
  return (
    <div className="space-y-4">
      <Card>
        <CardContent className="pt-6">
          <div className="space-y-4">
            <div className="flex space-x-4">
              <Skeleton className="flex-1 h-10" />
              <Skeleton className="h-10 w-24" />
              <Skeleton className="h-10 w-24" />
            </div>
            <div className="grid grid-cols-4 gap-4">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-10" />
              ))}
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <Skeleton className="h-6 w-48" />
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            {Array.from({ length: 10 }).map((_, i) => (
              <Skeleton key={i} className="h-12" />
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export default function AuditLogs() {
  const [searchTerm, setSearchTerm] = useState('');
  const [actionFilter, setActionFilter] = useState('all');
  const [successFilter, setSuccessFilter] = useState('all');
  const [dateFrom, setDateFrom] = useState('');
  const [dateTo, setDateTo] = useState('');
  const [page, setPage] = useState(1);
  const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null);
  const [showDetails, setShowDetails] = useState(false);

  const limit = 20;

  const { data, isLoading, error, refetch } = useAuditLogs({
    page,
    limit,
    action: actionFilter !== 'all' ? actionFilter : undefined,
    dateFrom: dateFrom ? new Date(dateFrom) : undefined,
    dateTo: dateTo ? new Date(dateTo) : undefined,
  });

  const filteredLogs =
    data?.logs?.filter(log => {
      const matchesSearch =
        log.userName?.toLowerCase().includes(searchTerm.toLowerCase()) ||
        log.action.toLowerCase().includes(searchTerm.toLowerCase()) ||
        log.resource.toLowerCase().includes(searchTerm.toLowerCase()) ||
        log.resourceId?.toLowerCase().includes(searchTerm.toLowerCase());

      const matchesSuccess =
        successFilter === 'all' ||
        (successFilter === 'success' && log.success) ||
        (successFilter === 'failed' && !log.success);

      return matchesSearch && matchesSuccess;
    }) || [];

  const handleViewDetails = (log: AuditLog) => {
    setSelectedLog(log);
    setShowDetails(true);
  };

  const handleExport = () => {
    // Implement CSV export functionality
    const csvContent = [
      'Timestamp,User,Action,Resource,Resource ID,Status,IP Address,Error Message',
      ...filteredLogs.map(log =>
        [
          new Date(log.timestamp).toISOString(),
          log.userName || log.userId,
          log.action,
          log.resource,
          log.resourceId || '',
          log.success ? 'Success' : 'Failed',
          log.ipAddress,
          log.errorMessage || '',
        ]
          .map(field => `"${field}"`)
          .join(',')
      ),
    ].join('\n');

    const blob = new Blob([csvContent], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `audit-logs-${new Date().toISOString().split('T')[0]}.csv`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const handleRefresh = () => {
    refetch();
  };

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (error) {
    return (
      <Card>
        <CardContent className="pt-6">
          <div className="text-center">
            <AlertCircle className="h-16 w-16 text-amber-500 mx-auto mb-4" />
            <h3 className="text-lg font-semibold mb-2">
              Failed to Load Audit Logs
            </h3>
            <p className="text-muted-foreground mb-4">{error.message}</p>
            <Button onClick={handleRefresh}>
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
          <h2 className="text-2xl font-bold tracking-tight">Audit Logs</h2>
          <p className="text-muted-foreground">
            Security and operation logs for compliance and monitoring
          </p>
        </div>

        <div className="flex items-center space-x-2">
          <Badge variant="outline" className="text-green-600 bg-green-50">
            <Shield className="h-3 w-3 mr-1" />
            Real-time Logging
          </Badge>
        </div>
      </div>

      {/* Filters */}
      <FilterControls
        searchTerm={searchTerm}
        onSearchChange={setSearchTerm}
        actionFilter={actionFilter}
        onActionFilterChange={setActionFilter}
        successFilter={successFilter}
        onSuccessFilterChange={setSuccessFilter}
        dateFrom={dateFrom}
        onDateFromChange={setDateFrom}
        dateTo={dateTo}
        onDateToChange={setDateTo}
        onRefresh={handleRefresh}
        onExport={handleExport}
      />

      {/* Audit Logs Table */}
      <Card>
        <CardHeader>
          <CardTitle>Audit Events ({filteredLogs.length})</CardTitle>
          <CardDescription>
            Showing {filteredLogs.length} of {data?.total || 0} total events
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Timestamp</TableHead>
                <TableHead>User</TableHead>
                <TableHead>Action</TableHead>
                <TableHead>Resource</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>IP Address</TableHead>
                <TableHead className="w-[70px]">Details</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredLogs.map(log => (
                <LogRow
                  key={log.id}
                  log={log}
                  onViewDetails={handleViewDetails}
                />
              ))}
            </TableBody>
          </Table>

          {filteredLogs.length === 0 && (
            <div className="text-center py-8 text-muted-foreground">
              <Shield className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p>No audit logs found matching your criteria</p>
              <p className="text-sm">
                Try adjusting your filters or date range
              </p>
            </div>
          )}

          {/* Pagination */}
          {data?.pages && data.pages > 1 && (
            <div className="flex items-center justify-between mt-6">
              <div className="text-sm text-muted-foreground">
                Page {page} of {data.pages}
              </div>
              <div className="flex space-x-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage(p => Math.max(1, p - 1))}
                  disabled={page === 1}
                >
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage(p => Math.min(data.pages, p + 1))}
                  disabled={page === data.pages}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Audit Log Details Dialog */}
      {selectedLog && (
        <AuditLogDetail
          log={selectedLog}
          isOpen={showDetails}
          onClose={() => {
            setShowDetails(false);
            setSelectedLog(null);
          }}
        />
      )}
    </div>
  );
}
