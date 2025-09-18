'use client';

import { useState } from 'react';
import {
  MoreHorizontal,
  Plus,
  Search,
  Shield,
  ShieldCheck,
  ShieldX,
  Trash2,
  User,
  UserCheck,
  UserX,
  Clock,
  Server,
  FileText,
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
  DialogTrigger,
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
import {
  useAdminUsers,
  useUpdateUserRole,
  useToggleUserStatus,
  useDeleteUser,
  type UserManagementData,
} from '@/hooks/api/use-admin';

interface UserActionMenuProps {
  user: UserManagementData;
  onEditRole: (user: UserManagementData) => void;
  onToggleStatus: (user: UserManagementData) => void;
  onDelete: (user: UserManagementData) => void;
}

function UserActionMenu({
  user,
  onEditRole,
  onToggleStatus,
  onDelete,
}: UserActionMenuProps) {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogTrigger asChild>
        <Button variant="ghost" size="sm">
          <MoreHorizontal className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>User Actions</DialogTitle>
          <DialogDescription>
            Manage {user.name || user.email}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-2">
          <Button
            variant="outline"
            className="w-full justify-start"
            onClick={() => {
              onEditRole(user);
              setIsOpen(false);
            }}
          >
            <Shield className="h-4 w-4 mr-2" />
            Change Role
          </Button>

          <Button
            variant="outline"
            className="w-full justify-start"
            onClick={() => {
              onToggleStatus(user);
              setIsOpen(false);
            }}
          >
            {user.active ? (
              <>
                <UserX className="h-4 w-4 mr-2" />
                Deactivate User
              </>
            ) : (
              <>
                <UserCheck className="h-4 w-4 mr-2" />
                Activate User
              </>
            )}
          </Button>

          <Button
            variant="outline"
            className="w-full justify-start text-red-600 hover:text-red-700"
            onClick={() => {
              onDelete(user);
              setIsOpen(false);
            }}
          >
            <Trash2 className="h-4 w-4 mr-2" />
            Delete User
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

interface RoleEditDialogProps {
  user: UserManagementData | null;
  isOpen: boolean;
  onClose: () => void;
  onSave: (userId: string, role: string) => void;
  isLoading?: boolean;
}

function RoleEditDialog({
  user,
  isOpen,
  onClose,
  onSave,
  isLoading,
}: RoleEditDialogProps) {
  const [selectedRole, setSelectedRole] = useState(user?.role || '');

  const roles = [
    { value: 'user', label: 'User', description: 'Basic user permissions' },
    {
      value: 'admin',
      label: 'Admin',
      description: 'Server and config management',
    },
    {
      value: 'super_admin',
      label: 'Super Admin',
      description: 'User management access',
    },
    {
      value: 'system_admin',
      label: 'System Admin',
      description: 'Full system access',
    },
  ];

  const handleSave = () => {
    if (user && selectedRole) {
      onSave(user.id, selectedRole);
    }
  };

  if (!user) return null;

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Change User Role</DialogTitle>
          <DialogDescription>
            Update role for {user.name || user.email}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div>
            <label className="text-sm font-medium">Current Role</label>
            <p className="text-sm text-muted-foreground capitalize">
              {user.role}
            </p>
          </div>

          <div>
            <label className="text-sm font-medium">New Role</label>
            <Select value={selectedRole} onValueChange={setSelectedRole}>
              <SelectTrigger>
                <SelectValue placeholder="Select a role" />
              </SelectTrigger>
              <SelectContent>
                {roles.map(role => (
                  <SelectItem key={role.value} value={role.value}>
                    <div>
                      <div className="font-medium">{role.label}</div>
                      <div className="text-xs text-muted-foreground">
                        {role.description}
                      </div>
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            disabled={!selectedRole || selectedRole === user.role || isLoading}
          >
            {isLoading ? 'Updating...' : 'Update Role'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

interface DeleteUserDialogProps {
  user: UserManagementData | null;
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (userId: string) => void;
  isLoading?: boolean;
}

function DeleteUserDialog({
  user,
  isOpen,
  onClose,
  onConfirm,
  isLoading,
}: DeleteUserDialogProps) {
  const handleConfirm = () => {
    if (user) {
      onConfirm(user.id);
    }
  };

  if (!user) return null;

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete User</DialogTitle>
          <DialogDescription>
            Are you sure you want to delete {user.name || user.email}? This
            action cannot be undone.
          </DialogDescription>
        </DialogHeader>

        <div className="bg-red-50 border border-red-200 rounded-md p-4">
          <div className="flex items-start space-x-2">
            <Trash2 className="h-5 w-5 text-red-600 mt-0.5" />
            <div>
              <h4 className="text-sm font-medium text-red-800">
                This will permanently:
              </h4>
              <ul className="text-sm text-red-700 mt-1 space-y-1">
                <li>• Remove all user data and sessions</li>
                <li>• Revoke all permissions and access</li>
                <li>• Delete audit log entries</li>
              </ul>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={handleConfirm}
            disabled={isLoading}
          >
            {isLoading ? 'Deleting...' : 'Delete User'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function UserRow({
  user,
  onEditRole,
  onToggleStatus,
  onDelete,
}: {
  user: UserManagementData;
  onEditRole: (user: UserManagementData) => void;
  onToggleStatus: (user: UserManagementData) => void;
  onDelete: (user: UserManagementData) => void;
}) {
  const getRoleBadgeVariant = (role: string) => {
    switch (role) {
      case 'system_admin':
        return 'destructive';
      case 'super_admin':
        return 'default';
      case 'admin':
        return 'secondary';
      default:
        return 'outline';
    }
  };

  const getRoleIcon = (role: string) => {
    switch (role) {
      case 'system_admin':
        return <ShieldCheck className="h-3 w-3" />;
      case 'super_admin':
        return <Shield className="h-3 w-3" />;
      case 'admin':
        return <ShieldX className="h-3 w-3" />;
      default:
        return <User className="h-3 w-3" />;
    }
  };

  return (
    <TableRow>
      <TableCell>
        <div className="flex items-center space-x-3">
          <div className="h-8 w-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-medium">
            {user.name?.charAt(0) || user.email.charAt(0)}
          </div>
          <div>
            <div className="font-medium">{user.name || 'No name'}</div>
            <div className="text-sm text-muted-foreground">{user.email}</div>
          </div>
        </div>
      </TableCell>

      <TableCell>
        <Badge variant={getRoleBadgeVariant(user.role)} className="capitalize">
          {getRoleIcon(user.role)}
          <span className="ml-1">{user.role.replace('_', ' ')}</span>
        </Badge>
      </TableCell>

      <TableCell>
        <Badge variant={user.active ? 'default' : 'secondary'}>
          {user.active ? 'Active' : 'Inactive'}
        </Badge>
      </TableCell>

      <TableCell>
        <div className="flex items-center space-x-1 text-sm text-muted-foreground">
          <Clock className="h-3 w-3" />
          <span>
            {user.lastActivity
              ? new Date(user.lastActivity).toLocaleDateString()
              : 'Never'}
          </span>
        </div>
      </TableCell>

      <TableCell>
        <div className="flex items-center space-x-4 text-sm">
          <div className="flex items-center space-x-1">
            <User className="h-3 w-3 text-muted-foreground" />
            <span>{user.sessionCount}</span>
          </div>
          <div className="flex items-center space-x-1">
            <Server className="h-3 w-3 text-muted-foreground" />
            <span>{user.serverCount}</span>
          </div>
          <div className="flex items-center space-x-1">
            <FileText className="h-3 w-3 text-muted-foreground" />
            <span>{user.auditLogCount}</span>
          </div>
        </div>
      </TableCell>

      <TableCell>
        <span className="text-sm text-muted-foreground">
          {new Date(user.createdAt).toLocaleDateString()}
        </span>
      </TableCell>

      <TableCell>
        <UserActionMenu
          user={user}
          onEditRole={onEditRole}
          onToggleStatus={onToggleStatus}
          onDelete={onDelete}
        />
      </TableCell>
    </TableRow>
  );
}

export default function UserManagement() {
  const [searchTerm, setSearchTerm] = useState('');
  const [roleFilter, setRoleFilter] = useState<string>('all');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [selectedUser, setSelectedUser] = useState<UserManagementData | null>(
    null
  );
  const [showRoleDialog, setShowRoleDialog] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  const { data: users, isLoading, error } = useAdminUsers();
  const updateUserRole = useUpdateUserRole();
  const toggleUserStatus = useToggleUserStatus();
  const deleteUser = useDeleteUser();

  const filteredUsers =
    users?.filter(user => {
      const matchesSearch =
        user.name?.toLowerCase().includes(searchTerm.toLowerCase()) ||
        user.email.toLowerCase().includes(searchTerm.toLowerCase());
      const matchesRole = roleFilter === 'all' || user.role === roleFilter;
      const matchesStatus =
        statusFilter === 'all' ||
        (statusFilter === 'active' && user.active) ||
        (statusFilter === 'inactive' && !user.active);

      return matchesSearch && matchesRole && matchesStatus;
    }) || [];

  const handleEditRole = (user: UserManagementData) => {
    setSelectedUser(user);
    setShowRoleDialog(true);
  };

  const handleToggleStatus = (user: UserManagementData) => {
    toggleUserStatus.mutate({
      userId: user.id,
      active: !user.active,
    });
  };

  const handleDelete = (user: UserManagementData) => {
    setSelectedUser(user);
    setShowDeleteDialog(true);
  };

  const handleSaveRole = (userId: string, role: string) => {
    updateUserRole.mutate(
      { userId, role },
      {
        onSuccess: () => {
          setShowRoleDialog(false);
          setSelectedUser(null);
        },
      }
    );
  };

  const handleConfirmDelete = (userId: string) => {
    deleteUser.mutate(userId, {
      onSuccess: () => {
        setShowDeleteDialog(false);
        setSelectedUser(null);
      },
    });
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-10 w-32" />
        </div>
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <Card>
        <CardContent className="pt-6">
          <div className="text-center">
            <p className="text-red-600">
              Failed to load users: {error.message}
            </p>
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
          <h2 className="text-2xl font-bold tracking-tight">User Management</h2>
          <p className="text-muted-foreground">
            Manage user accounts, roles, and permissions
          </p>
        </div>

        <Button>
          <Plus className="h-4 w-4 mr-2" />
          Add User
        </Button>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center space-x-4">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Search users..."
                  value={searchTerm}
                  onChange={e => setSearchTerm(e.target.value)}
                  className="pl-10"
                />
              </div>
            </div>

            <Select value={roleFilter} onValueChange={setRoleFilter}>
              <SelectTrigger className="w-40">
                <SelectValue placeholder="All Roles" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Roles</SelectItem>
                <SelectItem value="user">User</SelectItem>
                <SelectItem value="admin">Admin</SelectItem>
                <SelectItem value="super_admin">Super Admin</SelectItem>
                <SelectItem value="system_admin">System Admin</SelectItem>
              </SelectContent>
            </Select>

            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-32">
                <SelectValue placeholder="All Status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Status</SelectItem>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="inactive">Inactive</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* Users Table */}
      <Card>
        <CardHeader>
          <CardTitle>Users ({filteredUsers.length})</CardTitle>
          <CardDescription>
            Showing {filteredUsers.length} of {users?.length || 0} users
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>User</TableHead>
                <TableHead>Role</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Last Activity</TableHead>
                <TableHead>Usage</TableHead>
                <TableHead>Created</TableHead>
                <TableHead className="w-[70px]">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredUsers.map(user => (
                <UserRow
                  key={user.id}
                  user={user}
                  onEditRole={handleEditRole}
                  onToggleStatus={handleToggleStatus}
                  onDelete={handleDelete}
                />
              ))}
            </TableBody>
          </Table>

          {filteredUsers.length === 0 && (
            <div className="text-center py-8 text-muted-foreground">
              No users found matching your criteria
            </div>
          )}
        </CardContent>
      </Card>

      {/* Dialogs */}
      <RoleEditDialog
        user={selectedUser}
        isOpen={showRoleDialog}
        onClose={() => {
          setShowRoleDialog(false);
          setSelectedUser(null);
        }}
        onSave={handleSaveRole}
        isLoading={updateUserRole.isPending}
      />

      <DeleteUserDialog
        user={selectedUser}
        isOpen={showDeleteDialog}
        onClose={() => {
          setShowDeleteDialog(false);
          setSelectedUser(null);
        }}
        onConfirm={handleConfirmDelete}
        isLoading={deleteUser.isPending}
      />
    </div>
  );
}
