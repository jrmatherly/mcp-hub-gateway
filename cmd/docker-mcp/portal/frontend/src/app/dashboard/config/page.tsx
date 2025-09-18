'use client';

import { Download, FileText, Plus, Save, Settings, Upload } from 'lucide-react';
import React, { useState } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { ConfigurationEditor } from '@/components/config/ConfigurationEditor';
import { ConfigurationList } from '@/components/config/ConfigurationList';
import { ImportExportDialog } from '@/components/config/ImportExportDialog';
import { TemplateSelector } from '@/components/config/TemplateSelector';
import { ConfigurationDiff } from '@/components/config/ConfigurationDiff';
import { useConfig } from '@/hooks/useConfig';
import type { MCPConfig } from '@/types/api';

type ViewMode = 'list' | 'editor' | 'diff' | 'templates';

export default function ConfigurationPage() {
  const [viewMode, setViewMode] = useState<ViewMode>('list');
  const [selectedConfig, setSelectedConfig] = useState<MCPConfig | null>(null);
  const [comparisonConfig, setComparisonConfig] = useState<MCPConfig | null>(
    null
  );
  const [showImportExport, setShowImportExport] = useState(false);
  const [importExportMode, setImportExportMode] = useState<'import' | 'export'>(
    'export'
  );

  const {
    data: currentConfig,
    isLoading,
    error,
    updateConfig,
    exportConfig,
    importConfig,
  } = useConfig();

  const handleConfigSelect = (config: MCPConfig) => {
    setSelectedConfig(config);
    setViewMode('editor');
  };

  const handleConfigSave = async (config: MCPConfig) => {
    try {
      await updateConfig.mutateAsync(config);
      toast.success('Configuration saved successfully');
      setViewMode('list');
    } catch (error) {
      toast.error('Failed to save configuration');
      console.error('Save error:', error);
    }
  };

  const handleExport = async (format: 'json' | 'yaml') => {
    try {
      const data = await exportConfig.mutateAsync({ format });
      const blob = new Blob([data], {
        type: format === 'json' ? 'application/json' : 'application/x-yaml',
      });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `mcp-config.${format}`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
      toast.success(`Configuration exported as ${format.toUpperCase()}`);
    } catch (error) {
      toast.error('Failed to export configuration');
      console.error('Export error:', error);
    }
  };

  const handleImport = async (data: string, format: 'json' | 'yaml') => {
    try {
      await importConfig.mutateAsync({ data, format });
      toast.success('Configuration imported successfully');
      setShowImportExport(false);
    } catch (error) {
      toast.error('Failed to import configuration');
      console.error('Import error:', error);
    }
  };

  const handleDiffView = (config1: MCPConfig, config2: MCPConfig) => {
    setSelectedConfig(config1);
    setComparisonConfig(config2);
    setViewMode('diff');
  };

  const handleTemplateApply = (template: MCPConfig) => {
    setSelectedConfig(template);
    setViewMode('editor');
    toast.info('Template loaded. Review and save to apply.');
  };

  if (error) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="pt-6">
            <div className="text-center text-red-600">
              Failed to load configuration: {error.message}
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Configuration Management</h1>
          <p className="text-muted-foreground mt-2">
            Manage your MCP server configurations, import/export settings, and
            apply templates.
          </p>
        </div>

        <div className="flex items-center space-x-3">
          <Select
            value={viewMode}
            onValueChange={(value: ViewMode) => setViewMode(value)}
          >
            <SelectTrigger className="w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="list">
                <div className="flex items-center space-x-2">
                  <FileText className="h-4 w-4" />
                  <span>List View</span>
                </div>
              </SelectItem>
              <SelectItem value="editor">
                <div className="flex items-center space-x-2">
                  <Settings className="h-4 w-4" />
                  <span>Editor</span>
                </div>
              </SelectItem>
              <SelectItem value="diff">
                <div className="flex items-center space-x-2">
                  <FileText className="h-4 w-4" />
                  <span>Compare</span>
                </div>
              </SelectItem>
              <SelectItem value="templates">
                <div className="flex items-center space-x-2">
                  <Plus className="h-4 w-4" />
                  <span>Templates</span>
                </div>
              </SelectItem>
            </SelectContent>
          </Select>

          <Button
            variant="outline"
            onClick={() => {
              setImportExportMode('import');
              setShowImportExport(true);
            }}
          >
            <Upload className="h-4 w-4 mr-2" />
            Import
          </Button>

          <Button
            variant="outline"
            onClick={() => {
              setImportExportMode('export');
              setShowImportExport(true);
            }}
          >
            <Download className="h-4 w-4 mr-2" />
            Export
          </Button>

          {selectedConfig && viewMode === 'editor' && (
            <Button onClick={() => handleConfigSave(selectedConfig)}>
              <Save className="h-4 w-4 mr-2" />
              Save Changes
            </Button>
          )}
        </div>
      </div>

      {/* Main Content */}
      <div className="space-y-6">
        {viewMode === 'list' && (
          <ConfigurationList
            currentConfig={currentConfig}
            isLoading={isLoading}
            onConfigSelect={handleConfigSelect}
            onDiffView={handleDiffView}
          />
        )}

        {viewMode === 'editor' && (
          <ConfigurationEditor
            config={selectedConfig || currentConfig}
            isLoading={isLoading}
            onChange={setSelectedConfig}
            onSave={handleConfigSave}
          />
        )}

        {viewMode === 'diff' && selectedConfig && comparisonConfig && (
          <ConfigurationDiff
            config1={selectedConfig}
            config2={comparisonConfig}
            onBack={() => setViewMode('list')}
          />
        )}

        {viewMode === 'templates' && (
          <TemplateSelector
            onTemplateApply={handleTemplateApply}
            currentConfig={currentConfig}
          />
        )}
      </div>

      {/* Import/Export Dialog */}
      <ImportExportDialog
        open={showImportExport}
        onOpenChange={setShowImportExport}
        mode={importExportMode}
        onExport={handleExport}
        onImport={handleImport}
        isLoading={exportConfig.isPending || importConfig.isPending}
      />
    </div>
  );
}
