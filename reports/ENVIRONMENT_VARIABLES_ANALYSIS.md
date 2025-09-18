# Environment Variables Analysis Report

## Executive Summary

This analysis resolves discrepancies between QUICKSTART.md and .env.local.example regarding environment variables for the MCP Portal project. Based on actual code usage, I've identified 47 unique environment variables across frontend and backend components.

## Key Findings

### 1. **ENCRYPTION_KEY** - NOT Used in Code

- **Status**: Listed in .env.local.example but NOT referenced in any code
- **Action**: Remove from frontend configuration (backend-only if needed)

### 2. **Database/Redis URLs** - Frontend Exposure Issue

- **Status**: DATABASE_URL and REDIS_URL defined in frontend env.mjs but should be server-side only
- **Action**: Remove from frontend configuration (security risk)

### 3. **Missing Variables**

- **API_BASE_URL**: Referenced in QUICKSTART.md but not used in code
- **MCP_PORTAL_ENCRYPTION_KEY**: Backend may need encryption key with different naming

### 4. **Naming Inconsistencies**

- **NEXT_PUBLIC_API_URL** vs **API_BASE_URL**: Use NEXT_PUBLIC_API_URL (actually used in code)
- **Azure variables**: Consistent naming between server/client variants

## Categorized Environment Variables

### Frontend Required (NEXT*PUBLIC*\*)

These MUST be in .env.local for frontend operation:

1. **NEXT_PUBLIC_API_URL** - Backend API URL
2. **NEXT_PUBLIC_WS_URL** - WebSocket URL
3. **NEXT_PUBLIC_AZURE_REDIRECT_URI** - OAuth redirect
4. **NEXT_PUBLIC_AZURE_POST_LOGOUT_URI** - Post-logout redirect
5. **NEXT_PUBLIC_AZURE_AUTHORITY** - Azure AD authority (optional, constructed from tenant)
6. **NEXT_PUBLIC_AZURE_SCOPES** - OAuth scopes

   ### Frontend Optional (NEXT*PUBLIC*\*)

   These have defaults but can be overridden:

7. **NEXT_PUBLIC_DEBUG** - Debug logging (default: false)
8. **NEXT_PUBLIC_API_TIMEOUT** - API timeout (default: 30000ms)
9. **NEXT_PUBLIC_WS_RECONNECT_INTERVAL** - WebSocket reconnect (default: 5000ms)
10. **NEXT_PUBLIC_ENABLE_WEBSOCKET** - Enable WebSocket (default: true)
11. **NEXT_PUBLIC_ENABLE_SSE** - Enable Server-Sent Events (default: true)
12. **NEXT_PUBLIC_ENABLE_ADMIN** - Enable admin features (default: true)
13. **NEXT_PUBLIC_ENABLE_BULK_OPS** - Enable bulk operations (default: true)
14. **NEXT_PUBLIC_TOKEN_STORAGE** - Token storage method (default: localStorage)
15. **NEXT_PUBLIC_SESSION_TIMEOUT** - Session timeout (default: 60 minutes)
16. **NEXT_PUBLIC_ENABLE_CSRF** - CSRF protection (default: true)
17. **NEXT_PUBLIC_DEFAULT_THEME** - UI theme (default: system)
18. **NEXT_PUBLIC_DEFAULT_PAGE_SIZE** - Pagination (default: 20)
19. **NEXT_PUBLIC_STATUS_REFRESH_INTERVAL** - Status refresh (default: 10s)
20. **NEXT_PUBLIC_SENTRY_DSN** - Error reporting (optional)
21. **NEXT_PUBLIC_ENABLE_ERROR_REPORTING** - Enable error reporting (default: false)
22. **NEXT_PUBLIC_LOG_LEVEL** - Frontend log level (used in logger.ts)

    ### Server-Side Required

    These should NOT be in frontend .env.local (security sensitive):

23. **AZURE_TENANT_ID** - Azure AD tenant ID
24. **AZURE_CLIENT_ID** - Azure AD client ID
25. **AZURE_CLIENT_SECRET** - Azure AD client secret
26. **JWT_SECRET** - JWT signing key
27. **NODE_ENV** - Environment (development/production)

    ### Server-Side Optional

    These have defaults in backend config:

28. **SESSION_COOKIE_NAME** - Session cookie name (default: mcp-portal-session)
29. **SESSION_COOKIE_SECURE** - Cookie secure flag (default: false)
30. **SESSION_COOKIE_HTTPONLY** - Cookie HTTP only (default: true)
31. **SESSION_COOKIE_SAMESITE** - Cookie SameSite (default: lax)
32. **DATABASE_URL** - PostgreSQL connection (backend only)
33. **REDIS_URL** - Redis connection (backend only)

    ### Backend-Specific (MCP*PORTAL* prefix)

    Used by Go backend with Viper configuration:

34. **MCP_PORTAL_ENV** - Environment name
35. **MCP_PORTAL_DATABASE_PASSWORD** - Database password override
36. **MCP_PORTAL_REDIS_PASSWORD** - Redis password override
37. **MCP_PORTAL_AZURE_CLIENT_SECRET** - Azure secret override
38. **MCP_PORTAL_JWT_SIGNING_KEY** - JWT key override

    ### Build/Development Only

39. **SKIP_ENV_VALIDATION** - Skip T3 env validation
40. **ANALYZE** - Bundle analyzer (Next.js)
41. **CUSTOM_KEY** - Next.js custom env (next.config.js)
42. **SITE_URL** - Sitemap generation
43. **SENTRY_DSN** - Server-side Sentry
44. **APP_ENV** - Application environment
45. **APP_VERSION** - Application version
46. **NEXT_PUBLIC_APP_VERSION** - Client-side version
47. **SERVER_NAME** - Server name for Sentry

## Security Issues Identified

### 🚨 Critical Issues

1. **DATABASE_URL in Frontend**: Exposes database credentials to client
2. **REDIS_URL in Frontend**: Exposes Redis credentials to client
3. **ENCRYPTION_KEY in Frontend**: Listed but unused, potential security risk

### 🟡 Configuration Issues

1. **API_BASE_URL**: Documented in QUICKSTART.md but not used in code
2. **Inconsistent Azure Variables**: Some use NEXT*PUBLIC* others don't

## Resolutions

### 1. Corrected .env.local.example

Remove security-sensitive and unused variables, keep only client-safe variables.

### 2. Backend Environment Variables

These should be set at the server/container level, not in frontend .env.local.

### 3. Documentation Updates

Update QUICKSTART.md to reflect actual variable usage and security practices.

## Variable Usage Evidence

### Frontend Code References

- **src/env.mjs**: Defines and validates all environment variables
- **src/services/auth.service.ts**: Uses NEXT_PUBLIC_API_URL
- **src/lib/api-client.ts**: Uses NEXT_PUBLIC_API_URL
- **src/lib/logger.ts**: Uses NEXT_PUBLIC_LOG_LEVEL, NODE_ENV
- **next.config.js**: Uses NEXT_PUBLIC_API_URL for API proxy

### Backend Code References

- **config/config.go**: Uses MCP*PORTAL* prefixed variables with Viper
- **config/env.go**: Environment helper functions with MCP_PORTAL_ENV

### Missing from Code

- **ENCRYPTION_KEY**: Listed in .env.local.example but not found in any code
- **API_BASE_URL**: In QUICKSTART.md but NEXT_PUBLIC_API_URL is used instead

## Recommendations

1. **Remove from Frontend .env.local**:

   - DATABASE_URL
   - REDIS_URL
   - ENCRYPTION_KEY (unless backend needs it)
   - All server-side Azure variables

2. **Frontend Focus**: Only NEXT*PUBLIC* variables that are actually used by client code

3. **Security**: Keep sensitive variables (secrets, passwords, keys) server-side only

4. **Documentation**: Update QUICKSTART.md to match actual implementation

This analysis provides the definitive source of truth for environment variable configuration based on actual code usage rather than documentation assumptions.
