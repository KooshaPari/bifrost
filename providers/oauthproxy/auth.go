// Package oauthproxy - OAuth authentication for various providers
package oauthproxy

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// PKCECodes holds PKCE verification codes for OAuth2 PKCE flow
type PKCECodes struct {
	CodeVerifier  string `json:"code_verifier"`
	CodeChallenge string `json:"code_challenge"`
}

// GeneratePKCE generates PKCE codes for OAuth2 flow
func GeneratePKCE() (*PKCECodes, error) {
	verifier := make([]byte, 32)
	if _, err := rand.Read(verifier); err != nil {
		return nil, fmt.Errorf("failed to generate verifier: %w", err)
	}

	codeVerifier := base64.RawURLEncoding.EncodeToString(verifier)
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return &PKCECodes{
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
	}, nil
}

// TokenData holds OAuth token information
type TokenData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	IDToken      string    `json:"id_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	Email        string    `json:"email,omitempty"`
	AccountID    string    `json:"account_id,omitempty"`
}

// IsExpired checks if the token is expired
func (t *TokenData) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// NeedsRefresh checks if the token needs refresh (within 5 minutes of expiry)
func (t *TokenData) NeedsRefresh() bool {
	return time.Now().Add(5 * time.Minute).After(t.ExpiresAt)
}

// OAuthProvider defines the interface for OAuth providers
type OAuthProvider interface {
	GetName() string
	GenerateAuthURL(state string, pkce *PKCECodes) (string, error)
	ExchangeCode(ctx context.Context, code string, pkce *PKCECodes) (*TokenData, error)
	RefreshToken(ctx context.Context, refreshToken string) (*TokenData, error)
	GetAuthHeader(token *TokenData) string
}

// ProviderConfig holds OAuth provider configuration
type ProviderConfig struct {
	ClientID    string
	AuthURL     string
	TokenURL    string
	RedirectURI string
	Scopes      []string
}

// ClaudeOAuth implements OAuth for Anthropic Claude
type ClaudeOAuth struct {
	config     ProviderConfig
	httpClient *http.Client
}

// NewClaudeOAuth creates a new Claude OAuth provider
func NewClaudeOAuth() *ClaudeOAuth {
	return &ClaudeOAuth{
		config: ProviderConfig{
			ClientID:    "9d1c250a-e61b-44d9-88ed-5944d1962f5e",
			AuthURL:     "https://claude.ai/oauth/authorize",
			TokenURL:    "https://console.anthropic.com/v1/oauth/token",
			RedirectURI: "http://localhost:54545/callback",
			Scopes:      []string{"org:create_api_key", "user:profile", "user:inference"},
		},
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *ClaudeOAuth) GetName() string { return "claude" }

func (c *ClaudeOAuth) GenerateAuthURL(state string, pkce *PKCECodes) (string, error) {
	params := url.Values{
		"code":                  {"true"},
		"client_id":             {c.config.ClientID},
		"response_type":         {"code"},
		"redirect_uri":          {c.config.RedirectURI},
		"scope":                 {strings.Join(c.config.Scopes, " ")},
		"code_challenge":        {pkce.CodeChallenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
	}
	return fmt.Sprintf("%s?%s", c.config.AuthURL, params.Encode()), nil
}

func (c *ClaudeOAuth) ExchangeCode(ctx context.Context, code string, pkce *PKCECodes) (*TokenData, error) {
	reqBody := map[string]interface{}{
		"code":          code,
		"grant_type":    "authorization_code",
		"client_id":     c.config.ClientID,
		"redirect_uri":  c.config.RedirectURI,
		"code_verifier": pkce.CodeVerifier,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", c.config.TokenURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &TokenData{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}, nil
}

func (c *ClaudeOAuth) RefreshToken(ctx context.Context, refreshToken string) (*TokenData, error) {
	reqBody := map[string]interface{}{
		"grant_type":    "refresh_token",
		"client_id":     c.config.ClientID,
		"refresh_token": refreshToken,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", c.config.TokenURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &TokenData{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}, nil
}

func (c *ClaudeOAuth) GetAuthHeader(token *TokenData) string {
	return "Bearer " + token.AccessToken
}

// CodexOAuth implements OAuth for OpenAI Codex
type CodexOAuth struct {
	config     ProviderConfig
	httpClient *http.Client
}

// NewCodexOAuth creates a new Codex OAuth provider
func NewCodexOAuth() *CodexOAuth {
	return &CodexOAuth{
		config: ProviderConfig{
			ClientID:    "app_sso_codex_cli",
			AuthURL:     "https://auth.openai.com/authorize",
			TokenURL:    "https://auth.openai.com/oauth/token",
			RedirectURI: "http://localhost:54545/callback",
			Scopes:      []string{"openid", "profile", "email", "offline_access"},
		},
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *CodexOAuth) GetName() string { return "codex" }

func (c *CodexOAuth) GenerateAuthURL(state string, pkce *PKCECodes) (string, error) {
	params := url.Values{
		"client_id":             {c.config.ClientID},
		"response_type":         {"code"},
		"redirect_uri":          {c.config.RedirectURI},
		"scope":                 {strings.Join(c.config.Scopes, " ")},
		"code_challenge":        {pkce.CodeChallenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
		"audience":              {"https://api.openai.com/v1"},
	}
	return fmt.Sprintf("%s?%s", c.config.AuthURL, params.Encode()), nil
}

func (c *CodexOAuth) ExchangeCode(ctx context.Context, code string, pkce *PKCECodes) (*TokenData, error) {
	data := url.Values{
		"client_id":     {c.config.ClientID},
		"code":          {code},
		"code_verifier": {pkce.CodeVerifier},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {c.config.RedirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &TokenData{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}, nil
}

func (c *CodexOAuth) RefreshToken(ctx context.Context, refreshToken string) (*TokenData, error) {
	data := url.Values{
		"client_id":     {c.config.ClientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &TokenData{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}, nil
}

func (c *CodexOAuth) GetAuthHeader(token *TokenData) string {
	return "Bearer " + token.AccessToken
}

// OAuthManager manages OAuth tokens for multiple providers
type OAuthManager struct {
	providers map[string]OAuthProvider
	mu        sync.RWMutex
	cacheDir  string
}

// NewOAuthManager creates a new OAuthManager
func NewOAuthManager(cacheDir string) *OAuthManager {
	return &OAuthManager{
		providers: make(map[string]OAuthProvider),
		cacheDir:  cacheDir,
	}
}

// RegisterProvider registers an OAuth provider
func (m *OAuthManager) RegisterProvider(provider OAuthProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[provider.GetName()] = provider
}

// LoadTokens loads cached tokens from disk
func (m *OAuthManager) LoadTokens() (map[string]*TokenData, error) {
	tokens := make(map[string]*TokenData)
	tokenFile := filepath.Join(m.cacheDir, "oauth_tokens.json")

	data, err := os.ReadFile(tokenFile)
	if err != nil {
		if os.IsNotExist(err) {
			return tokens, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, err
	}

	return tokens, nil
}

// SaveTokens saves tokens to disk
func (m *OAuthManager) SaveTokens(tokens map[string]*TokenData) error {
	_ = os.MkdirAll(m.cacheDir, 0755)
	tokenFile := filepath.Join(m.cacheDir, "oauth_tokens.json")

	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tokenFile, data, 0600)
}
