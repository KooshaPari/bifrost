# Agents and Automation Guide - bifrost-extensions

This package is a **Zero-Fork Extension Architecture** for enhancing Bifrost with intelligent routing, learning, and CLI agent integration.

**Authority and Scope**
- This file is the canonical contract for all agent behavior in this package.
- Act autonomously; only pause when blocked by missing secrets, external access, or truly destructive actions.

---

## Table of Contents

1. [Package Overview](#1-package-overview)
2. [Core Architecture](#2-core-architecture)
3. [Development Commands](#3-development-commands)
4. [Plugin Development](#4-plugin-development)
5. [Provider Development](#5-provider-development)
6. [Bifrost Schemas Reference](#6-bifrost-schemas-reference)
7. [Testing Strategy](#7-testing-strategy)
8. [Common Workflows](#8-common-workflows)

---

## 1. Package Overview

### Purpose

`bifrost-extensions` extends [Bifrost](https://github.com/maximhq/bifrost) without forking by implementing:

1. **Intelligent Routing** - RouteLLM, Arch-Router, MIRT, semantic task classification
2. **Performance Learning** - Tracks latency, tokens, success rates; generates routing rules
3. **Smart Fallback** - Exponential backoff, budget-aware fallbacks, task-specific rules
4. **CLI Agents** - Wraps agentapi for Claude Code, Cursor, Goose, Auggie, etc.
5. **OAuth Providers** - Routes through CLIProxyAPI OAuth subscriptions

### Package Structure

```
bifrost-extensions-governance/
├── go.mod                          # Module: github.com/kooshapari/bifrost-extensions
├── account/
│   └── account.go                  # EnhancedAccount implements schemas.Account
├── cmd/
│   └── bifrost-enhanced/
│       └── main.go                 # Entry point wiring Bifrost + plugins
├── plugins/
│   ├── intelligentrouter/          # Unified routing engine
│   │   ├── router.go               # Main plugin (schemas.Plugin)
│   │   ├── semantic.go             # Task classification + feature extraction
│   │   ├── routellm.go             # RouteLLM cost-quality routing
│   │   ├── arch_router.go          # Arch-Router task classification
│   │   ├── mirt.go                 # 25D MIRT scoring
│   │   └── decision.go             # Routing decision pipeline
│   ├── learning/                   # Performance tracking
│   │   ├── learning.go             # Main plugin
│   │   ├── tracker.go              # Metrics collection
│   │   └── patterns.go             # Pattern detection
│   └── smartfallback/              # Fallback handling
│       ├── fallback.go             # Main plugin
│       ├── strategies.go           # Retry strategies
│       └── task_rules.go           # Task-specific rules
├── providers/
│   ├── agentcli/                   # CLI agent provider
│   │   ├── provider.go             # Bifrost provider wrapper
│   │   ├── provider_test.go        # Unit tests
│   │   └── client.go               # HTTP client for agentapi
│   └── oauthproxy/                 # OAuth subscription provider
│       ├── provider.go             # OAuth proxy provider
│       ├── auth.go                 # PKCE OAuth2 implementation
│       ├── auth_test.go            # Unit tests
│       └── types.go                # OpenAI-compatible types
├── config/                         # Configuration loading
│   ├── config.go                   # Viper-based config with YAML+env
│   └── config_test.go              # Unit tests
└── server/                         # HTTP server
    ├── server.go                   # Chi router HTTP server
    └── handlers.go                 # OpenAI-compatible endpoints
```

### Dependencies

```go
require (
    github.com/maximhq/bifrost/core  // Bifrost core schemas
    github.com/google/uuid           // UUID generation
)

// Development: local path replacement
replace github.com/maximhq/bifrost/core => ../bifrost/core
```

---

## 2. Core Architecture

### Zero-Fork Principle

**CRITICAL**: Never modify upstream repos. Always extend via interfaces:

```go
// ✅ CORRECT: Implement interface
type EnhancedAccount struct {
    fallback schemas.Account  // Delegate to upstream
    configs  map[schemas.ModelProvider]*schemas.ProviderConfig
}

func (a *EnhancedAccount) GetKeysForProvider(ctx *context.Context, provider schemas.ModelProvider) ([]schemas.Key, error) {
    // Check local first, then delegate
    if keys, ok := a.keys[provider]; ok {
        return keys, nil
    }
    return a.fallback.GetKeysForProvider(ctx, provider)
}

// ❌ WRONG: Forking upstream and modifying directly
```

### Plugin Interface (schemas.Plugin)

All plugins MUST implement:

```go
type Plugin interface {
    GetName() string

    TransportInterceptor(
        ctx *context.Context,
        url string,
        headers map[string]string,
        body map[string]any,
    ) (map[string]string, map[string]any, error)

    PreHook(
        ctx *context.Context,
        req *BifrostRequest,
    ) (*BifrostRequest, *PluginShortCircuit, error)

    PostHook(
        ctx *context.Context,
        resp *BifrostResponse,
        err *BifrostError,
    ) (*BifrostResponse, *BifrostError, error)

    Cleanup() error
}
```

### Account Interface (schemas.Account)

```go
type Account interface {
    GetConfiguredProviders() ([]ModelProvider, error)
    GetKeysForProvider(ctx *context.Context, providerKey ModelProvider) ([]Key, error)
    GetConfigForProvider(providerKey ModelProvider) (*ProviderConfig, error)
}
```

### Request/Response Types

```go
// BifrostRequest - union struct (only ONE field set)
type BifrostRequest struct {
    RequestType           RequestType
    ChatRequest           *BifrostChatRequest
    TextCompletionRequest *BifrostTextCompletionRequest
    ResponsesRequest      *BifrostResponsesRequest
    // ... other request types
}

// BifrostResponse - union struct
type BifrostResponse struct {
    ChatResponse           *BifrostChatResponse
    TextCompletionResponse *BifrostTextCompletionResponse
    // ... other response types
}

// BifrostError
type BifrostError struct {
    StatusCode     *int
    Error          *ErrorField
    AllowFallbacks *bool
}
```

---

## 3. Development Commands

### Build & Test

```bash
# Navigate to package
cd bifrost-extensions

# Tidy dependencies
go mod tidy

# Build all packages
go build ./...

# Run vet
go vet ./...

# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Build binary
go build -o bin/bifrost-enhanced ./cmd/bifrost-enhanced

# Run the server
./bin/bifrost-enhanced
```

### Environment Variables

```bash
# Provider API keys
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export GOOGLE_API_KEY="..."

# Optional: RouteLLM endpoint
export ROUTELLM_ENDPOINT="http://localhost:8000"

# Optional: Arch-Router endpoint
export ARCH_ROUTER_ENDPOINT="http://127.0.0.1:8008"
```

---

## 4. Plugin Development

### Creating a New Plugin

```go
package myplugin

import (
    "context"
    "github.com/maximhq/bifrost/core/schemas"
)

type MyPlugin struct {
    config *Config
}

func New(config *Config) *MyPlugin {
    return &MyPlugin{config: config}
}

func (p *MyPlugin) GetName() string {
    return "my-plugin"
}

func (p *MyPlugin) TransportInterceptor(
    ctx *context.Context,
    url string,
    headers map[string]string,
    body map[string]any,
) (map[string]string, map[string]any, error) {
    // Modify headers/body at transport level
    return headers, body, nil
}

func (p *MyPlugin) PreHook(
    ctx *context.Context,
    req *schemas.BifrostRequest,
) (*schemas.BifrostRequest, *schemas.PluginShortCircuit, error) {
    // Modify request before provider call
    // Return PluginShortCircuit to skip provider entirely
    return req, nil, nil
}

func (p *MyPlugin) PostHook(
    ctx *context.Context,
    resp *schemas.BifrostResponse,
    err *schemas.BifrostError,
) (*schemas.BifrostResponse, *schemas.BifrostError, error) {
    // Process response/error after provider call
    return resp, err, nil
}

func (p *MyPlugin) Cleanup() error {
    // Release resources
    return nil
}
```

### Routing Decision Pattern

```go
// Store decision in context for PostHook
type contextKey string
const routingDecisionKey contextKey = "routing_decision"

func (p *MyPlugin) PreHook(ctx *context.Context, req *schemas.BifrostRequest) (*schemas.BifrostRequest, *schemas.PluginShortCircuit, error) {
    decision := p.makeDecision(req)
    *ctx = context.WithValue(*ctx, routingDecisionKey, decision)

    // Modify request model/provider
    req.SetModel(decision.Model)
    req.SetProvider(decision.Provider)

    return req, nil, nil
}

func (p *MyPlugin) PostHook(ctx *context.Context, resp *schemas.BifrostResponse, err *schemas.BifrostError) (*schemas.BifrostResponse, *schemas.BifrostError, error) {
    // Retrieve decision for logging/learning
    if decision, ok := (*ctx).Value(routingDecisionKey).(*Decision); ok {
        p.logOutcome(decision, resp, err)
    }
    return resp, err, nil
}
```

---

## 5. Provider Development

### Custom Provider Pattern

Providers are NOT part of Bifrost plugin system - they're configured via Account:

```go
// Register custom provider via Account
acct := account.NewEnhancedAccount(nil)

// Set keys for custom provider
acct.SetKeys(schemas.ModelProvider("my-provider"), []schemas.Key{{
    ID:     "my-key",
    Value:  "secret",
    Models: []string{"model-a", "model-b"},
    Weight: 1.0,
}})

// Set config
acct.SetConfig(schemas.ModelProvider("my-provider"), &schemas.ProviderConfig{
    NetworkConfig: schemas.NetworkConfig{
        DefaultRequestTimeoutInSeconds: 60,
        MaxRetries:                     3,
    },
})
```

### CLI Agent Provider (agentcli)

```go
// Wraps agentapi HTTP API
provider := agentcli.New(&agentcli.Config{
    AgentType:   agentcli.AgentTypeClaude,
    BaseURL:     "http://localhost",
    Port:        3284,
    Timeout:     30 * time.Second,
    MaxWaitTime: 5 * time.Minute,
})

// ChatCompletion sends message, waits for stable, returns response
resp, err := provider.ChatCompletion(ctx, req)
```

### OAuth Proxy Provider (oauthproxy)

```go
// Proxies through CLIProxyAPI OAuth authentication
provider := oauthproxy.New(&oauthproxy.Config{
    ProviderType: oauthproxy.ProviderClaude,
    ProxyBaseURL: "http://localhost",
    ProxyPort:    8080,
})
```

---

## 6. Bifrost Schemas Reference

### Key Types

```go
// ModelProvider - typed string for providers
type ModelProvider string
const (
    OpenAI    ModelProvider = "openai"
    Anthropic ModelProvider = "anthropic"
    Gemini    ModelProvider = "gemini"
    // ... etc
)

// ChatMessageRole - typed string
type ChatMessageRole string
const (
    ChatMessageRoleUser      ChatMessageRole = "user"
    ChatMessageRoleAssistant ChatMessageRole = "assistant"
    ChatMessageRoleSystem    ChatMessageRole = "system"
)

// ChatMessage
type ChatMessage struct {
    Role    ChatMessageRole
    Content *ChatMessageContent
    // ...
}

// ChatMessageContent - union type
type ChatMessageContent struct {
    ContentStr    *string           // Simple string content
    ContentBlocks []ChatContentBlock // Multimodal content
}
```

### Request Helpers

```go
// Get request fields
provider, model, fallbacks := req.GetRequestFields()

// Set request fields
req.SetModel("gpt-4")
req.SetProvider(schemas.OpenAI)
req.SetFallbacks([]schemas.Fallback{...})

// Access chat request
if req.ChatRequest != nil {
    messages := req.ChatRequest.Input
    params := req.ChatRequest.Params
}

// Access text completion
if req.TextCompletionRequest != nil && req.TextCompletionRequest.Input != nil {
    if req.TextCompletionRequest.Input.PromptStr != nil {
        prompt := *req.TextCompletionRequest.Input.PromptStr
    }
}
```

### Response Helpers

```go
// Chat response
if resp.ChatResponse != nil {
    for _, choice := range resp.ChatResponse.Choices {
        if choice.ChatNonStreamResponseChoice != nil {
            msg := choice.ChatNonStreamResponseChoice.Message
        }
    }
    usage := resp.ChatResponse.Usage
}

// Create error
err := &schemas.BifrostError{
    StatusCode: ptrInt(500),
    Error:      &schemas.ErrorField{Message: "error message"},
}
```

---

## 7. Testing Strategy

### Unit Tests

```go
func TestIntelligentRouter_PreHook(t *testing.T) {
    router := intelligentrouter.New(intelligentrouter.DefaultConfig())

    ctx := context.Background()
    req := &schemas.BifrostRequest{
        ChatRequest: &schemas.BifrostChatRequest{
            Model: "gpt-4",
            Input: []schemas.ChatMessage{{
                Role: schemas.ChatMessageRoleUser,
                Content: &schemas.ChatMessageContent{
                    ContentStr: ptrString("Write a function"),
                },
            }},
        },
    }

    modifiedReq, shortCircuit, err := router.PreHook(&ctx, req)
    assert.NoError(t, err)
    assert.Nil(t, shortCircuit)
    assert.NotNil(t, modifiedReq)
}
```

### Integration Tests

```go
func TestBifrostWithPlugins(t *testing.T) {
    ctx := context.Background()

    acct := account.NewEnhancedAccount(nil)
    acct.SetKeys(schemas.OpenAI, []schemas.Key{{
        ID:    "test",
        Value: os.Getenv("OPENAI_API_KEY"),
    }})

    bf, err := bifrost.Init(ctx, schemas.BifrostConfig{
        Account: acct,
        Plugins: []schemas.Plugin{
            intelligentrouter.New(nil),
            learning.New(nil),
        },
    })
    require.NoError(t, err)
    defer bf.Shutdown()

    // Make request
    resp, bifrostErr := bf.ChatCompletion(ctx, &schemas.BifrostChatRequest{...})
    // Assert...
}
```

---

## 8. Common Workflows

### Adding a New Routing Strategy

1. Create file in `plugins/intelligentrouter/`
2. Implement scoring/classification logic
3. Integrate into `decision.go` pipeline
4. Add configuration options to `Config`
5. Write unit tests

### Adding a New CLI Agent

1. Add constant to `AgentType` in `providers/agentcli/provider.go`
2. Update any agent-specific handling if needed
3. Test with agentapi running locally

### Adding a New OAuth Provider

1. Add constant to `ProviderType` in `providers/oauthproxy/provider.go`
2. Update `getEndpoint()` for provider-specific API
3. Handle any response format differences

### Debugging Plugin Chain

```go
// Add logging to see plugin execution
func (p *MyPlugin) PreHook(ctx *context.Context, req *schemas.BifrostRequest) (*schemas.BifrostRequest, *schemas.PluginShortCircuit, error) {
    provider, model, _ := req.GetRequestFields()
    log.Printf("[%s] PreHook: provider=%s model=%s", p.GetName(), provider, model)
    // ...
}
```

---

## Quick Reference

| Task | Command |
|------|---------|
| Build | `go build ./...` |
| Test | `go test ./...` |
| Vet | `go vet ./...` |
| Run | `go run ./cmd/bifrost-enhanced` |
| Tidy deps | `go mod tidy` |

| Interface | Location |
|-----------|----------|
| `schemas.Plugin` | `bifrost/core/schemas/plugin.go` |
| `schemas.Account` | `bifrost/core/schemas/account.go` |
| `schemas.BifrostRequest` | `bifrost/core/schemas/bifrost.go` |
| `schemas.BifrostResponse` | `bifrost/core/schemas/bifrost.go` |


<!-- PHENOTYPE_GOVERNANCE_OVERLAY_V1 -->
## Phenotype Governance Overlay v1

- Enforce `TDD + BDD + SDD` for all feature and workflow changes.
- Enforce `Hexagonal + Clean + SOLID` boundaries by default.
- Favor explicit failures over silent degradation; required dependencies must fail clearly when unavailable.
- Keep local hot paths deterministic and low-latency; place distributed workflow logic behind durable orchestration boundaries.
- Require policy gating, auditability, and traceable correlation IDs for agent and workflow actions.
- Document architectural and protocol decisions before broad rollout changes.


## Bot Review Retrigger and Rate-Limit Governance

- Retrigger commands:
  - CodeRabbit: `@coderabbitai full review`
  - Gemini Code Assist: `@gemini-code-assist review` (fallback: `/gemini review`)
- Rate-limit contract:
  - Maximum one retrigger per bot per PR every 15 minutes.
  - Before triggering, check latest PR comments for existing trigger markers and bot quota/rate-limit responses.
  - If rate-limited, queue the retry for the later of 15 minutes or bot-provided retry time.
  - After two consecutive rate-limit responses for the same bot/PR, stop auto-retries and post queued status with next attempt time.
- Tracking marker required in PR comments for each trigger:
  - `bot-review-trigger: <bot> <iso8601-time> <reason>`



## Review Bot Governance

- Keep CodeRabbit PR blocking at the lowest level in `.coderabbit.yaml`: `pr_validation.block_on.severity: info`.
- Keep Gemini Code Assist severity at the lowest level in `.gemini/config.yaml`: `code_review.comment_severity_threshold: LOW`.
- Retrigger commands:
  - CodeRabbit: comment `@coderabbitai full review` on the PR.
  - Gemini Code Assist (when enabled in the repo): comment `@gemini-code-assist review` on the PR.
  - If comment-trigger is unavailable, retrigger both bots by pushing a no-op commit to the PR branch.
- Rate-limit discipline:
  - Use a FIFO queue for retriggers (oldest pending PR first).
  - Minimum spacing: one retrigger comment every 120 seconds per repo.
  - On rate-limit response, stop sending new triggers in that repo, wait 15 minutes, then resume queue processing.
  - Do not post duplicate trigger comments while a prior trigger is pending.

## Child-Agent and Delegation Policy
- Use child agents liberally for scoped discovery, audits, multi-repo scans, and implementation planning before direct parent-agent edits.
- Prefer delegating high-context or high-churn tasks to subagents, and keep parent-agent changes focused on integration and finalization.
- Reserve parent-agent direct writes for the narrowest, final decision layer.
