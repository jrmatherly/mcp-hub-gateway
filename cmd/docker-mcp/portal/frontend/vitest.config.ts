/// <reference types="vitest" />
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import { resolve } from 'path';

export default defineConfig({
  plugins: [
    react({
      // Support for React 19 and Next.js 15
      jsxRuntime: 'automatic',
    }),
  ],
  test: {
    // Test environment
    environment: 'jsdom',

    // Setup files
    setupFiles: ['./tests/setup.ts'],

    // Global test patterns
    include: [
      'src/**/*.{test,spec}.{js,mjs,cjs,ts,mts,cts,jsx,tsx}',
      'tests/**/*.{test,spec}.{js,mjs,cjs,ts,mts,cts,jsx,tsx}',
    ],
    exclude: [
      'node_modules/**',
      'dist/**',
      '.next/**',
      'out/**',
      'build/**',
      'coverage/**',
    ],

    // Global test configuration
    globals: true,

    // Coverage configuration
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html', 'lcov'],
      reportsDirectory: './coverage',
      exclude: [
        'node_modules/**',
        'tests/**',
        'src/**/*.d.ts',
        'src/**/*.config.{ts,js}',
        'src/**/*.stories.{ts,tsx,js,jsx}',
        'src/types/**',
        'src/config/**',
        'src/env.mjs',
        '**/*.d.ts',
        '**/*.config.{ts,js}',
        '**/{test,tests,spec,specs}/**',
        '**/coverage/**',
        '.next/**',
        'out/**',
        'dist/**',
        'build/**',
      ],
      all: true,
      skipFull: false,
      thresholds: {
        global: {
          branches: 80,
          functions: 80,
          lines: 80,
          statements: 80,
        },
      },
    },

    // Test timeout
    testTimeout: 10000,
    hookTimeout: 10000,
    teardownTimeout: 5000,

    // Pool configuration for parallel tests
    pool: 'forks',
    poolOptions: {
      forks: {
        singleFork: false,
        minForks: 1,
        maxForks: 4,
      },
    },

    // Reporter configuration
    reporters: ['verbose', 'junit'],
    outputFile: {
      junit: './coverage/junit.xml',
    },

    // Watch configuration
    watch: false,

    // Retry configuration
    retry: 2,

    // Concurrent test execution
    sequence: {
      concurrent: true,
      shuffle: false,
      hooks: 'stack',
    },
  },

  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
      '@/components': resolve(__dirname, './src/components'),
      '@/lib': resolve(__dirname, './src/lib'),
      '@/types': resolve(__dirname, './src/types'),
      '@/hooks': resolve(__dirname, './src/hooks'),
      '@/utils': resolve(__dirname, './src/utils'),
      '@/services': resolve(__dirname, './src/services'),
      '@/stores': resolve(__dirname, './src/stores'),
      '@/app': resolve(__dirname, './src/app'),
      '@/config': resolve(__dirname, './src/config'),
      '@/contexts': resolve(__dirname, './src/contexts'),
      '@/providers': resolve(__dirname, './src/providers'),
    },
  },

  // Define global variables for testing
  define: {
    'process.env.NODE_ENV': JSON.stringify('test'),
    'process.env.NEXT_PUBLIC_API_URL': JSON.stringify('http://localhost:8080'),
    // Add other environment variables as needed
  },

  // CSS handling
  css: {
    modules: {
      scopeBehaviour: 'local',
    },
  },

  // Optimize deps for testing
  optimizeDeps: {
    include: [
      '@testing-library/react',
      '@testing-library/jest-dom',
      '@testing-library/user-event',
      'jsdom',
      'msw',
    ],
  },
});
