#!/bin/bash
echo "🔍 Validating Azure configuration..."

# Required variables check
REQUIRED_VARS=(
  "AZURE_TENANT_ID"
  "AZURE_CLIENT_ID"
  "AZURE_CLIENT_SECRET"
  "JWT_SECRET"
  "KEY_VAULT_NAME"
)

for var in "${REQUIRED_VARS[@]}"; do
  if [[ -z "${!var}" ]]; then
    echo "❌ Missing required variable: $var"
    exit 1
  fi
done

# Test Azure AD connection
echo "Testing Azure AD connection..."
az ad app show --id $AZURE_CLIENT_ID --query "displayName" -o tsv > /dev/null
if [ $? -eq 0 ]; then
  echo "✅ Azure AD application accessible"
else
  echo "❌ Azure AD application not found or no access"
  exit 1
fi

# Test Key Vault access
echo "Testing Key Vault access..."
az keyvault secret list --vault-name $KEY_VAULT_NAME --query "length([])" -o tsv > /dev/null
if [ $? -eq 0 ]; then
  echo "✅ Key Vault accessible"
else
  echo "❌ Key Vault not accessible"
  exit 1
fi

# Test Graph API permissions
echo "Testing Graph API permissions..."
PERMISSIONS=$(az ad app permission list --id $AZURE_CLIENT_ID --query "[].resourceAccess[].id" -o tsv)
if [ -z "$PERMISSIONS" ]; then
  echo "❌ No permissions granted"
  exit 1
fi
echo "✅ Graph API permissions configured"

echo "🎉 All validations passed!"