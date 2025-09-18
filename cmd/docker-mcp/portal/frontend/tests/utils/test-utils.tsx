// Custom render utilities for testing React components
import React, { ReactElement } from 'react';
import { render, RenderOptions } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ThemeProvider } from 'next-themes';

// Custom render function that includes providers
const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        staleTime: 0,
        gcTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });

interface AllTheProvidersProps {
  children: React.ReactNode;
}

const AllTheProviders = ({ children }: AllTheProvidersProps) => {
  const queryClient = createTestQueryClient();

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider
        attribute="class"
        defaultTheme="light"
        enableSystem={false}
      >
        {children}
      </ThemeProvider>
    </QueryClientProvider>
  );
};

const customRender = (
  ui: ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) => render(ui, { wrapper: AllTheProviders, ...options });

// Re-export everything
export * from '@testing-library/react';
export { customRender as render };

// Additional test utilities
export const createWrapper = ({
  queryClient = createTestQueryClient(),
  theme = 'light',
} = {}) => {
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider
        attribute="class"
        defaultTheme={theme}
        enableSystem={false}
      >
        {children}
      </ThemeProvider>
    </QueryClientProvider>
  );
};

// Helper to wait for React Query to settle
export const waitForQueryToSettle = async () => {
  await new Promise(resolve => setTimeout(resolve, 0));
};

// Re-export user event properly
export { default as userEvent } from '@testing-library/user-event';

// Common test data factories
export const createMockServer = (overrides = {}) => ({
  id: 'test-server-1',
  name: 'Test Server',
  description: 'A test MCP server',
  status: 'running' as const,
  enabled: true,
  version: '1.0.0',
  health: 'healthy' as const,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  ...overrides,
});

export const createMockConfig = (overrides = {}) => ({
  gateway: {
    port: 8080,
    host: 'localhost',
    transport: 'stdio' as const,
  },
  servers: {},
  ...overrides,
});

// Assertion helpers
export const expectElementToBeVisible = (element: HTMLElement) => {
  expect(element).toBeInTheDocument();
  expect(element).toBeVisible();
};

export const expectElementToHaveText = (
  element: HTMLElement,
  text: string | RegExp
) => {
  expect(element).toBeInTheDocument();
  expect(element).toHaveTextContent(text);
};

// Mock component factory
export const createMockComponent = (name: string) => {
  const MockComponent = ({
    children,
    ...props
  }: {
    children?: React.ReactNode;
    [key: string]: unknown;
  }) => (
    <div data-testid={`mock-${name.toLowerCase()}`} {...props}>
      {children}
    </div>
  );
  MockComponent.displayName = `Mock${name}`;
  return MockComponent;
};
