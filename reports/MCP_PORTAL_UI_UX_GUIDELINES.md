# MCP Portal UI/UX Component Guidelines

**Version**: 1.0
**Date**: 2025-09-17
**Project**: MCP Portal Frontend
**Framework**: Next.js 15.5.3 with App Router

## Table of Contents

1. [Design Philosophy](#1-design-philosophy)
2. [Color System & Theming](#2-color-system--theming)
3. [Typography Standards](#3-typography-standards)
4. [Component Patterns](#4-component-patterns)
5. [Accessibility Requirements](#5-accessibility-requirements)
6. [Animation & Interaction Guidelines](#6-animation--interaction-guidelines)
7. [Responsive Design](#7-responsive-design)
8. [Form Design Patterns](#8-form-design-patterns)
9. [Error Handling & Feedback](#9-error-handling--feedback)
10. [Loading & Skeleton States](#10-loading--skeleton-states)
11. [Empty States & Onboarding](#11-empty-states--onboarding)
12. [Icon Usage Guidelines](#12-icon-usage-guidelines)
13. [Performance Guidelines](#13-performance-guidelines)
14. [Developer Guidelines](#14-developer-guidelines)

---

## 1. Design Philosophy

### Core Principles

**Enterprise-First Design**

- Professional, clean interface suitable for IT operations teams
- Prioritize functionality and clarity over decorative elements
- Consistency in visual hierarchy and information architecture

**Performance-First Approach**

- Optimized for fast loading and smooth interactions
- Minimal bundle size with efficient component loading
- Hardware-accelerated animations and transforms

**Accessibility by Default**

- WCAG 2.1 AA compliance built into every component
- Keyboard navigation support across all interactive elements
- Screen reader compatibility with proper semantic markup

**User-Centered Design**

- Task-oriented interface focused on MCP server management workflows
- Contextual information display with progressive disclosure
- Efficient data visualization for system monitoring

### Design Values

1. **Clarity**: Information hierarchy is immediately apparent
2. **Efficiency**: Common tasks require minimal user interaction
3. **Reliability**: Visual feedback confirms user actions and system state
4. **Flexibility**: Interface adapts to different screen sizes and contexts
5. **Consistency**: Predictable patterns across all interface elements

---

## 2. Color System & Theming

### Base Color Palette

**Primary Colors**

```css
/* Light Mode */
--primary: 221.2 83.2% 53.3%; /* Blue - Docker brand alignment */
--primary-foreground: 210 40% 98%; /* White text on primary */

/* Dark Mode */
--primary: 217.2 91.2% 59.8%; /* Lighter blue for dark mode */
--primary-foreground: 222.2 84% 4.9%; /* Dark text on primary */
```

**Semantic Colors**

```css
/* Status Colors - Light/Dark Mode Adaptive */
--success: #22c55e / #16a34a; /* Green for enabled/healthy states */
--warning: #f59e0b / #d97706; /* Amber for warnings/degraded states */
--error: #ef4444 / #dc2626; /* Red for errors/failed states */
--docker: #0ea5e9 / #0284c7; /* Docker blue for running states */
```

**Neutral Colors**

```css
/* Background & Surface */
--background: 0 0% 100% / 222.2 84% 4.9%; /* Main background */
--card: 0 0% 100% / 222.2 84% 4.9%; /* Card backgrounds */
--muted: 210 40% 96% / 217.2 32.6% 17.5%; /* Subtle backgrounds */

/* Text Colors */
--foreground: 222.2 84% 4.9% / 210 40% 98%; /* Primary text */
--muted-foreground: 215.4 16.3% 46.9% / 215 20.2% 65.1%; /* Secondary text */

/* Borders & Inputs */
--border: 214.3 31.8% 91.4% / 217.2 32.6% 17.5%; /* Border color */
--input: 214.3 31.8% 91.4% / 217.2 32.6% 17.5%; /* Input border */
--ring: 221.2 83.2% 53.3% / 224.3 76.3% 94.1%; /* Focus ring */
```

### Status Indicators

**Server Status Colors**

```tsx
const statusColors = {
  enabled:
    "bg-success-50 text-success-700 dark:bg-success-900/20 dark:text-success-300",
  disabled: "bg-gray-50 text-gray-700 dark:bg-gray-900/20 dark:text-gray-300",
  running:
    "bg-docker-50 text-docker-700 dark:bg-docker-900/20 dark:text-docker-300",
  stopped: "bg-gray-50 text-gray-700 dark:bg-gray-900/20 dark:text-gray-300",
  error: "bg-error-50 text-error-700 dark:bg-error-900/20 dark:text-error-300",
  unknown:
    "bg-warning-50 text-warning-700 dark:bg-warning-900/20 dark:text-warning-300",
};
```

**Health Status Colors**

```tsx
const healthColors = {
  healthy:
    "bg-success-50 text-success-700 dark:bg-success-900/20 dark:text-success-300",
  unhealthy:
    "bg-error-50 text-error-700 dark:bg-error-900/20 dark:text-error-300",
  degraded:
    "bg-warning-50 text-warning-700 dark:bg-warning-900/20 dark:text-warning-300",
  unknown: "bg-gray-50 text-gray-700 dark:bg-gray-900/20 dark:text-gray-300",
};
```

### Theme Implementation

**Dark Mode Strategy**

- System preference detection with manual override
- Smooth transitions between theme changes
- Consistent contrast ratios across themes

```tsx
// Theme Usage Example
import { useTheme } from "next-themes";

function ThemeToggle() {
  const { theme, setTheme } = useTheme();

  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
      className="theme-transition"
    >
      {theme === "dark" ? <SunIcon /> : <MoonIcon />}
    </Button>
  );
}
```

---

## 3. Typography Standards

### Font Stack

**Primary Font: Inter Variable**

- Modern, highly legible sans-serif optimized for interfaces
- Variable font technology for optimal performance
- Excellent support for technical content and data displays

```css
font-family: ['Inter Variable', 'Inter'],
font-feature-settings: '"cv02", "cv03", "cv04", "cv11"',
font-variation-settings: '"opsz" 32'
```

**Monospace Font: JetBrains Mono Variable**

- Used for code blocks, server names, and technical identifiers
- Enhanced readability for alphanumeric content

```css
font-family: ['JetBrains Mono Variable', 'JetBrains Mono'],
font-feature-settings: '"calt", "liga"'
```

### Typography Scale

**Heading Hierarchy**

```css
/* Display - Hero sections, page titles */
.text-display: 3rem (48px) / line-height: 1.2
.text-6xl: 3.75rem (60px) / line-height: 1

/* Headings - Section headers */
.text-4xl: 2.25rem (36px) / line-height: 2.5rem
.text-3xl: 1.875rem (30px) / line-height: 2.25rem
.text-2xl: 1.5rem (24px) / line-height: 2rem
.text-xl: 1.25rem (20px) / line-height: 1.75rem
.text-lg: 1.125rem (18px) / line-height: 1.75rem

/* Body Text */
.text-base: 1rem (16px) / line-height: 1.5rem     /* Primary body text */
.text-sm: 0.875rem (14px) / line-height: 1.25rem  /* Secondary text, labels */
.text-xs: 0.75rem (12px) / line-height: 1rem      /* Captions, metadata */
```

**Responsive Typography**

```css
/* Responsive text that scales smoothly */
.responsive-text {
  font-size: clamp(0.875rem, 0.8rem + 0.4vw, 1.125rem);
  line-height: 1.6;
}

.responsive-heading {
  font-size: clamp(1.5rem, 1.2rem + 1.5vw, 3rem);
  line-height: 1.2;
}
```

### Text Usage Guidelines

**Hierarchy Implementation**

```tsx
// Page Title
<h1 className="text-3xl font-bold text-foreground">MCP Servers</h1>

// Section Header
<h2 className="text-xl font-semibold text-foreground">Configuration</h2>

// Card Title
<h3 className="text-lg font-medium text-foreground">Server Status</h3>

// Body Text
<p className="text-base text-foreground">Server management interface</p>

// Secondary Text
<p className="text-sm text-muted-foreground">Last updated 2 minutes ago</p>

// Metadata/Captions
<span className="text-xs text-muted-foreground">ID: server-001</span>
```

**Text Wrapping**

```css
/* Use modern CSS text wrapping */
.text-balance {
  text-wrap: balance;
} /* For headings */
.text-pretty {
  text-wrap: pretty;
} /* For body text */
.text-no-wrap {
  text-wrap: nowrap;
} /* For labels/IDs */
```

---

## 4. Component Patterns

### Core Component Library

**Base Components (Shadcn/ui + Radix UI)**

- Button, Input, Select, Checkbox, Radio
- Dialog, Popover, Tooltip, Dropdown Menu
- Card, Accordion, Tabs, Alert
- Progress, Badge, Avatar, Separator

**Custom MCP Components**

- ServerCard, ServerList, ServerStatus
- ConfigurationForm, BulkOperations
- LogViewer, MetricsChart, HealthIndicator

### Component Architecture

**Composition Pattern**

```tsx
// Compound component pattern for complex UI
<ServerCard>
  <ServerCard.Header>
    <ServerCard.Title>Server Name</ServerCard.Title>
    <ServerCard.Actions>
      <Button>Enable</Button>
    </ServerCard.Actions>
  </ServerCard.Header>
  <ServerCard.Content>
    <ServerCard.Status status="running" />
    <ServerCard.Metrics data={metrics} />
  </ServerCard.Content>
</ServerCard>
```

**Variant System**

```tsx
// Using class-variance-authority for consistent variants
const buttonVariants = cva(
  "inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground hover:bg-primary/90",
        destructive:
          "bg-destructive text-destructive-foreground hover:bg-destructive/90",
        outline: "border border-input bg-background hover:bg-accent",
        secondary:
          "bg-secondary text-secondary-foreground hover:bg-secondary/80",
        ghost: "hover:bg-accent hover:text-accent-foreground",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: {
        default: "h-10 px-4 py-2",
        sm: "h-9 rounded-md px-3",
        lg: "h-11 rounded-md px-8",
        icon: "h-10 w-10",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);
```

### Layout Components

**Container System**

```tsx
// Responsive container with proper padding
<div className="container-responsive">
  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
    {/* Content */}
  </div>
</div>
```

**Grid System**

```css
/* Auto-fit grid for responsive cards */
.grid-auto-fit {
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
}

/* Auto-fill for consistent sizing */
.grid-auto-fill {
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
}

/* Responsive grid with minimum card size */
.grid-responsive {
  grid-template-columns: repeat(auto-fit, minmax(min(250px, 100%), 1fr));
}
```

### State Management Patterns

**Component State**

```tsx
// Using React Query for server state
const {
  data: servers,
  isLoading,
  error,
} = useQuery({
  queryKey: ["servers"],
  queryFn: fetchServers,
  refetchInterval: 30000, // Refresh every 30 seconds
});

// Using Zustand for client state
const useAppStore = create<AppState>((set) => ({
  sidebarOpen: false,
  toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
  selectedServer: null,
  setSelectedServer: (server) => set({ selectedServer: server }),
}));
```

---

## 5. Accessibility Requirements

### WCAG 2.1 AA Compliance

**Keyboard Navigation**

- All interactive elements accessible via keyboard
- Logical tab order throughout interface
- Escape key closes modals and dropdowns
- Arrow keys navigate within component groups

**Focus Management**

```tsx
// Custom focus trap for modals
import { useFocusTrap } from "@/hooks/useFocusTrap";

function Modal({ children, open, onClose }) {
  const trapRef = useFocusTrap(open);

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <Dialog.Content ref={trapRef} className="focus-visible">
        {children}
      </Dialog.Content>
    </Dialog>
  );
}
```

**Screen Reader Support**

```tsx
// Proper ARIA labels and descriptions
<Button
  aria-label="Enable server"
  aria-describedby="enable-help"
  onClick={enableServer}
>
  Enable
</Button>
<div id="enable-help" className="sr-only">
  This will start the MCP server and make it available for connections
</div>

// Status announcements
<div aria-live="polite" aria-atomic="true" className="sr-only">
  {statusMessage}
</div>
```

**Color & Contrast**

- Minimum 4.5:1 contrast ratio for normal text
- Minimum 3:1 contrast ratio for large text and UI components
- Color never used as the sole means of conveying information

**Motion & Animation**

```tsx
// Respect user's motion preferences
import { useReducedMotion } from "framer-motion";

function AnimatedComponent() {
  const shouldReduceMotion = useReducedMotion();

  return (
    <motion.div
      animate={{ opacity: 1, y: 0 }}
      initial={{ opacity: 0, y: shouldReduceMotion ? 0 : 20 }}
      transition={{ duration: shouldReduceMotion ? 0 : 0.2 }}
    >
      Content
    </motion.div>
  );
}
```

### Accessibility Testing

**Required Testing**

- Keyboard-only navigation testing
- Screen reader testing (NVDA, JAWS, VoiceOver)
- Color contrast validation
- Focus indicator visibility
- Alternative text for images and icons

**Automated Testing**

```tsx
// Using @testing-library for accessibility testing
import { render, screen } from "@testing-library/react";
import { axe, toHaveNoViolations } from "jest-axe";

expect.extend(toHaveNoViolations);

test("ServerCard should be accessible", async () => {
  const { container } = render(<ServerCard server={mockServer} />);
  const results = await axe(container);
  expect(results).toHaveNoViolations();
});
```

---

## 6. Animation & Interaction Guidelines

### Animation Principles

**Performance-First Animations**

- Hardware-accelerated transforms using `transform3d()`
- Prefer `transform` and `opacity` for smooth 60fps animations
- Use `will-change` property judiciously for better performance

**Animation Timing**

```css
/* Standard timing functions */
--transition-fast: 150ms cubic-bezier(0.4, 0, 0.2, 1);
--transition-normal: 250ms cubic-bezier(0.4, 0, 0.2, 1);
--transition-slow: 350ms cubic-bezier(0.4, 0, 0.2, 1);

/* Specialized easing */
--bounce-in: cubic-bezier(0.68, -0.55, 0.265, 1.55);
--smooth: cubic-bezier(0.4, 0, 0.2, 1);
--snappy: cubic-bezier(0.4, 0, 0.6, 1);
```

### Interaction States

**Button States**

```tsx
// Interactive button with proper state feedback
<Button
  className="
  transition-colors-fast
  hover:bg-primary/90
  focus-visible:ring-2
  focus-visible:ring-ring
  active:scale-[0.98]
  disabled:opacity-50
  disabled:pointer-events-none
"
>
  Enable Server
</Button>
```

**Card Interactions**

```tsx
// Interactive card with hover and focus states
<Card
  className="
  transition-all duration-200
  hover:shadow-md
  hover:scale-[1.02]
  focus-within:ring-2
  focus-within:ring-ring
  cursor-pointer
  transform-gpu
"
>
  <CardContent>Server Information</CardContent>
</Card>
```

### Animation Patterns

**Page Transitions**

```tsx
// Smooth page transitions with Framer Motion
<motion.div
  initial={{ opacity: 0, y: 20 }}
  animate={{ opacity: 1, y: 0 }}
  exit={{ opacity: 0, y: -20 }}
  transition={{ duration: 0.2, ease: "easeOut" }}
  className="transform-gpu"
>
  <PageContent />
</motion.div>
```

**List Animations**

```tsx
// Staggered list item animations
<motion.div
  variants={{
    hidden: { opacity: 0 },
    show: {
      opacity: 1,
      transition: {
        staggerChildren: 0.1,
      },
    },
  }}
  initial="hidden"
  animate="show"
>
  {items.map((item, index) => (
    <motion.div
      key={item.id}
      variants={{
        hidden: { opacity: 0, y: 20 },
        show: { opacity: 1, y: 0 },
      }}
      className="transform-gpu"
    >
      <ServerCard server={item} />
    </motion.div>
  ))}
</motion.div>
```

**Loading Animations**

```css
/* Optimized loading spinner */
.loading-spinner {
  @apply animate-spin h-4 w-4 border-2 border-primary border-r-transparent rounded-full;
  will-change: transform;
}

/* Skeleton loading with subtle animation */
.loading-skeleton {
  @apply animate-pulse bg-muted rounded;
  background: linear-gradient(
    90deg,
    hsl(var(--muted)) 25%,
    hsl(var(--muted-foreground) / 0.1) 50%,
    hsl(var(--muted)) 75%
  );
  background-size: 200% 100%;
  animation: shimmer 2s ease-in-out infinite;
}

@keyframes shimmer {
  0% {
    background-position: -200% 0;
  }
  100% {
    background-position: 200% 0;
  }
}
```

---

## 7. Responsive Design

### Breakpoint Strategy

**Tailwind Breakpoints**

```css
/* Mobile-first responsive design */
xs: 475px   /* Small phones */
sm: 640px   /* Large phones */
md: 768px   /* Tablets */
lg: 1024px  /* Small laptops */
xl: 1280px  /* Laptops/desktops */
2xl: 1400px /* Large desktops */
3xl: 1600px /* Extra large displays */
```

**Container Queries**

```css
/* Component-level responsive design */
.container-query {
  container-type: inline-size;
}

@container (min-width: 20rem) {
  .server-card {
    grid-template-columns: auto 1fr auto;
  }
}

@container (min-width: 32rem) {
  .server-card {
    padding: 1.5rem;
  }
}
```

### Responsive Patterns

**Navigation**

```tsx
// Responsive navigation with mobile drawer
function Navigation() {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      {/* Desktop Navigation */}
      <nav className="hidden lg:flex items-center space-x-6">
        <NavLinks />
      </nav>

      {/* Mobile Navigation Button */}
      <Button
        variant="ghost"
        size="icon"
        className="lg:hidden"
        onClick={() => setIsOpen(true)}
      >
        <MenuIcon />
      </Button>

      {/* Mobile Navigation Drawer */}
      <Sheet open={isOpen} onOpenChange={setIsOpen}>
        <SheetContent side="left">
          <NavLinks />
        </SheetContent>
      </Sheet>
    </>
  );
}
```

**Grid Layouts**

```tsx
// Responsive server grid
<div
  className="
  grid
  grid-cols-1
  sm:grid-cols-2
  lg:grid-cols-3
  xl:grid-cols-4
  gap-4
  sm:gap-6
"
>
  {servers.map((server) => (
    <ServerCard key={server.id} server={server} />
  ))}
</div>
```

**Typography Scaling**

```css
/* Fluid typography for better readability */
.page-title {
  font-size: clamp(1.5rem, 2vw + 1rem, 3rem);
  line-height: 1.2;
}

.section-title {
  font-size: clamp(1.125rem, 1.5vw + 0.5rem, 1.5rem);
  line-height: 1.4;
}
```

### Touch & Mobile Optimization

**Touch Targets**

- Minimum 44px × 44px touch target size
- Adequate spacing between interactive elements
- Larger buttons on mobile interfaces

```css
/* Touch-optimized button sizing */
@media (max-width: 768px) {
  .btn {
    @apply h-12 px-6 text-base;
  }

  .btn-icon {
    @apply h-12 w-12;
  }
}
```

**Mobile-Specific Interactions**

```tsx
// Mobile-optimized form inputs
<Input
  className="
    h-12
    text-base
    md:h-10
    md:text-sm
  "
  placeholder="Search servers..."
/>

// Touch-friendly dropdowns
<Select>
  <SelectTrigger className="h-12 md:h-10">
    <SelectValue />
  </SelectTrigger>
  <SelectContent>
    {/* Options */}
  </SelectContent>
</Select>
```

---

## 8. Form Design Patterns

### Form Architecture

**Form Layout Structure**

```tsx
// Consistent form layout pattern
<form className="space-y-6">
  <div className="space-y-4">
    <FormSection title="Basic Configuration">
      <FormField name="serverName" label="Server Name" required />
      <FormField name="description" label="Description" />
    </FormSection>

    <FormSection title="Advanced Settings">
      <FormField name="timeout" label="Timeout (seconds)" type="number" />
      <FormField name="retryAttempts" label="Retry Attempts" type="number" />
    </FormSection>
  </div>

  <FormActions>
    <Button type="button" variant="outline">
      Cancel
    </Button>
    <Button type="submit">Save Configuration</Button>
  </FormActions>
</form>
```

**Field Components**

```tsx
// Consistent form field with proper labeling
function FormField({ name, label, required, error, hint, ...props }) {
  const id = `field-${name}`;
  const hintId = hint ? `${id}-hint` : undefined;
  const errorId = error ? `${id}-error` : undefined;

  return (
    <div className="space-y-2">
      <Label htmlFor={id} className="text-sm font-medium">
        {label}
        {required && <span className="text-destructive ml-1">*</span>}
      </Label>

      <Input
        id={id}
        name={name}
        aria-describedby={clsx(hintId, errorId)}
        aria-invalid={!!error}
        className={clsx(
          "transition-colors",
          error && "border-destructive focus-visible:ring-destructive"
        )}
        {...props}
      />

      {hint && (
        <p id={hintId} className="text-xs text-muted-foreground">
          {hint}
        </p>
      )}

      {error && (
        <p id={errorId} className="text-xs text-destructive">
          {error}
        </p>
      )}
    </div>
  );
}
```

### Validation Patterns

**Real-time Validation**

```tsx
// Using React Hook Form with Zod validation
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";

const serverConfigSchema = z.object({
  name: z.string().min(1, "Server name is required"),
  port: z.number().min(1000).max(65535, "Port must be between 1000-65535"),
  timeout: z.number().min(5).max(300, "Timeout must be between 5-300 seconds"),
});

function ServerConfigForm() {
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    watch,
  } = useForm({
    resolver: zodResolver(serverConfigSchema),
    mode: "onBlur", // Validate on blur for better UX
  });

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <FormField
        {...register("name")}
        label="Server Name"
        error={errors.name?.message}
        required
      />

      <FormField
        {...register("port", { valueAsNumber: true })}
        label="Port"
        type="number"
        error={errors.port?.message}
        hint="Port number for the MCP server (1000-65535)"
      />

      <Button type="submit" disabled={isSubmitting}>
        {isSubmitting ? "Saving..." : "Save Configuration"}
      </Button>
    </form>
  );
}
```

**Bulk Operations Form**

```tsx
// Multi-select with bulk actions
function BulkOperationsForm({ servers }) {
  const [selectedServers, setSelectedServers] = useState<string[]>([]);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-2">
          <Checkbox
            checked={selectedServers.length === servers.length}
            onCheckedChange={(checked) => {
              setSelectedServers(checked ? servers.map((s) => s.id) : []);
            }}
          />
          <Label>Select All ({servers.length} servers)</Label>
        </div>

        {selectedServers.length > 0 && (
          <div className="flex items-center space-x-2">
            <Badge variant="secondary">{selectedServers.length} selected</Badge>
            <Button size="sm" onClick={() => bulkEnable(selectedServers)}>
              Enable Selected
            </Button>
            <Button
              size="sm"
              variant="destructive"
              onClick={() => bulkDisable(selectedServers)}
            >
              Disable Selected
            </Button>
          </div>
        )}
      </div>

      <div className="space-y-2">
        {servers.map((server) => (
          <ServerCheckbox
            key={server.id}
            server={server}
            checked={selectedServers.includes(server.id)}
            onCheckedChange={(checked) => {
              setSelectedServers((prev) =>
                checked
                  ? [...prev, server.id]
                  : prev.filter((id) => id !== server.id)
              );
            }}
          />
        ))}
      </div>
    </div>
  );
}
```

---

## 9. Error Handling & Feedback

### Error State Patterns

**Form Validation Errors**

```tsx
// Field-level error display
<FormField
  name="serverName"
  label="Server Name"
  error="Server name must be at least 3 characters"
  aria-invalid="true"
  className="border-destructive focus-visible:ring-destructive"
/>

// Form-level error summary
<Alert variant="destructive" className="mb-6">
  <AlertCircle className="h-4 w-4" />
  <AlertTitle>Configuration Error</AlertTitle>
  <AlertDescription>
    Please fix the following errors:
    <ul className="list-disc list-inside mt-2 space-y-1">
      <li>Server name is required</li>
      <li>Port must be between 1000-65535</li>
    </ul>
  </AlertDescription>
</Alert>
```

**API Error Handling**

```tsx
// Global error boundary for unhandled errors
import { ErrorBoundary } from "react-error-boundary";

function ErrorFallback({ error, resetErrorBoundary }) {
  return (
    <div className="min-h-[400px] flex items-center justify-center">
      <Card className="max-w-md">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-destructive" />
            Something went wrong
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground mb-4">
            {error.message || "An unexpected error occurred"}
          </p>
          <Button onClick={resetErrorBoundary} variant="outline">
            Try again
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

// Query error handling with React Query
function ServerList() {
  const {
    data: servers,
    error,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ["servers"],
    queryFn: fetchServers,
    retry: 2,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
  });

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertCircle className="h-4 w-4" />
        <AlertTitle>Failed to load servers</AlertTitle>
        <AlertDescription className="flex items-center justify-between">
          <span>{error.message}</span>
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            Retry
          </Button>
        </AlertDescription>
      </Alert>
    );
  }

  return <ServerGrid servers={servers} />;
}
```

### Toast Notifications

**Success/Error Feedback**

```tsx
import { toast } from "sonner";

// Success notification
function handleServerEnable(serverId: string) {
  try {
    await enableServer(serverId);
    toast.success("Server enabled successfully", {
      description: `Server ${serverId} is now running`,
      action: {
        label: "View Details",
        onClick: () => router.push(`/servers/${serverId}`),
      },
    });
  } catch (error) {
    toast.error("Failed to enable server", {
      description: error.message,
      action: {
        label: "Retry",
        onClick: () => handleServerEnable(serverId),
      },
    });
  }
}

// Progress notification for long operations
function handleBulkOperation(serverIds: string[]) {
  const toastId = toast.loading("Updating servers...", {
    description: `Processing ${serverIds.length} servers`,
  });

  try {
    await bulkUpdateServers(serverIds);
    toast.success("Bulk operation completed", {
      id: toastId,
      description: `Updated ${serverIds.length} servers successfully`,
    });
  } catch (error) {
    toast.error("Bulk operation failed", {
      id: toastId,
      description: error.message,
    });
  }
}
```

### Status Indicators

**Connection Status**

```tsx
function ConnectionStatus({
  status,
}: {
  status: "connected" | "disconnected" | "connecting";
}) {
  const statusConfig = {
    connected: {
      icon: CheckCircle,
      className: "text-success-600",
      label: "Connected",
    },
    disconnected: {
      icon: XCircle,
      className: "text-destructive",
      label: "Disconnected",
    },
    connecting: {
      icon: Loader2,
      className: "text-warning-600 animate-spin",
      label: "Connecting",
    },
  };

  const config = statusConfig[status];
  const Icon = config.icon;

  return (
    <div className="flex items-center gap-2">
      <Icon className={clsx("h-4 w-4", config.className)} />
      <span className="text-sm font-medium">{config.label}</span>
    </div>
  );
}
```

---

## 10. Loading & Skeleton States

### Loading State Patterns

**Skeleton Components**

```tsx
// Server card skeleton
function ServerCardSkeleton() {
  return (
    <Card className="p-6">
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="space-y-2">
            <div className="h-4 w-32 bg-muted rounded animate-pulse" />
            <div className="h-3 w-48 bg-muted rounded animate-pulse" />
          </div>
          <div className="h-6 w-16 bg-muted rounded-full animate-pulse" />
        </div>

        <div className="space-y-2">
          <div className="h-3 w-full bg-muted rounded animate-pulse" />
          <div className="h-3 w-3/4 bg-muted rounded animate-pulse" />
        </div>

        <div className="flex justify-end space-x-2">
          <div className="h-8 w-16 bg-muted rounded animate-pulse" />
          <div className="h-8 w-20 bg-muted rounded animate-pulse" />
        </div>
      </div>
    </Card>
  );
}

// Table skeleton
function TableSkeleton({ rows = 5, columns = 4 }) {
  return (
    <div className="space-y-3">
      {/* Header */}
      <div className="grid grid-cols-4 gap-4 p-4 border-b">
        {Array.from({ length: columns }).map((_, i) => (
          <div key={i} className="h-4 bg-muted rounded animate-pulse" />
        ))}
      </div>

      {/* Rows */}
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="grid grid-cols-4 gap-4 p-4">
          {Array.from({ length: columns }).map((_, j) => (
            <div key={j} className="h-4 bg-muted rounded animate-pulse" />
          ))}
        </div>
      ))}
    </div>
  );
}
```

**Loading States with Suspense**

```tsx
// Suspense boundaries for page sections
import { Suspense } from "react";

function DashboardPage() {
  return (
    <div className="space-y-8">
      <Suspense fallback={<MetricsSkeleton />}>
        <MetricsSection />
      </Suspense>

      <Suspense fallback={<ServerListSkeleton />}>
        <ServerList />
      </Suspense>

      <Suspense fallback={<LogsSkeleton />}>
        <RecentLogs />
      </Suspense>
    </div>
  );
}
```

**Progressive Loading**

```tsx
// Progressive enhancement with stale-while-revalidate
function ServerList() {
  const {
    data: servers,
    isLoading,
    isValidating,
    error,
  } = useQuery({
    queryKey: ["servers"],
    queryFn: fetchServers,
    staleTime: 30000, // Consider data fresh for 30 seconds
    refetchInterval: 60000, // Refetch every minute
  });

  if (isLoading && !servers) {
    return <ServerListSkeleton />;
  }

  return (
    <div className="space-y-4">
      {isValidating && (
        <div className="flex items-center justify-between p-2 bg-muted/50 rounded">
          <span className="text-sm text-muted-foreground">
            Refreshing data...
          </span>
          <Loader2 className="h-4 w-4 animate-spin" />
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {servers?.map((server) => (
          <ServerCard key={server.id} server={server} />
        ))}
      </div>
    </div>
  );
}
```

### Button Loading States

**Loading Button Variants**

```tsx
// Button with loading state
function LoadingButton({ children, loading, disabled, ...props }) {
  return (
    <Button disabled={loading || disabled} className="relative" {...props}>
      {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
      {children}
    </Button>
  );
}

// Usage example
<LoadingButton loading={isSubmitting} onClick={handleSubmit}>
  {isSubmitting ? "Saving..." : "Save Configuration"}
</LoadingButton>;
```

---

## 11. Empty States & Onboarding

### Empty State Patterns

**No Data States**

```tsx
// Empty server list
function EmptyServerList() {
  return (
    <div className="text-center py-12">
      <div className="mx-auto w-16 h-16 bg-muted rounded-full flex items-center justify-center mb-4">
        <Server className="h-8 w-8 text-muted-foreground" />
      </div>

      <h3 className="text-lg font-semibold mb-2">No servers configured</h3>
      <p className="text-muted-foreground mb-6 max-w-sm mx-auto">
        Get started by adding your first MCP server from the catalog or create a
        custom configuration.
      </p>

      <div className="flex flex-col sm:flex-row gap-3 justify-center">
        <Button onClick={() => router.push("/catalog")}>Browse Catalog</Button>
        <Button variant="outline" onClick={() => router.push("/servers/new")}>
          Add Custom Server
        </Button>
      </div>
    </div>
  );
}

// Empty search results
function EmptySearchResults({ query }: { query: string }) {
  return (
    <div className="text-center py-8">
      <div className="mx-auto w-12 h-12 bg-muted rounded-full flex items-center justify-center mb-4">
        <Search className="h-6 w-6 text-muted-foreground" />
      </div>

      <h3 className="text-base font-medium mb-2">No results found</h3>
      <p className="text-sm text-muted-foreground mb-4">
        No servers match "<span className="font-medium">{query}</span>"
      </p>

      <Button variant="outline" size="sm" onClick={() => clearSearch()}>
        Clear search
      </Button>
    </div>
  );
}
```

**Error States**

```tsx
// Network error state
function NetworkError({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="text-center py-12">
      <div className="mx-auto w-16 h-16 bg-destructive/10 rounded-full flex items-center justify-center mb-4">
        <WifiOff className="h-8 w-8 text-destructive" />
      </div>

      <h3 className="text-lg font-semibold mb-2">Connection failed</h3>
      <p className="text-muted-foreground mb-6 max-w-sm mx-auto">
        Unable to connect to the MCP Gateway. Please check your connection and
        try again.
      </p>

      <Button onClick={onRetry}>Try again</Button>
    </div>
  );
}
```

### Onboarding Flow

**First-time User Experience**

```tsx
// Onboarding wizard
function OnboardingWizard() {
  const [currentStep, setCurrentStep] = useState(0);

  const steps = [
    {
      title: "Welcome to MCP Portal",
      description:
        "Manage your Model Context Protocol servers from one central dashboard",
      component: <WelcomeStep />,
    },
    {
      title: "Add Your First Server",
      description:
        "Choose from our catalog or add a custom server configuration",
      component: <AddServerStep />,
    },
    {
      title: "Monitor & Manage",
      description:
        "Keep track of your servers with real-time monitoring and logs",
      component: <MonitoringStep />,
    },
  ];

  return (
    <Card className="max-w-2xl mx-auto">
      <CardHeader>
        <div className="flex items-center justify-between mb-4">
          <div className="flex space-x-2">
            {steps.map((_, index) => (
              <div
                key={index}
                className={clsx(
                  "w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium",
                  index <= currentStep
                    ? "bg-primary text-primary-foreground"
                    : "bg-muted text-muted-foreground"
                )}
              >
                {index + 1}
              </div>
            ))}
          </div>
          <Badge variant="secondary">
            Step {currentStep + 1} of {steps.length}
          </Badge>
        </div>

        <CardTitle>{steps[currentStep].title}</CardTitle>
        <CardDescription>{steps[currentStep].description}</CardDescription>
      </CardHeader>

      <CardContent>{steps[currentStep].component}</CardContent>

      <CardFooter className="flex justify-between">
        <Button
          variant="outline"
          onClick={() => setCurrentStep(currentStep - 1)}
          disabled={currentStep === 0}
        >
          Previous
        </Button>

        <Button
          onClick={() => {
            if (currentStep < steps.length - 1) {
              setCurrentStep(currentStep + 1);
            } else {
              completeOnboarding();
            }
          }}
        >
          {currentStep < steps.length - 1 ? "Next" : "Get Started"}
        </Button>
      </CardFooter>
    </Card>
  );
}
```

**Feature Introduction**

```tsx
// Progressive disclosure for advanced features
function FeatureSpotlight({
  feature,
  onDismiss,
}: {
  feature: string;
  onDismiss: () => void;
}) {
  return (
    <div className="relative">
      <div className="absolute inset-0 bg-black/50 z-40" />
      <div className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 z-50">
        <Card className="max-w-sm">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Sparkles className="h-5 w-5 text-primary" />
              New Feature: {feature}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground mb-4">
              Discover how this new feature can improve your workflow.
            </p>
            <div className="flex space-x-2">
              <Button size="sm">Learn More</Button>
              <Button size="sm" variant="outline" onClick={onDismiss}>
                Maybe Later
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
```

---

## 12. Icon Usage Guidelines

### Icon Libraries

**Primary: Heroicons**

- Consistent style with two variants: outline and solid
- Optimized for web interfaces
- Excellent at small sizes (16px, 20px, 24px)

**Secondary: Lucide React**

- Additional icons not available in Heroicons
- Consistent stroke width and style
- Good for complex technical icons

**Radix Icons**

- UI-specific icons for components
- Minimal, functional design
- Perfect for button and form icons

### Icon Implementation

**Consistent Icon Component**

```tsx
// Standardized icon wrapper
import { cva, type VariantProps } from 'class-variance-authority';
import * as HeroIcons from '@heroicons/react/24/outline';
import * as LucideIcons from 'lucide-react';

const iconVariants = cva("flex-shrink-0", {
  variants: {
    size: {
      xs: "h-3 w-3",
      sm: "h-4 w-4",
      md: "h-5 w-5",
      lg: "h-6 w-6",
      xl: "h-8 w-8"
    }
  },
  defaultVariants: {
    size: "md"
  }
});

interface IconProps extends VariantProps<typeof iconVariants> {
  name: keyof typeof HeroIcons | keyof typeof LucideIcons;
  className?: string;
}

function Icon({ name, size, className }: IconProps) {
  const HeroIcon = HeroIcons[name as keyof typeof HeroIcons];
  const LucideIcon = LucideIcons[name as keyof typeof LucideIcons];

  const IconComponent = HeroIcon || LucideIcon;

  if (!IconComponent) {
    console.warn(`Icon "${name}" not found`);
    return null;
  }

  return (
    <IconComponent
      className={clsx(iconVariants({ size }), className)}
    />
  );
}

// Usage examples
<Icon name="ServerIcon" size="md" className="text-primary" />
<Icon name="CheckCircleIcon" size="sm" className="text-success-600" />
```

### Semantic Icon Usage

**Status Icons**

```tsx
const statusIcons = {
  enabled: { icon: "CheckCircleIcon", className: "text-success-600" },
  disabled: { icon: "MinusCircleIcon", className: "text-muted-foreground" },
  running: { icon: "PlayCircleIcon", className: "text-docker-600" },
  stopped: { icon: "StopCircleIcon", className: "text-muted-foreground" },
  error: { icon: "XCircleIcon", className: "text-destructive" },
  loading: { icon: "ArrowPathIcon", className: "text-primary animate-spin" },
  warning: { icon: "ExclamationTriangleIcon", className: "text-warning-600" },
};

function StatusIcon({ status }: { status: keyof typeof statusIcons }) {
  const { icon, className } = statusIcons[status];
  return <Icon name={icon} className={className} />;
}
```

**Action Icons**

```tsx
// Consistent action button icons
const actionIcons = {
  edit: "PencilIcon",
  delete: "TrashIcon",
  view: "EyeIcon",
  download: "ArrowDownTrayIcon",
  upload: "ArrowUpTrayIcon",
  refresh: "ArrowPathIcon",
  settings: "CogIcon",
  info: "InformationCircleIcon",
  external: "ArrowTopRightOnSquareIcon",
  copy: "DocumentDuplicateIcon",
};

function ActionButton({
  action,
  children,
  ...props
}: {
  action: keyof typeof actionIcons;
  children: React.ReactNode;
}) {
  return (
    <Button {...props}>
      <Icon name={actionIcons[action]} size="sm" className="mr-2" />
      {children}
    </Button>
  );
}
```

### Icon Size Guidelines

**Context-Appropriate Sizing**

```tsx
// Button icons
<Button size="sm">
  <Icon name="PlusIcon" size="xs" className="mr-1" />
  Add Server
</Button>

<Button size="default">
  <Icon name="PlayIcon" size="sm" className="mr-2" />
  Start Server
</Button>

<Button size="lg">
  <Icon name="DownloadIcon" size="md" className="mr-2" />
  Download Logs
</Button>

// Status indicators
<div className="flex items-center gap-2">
  <Icon name="CheckCircleIcon" size="sm" className="text-success-600" />
  <span className="text-sm">Server running</span>
</div>

// Navigation icons
<nav className="flex items-center space-x-4">
  <NavItem>
    <Icon name="HomeIcon" size="md" />
    Dashboard
  </NavItem>
  <NavItem>
    <Icon name="ServerIcon" size="md" />
    Servers
  </NavItem>
</nav>
```

---

## 13. Performance Guidelines

### Code Splitting & Lazy Loading

**Page-Level Code Splitting**

```tsx
// Lazy load heavy pages
import { lazy, Suspense } from "react";

const ServerDetailsPage = lazy(() => import("@/app/servers/[id]/page"));
const LogViewerPage = lazy(() => import("@/app/logs/page"));
const MetricsPage = lazy(() => import("@/app/metrics/page"));

// Route with loading fallback
function AppRouter() {
  return (
    <Router>
      <Routes>
        <Route
          path="/servers/:id"
          element={
            <Suspense fallback={<PageSkeleton />}>
              <ServerDetailsPage />
            </Suspense>
          }
        />
      </Routes>
    </Router>
  );
}
```

**Component-Level Optimization**

```tsx
// Lazy load heavy components
const LogViewer = lazy(() => import("@/components/LogViewer"));
const MetricsChart = lazy(() => import("@/components/MetricsChart"));

// Only load when needed
function ServerDetails({ server }) {
  const [showLogs, setShowLogs] = useState(false);
  const [showMetrics, setShowMetrics] = useState(false);

  return (
    <div className="space-y-6">
      <ServerInfo server={server} />

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="logs">Logs</TabsTrigger>
          <TabsTrigger value="metrics">Metrics</TabsTrigger>
        </TabsList>

        <TabsContent value="logs">
          {showLogs && (
            <Suspense fallback={<LogsSkeleton />}>
              <LogViewer serverId={server.id} />
            </Suspense>
          )}
        </TabsContent>

        <TabsContent value="metrics">
          {showMetrics && (
            <Suspense fallback={<MetricsSkeleton />}>
              <MetricsChart serverId={server.id} />
            </Suspense>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}
```

### Bundle Optimization

**Import Optimization**

```tsx
// Tree-shake icon imports
import { CheckCircleIcon, XCircleIcon } from "@heroicons/react/24/outline";

// Instead of importing entire libraries
// ❌ import * as HeroIcons from '@heroicons/react/24/outline';

// Dynamic imports for heavy utilities
const formatBytes = lazy(() => import("@/utils/formatBytes"));
const downloadFile = lazy(() => import("@/utils/downloadFile"));

// Barrel export optimization - avoid deep imports
// ✅ import { ServerCard, ServerList } from '@/components/server';
// ❌ import ServerCard from '@/components/server/ServerCard';
```

**Image Optimization**

```tsx
// Next.js Image component with optimization
import Image from "next/image";

function ServerLogo({ server }: { server: Server }) {
  return (
    <Image
      src={server.logoUrl}
      alt={`${server.name} logo`}
      width={64}
      height={64}
      className="rounded-lg"
      placeholder="blur"
      blurDataURL="data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD..."
      priority={false} // Only true for above-the-fold images
    />
  );
}
```

### Virtual Scrolling

**Large Dataset Handling**

```tsx
// Virtual scrolling for large server lists
import { FixedSizeList as List } from "react-window";

function VirtualizedServerList({ servers }: { servers: Server[] }) {
  const Row = ({
    index,
    style,
  }: {
    index: number;
    style: React.CSSProperties;
  }) => (
    <div style={style}>
      <ServerCard server={servers[index]} />
    </div>
  );

  return (
    <List
      height={600}
      itemCount={servers.length}
      itemSize={120}
      className="scrollbar-thin"
    >
      {Row}
    </List>
  );
}

// Intersection observer for infinite loading
function useInfiniteServers() {
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useInfiniteQuery({
      queryKey: ["servers"],
      queryFn: ({ pageParam = 0 }) => fetchServers(pageParam),
      getNextPageParam: (lastPage, pages) => lastPage.nextCursor,
    });

  const lastServerRef = useCallback(
    (node: HTMLDivElement) => {
      if (isFetchingNextPage) return;
      if (observer.current) observer.current.disconnect();

      observer.current = new IntersectionObserver((entries) => {
        if (entries[0].isIntersecting && hasNextPage) {
          fetchNextPage();
        }
      });

      if (node) observer.current.observe(node);
    },
    [isFetchingNextPage, fetchNextPage, hasNextPage]
  );

  return { data, lastServerRef, isFetchingNextPage };
}
```

### Memory Management

**Component Cleanup**

```tsx
// Proper cleanup for subscriptions and timers
function useServerStatus(serverId: string) {
  const [status, setStatus] = useState<ServerStatus>("unknown");

  useEffect(() => {
    const ws = new WebSocket(`/api/servers/${serverId}/status`);

    ws.onmessage = (event) => {
      setStatus(JSON.parse(event.data).status);
    };

    // Cleanup WebSocket connection
    return () => {
      ws.close();
    };
  }, [serverId]);

  return status;
}

// Debounced search to prevent excessive API calls
import { useDebouncedCallback } from "use-debounce";

function SearchInput() {
  const [query, setQuery] = useState("");

  const debouncedSearch = useDebouncedCallback(
    (searchQuery: string) => {
      // Perform search
      searchServers(searchQuery);
    },
    300 // Wait 300ms after user stops typing
  );

  useEffect(() => {
    debouncedSearch(query);
  }, [query, debouncedSearch]);

  return (
    <Input
      value={query}
      onChange={(e) => setQuery(e.target.value)}
      placeholder="Search servers..."
    />
  );
}
```

---

## 14. Developer Guidelines

### Component Development Standards

**File Structure**

```
src/components/
├── ui/                 # Base UI components (shadcn/ui)
│   ├── button.tsx
│   ├── input.tsx
│   └── index.ts
├── server/            # Server-related components
│   ├── ServerCard.tsx
│   ├── ServerList.tsx
│   ├── ServerStatus.tsx
│   └── index.ts
├── forms/             # Form components
│   ├── FormField.tsx
│   ├── FormSection.tsx
│   └── index.ts
└── layout/            # Layout components
    ├── Header.tsx
    ├── Sidebar.tsx
    └── index.ts
```

**Component Template**

```tsx
// src/components/server/ServerCard.tsx
import { forwardRef } from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { clsx } from "clsx";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icon";

import type { Server } from "@/types/server";

// Component variants using CVA
const cardVariants = cva("transition-all duration-200 hover:shadow-md", {
  variants: {
    status: {
      enabled: "border-success-200 bg-success-50/50",
      disabled: "border-border bg-card",
      error: "border-destructive-200 bg-destructive-50/50",
    },
    size: {
      default: "p-6",
      compact: "p-4",
    },
  },
  defaultVariants: {
    status: "disabled",
    size: "default",
  },
});

// Component props interface
interface ServerCardProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof cardVariants> {
  server: Server;
  onEnable?: () => void;
  onDisable?: () => void;
  onView?: () => void;
}

// Component with forwardRef for proper ref handling
const ServerCard = forwardRef<HTMLDivElement, ServerCardProps>(
  (
    { server, status, size, onEnable, onDisable, onView, className, ...props },
    ref
  ) => {
    return (
      <Card
        ref={ref}
        className={clsx(cardVariants({ status, size }), className)}
        {...props}
      >
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-base font-medium">{server.name}</CardTitle>
          <Badge variant={server.enabled ? "default" : "secondary"}>
            {server.enabled ? "Enabled" : "Disabled"}
          </Badge>
        </CardHeader>

        <CardContent>
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              {server.description}
            </p>

            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <Icon
                  name={
                    server.status === "running"
                      ? "PlayCircleIcon"
                      : "StopCircleIcon"
                  }
                  size="sm"
                  className={
                    server.status === "running"
                      ? "text-success-600"
                      : "text-muted-foreground"
                  }
                />
                <span className="text-xs text-muted-foreground capitalize">
                  {server.status}
                </span>
              </div>

              <div className="flex space-x-1">
                <Button size="sm" variant="outline" onClick={onView}>
                  View
                </Button>
                {server.enabled ? (
                  <Button size="sm" variant="destructive" onClick={onDisable}>
                    Disable
                  </Button>
                ) : (
                  <Button size="sm" onClick={onEnable}>
                    Enable
                  </Button>
                )}
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    );
  }
);

ServerCard.displayName = "ServerCard";

export { ServerCard, type ServerCardProps };
```

### Testing Standards

**Component Testing**

```tsx
// src/components/server/__tests__/ServerCard.test.tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { axe, toHaveNoViolations } from "jest-axe";

import { ServerCard } from "../ServerCard";
import type { Server } from "@/types/server";

expect.extend(toHaveNoViolations);

const mockServer: Server = {
  id: "test-server",
  name: "Test Server",
  description: "A test MCP server",
  enabled: true,
  status: "running",
};

describe("ServerCard", () => {
  it("renders server information correctly", () => {
    render(<ServerCard server={mockServer} />);

    expect(screen.getByText("Test Server")).toBeInTheDocument();
    expect(screen.getByText("A test MCP server")).toBeInTheDocument();
    expect(screen.getByText("Enabled")).toBeInTheDocument();
  });

  it("calls onEnable when enable button is clicked", () => {
    const onEnable = jest.fn();
    const disabledServer = { ...mockServer, enabled: false };

    render(<ServerCard server={disabledServer} onEnable={onEnable} />);

    const enableButton = screen.getByText("Enable");
    fireEvent.click(enableButton);

    expect(onEnable).toHaveBeenCalledTimes(1);
  });

  it("should be accessible", async () => {
    const { container } = render(<ServerCard server={mockServer} />);
    const results = await axe(container);
    expect(results).toHaveNoViolations();
  });

  it("supports custom className", () => {
    const { container } = render(
      <ServerCard server={mockServer} className="custom-class" />
    );

    expect(container.firstChild).toHaveClass("custom-class");
  });
});
```

### Documentation Standards

**Component Documentation**

````tsx
/**
 * ServerCard - Displays MCP server information with action buttons
 *
 * @example
 * ```tsx
 * <ServerCard
 *   server={server}
 *   onEnable={() => enableServer(server.id)}
 *   onDisable={() => disableServer(server.id)}
 *   onView={() => router.push(`/servers/${server.id}`)}
 * />
 * ```
 *
 * @param server - Server object containing id, name, description, status
 * @param onEnable - Callback fired when enable button is clicked
 * @param onDisable - Callback fired when disable button is clicked
 * @param onView - Callback fired when view button is clicked
 * @param status - Visual status variant (enabled, disabled, error)
 * @param size - Card size variant (default, compact)
 */
````

**Storybook Stories**

```tsx
// src/components/server/ServerCard.stories.tsx
import type { Meta, StoryObj } from "@storybook/react";
import { ServerCard } from "./ServerCard";

const meta: Meta<typeof ServerCard> = {
  title: "Components/Server/ServerCard",
  component: ServerCard,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Card component for displaying MCP server information and actions.",
      },
    },
  },
  argTypes: {
    status: {
      control: "radio",
      options: ["enabled", "disabled", "error"],
    },
    size: {
      control: "radio",
      options: ["default", "compact"],
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    server: {
      id: "example-server",
      name: "Example Server",
      description: "An example MCP server for demonstration",
      enabled: true,
      status: "running",
    },
  },
};

export const Disabled: Story = {
  args: {
    ...Default.args,
    server: {
      ...Default.args.server,
      enabled: false,
      status: "stopped",
    },
  },
};

export const Error: Story = {
  args: {
    ...Default.args,
    server: {
      ...Default.args.server,
      status: "error",
    },
    status: "error",
  },
};

export const Compact: Story = {
  args: {
    ...Default.args,
    size: "compact",
  },
};
```

### Code Quality Standards

**Linting Configuration**

```json
// .eslintrc.json
{
  "extends": [
    "next/core-web-vitals",
    "@typescript-eslint/recommended",
    "prettier"
  ],
  "rules": {
    "@typescript-eslint/no-unused-vars": "error",
    "@typescript-eslint/prefer-const": "error",
    "react-hooks/exhaustive-deps": "error",
    "react/prop-types": "off",
    "react/react-in-jsx-scope": "off"
  }
}
```

**TypeScript Configuration**

```json
// tsconfig.json
{
  "compilerOptions": {
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "noImplicitReturns": true,
    "noFallthroughCasesInSwitch": true,
    "noImplicitOverride": true
  }
}
```

---

## Conclusion

These UI/UX guidelines establish a comprehensive foundation for building the MCP Portal frontend with consistency, accessibility, and performance at its core. The guidelines prioritize:

1. **Enterprise-grade quality** suitable for IT operations teams
2. **Accessibility compliance** with WCAG 2.1 AA standards
3. **Performance optimization** for smooth user experiences
4. **Developer experience** with clear patterns and documentation
5. **Maintainability** through consistent component architecture

### Implementation Priorities

**Phase 1: Core Components** (Weeks 1-2)

- Base UI component library setup
- Color system and theming implementation
- Typography and layout foundations

**Phase 2: Complex Components** (Weeks 3-4)

- Server management components
- Form patterns and validation
- Loading and error states

**Phase 3: Advanced Features** (Weeks 5-6)

- Animation and interaction patterns
- Advanced accessibility features
- Performance optimization

**Phase 4: Testing & Documentation** (Weeks 7-8)

- Comprehensive component testing
- Storybook documentation
- Accessibility audit and fixes

### Resources for Implementation

- **Design Tokens**: `/frontend/src/styles/tokens.css`
- **Component Library**: `/frontend/src/components/ui/`
- **Documentation**: Storybook + TypeDoc
- **Testing**: Jest + Testing Library + Axe
- **Performance**: Bundle Analyzer + Lighthouse CI

This comprehensive guide ensures the MCP Portal delivers a professional, accessible, and performant user experience that meets enterprise standards while providing developers with clear, consistent patterns for efficient development.
