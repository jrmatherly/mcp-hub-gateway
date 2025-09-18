'use client';

/**
 * Dynamic Grid Layout Component
 *
 * Code-split grid layout with proper client-side handling
 * and responsive behavior.
 */

import { useEffect, useState, useCallback } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { GridLayout, preloadHeavyComponents } from '@/lib/dynamic-imports';
import { GripVertical, X } from 'lucide-react';

// Import CSS for react-grid-layout
import 'react-grid-layout/css/styles.css';
import 'react-resizable/css/styles.css';

interface GridItem {
  i: string;
  x: number;
  y: number;
  w: number;
  h: number;
  minW?: number;
  minH?: number;
  maxW?: number;
  maxH?: number;
}

interface DashboardWidget {
  id: string;
  title: string;
  content: React.ReactNode;
  defaultSize?: { w: number; h: number };
}

interface DynamicGridLayoutProps {
  widgets: DashboardWidget[];
  editable?: boolean;
  onLayoutChange?: (layout: GridItem[]) => void;
  className?: string;
}

export function DynamicGridLayout({
  widgets,
  editable = false,
  onLayoutChange,
  className,
}: DynamicGridLayoutProps) {
  const [isClient, setIsClient] = useState(false);
  const [gridLoaded, setGridLoaded] = useState(false);
  const [layout, setLayout] = useState<GridItem[]>([]);
  const [isEditing, setIsEditing] = useState(false);

  // Generate initial layout from widgets
  useEffect(() => {
    const initialLayout: GridItem[] = widgets.map((widget, index) => ({
      i: widget.id,
      x: (index * 2) % 12,
      y: Math.floor(index / 6),
      w: widget.defaultSize?.w || 2,
      h: widget.defaultSize?.h || 2,
      minW: 1,
      minH: 1,
    }));
    setLayout(initialLayout);
  }, [widgets]);

  // Ensure we're on the client side and preload grid layout
  useEffect(() => {
    setIsClient(true);
    preloadHeavyComponents.gridLayout().then(() => {
      setGridLoaded(true);
    });
  }, []);

  const handleLayoutChange = useCallback(
    (newLayout: GridItem[]) => {
      setLayout(newLayout);
      onLayoutChange?.(newLayout);
    },
    [onLayoutChange]
  );

  const toggleEditMode = useCallback(() => {
    setIsEditing(!isEditing);
  }, [isEditing]);

  const removeWidget = useCallback((widgetId: string) => {
    setLayout(prevLayout => prevLayout.filter(item => item.i !== widgetId));
  }, []);

  // Show loading state until client-side and grid is loaded
  if (!isClient || !gridLoaded) {
    return (
      <div className={className}>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {Array.from({ length: widgets.length }).map((_, i) => (
            <Card key={i}>
              <CardHeader>
                <Skeleton className="h-4 w-3/4" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-32 w-full" />
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    );
  }

  const breakpoints = { lg: 1200, md: 996, sm: 768, xs: 480, xxs: 0 };
  const cols = { lg: 12, md: 10, sm: 6, xs: 4, xxs: 2 };

  return (
    <div className={className}>
      {editable && (
        <div className="mb-4 flex justify-between items-center">
          <div className="flex gap-2">
            <Button
              onClick={toggleEditMode}
              variant={isEditing ? 'default' : 'outline'}
              size="sm"
            >
              <GripVertical className="w-4 h-4 mr-2" />
              {isEditing ? 'Done Editing' : 'Edit Layout'}
            </Button>
          </div>
        </div>
      )}

      <GridLayout
        className="layout"
        layouts={{ lg: layout }}
        onLayoutChange={handleLayoutChange}
        breakpoints={breakpoints}
        cols={cols}
        rowHeight={60}
        isDraggable={editable && isEditing}
        isResizable={editable && isEditing}
        margin={[16, 16]}
        containerPadding={[0, 0]}
        useCSSTransforms={true}
        preventCollision={false}
        compactType="vertical"
      >
        {widgets.map(widget => (
          <div key={widget.id} className="relative">
            <Card className="h-full">
              <CardHeader className="pb-2">
                <div className="flex justify-between items-center">
                  <CardTitle className="text-sm font-medium">
                    {widget.title}
                  </CardTitle>
                  {editable && isEditing && (
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => removeWidget(widget.id)}
                      className="h-6 w-6 p-0 text-muted-foreground hover:text-destructive"
                    >
                      <X className="w-4 h-4" />
                    </Button>
                  )}
                </div>
              </CardHeader>
              <CardContent className="pt-0 h-[calc(100%-4rem)] overflow-hidden">
                <div className="h-full overflow-auto">{widget.content}</div>
              </CardContent>
            </Card>
            {editable && isEditing && (
              <div className="absolute top-2 right-8 opacity-50 hover:opacity-100">
                <GripVertical className="w-4 h-4 text-muted-foreground cursor-move" />
              </div>
            )}
          </div>
        ))}
      </GridLayout>
    </div>
  );
}

/**
 * Example usage component
 */
export function DashboardWithGridLayout() {
  const [isClient, setIsClient] = useState(false);

  useEffect(() => {
    setIsClient(true);
  }, []);

  const sampleWidgets: DashboardWidget[] = [
    {
      id: 'servers',
      title: 'Active Servers',
      content: (
        <div className="space-y-2">
          <div className="text-2xl font-bold">12</div>
          <div className="text-sm text-muted-foreground">
            Running containers
          </div>
        </div>
      ),
      defaultSize: { w: 2, h: 2 },
    },
    {
      id: 'cpu',
      title: 'CPU Usage',
      content: (
        <div className="space-y-2">
          <div className="text-2xl font-bold">45%</div>
          <div className="text-sm text-muted-foreground">Average load</div>
        </div>
      ),
      defaultSize: { w: 2, h: 2 },
    },
    {
      id: 'memory',
      title: 'Memory Usage',
      content: (
        <div className="space-y-2">
          <div className="text-2xl font-bold">2.1GB</div>
          <div className="text-sm text-muted-foreground">Of 8GB available</div>
        </div>
      ),
      defaultSize: { w: 2, h: 2 },
    },
    {
      id: 'requests',
      title: 'Recent Activity',
      content: (
        <div className="space-y-1">
          <div className="text-xs">Server enabled: web-search</div>
          <div className="text-xs">Server disabled: file-manager</div>
          <div className="text-xs">Config updated</div>
        </div>
      ),
      defaultSize: { w: 4, h: 3 },
    },
  ];

  if (!isClient) {
    return <Skeleton className="h-96 w-full" />;
  }

  return (
    <DynamicGridLayout
      widgets={sampleWidgets}
      editable={true}
      onLayoutChange={layout => {
        console.warn('Layout changed:', layout);
        // Save layout to localStorage or API
      }}
      className="min-h-96"
    />
  );
}
