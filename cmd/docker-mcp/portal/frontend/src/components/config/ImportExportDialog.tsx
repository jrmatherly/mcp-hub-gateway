'use client';

import { Download, FileText, Loader2, Upload } from 'lucide-react';
import React, { useState } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { cn } from '@/lib/utils';

interface ImportExportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode: 'import' | 'export';
  onExport?: (format: 'json' | 'yaml') => void;
  onImport?: (data: string, format: 'json' | 'yaml') => void;
  isLoading?: boolean;
}

function FileUpload({
  onFileSelect,
  accept,
  className,
}: {
  onFileSelect: (content: string, filename: string) => void;
  accept?: string;
  className?: string;
}) {
  const [isDragOver, setIsDragOver] = useState(false);

  const handleFileRead = (file: File) => {
    const reader = new FileReader();
    reader.onload = e => {
      const content = e.target?.result as string;
      onFileSelect(content, file.name);
    };
    reader.readAsText(file);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(false);
    const files = Array.from(e.dataTransfer.files);
    if (files.length > 0) {
      handleFileRead(files[0]);
    }
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || []);
    if (files.length > 0) {
      handleFileRead(files[0]);
    }
  };

  return (
    <div
      className={cn(
        'border-2 border-dashed rounded-lg p-6 text-center transition-colors',
        isDragOver
          ? 'border-primary bg-primary/5'
          : 'border-muted-foreground/25 hover:border-muted-foreground/50',
        className
      )}
      onDrop={handleDrop}
      onDragOver={e => {
        e.preventDefault();
        setIsDragOver(true);
      }}
      onDragLeave={() => setIsDragOver(false)}
    >
      <Upload className="h-10 w-10 mx-auto mb-4 text-muted-foreground" />
      <div className="space-y-2">
        <p className="text-sm font-medium">
          Drop your configuration file here, or{' '}
          <label className="text-primary hover:underline cursor-pointer">
            browse
            <input
              type="file"
              className="hidden"
              accept={accept}
              onChange={handleFileChange}
            />
          </label>
        </p>
        <p className="text-xs text-muted-foreground">
          Supports JSON and YAML formats
        </p>
      </div>
    </div>
  );
}

function ExportTab({
  onExport,
  isLoading,
}: {
  onExport: (format: 'json' | 'yaml') => void;
  isLoading: boolean;
}) {
  const [format, setFormat] = useState<'json' | 'yaml'>('json');
  const [includeSecrets, setIncludeSecrets] = useState(false);

  const handleExport = () => {
    onExport(format);
  };

  return (
    <div className="space-y-6">
      <div className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="export-format">Export Format</Label>
          <Select
            value={format}
            onValueChange={(value: 'json' | 'yaml') => setFormat(value)}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="json">JSON</SelectItem>
              <SelectItem value="yaml">YAML</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-3">
          <Label>Export Options</Label>
          <div className="space-y-2">
            <label className="flex items-center space-x-2">
              <input
                type="checkbox"
                checked={includeSecrets}
                onChange={e => setIncludeSecrets(e.target.checked)}
                className="rounded border-gray-300"
              />
              <span className="text-sm">
                Include secrets (not recommended for sharing)
              </span>
            </label>
          </div>
        </div>
      </div>

      <div className="bg-yellow-50 dark:bg-yellow-900/20 p-4 rounded-md">
        <h4 className="text-sm font-medium text-yellow-800 dark:text-yellow-200 mb-2">
          Security Notice
        </h4>
        <p className="text-sm text-yellow-700 dark:text-yellow-300">
          Exported configurations may contain sensitive information. Only share
          configuration files with trusted parties and avoid including secrets
          when possible.
        </p>
      </div>

      <Button onClick={handleExport} disabled={isLoading} className="w-full">
        {isLoading ? (
          <>
            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
            Exporting...
          </>
        ) : (
          <>
            <Download className="h-4 w-4 mr-2" />
            Export Configuration
          </>
        )}
      </Button>
    </div>
  );
}

function ImportTab({
  onImport,
  isLoading,
}: {
  onImport: (data: string, format: 'json' | 'yaml') => void;
  isLoading: boolean;
}) {
  const [importMethod, setImportMethod] = useState<'file' | 'text'>('file');
  const [textContent, setTextContent] = useState('');
  const [detectedFormat, setDetectedFormat] = useState<'json' | 'yaml' | null>(
    null
  );
  const [fileName, setFileName] = useState('');

  const detectFormat = (content: string): 'json' | 'yaml' => {
    const trimmed = content.trim();
    if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
      return 'json';
    }
    return 'yaml';
  };

  const handleFileSelect = (content: string, filename: string) => {
    setTextContent(content);
    setFileName(filename);
    setDetectedFormat(detectFormat(content));
    setImportMethod('file');
  };

  const handleTextChange = (content: string) => {
    setTextContent(content);
    if (content.trim()) {
      setDetectedFormat(detectFormat(content));
    } else {
      setDetectedFormat(null);
    }
  };

  const handleImport = () => {
    if (!textContent.trim()) {
      toast.error('Please provide configuration content');
      return;
    }

    const format = detectedFormat || 'json';
    onImport(textContent, format);
  };

  const validateContent = () => {
    if (!textContent.trim()) return { valid: false, error: 'Content is empty' };

    try {
      if (detectedFormat === 'json') {
        JSON.parse(textContent);
      }
      return { valid: true, error: null };
    } catch (error) {
      return {
        valid: false,
        error: `Invalid ${detectedFormat?.toUpperCase()}: ${error instanceof Error ? error.message : 'Unknown error'}`,
      };
    }
  };

  const validation = validateContent();

  return (
    <div className="space-y-6">
      <Tabs
        value={importMethod}
        onValueChange={value => setImportMethod(value as 'file' | 'text')}
      >
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="file">Upload File</TabsTrigger>
          <TabsTrigger value="text">Paste Text</TabsTrigger>
        </TabsList>

        <TabsContent value="file" className="mt-4">
          <FileUpload
            onFileSelect={handleFileSelect}
            accept=".json,.yaml,.yml"
          />
          {fileName && (
            <div className="mt-4 p-3 bg-green-50 dark:bg-green-900/20 rounded-md">
              <div className="flex items-center space-x-2">
                <FileText className="h-4 w-4 text-green-600" />
                <span className="text-sm font-medium text-green-800 dark:text-green-200">
                  {fileName}
                </span>
                {detectedFormat && (
                  <span className="text-xs px-2 py-1 bg-green-100 dark:bg-green-800 text-green-800 dark:text-green-100 rounded">
                    {detectedFormat.toUpperCase()}
                  </span>
                )}
              </div>
            </div>
          )}
        </TabsContent>

        <TabsContent value="text" className="mt-4">
          <div className="space-y-2">
            <Label htmlFor="config-text">Configuration Content</Label>
            <textarea
              id="config-text"
              value={textContent}
              onChange={e => handleTextChange(e.target.value)}
              placeholder="Paste your JSON or YAML configuration here..."
              className="w-full h-64 p-3 border rounded-md font-mono text-sm resize-none focus:outline-none focus:ring-2 focus:ring-ring"
            />
            {detectedFormat && (
              <div className="flex items-center space-x-2">
                <span className="text-xs text-muted-foreground">
                  Detected format:
                </span>
                <span className="text-xs px-2 py-1 bg-muted text-muted-foreground rounded">
                  {detectedFormat.toUpperCase()}
                </span>
              </div>
            )}
          </div>
        </TabsContent>
      </Tabs>

      {/* Validation */}
      {textContent && (
        <div
          className={cn(
            'p-3 rounded-md text-sm',
            validation.valid
              ? 'bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-300'
              : 'bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-300'
          )}
        >
          {validation.valid
            ? '✓ Configuration appears valid'
            : `✗ ${validation.error}`}
        </div>
      )}

      {/* Import Options */}
      <div className="space-y-3">
        <Label>Import Options</Label>
        <div className="space-y-2">
          <label className="flex items-center space-x-2">
            <input
              type="checkbox"
              defaultChecked
              className="rounded border-gray-300"
            />
            <span className="text-sm">Merge with existing configuration</span>
          </label>
          <label className="flex items-center space-x-2">
            <input type="checkbox" className="rounded border-gray-300" />
            <span className="text-sm">Create backup before import</span>
          </label>
        </div>
      </div>

      <Button
        onClick={handleImport}
        disabled={!validation.valid || isLoading}
        className="w-full"
      >
        {isLoading ? (
          <>
            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
            Importing...
          </>
        ) : (
          <>
            <Upload className="h-4 w-4 mr-2" />
            Import Configuration
          </>
        )}
      </Button>
    </div>
  );
}

export function ImportExportDialog({
  open,
  onOpenChange,
  mode,
  onExport,
  onImport,
  isLoading = false,
}: ImportExportDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle className="flex items-center space-x-2">
            {mode === 'export' ? (
              <Download className="h-5 w-5" />
            ) : (
              <Upload className="h-5 w-5" />
            )}
            <span>
              {mode === 'export'
                ? 'Export Configuration'
                : 'Import Configuration'}
            </span>
          </DialogTitle>
          <DialogDescription>
            {mode === 'export'
              ? 'Download your current configuration in JSON or YAML format.'
              : 'Upload or paste a configuration file to import settings.'}
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          {mode === 'export' && onExport ? (
            <ExportTab onExport={onExport} isLoading={isLoading} />
          ) : mode === 'import' && onImport ? (
            <ImportTab onImport={onImport} isLoading={isLoading} />
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              Invalid configuration
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
