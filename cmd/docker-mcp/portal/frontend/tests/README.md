# Testing Setup

This directory contains the testing configuration and test files for the MCP Portal frontend.

## Overview

The testing setup uses:

- **Vitest** - Fast unit test runner built for Vite
- **@testing-library/react** - React testing utilities
- **@testing-library/jest-dom** - Custom DOM matchers
- **@testing-library/user-event** - User interaction simulation
- **MSW (Mock Service Worker)** - API mocking
- **jsdom** - DOM environment for Node.js

## Directory Structure

```
tests/
├── README.md                 # This file
├── setup.ts                  # Global test setup
├── utils/
│   ├── test-utils.tsx        # Custom render utilities
│   └── setup.test.ts         # Setup verification tests
├── mocks/
│   ├── server.ts             # MSW server setup
│   └── handlers.ts           # API mock handlers
├── components/               # Component tests
│   └── ServerCard.test.tsx   # Example component test
├── hooks/                    # Hook tests
│   └── useServers.test.tsx   # Example hook test
├── app/                      # Next.js App Router tests
│   └── dashboard.test.tsx    # Example page test
└── e2e/                      # End-to-end tests (future)
```

## Running Tests

### Basic Commands

```bash
# Run all tests
npm test

# Run tests once
npm run test:run

# Run tests with coverage
npm run test:coverage

# Run tests with UI
npm run test:ui

# Watch mode
npm run test:watch
```

### Specific Test Types

```bash
# Unit tests (components, hooks, utils)
npm run test:unit

# Integration tests (pages, app router)
npm run test:integration

# E2E tests
npm run test:e2e

# CI mode (with coverage and JUnit output)
npm run test:ci
```

## Configuration

### Main Config (`vitest.config.ts`)

- **Environment**: jsdom for DOM simulation
- **Setup**: Global setup file loaded before all tests
- **Path Aliases**: Matches TypeScript paths (@/\* imports)
- **Coverage**: V8 provider with 80% thresholds
- **Environment Variables**: Test-specific env vars

### Setup File (`tests/setup.ts`)

Global setup includes:

- Testing library matchers
- Next.js mocking (navigation, image, link)
- Azure MSAL mocking for auth
- WebSocket, IntersectionObserver, ResizeObserver mocks
- LocalStorage/SessionStorage mocks
- MSW server lifecycle

## Writing Tests

### Component Tests

```typescript
import { render, screen } from '../utils/test-utils'
import { MyComponent } from '@/components/MyComponent'

describe('MyComponent', () => {
  it('renders correctly', () => {
    render(<MyComponent />)
    expect(screen.getByText('Hello')).toBeInTheDocument()
  })
})
```

### Hook Tests

```typescript
import { renderHook } from '@testing-library/react';
import { createWrapper } from '../utils/test-utils';
import { useMyHook } from '@/hooks/useMyHook';

describe('useMyHook', () => {
  it('returns expected data', async () => {
    const { result } = renderHook(() => useMyHook(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data).toBeDefined();
    });
  });
});
```

### API Mocking

MSW handlers are defined in `mocks/handlers.ts`:

```typescript
import { http, HttpResponse } from 'msw';

export const handlers = [
  http.get('/api/servers', () => {
    return HttpResponse.json({ servers: [] });
  }),
];
```

Use `server.use()` to add temporary handlers in tests:

```typescript
import { server } from '../mocks/server';
import { createErrorHandler } from '../mocks/handlers';

it('handles API errors', async () => {
  server.use(createErrorHandler('/api/servers', 500, 'Server Error'));

  // Test error handling
});
```

### Next.js Features

#### App Router Navigation

```typescript
import { useRouter } from 'next/navigation';

// Mocked automatically in setup.ts
const mockRouter = useRouter();
mockRouter.push('/dashboard');
```

#### Search Params

```typescript
import { useSearchParams } from 'next/navigation';

// Mock in test
vi.mocked(useSearchParams).mockReturnValue({
  get: vi.fn().mockReturnValue('enabled'),
  // ... other methods
});
```

## Best Practices

### Test Organization

1. **Arrange**: Set up test data and mocks
2. **Act**: Trigger the behavior being tested
3. **Assert**: Verify the expected outcome

### Naming Conventions

- Test files: `*.test.ts` or `*.test.tsx`
- Describe blocks: Feature or component name
- Test cases: "should [behavior] when [condition]"

### Async Testing

```typescript
// Wait for elements to appear
await waitFor(() => {
  expect(screen.getByText('Loaded')).toBeInTheDocument();
});

// Wait for user events
await user.click(button);
```

### Accessibility Testing

```typescript
// Check roles
expect(screen.getByRole('button')).toBeInTheDocument();

// Check aria attributes
expect(button).toHaveAttribute('aria-label', 'Close');

// Check focus management
expect(input).toHaveFocus();
```

### Error Testing

```typescript
// Test error boundaries
const spy = vi.spyOn(console, 'error').mockImplementation(() => {});
// ... trigger error
expect(spy).toHaveBeenCalled();
spy.mockRestore();
```

## Debugging Tests

### VS Code Integration

Install the Vitest extension for VS Code to run tests inline.

### Debug Mode

```bash
# Run with debug output
npm test -- --reporter=verbose

# Run specific test file
npm test -- ServerCard.test.tsx

# Run with filter
npm test -- --grep "renders correctly"
```

### Coverage Reports

Coverage reports are generated in `./coverage/`:

- `index.html` - Interactive HTML report
- `lcov.info` - LCOV format for CI/CD
- `coverage-final.json` - JSON format

## CI/CD Integration

The `test:ci` script generates:

- Coverage reports in multiple formats
- JUnit XML for test result reporting
- Exit codes for build pipeline integration

## Troubleshooting

### Common Issues

1. **Import Errors**: Check path aliases in `vitest.config.ts`
2. **Module Not Found**: Ensure dependencies are installed
3. **Timeout Errors**: Increase timeout in test configuration
4. **Mock Issues**: Verify mocks are set up in `setup.ts`

### Environment Variables

Test environment variables are defined in `vitest.config.ts`:

```typescript
define: {
  'process.env.NODE_ENV': JSON.stringify('test'),
  'process.env.NEXT_PUBLIC_API_URL': JSON.stringify('http://localhost:8080'),
}
```

Add new environment variables as needed for your tests.
