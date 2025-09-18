// Test setup file for Vitest
import '@testing-library/jest-dom';
import { cleanup } from '@testing-library/react';
import { server } from './mocks/server';

// Mock next/navigation for App Router
vi.mock('next/navigation', () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    prefetch: vi.fn(),
    back: vi.fn(),
    forward: vi.fn(),
    refresh: vi.fn(),
  }),
  useSearchParams: () => new URLSearchParams(),
  usePathname: () => '/',
  useParams: () => ({}),
  notFound: vi.fn(),
  redirect: vi.fn(),
}));

// Mock next/router for Pages Router compatibility
vi.mock('next/router', async () => {
  const { default: mockRouter } = await import('next-router-mock');
  return mockRouter;
});

// Mock next/image for testing
vi.mock('next/image', async () => {
  const React = await import('react');
  return {
    __esModule: true,
    default: (props: React.ImgHTMLAttributes<HTMLImageElement>) => {
      return React.createElement('img', { ...props, alt: props.alt || '' });
    },
  };
});

// Mock next/link
vi.mock('next/link', async () => {
  const React = await import('react');
  return {
    __esModule: true,
    default: ({
      children,
      href,
      ...props
    }: React.AnchorHTMLAttributes<HTMLAnchorElement> & {
      href: string;
      children: React.ReactNode;
    }) => {
      return React.createElement('a', { href, ...props }, children);
    },
  };
});

// Mock Azure MSAL for authentication testing
vi.mock('@azure/msal-react', () => ({
  MsalProvider: ({ children }: { children: React.ReactNode }) => children,
  useMsal: () => ({
    instance: {
      acquireTokenSilent: vi
        .fn()
        .mockResolvedValue({ accessToken: 'mock-token' }),
      loginRedirect: vi.fn(),
      logout: vi.fn(),
      getAllAccounts: vi.fn().mockReturnValue([]),
    },
    accounts: [],
    inProgress: 'none',
  }),
  useIsAuthenticated: () => false,
  AuthenticatedTemplate: ({
    children: _children,
  }: {
    children: React.ReactNode;
  }) => null,
  UnauthenticatedTemplate: ({ children }: { children: React.ReactNode }) =>
    children,
}));

// Mock Azure MSAL browser
vi.mock('@azure/msal-browser', () => ({
  PublicClientApplication: vi.fn().mockImplementation(() => ({
    initialize: vi.fn().mockResolvedValue(undefined),
    acquireTokenSilent: vi
      .fn()
      .mockResolvedValue({ accessToken: 'mock-token' }),
    loginRedirect: vi.fn(),
    logout: vi.fn(),
    getAllAccounts: vi.fn().mockReturnValue([]),
  })),
  LogLevel: {
    Error: 0,
    Warning: 1,
    Info: 2,
    Verbose: 3,
  },
  InteractionType: {
    Redirect: 'redirect',
    Popup: 'popup',
  },
}));

// Mock WebSocket for real-time functionality
const mockWebSocket = vi.fn().mockImplementation(() => ({
  addEventListener: vi.fn(),
  removeEventListener: vi.fn(),
  send: vi.fn(),
  close: vi.fn(),
  readyState: 1, // OPEN
}));

Object.defineProperty(global, 'WebSocket', {
  value: mockWebSocket,
  writable: true,
});

// Mock IntersectionObserver
const mockIntersectionObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
}));

Object.defineProperty(global, 'IntersectionObserver', {
  value: mockIntersectionObserver,
  writable: true,
});

// Mock ResizeObserver
const mockResizeObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
}));

Object.defineProperty(global, 'ResizeObserver', {
  value: mockResizeObserver,
  writable: true,
});

// Mock matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(), // deprecated
    removeListener: vi.fn(), // deprecated
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
});

// Mock scrollTo
Object.defineProperty(window, 'scrollTo', {
  value: vi.fn(),
  writable: true,
});

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
  length: 0,
  key: vi.fn(),
};

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
  writable: true,
});

// Mock sessionStorage
Object.defineProperty(window, 'sessionStorage', {
  value: localStorageMock,
  writable: true,
});

// Mock URL.createObjectURL
Object.defineProperty(global.URL, 'createObjectURL', {
  value: vi.fn(() => 'mock-object-url'),
  writable: true,
});

Object.defineProperty(global.URL, 'revokeObjectURL', {
  value: vi.fn(),
  writable: true,
});

// Mock console.warn and console.error to reduce noise in tests
const originalWarn = console.warn;
const originalError = console.error;

// Environment variables for testing
// Note: In Vitest, process.env.NODE_ENV is already set to 'test'
// We just need to set additional environment variables
process.env.NEXT_PUBLIC_API_URL = 'http://localhost:8080';

// MSW server setup
beforeAll(() => {
  // Start MSW server
  server.listen({
    onUnhandledRequest: 'warn',
  });

  // Suppress console warnings during tests unless explicitly testing them
  console.warn = vi.fn();
  console.error = vi.fn();
});

afterEach(() => {
  // Clean up DOM after each test
  cleanup();

  // Reset MSW handlers
  server.resetHandlers();

  // Clear all mocks
  vi.clearAllMocks();

  // Reset localStorage/sessionStorage
  localStorageMock.clear();
});

afterAll(() => {
  // Stop MSW server
  server.close();

  // Restore console methods
  console.warn = originalWarn;
  console.error = originalError;

  // Restore all mocks
  vi.restoreAllMocks();
});

// Custom test utilities
export const createMockRouter = (overrides = {}) => ({
  push: vi.fn(),
  replace: vi.fn(),
  prefetch: vi.fn(),
  back: vi.fn(),
  forward: vi.fn(),
  refresh: vi.fn(),
  pathname: '/',
  route: '/',
  query: {},
  asPath: '/',
  ...overrides,
});

export const createMockUser = (overrides = {}) => ({
  id: 'test-user-id',
  email: 'test@example.com',
  name: 'Test User',
  picture: 'https://example.com/avatar.jpg',
  ...overrides,
});

// Export commonly used testing utilities
export * from '@testing-library/react';
export { default as userEvent } from '@testing-library/user-event';
