'use client';

import { Download, Play, RotateCcw, Square, Trash2 } from 'lucide-react';
import React from 'react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { useBulkServerOperation } from '@/hooks/api';
import type { BulkServerOperation } from '@/types/api';

interface ServerBulkActionsProps {
  selectedServers: Set<string>;
  onClearSelection: () => void;
  onSelectAll: () => void;
  totalServers: number;
  className?: string;
}

export function ServerBulkActions({
  selectedServers,
  onClearSelection,
  onSelectAll,
  totalServers,
  className,
}: ServerBulkActionsProps) {
  const bulkOperation = useBulkServerOperation();

  const selectedCount = selectedServers.size;

  const handleBulkOperation = async (
    operation: BulkServerOperation['operation']
  ) => {
    if (selectedCount === 0) return;

    try {
      await bulkOperation.mutateAsync({
        server_names: Array.from(selectedServers),
        operation,
      });
    } catch {
      // Error handling is done in the hook
    }
  };

  const handleEnableSelected = () => handleBulkOperation('enable');
  const handleDisableSelected = () => handleBulkOperation('disable');
  const handleRestartSelected = () => handleBulkOperation('restart');
  const handleDeleteSelected = () => {
    if (
      confirm(
        `Are you sure you want to delete ${selectedCount} server(s)? This action cannot be undone.`
      )
    ) {
      handleBulkOperation('delete');
    }
  };

  const handleExportSelected = () => {
    // Implementation for exporting selected servers configuration
    const serverNames = Array.from(selectedServers);
    const exportData = {
      servers: serverNames,
      exported_at: new Date().toISOString(),
      version: '1.0',
    };

    const blob = new Blob([JSON.stringify(exportData, null, 2)], {
      type: 'application/json',
    });

    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `mcp-servers-${new Date().toISOString().split('T')[0]}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  if (selectedCount === 0) {
    return null;
  }

  return (
    <div
      className={`flex items-center justify-between p-4 bg-accent/50 border rounded-lg ${className || ''}`}
    >
      <div className="flex items-center space-x-4">
        <div className="flex items-center space-x-2">
          <Badge variant="secondary" className="font-medium">
            {selectedCount} selected
          </Badge>
          {selectedCount === totalServers && totalServers > 0 && (
            <Badge variant="outline">All servers</Badge>
          )}
        </div>

        <div className="flex items-center space-x-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={onSelectAll}
            disabled={selectedCount === totalServers}
          >
            Select All ({totalServers})
          </Button>
          <Button variant="ghost" size="sm" onClick={onClearSelection}>
            Clear Selection
          </Button>
        </div>
      </div>

      <div className="flex items-center space-x-2">
        {/* Primary actions */}
        <Button
          variant="outline"
          size="sm"
          onClick={handleEnableSelected}
          disabled={bulkOperation.isPending}
          className="flex items-center space-x-1"
        >
          <Play className="h-4 w-4" />
          <span>Enable</span>
        </Button>

        <Button
          variant="outline"
          size="sm"
          onClick={handleDisableSelected}
          disabled={bulkOperation.isPending}
          className="flex items-center space-x-1"
        >
          <Square className="h-4 w-4" />
          <span>Disable</span>
        </Button>

        <Button
          variant="outline"
          size="sm"
          onClick={handleRestartSelected}
          disabled={bulkOperation.isPending}
          className="flex items-center space-x-1"
        >
          <RotateCcw className="h-4 w-4" />
          <span>Restart</span>
        </Button>

        {/* Export action */}
        <Button
          variant="outline"
          size="sm"
          onClick={handleExportSelected}
          className="flex items-center space-x-1"
        >
          <Download className="h-4 w-4" />
          <span>Export</span>
        </Button>

        {/* Destructive actions */}
        <Button
          variant="destructive"
          size="sm"
          onClick={handleDeleteSelected}
          disabled={bulkOperation.isPending}
          className="flex items-center space-x-1"
        >
          <Trash2 className="h-4 w-4" />
          <span>Delete</span>
        </Button>
      </div>
    </div>
  );
}
