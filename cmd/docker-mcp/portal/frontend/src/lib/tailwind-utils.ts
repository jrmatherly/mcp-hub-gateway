/**
 * Tailwind CSS v4 Utility Functions
 * Optimized for performance and type safety
 */

import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

/**
 * Enhanced class name utility that merges Tailwind classes correctly
 * Optimized for v4 with better performance and type safety
 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * V4: Performance-optimized class name utility for high-frequency usage
 * Uses a WeakMap cache for better performance
 */
const classCache = new WeakMap<object, string>();

export function cnCached(...inputs: ClassValue[]): string {
  // Create cache key from inputs
  const cacheKey = { inputs };

  // Check cache first
  const cached = classCache.get(cacheKey);
  if (cached !== undefined) {
    return cached;
  }

  // Compute and cache result
  const result = twMerge(clsx(inputs));
  classCache.set(cacheKey, result);

  return result;
}

/**
 * V4: Responsive class utility generator
 * Generates responsive classes for all breakpoints
 */
export function responsive(classes: {
  base?: string;
  xs?: string;
  sm?: string;
  md?: string;
  lg?: string;
  xl?: string;
  '2xl'?: string;
  '3xl'?: string;
}): string {
  const breakpoints = [
    'base',
    'xs',
    'sm',
    'md',
    'lg',
    'xl',
    '2xl',
    '3xl',
  ] as const;

  return breakpoints
    .map(bp => {
      const className = classes[bp === 'base' ? 'base' : bp];
      if (!className) return '';

      return bp === 'base' ? className : `${bp}:${className}`;
    })
    .filter(Boolean)
    .join(' ');
}

/**
 * V4: Animation utility for performance-optimized animations
 */
export function animationClass(
  animation: string,
  options?: {
    duration?: 'fast' | 'normal' | 'slow';
    delay?: number;
    fillMode?: 'both' | 'forwards' | 'backwards' | 'none';
    gpu?: boolean;
  }
): string {
  const classes = [animation];

  if (options?.duration) {
    classes.push(
      `duration-${options.duration === 'fast' ? '150' : options.duration === 'slow' ? '350' : '250'}`
    );
  }

  if (options?.delay) {
    classes.push(`delay-${options.delay}`);
  }

  if (options?.fillMode === 'both') {
    classes.push('fill-both');
  }

  if (options?.gpu) {
    classes.push('transform-gpu');
  }

  return classes.join(' ');
}

/**
 * V4: Color utility with theme support
 */
export function colorClass(
  color: 'primary' | 'secondary' | 'destructive' | 'muted' | 'accent',
  type: 'bg' | 'text' | 'border' = 'bg',
  variant?: 'foreground'
): string {
  const suffix = variant ? `-${variant}` : '';
  return `${type}-${color}${suffix}`;
}

/**
 * V4: Status color utility for MCP Portal
 */
export function statusClass(
  status:
    | 'enabled'
    | 'disabled'
    | 'running'
    | 'stopped'
    | 'error'
    | 'unknown'
    | 'healthy'
    | 'unhealthy'
    | 'degraded',
  type: 'indicator' | 'health' = 'indicator'
): string {
  const prefix = type === 'health' ? 'health' : 'status';
  return `${prefix}-${status}`;
}

/**
 * V4: Grid utility for responsive layouts
 */
export function gridClass(
  columns: number | 'auto-fit' | 'auto-fill' | 'responsive',
  gap?: number
): string {
  const classes = [];

  if (columns === 'auto-fit') {
    classes.push('grid-auto-fit');
  } else if (columns === 'auto-fill') {
    classes.push('grid-auto-fill');
  } else if (columns === 'responsive') {
    classes.push('grid-responsive');
  } else {
    classes.push(`grid-cols-${columns}`);
  }

  if (gap) {
    classes.push(`gap-${gap}`);
  }

  return classes.join(' ');
}

/**
 * V4: Transition utility with performance optimization
 */
export function transitionClass(
  property: 'all' | 'colors' | 'transform' | 'opacity',
  speed: 'fast' | 'normal' | 'slow' = 'normal'
): string {
  if (property === 'colors') {
    return `transition-colors-${speed === 'normal' ? 'fast' : speed}`;
  }

  return `transition-${speed === 'normal' ? 'normal' : speed}`;
}

/**
 * V4: Glass effect utility
 */
export function glassClass(strength: 'normal' | 'strong' = 'normal'): string {
  return strength === 'strong' ? 'glass-strong' : 'glass';
}

/**
 * V4: Container utility with responsive padding
 */
export function containerClass(responsive = true): string {
  return responsive ? 'container-responsive' : 'container';
}

/**
 * V4: Typography utility with responsive scaling
 */
export function typographyClass(
  size: 'text' | 'heading',
  responsive = true
): string {
  return responsive ? `responsive-${size}` : '';
}

/**
 * V4: Focus utility for accessibility
 */
export function focusClass(
  style: 'default' | 'ring' | 'visible' = 'default'
): string {
  switch (style) {
    case 'ring':
      return 'focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background';
    case 'visible':
      return 'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2';
    default:
      return 'focusable';
  }
}

/**
 * V4: Performance utility for content visibility
 */
export function visibilityClass(
  visibility: 'auto' | 'hidden' | 'visible'
): string {
  return `content-${visibility}`;
}

/**
 * V4: Theme transition utility
 */
export function themeTransitionClass(): string {
  return 'theme-transition';
}

/**
 * V4: Safe area utility for mobile devices
 */
export function safeAreaClass(direction: 'x' | 'y'): string {
  return `space-${direction}-safe`;
}

/**
 * V4: Container query utility
 */
export function containerQueryClass(
  type: 'default' | 'normal' = 'default'
): string {
  return type === 'normal' ? 'container-query-normal' : 'container-query';
}

/**
 * V4: Performance will-change utility
 */
export function willChangeClass(
  property: 'transform' | 'scroll' | 'auto'
): string {
  return `will-change-${property}`;
}

/**
 * V4: Scroll behavior utility
 */
export function scrollClass(
  behavior: 'smooth' | 'auto',
  overscroll?: 'contain' | 'none'
): string {
  const classes = [`scroll-${behavior}`];

  if (overscroll) {
    classes.push(`overscroll-${overscroll}`);
  }

  return classes.join(' ');
}

/**
 * V4: Text wrapping utility
 */
export function textWrapClass(wrap: 'balance' | 'pretty' | 'no-wrap'): string {
  return `text-${wrap === 'no-wrap' ? 'no-wrap' : `wrap-${wrap}`}`;
}
