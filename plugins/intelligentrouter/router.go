// Package intelligentrouter provides an intelligent routing plugin for Bifrost.
// It consolidates DualRouter, SemanticRouter, ArchRouter, RouteLLM, and MIRT
// into a single unified routing engine with cost-aware endpoint selection.
package intelligentrouter

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/maximhq/bifrost/core/schemas"

	"github.com/kooshapari/bifrost-extensions/costengine"
	"github.com/kooshapari/bifrost-extensions/db/sqlc"
	"github.com/kooshapari/bifrost-extensions/slm"
)

// TaskType represents the type of task being routed
type TaskType string

const (
	TaskTypeToolCall     TaskType = "tool_call"
	TaskTypeCodeGen      TaskType = "code_generation"
	TaskTypeReasoning    TaskType = "reasoning"
	TaskTypeConversation TaskType = "conversation"
	TaskTypeDefault      TaskType = "default"
)

// RiskLevel represents the risk level of a task
type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// RoutingDecision contains the routing decision details
type RoutingDecision struct {
	SelectedModel      string
	SelectedProvider   schemas.ModelProvider
	SelectedEndpointID uuid.UUID
	TaskType           TaskType
	RiskLevel          RiskLevel
	Confidence         float64
	Alternatives       []string
	FallbackEndpoints  []uuid.UUID
	ToolProfile        slm.ToolProfile
	ContextStrategy    string
	Reasoning          string
	CostEstimate       float64
	QuotaHeadroom      float64
}

// Config configures the intelligent router
type Config struct {
	// RouteLLM settings
	RouteLLMEnabled   bool    `json:"routellm_enabled"`
	RouteLLMThreshold float64 `json:"routellm_threshold"` // 0.0-1.0
	RouteLLMRouter    string  `json:"routellm_router"`    // "mf", "bert", "sw_ranking"

	// Semantic routing rules
	SemanticRulesPath string `json:"semantic_rules_path"`

	// MIRT settings (quality-cost optimization)
	MIRTEnabled bool `json:"mirt_enabled"`

	// Arch-Router settings (task classification)
	ArchRouterEndpoint string `json:"arch_router_endpoint"`
	ArchRouterModel    string `json:"arch_router_model"`

	// SLM Router settings (local router SLM)
	RouterSLMURL     string `json:"router_slm_url"`
	SummarizerSLMURL string `json:"summarizer_slm_url"`

	// Model preferences
	PreferredProviders []schemas.ModelProvider `json:"preferred_providers"`
	FallbackProviders  []schemas.ModelProvider `json:"fallback_providers"`

	// Cost engine settings
	UseCostEngine        bool    `json:"use_cost_engine"`
	MaxCostPerRequest    float64 `json:"max_cost_per_request"`
	PreferSubscriptions  bool    `json:"prefer_subscriptions"`   // prefer subscription buckets
	AllowScarceEndpoints bool    `json:"allow_scarce_endpoints"` // allow scarce_premium
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *Config {
	return &Config{
		RouteLLMEnabled:      true,
		RouteLLMThreshold:    0.7,
		RouteLLMRouter:       "mf", // Matrix Factorization - best APGR
		MIRTEnabled:          true,
		ArchRouterEndpoint:   "http://127.0.0.1:8008",
		ArchRouterModel:      "katanemo/Arch-Router-1.5B",
		RouterSLMURL:         "http://localhost:9001",
		SummarizerSLMURL:     "http://localhost:9002",
		UseCostEngine:        true,
		MaxCostPerRequest:    1.0, // $1 max per request
		PreferSubscriptions:  true,
		AllowScarceEndpoints: false,
		PreferredProviders: []schemas.ModelProvider{
			schemas.OpenAI,
			schemas.Anthropic,
		},
	}
}

// IntelligentRouter is the unified routing plugin
type IntelligentRouter struct {
	config *Config
	mu     sync.RWMutex

	// Sub-routers (consolidated from CLIProxyAPI)
	semanticRouter *SemanticRouter
	archClient     *ArchRouterClient
	mirtClient     *MIRTClient
	routeLLM       *RouteLLMClient

	// New infrastructure
	costEngine *costengine.Engine
	slmClients *slm.Clients
	queries    *sqlc.Queries
}

// New creates a new IntelligentRouter plugin
func New(config *Config) *IntelligentRouter {
	if config == nil {
		config = DefaultConfig()
	}

	return &IntelligentRouter{
		config:         config,
		semanticRouter: NewSemanticRouter(config),
		// archClient, mirtClient, routeLLM initialized lazily
	}
}

// GetName returns the plugin name
func (ir *IntelligentRouter) GetName() string {
	return "intelligent-router"
}

// TransportInterceptor is called at HTTP transport layer
func (ir *IntelligentRouter) TransportInterceptor(
	ctx *context.Context,
	url string,
	headers map[string]string,
	body map[string]any,
) (map[string]string, map[string]any, error) {
	// No transport-level modifications needed for routing
	return headers, body, nil
}

// PreHook performs intelligent routing before provider call
func (ir *IntelligentRouter) PreHook(
	ctx *context.Context,
	req *schemas.BifrostRequest,
) (*schemas.BifrostRequest, *schemas.PluginShortCircuit, error) {
	ir.mu.RLock()
	defer ir.mu.RUnlock()

	// Extract features from request
	features := ir.extractFeatures(req)

	// Make routing decision
	decision := ir.route(*ctx, features, req)

	// Apply routing decision to request
	modifiedReq := ir.applyDecision(req, decision)

	// Store decision in context for PostHook/logging
	*ctx = context.WithValue(*ctx, routingDecisionKey, decision)

	return modifiedReq, nil, nil
}

// PostHook processes response after provider call
func (ir *IntelligentRouter) PostHook(
	ctx *context.Context,
	resp *schemas.BifrostResponse,
	err *schemas.BifrostError,
) (*schemas.BifrostResponse, *schemas.BifrostError, error) {
	// Log routing decision outcome for learning
	if decision, ok := (*ctx).Value(routingDecisionKey).(*RoutingDecision); ok {
		ir.logDecisionOutcome(decision, resp, err)
	}

	return resp, err, nil
}

// Cleanup releases resources
func (ir *IntelligentRouter) Cleanup() error {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	// Cleanup sub-routers
	return nil
}

// Private context key for routing decision
type contextKey string

const routingDecisionKey contextKey = "routing_decision"
