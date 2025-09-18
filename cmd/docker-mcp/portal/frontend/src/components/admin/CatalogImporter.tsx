'use client';

import { useState } from 'react';
import {
  Upload,
  FileText,
  AlertCircle,
  CheckCircle,
  Download,
  Copy,
} from 'lucide-react';

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
import { Textarea } from '@/components/ui/textarea';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { TooltipProvider } from '@/components/ui/tooltip';

import type { CatalogImporterProps } from '@/types/catalog';

export function CatalogImporter({
  isOpen,
  onClose,
  onImport,
  isLoading,
}: CatalogImporterProps) {
  const [importMethod, setImportMethod] = useState<'file' | 'text' | 'url'>(
    'file'
  );
  const [format, setFormat] = useState<'json' | 'yaml'>('json');
  const [fileContent, setFileContent] = useState('');
  const [sourceUrl, setSourceUrl] = useState('');
  const [validationError, setValidationError] = useState<string | null>(null);
  const [isValidContent, setIsValidContent] = useState(false);

  const handleFileUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    // Determine format from file extension
    const extension = file.name.split('.').pop()?.toLowerCase();
    if (extension === 'yaml' || extension === 'yml') {
      setFormat('yaml');
    } else if (extension === 'json') {
      setFormat('json');
    }

    const reader = new FileReader();
    reader.onload = e => {
      const content = e.target?.result as string;
      setFileContent(content);
      validateContent(content, format);
    };
    reader.readAsText(file);
  };

  const handleTextChange = (content: string) => {
    setFileContent(content);
    validateContent(content, format);
  };

  const validateContent = (content: string, contentFormat: 'json' | 'yaml') => {
    if (!content.trim()) {
      setValidationError(null);
      setIsValidContent(false);
      return;
    }

    try {
      if (contentFormat === 'json') {
        const parsed = JSON.parse(content);
        // Basic validation for catalog structure
        if (typeof parsed === 'object' && parsed !== null) {
          if (parsed.name && parsed.display_name) {
            setValidationError(null);
            setIsValidContent(true);
          } else {
            setValidationError(
              'Catalog must have "name" and "display_name" fields'
            );
            setIsValidContent(false);
          }
        } else {
          setValidationError('Invalid JSON structure');
          setIsValidContent(false);
        }
      } else {
        // For YAML, we'll do basic validation
        // In a real implementation, you'd use a YAML parser
        if (content.includes('name:') && content.includes('display_name:')) {
          setValidationError(null);
          setIsValidContent(true);
        } else {
          setValidationError(
            'YAML catalog must have "name" and "display_name" fields'
          );
          setIsValidContent(false);
        }
      }
    } catch (error) {
      setValidationError(
        `Invalid ${contentFormat.toUpperCase()}: ${error instanceof Error ? error.message : 'Unknown error'}`
      );
      setIsValidContent(false);
    }
  };

  const handleFormatChange = (newFormat: 'json' | 'yaml') => {
    setFormat(newFormat);
    if (fileContent) {
      validateContent(fileContent, newFormat);
    }
  };

  const handleImport = () => {
    if (!fileContent.trim()) {
      setValidationError('Please provide catalog content to import');
      return;
    }

    if (!isValidContent) {
      setValidationError('Please fix validation errors before importing');
      return;
    }

    onImport(fileContent, format);
  };

  const handleClose = () => {
    setFileContent('');
    setSourceUrl('');
    setValidationError(null);
    setIsValidContent(false);
    setImportMethod('file');
    setFormat('json');
    onClose();
  };

  const exampleCatalog = {
    name: 'example-catalog',
    display_name: 'Example Catalog',
    description: 'An example admin base catalog',
    type: 'admin_base',
    status: 'active',
    is_public: true,
    is_default: false,
    tags: ['example', 'demo'],
    registry: {
      'example-server': {
        name: 'example-server',
        display_name: 'Example Server',
        description: 'An example MCP server',
        image: 'example/server:latest',
        environment: {
          API_KEY: '${API_KEY}',
        },
        is_enabled: true,
      },
    },
  };

  const exampleYaml = `name: example-catalog
display_name: Example Catalog
description: An example admin base catalog
type: admin_base
status: active
is_public: true
is_default: false
tags:
  - example
  - demo
registry:
  example-server:
    name: example-server
    display_name: Example Server
    description: An example MCP server
    image: example/server:latest
    environment:
      API_KEY: \${API_KEY}
    is_enabled: true`;

  const copyExample = () => {
    const example =
      format === 'json' ? JSON.stringify(exampleCatalog, null, 2) : exampleYaml;

    navigator.clipboard.writeText(example);
    setFileContent(example);
    validateContent(example, format);
  };

  return (
    <TooltipProvider>
      <Dialog open={isOpen} onOpenChange={handleClose}>
        <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Upload className="h-5 w-5" />
              Import Catalog
            </DialogTitle>
            <DialogDescription>
              Import an admin base catalog from a file, text, or URL. The
              catalog will be available to all users.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-6">
            {/* Format Selection */}
            <div className="space-y-2">
              <Label>Format</Label>
              <Select value={format} onValueChange={handleFormatChange}>
                <SelectTrigger className="w-48">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="json">JSON</SelectItem>
                  <SelectItem value="yaml">YAML</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Import Method Tabs */}
            <Tabs
              value={importMethod}
              onValueChange={value =>
                setImportMethod(value as 'file' | 'text' | 'url')
              }
            >
              <TabsList className="grid w-full grid-cols-3">
                <TabsTrigger value="file">Upload File</TabsTrigger>
                <TabsTrigger value="text">Paste Text</TabsTrigger>
                <TabsTrigger value="url">From URL</TabsTrigger>
              </TabsList>

              <TabsContent value="file" className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="file-upload">
                    Select Catalog File (.json or .yaml)
                  </Label>
                  <Input
                    id="file-upload"
                    type="file"
                    accept=".json,.yaml,.yml"
                    onChange={handleFileUpload}
                    className="file:mr-4 file:py-2 file:px-4 file:rounded-md file:border-0 file:text-sm file:font-medium file:bg-primary file:text-primary-foreground hover:file:bg-primary/90"
                  />
                </div>
              </TabsContent>

              <TabsContent value="text" className="space-y-4">
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <Label htmlFor="catalog-text">
                      Catalog Content ({format.toUpperCase()})
                    </Label>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={copyExample}
                      className="gap-2"
                    >
                      <Copy className="h-4 w-4" />
                      Use Example
                    </Button>
                  </div>
                  <Textarea
                    id="catalog-text"
                    placeholder={`Paste your ${format.toUpperCase()} catalog content here...`}
                    value={fileContent}
                    onChange={e => handleTextChange(e.target.value)}
                    rows={12}
                    className="font-mono text-sm"
                  />
                </div>
              </TabsContent>

              <TabsContent value="url" className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="catalog-url">Catalog URL</Label>
                  <Input
                    id="catalog-url"
                    type="url"
                    placeholder="https://example.com/catalog.json"
                    value={sourceUrl}
                    onChange={e => setSourceUrl(e.target.value)}
                  />
                  <p className="text-sm text-muted-foreground">
                    The URL should point to a publicly accessible catalog file.
                  </p>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    // In a real implementation, you'd fetch from the URL
                    setValidationError('URL import not yet implemented');
                  }}
                  disabled={!sourceUrl}
                >
                  <Download className="h-4 w-4 mr-2" />
                  Fetch from URL
                </Button>
              </TabsContent>
            </Tabs>

            {/* Validation Status */}
            {fileContent && (
              <div className="space-y-2">
                <Label>Validation Status</Label>
                <div
                  className={`p-3 rounded-md border ${
                    isValidContent
                      ? 'border-green-200 bg-green-50 text-green-800'
                      : validationError
                        ? 'border-red-200 bg-red-50 text-red-800'
                        : 'border-muted bg-muted'
                  }`}
                >
                  <div className="flex items-center gap-2">
                    {isValidContent ? (
                      <CheckCircle className="h-4 w-4" />
                    ) : validationError ? (
                      <AlertCircle className="h-4 w-4" />
                    ) : (
                      <FileText className="h-4 w-4" />
                    )}
                    <span className="text-sm font-medium">
                      {isValidContent
                        ? 'Valid catalog format'
                        : validationError || 'Ready for validation'}
                    </span>
                  </div>
                  {validationError && (
                    <p className="text-sm mt-1">{validationError}</p>
                  )}
                </div>
              </div>
            )}

            {/* Example Structure */}
            <div className="space-y-2">
              <Label>Example Catalog Structure</Label>
              <div className="p-4 bg-muted rounded-md">
                <pre className="text-sm text-muted-foreground overflow-x-auto">
                  {format === 'json'
                    ? JSON.stringify(exampleCatalog, null, 2)
                    : exampleYaml}
                </pre>
              </div>
            </div>

            {/* Import Information */}
            <div className="bg-blue-50 border border-blue-200 rounded-md p-4">
              <div className="flex items-start gap-2">
                <AlertCircle className="h-5 w-5 text-blue-600 mt-0.5" />
                <div className="space-y-1">
                  <p className="text-sm font-medium text-blue-800">
                    Import Information
                  </p>
                  <ul className="text-sm text-blue-700 space-y-1">
                    <li>
                      • The imported catalog will be created as an admin base
                      catalog
                    </li>
                    <li>• All users will inherit servers from this catalog</li>
                    <li>
                      • Existing catalogs with the same name will not be
                      overwritten
                    </li>
                    <li>• You can modify the catalog after import if needed</li>
                  </ul>
                </div>
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            <Button
              onClick={handleImport}
              disabled={!isValidContent || isLoading}
              className="min-w-[100px]"
            >
              {isLoading ? (
                'Importing...'
              ) : (
                <>
                  <Upload className="h-4 w-4 mr-2" />
                  Import Catalog
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </TooltipProvider>
  );
}
