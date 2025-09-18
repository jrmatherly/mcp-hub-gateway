'use client';

/**
 * Dynamic Charts Component
 *
 * Example implementation of code-split charts with proper
 * client-side directives for browser-only functionality.
 */

import { useEffect, useState } from 'react';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import {
  LineChart,
  BarChart,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  Line,
  Bar,
  preloadHeavyComponents,
} from '@/lib/dynamic-imports';

interface ChartData {
  name: string;
  value: number;
  timestamp: string;
}

interface DynamicChartsProps {
  data: ChartData[];
  title?: string;
  type?: 'line' | 'bar';
  className?: string;
}

export function DynamicCharts({
  data,
  title = 'Server Metrics',
  type = 'line',
  className,
}: DynamicChartsProps) {
  const [isClient, setIsClient] = useState(false);
  const [chartsLoaded, setChartsLoaded] = useState(false);

  // Ensure we're on the client side
  useEffect(() => {
    setIsClient(true);

    // Preload charts for better UX
    preloadHeavyComponents.charts().then(() => {
      setChartsLoaded(true);
    });
  }, []);

  // Show loading state until client-side and charts are loaded
  if (!isClient || !chartsLoaded) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>{title}</CardTitle>
          <CardDescription>Loading chart data...</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <Skeleton className="h-4 w-1/4" />
            <Skeleton className="h-64 w-full" />
            <div className="flex space-x-2">
              <Skeleton className="h-4 w-16" />
              <Skeleton className="h-4 w-16" />
              <Skeleton className="h-4 w-16" />
            </div>
          </div>
        </CardContent>
      </Card>
    );
  }

  const chartProps = {
    width: 600,
    height: 300,
    data,
    margin: {
      top: 5,
      right: 30,
      left: 20,
      bottom: 5,
    },
  };

  return (
    <Card className={className}>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>Real-time server performance metrics</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="w-full overflow-x-auto">
          {type === 'line' ? (
            <LineChart {...chartProps}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip />
              <Legend />
              <Line
                type="monotone"
                dataKey="value"
                stroke="#8884d8"
                strokeWidth={2}
                dot={{ fill: '#8884d8' }}
                activeDot={{ r: 8 }}
              />
            </LineChart>
          ) : (
            <BarChart {...chartProps}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip />
              <Legend />
              <Bar dataKey="value" fill="#8884d8" />
            </BarChart>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

/**
 * Advanced Chart Component with Multiple Chart Types
 */
interface MetricsData {
  cpu: number;
  memory: number;
  requests: number;
  timestamp: string;
}

interface AdvancedMetricsChartProps {
  data: MetricsData[];
  className?: string;
}

export function AdvancedMetricsChart({
  data,
  className,
}: AdvancedMetricsChartProps) {
  const [isClient, setIsClient] = useState(false);

  useEffect(() => {
    setIsClient(true);
    // Preload charts early for better perceived performance
    preloadHeavyComponents.charts();
  }, []);

  if (!isClient) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>System Metrics</CardTitle>
        </CardHeader>
        <CardContent>
          <Skeleton className="h-80 w-full" />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className={className}>
      <CardHeader>
        <CardTitle>System Metrics Dashboard</CardTitle>
        <CardDescription>
          CPU, Memory, and Request metrics over time
        </CardDescription>
      </CardHeader>
      <CardContent>
        <LineChart
          width={800}
          height={400}
          data={data}
          margin={{
            top: 5,
            right: 30,
            left: 20,
            bottom: 5,
          }}
        >
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="timestamp" />
          <YAxis />
          <Tooltip />
          <Legend />
          <Line type="monotone" dataKey="cpu" stroke="#8884d8" name="CPU %" />
          <Line
            type="monotone"
            dataKey="memory"
            stroke="#82ca9d"
            name="Memory %"
          />
          <Line
            type="monotone"
            dataKey="requests"
            stroke="#ffc658"
            name="Requests/min"
          />
        </LineChart>
      </CardContent>
    </Card>
  );
}

/**
 * Chart wrapper with error boundary
 */
export function SafeChart({
  children,
  fallback,
}: {
  children: React.ReactNode;
  fallback?: React.ReactNode;
}) {
  const [hasError, setHasError] = useState(false);

  useEffect(() => {
    setHasError(false);
  }, [children]);

  if (hasError) {
    return (
      <Card>
        <CardContent className="p-6">
          <div className="text-center text-muted-foreground">
            {fallback || 'Chart failed to load'}
          </div>
        </CardContent>
      </Card>
    );
  }

  try {
    return <>{children}</>;
  } catch (_error) {
    setHasError(true);
    return null;
  }
}
