// ===========================
// Catalog Management Types
// ===========================

export type CatalogType =
  | 'official'
  | 'team'
  | 'personal'
  | 'imported'
  | 'custom'
  | 'admin_base'
  | 'system_default';

export type CatalogStatus =
  | 'active'
  | 'deprecated'
  | 'experimental'
  | 'archived';

export type ServerType =
  | 'filesystem'
  | 'database'
  | 'api'
  | 'development'
  | 'monitoring'
  | 'automation'
  | 'ml_ai'
  | 'productivity'
  | 'other';

// ===========================
// Core Catalog Interfaces
// ===========================

export interface AdminCatalog {
  id: string;
  name: string;
  display_name: string;
  description: string;
  type: CatalogType;
  status: CatalogStatus;
  version: string;
  owner_id: string;
  tenant_id: string;
  is_public: boolean;
  is_default: boolean;
  source_url?: string;
  source_type?: string;
  source_config?: Record<string, string>;
  registry?: Record<string, CatalogServerConfig>;
  disabled_servers?: Record<string, boolean>;
  metadata?: Record<string, unknown>;
  tags: string[];
  homepage?: string;
  repository?: string;
  license?: string;
  maintainer?: string;
  server_count: number;
  download_count: number;
  last_synced_at?: Date;
  created_at: Date;
  updated_at: Date;
  deleted_at?: Date;
}

export interface CatalogServerConfig {
  name: string;
  display_name?: string;
  description?: string;
  image: string;
  tag?: string;
  command?: string[];
  environment?: Record<string, string>;
  volumes?: VolumeMapping[];
  ports?: PortMapping[];
  working_dir?: string;
  metadata?: Record<string, unknown>;
  is_enabled: boolean;
  is_mandatory?: boolean;
}

export interface PortMapping {
  host: string;
  container: string;
  protocol: string;
}

export interface VolumeMapping {
  host: string;
  container: string;
  mode: string;
}

// ===========================
// Request/Response Types
// ===========================

export interface CreateCatalogRequest {
  name: string;
  display_name?: string;
  description?: string;
  type: CatalogType;
  is_public?: boolean;
  is_default?: boolean;
  tags?: string[];
  source_url?: string;
  source_type?: string;
  config?: Record<string, string>;
}

export interface UpdateCatalogRequest {
  display_name?: string;
  description?: string;
  is_public?: boolean;
  tags?: string[];
  source_url?: string;
  config?: Record<string, string>;
}

export interface UserCatalogCustomizationRequest {
  disabled_servers?: Record<string, boolean>;
  custom_servers?: Record<string, CatalogServerConfig>;
  overrides?: Record<string, Partial<CatalogServerConfig>>;
}

export interface ResolvedCatalog {
  merged_catalog: AdminCatalog;
  admin_servers: number;
  user_overrides: number;
  custom_servers: number;
  timestamp: Date;
}

export interface CatalogStats {
  total_servers: number;
  admin_servers: number;
  user_overrides: number;
  custom_servers: number;
  disabled_servers: number;
  last_updated: Date;
}

// ===========================
// Filter and Pagination
// ===========================

export interface CatalogFilter {
  type?: CatalogType[];
  status?: CatalogStatus[];
  owner_id?: string;
  tenant_id?: string;
  is_public?: boolean;
  is_default?: boolean;
  tags?: string[];
  search?: string;
  limit?: number;
  offset?: number;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

// ===========================
// UI Component Props
// ===========================

export interface CatalogTableProps {
  catalogs: AdminCatalog[];
  isLoading?: boolean;
  onEdit: (catalog: AdminCatalog) => void;
  onDelete: (catalog: AdminCatalog) => void;
  onToggleDefault: (catalog: AdminCatalog) => void;
  onTogglePublic: (catalog: AdminCatalog) => void;
}

export interface CatalogEditorProps {
  catalog?: AdminCatalog;
  isOpen: boolean;
  onClose: () => void;
  onSave: (request: CreateCatalogRequest | UpdateCatalogRequest) => void;
  isLoading?: boolean;
}

export interface CatalogImporterProps {
  isOpen: boolean;
  onClose: () => void;
  onImport: (data: string, format: 'json' | 'yaml') => void;
  isLoading?: boolean;
}
