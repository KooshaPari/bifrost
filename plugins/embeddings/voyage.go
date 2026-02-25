// Package embeddings provides unified embedding and reranking clients
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

// VoyageAI client for embeddings and reranking
// Models: voyage-3.5 ($0.06/M), voyage-multimodal-3, voyage-code-3, rerank-2.5 ($0.05/M)
// Free tier: 200M tokens for embeddings, 200M for reranking

const (
	VoyageBaseURL       = "https://api.voyageai.com/v1"
	VoyageEmbedEndpoint = "/embeddings"
	VoyageRerankEndpoint = "/rerank"
)

// VoyageModel represents available Voyage models
type VoyageModel string

const (
	// Embedding models
	Voyage35       VoyageModel = "voyage-3.5"       // Best value: $0.06/M, 200M free
	Voyage35Lite   VoyageModel = "voyage-3.5-lite"  // Budget: $0.02/M, 200M free
	Voyage3Large   VoyageModel = "voyage-3-large"   // Highest quality: $0.18/M
	VoyageCode3    VoyageModel = "voyage-code-3"    // Code-specific: $0.18/M
	VoyageMulti3   VoyageModel = "voyage-multimodal-3" // Images+text: $0.12/M text

	// Reranker models
	Rerank25     VoyageModel = "rerank-2.5"      // Best: $0.05/M, 200M free
	Rerank25Lite VoyageModel = "rerank-2.5-lite" // Budget: $0.02/M
)

// VoyageClient handles VoyageAI API calls
type VoyageClient struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewVoyageClient creates a new VoyageAI client
func NewVoyageClient(apiKey string) *VoyageClient {
	return &VoyageClient{
		apiKey:  apiKey,
		baseURL: VoyageBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// EmbeddingRequest for VoyageAI embeddings API
type EmbeddingRequest struct {
	Input     []string    `json:"input"`
	Model     VoyageModel `json:"model"`
	InputType string      `json:"input_type,omitempty"` // "query" or "document"
}

// EmbeddingResponse from VoyageAI
type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// RerankRequest for VoyageAI reranking API
type RerankRequest struct {
	Query     string      `json:"query"`
	Documents []string    `json:"documents"`
	Model     VoyageModel `json:"model"`
	TopK      int         `json:"top_k,omitempty"`
}

// RerankResponse from VoyageAI
type RerankResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Index          int     `json:"index"`
		RelevanceScore float64 `json:"relevance_score"`
		Document       string  `json:"document,omitempty"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// Embed generates embeddings for the given texts
func (c *VoyageClient) Embed(ctx context.Context, texts []string, model VoyageModel, inputType string) (*EmbeddingResponse, error) {
	if model == "" {
		model = Voyage35 // Default to best value model
	}

	req := EmbeddingRequest{
		Input:     texts,
		Model:     model,
		InputType: inputType,
	}

	return doRequest[EmbeddingRequest, EmbeddingResponse](ctx, c, VoyageEmbedEndpoint, req)
}

// Rerank reorders documents by relevance to query
func (c *VoyageClient) Rerank(ctx context.Context, query string, documents []string, topK int) (*RerankResponse, error) {
	req := RerankRequest{
		Query:     query,
		Documents: documents,
		Model:     Rerank25,
		TopK:      topK,
	}

	return doRequest[RerankRequest, RerankResponse](ctx, c, VoyageRerankEndpoint, req)
}

// doRequest is a generic helper for API calls
func doRequest[Req, Resp any](ctx context.Context, c *VoyageClient, endpoint string, reqBody Req) (*Resp, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

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

	var result Resp
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

