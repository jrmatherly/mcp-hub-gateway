# MCP Portal Authentication Architecture

## Comprehensive Security & Integration Documentation

**Document Version**: 1.0.0
**Last Updated**: September 19, 2025
**Classification**: SENSITIVE - Security Architecture
**Authors**: MCP Portal Development Team

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Azure AD Configuration](#azure-ad-configuration)
4. [Frontend Authentication (SPA)](#frontend-authentication-spa)
5. [Backend Token Validation](#backend-token-validation)
6. [MCP Protocol Integration](#mcp-protocol-integration)
7. [Security Implementation](#security-implementation)
8. [Secret Management](#secret-management)
9. [Token Flow Diagrams](#token-flow-diagrams)
10. [Configuration Guide](#configuration-guide)
11. [Troubleshooting](#troubleshooting)
12. [Security Considerations](#security-considerations)
13. [Future Roadmap](#future-roadmap)

---

## Executive Summary

The MCP Portal implements a **hybrid authentication architecture** using Azure Active Directory (Azure AD/Entra ID) as the primary identity provider. The system uses a **single Azure AD app registration with dual platform configuration** (Single-Page Application + Web) to support both browser-based authentication and backend API validation.

### Key Architecture Decisions

1. **Single App Registration**: One Azure AD app registration serves both frontend and backend
2. **Dual Platform Configuration**: SPA platform for frontend, Web platform for backend optional flows
3. **Token Validation**: Backend validates tokens using JWKS without requiring client secrets
4. **MCP Integration**: Bridge service pattern for Dynamic Client Registration (DCR) support

### Security Highlights

- **Zero Trust Model**: All tokens validated on every request
- **PKCE Flow**: Frontend uses Authorization Code Flow with PKCE (no implicit flow)
- **JWKS Validation**: Cryptographic token validation without secret exposure
- **Secret Isolation**: Client secrets never exposed to frontend

---

## Architecture Overview

### System Components

```mermaid
graph TB
    subgraph "Frontend (Browser)"
        A[Next.js App]
        B[MSAL.js 2.0]
        C[Session Storage]
    end

    subgraph "Azure AD"
        D[Authorization Server]
        E[Token Endpoint]
        F[JWKS Endpoint]
        G[Microsoft Graph]
    end

    subgraph "Backend (Go)"
        H[API Server]
        I[Token Validator]
        J[Session Manager]
        K[Redis Cache]
    end

    subgraph "MCP Tools"
        L[MCP Servers]
        M[DCR Bridge]
        N[Key Vault]
    end

    A --> B
    B --> D
    D --> E
    E --> B
    B --> C
    A --> H
    H --> I
    I --> F
    H --> J
    J --> K
    L --> M
    M --> N
    M --> D
```

### Authentication Flow

1. **User Authentication**: Browser → Azure AD → Token acquisition
2. **Token Storage**: Session storage in browser (secure, non-persistent)
3. **API Calls**: Frontend sends token in Authorization header
4. **Token Validation**: Backend validates using JWKS endpoint
5. **Session Management**: Redis-backed session state

---

## Azure AD Configuration

### App Registration Requirements

```yaml
Application Registration:
  Name: MCP-Portal
  Application ID: 1826685e-1eef-45c9-9b1a-8f03e16f4ae3
  Tenant ID: f505346f-75cf-458b-baeb-10708d41967d

  Platforms:
    Single-Page Application:
      Redirect URIs:
        - http://localhost:3000
        - http://localhost:3001
        - http://localhost:3000/auth/callback
        - https://portal.yourdomain.com (production)
        - https://portal.yourdomain.com/auth/callback (production)
      Settings:
        - Authorization Code Flow with PKCE: Enabled
        - Implicit Flow: Disabled (security best practice)

    Web Platform:
      Redirect URIs:
        - http://localhost:8080/auth/callback
        - https://api.yourdomain.com/auth/callback (production)
      Settings:
        - Client Secret: Generated (stored in Key Vault)
        - Certificate: Optional (for production)

  API Permissions:
    Microsoft Graph:
      - openid (delegated)
      - profile (delegated)
      - email (delegated)
      - User.Read (delegated)
      - User.Read.All (optional, for user photos)

  Token Configuration:
    - Access Token Version: v2.0
    - ID Token Version: v2.0
    - Refresh Token: Enabled
    - Access Token Lifetime: 1 hour (default)
    - Refresh Token Lifetime: 90 days (default)
```

### Critical Azure Portal Settings

⚠️ **IMPORTANT**: The following settings are REQUIRED for proper operation:

1. **Authentication Tab**:

   - ✅ Single-page application platform MUST be configured
   - ✅ Web platform SHOULD be configured (for backend flexibility)
   - ❌ Implicit grant flow MUST be disabled (security)

2. **Certificates & Secrets Tab**:

   - Client secret for Web platform (backend only)
   - Never expose to frontend code

3. **Token Configuration Tab**:
   - Optional claims can be added (groups, roles)
   - Ensure v2.0 tokens are configured

---

## Frontend Authentication (SPA)

### MSAL.js Configuration

```typescript
// src/config/msal.config.ts
import { Configuration, LogLevel } from "@azure/msal-browser";

export const msalConfig: Configuration = {
  auth: {
    clientId: process.env.NEXT_PUBLIC_AZURE_CLIENT_ID!,
    authority: `https://login.microsoftonline.com/${process.env.NEXT_PUBLIC_AZURE_TENANT_ID}/v2.0`,
    redirectUri:
      process.env.NEXT_PUBLIC_REDIRECT_URI || "http://localhost:3000",
    postLogoutRedirectUri:
      process.env.NEXT_PUBLIC_POST_LOGOUT_URI || "http://localhost:3000",
    navigateToLoginRequestUrl: false, // Prevent redirect loops
  },
  cache: {
    cacheLocation: "sessionStorage", // More secure than localStorage
    storeAuthStateInCookie: false, // Set to true for IE11 support
    secureCookies: true, // HTTPS only in production
  },
  system: {
    loggerOptions: {
      loggerCallback: (
        level: LogLevel,
        message: string,
        containsPii: boolean
      ) => {
        if (containsPii) return; // Never log PII
        console.log(message);
      },
      logLevel: LogLevel.Error,
      piiLoggingEnabled: false,
    },
  },
};

// Login request scopes
export const loginRequest = {
  scopes: ["openid", "profile", "email", "User.Read"],
  prompt: "select_account", // Always show account selection
};
```

### Authentication Provider Setup

```typescript
// src/providers/AuthProvider.tsx
import { MsalProvider } from "@azure/msal-react";
import { PublicClientApplication } from "@azure/msal-browser";
import { msalConfig } from "@/config/msal.config";

const msalInstance = new PublicClientApplication(msalConfig);

// Initialize MSAL
await msalInstance.initialize();

export function AuthProvider({ children }: { children: React.ReactNode }) {
  return <MsalProvider instance={msalInstance}>{children}</MsalProvider>;
}
```

### Token Acquisition Pattern

```typescript
// src/hooks/useAuth.ts
import { useMsal } from "@azure/msal-react";

export function useAuth() {
  const { instance, accounts } = useMsal();

  const getAccessToken = async (): Promise<string | null> => {
    if (accounts.length === 0) return null;

    try {
      // Silent token acquisition (uses refresh token if needed)
      const response = await instance.acquireTokenSilent({
        scopes: ["openid", "profile", "email"],
        account: accounts[0],
      });
      return response.accessToken;
    } catch (error) {
      // Fall back to interactive if silent fails
      const response = await instance.acquireTokenPopup({
        scopes: ["openid", "profile", "email"],
      });
      return response.accessToken;
    }
  };

  return { getAccessToken };
}
```

---

## Backend Token Validation

### Go Token Validator Implementation

```go
// auth/validator.go
package auth

import (
    "context"
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/lestrrat-go/jwx/jwk"
)

type TokenValidator struct {
    jwksURI    string
    issuer     string
    audience   string
    jwkSet     jwk.Set
    lastUpdate time.Time
}

func NewTokenValidator(tenantID, clientID string) *TokenValidator {
    return &TokenValidator{
        jwksURI:  fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/v2.0/keys", tenantID),
        issuer:   fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tenantID),
        audience: clientID,
    }
}

func (v *TokenValidator) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
    // Refresh JWKS if older than 24 hours
    if time.Since(v.lastUpdate) > 24*time.Hour {
        if err := v.refreshJWKS(ctx); err != nil {
            return nil, fmt.Errorf("failed to refresh JWKS: %w", err)
        }
    }

    // Parse and validate token
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        // Validate signing method
        if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }

        // Get key ID from token header
        kid, ok := token.Header["kid"].(string)
        if !ok {
            return nil, fmt.Errorf("kid header not found")
        }

        // Find key in JWKS
        key, found := v.jwkSet.LookupKeyID(kid)
        if !found {
            return nil, fmt.Errorf("key %s not found in JWKS", kid)
        }

        // Convert to RSA public key
        var pubKey interface{}
        if err := key.Raw(&pubKey); err != nil {
            return nil, fmt.Errorf("failed to get raw key: %w", err)
        }

        return pubKey, nil
    })

    if err != nil {
        return nil, fmt.Errorf("token validation failed: %w", err)
    }

    // Validate claims
    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid token claims")
    }

    // Validate issuer
    if claims.Issuer != v.issuer {
        return nil, fmt.Errorf("invalid issuer: expected %s, got %s", v.issuer, claims.Issuer)
    }

    // Validate audience
    if !contains(claims.Audience, v.audience) {
        return nil, fmt.Errorf("invalid audience")
    }

    return claims, nil
}

func (v *TokenValidator) refreshJWKS(ctx context.Context) error {
    jwkSet, err := jwk.Fetch(ctx, v.jwksURI)
    if err != nil {
        return fmt.Errorf("failed to fetch JWKS: %w", err)
    }

    v.jwkSet = jwkSet
    v.lastUpdate = time.Now()
    return nil
}
```

### Middleware Implementation

```go
// middleware/auth.go
func AuthMiddleware(validator *TokenValidator) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Extract token from Authorization header
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"error": "Missing authorization header"})
            c.Abort()
            return
        }

        // Verify Bearer prefix
        const bearerPrefix = "Bearer "
        if !strings.HasPrefix(authHeader, bearerPrefix) {
            c.JSON(401, gin.H{"error": "Invalid authorization format"})
            c.Abort()
            return
        }

        token := authHeader[len(bearerPrefix):]

        // Validate token
        claims, err := validator.ValidateToken(c.Request.Context(), token)
        if err != nil {
            c.JSON(401, gin.H{"error": "Invalid token", "details": err.Error()})
            c.Abort()
            return
        }

        // Store claims in context
        c.Set("user_claims", claims)
        c.Set("user_id", claims.Subject)
        c.Next()
    }
}
```

---

## MCP Protocol Integration

### Challenge: Dynamic Client Registration (DCR)

The Model Context Protocol specification requires support for Dynamic Client Registration (RFC 7591), but Azure AD does not natively support this feature. We implement a bridge service pattern to handle this requirement.

### DCR Bridge Service Architecture

```yaml
DCR Bridge Service:
  Purpose: Translate DCR requests to Azure AD app registrations
  Components:
    - API Gateway: Receives RFC 7591 compliant requests
    - Graph API Client: Creates Azure AD app registrations
    - Credential Store: Azure Key Vault for secrets
    - Response Formatter: Returns RFC 7591 responses
```

### Implementation Strategy

```go
// mcp/dcr_bridge.go
package mcp

import (
    "context"
    "encoding/json"
    "fmt"

    msgraph "github.com/microsoftgraph/msgraph-sdk-go"
    "github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
)

type DCRBridge struct {
    graphClient     *msgraph.GraphServiceClient
    keyVaultClient  *azsecrets.Client
    tenantID        string
}

// DCR Request per RFC 7591
type DCRRequest struct {
    ClientName              string   `json:"client_name"`
    RedirectURIs           []string `json:"redirect_uris"`
    GrantTypes             []string `json:"grant_types"`
    ResponseTypes          []string `json:"response_types"`
    Scope                  string   `json:"scope"`
    TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// DCR Response per RFC 7591
type DCRResponse struct {
    ClientID                string   `json:"client_id"`
    ClientSecret            string   `json:"client_secret,omitempty"`
    ClientSecretExpiresAt   int64    `json:"client_secret_expires_at,omitempty"`
    RedirectURIs           []string `json:"redirect_uris"`
    GrantTypes             []string `json:"grant_types"`
    ResponseTypes          []string `json:"response_types"`
    Scope                  string   `json:"scope"`
    TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

func (b *DCRBridge) RegisterClient(ctx context.Context, req DCRRequest) (*DCRResponse, error) {
    // Validate request
    if err := b.validateDCRRequest(req); err != nil {
        return nil, fmt.Errorf("invalid DCR request: %w", err)
    }

    // Create Azure AD app registration
    app, err := b.createAppRegistration(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("failed to create app registration: %w", err)
    }

    // Generate client secret
    secret, err := b.generateClientSecret(ctx, app.ID)
    if err != nil {
        return nil, fmt.Errorf("failed to generate client secret: %w", err)
    }

    // Store in Key Vault
    if err := b.storeCredentials(ctx, app.ID, secret); err != nil {
        return nil, fmt.Errorf("failed to store credentials: %w", err)
    }

    // Return RFC 7591 compliant response
    return &DCRResponse{
        ClientID:                app.AppID,
        ClientSecret:            secret.Value,
        ClientSecretExpiresAt:   secret.ExpiresAt.Unix(),
        RedirectURIs:           req.RedirectURIs,
        GrantTypes:             req.GrantTypes,
        ResponseTypes:          req.ResponseTypes,
        Scope:                  req.Scope,
        TokenEndpointAuthMethod: "client_secret_post",
    }, nil
}
```

### MCP Tool Authentication Flow

```mermaid
sequenceDiagram
    participant MCP as MCP Tool
    participant Bridge as DCR Bridge
    participant Graph as Graph API
    participant KV as Key Vault
    participant AAD as Azure AD

    MCP->>Bridge: POST /register (RFC 7591)
    Bridge->>Graph: Create App Registration
    Graph->>AAD: Register Application
    AAD->>Graph: Return App Details
    Graph->>Bridge: App Created
    Bridge->>Graph: Generate Secret
    Graph->>Bridge: Secret Created
    Bridge->>KV: Store Credentials
    KV->>Bridge: Stored
    Bridge->>MCP: DCR Response (client_id, secret)
    MCP->>AAD: OAuth Flow with Credentials
    AAD->>MCP: Access Token
```

---

## Security Implementation

### Defense in Depth Strategy

```yaml
Security Layers:
  1. Network Security:
    - HTTPS only (TLS 1.2+)
    - Certificate pinning (production)
    - Rate limiting on auth endpoints

  2. Application Security:
    - CORS properly configured
    - CSP headers implemented
    - XSS protection enabled
    - CSRF tokens for state-changing operations

  3. Authentication Security:
    - PKCE for SPA flow
    - No implicit grant flow
    - Token binding (when available)
    - Session timeout (configurable)

  4. Authorization Security:
    - Role-based access control (RBAC)
    - Principle of least privilege
    - Regular permission audits

  5. Data Security:
    - Encryption at rest (AES-256)
    - Encryption in transit (TLS)
    - PII data minimization
    - Audit logging
```

### Security Headers Configuration

```go
// middleware/security.go
func SecurityHeaders() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Prevent clickjacking
        c.Header("X-Frame-Options", "DENY")

        // Prevent MIME type sniffing
        c.Header("X-Content-Type-Options", "nosniff")

        // Enable XSS protection
        c.Header("X-XSS-Protection", "1; mode=block")

        // Content Security Policy
        c.Header("Content-Security-Policy",
            "default-src 'self'; " +
            "script-src 'self' 'unsafe-inline' https://login.microsoftonline.com; " +
            "style-src 'self' 'unsafe-inline'; " +
            "img-src 'self' data: https:; " +
            "connect-src 'self' https://login.microsoftonline.com https://graph.microsoft.com;")

        // Strict Transport Security (HSTS)
        if c.Request.TLS != nil {
            c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        }

        c.Next()
    }
}
```

### Rate Limiting Implementation

```go
// middleware/ratelimit.go
func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
    limiter := rate.NewLimiter(rate.Every(time.Minute/time.Duration(requestsPerMinute)), requestsPerMinute)

    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.JSON(429, gin.H{"error": "Rate limit exceeded"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

---

## Secret Management

### Azure Key Vault Integration

```yaml
Key Vault Configuration:
  Name: mcp-portal-kv
  Location: Same region as application

  Secrets:
    - azure-client-secret: Azure AD app client secret
    - jwt-signing-key: Backend JWT signing key (RSA 2048)
    - redis-connection: Redis connection string
    - database-connection: PostgreSQL connection string

  Access Policies:
    - Application (Managed Identity):
        - Get secrets
        - List secrets
    - Administrators:
        - Full access
    - DevOps Pipeline:
        - Get secrets only
```

### Secret Rotation Strategy

```go
// secrets/rotation.go
type SecretRotation struct {
    keyVault       *keyvault.Client
    graphClient    *msgraph.Client
    rotationPeriod time.Duration
}

func (sr *SecretRotation) RotateClientSecret(ctx context.Context) error {
    // Generate new secret
    newSecret := generateSecureRandomString(32)

    // Update in Azure AD
    if err := sr.updateAzureADSecret(ctx, newSecret); err != nil {
        return fmt.Errorf("failed to update Azure AD: %w", err)
    }

    // Store in Key Vault with version
    version := fmt.Sprintf("v%d", time.Now().Unix())
    if err := sr.keyVault.SetSecret(ctx, "azure-client-secret", newSecret, version); err != nil {
        return fmt.Errorf("failed to store in Key Vault: %w", err)
    }

    // Keep previous version for rollback
    // Schedule removal of old version after grace period

    return nil
}
```

### Environment Variables

```bash
# .env.example (Never commit actual values)

# Azure AD Configuration
AZURE_TENANT_ID=your-tenant-id
AZURE_CLIENT_ID=your-client-id
AZURE_CLIENT_SECRET= # Retrieved from Key Vault

# Frontend (public variables)
NEXT_PUBLIC_AZURE_CLIENT_ID=your-client-id
NEXT_PUBLIC_AZURE_TENANT_ID=your-tenant-id
NEXT_PUBLIC_REDIRECT_URI=http://localhost:3000

# Backend Configuration
JWT_SIGNING_KEY= # Retrieved from Key Vault
JWT_ISSUER=mcp-portal
JWT_AUDIENCE=mcp-portal-api

# Database
DATABASE_URL= # Retrieved from Key Vault

# Redis
REDIS_URL= # Retrieved from Key Vault

# Security
CORS_ORIGINS=http://localhost:3000,http://localhost:3001
SESSION_TIMEOUT=3600
TOKEN_EXPIRY=3600
REFRESH_TOKEN_EXPIRY=604800
```

---

## Token Flow Diagrams

### Initial Authentication Flow

```mermaid
sequenceDiagram
    participant User
    participant Browser
    participant NextJS as Next.js App
    participant MSAL as MSAL.js
    participant AAD as Azure AD
    participant Backend as Go Backend
    participant JWKS as JWKS Endpoint

    User->>Browser: Click Login
    Browser->>NextJS: Login Request
    NextJS->>MSAL: loginPopup()
    MSAL->>AAD: Authorization Request (PKCE)
    AAD->>User: Login Prompt
    User->>AAD: Credentials
    AAD->>AAD: Validate & Generate Code
    AAD->>MSAL: Authorization Code
    MSAL->>AAD: Exchange Code (with PKCE verifier)
    AAD->>MSAL: ID Token + Access Token
    MSAL->>Browser: Store in sessionStorage
    Browser->>NextJS: Login Success
    NextJS->>Backend: API Request + Bearer Token
    Backend->>JWKS: Fetch Public Keys
    JWKS->>Backend: RSA Public Keys
    Backend->>Backend: Validate Token Signature
    Backend->>NextJS: Protected Resource
    NextJS->>User: Display Content
```

### Token Refresh Flow

```mermaid
sequenceDiagram
    participant Browser
    participant MSAL as MSAL.js
    participant AAD as Azure AD
    participant Backend as Go Backend

    Browser->>MSAL: acquireTokenSilent()
    MSAL->>MSAL: Check Token Expiry
    alt Token Expired
        MSAL->>AAD: Refresh Token Request
        AAD->>AAD: Validate Refresh Token
        AAD->>MSAL: New Access Token
        MSAL->>Browser: Update sessionStorage
    else Token Valid
        MSAL->>Browser: Return Cached Token
    end
    Browser->>Backend: API Request + Bearer Token
    Backend->>Backend: Validate Token
    Backend->>Browser: Protected Resource
```

---

## Configuration Guide

### Step 1: Azure AD App Registration

```bash
# Using Azure CLI
az login

# Create app registration
az ad app create \
  --display-name "MCP-Portal" \
  --sign-in-audience "AzureADMyOrg"

# Get the app ID
APP_ID=$(az ad app list --display-name "MCP-Portal" --query "[0].appId" -o tsv)

# Add SPA redirect URIs
az ad app update --id $APP_ID \
  --spa-redirect-uris \
    "http://localhost:3000" \
    "http://localhost:3001" \
    "http://localhost:3000/auth/callback"

# Add Web redirect URI
az ad app update --id $APP_ID \
  --web-redirect-uris \
    "http://localhost:8080/auth/callback"

# Create service principal
az ad sp create --id $APP_ID

# Generate client secret
az ad app credential reset --id $APP_ID \
  --display-name "backend-secret" \
  --years 2

# Add API permissions
az ad app permission add --id $APP_ID \
  --api 00000003-0000-0000-c000-000000000000 \
  --api-permissions \
    e1fe6dd8-ba31-4d61-89e7-88639da4683d=Scope \
    37f7f235-527c-4136-accd-4a02d197296e=Scope \
    14dad69e-099b-42c9-810b-d002981feec1=Scope

# Grant admin consent
az ad app permission admin-consent --id $APP_ID
```

### Step 2: Frontend Environment Setup

```bash
# frontend/.env.local
NEXT_PUBLIC_AZURE_CLIENT_ID=your-client-id
NEXT_PUBLIC_AZURE_TENANT_ID=your-tenant-id
NEXT_PUBLIC_REDIRECT_URI=http://localhost:3000
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### Step 3: Backend Environment Setup

```bash
# backend/.env
AZURE_TENANT_ID=your-tenant-id
AZURE_CLIENT_ID=your-client-id
AZURE_CLIENT_SECRET=your-client-secret # From Key Vault in production
JWT_SIGNING_KEY=your-rsa-private-key # Generated or from Key Vault
DATABASE_URL=postgresql://user:pass@localhost/mcp_portal
REDIS_URL=redis://localhost:6379
```

### Step 4: Key Generation

```bash
# Generate RSA key pair for JWT signing
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem

# Convert to base64 for environment variable
base64 -i private.pem -o private_base64.txt
```

---

## Troubleshooting

### Common Issues and Solutions

#### Issue 1: AADSTS9002326 Error

**Error**: "Cross-origin token redemption is permitted only for the 'Single-Page Application' client-type"

**Cause**: Azure AD app is configured with "Web" platform only

**Solution**:

```bash
# Add SPA platform to existing app
az ad app update --id $APP_ID \
  --spa-redirect-uris "http://localhost:3000"
```

#### Issue 2: CORS Errors

**Error**: "Access to XMLHttpRequest blocked by CORS policy"

**Cause**: Backend not configured to accept frontend origin

**Solution**:

```go
// Add CORS middleware
router.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Authorization", "Content-Type"},
    ExposeHeaders:    []string{"Content-Length"},
    AllowCredentials: true,
    MaxAge:          12 * time.Hour,
}))
```

#### Issue 3: Token Validation Failures

**Error**: "Token signature validation failed"

**Cause**: JWKS cache outdated or network issues

**Solution**:

```go
// Force JWKS refresh
validator.refreshJWKS(ctx)
```

#### Issue 4: Session Loss on Refresh

**Error**: User logged out after page refresh

**Cause**: Using localStorage instead of sessionStorage, or MSAL not initialized

**Solution**:

```typescript
// Ensure MSAL initialization on app load
useEffect(() => {
  const initializeMsal = async () => {
    await msalInstance.initialize();
    await msalInstance.handleRedirectPromise();
  };
  initializeMsal();
}, []);
```

### Debug Logging

```typescript
// Frontend debug logging
if (process.env.NODE_ENV === 'development') {
  msalConfig.system.loggerOptions.logLevel = LogLevel.Verbose;
}

// Backend debug logging
if os.Getenv("DEBUG") == "true" {
    gin.SetMode(gin.DebugMode)
    log.SetLevel(log.DebugLevel)
}
```

---

## Security Considerations

### Critical Security Requirements

1. **Never Expose Secrets**

   - Client secrets must NEVER be in frontend code
   - Use environment variables and Key Vault
   - Rotate secrets regularly

2. **Token Storage**

   - Use sessionStorage, not localStorage
   - Clear tokens on logout
   - Implement token expiry handling

3. **HTTPS Enforcement**

   - Production MUST use HTTPS
   - Enable HSTS headers
   - Certificate pinning recommended

4. **Input Validation**

   - Validate all inputs server-side
   - Sanitize for XSS prevention
   - Use parameterized queries

5. **Audit Logging**
   - Log all authentication events
   - Log authorization failures
   - Never log sensitive data

### Security Checklist

- [ ] Azure AD app configured with SPA platform
- [ ] Implicit flow disabled
- [ ] PKCE enabled for SPA
- [ ] Client secrets in Key Vault
- [ ] HTTPS enforced in production
- [ ] CORS properly configured
- [ ] Security headers implemented
- [ ] Rate limiting enabled
- [ ] Token validation on every request
- [ ] Audit logging configured
- [ ] Secret rotation scheduled
- [ ] Penetration testing completed
- [ ] Security review conducted

---

## Future Roadmap

### Phase 1: Current Implementation (Q1 2025)

- ✅ Basic Azure AD authentication
- ✅ Frontend SPA with MSAL.js
- ✅ Backend token validation
- ⏳ Session management with Redis

### Phase 2: MCP Integration (Q2 2025)

- [ ] DCR bridge service implementation
- [ ] Automated MCP tool registration
- [ ] Graph API integration
- [ ] Key Vault automation

### Phase 3: Enhanced Security (Q3 2025)

- [ ] Certificate-based authentication
- [ ] Conditional Access policies
- [ ] Advanced threat protection
- [ ] Zero Trust network access

### Phase 4: Scale & Performance (Q4 2025)

- [ ] Token caching optimization
- [ ] Distributed session management
- [ ] Geographic redundancy
- [ ] Performance monitoring

### Considerations for Azure AD B2C Migration

Azure AD B2C will be deprecated in favor of External ID for customers by 2026. Plan migration:

1. Evaluate External ID features
2. Test migration in sandbox
3. Update authentication flows
4. Migrate user identities
5. Update documentation

---

## Appendix

### References

- [Microsoft Identity Platform Documentation](https://docs.microsoft.com/en-us/azure/active-directory/develop/)
- [MSAL.js Documentation](https://github.com/AzureAD/microsoft-authentication-library-for-js)
- [OAuth 2.0 RFC 6749](https://tools.ietf.org/html/rfc6749)
- [OpenID Connect Specification](https://openid.net/connect/)
- [Dynamic Client Registration RFC 7591](https://tools.ietf.org/html/rfc7591)
- [Model Context Protocol Specification](https://modelcontextprotocol.io/specification/)

### Glossary

- **DCR**: Dynamic Client Registration
- **JWKS**: JSON Web Key Set
- **PKCE**: Proof Key for Code Exchange
- **SPA**: Single-Page Application
- **MSAL**: Microsoft Authentication Library
- **MCP**: Model Context Protocol

### Support Contacts

- Security Issues: security@mcp-portal.com
- Technical Support: support@mcp-portal.com
- Documentation: docs@mcp-portal.com

---

**End of Document**

_This document contains sensitive security information. Handle with appropriate care and restrict access to authorized personnel only._
