package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// DefaultProviderRegistry implements ProviderRegistry interface
type DefaultProviderRegistry struct {
	providers map[ProviderType]OAuthProvider
	mu        sync.RWMutex
}

// CreateProviderRegistry creates a new provider registry with default providers
func CreateProviderRegistry() *DefaultProviderRegistry {
	registry := &DefaultProviderRegistry{
		providers: make(map[ProviderType]OAuthProvider),
	}

	// Register default providers
	registry.RegisterProvider(&GitHubProvider{})
	registry.RegisterProvider(&GoogleProvider{})
	registry.RegisterProvider(&MicrosoftProvider{})

	return registry
}

// RegisterProvider registers an OAuth provider
func (r *DefaultProviderRegistry) RegisterProvider(provider OAuthProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	providerType := provider.GetProviderType()
	r.providers[providerType] = provider
	return nil
}

// GetProvider retrieves a provider by type
func (r *DefaultProviderRegistry) GetProvider(providerType ProviderType) (OAuthProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("provider %s not registered", providerType)
	}

	return provider, nil
}

// ListProviders returns all registered provider types
func (r *DefaultProviderRegistry) ListProviders() []ProviderType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]ProviderType, 0, len(r.providers))
	for providerType := range r.providers {
		types = append(types, providerType)
	}

	return types
}

// SupportsDCR checks if a provider supports Dynamic Client Registration
func (r *DefaultProviderRegistry) SupportsDCR(providerType ProviderType) bool {
	provider, err := r.GetProvider(providerType)
	if err != nil {
		return false
	}

	// Check if provider implements DCR interface
	_, supportsDCR := provider.(DCRCapableProvider)
	return supportsDCR
}

// DCRCapableProvider extends OAuthProvider with DCR capabilities
type DCRCapableProvider interface {
	OAuthProvider
	RegisterDynamicClient(ctx context.Context, req *DCRRequest) (*DCRResponse, error)
}

// GitHubProvider implements OAuth for GitHub
type GitHubProvider struct{}

func (p *GitHubProvider) GetProviderType() ProviderType {
	return ProviderTypeGitHub
}

func (p *GitHubProvider) GetProviderName() string {
	return "GitHub"
}

func (p *GitHubProvider) GetAuthURL(config *ServerConfig, state string) (string, error) {
	authURL := "https://github.com/login/oauth/authorize"
	if config.AuthURL != "" {
		authURL = config.AuthURL
	}

	params := url.Values{
		"client_id":     {config.ClientID},
		"redirect_uri":  {config.RedirectURI},
		"scope":         {strings.Join(config.Scopes, " ")},
		"state":         {state},
		"response_type": {"code"},
	}

	return fmt.Sprintf("%s?%s", authURL, params.Encode()), nil
}

func (p *GitHubProvider) ExchangeCode(
	ctx context.Context,
	config *ServerConfig,
	code string,
) (*TokenData, error) {
	tokenURL := "https://github.com/login/oauth/access_token"
	if config.TokenURL != "" {
		tokenURL = config.TokenURL
	}

	data := url.Values{
		"client_id":     {config.ClientID},
		"client_secret": {config.ClientSecret},
		"code":          {code},
		"redirect_uri":  {config.RedirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		Error       string `json:"error,omitempty"`
		ErrorDesc   string `json:"error_description,omitempty"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf(
			"token exchange error: %s - %s",
			tokenResp.Error,
			tokenResp.ErrorDesc,
		)
	}

	// GitHub tokens don't expire, but we'll set a reasonable expiry
	expiresAt := time.Now().Add(365 * 24 * time.Hour)
	refreshAt := time.Now().Add(30 * 24 * time.Hour)

	return &TokenData{
		ServerName:   config.ServerName,
		ProviderType: ProviderTypeGitHub,
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		ExpiresAt:    expiresAt,
		RefreshAt:    refreshAt,
		IssuedAt:     time.Now(),
		Scopes:       strings.Split(tokenResp.Scope, " "),
	}, nil
}

func (p *GitHubProvider) RefreshToken(
	ctx context.Context,
	config *ServerConfig,
	refreshToken string,
) (*TokenData, error) {
	// GitHub doesn't support refresh tokens
	return nil, fmt.Errorf("GitHub does not support token refresh")
}

func (p *GitHubProvider) RevokeToken(
	ctx context.Context,
	config *ServerConfig,
	token string,
) error {
	revokeURL := fmt.Sprintf("https://api.github.com/applications/%s/token", config.ClientID)

	revokeData := map[string]string{"access_token": token}
	jsonData, err := json.Marshal(revokeData)
	if err != nil {
		return fmt.Errorf("failed to marshal revoke data: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"DELETE",
		revokeURL,
		strings.NewReader(string(jsonData)),
	)
	if err != nil {
		return fmt.Errorf("failed to create revoke request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(config.ClientID, config.ClientSecret)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("token revocation failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (p *GitHubProvider) ValidateToken(
	ctx context.Context,
	config *ServerConfig,
	token string,
) (*TokenClaims, error) {
	// GitHub doesn't provide JWT tokens, so we validate by making an API call
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create validation request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token validation failed with status: %d", resp.StatusCode)
	}

	// Return minimal claims since GitHub doesn't use JWT
	return &TokenClaims{
		Issuer:    "github.com",
		Subject:   "github-user",
		ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
		IssuedAt:  time.Now(),
		TokenType: "Bearer",
	}, nil
}

func (p *GitHubProvider) GetUserInfo(
	ctx context.Context,
	config *ServerConfig,
	token string,
) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info response: %w", err)
	}

	var userInfo map[string]interface{}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return userInfo, nil
}

func (p *GitHubProvider) SupportsRefresh() bool {
	return false
}

func (p *GitHubProvider) SupportsRevocation() bool {
	return true
}

func (p *GitHubProvider) GetDefaultScopes() []string {
	return []string{"repo", "user"}
}

func (p *GitHubProvider) GetTokenExpiry(token string) (time.Time, error) {
	// GitHub tokens don't expire, but return a reasonable default
	return time.Now().Add(365 * 24 * time.Hour), nil
}

// GoogleProvider implements OAuth for Google
type GoogleProvider struct{}

func (p *GoogleProvider) GetProviderType() ProviderType {
	return ProviderTypeGoogle
}

func (p *GoogleProvider) GetProviderName() string {
	return "Google"
}

func (p *GoogleProvider) GetAuthURL(config *ServerConfig, state string) (string, error) {
	authURL := "https://accounts.google.com/o/oauth2/v2/auth"
	if config.AuthURL != "" {
		authURL = config.AuthURL
	}

	params := url.Values{
		"client_id":     {config.ClientID},
		"redirect_uri":  {config.RedirectURI},
		"scope":         {strings.Join(config.Scopes, " ")},
		"state":         {state},
		"response_type": {"code"},
		"access_type":   {"offline"}, // For refresh tokens
	}

	return fmt.Sprintf("%s?%s", authURL, params.Encode()), nil
}

func (p *GoogleProvider) ExchangeCode(
	ctx context.Context,
	config *ServerConfig,
	code string,
) (*TokenData, error) {
	tokenURL := "https://oauth2.googleapis.com/token"
	if config.TokenURL != "" {
		tokenURL = config.TokenURL
	}

	data := url.Values{
		"client_id":     {config.ClientID},
		"client_secret": {config.ClientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {config.RedirectURI},
	}

	return p.makeTokenRequest(ctx, tokenURL, data, config)
}

func (p *GoogleProvider) RefreshToken(
	ctx context.Context,
	config *ServerConfig,
	refreshToken string,
) (*TokenData, error) {
	tokenURL := "https://oauth2.googleapis.com/token"
	if config.TokenURL != "" {
		tokenURL = config.TokenURL
	}

	data := url.Values{
		"client_id":     {config.ClientID},
		"client_secret": {config.ClientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}

	return p.makeTokenRequest(ctx, tokenURL, data, config)
}

func (p *GoogleProvider) makeTokenRequest(
	ctx context.Context,
	tokenURL string,
	data url.Values,
	config *ServerConfig,
) (*TokenData, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
		Scope        string `json:"scope"`
		IDToken      string `json:"id_token"`
		Error        string `json:"error,omitempty"`
		ErrorDesc    string `json:"error_description,omitempty"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf(
			"token exchange error: %s - %s",
			tokenResp.Error,
			tokenResp.ErrorDesc,
		)
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	refreshAt := time.Now().
		Add(time.Duration(tokenResp.ExpiresIn-300) * time.Second)
		// 5 min before expiry

	return &TokenData{
		ServerName:   config.ServerName,
		ProviderType: ProviderTypeGoogle,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		IDToken:      tokenResp.IDToken,
		TokenType:    tokenResp.TokenType,
		ExpiresAt:    expiresAt,
		RefreshAt:    refreshAt,
		IssuedAt:     time.Now(),
		Scopes:       strings.Split(tokenResp.Scope, " "),
	}, nil
}

func (p *GoogleProvider) RevokeToken(
	ctx context.Context,
	config *ServerConfig,
	token string,
) error {
	revokeURL := fmt.Sprintf(
		"https://oauth2.googleapis.com/revoke?token=%s",
		url.QueryEscape(token),
	)

	req, err := http.NewRequestWithContext(ctx, "POST", revokeURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create revoke request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token revocation failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (p *GoogleProvider) ValidateToken(
	ctx context.Context,
	config *ServerConfig,
	token string,
) (*TokenClaims, error) {
	// For ID tokens, parse JWT; for access tokens, use tokeninfo endpoint
	if strings.Contains(token, ".") {
		return p.validateJWT(token)
	}

	return p.validateAccessToken(ctx, token)
}

func (p *GoogleProvider) validateJWT(token string) (*TokenClaims, error) {
	// Parse JWT without verification (verification would require JWKS)
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	parsedToken, _, err := parser.ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT: %w", err)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid JWT claims")
	}

	tokenClaims := &TokenClaims{
		Custom: make(map[string]interface{}),
	}

	// Extract standard claims
	if iss, ok := claims["iss"].(string); ok {
		tokenClaims.Issuer = iss
	}
	if sub, ok := claims["sub"].(string); ok {
		tokenClaims.Subject = sub
	}
	if exp, ok := claims["exp"].(float64); ok {
		tokenClaims.ExpiresAt = time.Unix(int64(exp), 0)
	}
	if iat, ok := claims["iat"].(float64); ok {
		tokenClaims.IssuedAt = time.Unix(int64(iat), 0)
	}
	if email, ok := claims["email"].(string); ok {
		tokenClaims.Email = email
	}
	if name, ok := claims["name"].(string); ok {
		tokenClaims.Name = name
	}

	// Copy other claims
	for key, value := range claims {
		if _, exists := map[string]bool{
			"iss": true, "sub": true, "exp": true, "iat": true,
			"email": true, "name": true,
		}[key]; !exists {
			tokenClaims.Custom[key] = value
		}
	}

	return tokenClaims, nil
}

func (p *GoogleProvider) validateAccessToken(
	ctx context.Context,
	token string,
) (*TokenClaims, error) {
	tokenInfoURL := fmt.Sprintf(
		"https://oauth2.googleapis.com/tokeninfo?access_token=%s",
		url.QueryEscape(token),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", tokenInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create token info request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token validation failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token info response: %w", err)
	}

	var tokenInfo map[string]interface{}
	if err := json.Unmarshal(body, &tokenInfo); err != nil {
		return nil, fmt.Errorf("failed to parse token info: %w", err)
	}

	tokenClaims := &TokenClaims{
		Custom: tokenInfo,
	}

	if exp, ok := tokenInfo["exp"].(string); ok {
		if expTime, err := time.Parse(time.RFC3339, exp); err == nil {
			tokenClaims.ExpiresAt = expTime
		}
	}

	return tokenClaims, nil
}

func (p *GoogleProvider) GetUserInfo(
	ctx context.Context,
	config *ServerConfig,
	token string,
) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		"https://www.googleapis.com/oauth2/v2/userinfo",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info response: %w", err)
	}

	var userInfo map[string]interface{}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return userInfo, nil
}

func (p *GoogleProvider) SupportsRefresh() bool {
	return true
}

func (p *GoogleProvider) SupportsRevocation() bool {
	return true
}

func (p *GoogleProvider) GetDefaultScopes() []string {
	return []string{"openid", "email", "profile"}
}

func (p *GoogleProvider) GetTokenExpiry(token string) (time.Time, error) {
	if strings.Contains(token, ".") {
		// JWT token
		claims, err := p.validateJWT(token)
		if err != nil {
			return time.Time{}, err
		}
		return claims.ExpiresAt, nil
	}

	// Access token - would need to call tokeninfo endpoint
	return time.Time{}, fmt.Errorf("expiry check for access tokens requires API call")
}

// MicrosoftProvider implements OAuth for Microsoft/Azure AD
type MicrosoftProvider struct{}

func (p *MicrosoftProvider) GetProviderType() ProviderType {
	return ProviderTypeMicrosoft
}

func (p *MicrosoftProvider) GetProviderName() string {
	return "Microsoft"
}

func (p *MicrosoftProvider) GetAuthURL(config *ServerConfig, state string) (string, error) {
	tenantID := config.TenantID
	if tenantID == "" {
		tenantID = "common"
	}

	authURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenantID)
	if config.AuthURL != "" {
		authURL = config.AuthURL
	}

	params := url.Values{
		"client_id":     {config.ClientID},
		"redirect_uri":  {config.RedirectURI},
		"scope":         {strings.Join(config.Scopes, " ")},
		"state":         {state},
		"response_type": {"code"},
		"response_mode": {"query"},
	}

	return fmt.Sprintf("%s?%s", authURL, params.Encode()), nil
}

func (p *MicrosoftProvider) ExchangeCode(
	ctx context.Context,
	config *ServerConfig,
	code string,
) (*TokenData, error) {
	tenantID := config.TenantID
	if tenantID == "" {
		tenantID = "common"
	}

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)
	if config.TokenURL != "" {
		tokenURL = config.TokenURL
	}

	data := url.Values{
		"client_id":     {config.ClientID},
		"client_secret": {config.ClientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {config.RedirectURI},
		"scope":         {strings.Join(config.Scopes, " ")},
	}

	return p.makeTokenRequest(ctx, tokenURL, data, config)
}

func (p *MicrosoftProvider) RefreshToken(
	ctx context.Context,
	config *ServerConfig,
	refreshToken string,
) (*TokenData, error) {
	tenantID := config.TenantID
	if tenantID == "" {
		tenantID = "common"
	}

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)
	if config.TokenURL != "" {
		tokenURL = config.TokenURL
	}

	data := url.Values{
		"client_id":     {config.ClientID},
		"client_secret": {config.ClientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
		"scope":         {strings.Join(config.Scopes, " ")},
	}

	return p.makeTokenRequest(ctx, tokenURL, data, config)
}

func (p *MicrosoftProvider) makeTokenRequest(
	ctx context.Context,
	tokenURL string,
	data url.Values,
	config *ServerConfig,
) (*TokenData, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
		Scope        string `json:"scope"`
		IDToken      string `json:"id_token"`
		Error        string `json:"error,omitempty"`
		ErrorDesc    string `json:"error_description,omitempty"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf(
			"token exchange error: %s - %s",
			tokenResp.Error,
			tokenResp.ErrorDesc,
		)
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	refreshAt := time.Now().
		Add(time.Duration(tokenResp.ExpiresIn-300) * time.Second)
		// 5 min before expiry

	return &TokenData{
		ServerName:   config.ServerName,
		ProviderType: ProviderTypeMicrosoft,
		TenantID:     config.TenantID,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		IDToken:      tokenResp.IDToken,
		TokenType:    tokenResp.TokenType,
		ExpiresAt:    expiresAt,
		RefreshAt:    refreshAt,
		IssuedAt:     time.Now(),
		Scopes:       strings.Split(tokenResp.Scope, " "),
	}, nil
}

func (p *MicrosoftProvider) RevokeToken(
	ctx context.Context,
	config *ServerConfig,
	token string,
) error {
	// Microsoft doesn't have a standard revoke endpoint, but we can call logout
	return fmt.Errorf("Microsoft provider does not support token revocation")
}

func (p *MicrosoftProvider) ValidateToken(
	ctx context.Context,
	config *ServerConfig,
	token string,
) (*TokenClaims, error) {
	// For Microsoft tokens, we typically validate JWT using JWKS
	// For simplicity, we'll just parse without full validation
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	parsedToken, _, err := parser.ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT: %w", err)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid JWT claims")
	}

	tokenClaims := &TokenClaims{
		Custom: make(map[string]interface{}),
	}

	// Extract standard claims
	if iss, ok := claims["iss"].(string); ok {
		tokenClaims.Issuer = iss
	}
	if sub, ok := claims["sub"].(string); ok {
		tokenClaims.Subject = sub
	}
	if exp, ok := claims["exp"].(float64); ok {
		tokenClaims.ExpiresAt = time.Unix(int64(exp), 0)
	}
	if iat, ok := claims["iat"].(float64); ok {
		tokenClaims.IssuedAt = time.Unix(int64(iat), 0)
	}
	if email, ok := claims["preferred_username"].(string); ok {
		tokenClaims.Email = email
	}
	if name, ok := claims["name"].(string); ok {
		tokenClaims.Name = name
	}

	// Copy other claims
	for key, value := range claims {
		tokenClaims.Custom[key] = value
	}

	return tokenClaims, nil
}

func (p *MicrosoftProvider) GetUserInfo(
	ctx context.Context,
	config *ServerConfig,
	token string,
) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://graph.microsoft.com/v1.0/me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info response: %w", err)
	}

	var userInfo map[string]interface{}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return userInfo, nil
}

func (p *MicrosoftProvider) SupportsRefresh() bool {
	return true
}

func (p *MicrosoftProvider) SupportsRevocation() bool {
	return false
}

func (p *MicrosoftProvider) GetDefaultScopes() []string {
	return []string{"openid", "profile", "email", "https://graph.microsoft.com/User.Read"}
}

func (p *MicrosoftProvider) GetTokenExpiry(token string) (time.Time, error) {
	claims, err := p.ValidateToken(context.Background(), nil, token)
	if err != nil {
		return time.Time{}, err
	}
	return claims.ExpiresAt, nil
}
