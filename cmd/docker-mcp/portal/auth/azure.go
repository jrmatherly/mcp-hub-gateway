package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jrmatherly/mcp-hub-gateway/cmd/docker-mcp/portal/config"
)

// AzureADService implements Azure AD authentication
type AzureADService struct {
	config       *config.AzureConfig
	httpClient   *http.Client
	jwks         *JWKSProvider
	sessionStore SessionManager
}

// CreateAzureADService creates a new Azure AD authentication service
func CreateAzureADService(
	cfg *config.AzureConfig,
	sessionStore SessionManager,
) (*AzureADService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("azure configuration is required")
	}

	// Initialize HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Initialize JWKS provider
	jwksURL := fmt.Sprintf("%s/%s/discovery/v2.0/keys", cfg.Authority, cfg.TenantID)
	jwks := CreateJWKSProvider(jwksURL, httpClient)

	return &AzureADService{
		config:       cfg,
		httpClient:   httpClient,
		jwks:         jwks,
		sessionStore: sessionStore,
	}, nil
}

// ExchangeCode exchanges an authorization code for tokens
func (s *AzureADService) ExchangeCode(ctx context.Context, code string) (*TokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/%s/oauth2/v2.0/token",
		s.config.Authority,
		s.config.TenantID)

	// Prepare form data
	data := url.Values{}
	data.Set("client_id", s.config.ClientID)
	data.Set("client_secret", s.config.ClientSecret)
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", s.config.RedirectURL)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	// Parse response
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Create response
	return &TokenResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		IDToken:      tokenResp.IDToken,
		TokenType:    tokenResp.TokenType,
		ExpiresIn:    tokenResp.ExpiresIn,
		Scope:        tokenResp.Scope,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *AzureADService) RefreshToken(
	ctx context.Context,
	refreshToken string,
) (*TokenPair, error) {
	tokenURL := fmt.Sprintf("%s/%s/oauth2/v2.0/token",
		s.config.Authority,
		s.config.TenantID)

	// Prepare form data
	data := url.Values{}
	data.Set("client_id", s.config.ClientID)
	data.Set("client_secret", s.config.ClientSecret)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("scope", strings.Join(s.config.Scopes, " "))

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed: %s", string(body))
	}

	// Parse response
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode refresh response: %w", err)
	}

	// Create response
	return &TokenPair{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}, nil
}

// GetUserInfo retrieves user information from Azure AD using an access token
func (s *AzureADService) GetUserInfo(ctx context.Context, accessToken string) (*User, error) {
	// Microsoft Graph API endpoint for user info
	userInfoURL := "https://graph.microsoft.com/v1.0/me"

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("user info request failed: %s", string(body))
	}

	// Parse response
	var userInfo struct {
		ID                string `json:"id"`
		DisplayName       string `json:"displayName"`
		GivenName         string `json:"givenName"`
		Surname           string `json:"surname"`
		UserPrincipalName string `json:"userPrincipalName"`
		Mail              string `json:"mail"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Get user's groups/roles
	groups, err := s.getUserGroups(ctx, accessToken, userInfo.ID)
	if err != nil {
		// Log error but don't fail authentication
		groups = []string{}
	}

	// Create user object
	user := &User{
		ID:         uuid.New(),
		TenantID:   s.config.TenantID,
		AzureID:    userInfo.ID,
		Email:      userInfo.Mail,
		Name:       userInfo.DisplayName,
		GivenName:  userInfo.GivenName,
		FamilyName: userInfo.Surname,
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Claims:     make(map[string]any),
	}

	// Set role based on groups
	user.Role = s.determineUserRole(groups)

	// Set permissions based on role
	user.Permissions = s.getRolePermissions(user.Role)

	return user, nil
}

// getUserGroups retrieves the user's group memberships
func (s *AzureADService) getUserGroups(
	ctx context.Context,
	accessToken, userID string,
) ([]string, error) {
	groupsURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/memberOf", userID)

	req, err := http.NewRequestWithContext(ctx, "GET", groupsURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user groups: %d", resp.StatusCode)
	}

	var groupsResp struct {
		Value []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&groupsResp); err != nil {
		return nil, err
	}

	groups := make([]string, len(groupsResp.Value))
	for i, group := range groupsResp.Value {
		groups[i] = group.DisplayName
	}

	return groups, nil
}

// determineUserRole determines the user's role based on their group memberships
func (s *AzureADService) determineUserRole(groups []string) UserRole {
	// Check groups for role assignment
	for _, group := range groups {
		switch strings.ToLower(group) {
		case "mcp-admins", "administrators":
			return RoleAdmin
		case "mcp-superadmins", "super-administrators":
			return RoleSuperAdmin
		case "mcp-users", "users":
			return RoleUser
		}
	}

	// Default to guest role
	return RoleGuest
}

// getRolePermissions returns the permissions for a given role
func (s *AzureADService) getRolePermissions(role UserRole) []Permission {
	switch role {
	case RoleSuperAdmin, RoleSystemAdmin:
		return []Permission{
			PermissionServerList,
			PermissionServerEnable,
			PermissionServerDisable,
			PermissionServerInspect,
			PermissionGatewayRun,
			PermissionGatewayStop,
			PermissionGatewayConfig,
			PermissionSecretSet,
			PermissionSecretRemove,
			PermissionUserManage,
		}
	case RoleAdmin:
		return []Permission{
			PermissionServerList,
			PermissionServerEnable,
			PermissionServerDisable,
			PermissionServerInspect,
			PermissionGatewayRun,
			PermissionGatewayStop,
			PermissionGatewayConfig,
		}
	case RoleUser:
		return []Permission{
			PermissionServerList,
			PermissionServerInspect,
			PermissionGatewayRun,
		}
	default:
		return []Permission{
			PermissionServerList,
		}
	}
}

// ValidateIDToken validates an ID token using Azure AD's JWKS
func (s *AzureADService) ValidateIDToken(ctx context.Context, idToken string) (*Claims, error) {
	// Ensure JWKS cache is populated
	_, err := s.jwks.GetKeySet(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWKS: %w", err)
	}

	// Parse and validate the token
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(idToken, claims, func(token *jwt.Token) (any, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get the key ID from the token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid header not found")
		}

		// Find the key in the JWKS
		key := s.jwks.LookupKeyID(kid)
		if key == nil {
			return nil, fmt.Errorf("key not found in JWKS")
		}

		// Return the key
		return key, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	// Validate standard claims
	if err := claims.Valid(); err != nil {
		return nil, fmt.Errorf("invalid claims: %w", err)
	}

	// Validate issuer
	expectedIssuer := fmt.Sprintf("%s/%s/v2.0", s.config.Authority, s.config.TenantID)
	if claims.Issuer != expectedIssuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", expectedIssuer, claims.Issuer)
	}

	// Validate audience
	if !claims.VerifyAudience(s.config.ClientID, true) {
		return nil, fmt.Errorf("invalid audience")
	}

	return claims, nil
}

// RevokeToken revokes a refresh token
func (s *AzureADService) RevokeToken(ctx context.Context, refreshToken string) error {
	// Azure AD doesn't provide a token revocation endpoint in the standard OAuth2 flow
	// Instead, we invalidate the session in our session store
	return s.sessionStore.InvalidateByRefreshToken(ctx, refreshToken)
}

// GetUser retrieves a user by ID
func (s *AzureADService) GetUser(ctx context.Context, userID uuid.UUID) (*User, error) {
	// In a real implementation, this would fetch user data from Azure AD or local database
	// For now, return a basic implementation
	return &User{
		ID:    userID,
		Email: "user@example.com",
		Name:  "Example User",
		Role:  RoleUser,
	}, nil
}

// UpdateUser updates user information
func (s *AzureADService) UpdateUser(ctx context.Context, user *User) error {
	// In a real implementation, this would update user data in Azure AD or local database
	// For now, return nil (no-op)
	return nil
}

// DeactivateUser deactivates a user account
func (s *AzureADService) DeactivateUser(ctx context.Context, userID uuid.UUID) error {
	// In a real implementation, this would deactivate the user in Azure AD or local database
	// For now, return nil (no-op)
	return nil
}

// HasPermission checks if a user has a specific permission
func (s *AzureADService) HasPermission(
	ctx context.Context,
	userID uuid.UUID,
	permission Permission,
) (bool, error) {
	// Get user to determine their role
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user: %w", err)
	}

	// Get role permissions
	permissions := s.getRolePermissions(user.Role)

	// Check if the user has the required permission
	for _, perm := range permissions {
		if perm == permission {
			return true, nil
		}
	}

	return false, nil
}

// GetUserPermissions retrieves all permissions for a user
func (s *AzureADService) GetUserPermissions(
	ctx context.Context,
	userID uuid.UUID,
) ([]Permission, error) {
	// Get user to determine their role
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Return role permissions
	return s.getRolePermissions(user.Role), nil
}

// ValidateToken validates a JWT token and returns claims
func (s *AzureADService) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	// This is effectively the same as ValidateIDToken
	return s.ValidateIDToken(ctx, tokenString)
}

// GetAuthURL with redirect URI parameter to match interface
func (s *AzureADService) GetAuthURL(state string, redirectURI string) (string, error) {
	// Use the original GetAuthURL method but with the provided redirectURI if specified
	authURL := s.getAuthURLWithRedirect(state, redirectURI)
	return authURL, nil
}

// getAuthURLWithRedirect is the updated version that handles redirect URI
func (s *AzureADService) getAuthURLWithRedirect(state string, redirectURI string) string {
	authURL := fmt.Sprintf("%s/%s/oauth2/v2.0/authorize", s.config.Authority, s.config.TenantID)

	params := url.Values{}
	params.Add("client_id", s.config.ClientID)
	params.Add("response_type", "code")
	params.Add("scope", strings.Join(s.config.Scopes, " "))
	params.Add("state", state)

	// Use provided redirectURI or default
	if redirectURI != "" {
		params.Add("redirect_uri", redirectURI)
	} else {
		params.Add("redirect_uri", s.config.RedirectURL)
	}

	return authURL + "?" + params.Encode()
}

// HandleCallback handles OAuth2 callback and returns tokens and user info
func (s *AzureADService) HandleCallback(
	ctx context.Context,
	code, state, redirectURI string,
) (*TokenPair, *User, error) {
	// Exchange code for tokens
	tokenResponse, err := s.ExchangeCode(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info from access token
	user, err := s.GetUserInfo(ctx, tokenResponse.AccessToken)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Create token pair
	tokenPair := &TokenPair{
		AccessToken:  tokenResponse.AccessToken,
		RefreshToken: tokenResponse.RefreshToken,
		ExpiresAt:    tokenResponse.ExpiresAt,
		TokenType:    tokenResponse.TokenType,
	}

	return tokenPair, user, nil
}
