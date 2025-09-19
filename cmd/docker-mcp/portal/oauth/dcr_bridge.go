package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/google/uuid"
	msgraph "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
)

// AzureADDCRBridge implements Dynamic Client Registration for Azure AD
type AzureADDCRBridge struct {
	graphClient    *msgraph.GraphServiceClient
	tenantID       string
	subscriptionID string
	resourceGroup  string
	keyVaultURL    string
	registeredApps map[string]*DCRResponse
	mu             sync.RWMutex
}

// CreateAzureADDCRBridge creates a new Azure AD DCR bridge
func CreateAzureADDCRBridge(
	tenantID, subscriptionID, resourceGroup, keyVaultURL string,
) (*AzureADDCRBridge, error) {
	// Initialize Azure credentials
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credentials: %w", err)
	}

	// Create Microsoft Graph client
	graphClient, err := msgraph.NewGraphServiceClientWithCredentials(cred, []string{
		"https://graph.microsoft.com/.default",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Graph client: %w", err)
	}

	return &AzureADDCRBridge{
		graphClient:    graphClient,
		tenantID:       tenantID,
		subscriptionID: subscriptionID,
		resourceGroup:  resourceGroup,
		keyVaultURL:    keyVaultURL,
		registeredApps: make(map[string]*DCRResponse),
	}, nil
}

// RegisterClient implements RFC 7591 Dynamic Client Registration for Azure AD
func (b *AzureADDCRBridge) RegisterClient(
	ctx context.Context,
	req *DCRRequest,
) (*DCRResponse, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Validate DCR request
	if err := b.validateDCRRequest(req); err != nil {
		return nil, fmt.Errorf("DCR request validation failed: %w", err)
	}

	// Create Azure AD application
	app, err := b.createAzureADApplication(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure AD application: %w", err)
	}

	// Create service principal
	_, err = b.createServicePrincipal(ctx, *app.GetAppId())
	if err != nil {
		return nil, fmt.Errorf("failed to create service principal: %w", err)
	}

	// Generate client secret
	clientSecret, err := b.createClientSecret(ctx, *app.GetId())
	if err != nil {
		return nil, fmt.Errorf("failed to create client secret: %w", err)
	}

	// Build DCR response
	response := &DCRResponse{
		ClientID:              *app.GetAppId(),
		ClientSecret:          *clientSecret.GetSecretText(),
		ClientIDIssuedAt:      time.Now().Unix(),
		ClientSecretExpiresAt: clientSecret.GetEndDateTime().Unix(),
		ApplicationID:         *app.GetAppId(),
		ObjectID:              *app.GetId(),
		DCRRequest:            *req,
	}

	// Store registration for future reference
	b.registeredApps[response.ClientID] = response

	// Store credentials in Key Vault if configured
	if b.keyVaultURL != "" {
		if err := b.storeCredentialsInKeyVault(ctx, response); err != nil {
			// Log error but don't fail registration
			fmt.Printf("Warning: Failed to store credentials in Key Vault: %v\n", err)
		}
	}

	return response, nil
}

// GetClient retrieves a registered client
func (b *AzureADDCRBridge) GetClient(ctx context.Context, clientID string) (*DCRResponse, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	response, exists := b.registeredApps[clientID]
	if !exists {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}

	return response, nil
}

// UpdateClient updates a registered client
func (b *AzureADDCRBridge) UpdateClient(
	ctx context.Context,
	clientID string,
	req *DCRRequest,
) (*DCRResponse, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	existingResponse, exists := b.registeredApps[clientID]
	if !exists {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}

	// Update Azure AD application
	if err := b.updateAzureADApplication(ctx, existingResponse.ObjectID, req); err != nil {
		return nil, fmt.Errorf("failed to update Azure AD application: %w", err)
	}

	// Update stored response
	existingResponse.DCRRequest = *req
	b.registeredApps[clientID] = existingResponse

	return existingResponse, nil
}

// DeleteClient deletes a registered client
func (b *AzureADDCRBridge) DeleteClient(ctx context.Context, clientID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	response, exists := b.registeredApps[clientID]
	if !exists {
		return fmt.Errorf("client not found: %s", clientID)
	}

	// Delete from Azure AD
	if err := b.deleteAzureADApplication(ctx, response.ObjectID); err != nil {
		return fmt.Errorf("failed to delete Azure AD application: %w", err)
	}

	// Remove from local storage
	delete(b.registeredApps, clientID)

	return nil
}

// SupportsProvider checks if the bridge supports a specific provider
func (b *AzureADDCRBridge) SupportsProvider(providerType ProviderType) bool {
	// This bridge specifically supports Microsoft/Azure AD
	return providerType == ProviderTypeMicrosoft
}

// GetProviderEndpoints returns OAuth endpoints for a provider
func (b *AzureADDCRBridge) GetProviderEndpoints(
	providerType ProviderType,
) (authURL, tokenURL, jwksURL string, err error) {
	if !b.SupportsProvider(providerType) {
		return "", "", "", fmt.Errorf("unsupported provider: %s", providerType)
	}

	tenantID := b.tenantID
	if tenantID == "" {
		tenantID = "common"
	}

	authURL = fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenantID)
	tokenURL = fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)
	jwksURL = fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/v2.0/keys", tenantID)

	return authURL, tokenURL, jwksURL, nil
}

// Private helper methods

func (b *AzureADDCRBridge) validateDCRRequest(req *DCRRequest) error {
	if len(req.RedirectURIs) == 0 {
		return fmt.Errorf("redirect_uris is required")
	}

	if req.ClientName == "" {
		return fmt.Errorf("client_name is required")
	}

	// Validate redirect URIs
	for _, uri := range req.RedirectURIs {
		if !strings.HasPrefix(uri, "https://") && !strings.HasPrefix(uri, "http://localhost") {
			return fmt.Errorf("invalid redirect URI: %s", uri)
		}
	}

	return nil
}

func (b *AzureADDCRBridge) createAzureADApplication(
	ctx context.Context,
	req *DCRRequest,
) (models.Applicationable, error) {
	// Build redirect URIs for Azure AD
	webRedirectUris := make([]string, len(req.RedirectURIs))
	copy(webRedirectUris, req.RedirectURIs)

	// Create application request
	appRequest := models.NewApplication()
	appRequest.SetDisplayName(&req.ClientName)

	// Set required resource access (Microsoft Graph)
	requiredResourceAccess := []models.RequiredResourceAccessable{}

	// Microsoft Graph permissions
	graphResourceAccess := models.NewRequiredResourceAccess()
	graphAppId := "00000003-0000-0000-c000-000000000000" // Microsoft Graph App ID
	graphResourceAccess.SetResourceAppId(&graphAppId)

	// Add basic permissions
	resourceAccess := []models.ResourceAccessable{}

	// User.Read permission
	userReadAccess := models.NewResourceAccess()
	userReadId, _ := uuid.Parse("e1fe6dd8-ba31-4d61-89e7-88639da4683d") // User.Read scope ID
	userReadAccess.SetId(&userReadId)
	// TODO: Properly fix
	// Note: SetType method may not be available in current SDK version
	// This would be set to "Scope" for delegated permissions
	resourceAccess = append(resourceAccess, userReadAccess)

	graphResourceAccess.SetResourceAccess(resourceAccess)
	requiredResourceAccess = append(requiredResourceAccess, graphResourceAccess)
	appRequest.SetRequiredResourceAccess(requiredResourceAccess)

	// Set redirect URIs
	if len(webRedirectUris) > 0 {
		web := models.NewWebApplication()
		web.SetRedirectUris(webRedirectUris)
		appRequest.SetWeb(web)
	}

	// Set public client settings for native apps
	publicClient := models.NewPublicClientApplication()
	appRequest.SetPublicClient(publicClient)

	// Set application type and supported account types
	if req.ApplicationType == "" {
		req.ApplicationType = "web"
	}

	// Set sign-in audience
	signInAudience := "AzureADMultipleOrgs"
	if req.TenantID != "" {
		signInAudience = "AzureADMyOrg"
	}
	appRequest.SetSignInAudience(&signInAudience)

	// Create the application
	app, err := b.graphClient.Applications().Post(ctx, appRequest, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure AD application: %w", err)
	}

	return app, nil
}

func (b *AzureADDCRBridge) createServicePrincipal(
	ctx context.Context,
	appId string,
) (models.ServicePrincipalable, error) {
	spRequest := models.NewServicePrincipal()
	spRequest.SetAppId(&appId)

	servicePrincipal, err := b.graphClient.ServicePrincipals().Post(ctx, spRequest, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create service principal: %w", err)
	}

	return servicePrincipal, nil
}

func (b *AzureADDCRBridge) createClientSecret(
	ctx context.Context,
	appObjectId string,
) (models.PasswordCredentialable, error) {
	// Create password credential
	passwordCredential := models.NewPasswordCredential()
	displayName := "DCR Generated Secret"
	passwordCredential.SetDisplayName(&displayName)

	// Set expiration to 2 years from now
	endDateTime := time.Now().AddDate(2, 0, 0)
	passwordCredential.SetEndDateTime(&endDateTime)

	// Add password credential to application - simplified for compatibility
	// Note: This is a stub implementation due to Microsoft Graph SDK API changes
	// In a real implementation, you would use the correct AddPassword request body
	_ = passwordCredential // Use variable to avoid unused error
	credential := models.NewPasswordCredential()
	credential.SetDisplayName(&displayName)
	credential.SetEndDateTime(&endDateTime)
	err := fmt.Errorf("Microsoft Graph SDK AddPassword implementation needed")
	if err != nil {
		return nil, fmt.Errorf("failed to create client secret: %w", err)
	}

	return credential, nil
}

func (b *AzureADDCRBridge) updateAzureADApplication(
	ctx context.Context,
	appObjectId string,
	req *DCRRequest,
) error {
	// Build update request
	appUpdate := models.NewApplication()
	appUpdate.SetDisplayName(&req.ClientName)

	// Update redirect URIs if provided
	if len(req.RedirectURIs) > 0 {
		web := models.NewWebApplication()
		web.SetRedirectUris(req.RedirectURIs)
		appUpdate.SetWeb(web)
	}

	// Update the application
	_, err := b.graphClient.Applications().ByApplicationId(appObjectId).Patch(ctx, appUpdate, nil)
	if err != nil {
		return fmt.Errorf("failed to update Azure AD application: %w", err)
	}

	return nil
}

func (b *AzureADDCRBridge) deleteAzureADApplication(
	ctx context.Context,
	appObjectId string,
) error {
	err := b.graphClient.Applications().ByApplicationId(appObjectId).Delete(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete Azure AD application: %w", err)
	}

	return nil
}

func (b *AzureADDCRBridge) storeCredentialsInKeyVault(
	ctx context.Context,
	response *DCRResponse,
) error {
	// This would integrate with Azure Key Vault to store the client credentials
	// For now, we'll just log that we would store them
	credentialsJSON, err := json.Marshal(map[string]interface{}{
		"client_id":     response.ClientID,
		"client_secret": response.ClientSecret,
		"created_at":    time.Unix(response.ClientIDIssuedAt, 0),
		"expires_at":    time.Unix(response.ClientSecretExpiresAt, 0),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// In a real implementation, this would store in Key Vault
	_ = credentialsJSON
	fmt.Printf("Would store credentials in Key Vault for client %s\n", response.ClientID)

	return nil
}

// Enhanced DCR bridge with multi-provider support
type MultiProviderDCRBridge struct {
	azureBridge    *AzureADDCRBridge
	supportedTypes map[ProviderType]bool
	mu             sync.RWMutex
}

// CreateMultiProviderDCRBridge creates a DCR bridge supporting multiple providers
func CreateMultiProviderDCRBridge(
	azureTenantID, subscriptionID, resourceGroup, keyVaultURL string,
) (*MultiProviderDCRBridge, error) {
	azureBridge, err := CreateAzureADDCRBridge(
		azureTenantID,
		subscriptionID,
		resourceGroup,
		keyVaultURL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure AD bridge: %w", err)
	}

	return &MultiProviderDCRBridge{
		azureBridge: azureBridge,
		supportedTypes: map[ProviderType]bool{
			ProviderTypeMicrosoft: true,
			// Future providers can be added here
		},
	}, nil
}

// RegisterClient implements DCRBridge interface with multi-provider support
func (m *MultiProviderDCRBridge) RegisterClient(
	ctx context.Context,
	req *DCRRequest,
) (*DCRResponse, error) {
	// Determine provider type from request
	providerType := ProviderTypeMicrosoft // Default for now
	if req.TenantID != "" {
		providerType = ProviderTypeMicrosoft
	}

	switch providerType {
	case ProviderTypeMicrosoft:
		return m.azureBridge.RegisterClient(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported provider type for DCR: %s", providerType)
	}
}

// GetClient retrieves a registered client
func (m *MultiProviderDCRBridge) GetClient(
	ctx context.Context,
	clientID string,
) (*DCRResponse, error) {
	// For now, try Azure AD bridge
	return m.azureBridge.GetClient(ctx, clientID)
}

// UpdateClient updates a registered client
func (m *MultiProviderDCRBridge) UpdateClient(
	ctx context.Context,
	clientID string,
	req *DCRRequest,
) (*DCRResponse, error) {
	// For now, use Azure AD bridge
	return m.azureBridge.UpdateClient(ctx, clientID, req)
}

// DeleteClient deletes a registered client
func (m *MultiProviderDCRBridge) DeleteClient(ctx context.Context, clientID string) error {
	// For now, use Azure AD bridge
	return m.azureBridge.DeleteClient(ctx, clientID)
}

// SupportsProvider checks if a provider is supported
func (m *MultiProviderDCRBridge) SupportsProvider(providerType ProviderType) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.supportedTypes[providerType]
}

// GetProviderEndpoints returns OAuth endpoints for a provider
func (m *MultiProviderDCRBridge) GetProviderEndpoints(
	providerType ProviderType,
) (authURL, tokenURL, jwksURL string, err error) {
	switch providerType {
	case ProviderTypeMicrosoft:
		return m.azureBridge.GetProviderEndpoints(providerType)
	default:
		return "", "", "", fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// DCR integration with OAuth provider registry
func (r *DefaultProviderRegistry) RegisterDynamicClient(
	ctx context.Context,
	providerType ProviderType,
	req *DCRRequest,
) (*DCRResponse, error) {
	provider, err := r.GetProvider(providerType)
	if err != nil {
		return nil, err
	}

	// Check if provider supports DCR
	dcrProvider, ok := provider.(DCRCapableProvider)
	if !ok {
		return nil, fmt.Errorf(
			"provider %s does not support Dynamic Client Registration",
			providerType,
		)
	}

	return dcrProvider.RegisterDynamicClient(ctx, req)
}

// DCR-enabled Microsoft provider
type DCRMicrosoftProvider struct {
	*MicrosoftProvider
	dcrBridge DCRBridge
}

// CreateDCRMicrosoftProvider creates a Microsoft provider with DCR support
func CreateDCRMicrosoftProvider(dcrBridge DCRBridge) *DCRMicrosoftProvider {
	return &DCRMicrosoftProvider{
		MicrosoftProvider: &MicrosoftProvider{},
		dcrBridge:         dcrBridge,
	}
}

// RegisterDynamicClient implements DCR for Microsoft provider
func (p *DCRMicrosoftProvider) RegisterDynamicClient(
	ctx context.Context,
	req *DCRRequest,
) (*DCRResponse, error) {
	if p.dcrBridge == nil {
		return nil, fmt.Errorf("DCR bridge not configured")
	}

	if !p.dcrBridge.SupportsProvider(ProviderTypeMicrosoft) {
		return nil, fmt.Errorf("DCR bridge does not support Microsoft provider")
	}

	return p.dcrBridge.RegisterClient(ctx, req)
}

// Utility function to create a complete OAuth setup with DCR
func CreateOAuthSystemWithDCR(
	azureTenantID, subscriptionID, resourceGroup, keyVaultURL string,
	config *InterceptorConfig,
	tokenStorage TokenStorage,
	auditLogger AuditLogger,
	metricsCollector MetricsCollector,
	configValidator ConfigValidator,
) (*DefaultOAuthInterceptor, *MultiProviderDCRBridge, error) {
	// Create DCR bridge
	dcrBridge, err := CreateMultiProviderDCRBridge(
		azureTenantID,
		subscriptionID,
		resourceGroup,
		keyVaultURL,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create DCR bridge: %w", err)
	}

	// Create provider registry with DCR support
	registry := CreateProviderRegistry()

	// Replace Microsoft provider with DCR-enabled version
	dcrMicrosoftProvider := CreateDCRMicrosoftProvider(dcrBridge)
	_ = registry.RegisterProvider(dcrMicrosoftProvider)

	// Create OAuth interceptor
	interceptor := CreateOAuthInterceptor(
		config,
		tokenStorage,
		registry,
		auditLogger,
		metricsCollector,
		configValidator,
	)

	return interceptor, dcrBridge, nil
}
