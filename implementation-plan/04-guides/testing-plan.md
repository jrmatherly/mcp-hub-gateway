# Testing Plan

## Overview

Comprehensive testing strategy for the MCP Portal covering unit, integration, end-to-end, performance, and security testing.

## Testing Strategy

### Test Coverage Goals

- **Unit Tests**: 80% code coverage minimum
- **Integration Tests**: All critical paths
- **E2E Tests**: All user workflows
- **Performance Tests**: Meeting all SLA requirements
- **Security Tests**: Zero critical vulnerabilities

### Testing Pyramid

```
         /\
        /E2E\
       /------\
      /  API   \
     /----------\
    /Integration \
   /--------------\
  /   Unit Tests   \
 /------------------\
```

## Unit Testing

### Backend (Go)

#### Test Structure

```go
// server_test.go
package portal

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestEnableServer(t *testing.T) {
    t.Run("successful enable", func(t *testing.T) {
        // Arrange
        mockDocker := new(MockDockerClient)
        mockDB := new(MockDatabase)
        service := NewPortalService(mockDocker, mockDB)

        // Act
        err := service.EnableServer("user123", "server456")

        // Assert
        assert.NoError(t, err)
        mockDocker.AssertExpectations(t)
        mockDB.AssertExpectations(t)
    })
}
```

#### Key Test Areas

```yaml
auth_package:
  - JWT token generation
  - Token validation
  - Azure AD integration
  - Role-based access control

database_package:
  - Connection pool management
  - Query execution
  - Transaction handling
  - Encryption/decryption

docker_package:
  - Container lifecycle
  - State management
  - Resource limits
  - Error handling

handlers_package:
  - Request validation
  - Response formatting
  - Error responses
  - Middleware chain
```

### Frontend (TypeScript/React)

#### Test Structure

```typescript
// ServerCard.test.tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ServerCard } from "./ServerCard";

describe("ServerCard", () => {
  it("should toggle server state", async () => {
    // Arrange
    const mockToggle = jest.fn();
    const server = {
      id: "123",
      name: "GitHub",
      enabled: false,
    };

    // Act
    render(
      <QueryClientProvider client={new QueryClient()}>
        <ServerCard server={server} onToggle={mockToggle} />
      </QueryClientProvider>
    );

    fireEvent.click(screen.getByRole("switch"));

    // Assert
    expect(mockToggle).toHaveBeenCalledWith("123", true);
  });
});
```

#### Component Testing

```yaml
components:
  - ServerCard
  - ServerGrid
  - BulkActions
  - ConfigExport
  - AdminPanel
  - AuditLog

hooks:
  - useAuth
  - useServers
  - useWebSocket
  - useConfig

utilities:
  - API client
  - Encryption helpers
  - Validation functions
```

## Integration Testing

### Database Integration

```go
func TestDatabaseIntegration(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)

    t.Run("user isolation with RLS", func(t *testing.T) {
        // Create two users
        user1 := createTestUser(t, db, "user1@test.com")
        user2 := createTestUser(t, db, "user2@test.com")

        // Create configs for user1
        config1 := createServerConfig(t, db, user1.ID, "server1")

        // Set context to user2
        setUserContext(t, db, user2.ID)

        // Try to read user1's config
        configs := queryUserConfigs(t, db)

        // Should not see user1's configs
        assert.Empty(t, configs)
    })
}
```

### API Integration

```typescript
describe("API Integration", () => {
  let apiClient: APIClient;

  beforeAll(async () => {
    apiClient = new APIClient(TEST_BASE_URL);
    await apiClient.authenticate(TEST_CREDENTIALS);
  });

  test("server lifecycle", async () => {
    // Enable server
    const enableResult = await apiClient.enableServer("server-123");
    expect(enableResult.status).toBe("accepted");

    // Wait for container to start
    await waitFor(async () => {
      const status = await apiClient.getServerStatus("server-123");
      return status.state === "running";
    });

    // Disable server
    const disableResult = await apiClient.disableServer("server-123");
    expect(disableResult.status).toBe("accepted");
  });
});
```

### Docker Integration

```go
func TestDockerIntegration(t *testing.T) {
    client := newDockerClient(t)

    t.Run("container lifecycle", func(t *testing.T) {
        // Create container
        containerID := createContainer(t, client, "test-image")

        // Start container
        err := client.StartContainer(containerID)
        assert.NoError(t, err)

        // Verify running
        status := getContainerStatus(t, client, containerID)
        assert.Equal(t, "running", status)

        // Stop container
        err = client.StopContainer(containerID)
        assert.NoError(t, err)

        // Cleanup
        client.RemoveContainer(containerID)
    })
}
```

## End-to-End Testing

### Test Scenarios

```typescript
// e2e/server-management.spec.ts
import { test, expect } from "@playwright/test";

test.describe("Server Management", () => {
  test("complete user workflow", async ({ page }) => {
    // Login
    await page.goto("/login");
    await page.click('button:has-text("Login with Azure")');
    await handleAzureLogin(page);

    // Navigate to dashboard
    await expect(page).toHaveURL("/dashboard");

    // Enable a server
    const serverCard = page.locator('[data-testid="server-github"]');
    await serverCard.locator('button[role="switch"]').click();

    // Wait for container to start
    await expect(serverCard.locator('[data-status="running"]')).toBeVisible({
      timeout: 10000,
    });

    // Bulk operation
    await page.click('[data-testid="select-all"]');
    await page.click('[data-testid="bulk-enable"]');

    // Verify all enabled
    const switches = page.locator('button[role="switch"][aria-checked="true"]');
    await expect(switches).toHaveCount(5);

    // Export configuration
    await page.click('[data-testid="export-config"]');
    const download = await page.waitForEvent("download");
    expect(download.suggestedFilename()).toBe("mcp-config.json");
  });
});
```

### Critical User Paths

```yaml
authentication:
  - Azure AD login
  - Token refresh
  - Logout
  - Session timeout

server_management:
  - List servers
  - Enable server
  - Disable server
  - View server details
  - View server logs

bulk_operations:
  - Select multiple servers
  - Bulk enable
  - Bulk disable
  - Export configuration
  - Import configuration

admin_functions:
  - View all users
  - Change user roles
  - View audit logs
  - View system metrics
```

## Performance Testing

### Load Testing with k6

```javascript
// load-test.js
import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  stages: [
    { duration: "2m", target: 100 }, // Ramp up
    { duration: "5m", target: 100 }, // Stay at 100 users
    { duration: "2m", target: 200 }, // Ramp up
    { duration: "5m", target: 200 }, // Stay at 200 users
    { duration: "2m", target: 0 }, // Ramp down
  ],
  thresholds: {
    http_req_duration: ["p(95)<500"], // 95% of requests under 500ms
    http_req_failed: ["rate<0.1"], // Error rate under 10%
  },
};

export default function () {
  const token = login();

  // List servers
  const listResponse = http.get(`${BASE_URL}/api/v1/servers`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  check(listResponse, {
    "list servers status 200": (r) => r.status === 200,
    "list servers fast": (r) => r.timings.duration < 200,
  });

  sleep(1);

  // Toggle server
  const toggleResponse = http.post(
    `${BASE_URL}/api/v1/servers/${SERVER_ID}/enable`,
    {},
    { headers: { Authorization: `Bearer ${token}` } }
  );
  check(toggleResponse, {
    "toggle server accepted": (r) => r.status === 202,
  });

  sleep(2);
}
```

### Performance Metrics

```yaml
api_performance:
  - Response time (p50, p95, p99)
  - Throughput (requests/second)
  - Error rate
  - Concurrent users

database_performance:
  - Query execution time
  - Connection pool utilization
  - Lock contention
  - Index effectiveness

frontend_performance:
  - First Contentful Paint (FCP)
  - Time to Interactive (TTI)
  - Largest Contentful Paint (LCP)
  - Cumulative Layout Shift (CLS)
  - Bundle size
```

## Security Testing

### Security Scanning

```bash
# OWASP ZAP scan
docker run -t owasp/zap2docker-stable zap-baseline.py \
  -t https://mcp-portal.company.com \
  -r security-report.html

# Dependency scanning
npm audit
go mod audit

# Container scanning
docker scan mcp-portal:latest

# Secret scanning
gitleaks detect --source . -v
```

### Security Test Cases

```yaml
authentication:
  - SQL injection attempts
  - XSS in user inputs
  - CSRF token validation
  - JWT token tampering
  - Session fixation

authorization:
  - Privilege escalation
  - IDOR vulnerabilities
  - Role bypass attempts
  - API access without auth

data_protection:
  - Encryption verification
  - Secure transmission (TLS)
  - Input validation
  - Output encoding
```

### Penetration Testing Checklist

```markdown
- [ ] Authentication bypass attempts
- [ ] Authorization flaws
- [ ] Injection vulnerabilities (SQL, NoSQL, Command)
- [ ] XSS (Stored, Reflected, DOM-based)
- [ ] CSRF vulnerabilities
- [ ] Insecure direct object references
- [ ] Security misconfigurations
- [ ] Sensitive data exposure
- [ ] Broken access control
- [ ] Insufficient logging
```

## Accessibility Testing

### Automated Testing

```javascript
// accessibility.test.js
const { AxePuppeteer } = require("@axe-core/puppeteer");
const puppeteer = require("puppeteer");

describe("Accessibility", () => {
  it("should have no accessibility violations", async () => {
    const browser = await puppeteer.launch();
    const page = await browser.newPage();
    await page.goto("http://localhost:3000");

    const results = await new AxePuppeteer(page).analyze();
    expect(results.violations).toHaveLength(0);

    await browser.close();
  });
});
```

### Manual Testing Checklist

```markdown
- [ ] Keyboard navigation
- [ ] Screen reader compatibility
- [ ] Focus indicators
- [ ] Color contrast (WCAG AA)
- [ ] Form labels and errors
- [ ] Alt text for images
- [ ] ARIA landmarks
- [ ] Heading hierarchy
```

## Test Data Management

### Test Data Strategy

```yaml
approach: "Builder Pattern with Factories"

factories:
  - UserFactory
  - ServerFactory
  - ConfigFactory
  - AuditLogFactory

data_sets:
  minimal:
    users: 2
    servers: 5
    configs: 10

  standard:
    users: 10
    servers: 20
    configs: 100

  stress:
    users: 1000
    servers: 50
    configs: 5000

cleanup: "After each test suite"
```

### Test Database Setup

```sql
-- test-setup.sql
CREATE DATABASE mcp_portal_test;

-- Create test user with specific permissions
CREATE USER test_user WITH PASSWORD 'test_password';
GRANT ALL PRIVILEGES ON DATABASE mcp_portal_test TO test_user;

-- Apply migrations
\i migrations/001_initial_schema.sql
\i migrations/002_enable_rls.sql
\i migrations/003_test_data.sql
```

## Continuous Integration

### GitHub Actions Workflow

```yaml
name: Test Suite

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: "1.24"
      - run: go test ./... -cover

  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:17-alpine
      redis:
        image: redis:8-alpine
    steps:
      - uses: actions/checkout@v3
      - run: make integration-test

  e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
      - run: npm ci
      - run: npx playwright install
      - run: npm run test:e2e

  security-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - run: |
          npm audit
          go mod audit
          docker scan .
```

## Test Reporting

### Coverage Reports

```bash
# Go coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# JavaScript coverage
npm run test:coverage
```

### Test Results Dashboard

```yaml
metrics:
  - Total tests
  - Pass rate
  - Execution time
  - Coverage percentage
  - Flaky tests

reports:
  - JUnit XML
  - HTML reports
  - Coverage badges
  - Trend analysis
```

## Test Maintenance

### Test Review Checklist

```markdown
- [ ] Tests are readable and maintainable
- [ ] No hardcoded values
- [ ] Proper test isolation
- [ ] Meaningful assertions
- [ ] Good error messages
- [ ] No test interdependencies
- [ ] Appropriate use of mocks
- [ ] Test data cleanup
```

### Flaky Test Management

```yaml
identification:
  - Monitor test failure patterns
  - Track intermittent failures
  - Use test retry mechanisms

resolution:
  - Add proper waits/timeouts
  - Improve test isolation
  - Mock external dependencies
  - Use deterministic data

prevention:
  - Avoid time-based assertions
  - Use stable selectors
  - Proper async handling
  - Control test environment
```

## Testing Schedule

| Phase   | Week | Focus                     | Coverage Target |
| ------- | ---- | ------------------------- | --------------- |
| Phase 1 | 1-2  | Unit tests for foundation | 70%             |
| Phase 2 | 3-4  | Integration tests         | 80%             |
| Phase 3 | 5-6  | Frontend & E2E tests      | 85%             |
| Phase 4 | 7-8  | Performance & Security    | 90%             |

## Success Criteria

- All tests passing in CI/CD pipeline
- Code coverage > 80%
- No critical security vulnerabilities
- Performance SLAs met
- Accessibility score > 95
- Zero P1 bugs in production
