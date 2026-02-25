package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// ServiceProvider represents a configured LLM provider
type ServiceProvider struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	DisplayName string            `json:"display_name"`
	BaseURL     string            `json:"base_url"`
	APIKey      string            `json:"api_key,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	IsConnected bool              `json:"is_connected"`
	Models      []ProviderModel   `json:"models,omitempty"`
}

// ProviderModel represents a model offered by a provider
type ProviderModel struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description,omitempty"`
}

// ProviderTypeInfo represents metadata about a supported provider type
type ProviderTypeInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	BaseURL     string `json:"base_url"`
	AuthHeader  string `json:"auth_header"`
	AuthPrefix  string `json:"auth_prefix"`
	DocURL      string `json:"doc_url"`
}

// ServiceDiscoveryInfo represents a discovered service
type ServiceDiscoveryInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	DisplayName   string `json:"display_name"`
	Icon          string `json:"icon,omitempty"`
	Provider      string `json:"provider"`
	Available     bool   `json:"available"`
	IsConfigBased bool   `json:"is_config_based"`
	ModelCount    int    `json:"model_count"`
}

// ProviderManager manages service providers with caching and periodic refresh
type ProviderManager struct {
	mu              sync.RWMutex
	apiClient       *APIClient
	providers       []ServiceProvider
	providerTypes   []ProviderTypeInfo
	services        []ServiceDiscoveryInfo
	cachedModels    map[string][]ProviderModel
	lastUpdated     time.Time
	lastModelFetch  map[string]time.Time
	isLoading       bool
	cacheTTL        time.Duration
	refreshInterval time.Duration
	stopRefresh     chan struct{}
	onChange        func() // Callback when data changes
}

// NewProviderManager creates a new provider manager with API client
func NewProviderManager(gatewayURL string) *ProviderManager {
	pm := &ProviderManager{
		apiClient:       NewAPIClient(gatewayURL),
		providers:       []ServiceProvider{},
		providerTypes:   []ProviderTypeInfo{},
		services:        []ServiceDiscoveryInfo{},
		cachedModels:    make(map[string][]ProviderModel),
		lastModelFetch:  make(map[string]time.Time),
		cacheTTL:        time.Hour,        // 1 hour cache TTL
		refreshInterval: 5 * time.Minute,  // 5 minute refresh interval
		stopRefresh:     make(chan struct{}),
	}
	pm.setDefaultProviderTypes()
	pm.setDefaultServices()
	return pm
}

// SetOnChange sets the callback for data changes
func (pm *ProviderManager) SetOnChange(callback func()) {
	pm.onChange = callback
}

// GetProviders returns all configured providers
func (pm *ProviderManager) GetProviders() []ServiceProvider {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.providers
}

// GetProviderTypes returns all available provider types
func (pm *ProviderManager) GetProviderTypes() []ProviderTypeInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.providerTypes
}

// GetServices returns all discovered services
func (pm *ProviderManager) GetServices() []ServiceDiscoveryInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.services
}

// IsLoading returns whether the manager is currently loading
func (pm *ProviderManager) IsLoading() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.isLoading
}

// IsCacheValid checks if the cache is still valid
func (pm *ProviderManager) IsCacheValid() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return time.Since(pm.lastUpdated) < pm.cacheTTL
}

// StartPeriodicRefresh starts the background refresh goroutine
func (pm *ProviderManager) StartPeriodicRefresh() {
	go func() {
		ticker := time.NewTicker(pm.refreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				log.Println("[ProviderManager] Periodic refresh triggered")
				pm.DiscoverServices(false)
			case <-pm.stopRefresh:
				log.Println("[ProviderManager] Stopping periodic refresh")
				return
			}
		}
	}()
	log.Printf("[ProviderManager] Started periodic refresh every %v", pm.refreshInterval)
}

// StopPeriodicRefresh stops the background refresh
func (pm *ProviderManager) StopPeriodicRefresh() {
	close(pm.stopRefresh)
}

// setDefaultServices sets default service providers
func (pm *ProviderManager) setDefaultServices() {
	pm.services = []ServiceDiscoveryInfo{
		{ID: "anthropic", DisplayName: "Anthropic (Claude)", Icon: "claude", Available: true},
		{ID: "openai", DisplayName: "OpenAI (GPT)", Icon: "openai", Available: true},
		{ID: "google", DisplayName: "Google (Gemini)", Icon: "gemini", Available: true},
		{ID: "local-slm", DisplayName: "Local SLM", Icon: "local", Available: true, IsConfigBased: true},
	}
	pm.lastUpdated = time.Now()
}

// setDefaultProviderTypes sets default provider types
func (pm *ProviderManager) setDefaultProviderTypes() {
	pm.providerTypes = []ProviderTypeInfo{
		{Name: "anthropic", DisplayName: "Anthropic", Description: "Claude AI models", BaseURL: "https://api.anthropic.com", AuthHeader: "x-api-key", AuthPrefix: "", DocURL: "https://docs.anthropic.com"},
		{Name: "openai", DisplayName: "OpenAI", Description: "GPT models", BaseURL: "https://api.openai.com", AuthHeader: "Authorization", AuthPrefix: "Bearer ", DocURL: "https://platform.openai.com/docs"},
		{Name: "google", DisplayName: "Google AI", Description: "Gemini models", BaseURL: "https://generativelanguage.googleapis.com", AuthHeader: "x-goog-api-key", AuthPrefix: "", DocURL: "https://ai.google.dev/docs"},
		{Name: "openrouter", DisplayName: "OpenRouter", Description: "Multi-provider gateway", BaseURL: "https://openrouter.ai/api", AuthHeader: "Authorization", AuthPrefix: "Bearer ", DocURL: "https://openrouter.ai/docs"},
		{Name: "ollama", DisplayName: "Ollama", Description: "Local LLM server", BaseURL: "http://localhost:11434", AuthHeader: "", AuthPrefix: "", DocURL: "https://ollama.ai/docs"},
	}
}

// FetchProviderTypes fetches provider types from API with fallback
func (pm *ProviderManager) FetchProviderTypes() error {
	types, err := pm.apiClient.FetchProviderTypes()
	if err != nil {
		log.Printf("[ProviderManager] Failed to fetch provider types: %v, using defaults", err)
		return err
	}

	pm.mu.Lock()
	pm.providerTypes = make([]ProviderTypeInfo, len(types))
	for i, t := range types {
		pm.providerTypes[i] = ProviderTypeInfo{
			Name:        t.Name,
			DisplayName: t.DisplayName,
			BaseURL:     t.BaseURL,
			AuthHeader:  t.AuthHeader,
			AuthPrefix:  t.AuthPrefix,
			DocURL:      t.DocURL,
		}
	}
	pm.mu.Unlock()

	log.Printf("[ProviderManager] Fetched %d provider types", len(types))
	return nil
}

// DiscoverServices discovers available services from the gateway
// forceRefresh bypasses the cache
func (pm *ProviderManager) DiscoverServices(forceRefresh bool) error {
	// Check cache validity
	if !forceRefresh && pm.IsCacheValid() && len(pm.services) > 0 {
		log.Println("[ProviderManager] Using cached services")
		return nil
	}

	pm.mu.Lock()
	pm.isLoading = true
	pm.mu.Unlock()

	defer func() {
		pm.mu.Lock()
		pm.isLoading = false
		pm.mu.Unlock()
	}()

	log.Println("[ProviderManager] Fetching services from API...")

	// Try to fetch from API
	services, err := pm.apiClient.GetAvailableServices()
	if err != nil {
		log.Printf("[ProviderManager] API fetch failed: %v, using fallback", err)
		// Keep existing services if we have them (fallback to cache)
		if len(pm.services) > 0 {
			return nil
		}
		// Otherwise use defaults
		pm.setDefaultServices()
		return err
	}

	// Convert API response to our format
	pm.mu.Lock()
	pm.services = make([]ServiceDiscoveryInfo, len(services))
	for i, svc := range services {
		pm.services[i] = ServiceDiscoveryInfo{
			ID:            svc.ID,
			Name:          svc.Name,
			DisplayName:   svc.DisplayName,
			Icon:          svc.Icon,
			Provider:      svc.Provider,
			Available:     svc.Available,
			IsConfigBased: svc.IsConfigBased,
			ModelCount:    svc.ModelCount,
		}
	}
	pm.lastUpdated = time.Now()
	pm.mu.Unlock()

	log.Printf("[ProviderManager] Discovered %d services", len(services))

	// Notify listeners
	if pm.onChange != nil {
		pm.onChange()
	}

	return nil
}

// DiscoverServicesAsync discovers services asynchronously with callback
func (pm *ProviderManager) DiscoverServicesAsync(forceRefresh bool, callback func()) {
	go func() {
		pm.DiscoverServices(forceRefresh)
		if callback != nil {
			callback()
		}
	}()
}

// FetchModels fetches available models for a provider with caching
func (pm *ProviderManager) FetchModels(providerID string) []ProviderModel {
	// Check cache
	pm.mu.RLock()
	if cached, ok := pm.cachedModels[providerID]; ok {
		if lastFetch, ok := pm.lastModelFetch[providerID]; ok {
			if time.Since(lastFetch) < pm.cacheTTL {
				pm.mu.RUnlock()
				log.Printf("[ProviderManager] Using cached models for %s", providerID)
				return cached
			}
		}
	}
	pm.mu.RUnlock()

	log.Printf("[ProviderManager] Fetching models for %s from API...", providerID)

	// Try API first
	apiModels, err := pm.apiClient.GetAvailableModels(providerID)
	if err != nil {
		log.Printf("[ProviderManager] API fetch failed for %s: %v, using fallback", providerID, err)
		return pm.getDefaultModels(providerID)
	}

	// Convert to our format
	models := make([]ProviderModel, len(apiModels))
	for i, m := range apiModels {
		displayName := m.DisplayName
		if displayName == "" {
			displayName = m.Name
		}
		models[i] = ProviderModel{
			Name:        m.Name,
			DisplayName: displayName,
			Description: m.Description,
		}
	}

	// Cache the results
	pm.mu.Lock()
	pm.cachedModels[providerID] = models
	pm.lastModelFetch[providerID] = time.Now()
	pm.mu.Unlock()

	log.Printf("[ProviderManager] Fetched %d models for %s", len(models), providerID)
	return models
}

// FetchModelsAsync fetches models asynchronously with callback
func (pm *ProviderManager) FetchModelsAsync(providerID string, callback func([]ProviderModel)) {
	go func() {
		models := pm.FetchModels(providerID)
		if callback != nil {
			callback(models)
		}
	}()
}

// getDefaultModels returns default models for a provider
func (pm *ProviderManager) getDefaultModels(providerID string) []ProviderModel {
	switch providerID {
	case "anthropic":
		return []ProviderModel{
			{Name: "claude-sonnet-4-20250514", DisplayName: "Claude Sonnet 4", Description: "Most intelligent"},
			{Name: "claude-3-5-sonnet-20241022", DisplayName: "Claude 3.5 Sonnet", Description: "Balanced"},
			{Name: "claude-3-haiku-20240307", DisplayName: "Claude 3 Haiku", Description: "Fastest"},
		}
	case "openai":
		return []ProviderModel{
			{Name: "gpt-4o", DisplayName: "GPT-4o", Description: "Most capable"},
			{Name: "gpt-4o-mini", DisplayName: "GPT-4o Mini", Description: "Fast and affordable"},
		}
	case "google":
		return []ProviderModel{
			{Name: "gemini-1.5-pro", DisplayName: "Gemini 1.5 Pro", Description: "Most capable"},
			{Name: "gemini-1.5-flash", DisplayName: "Gemini 1.5 Flash", Description: "Fast"},
		}
	case "local-slm":
		return []ProviderModel{
			{Name: "mlx-community/Qwen2.5-Coder-32B-Instruct-8bit", DisplayName: "Qwen 2.5 Coder 32B"},
		}
	default:
		return []ProviderModel{}
	}
}

// AddProvider adds a new provider
func (pm *ProviderManager) AddProvider(provider ServiceProvider) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, p := range pm.providers {
		if p.ID == provider.ID {
			return fmt.Errorf("provider %s already exists", provider.ID)
		}
	}
	pm.providers = append(pm.providers, provider)
	return nil
}

// RemoveProvider removes a provider
func (pm *ProviderManager) RemoveProvider(providerID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for i, p := range pm.providers {
		if p.ID == providerID {
			pm.providers = append(pm.providers[:i], pm.providers[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("provider %s not found", providerID)
}

// TestConnection tests if a URL is reachable
func (pm *ProviderManager) TestConnection(baseURL string) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}

// ModelInfo represents a model from HuggingFace or other sources
type ModelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Author      string `json:"author"`
	Downloads   int    `json:"downloads"`
	IsFavorite  bool   `json:"is_favorite"`
	Description string `json:"description,omitempty"`
}

// GetFavoriteModels returns the hardcoded favorite models for quick access
func GetFavoriteModels(backend string) []ModelInfo {
	if backend == "mlx" {
		return []ModelInfo{
			{ID: "mlx-community/Qwen2.5-Coder-32B-Instruct-8bit", Name: "Qwen 2.5 Coder 32B (8-bit)", Author: "mlx-community", IsFavorite: true},
			{ID: "mlx-community/Qwen2.5-Coder-14B-Instruct-8bit", Name: "Qwen 2.5 Coder 14B (8-bit)", Author: "mlx-community", IsFavorite: true},
			{ID: "mlx-community/Qwen2.5-Coder-7B-Instruct-8bit", Name: "Qwen 2.5 Coder 7B (8-bit)", Author: "mlx-community", IsFavorite: true},
			{ID: "mlx-community/Qwen2.5-3B-Instruct-4bit", Name: "Qwen 2.5 3B (4-bit)", Author: "mlx-community", IsFavorite: true},
			{ID: "mlx-community/DeepSeek-R1-Distill-Qwen-32B-8bit", Name: "DeepSeek R1 Distill 32B", Author: "mlx-community", IsFavorite: true},
			{ID: "mlx-community/Llama-3.3-70B-Instruct-4bit", Name: "Llama 3.3 70B (4-bit)", Author: "mlx-community", IsFavorite: true},
		}
	}
	// vLLM models
	return []ModelInfo{
		{ID: "Qwen/Qwen2.5-Coder-32B-Instruct", Name: "Qwen 2.5 Coder 32B", Author: "Qwen", IsFavorite: true},
		{ID: "Qwen/Qwen2.5-Coder-14B-Instruct", Name: "Qwen 2.5 Coder 14B", Author: "Qwen", IsFavorite: true},
		{ID: "Qwen/Qwen2.5-Coder-7B-Instruct", Name: "Qwen 2.5 Coder 7B", Author: "Qwen", IsFavorite: true},
		{ID: "Qwen/Qwen2.5-3B-Instruct", Name: "Qwen 2.5 3B", Author: "Qwen", IsFavorite: true},
		{ID: "deepseek-ai/DeepSeek-R1-Distill-Qwen-32B", Name: "DeepSeek R1 Distill 32B", Author: "deepseek-ai", IsFavorite: true},
		{ID: "meta-llama/Llama-3.3-70B-Instruct", Name: "Llama 3.3 70B", Author: "meta-llama", IsFavorite: true},
	}
}

// SearchHuggingFaceModels searches HuggingFace for models matching a query
func SearchHuggingFaceModels(query string, backend string) []ModelInfo {
	// Filter by mlx-community for MLX backend
	author := ""
	if backend == "mlx" {
		author = "mlx-community"
	}

	url := fmt.Sprintf("https://huggingface.co/api/models?search=%s&limit=20&sort=downloads&direction=-1", query)
	if author != "" {
		url += "&author=" + author
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return []ModelInfo{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []ModelInfo{}
	}

	var hfModels []struct {
		ID        string `json:"id"`
		Author    string `json:"author"`
		Downloads int    `json:"downloads"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hfModels); err != nil {
		return []ModelInfo{}
	}

	models := make([]ModelInfo, 0, len(hfModels))
	for _, m := range hfModels {
		// Extract name from ID (e.g., "mlx-community/Qwen2.5-3B" -> "Qwen2.5-3B")
		name := m.ID
		if idx := len(m.Author); idx > 0 && len(m.ID) > idx+1 {
			name = m.ID[idx+1:]
		}
		models = append(models, ModelInfo{
			ID:        m.ID,
			Name:      name,
			Author:    m.Author,
			Downloads: m.Downloads,
		})
	}
	return models
}

// FetchOllamaModels fetches available models from a local Ollama instance
func FetchOllamaModels(host string) []ModelInfo {
	url := fmt.Sprintf("http://%s/api/tags", host)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return []ModelInfo{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []ModelInfo{}
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
			Size int64  `json:"size"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return []ModelInfo{}
	}

	models := make([]ModelInfo, 0, len(result.Models))
	for _, m := range result.Models {
		models = append(models, ModelInfo{
			ID:     m.Name,
			Name:   m.Name,
			Author: "ollama",
		})
	}
	return models
}

