# Azure Configuration Guide for MCP Portal OAuth

**Last Updated**: September 19, 2025
**Version**: 1.0
**Status**: Production Ready

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Azure AD Application Registration](#azure-ad-application-registration)
4. [Microsoft Graph API Permissions](#microsoft-graph-api-permissions)
5. [Azure Key Vault Setup](#azure-key-vault-setup)
6. [Environment Configuration](#environment-configuration)
7. [Authentication Methods](#authentication-methods)
8. [Testing Your Configuration](#testing-your-configuration)
9. [Troubleshooting](#troubleshooting)
10. [Security Best Practices](#security-best-practices)

## Overview

The MCP Portal OAuth implementation uses Azure services for secure credential management and dynamic client registration. This guide covers the complete setup process for:

- **Azure AD Application Registration** for OAuth provider management
- **Microsoft Graph API** for dynamic client registration (DCR)
- **Azure Key Vault** for secure credential storage
- **Authentication** using Azure SDK for Go

### Architecture Components

```
MCP Portal → Azure AD (Authentication)
    ↓
DCR Bridge → Microsoft Graph API (App Registration)
    ↓
Key Vault → Secure Credential Storage
```

## Prerequisites

### Required Azure Resources

- Azure subscription with appropriate permissions
- Azure AD tenant (Entra ID)
- Azure Key Vault instance
- Azure CLI installed locally

### Required Permissions

You'll need one of the following roles:

- **Privileged Role Administrator** (can both add and grant permissions)
- **Application Administrator** or **Cloud Application Administrator** (can add permissions only)

### SDK Versions (Already Integrated)

```go
// go.mod - Already configured in the project
require (
    github.com/Azure/azure-sdk-for-go/sdk/azcore v1.18.0
    github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.10.1
    github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets v1.4.0
    github.com/microsoftgraph/msgraph-sdk-go v1.64.0
)
```

## Azure AD Application Registration

### Step 1: Create the Application

```bash
# Using Azure CLI
az ad app create \
  --display-name "MCP Portal OAuth Manager" \
  --sign-in-audience "AzureADMyOrg" \
  --enable-access-token-issuance true \
  --enable-id-token-issuance true
```

### Step 2: Create a Service Principal

```bash
# Get the app ID from the previous step
APP_ID=$(az ad app list --display-name "MCP Portal OAuth Manager" --query "[0].appId" -o tsv)

# Create service principal
az ad sp create --id $APP_ID
```

### Step 3: Create Client Secret

```bash
# Create a client secret (valid for 2 years)
az ad app credential reset \
  --id $APP_ID \
  --years 2 \
  --display-name "MCP Portal Production"
```

Save the returned credentials securely:

- `appId`: Your Azure AD Application (client) ID
- `password`: Your client secret
- `tenant`: Your Azure AD tenant ID

## Microsoft Graph API Permissions

### Required Permissions

The DCR Bridge requires the following Microsoft Graph API permissions:

| Permission                        | Type        | Purpose                             |
| --------------------------------- | ----------- | ----------------------------------- |
| `Application.ReadWrite.All`       | Application | Create and manage app registrations |
| `Directory.ReadWrite.All`         | Application | Manage directory objects            |
| `AppRoleAssignment.ReadWrite.All` | Application | Grant permissions to created apps   |

### Grant Permissions via Azure Portal

1. Navigate to [Azure Portal](https://portal.azure.com)
2. Go to **Azure Active Directory** → **App registrations**
3. Select your "MCP Portal OAuth Manager" application
4. Click **API permissions** → **Add a permission**
5. Select **Microsoft Graph** → **Application permissions**
6. Add the required permissions listed above
7. Click **Grant admin consent** (requires admin privileges)

### Grant Permissions via Azure CLI

```bash
# Get the Microsoft Graph service principal ID
GRAPH_SP_ID=$(az ad sp list --query "[?displayName=='Microsoft Graph'].id" -o tsv --all)

# Grant Application.ReadWrite.All
az ad app permission add \
  --id $APP_ID \
  --api 00000003-0000-0000-c000-000000000000 \
  --api-permissions 1bfefb4e-e0b5-418b-a88f-73c46d2cc8e9=Role

# Grant Directory.ReadWrite.All
az ad app permission add \
  --id $APP_ID \
  --api 00000003-0000-0000-c000-000000000000 \
  --api-permissions 19dbc75e-c2e2-444c-a770-ec69d8559fc7=Role

# Grant admin consent
az ad app permission admin-consent --id $APP_ID
```

## Azure Key Vault Setup

### Step 1: Create Key Vault

```bash
# Create resource group
az group create --name mcp-portal-rg --location eastus

# Create Key Vault with unique name
KEYVAULT_NAME="mcp-portal-kv-$(openssl rand -hex 4)"
az keyvault create \
  --name $KEYVAULT_NAME \
  --resource-group mcp-portal-rg \
  --location eastus \
  --enable-rbac-authorization
```

### Step 2: Grant Key Vault Access

```bash
# Get your user principal name
USER_UPN=$(az ad signed-in-user show --query userPrincipalName -o tsv)

# Grant Key Vault Secrets Officer role
az role assignment create \
  --role "Key Vault Secrets Officer" \
  --assignee $USER_UPN \
  --scope /subscriptions/$(az account show --query id -o tsv)/resourceGroups/mcp-portal-rg/providers/Microsoft.KeyVault/vaults/$KEYVAULT_NAME

# Grant access to the service principal
SP_OBJECT_ID=$(az ad sp show --id $APP_ID --query id -o tsv)
az role assignment create \
  --role "Key Vault Secrets User" \
  --assignee-object-id $SP_OBJECT_ID \
  --scope /subscriptions/$(az account show --query id -o tsv)/resourceGroups/mcp-portal-rg/providers/Microsoft.KeyVault/vaults/$KEYVAULT_NAME
```

### Step 3: Store Initial Secrets

```bash
# Store application credentials
az keyvault secret set \
  --vault-name $KEYVAULT_NAME \
  --name "azure-client-id" \
  --value $APP_ID

az keyvault secret set \
  --vault-name $KEYVAULT_NAME \
  --name "azure-client-secret" \
  --value "YOUR_CLIENT_SECRET_HERE"

az keyvault secret set \
  --vault-name $KEYVAULT_NAME \
  --name "azure-tenant-id" \
  --value $(az account show --query tenantId -o tsv)
```

## Environment Configuration

### Development Environment (.env)

```bash
# Azure AD Configuration
AZURE_TENANT_ID=your-tenant-id
AZURE_CLIENT_ID=your-client-id
AZURE_CLIENT_SECRET=your-client-secret

# Key Vault Configuration
AZURE_KEY_VAULT_URL=https://your-keyvault-name.vault.azure.net/

# Microsoft Graph Configuration
GRAPH_API_ENDPOINT=https://graph.microsoft.com

# OAuth Configuration
OAUTH_REDIRECT_URI=http://localhost:8080/callback
OAUTH_AUTHORITY=https://login.microsoftonline.com/your-tenant-id
```

### Production Environment

For production, use managed identities instead of client secrets:

```bash
# Enable system-assigned managed identity
az containerapp identity assign \
  --name mcp-portal \
  --resource-group mcp-portal-rg \
  --system-assigned

# Grant managed identity access to Key Vault
IDENTITY_ID=$(az containerapp identity show \
  --name mcp-portal \
  --resource-group mcp-portal-rg \
  --query principalId -o tsv)

az role assignment create \
  --role "Key Vault Secrets User" \
  --assignee-object-id $IDENTITY_ID \
  --scope /subscriptions/$(az account show --query id -o tsv)/resourceGroups/mcp-portal-rg/providers/Microsoft.KeyVault/vaults/$KEYVAULT_NAME
```

## Authentication Methods

### Local Development

The implementation uses `DefaultAzureCredential` which automatically detects the authentication method:

1. **Azure CLI Authentication** (Recommended for local development)

   ```bash
   az login
   ```

2. **Service Principal** (CI/CD environments)
   ```bash
   export AZURE_TENANT_ID=your-tenant-id
   export AZURE_CLIENT_ID=your-client-id
   export AZURE_CLIENT_SECRET=your-client-secret
   ```

### Production (Managed Identity)

The code automatically uses the managed identity when deployed:

```go
// This code is already implemented in dcr_bridge.go
credential, err := azidentity.NewDefaultAzureCredential(nil)
if err != nil {
    return nil, fmt.Errorf("failed to create Azure credential: %w", err)
}
```

## Testing Your Configuration

### Step 1: Verify Azure AD Application

```bash
# Verify app registration
az ad app show --id $APP_ID --query "{id:id, displayName:displayName, appId:appId}"

# Verify permissions
az ad app permission list --id $APP_ID --output table
```

### Step 2: Test Key Vault Access

```bash
# Test secret retrieval
az keyvault secret show \
  --vault-name $KEYVAULT_NAME \
  --name "azure-client-id"
```

### Step 3: Test Graph API Access

```go
// Test program to verify Graph API access
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
    msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
)

func main() {
    cred, err := azidentity.NewDefaultAzureCredential(nil)
    if err != nil {
        log.Fatal(err)
    }

    client, err := msgraphsdk.NewGraphServiceClientWithCredentials(cred, []string{})
    if err != nil {
        log.Fatal(err)
    }

    // Test: List applications
    apps, err := client.Applications().Get(context.Background(), nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Successfully connected to Graph API. Found %d applications\n",
        len(apps.GetValue()))
}
```

## Troubleshooting

### Common Issues and Solutions

#### 1. "Insufficient privileges" Error

**Problem**: Graph API returns 403 Forbidden
**Solution**: Ensure admin consent is granted for all permissions

```bash
az ad app permission admin-consent --id $APP_ID
```

#### 2. "Key Vault access denied" Error

**Problem**: Cannot read/write secrets to Key Vault
**Solution**: Verify role assignments

```bash
# Check current role assignments
az role assignment list \
  --assignee $SP_OBJECT_ID \
  --scope /subscriptions/$(az account show --query id -o tsv)/resourceGroups/mcp-portal-rg/providers/Microsoft.KeyVault/vaults/$KEYVAULT_NAME
```

#### 3. "Invalid client secret" Error

**Problem**: Authentication fails with invalid credentials
**Solution**: Regenerate client secret

```bash
az ad app credential reset --id $APP_ID --years 2
```

#### 4. DefaultAzureCredential Not Working Locally

**Problem**: Local development authentication fails
**Solution**: Ensure Azure CLI is logged in

```bash
az logout
az login
az account set --subscription "Your Subscription Name"
```

### Debug Logging

Enable debug logging for Azure SDK:

```go
import "github.com/Azure/azure-sdk-for-go/sdk/azcore/log"

// Enable logging
log.SetEvents(log.EventRequest, log.EventResponse)
log.SetListener(func(event log.Event, message string) {
    fmt.Printf("[%s] %s\n", event, message)
})
```

## Security Best Practices

### 1. Credential Rotation

- Rotate client secrets every 90 days
- Use managed identities in production
- Never commit credentials to source control

### 2. Least Privilege Access

- Grant only required permissions
- Use separate service principals for different environments
- Regularly audit permission grants

### 3. Key Vault Security

- Enable soft-delete and purge protection
- Use private endpoints in production
- Enable Key Vault logging and monitoring

```bash
# Enable soft-delete and purge protection
az keyvault update \
  --name $KEYVAULT_NAME \
  --enable-soft-delete true \
  --enable-purge-protection true
```

### 4. Network Security

- Restrict Key Vault network access
- Use private endpoints for production
- Enable firewall rules

```bash
# Restrict to specific IP ranges
az keyvault network-rule add \
  --name $KEYVAULT_NAME \
  --ip-address "YOUR_IP_RANGE"
```

### 5. Monitoring and Alerts

```bash
# Enable diagnostic logging
az monitor diagnostic-settings create \
  --name "keyvault-diagnostics" \
  --resource /subscriptions/$(az account show --query id -o tsv)/resourceGroups/mcp-portal-rg/providers/Microsoft.KeyVault/vaults/$KEYVAULT_NAME \
  --logs '[{"category": "AuditEvent", "enabled": true}]' \
  --workspace "YOUR_LOG_ANALYTICS_WORKSPACE_ID"
```

## Validation Checklist

Before deploying to production, ensure:

- [ ] Azure AD application created and configured
- [ ] Microsoft Graph permissions granted with admin consent
- [ ] Key Vault created and accessible
- [ ] Client credentials stored securely in Key Vault
- [ ] Environment variables configured correctly
- [ ] Managed identity enabled for production resources
- [ ] Network restrictions applied to Key Vault
- [ ] Monitoring and alerting configured
- [ ] Credential rotation schedule established
- [ ] Backup and recovery procedures documented

## Support Resources

- [Azure AD Documentation](https://docs.microsoft.com/en-us/azure/active-directory/)
- [Microsoft Graph API Reference](https://docs.microsoft.com/en-us/graph/api/overview)
- [Azure Key Vault Documentation](https://docs.microsoft.com/en-us/azure/key-vault/)
- [Azure SDK for Go](https://github.com/Azure/azure-sdk-for-go)
- [MCP Portal Issues](https://github.com/jrmatherly/mcp-hub-gateway/issues)

## Next Steps

1. Complete the configuration steps in this guide
2. Run the test programs to verify connectivity
3. Deploy to a development environment
4. Perform end-to-end OAuth flow testing
5. Schedule security review before production deployment

---

**Document Version**: 1.0
**Last Review**: September 19, 2025
**Next Review**: October 19, 2025
