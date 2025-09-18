// Import and validate environment variables at build time
import('./src/env.mjs');

/** @type {import('next').NextConfig} */
const nextConfig = {
  // Output configuration for embedding in Go binary
  output: 'standalone',
  trailingSlash: false,
  poweredByHeader: false,
  compress: true,

  // Environment variables validation
  env: {
    CUSTOM_KEY: process.env.CUSTOM_KEY,
  },

  // API proxy to backend Go service
  async rewrites() {
    const apiBaseUrl =
      process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

    return [
      {
        source: '/api/:path*',
        destination: `${apiBaseUrl}/api/:path*`,
      },
      // WebSocket connections handled via direct client connection, not rewrites
      // WebSocket URL should be configured in client-side code
    ];
  },

  // Comprehensive security headers
  async headers() {
    return [
      {
        source: '/(.*)',
        headers: [
          // Frame protection
          {
            key: 'X-Frame-Options',
            value: 'DENY',
          },
          // Content type protection
          {
            key: 'X-Content-Type-Options',
            value: 'nosniff',
          },
          // XSS protection
          {
            key: 'X-XSS-Protection',
            value: '1; mode=block',
          },
          // Referrer policy
          {
            key: 'Referrer-Policy',
            value: 'strict-origin-when-cross-origin',
          },
          // HSTS (HTTP Strict Transport Security)
          {
            key: 'Strict-Transport-Security',
            value: 'max-age=31536000; includeSubDomains; preload',
          },
          // Permissions policy
          {
            key: 'Permissions-Policy',
            value:
              'camera=(), microphone=(), geolocation=(), interest-cohort=()',
          },
          // Content Security Policy for Azure AD and MSAL
          {
            key: 'Content-Security-Policy',
            value: [
              "default-src 'self'",
              "script-src 'self' 'unsafe-inline' 'unsafe-eval' https://login.microsoftonline.com",
              "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
              "font-src 'self' https://fonts.gstatic.com",
              "img-src 'self' data: https:",
              "connect-src 'self' https://login.microsoftonline.com https://graph.microsoft.com wss: ws:",
              "frame-src 'self' https://login.microsoftonline.com",
              "object-src 'none'",
              "base-uri 'self'",
              "form-action 'self'",
              'upgrade-insecure-requests',
            ].join('; '),
          },
        ],
      },
    ];
  },

  // TypeScript configuration
  typescript: {
    ignoreBuildErrors: false,
  },

  // ESLint configuration
  eslint: {
    ignoreDuringBuilds: false,
    dirs: ['app', 'lib', 'components', 'hooks'],
  },

  // Image optimization configuration
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        hostname: 'graph.microsoft.com',
        port: '',
        pathname: '/v1.0/me/photo/**',
      },
    ],
    formats: ['image/webp', 'image/avif'],
    deviceSizes: [640, 750, 828, 1080, 1200, 1920, 2048, 3840],
    imageSizes: [16, 32, 48, 64, 96, 128, 256, 384],
    dangerouslyAllowSVG: true,
    contentDispositionType: 'attachment',
    contentSecurityPolicy: "default-src 'self'; script-src 'none'; sandbox;",
  },

  // Webpack configuration for WebSocket and production optimization
  webpack: (config, { dev, isServer }) => {
    // WebSocket support for client-side
    if (!isServer) {
      config.resolve.fallback = {
        ...config.resolve.fallback,
        net: false,
        tls: false,
        fs: false,
        child_process: false,
      };
    }

    // Production optimizations with enhanced code splitting
    if (!dev) {
      config.optimization = {
        ...config.optimization,
        moduleIds: 'deterministic',
        splitChunks: {
          chunks: 'all',
          minSize: 20000,
          maxSize: 244000,
          minChunks: 1,
          maxAsyncRequests: 30,
          maxInitialRequests: 30,
          enforceSizeThreshold: 50000,
          cacheGroups: {
            // Framework chunk (React, Next.js core)
            framework: {
              test: /[\\/]node_modules[\\/](react|react-dom|next)[\\/]/,
              name: 'framework',
              chunks: 'all',
              priority: 40,
              enforce: true,
            },
            // Heavy visualization libraries
            charts: {
              test: /[\\/]node_modules[\\/](recharts|d3|victory)[\\/]/,
              name: 'charts',
              chunks: 'async',
              priority: 35,
              enforce: true,
            },
            // Grid layout components
            gridLayout: {
              test: /[\\/]node_modules[\\/](react-grid-layout|react-resizable)[\\/]/,
              name: 'grid-layout',
              chunks: 'async',
              priority: 35,
              enforce: true,
            },
            // Animation and canvas libraries
            animations: {
              test: /[\\/]node_modules[\\/](canvas-confetti|framer-motion|motion)[\\/]/,
              name: 'animations',
              chunks: 'async',
              priority: 35,
              enforce: true,
            },
            // UI components chunk (Radix UI, Headless UI)
            ui: {
              test: /[\\/]node_modules[\\/](@radix-ui|@headlessui|@heroicons)[\\/]/,
              name: 'ui-components',
              chunks: 'all',
              priority: 30,
              enforce: true,
            },
            // Azure/MSAL authentication
            auth: {
              test: /[\\/]node_modules[\\/](@azure\/msal)[\\/]/,
              name: 'auth',
              chunks: 'all',
              priority: 25,
              enforce: true,
            },
            // Utilities chunk
            utils: {
              test: /[\\/]node_modules[\\/](lodash|date-fns|clsx|class-variance-authority)[\\/]/,
              name: 'utils',
              chunks: 'all',
              priority: 20,
            },
            // Default vendor chunk for remaining packages
            vendor: {
              test: /[\\/]node_modules[\\/]/,
              name: 'vendors',
              chunks: 'all',
              priority: 10,
            },
            // Common chunks across pages
            common: {
              name: 'common',
              minChunks: 2,
              chunks: 'all',
              priority: 5,
              enforce: true,
            },
          },
        },
        // Enable tree shaking for better dead code elimination
        usedExports: true,
        sideEffects: false,
      };
    }

    // Bundle analyzer (conditional)
    if (process.env.ANALYZE === 'true') {
      // eslint-disable-next-line @typescript-eslint/no-require-imports
      const { BundleAnalyzerPlugin } = require('webpack-bundle-analyzer');
      config.plugins.push(
        new BundleAnalyzerPlugin({
          analyzerMode: 'static',
          openAnalyzer: false,
          generateStatsFile: true,
          statsFilename: 'bundle-stats.json',
        })
      );
    }

    return config;
  },

  // Next.js 15 specific configurations
  experimental: {
    // Optimize package imports for better tree shaking
    optimizePackageImports: [
      '@radix-ui/react-icons',
      'lucide-react',
      '@heroicons/react',
      'react-use-measure',
      'usehooks-ts',
    ],
    // Enable React 19 concurrent features
    esmExternals: true,
    // Improved client-side navigation
    optimisticClientCache: true,
    // Enhanced static optimization
    optimizeCss: true,
  },

  // Turbopack configuration (moved from experimental)
  turbopack: {
    rules: {
      '*.svg': {
        loaders: ['@svgr/webpack'],
        as: '*.js',
      },
    },
    resolveAlias: {
      // Optimize heavy package resolution
      'react-grid-layout': 'react-grid-layout/build/css/styles.css',
      recharts: 'recharts/esm',
    },
  },

  // Server components optimization (moved from experimental)
  serverExternalPackages: ['canvas-confetti'],

  // Compiler optimizations
  compiler: {
    removeConsole:
      process.env.NODE_ENV === 'production' ? { exclude: ['error'] } : false,
  },

  // Output file tracing configuration - use relative path for portability
  // This resolves to the repository root dynamically
  outputFileTracingRoot:
    process.env.NODE_ENV === 'production' ? process.cwd() : undefined, // Let Next.js auto-detect in development
  generateEtags: false,

  // Custom redirects for authentication flows
  async redirects() {
    return [
      // Redirect root to dashboard for authenticated users
      {
        source: '/',
        destination: '/dashboard',
        permanent: false,
        has: [
          {
            type: 'cookie',
            key: 'msal.instance',
          },
        ],
      },
    ];
  },
};

// Bundle analyzer setup
if (process.env.ANALYZE === 'true') {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const withBundleAnalyzer = require('@next/bundle-analyzer')({
    enabled: true,
  });
  module.exports = withBundleAnalyzer(nextConfig);
} else {
  module.exports = nextConfig;
}
