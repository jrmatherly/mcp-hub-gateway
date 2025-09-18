'use client';

import { useState, useEffect } from 'react';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  Save,
  X,
  Plus,
  Globe,
  Lock,
  Star,
  AlertCircle,
  Info,
} from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';

import type {
  CatalogEditorProps,
  CreateCatalogRequest,
  UpdateCatalogRequest,
} from '@/types/catalog';

// Form validation schema
const catalogFormSchema = z.object({
  name: z
    .string()
    .min(3, 'Name must be at least 3 characters')
    .max(50, 'Name must be less than 50 characters')
    .regex(
      /^[a-z0-9_-]+$/,
      'Name can only contain lowercase letters, numbers, hyphens, and underscores'
    ),
  display_name: z
    .string()
    .min(3, 'Display name must be at least 3 characters')
    .max(100, 'Display name must be less than 100 characters'),
  description: z
    .string()
    .max(500, 'Description must be less than 500 characters')
    .optional(),
  type: z.enum([
    'official',
    'team',
    'personal',
    'imported',
    'custom',
    'admin_base',
    'system_default',
  ]),
  status: z.enum(['active', 'deprecated', 'experimental', 'archived']),
  is_public: z.boolean(),
  is_default: z.boolean(),
  tags: z.array(z.string()).optional(),
  source_url: z
    .union([z.literal(''), z.string().url('Must be a valid URL')])
    .optional(),
  source_type: z.string().optional(),
  homepage: z
    .union([z.literal(''), z.string().url('Must be a valid URL')])
    .optional(),
  repository: z
    .union([z.literal(''), z.string().url('Must be a valid URL')])
    .optional(),
  license: z.string().optional(),
  maintainer: z.string().optional(),
});

type CatalogFormData = z.infer<typeof catalogFormSchema>;

export function CatalogEditor({
  catalog,
  isOpen,
  onClose,
  onSave,
  isLoading,
}: CatalogEditorProps) {
  const [tags, setTags] = useState<string[]>([]);
  const [newTag, setNewTag] = useState('');

  const isEditing = !!catalog;
  const title = isEditing ? 'Edit Catalog' : 'Create New Catalog';

  const {
    register,
    control,
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { errors, isValid },
  } = useForm<CatalogFormData>({
    resolver: zodResolver(catalogFormSchema),
    defaultValues: {
      name: '',
      display_name: '',
      description: '',
      type: 'admin_base',
      status: 'active',
      is_public: true,
      is_default: false,
      tags: [],
      source_url: '',
      source_type: '',
      homepage: '',
      repository: '',
      license: '',
      maintainer: '',
    },
  });

  const watchedIsPublic = watch('is_public');
  const watchedIsDefault = watch('is_default');

  // Initialize form when catalog changes
  useEffect(() => {
    if (catalog) {
      reset({
        name: catalog.name,
        display_name: catalog.display_name || catalog.name,
        description: catalog.description || '',
        type: catalog.type,
        status: catalog.status,
        is_public: catalog.is_public,
        is_default: catalog.is_default,
        tags: catalog.tags || [],
        source_url: catalog.source_url || '',
        source_type: catalog.source_type || '',
        homepage: catalog.homepage || '',
        repository: catalog.repository || '',
        license: catalog.license || '',
        maintainer: catalog.maintainer || '',
      });
      setTags(catalog.tags || []);
    } else {
      reset();
      setTags([]);
    }
  }, [catalog, reset]);

  // Update tags in form when tags state changes
  useEffect(() => {
    setValue('tags', tags);
  }, [tags, setValue]);

  const handleAddTag = () => {
    if (newTag.trim() && !tags.includes(newTag.trim())) {
      setTags([...tags, newTag.trim()]);
      setNewTag('');
    }
  };

  const handleRemoveTag = (tagToRemove: string) => {
    setTags(tags.filter(tag => tag !== tagToRemove));
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleAddTag();
    }
  };

  const onSubmit = (data: CatalogFormData) => {
    // Build config object, filtering out undefined values
    const config: Record<string, string> = {};
    if (data.homepage) config.homepage = data.homepage;
    if (data.repository) config.repository = data.repository;
    if (data.license) config.license = data.license;
    if (data.maintainer) config.maintainer = data.maintainer;

    if (isEditing) {
      // Update request - name and type are immutable
      const request: UpdateCatalogRequest = {
        display_name: data.display_name,
        description: data.description || undefined,
        is_public: data.is_public,
        tags: data.tags?.length ? data.tags : undefined,
        source_url: data.source_url || undefined,
        config: Object.keys(config).length > 0 ? config : undefined,
      };
      onSave(request);
    } else {
      // Create request - name and type are required
      const request: CreateCatalogRequest = {
        name: data.name,
        display_name: data.display_name,
        description: data.description || undefined,
        type: data.type,
        is_public: data.is_public,
        is_default: data.is_default,
        tags: data.tags?.length ? data.tags : undefined,
        source_url: data.source_url || undefined,
        source_type: data.source_type || undefined,
        config: Object.keys(config).length > 0 ? config : undefined,
      };
      onSave(request);
    }
  };

  const handleClose = () => {
    reset();
    setTags([]);
    setNewTag('');
    onClose();
  };

  return (
    <TooltipProvider>
      <Dialog open={isOpen} onOpenChange={handleClose}>
        <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              {title}
              {isEditing && (
                <Badge variant="outline" className="text-xs">
                  {catalog?.type}
                </Badge>
              )}
            </DialogTitle>
            <DialogDescription>
              {isEditing
                ? 'Update the catalog configuration and metadata.'
                : 'Create a new admin base catalog that will be inherited by all users.'}
            </DialogDescription>
          </DialogHeader>

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
            {/* Basic Information */}
            <div className="space-y-4">
              <h3 className="text-lg font-medium">Basic Information</h3>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="name">
                    Name *
                    {isEditing && (
                      <Tooltip>
                        <TooltipTrigger>
                          <Info className="h-4 w-4 inline ml-1 text-muted-foreground" />
                        </TooltipTrigger>
                        <TooltipContent>
                          Name cannot be changed after creation
                        </TooltipContent>
                      </Tooltip>
                    )}
                  </Label>
                  <Input
                    id="name"
                    {...register('name')}
                    placeholder="my-catalog"
                    disabled={isEditing}
                    className={isEditing ? 'bg-muted' : ''}
                  />
                  {errors.name && (
                    <p className="text-sm text-red-600 flex items-center gap-1">
                      <AlertCircle className="h-4 w-4" />
                      {errors.name.message}
                    </p>
                  )}
                </div>

                <div className="space-y-2">
                  <Label htmlFor="display_name">Display Name *</Label>
                  <Input
                    id="display_name"
                    {...register('display_name')}
                    placeholder="My Catalog"
                  />
                  {errors.display_name && (
                    <p className="text-sm text-red-600 flex items-center gap-1">
                      <AlertCircle className="h-4 w-4" />
                      {errors.display_name.message}
                    </p>
                  )}
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  {...register('description')}
                  placeholder="A brief description of this catalog..."
                  rows={3}
                />
                {errors.description && (
                  <p className="text-sm text-red-600 flex items-center gap-1">
                    <AlertCircle className="h-4 w-4" />
                    {errors.description.message}
                  </p>
                )}
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="type">
                    Type
                    {isEditing && (
                      <Tooltip>
                        <TooltipTrigger>
                          <Info className="h-4 w-4 inline ml-1 text-muted-foreground" />
                        </TooltipTrigger>
                        <TooltipContent>
                          Type cannot be changed after creation
                        </TooltipContent>
                      </Tooltip>
                    )}
                  </Label>
                  <Controller
                    name="type"
                    control={control}
                    render={({ field }) => (
                      <Select
                        value={field.value}
                        onValueChange={field.onChange}
                        disabled={isEditing}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Select catalog type" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="admin_base">Admin Base</SelectItem>
                          <SelectItem value="official">Official</SelectItem>
                          <SelectItem value="team">Team</SelectItem>
                          <SelectItem value="imported">Imported</SelectItem>
                          <SelectItem value="custom">Custom</SelectItem>
                        </SelectContent>
                      </Select>
                    )}
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="status">Status</Label>
                  <Controller
                    name="status"
                    control={control}
                    render={({ field }) => (
                      <Select
                        value={field.value}
                        onValueChange={field.onChange}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Select status" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="active">Active</SelectItem>
                          <SelectItem value="experimental">
                            Experimental
                          </SelectItem>
                          <SelectItem value="deprecated">Deprecated</SelectItem>
                          <SelectItem value="archived">Archived</SelectItem>
                        </SelectContent>
                      </Select>
                    )}
                  />
                </div>
              </div>
            </div>

            {/* Visibility Settings */}
            <div className="space-y-4">
              <h3 className="text-lg font-medium">Visibility & Access</h3>

              <div className="space-y-4">
                <div className="flex items-center justify-between p-4 border rounded-lg">
                  <div className="flex items-center gap-3">
                    {watchedIsPublic ? (
                      <Globe className="h-5 w-5 text-green-600" />
                    ) : (
                      <Lock className="h-5 w-5 text-muted-foreground" />
                    )}
                    <div>
                      <Label
                        htmlFor="is_public"
                        className="text-base font-medium"
                      >
                        Public Catalog
                      </Label>
                      <p className="text-sm text-muted-foreground">
                        {watchedIsPublic
                          ? 'Available to all users in the system'
                          : 'Only available to specific users or groups'}
                      </p>
                    </div>
                  </div>
                  <Controller
                    name="is_public"
                    control={control}
                    render={({ field }) => (
                      <Switch
                        id="is_public"
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    )}
                  />
                </div>

                <div className="flex items-center justify-between p-4 border rounded-lg">
                  <div className="flex items-center gap-3">
                    <Star
                      className={`h-5 w-5 ${
                        watchedIsDefault
                          ? 'text-amber-500 fill-current'
                          : 'text-muted-foreground'
                      }`}
                    />
                    <div>
                      <Label
                        htmlFor="is_default"
                        className="text-base font-medium"
                      >
                        Default Catalog
                      </Label>
                      <p className="text-sm text-muted-foreground">
                        {watchedIsDefault
                          ? 'Automatically applied to new users'
                          : 'Users must manually enable this catalog'}
                      </p>
                    </div>
                  </div>
                  <Controller
                    name="is_default"
                    control={control}
                    render={({ field }) => (
                      <Switch
                        id="is_default"
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    )}
                  />
                </div>
              </div>
            </div>

            {/* Tags */}
            <div className="space-y-4">
              <h3 className="text-lg font-medium">Tags</h3>
              <div className="space-y-2">
                <div className="flex gap-2">
                  <Input
                    placeholder="Add a tag..."
                    value={newTag}
                    onChange={e => setNewTag(e.target.value)}
                    onKeyDown={handleKeyPress}
                    className="flex-1"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handleAddTag}
                    disabled={!newTag.trim() || tags.includes(newTag.trim())}
                  >
                    <Plus className="h-4 w-4" />
                  </Button>
                </div>
                {tags.length > 0 && (
                  <div className="flex flex-wrap gap-2">
                    {tags.map(tag => (
                      <Badge key={tag} variant="secondary" className="gap-1">
                        {tag}
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="h-auto p-0 ml-1"
                          onClick={() => handleRemoveTag(tag)}
                        >
                          <X className="h-3 w-3" />
                        </Button>
                      </Badge>
                    ))}
                  </div>
                )}
              </div>
            </div>

            {/* External Links */}
            <div className="space-y-4">
              <h3 className="text-lg font-medium">External Links</h3>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="source_url">Source URL</Label>
                  <Input
                    id="source_url"
                    {...register('source_url')}
                    placeholder="https://github.com/owner/repo"
                    type="url"
                  />
                  {errors.source_url && (
                    <p className="text-sm text-red-600 flex items-center gap-1">
                      <AlertCircle className="h-4 w-4" />
                      {errors.source_url.message}
                    </p>
                  )}
                </div>

                <div className="space-y-2">
                  <Label htmlFor="homepage">Homepage</Label>
                  <Input
                    id="homepage"
                    {...register('homepage')}
                    placeholder="https://example.com"
                    type="url"
                  />
                  {errors.homepage && (
                    <p className="text-sm text-red-600 flex items-center gap-1">
                      <AlertCircle className="h-4 w-4" />
                      {errors.homepage.message}
                    </p>
                  )}
                </div>

                <div className="space-y-2">
                  <Label htmlFor="repository">Repository</Label>
                  <Input
                    id="repository"
                    {...register('repository')}
                    placeholder="https://github.com/owner/repo"
                    type="url"
                  />
                  {errors.repository && (
                    <p className="text-sm text-red-600 flex items-center gap-1">
                      <AlertCircle className="h-4 w-4" />
                      {errors.repository.message}
                    </p>
                  )}
                </div>

                <div className="space-y-2">
                  <Label htmlFor="license">License</Label>
                  <Input
                    id="license"
                    {...register('license')}
                    placeholder="MIT, Apache 2.0, etc."
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="maintainer">Maintainer</Label>
                <Input
                  id="maintainer"
                  {...register('maintainer')}
                  placeholder="Name or email of the maintainer"
                />
              </div>
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={!isValid || isLoading}
                className="min-w-[100px]"
              >
                {isLoading ? (
                  'Saving...'
                ) : (
                  <>
                    <Save className="h-4 w-4 mr-2" />
                    {isEditing ? 'Update' : 'Create'}
                  </>
                )}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </TooltipProvider>
  );
}
