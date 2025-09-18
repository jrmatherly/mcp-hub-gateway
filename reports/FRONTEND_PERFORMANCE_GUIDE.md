# Next.js 15 + React 19 Performance Guide

This guide covers performance optimization patterns and best practices for the MCP Portal frontend application.

## 🚀 Code Splitting Implementation

### 1. Heavy Package Optimization

Our application uses several heavy packages that benefit from code splitting:

- **Recharts** (~180KB): Chart visualization library
- **React Grid Layout** (~95KB): Dashboard grid system
- **Canvas Confetti** (~15KB): Celebration animations
- **Motion/Framer Motion** (~85KB): Animation library
- **Azure MSAL** (~65KB): Authentication library

### 2. Dynamic Import Patterns

```typescript
// ✅ Good: Dynamic import with loading state
const ChartComponent = dynamic(
  () => import("@/components/charts/DynamicCharts"),
  {
    loading: () => <Skeleton className="h-64 w-full" />,
    ssr: false,
  }
);

// ❌ Bad: Static import of heavy component
import { LineChart } from "recharts";
```

### 3. Client-Side Component Structure

```typescript
"use client";

import { useEffect, useState } from "react";
import { preloadHeavyComponents } from "@/lib/dynamic-imports";

export function ClientComponent() {
  const [isClient, setIsClient] = useState(false);
  const [componentLoaded, setComponentLoaded] = useState(false);

  useEffect(() => {
    setIsClient(true);
    // Preload heavy components
    preloadHeavyComponents.charts().then(() => {
      setComponentLoaded(true);
    });
  }, []);

  // Always check client-side rendering
  if (!isClient || !componentLoaded) {
    return <LoadingSkeleton />;
  }

  return <HeavyComponent />;
}
```

## 🎯 Bundle Optimization

### 1. Webpack Configuration

Our `next.config.js` includes optimized chunk splitting:

```javascript
splitChunks: {
  cacheGroups: {
    // Framework chunk (React, Next.js)
    framework: {
      test: /[\\/]node_modules[\\/](react|react-dom|next)[\\/]/,
      name: 'framework',
      chunks: 'all',
      priority: 40,
    },
    // Charts chunk (async loading)
    charts: {
      test: /[\\/]node_modules[\\/](recharts|d3)[\\/]/,
      name: 'charts',
      chunks: 'async',
      priority: 35,
    },
    // UI components chunk
    ui: {
      test: /[\\/]node_modules[\\/](@radix-ui|@headlessui)[\\/]/,
      name: 'ui-components',
      chunks: 'all',
      priority: 30,
    }
  }
}
```

### 2. Performance Budget

| Metric            | Target  | Current                      |
| ----------------- | ------- | ---------------------------- |
| First Load JS     | < 250KB | Check with `npm run analyze` |
| Individual Chunks | < 100KB | Monitored automatically      |
| Total Bundle      | < 1MB   | Tracked in CI                |

### 3. Bundle Analysis Commands

```bash
# Full bundle analysis
npm run analyze

# Performance analysis
npm run perf

# Bundle size check
npm run bundle-size
```

## ⚡ React 19 Optimizations

### 1. Concurrent Features

```typescript
// Enable React 19 concurrent features in next.config.js
experimental: {
  esmExternals: true,
  optimisticClientCache: true,
  optimizeCss: true,
}
```

### 2. Server Components

```typescript
// ✅ Good: Server Component for static content
export default async function ServerDashboard() {
  const data = await fetchServerData();
  return <StaticDashboard data={data} />;
}

// ✅ Good: Client Component for interactivity
("use client");
export function InteractiveDashboard({ data }) {
  const [filter, setFilter] = useState("");
  return <FilterableList data={data} filter={filter} />;
}
```

### 3. Suspense Boundaries

```typescript
import { Suspense } from "react";

export function DashboardPage() {
  return (
    <div>
      <Suspense fallback={<DashboardSkeleton />}>
        <DynamicCharts />
      </Suspense>
      <Suspense fallback={<GridSkeleton />}>
        <DynamicGridLayout />
      </Suspense>
    </div>
  );
}
```

## 🎨 Loading States & UX

### 1. Skeleton Components

```typescript
// Chart loading skeleton
const ChartLoading = () => (
  <div className="space-y-3">
    <Skeleton className="h-4 w-1/4" />
    <Skeleton className="h-64 w-full" />
    <div className="flex space-x-2">
      <Skeleton className="h-4 w-16" />
      <Skeleton className="h-4 w-16" />
    </div>
  </div>
);
```

### 2. Progressive Enhancement

```typescript
export function ProgressiveChart({ data }) {
  const [chartLoaded, setChartLoaded] = useState(false);

  return (
    <div>
      {/* Show static data first */}
      <SimpleDataTable data={data} />

      {/* Load interactive chart progressively */}
      <Suspense fallback={<ChartSkeleton />}>
        <DynamicChart data={data} onLoad={() => setChartLoaded(true)} />
      </Suspense>
    </div>
  );
}
```

## 🔧 Development Tools

### 1. Bundle Analyzer

```bash
# Generate bundle analysis
npm run analyze

# View results
open .next/analyze/client.html
```

### 2. Performance Monitoring

```bash
# Check bundle sizes
npm run build | grep "First Load JS"

# Lighthouse CI (add to package.json)
"lighthouse": "lhci autorun"
```

### 3. Development Scripts

```bash
# Development with Turbopack
npm run dev

# Type checking
npm run type-check

# Performance build
npm run perf
```

## 📊 Performance Metrics

### 1. Core Web Vitals Targets

- **LCP (Largest Contentful Paint)**: < 2.5s
- **FID (First Input Delay)**: < 100ms
- **CLS (Cumulative Layout Shift)**: < 0.1

### 2. Bundle Size Targets

- **Framework chunk**: ~100KB (React + Next.js)
- **UI components**: ~80KB (Radix UI)
- **Charts**: ~180KB (loaded on demand)
- **Total initial**: < 250KB

### 3. Monitoring Setup

```typescript
// pages/_app.tsx
import { reportWebVitals } from "next/web-vitals";

export function reportWebVitals(metric) {
  if (process.env.NODE_ENV === "production") {
    // Send to analytics
    analytics.track("web-vital", metric);
  }
}
```

## 🚨 Common Performance Issues

### 1. Heavy Components in Server Components

```typescript
// ❌ Bad: Heavy client component in server component
export default function ServerPage() {
  return (
    <div>
      <HeavyChartComponent /> {/* This breaks SSR */}
    </div>
  );
}

// ✅ Good: Proper client boundary
export default function ServerPage() {
  return (
    <div>
      <Suspense fallback={<ChartSkeleton />}>
        <ClientChartWrapper />
      </Suspense>
    </div>
  );
}
```

### 2. Missing Client-Side Checks

```typescript
// ❌ Bad: No client-side check
export function ClientComponent() {
  const data = window.localStorage.getItem("data"); // Breaks SSR
  return <div>{data}</div>;
}

// ✅ Good: Proper client-side check
export function ClientComponent() {
  const [data, setData] = useState("");

  useEffect(() => {
    const stored = window.localStorage.getItem("data");
    setData(stored || "");
  }, []);

  return <div>{data}</div>;
}
```

### 3. Large Bundle Imports

```typescript
// ❌ Bad: Import entire library
import * as Charts from "recharts";

// ✅ Good: Import specific components
import { LineChart, XAxis, YAxis } from "recharts";

// ✅ Better: Use dynamic imports
import { LineChart } from "@/lib/dynamic-imports";
```

## 🔄 Optimization Checklist

### Build Time

- [ ] Bundle analysis shows chunks < 100KB each
- [ ] No dynamic imports in Server Components
- [ ] All heavy packages are code-split
- [ ] TypeScript compilation is clean
- [ ] ESLint passes without warnings

### Runtime

- [ ] Client-side checks for browser APIs
- [ ] Proper loading states for all async components
- [ ] Suspense boundaries around dynamic imports
- [ ] Error boundaries for heavy components
- [ ] Progressive enhancement patterns

### User Experience

- [ ] Fast initial page load (< 2s LCP)
- [ ] Responsive interactions (< 100ms FID)
- [ ] Minimal layout shift (< 0.1 CLS)
- [ ] Meaningful loading skeletons
- [ ] Graceful error handling

## 📚 Additional Resources

- [Next.js Performance Documentation](https://nextjs.org/docs/advanced-features/performance)
- [React 19 Concurrent Features](https://react.dev/blog/2024/04/25/react-19)
- [Core Web Vitals](https://web.dev/vitals/)
- [Bundle Analyzer Documentation](https://github.com/vercel/next.js/tree/canary/packages/next-bundle-analyzer)
