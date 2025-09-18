/**
 * Tailwind CSS v4 Performance Configuration
 * Optimized for production builds and development speed
 */

import { fontFamily } from 'tailwindcss/defaultTheme';
import tailwindcssAnimate from 'tailwindcss-animate';

/** @type {import('tailwindcss').Config} */
const config = {
  // Production mode optimizations
  mode: process.env.NODE_ENV === 'production' ? 'jit' : 'watch',

  // Enhanced content configuration for optimal purging
  content: {
    files: [
      './src/**/*.{js,ts,jsx,tsx,mdx}',
      './components/**/*.{js,ts,jsx,tsx}',
      './pages/**/*.{js,ts,jsx,tsx}',
      './app/**/*.{js,ts,jsx,tsx}',
    ],
    // Transform content for better class extraction
    transform: {
      tsx: content => {
        // Extract all potential class names including dynamic ones
        const classRegex =
          /(?:className|class)=\{[^}]*\}|(?:className|class)="[^"]*"|(?:className|class)='[^']*'/g;
        const matches = content.match(classRegex) || [];
        const classes = matches
          .map(match => {
            // Extract class names from template literals and expressions
            const innerContent = match
              .replace(/(?:className|class)=["'{]/, '')
              .replace(/["'}]$/, '');
            return innerContent;
          })
          .join(' ');

        return content + ' ' + classes;
      },
    },
    // Extract additional patterns
    extract: {
      tsx: content => {
        // Extract clsx, cn, and cva calls
        const dynamicClassRegex = /(?:clsx|cn|cva)\s*\([^)]*\)/g;
        const matches = content.match(dynamicClassRegex) || [];
        return matches.join(' ').replace(/[(){}[\]"'`]/g, ' ');
      },
    },
    // Safelist critical classes that might be missed
    safelist: [
      // Animation classes
      { pattern: /^animate-/ },
      { pattern: /^duration-/ },
      { pattern: /^delay-/ },

      // Grid and layout classes
      { pattern: /^(grid-cols|col-span|row-span)-/ },
      { pattern: /^gap-/ },

      // Status and theme classes
      {
        pattern:
          /^(status|health)-(enabled|disabled|running|stopped|error|unknown|healthy|unhealthy|degraded)$/,
      },

      // Responsive breakpoints
      { pattern: /^(xs|sm|md|lg|xl|2xl|3xl):/ },

      // Container queries
      { pattern: /^container-/ },

      // Dynamic colors
      {
        pattern:
          /^(bg|text|border)-(primary|secondary|destructive|muted|accent)(-foreground)?$/,
      },
      { pattern: /^(bg|text|border)-(success|warning|error|docker)-/ },

      // Radix UI states
      'data-[state=open]:animate-in',
      'data-[state=closed]:animate-out',
      'data-[state=open]:fade-in-0',
      'data-[state=closed]:fade-out-0',
      'data-[state=open]:zoom-in-95',
      'data-[state=closed]:zoom-out-95',
      'data-[side=bottom]:slide-in-from-top-2',
      'data-[side=left]:slide-in-from-right-2',
      'data-[side=right]:slide-in-from-left-2',
      'data-[side=top]:slide-in-from-bottom-2',
    ],
  },

  // Performance-focused theme configuration
  theme: {
    // Optimize color palette - only include used colors
    colors: {
      // Keep only essential colors, extend with CSS variables
      transparent: 'transparent',
      current: 'currentColor',
      inherit: 'inherit',

      // CSS variable-based colors for better performance
      background: 'hsl(var(--background))',
      foreground: 'hsl(var(--foreground))',
      card: 'hsl(var(--card))',
      'card-foreground': 'hsl(var(--card-foreground))',
      popover: 'hsl(var(--popover))',
      'popover-foreground': 'hsl(var(--popover-foreground))',
      primary: 'hsl(var(--primary))',
      'primary-foreground': 'hsl(var(--primary-foreground))',
      secondary: 'hsl(var(--secondary))',
      'secondary-foreground': 'hsl(var(--secondary-foreground))',
      muted: 'hsl(var(--muted))',
      'muted-foreground': 'hsl(var(--muted-foreground))',
      accent: 'hsl(var(--accent))',
      'accent-foreground': 'hsl(var(--accent-foreground))',
      destructive: 'hsl(var(--destructive))',
      'destructive-foreground': 'hsl(var(--destructive-foreground))',
      border: 'hsl(var(--border))',
      input: 'hsl(var(--input))',
      ring: 'hsl(var(--ring))',

      // Status colors with CSS variables
      success: {
        50: 'hsl(142 76% 96%)',
        500: 'hsl(142 71% 45%)',
        700: 'hsl(142 72% 29%)',
      },
      warning: {
        50: 'hsl(48 100% 96%)',
        500: 'hsl(45 93% 47%)',
        700: 'hsl(37 91% 55%)',
      },
      error: {
        50: 'hsl(0 86% 97%)',
        500: 'hsl(0 84% 60%)',
        700: 'hsl(0 70% 35%)',
      },
      docker: {
        50: 'hsl(199 100% 96%)',
        500: 'hsl(199 89% 48%)',
        700: 'hsl(201 96% 32%)',
      },
    },

    // Optimize spacing scale - remove unused values
    spacing: {
      px: '1px',
      0: '0px',
      0.5: '0.125rem',
      1: '0.25rem',
      1.5: '0.375rem',
      2: '0.5rem',
      2.5: '0.625rem',
      3: '0.75rem',
      3.5: '0.875rem',
      4: '1rem',
      5: '1.25rem',
      6: '1.5rem',
      7: '1.75rem',
      8: '2rem',
      9: '2.25rem',
      10: '2.5rem',
      11: '2.75rem',
      12: '3rem',
      14: '3.5rem',
      16: '4rem',
      18: '4.5rem',
      20: '5rem',
      24: '6rem',
      28: '7rem',
      32: '8rem',
      36: '9rem',
      40: '10rem',
      44: '11rem',
      48: '12rem',
      52: '13rem',
      56: '14rem',
      60: '15rem',
      64: '16rem',
      72: '18rem',
      80: '20rem',
      96: '24rem',
    },

    // Optimize font sizes - only include used sizes
    fontSize: {
      xs: ['0.75rem', { lineHeight: '1rem' }],
      sm: ['0.875rem', { lineHeight: '1.25rem' }],
      base: ['1rem', { lineHeight: '1.5rem' }],
      lg: ['1.125rem', { lineHeight: '1.75rem' }],
      xl: ['1.25rem', { lineHeight: '1.75rem' }],
      '2xl': ['1.5rem', { lineHeight: '2rem' }],
      '3xl': ['1.875rem', { lineHeight: '2.25rem' }],
      '4xl': ['2.25rem', { lineHeight: '2.5rem' }],
    },

    // Performance-optimized animations
    animation: {
      'fade-in': 'fadeIn 150ms ease-out',
      'fade-out': 'fadeOut 150ms ease-in',
      'slide-up': 'slideUp 200ms ease-out',
      'slide-down': 'slideDown 200ms ease-out',
      'scale-in': 'scaleIn 150ms ease-out',
      'scale-out': 'scaleOut 150ms ease-in',
      'spin-slow': 'spin 3s linear infinite',
    },

    // Essential keyframes only
    keyframes: {
      fadeIn: {
        '0%': { opacity: '0' },
        '100%': { opacity: '1' },
      },
      fadeOut: {
        '0%': { opacity: '1' },
        '100%': { opacity: '0' },
      },
      slideUp: {
        '0%': { transform: 'translateY(10px)', opacity: '0' },
        '100%': { transform: 'translateY(0)', opacity: '1' },
      },
      slideDown: {
        '0%': { transform: 'translateY(-10px)', opacity: '0' },
        '100%': { transform: 'translateY(0)', opacity: '1' },
      },
      scaleIn: {
        '0%': { transform: 'scale(0.95)', opacity: '0' },
        '100%': { transform: 'scale(1)', opacity: '1' },
      },
      scaleOut: {
        '0%': { transform: 'scale(1)', opacity: '1' },
        '100%': { transform: 'scale(0.95)', opacity: '0' },
      },
    },

    extend: {
      // Only extend what's necessary
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
      },
      fontFamily: {
        sans: ['Inter Variable', ...fontFamily.sans],
        mono: ['JetBrains Mono Variable', ...fontFamily.mono],
      },
    },
  },

  // Disable unused core plugins for smaller bundle
  corePlugins: {
    // Essential plugins only
    preflight: true,
    container: true,
    space: true,
    divideWidth: false,
    divideColor: false,
    divideStyle: false,
    divideOpacity: false,
    accessibility: true,
    pointerEvents: true,
    visibility: true,
    position: true,
    inset: true,
    isolation: true,
    zIndex: true,
    order: true,
    gridColumn: true,
    gridColumnStart: true,
    gridColumnEnd: true,
    gridRow: true,
    gridRowStart: true,
    gridRowEnd: true,
    flexDirection: true,
    flexWrap: true,
    flex: true,
    flexGrow: true,
    flexShrink: true,
    tableLayout: false, // Disable if not using tables
    borderCollapse: false, // Disable if not using tables
    transform: true,
    transformOrigin: true,
    scale: true,
    rotate: true,
    translate: true,
    skew: true,
    transitionProperty: true,
    transitionDelay: true,
    transitionDuration: true,
    transitionTimingFunction: true,
    animation: true,
    cursor: true,
    userSelect: true,
    resize: false, // Disable if not needed
    scrollSnapType: false, // Disable if not needed
    scrollSnapAlign: false, // Disable if not needed
    scrollSnapStop: false, // Disable if not needed
    scrollMargin: false, // Disable if not needed
    scrollPadding: false, // Disable if not needed
    listStyleType: false, // Disable if not using lists
    listStylePosition: false, // Disable if not using lists
    appearance: true,
    columns: false, // Disable if not using CSS columns
    breakBefore: false, // Disable if not using print styles
    breakInside: false, // Disable if not using print styles
    breakAfter: false, // Disable if not using print styles
    gridAutoColumns: false, // Disable if not using auto grid
    gridAutoFlow: false, // Disable if not using auto grid
    gridAutoRows: false, // Disable if not using auto grid
    gridTemplateColumns: true,
    gridTemplateRows: true,
    gap: true,
    justifyContent: true,
    justifyItems: true,
    justifySelf: true,
    alignContent: true,
    alignItems: true,
    alignSelf: true,
    placeContent: false, // Use justify/align instead
    placeItems: false, // Use justify/align instead
    placeSelf: false, // Use justify/align instead
    overflow: true,
    overscrollBehavior: true,
    scrollBehavior: true,
    textOverflow: true,
    whitespace: true,
    wordBreak: true,
    content: false, // Disable CSS content property
  },

  // Minimal plugins for production
  plugins: [
    // Only include essential plugins
    tailwindcssAnimate,

    // Custom minimal utilities
    function ({ addUtilities, addComponents }) {
      // Essential utilities only
      addUtilities({
        '.text-balance': {
          'text-wrap': 'balance',
        },
        '.transform-gpu': {
          transform: 'translate3d(0, 0, 0)',
        },
        '.backface-hidden': {
          'backface-visibility': 'hidden',
        },
      });

      // Essential components only
      addComponents({
        '.focus-ring': {
          '@apply focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2':
            {},
        },
        '.transition-fast': {
          transition: 'all 150ms cubic-bezier(0.4, 0, 0.2, 1)',
        },
      });
    },
  ],

  // Production optimizations
  ...(process.env.NODE_ENV === 'production' && {
    // Enable all optimizations for production
    experimental: {
      optimizeUniversalDefaults: true,
    },

    // Disable development features
    devtools: false,
  }),
};

export default config;
