/**
 * Dynamic Import Utilities for Heavy Components
 *
 * This file provides optimized dynamic imports for heavy packages
 * with proper TypeScript support and loading states.
 */

import dynamic from 'next/dynamic';
import { ComponentType, Suspense } from 'react';
import { Skeleton } from '@/components/ui/skeleton';

// Type definitions for dynamic imports
type DynamicComponent<T = Record<string, unknown>> = ComponentType<T>;
type LoadingComponent = ComponentType;
type ImportFunction<T> = () => Promise<{ default: T }>;

// Loading component for chart components
const ChartLoading = () => (
  <div className="space-y-3">
    <Skeleton className="h-4 w-1/4" />
    <Skeleton className="h-64 w-full" />
    <div className="flex space-x-2">
      <Skeleton className="h-4 w-16" />
      <Skeleton className="h-4 w-16" />
      <Skeleton className="h-4 w-16" />
    </div>
  </div>
);

// Loading component for grid layout
const GridLoading = () => (
  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
    {Array.from({ length: 6 }).map((_, i) => (
      <Skeleton key={i} className="h-32 w-full" />
    ))}
  </div>
);

// Loading component for confetti/animations
const AnimationLoading = () => (
  <div className="flex items-center justify-center h-32">
    <div className="animate-pulse text-muted-foreground">
      Loading animation...
    </div>
  </div>
);

/**
 * Recharts Dynamic Imports
 * Split charts into separate chunks for better performance
 */
export const LineChart = dynamic(
  () => import('recharts').then(mod => ({ default: mod.LineChart })),
  {
    loading: ChartLoading,
    ssr: false,
  }
);

export const BarChart = dynamic(
  () => import('recharts').then(mod => ({ default: mod.BarChart })),
  {
    loading: ChartLoading,
    ssr: false,
  }
);

export const PieChart = dynamic(
  () => import('recharts').then(mod => ({ default: mod.PieChart })),
  {
    loading: ChartLoading,
    ssr: false,
  }
);

export const AreaChart = dynamic(
  () => import('recharts').then(mod => ({ default: mod.AreaChart })),
  {
    loading: ChartLoading,
    ssr: false,
  }
);

// Recharts components
export const XAxis = dynamic(
  () => import('recharts').then(mod => ({ default: mod.XAxis })),
  { ssr: false }
);

export const YAxis = dynamic(
  () => import('recharts').then(mod => ({ default: mod.YAxis })),
  { ssr: false }
);

export const CartesianGrid = dynamic(
  () => import('recharts').then(mod => ({ default: mod.CartesianGrid })),
  { ssr: false }
);

export const Tooltip = dynamic(
  () => import('recharts').then(mod => ({ default: mod.Tooltip })),
  { ssr: false }
);

export const Legend = dynamic(
  () =>
    import('recharts').then(mod => ({ default: mod.Legend })) as Promise<{
      default: ComponentType;
    }>,
  { ssr: false }
);

export const Line = dynamic(
  () => import('recharts').then(mod => ({ default: mod.Line })),
  { ssr: false }
);

export const Bar = dynamic(
  () => import('recharts').then(mod => ({ default: mod.Bar })),
  { ssr: false }
);

export const Area = dynamic(
  () => import('recharts').then(mod => ({ default: mod.Area })),
  { ssr: false }
);

export const Pie = dynamic(
  () => import('recharts').then(mod => ({ default: mod.Pie })),
  { ssr: false }
);

export const Cell = dynamic(
  () => import('recharts').then(mod => ({ default: mod.Cell })),
  { ssr: false }
);

/**
 * React Grid Layout Dynamic Import
 */
export const GridLayout = dynamic(
  () =>
    import('react-grid-layout').then(mod => ({
      default: mod.WidthProvider(mod.Responsive),
    })),
  {
    loading: GridLoading,
    ssr: false,
  }
);

export const ResponsiveGridLayout = dynamic(
  () =>
    import('react-grid-layout').then(mod => ({
      default: mod.WidthProvider(mod.Responsive),
    })),
  {
    loading: GridLoading,
    ssr: false,
  }
);

/**
 * Canvas Confetti Dynamic Import
 */
export const ConfettiComponent = dynamic(() => import('canvas-confetti'), {
  ssr: false,
  loading: AnimationLoading,
});

export const useConfetti = () => {
  return ConfettiComponent;
};

/**
 * Motion/Framer Motion Dynamic Imports
 */
export const Motion = dynamic(
  () =>
    import('motion/react').then(mod => ({ default: mod.motion })) as Promise<{
      default: ComponentType;
    }>,
  {
    ssr: false,
    loading: AnimationLoading,
  }
);

export const AnimatePresence = dynamic(
  () =>
    import('motion/react').then(mod => ({
      default: mod.AnimatePresence,
    })) as Promise<{ default: ComponentType }>,
  {
    ssr: false,
    loading: AnimationLoading,
  }
);

/**
 * Higher-order component for adding Suspense boundaries
 */
export function withSuspense<P extends Record<string, unknown>>(
  Component: ComponentType<P>,
  FallbackComponent?: ComponentType
) {
  const SuspendedComponent = (props: P) => (
    <Suspense
      fallback={
        FallbackComponent ? (
          <FallbackComponent />
        ) : (
          <Skeleton className="h-32 w-full" />
        )
      }
    >
      <Component {...props} />
    </Suspense>
  );

  SuspendedComponent.displayName = `withSuspense(${
    Component.displayName || Component.name
  })`;

  return SuspendedComponent;
}

/**
 * Utility for creating dynamic imports with consistent loading states
 */
export function createDynamicImport<T extends DynamicComponent>(
  importFn: ImportFunction<T>,
  options?: {
    loading?: LoadingComponent;
    ssr?: boolean;
    fallback?: LoadingComponent;
  }
) {
  const { loading, ssr = false } = options || {};
  const LoadingFallback =
    loading || (() => <Skeleton className="h-32 w-full" />);

  return dynamic(importFn, {
    loading: () => <LoadingFallback />,
    ssr,
  });
}

/**
 * Preload function for critical heavy components
 * Call this in useEffect or during user interactions
 */
export const preloadHeavyComponents = {
  charts: () => import('recharts'),
  gridLayout: () => import('react-grid-layout'),
  confetti: () => import('canvas-confetti'),
  motion: () => import('motion/react'),
};

/**
 * Bundle size estimates for heavy components
 * Use this for monitoring and decision making
 */
export const BUNDLE_SIZES = {
  recharts: '~180KB gzipped',
  'react-grid-layout': '~95KB gzipped',
  'canvas-confetti': '~15KB gzipped',
  motion: '~85KB gzipped',
  '@azure/msal-browser': '~65KB gzipped',
} as const;
