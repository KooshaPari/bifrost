package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenRouter embeddings client - fallback provider
// Also used for LLM assistance (Gemini 2.5 Flash Lite, Qwen3-235B)

const (
	OpenRouterBaseURL = "https://openrouter.ai/api/v1"
)

// OpenRouterModel represents available models
type OpenRouterModel string

const (
	// Embedding models
	OpenAIEmbed3Large OpenRouterModel = "openai/text-embedding-3-large"
	OpenAIEmbed3Small OpenRouterModel = "openai/text-embedding-3-small"

	// LLM assistance models (cheap, fast)
	GeminiFlashLite OpenRouterModel = "google/gemini-2.5-flash-lite-preview-09-2025"
	Qwen3_235B      OpenRouterModel = "qwen/qwen3-235b-a22b-2507" // 10% context of Gemini but cheaper
)

// OpenRouterClient handles OpenRouter API calls
type OpenRouterClient struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
	appName    string // For OpenRouter attribution
}

// NewOpenRouterClient creates a new OpenRouter client
func NewOpenRouterClient(apiKey, appName string) *OpenRouterClient {
	return &OpenRouterClient{
		apiKey:  apiKey,
		baseURL: OpenRouterBaseURL,
		appName: appName,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// OpenRouterEmbedRequest for embeddings
type OpenRouterEmbedRequest struct {
	Input []string        `json:"input"`
	Model OpenRouterModel `json:"model"`
}

// OpenRouterEmbedResponse from API
type OpenRouterEmbedResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// ChatRequest for LLM assistance
type ChatRequest struct {
	Model    OpenRouterModel `json:"model"`
	Messages []ChatMessage   `json:"messages"`
	Stream   bool            `json:"stream,omitempty"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}

// ChatResponse from LLM
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Embed generates embeddings via OpenRouter
func (c *OpenRouterClient) Embed(ctx context.Context, texts []string, model OpenRouterModel) (*OpenRouterEmbedResponse, error) {
	if model == "" {
		model = OpenAIEmbed3Large
	}

	req := OpenRouterEmbedRequest{
		Input: texts,
		Model: model,
	}

	return c.doRequest(ctx, "/embeddings", req)
}

// Chat sends a chat completion request for LLM assistance
func (c *OpenRouterClient) Chat(ctx context.Context, messages []ChatMessage, model OpenRouterModel) (*ChatResponse, error) {
	if model == "" {
		model = GeminiFlashLite // Default to fast, cheap model
	}

	req := ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
	}

	return c.doChatRequest(ctx, "/chat/completions", req)
}

// doRequest handles embedding requests
func (c *OpenRouterClient) doRequest(ctx context.Context, endpoint string, reqBody OpenRouterEmbedRequest) (*OpenRouterEmbedResponse, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result OpenRouterEmbedResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}
