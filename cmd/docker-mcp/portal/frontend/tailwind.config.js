/** @type {import('tailwindcss').Config} */
import tailwindcssAnimate from 'tailwindcss-animate';
import tailwindcssForms from '@tailwindcss/forms';

const config = {
  // Tailwind v4: Enhanced dark mode with system preference detection
  darkMode: ['class', '[data-theme="dark"]'],

  // Optimized content paths for v4 with better performance
  content: {
    files: [
      './src/app/**/*.{ts,tsx,mdx}',
      './src/components/**/*.{ts,tsx}',
      './src/hooks/**/*.{ts,tsx}',
      './src/lib/**/*.{ts,tsx}',
      './src/providers/**/*.{ts,tsx}',
      './src/contexts/**/*.{ts,tsx}',
      './src/stores/**/*.{ts,tsx}',
      './src/utils/**/*.{ts,tsx}',
      './src/services/**/*.{ts,tsx}',
      './src/config/**/*.{ts,tsx}',
      './src/types/**/*.{ts,tsx}',
    ],
    // v4: Enhanced content extraction for better tree-shaking
    extract: {
      tsx: content => {
        // Extract dynamic class names from clsx, cn, and className variables
        const matches =
          content.match(
            /(?:class(?:Name)?|cn|clsx)\s*[:=]\s*["'`]([^"'`]*)["'`]/g
          ) || [];
        return matches
          .map(match => match.replace(/.*["'`]([^"'`]*)["'`].*/, '$1'))
          .join(' ');
      },
    },
    // Safelist for dynamic classes used in components
    safelist: [
      // Status indicators
      'bg-success-50',
      'bg-success-900/20',
      'text-success-700',
      'text-success-300',
      'bg-error-50',
      'bg-error-900/20',
      'text-error-700',
      'text-error-300',
      'bg-warning-50',
      'bg-warning-900/20',
      'text-warning-700',
      'text-warning-300',
      'bg-docker-50',
      'bg-docker-900/20',
      'text-docker-700',
      'text-docker-300',
      // Grid layout classes
      'col-span-1',
      'col-span-2',
      'col-span-3',
      'col-span-4',
      'col-span-6',
      'col-span-12',
      'grid-cols-1',
      'grid-cols-2',
      'grid-cols-3',
      'grid-cols-4',
      'grid-cols-6',
      'grid-cols-12',
      // Animation delays for staggered animations
      'delay-100',
      'delay-200',
      'delay-300',
      'delay-500',
    ],
  },
  // v4: No prefix needed, better performance
  prefix: '',
  theme: {
    // v4: Enhanced container with responsive padding
    container: {
      center: true,
      padding: {
        DEFAULT: '1rem',
        sm: '1.5rem',
        lg: '2rem',
        xl: '2.5rem',
        '2xl': '3rem',
      },
      screens: {
        sm: '640px',
        md: '768px',
        lg: '1024px',
        xl: '1280px',
        '2xl': '1400px',
      },
    },
    // v4: Enhanced breakpoints for better responsive design
    screens: {
      xs: '475px',
      sm: '640px',
      md: '768px',
      lg: '1024px',
      xl: '1280px',
      '2xl': '1400px',
      '3xl': '1600px',
    },
    extend: {
      colors: {
        border: 'hsl(var(--border))',
        input: 'hsl(var(--input))',
        ring: 'hsl(var(--ring))',
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))',
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))',
        },
        destructive: {
          DEFAULT: 'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))',
        },
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))',
        },
        accent: {
          DEFAULT: 'hsl(var(--accent))',
          foreground: 'hsl(var(--accent-foreground))',
        },
        popover: {
          DEFAULT: 'hsl(var(--popover))',
          foreground: 'hsl(var(--popover-foreground))',
        },
        card: {
          DEFAULT: 'hsl(var(--card))',
          foreground: 'hsl(var(--card-foreground))',
        },
        // MCP Portal brand colors
        docker: {
          50: '#f0f9ff',
          100: '#e0f2fe',
          200: '#bae6fd',
          300: '#7dd3fc',
          400: '#38bdf8',
          500: '#0ea5e9',
          600: '#0284c7',
          700: '#0369a1',
          800: '#075985',
          900: '#0c4a6e',
        },
        success: {
          50: '#f0fdf4',
          100: '#dcfce7',
          200: '#bbf7d0',
          300: '#86efac',
          400: '#4ade80',
          500: '#22c55e',
          600: '#16a34a',
          700: '#15803d',
          800: '#166534',
          900: '#14532d',
        },
        warning: {
          50: '#fffbeb',
          100: '#fef3c7',
          200: '#fde68a',
          300: '#fcd34d',
          400: '#fbbf24',
          500: '#f59e0b',
          600: '#d97706',
          700: '#b45309',
          800: '#92400e',
          900: '#78350f',
        },
        error: {
          50: '#fef2f2',
          100: '#fee2e2',
          200: '#fecaca',
          300: '#fca5a5',
          400: '#f87171',
          500: '#ef4444',
          600: '#dc2626',
          700: '#b91c1c',
          800: '#991b1b',
          900: '#7f1d1d',
        },
      },
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
      },
      keyframes: {
        'accordion-down': {
          from: { height: '0' },
          to: { height: 'var(--radix-accordion-content-height)' },
        },
        'accordion-up': {
          from: { height: 'var(--radix-accordion-content-height)' },
          to: { height: '0' },
        },
        'fade-in': {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        'slide-in-right': {
          '0%': { transform: 'translateX(100%)' },
          '100%': { transform: 'translateX(0)' },
        },
        'slide-in-left': {
          '0%': { transform: 'translateX(-100%)' },
          '100%': { transform: 'translateX(0)' },
        },
        pulse: {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0.5' },
        },
      },
      animation: {
        'accordion-down': 'accordion-down 0.2s ease-out',
        'accordion-up': 'accordion-up 0.2s ease-out',
        'fade-in': 'fade-in 0.2s ease-out',
        'slide-in-right': 'slide-in-right 0.3s ease-out',
        'slide-in-left': 'slide-in-left 0.3s ease-out',
        pulse: 'pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
      // v4: Optimized font families with variable fonts
      fontFamily: {
        sans: [
          ['Inter Variable', 'Inter'],
          {
            fontFeatureSettings: '"cv02", "cv03", "cv04", "cv11"',
            fontVariationSettings: '"opsz" 32',
          },
        ],
        mono: [
          ['JetBrains Mono Variable', 'JetBrains Mono'],
          {
            fontFeatureSettings: '"calt", "liga"',
          },
        ],
        display: [
          ['Inter Display', 'Inter'],
          {
            fontFeatureSettings: '"cv02", "cv03", "cv04", "cv11"',
            fontVariationSettings: '"opsz" 72',
          },
        ],
      },
      // v4: Enhanced typography scale
      fontSize: {
        xs: ['0.75rem', { lineHeight: '1rem' }],
        sm: ['0.875rem', { lineHeight: '1.25rem' }],
        base: ['1rem', { lineHeight: '1.5rem' }],
        lg: ['1.125rem', { lineHeight: '1.75rem' }],
        xl: ['1.25rem', { lineHeight: '1.75rem' }],
        '2xl': ['1.5rem', { lineHeight: '2rem' }],
        '3xl': ['1.875rem', { lineHeight: '2.25rem' }],
        '4xl': ['2.25rem', { lineHeight: '2.5rem' }],
        '5xl': ['3rem', { lineHeight: '1' }],
        '6xl': ['3.75rem', { lineHeight: '1' }],
        '7xl': ['4.5rem', { lineHeight: '1' }],
        '8xl': ['6rem', { lineHeight: '1' }],
        '9xl': ['8rem', { lineHeight: '1' }],
      },
      // v4: Enhanced spacing scale
      spacing: {
        18: '4.5rem',
        88: '22rem',
        128: '32rem',
        144: '36rem',
      },
      // v4: Enhanced animation timing
      transitionDuration: {
        250: '250ms',
        350: '350ms',
        400: '400ms',
        600: '600ms',
        800: '800ms',
        900: '900ms',
      },
      // v4: Enhanced easing functions
      transitionTimingFunction: {
        'bounce-in': 'cubic-bezier(0.68, -0.55, 0.265, 1.55)',
        smooth: 'cubic-bezier(0.4, 0, 0.2, 1)',
        snappy: 'cubic-bezier(0.4, 0, 0.6, 1)',
      },
    },
  },
  // v4: Enhanced plugins with performance optimizations
  plugins: [
    tailwindcssAnimate,
    tailwindcssForms({
      strategy: 'class', // Use class strategy for better control
    }),
    // v4: Custom plugin for component utilities
    function ({ addUtilities, addComponents }) {
      // Performance-optimized utilities
      addUtilities({
        '.text-wrap-balance': {
          'text-wrap': 'balance',
        },
        '.text-wrap-pretty': {
          'text-wrap': 'pretty',
        },
        '.backface-hidden': {
          'backface-visibility': 'hidden',
        },
        '.transform-gpu': {
          transform: 'translate3d(0, 0, 0)',
        },
        '.will-change-transform': {
          'will-change': 'transform',
        },
        '.will-change-auto': {
          'will-change': 'auto',
        },
      });

      // Component-level utilities for better performance
      addComponents({
        '.glass-effect': {
          'backdrop-filter': 'blur(12px) saturate(150%)',
          'background-color': 'rgba(255, 255, 255, 0.05)',
          border: '1px solid rgba(255, 255, 255, 0.1)',
        },
        '.scroll-smooth': {
          'scroll-behavior': 'smooth',
        },
      });
    },
  ],

  // v4: Experimental features for better performance
  experimental: {
    // Enable CSS-in-JS optimization
    optimizeUniversalDefaults: true,
    // Enable advanced purging
    extendedSpacingScale: true,
    // Enable container queries
    matchVariant: true,
  },

  // v4: Performance optimizations
  corePlugins: {
    // Disable unused core plugins for smaller bundle
    preflight: true,
    container: true,
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
    float: false, // Disable float utilities (modern layout)
    clear: false, // Disable clear utilities (modern layout)
    objectPosition: true,
    overflow: true,
    overscrollBehavior: true,
    textOverflow: true,
    whitespace: true,
  },
};

export default config;
