package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// APIClient is the central API client for all gateway operations
type APIClient struct {
	baseURL    string
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SetBaseURL updates the base URL
func (c *APIClient) SetBaseURL(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.baseURL = url
}

// GetBaseURL returns the current base URL
func (c *APIClient) GetBaseURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.baseURL
}

// --- Health Check ---

// IsAPIAvailable checks if the API is reachable
func (c *APIClient) IsAPIAvailable() bool {
	url := fmt.Sprintf("%s/health", c.GetBaseURL())
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// --- Service Discovery ---

// ServiceDiscoveryResponse wraps the services response
type ServiceDiscoveryResponse struct {
	Services []ServiceInfo `json:"services"`
	Count    int           `json:"count"`
}

// ServiceInfo represents a discovered service
type ServiceInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	DisplayName   string `json:"display_name"`
	Icon          string `json:"icon,omitempty"`
	Provider      string `json:"provider"`
	IsConfigBased bool   `json:"is_config_based"`
	ModelCount    int    `json:"model_count"`
	Available     bool   `json:"available"`
}

// GetAvailableServices fetches all available services from the API
func (c *APIClient) GetAvailableServices() ([]ServiceInfo, error) {
	url := fmt.Sprintf("%s/api/v1/services", c.GetBaseURL())
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var response ServiceDiscoveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response.Services, nil
}

// GetService fetches a specific service by ID
func (c *APIClient) GetService(id string) (*ServiceInfo, error) {
	url := fmt.Sprintf("%s/api/v1/services/%s", c.GetBaseURL(), id)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("service not found")
	}

	var service ServiceInfo
	if err := json.NewDecoder(resp.Body).Decode(&service); err != nil {
		return nil, err
	}
	return &service, nil
}

// --- Model Discovery ---

// ModelInfoAPI represents a model from the API
type ModelInfoAPI struct {
	Name              string `json:"name"`
	DisplayName       string `json:"displayName,omitempty"`
	Description       string `json:"description,omitempty"`
	MaxTokens         int    `json:"maxTokens,omitempty"`
	ContextWindow     int    `json:"contextWindow,omitempty"`
	SupportsStreaming bool   `json:"supportsStreaming,omitempty"`
}

// GetAvailableModels fetches models for a specific service
func (c *APIClient) GetAvailableModels(serviceID string) ([]ModelInfoAPI, error) {
	url := fmt.Sprintf("%s/api/services/%s/models", c.GetBaseURL(), serviceID)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch models")
	}

	var models []ModelInfoAPI
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, err
	}
	return models, nil
}

// --- Gateway Provider Management ---

// ProviderTypeInfoAPI represents metadata about a provider type
type ProviderTypeInfoAPI struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	AuthHeader  string `json:"authHeader"`
	AuthPrefix  string `json:"authPrefix"`
	BaseURL     string `json:"baseURL"`
	DocURL      string `json:"docURL"`
}

// GatewayProviderAPI represents a configured gateway provider
type GatewayProviderAPI struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	BaseURL    string            `json:"baseURL"`
	APIKey     string            `json:"apiKey,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Models     []string          `json:"models,omitempty"`
	IsEnabled  bool              `json:"isEnabled"`
	CreatedAt  string            `json:"createdAt,omitempty"`
	ModifiedAt string            `json:"modifiedAt,omitempty"`
}

// FetchProviderTypes fetches all available provider types
func (c *APIClient) FetchProviderTypes() ([]ProviderTypeInfoAPI, error) {
	url := fmt.Sprintf("%s/v0/management/gateway/provider-types", c.GetBaseURL())
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch provider types")
	}

	var response struct {
		Providers []ProviderTypeInfoAPI `json:"providers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response.Providers, nil
}

// FetchProviderModels fetches available models for a provider type
func (c *APIClient) FetchProviderModels(providerType string) ([]ModelInfoAPI, error) {
	url := fmt.Sprintf("%s/v0/management/gateway/provider-models?type=%s", c.GetBaseURL(), providerType)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch provider models")
	}

	var response struct {
		Models []ModelInfoAPI `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response.Models, nil
}

// FetchGatewayProviders fetches all configured gateway providers
func (c *APIClient) FetchGatewayProviders() ([]GatewayProviderAPI, error) {
	url := fmt.Sprintf("%s/v0/management/gateway/providers", c.GetBaseURL())
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch gateway providers")
	}

	var response struct {
		Providers []GatewayProviderAPI `json:"providers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response.Providers, nil
}

// AddGatewayProvider adds a new gateway provider
func (c *APIClient) AddGatewayProvider(provider GatewayProviderAPI) error {
	url := fmt.Sprintf("%s/v0/management/gateway/providers", c.GetBaseURL())

	body := struct {
		Provider GatewayProviderAPI `json:"provider"`
	}{Provider: provider}

	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add provider: status %d", resp.StatusCode)
	}
	return nil
}

// UpdateGatewayProvider updates an existing gateway provider
func (c *APIClient) UpdateGatewayProvider(provider GatewayProviderAPI) error {
	url := fmt.Sprintf("%s/v0/management/gateway/providers", c.GetBaseURL())

	data, err := json.Marshal(provider)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update provider: status %d", resp.StatusCode)
	}
	return nil
}

// DeleteGatewayProvider deletes a gateway provider
func (c *APIClient) DeleteGatewayProvider(name string) error {
	url := fmt.Sprintf("%s/v0/management/gateway/providers?name=%s", c.GetBaseURL(), name)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete provider: status %d", resp.StatusCode)
	}
	return nil
}

// --- Authentication ---

// AuthenticateService initiates authentication for a service
func (c *APIClient) AuthenticateService(serviceID string) (bool, string) {
	url := fmt.Sprintf("%s/api/v1/auth/%s", c.GetBaseURL(), serviceID)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return false, err.Error()
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, fmt.Sprintf("✓ %s authenticated successfully", serviceID)
	}
	return false, fmt.Sprintf("✗ Authentication failed with status %d", resp.StatusCode)
}

// --- Learning System ---

// GetGeneratedRules fetches generated rules from the learning system
func (c *APIClient) GetGeneratedRules() (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v1/learning/rules", c.GetBaseURL())
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch rules")
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetRecommendedModels fetches recommended models from the learning system
func (c *APIClient) GetRecommendedModels() ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v1/learning/recommendations", c.GetBaseURL())
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch recommendations")
	}

	var result struct {
		Recommendations []map[string]interface{} `json:"recommendations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Recommendations, nil
}

// --- Helper ---

