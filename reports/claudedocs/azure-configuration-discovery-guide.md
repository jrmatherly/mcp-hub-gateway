# Azure Configuration Values Discovery Guide

Complete step-by-step guide for obtaining all required Azure configuration values for application integration.

## Table of Contents

1. [Tenant ID Discovery](#1-tenant-id-discovery)
2. [Subscription ID Location](#2-subscription-id-location)
3. [Client ID and Client Secret Generation](#3-client-id-and-client-secret-generation)
4. [Object ID vs Application ID Differences](#4-object-id-vs-application-id-differences)
5. [Resource Group and Key Vault Naming Requirements](#5-resource-group-and-key-vault-naming-requirements)
6. [Quick Reference Commands](#6-quick-reference-commands)

---

## 1. Tenant ID Discovery

The **Tenant ID** (also called Directory ID) uniquely identifies your Microsoft Entra ID (Azure Active Directory) instance.

### Method 1: Azure Portal

1. Sign in to the [Azure Portal](https://portal.azure.com)
2. Confirm you're in the correct tenant (check upper-right corner)
3. Navigate to **Microsoft Entra ID** (use search if not visible)
4. Go to **Overview** > **Properties** (or directly to **Properties**)
5. Find **Tenant ID** in the **Basic Information** section
6. Click the copy icon next to the Tenant ID value

### Method 2: Microsoft Entra Admin Center

1. Sign in to [Microsoft Entra Admin Center](https://entra.microsoft.com)
2. Browse to **Entra ID** > **Overview** > **Properties**
3. Copy the **Tenant ID** value

### Method 3: Azure CLI

```bash
# Login and show current tenant
az login
az account show --query tenantId --output tsv

# List all accessible tenants
az account tenant list --query "[].{Name:displayName, TenantId:tenantId}" --output table

# Get tenant for specific subscription
az account list --query "[?name=='Your Subscription Name'].tenantId" --output tsv
```

### Method 4: Azure PowerShell

```powershell
# Connect and get tenant information
Connect-AzAccount
Get-AzTenant

# Get current context tenant
(Get-AzContext).Tenant.Id

# Get tenant by subscription
Get-AzSubscription -SubscriptionName "Your Subscription Name" | Select-Object TenantId
```

### Method 5: Programmatic Discovery

```bash
# For Azure Databricks workspaces
curl -v <per-workspace-URL>/aad/auth
# Look for: location: https://login.microsoftonline.com/00000000-0000-0000-0000-000000000000
# The GUID is your tenant ID
```

---

## 2. Subscription ID Location

The **Subscription ID** uniquely identifies your Azure subscription within a tenant.

### Method 1: Azure Portal

1. Sign in to the [Azure Portal](https://portal.azure.com)
2. Navigate to **Subscriptions** (use search if not visible)
3. Locate your subscription in the list
4. Note the **Subscription ID** in the second column
5. Click on the subscription name for detailed view
6. Use the copy icon next to **Subscription ID** in the **Essentials** section

**Troubleshooting**: If no subscriptions appear, you may need to [switch directories](https://docs.microsoft.com/azure/azure-portal/set-preferences#switch-and-manage-directories).

### Method 2: Azure CLI

```bash
# Show current active subscription
az account show --output table
az account show --query id --output tsv

# List all subscriptions
az account list --output table
az account list --query "[].{Name:name, SubscriptionId:id, TenantId:tenantId}" --output table

# Find subscription by name
az account list --query "[?name=='My Subscription Name'].id" --output tsv

# Store in variable (Bash)
subscriptionId="$(az account list --query "[?isDefault].id" --output tsv)"
echo $subscriptionId

# Store in variable (PowerShell)
$subscriptionId = az account list --query "[?isDefault].id" --output tsv
Write-Host $subscriptionId
```

### Method 3: Azure PowerShell

```powershell
# Connect and get subscriptions
Connect-AzAccount
Get-AzSubscription

# Get current subscription
(Get-AzContext).Subscription.Id

# Get subscription by name
Get-AzSubscription -SubscriptionName "My Subscription Name" | Select-Object Id

# Set specific subscription as active
Set-AzContext -SubscriptionId "00000000-0000-0000-0000-000000000000"
```

---

## 3. Client ID and Client Secret Generation

**Client ID** (Application ID) and **Client Secret** are required for application authentication.

### Prerequisites: Create App Registration

#### Azure Portal Method

1. Sign in to [Azure Portal](https://portal.azure.com)
2. Navigate to **Microsoft Entra ID** > **App registrations**
3. Click **+ New registration**
4. Configure the registration:
   - **Name**: Enter a meaningful application name
   - **Supported account types**: Choose appropriate option
   - **Redirect URI**: Optional for server-to-server scenarios
5. Click **Register**

#### Azure CLI Method

```bash
# Create service principal with app registration
az ad sp create-for-rbac -n "MyApplication"

# Output includes:
# {
#   "appId": "00001111-aaaa-2222-bbbb-3333cccc4444",     # This is your Client ID
#   "displayName": "MyApplication",
#   "password": "123456.ABCDE.~XYZ876123ABcEdB7169",   # This is your Client Secret
#   "tenant": "aaaa0a0a-bb1b-cc2c-dd3d-eeeeee4e4e4e"   # This is your Tenant ID
# }
```

### Obtaining Client ID (Application ID)

#### Azure Portal

1. From your app registration overview page
2. Copy the **Application (client) ID** value
3. This is your **Client ID**

#### Azure CLI

```bash
# List apps and get Client ID
az ad app list --display-name "MyApplication" --query "[].appId" --output tsv

# Show specific app
az ad app show --id "00001111-aaaa-2222-bbbb-3333cccc4444" --query appId --output tsv
```

### Creating Client Secret

#### Azure Portal Method

1. From your app registration, navigate to **Certificates & secrets**
2. Under **Client secrets**, click **+ New client secret**
3. Provide:
   - **Description**: Meaningful description (e.g., "Production API Secret")
   - **Expires**: Select duration based on security policy
4. Click **Add**
5. **IMMEDIATELY COPY THE SECRET VALUE** - it won't be shown again
6. Store securely (consider Azure Key Vault)

#### Azure CLI Method

```bash
# Create new client secret
az ad app credential reset --id "00001111-aaaa-2222-bbbb-3333cccc4444" --append

# With custom description and duration
az ad app credential reset --id "00001111-aaaa-2222-bbbb-3333cccc4444" \
  --credential-description "Production Secret" \
  --years 1 \
  --append
```

### Important Security Notes

- **Client secrets are sensitive**: Store in Azure Key Vault or secure credential storage
- **Limited visibility**: Secret values can only be viewed immediately after creation
- **Rotation**: Implement regular secret rotation for security
- **Production environments**: Use certificates instead of secrets when possible
- **Permissions**: Grant minimal required permissions to the service principal

---

## 4. Object ID vs Application ID Differences

Understanding the distinction between **Object ID** and **Application ID** is crucial for proper Azure configuration.

### Key Differences

| Aspect           | Application ID (Client ID)                               | Object ID                                                               |
| ---------------- | -------------------------------------------------------- | ----------------------------------------------------------------------- |
| **Definition**   | Global identifier for the application across all tenants | Unique identifier for the service principal object in a specific tenant |
| **Scope**        | Global (same across all tenants)                         | Tenant-specific                                                         |
| **Use Case**     | Authentication token requests                            | Role assignments, permissions, tenant-specific operations               |
| **Format**       | GUID (e.g., `00001111-aaaa-2222-bbbb-3333cccc4444`)      | GUID (e.g., `22334455-bbbb-6666-cccc-7777dddd8888`)                     |
| **Relationship** | One-to-many with Object IDs                              | One-to-one with specific tenant                                         |

### When to Use Each

#### Application ID (Client ID)

- **Token requests**: When requesting OAuth tokens
- **Authentication flows**: Client credentials, authorization code flows
- **API permissions**: When configuring API access
- **Cross-tenant**: When the same app is used across multiple tenants

#### Object ID (Service Principal Object ID)

- **Role assignments**: Assigning RBAC roles to the application
- **Workspace permissions**: Adding app as admin to Power BI workspaces
- **Resource access**: Granting access to specific Azure resources
- **Tenant-specific operations**: Any operation within a specific tenant

### Finding Object ID

#### Azure Portal

1. Navigate to **Microsoft Entra ID** > **Enterprise applications**
2. Find your application (search by name or Application ID)
3. Click on the application
4. The **Object ID** is shown in the overview page

#### Azure CLI

```bash
# Get Object ID by Application ID
az ad sp list --filter "appId eq '00001111-aaaa-2222-bbbb-3333cccc4444'" --query "[].id" --output tsv

# Get Object ID by display name
az ad sp list --display-name "MyApplication" --query "[].id" --output tsv
```

#### Azure PowerShell

```powershell
# Get Object ID by Application ID
Get-MgServicePrincipal -Filter "appId eq '00001111-aaaa-2222-bbbb-3333cccc4444'" | Select-Object Id

# Get Object ID by display name
Get-MgServicePrincipal -Filter "displayName eq 'MyApplication'" | Select-Object Id
```

### Example Role Assignment

```bash
# Correct: Use Application ID for role assignment commands
az role assignment create --assignee "00001111-aaaa-2222-bbbb-3333cccc4444" --role "Reader"

# Incorrect: Don't use Object ID for Azure CLI role assignments
# az role assignment create --assignee "22334455-bbbb-6666-cccc-7777dddd8888" --role "Reader"
```

---

## 5. Resource Group and Key Vault Naming Requirements

Understanding naming conventions and requirements ensures successful resource creation.

### Resource Group Naming

#### Naming Rules

- **Scope**: Subscription-level uniqueness
- **Length**: 1-90 characters
- **Valid characters**: Alphanumerics, underscores, parentheses, hyphens, periods
- **Restrictions**:
  - Cannot end with period
  - Case-insensitive but preserves case
  - Cannot use certain reserved names

#### Recommended Patterns

```
# Standard pattern
rg-<workload>-<environment>-<region>-<instance>

# Examples
rg-webapp-prod-eastus-001
rg-analytics-dev-westus-001
rg-shared-prod-centralus-001

# Alternative patterns
rg-<project>-<environment>
rg-mcpgateway-prod
rg-mcpgateway-dev
```

#### Creation Examples

**Azure CLI:**

```bash
# Create resource group
az group create --name "rg-mcpgateway-prod-eastus-001" --location "eastus"

# List resource groups
az group list --output table

# Check name availability (implicit - will fail if exists)
az group show --name "rg-mcpgateway-prod-eastus-001"
```

**Azure PowerShell:**

```powershell
# Create resource group
New-AzResourceGroup -Name "rg-mcpgateway-prod-eastus-001" -Location "East US"

# List resource groups
Get-AzResourceGroup

# Check if exists
Get-AzResourceGroup -Name "rg-mcpgateway-prod-eastus-001" -ErrorAction SilentlyContinue
```

### Key Vault Naming

#### Naming Rules

- **Scope**: **Global** (must be unique across all of Azure)
- **Length**: 3-24 characters
- **Valid characters**: Alphanumerics and hyphens only
- **Restrictions**:
  - Must start with a letter
  - Must end with letter or number
  - Cannot contain consecutive hyphens
  - Cannot use reserved words

#### Recommended Patterns

```
# Standard pattern
kv-<workload>-<environment>-<region>-<instance>

# Examples (if length permits)
kv-mcpgateway-prod-001
kv-analytics-dev-001

# Shortened for length constraints
kv-mcp-prod-001
kv-app-dev-eastus

# Company prefix
kv-contoso-mcp-prod
```

#### Creation Examples

**Azure CLI:**

```bash
# Create Key Vault
az keyvault create \
  --name "kv-mcpgateway-prod-001" \
  --resource-group "rg-mcpgateway-prod-eastus-001" \
  --location "eastus"

# Check name availability
az keyvault list --query "[?name=='kv-mcpgateway-prod-001']"
```

**Azure PowerShell:**

```powershell
# Create Key Vault
New-AzKeyVault \
  -Name "kv-mcpgateway-prod-001" \
  -ResourceGroupName "rg-mcpgateway-prod-eastus-001" \
  -Location "East US"

# Check if name exists
Get-AzKeyVault -VaultName "kv-mcpgateway-prod-001" -ErrorAction SilentlyContinue
```

### Naming Best Practices

1. **Consistency**: Use the same pattern across all resources
2. **Abbreviations**: Use [official Azure abbreviations](https://docs.microsoft.com/azure/cloud-adoption-framework/ready/azure-best-practices/resource-abbreviations)
3. **Environments**: Standardize environment names (dev, test, prod, staging)
4. **Regions**: Use consistent region abbreviations (eastus, westus, etc.)
5. **Avoid sensitive data**: Don't include sensitive information in names
6. **Documentation**: Maintain a naming convention document

---

## 6. Quick Reference Commands

### One-Line Discovery Commands

```bash
# Get all key Azure identifiers at once
echo "Tenant ID: $(az account show --query tenantId -o tsv)"
echo "Subscription ID: $(az account show --query id -o tsv)"
echo "Subscription Name: $(az account show --query name -o tsv)"

# Create app registration and get all IDs
az ad sp create-for-rbac -n "MyApp" --query '{ClientID:appId, TenantID:tenant, Secret:password}'

# Get Object ID from Application ID
az ad sp list --filter "appId eq 'YOUR_APP_ID'" --query "[].id" -o tsv
```

### PowerShell Quick Reference

```powershell
# Get all key identifiers
$context = Get-AzContext
Write-Host "Tenant ID: $($context.Tenant.Id)"
Write-Host "Subscription ID: $($context.Subscription.Id)"
Write-Host "Subscription Name: $($context.Subscription.Name)"

# Get Object ID from Application ID
$appId = "YOUR_APP_ID"
$objectId = (Get-MgServicePrincipal -Filter "appId eq '$appId'").Id
Write-Host "Object ID: $objectId"
```

### Validation Commands

```bash
# Verify your configuration
az account show --output table
az ad sp show --id "YOUR_APP_ID" --query "{DisplayName:displayName, AppId:appId}" --output table
az group show --name "YOUR_RESOURCE_GROUP" --query "{Name:name, Location:location}" --output table
az keyvault show --name "YOUR_KEY_VAULT" --query "{Name:name, Location:location}" --output table
```

### Environment Setup Script

```bash
#!/bin/bash
# Azure Configuration Discovery Script

echo "=== Azure Configuration Discovery ==="

# Login check
if ! az account show &>/dev/null; then
    echo "Please login to Azure first:"
    echo "az login"
    exit 1
fi

# Get basic information
TENANT_ID=$(az account show --query tenantId -o tsv)
SUBSCRIPTION_ID=$(az account show --query id -o tsv)
SUBSCRIPTION_NAME=$(az account show --query name -o tsv)

echo "Current Azure Context:"
echo "  Tenant ID: $TENANT_ID"
echo "  Subscription ID: $SUBSCRIPTION_ID"
echo "  Subscription Name: $SUBSCRIPTION_NAME"

# List available subscriptions
echo -e "\nAvailable Subscriptions:"
az account list --query "[].{Name:name, SubscriptionId:id, TenantId:tenantId}" --output table

echo -e "\nTo get Application ID and Secret, run:"
echo "az ad sp create-for-rbac -n \"YourAppName\""

echo -e "\nTo create resources with proper naming:"
echo "az group create --name \"rg-yourproject-prod-eastus-001\" --location \"eastus\""
echo "az keyvault create --name \"kv-yourproject-prod-001\" --resource-group \"rg-yourproject-prod-eastus-001\" --location \"eastus\""
```

---

## Troubleshooting Common Issues

### Tenant ID Issues

- **Multiple tenants**: Use `az account tenant list` to see all accessible tenants
- **Wrong tenant**: Use `az login --tenant TENANT_ID` to login to specific tenant
- **Guest accounts**: May require explicit tenant specification

### Subscription Access

- **No subscriptions**: Contact Azure administrator for subscription access
- **Hidden subscriptions**: Try switching directories in Azure Portal
- **Permissions**: Ensure you have at least Reader role on subscription

### App Registration Problems

- **Insufficient permissions**: Need Application Developer or Global Administrator role
- **Client Secret expiry**: Set appropriate expiration based on security policies
- **Missing permissions**: Grant necessary API permissions after app registration

### Naming Conflicts

- **Key Vault names**: Must be globally unique - try adding random suffix
- **Reserved names**: Avoid system reserved words and common patterns
- **Length constraints**: Plan for shortest naming requirements across all resources

### CLI Authentication

- **Token expiry**: Run `az login` to refresh authentication
- **Multiple accounts**: Use `az account set --subscription SUBSCRIPTION_ID`
- **Service principal login**: Use `az login --service-principal -u APP_ID -p SECRET --tenant TENANT_ID`

This comprehensive guide provides all the necessary methods to discover and configure Azure values for application integration. Keep this reference handy when setting up Azure-based applications and services.
