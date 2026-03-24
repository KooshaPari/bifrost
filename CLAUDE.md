# Claude AI Agent Guide - bifrost-extensions

This package extends Bifrost with intelligent routing, learning, and CLI agent integration using a **Zero-Fork Architecture**.

**Authority and Scope**
- This file is the canonical contract for all agent behavior in this package.
- Act autonomously; only pause when blocked by missing secrets, external access, or truly destructive actions.

---

## Quick Start

```bash
cd bifrost-extensions
go mod tidy
go build ./...
go test ./...
```

---

## 1. Core Expectations for Agents

### Autonomous Operation

**When to proceed without asking:**
- Implementation details and technical approach decisions
- Adding new routing strategies or plugins
- Refactoring and optimization
- Bug fixes and test improvements
- Documentation updates

**Only ask when truly blocked by:**
- Missing API keys/secrets
- External service access permissions
- Genuine product ambiguity
- Destructive operations (production data, forced pushes)

### Research-First Development

Before implementing ANY feature:

```bash
# Find similar implementations
rg "pattern_name" --type go -A 5 -B 5

# Trace interface implementations
rg "func.*GetName\(\)" --type go

# Find test patterns
rg "func Test" --type go -A 10

# Check Bifrost schemas
cat ../bifrost/core/schemas/*.go | head -200
```

---

## 2. Critical Type Gotchas

### ChatMessageRole Conversion

```go
// ❌ WRONG: Direct string comparison
if msg.Role == "user" { ... }

// ✅ CORRECT: Use typed constant
if msg.Role == schemas.ChatMessageRoleUser { ... }

// ❌ WRONG: Assign string directly
msg.Role = "assistant"

// ✅ CORRECT: Use type conversion
msg.Role = schemas.ChatMessageRole("assistant")
```

### ChatParameters Fields

```go
// ❌ WRONG: MaxTokens doesn't exist
params.MaxTokens = 1000

// ✅ CORRECT: Use MaxCompletionTokens
params.MaxCompletionTokens = ptrInt(1000)
```

### TextCompletionRequest Prompt

```go
// ❌ WRONG: Prompt field doesn't exist
prompt := req.TextCompletionRequest.Prompt

// ✅ CORRECT: Use Input.PromptStr or Input.PromptArray
if req.TextCompletionRequest.Input != nil {
    if req.TextCompletionRequest.Input.PromptStr != nil {
        prompt := *req.TextCompletionRequest.Input.PromptStr
    }
}
```

### Pointer Helpers

```go
// Helper functions for pointer types
func ptrString(s string) *string { return &s }
func ptrInt(i int) *int { return &i }
func ptrFloat(f float64) *float64 { return &f }
func ptrBool(b bool) *bool { return &b }
```

---

## 3. Plugin Interface Reference

### Required Methods

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

### PreHook Return Values

| Return | Meaning |
|--------|---------|
| `(req, nil, nil)` | Continue with modified request |
| `(nil, shortCircuit, nil)` | Skip provider, return cached/computed response |
| `(nil, nil, err)` | Abort with error |

### PostHook Return Values

| Return | Meaning |
|--------|---------|
| `(resp, nil, nil)` | Continue with modified response |
| `(nil, err, nil)` | Replace response with error |
| `(nil, nil, err)` | Abort with error |

---

## 4. Request/Response Patterns

### Accessing Chat Messages

```go
if req.ChatRequest != nil {
    for _, msg := range req.ChatRequest.Input {
        role := string(msg.Role)

        // Get content
        if msg.Content != nil {
            if msg.Content.ContentStr != nil {
                text := *msg.Content.ContentStr
            } else if len(msg.Content.ContentBlocks) > 0 {
                // Handle multimodal content
            }
        }
    }
}
```

### Modifying Request

```go
func (p *MyPlugin) PreHook(ctx *context.Context, req *schemas.BifrostRequest) (*schemas.BifrostRequest, *schemas.PluginShortCircuit, error) {
    // Change model
    req.SetModel("gpt-4-turbo")

    // Change provider
    req.SetProvider(schemas.OpenAI)

    // Add fallbacks
    req.SetFallbacks([]schemas.Fallback{
        {Model: "gpt-3.5-turbo", Provider: schemas.OpenAI},
        {Model: "claude-3-haiku", Provider: schemas.Anthropic},
    })

    return req, nil, nil
}
```

### Creating Short-Circuit Response

```go
func (p *CachePlugin) PreHook(ctx *context.Context, req *schemas.BifrostRequest) (*schemas.BifrostRequest, *schemas.PluginShortCircuit, error) {
    if cached := p.cache.Get(req); cached != nil {
        return nil, &schemas.PluginShortCircuit{
            Response: cached,
        }, nil
    }
    return req, nil, nil
}
```

---

## 5. Account Interface

### Implementation Pattern

```go
type EnhancedAccount struct {
    fallback schemas.Account
    mu       sync.RWMutex
    keys     map[schemas.ModelProvider][]schemas.Key
    configs  map[schemas.ModelProvider]*schemas.ProviderConfig
}

func NewEnhancedAccount(fallback schemas.Account) *EnhancedAccount {
    return &EnhancedAccount{
        fallback: fallback,
        keys:     make(map[schemas.ModelProvider][]schemas.Key),
        configs:  make(map[schemas.ModelProvider]*schemas.ProviderConfig),
    }
}

func (a *EnhancedAccount) GetKeysForProvider(ctx *context.Context, provider schemas.ModelProvider) ([]schemas.Key, error) {
    a.mu.RLock()
    defer a.mu.RUnlock()

    if keys, ok := a.keys[provider]; ok {
        return keys, nil
    }
    if a.fallback != nil {
        return a.fallback.GetKeysForProvider(ctx, provider)
    }
    return nil, nil
}
```

---

## 6. File Organization

### Package Structure

```
bifrost-extensions-governance/
├── go.mod                      # Module definition
├── account/account.go          # EnhancedAccount
├── cmd/bifrost-enhanced/       # Entry point
├── config/
│   ├── config.go               # Viper-based YAML+env config
│   └── config_test.go          # Tests
├── plugins/
│   ├── intelligentrouter/      # Routing plugin (6 files)
│   ├── learning/               # Learning plugin + tests
│   └── smartfallback/          # Fallback plugin + tests
├── providers/
│   ├── agentcli/               # CLI agent provider + tests
│   └── oauthproxy/             # OAuth proxy + PKCE auth + tests
└── server/
    ├── server.go               # Chi HTTP server
    └── handlers.go             # OpenAI-compatible endpoints
```

### File Size Limits

- **Target**: ≤350 lines per file
- **Hard limit**: ≤500 lines per file
- **Decompose** when approaching limits

---

## 7. Testing

### Unit Test Pattern

```go
func TestPlugin_PreHook(t *testing.T) {
    plugin := myplugin.New(nil)

    ctx := context.Background()
    req := &schemas.BifrostRequest{
        ChatRequest: &schemas.BifrostChatRequest{
            Model: "gpt-4",
            Input: []schemas.ChatMessage{{
                Role: schemas.ChatMessageRoleUser,
                Content: &schemas.ChatMessageContent{
                    ContentStr: ptrString("Hello"),
                },
            }},
        },
    }

    modifiedReq, shortCircuit, err := plugin.PreHook(&ctx, req)

    assert.NoError(t, err)
    assert.Nil(t, shortCircuit)
    assert.NotNil(t, modifiedReq)
}
```

### Run Tests

```bash
# All tests
go test ./...

# Specific package
go test ./plugins/intelligentrouter/...

# With coverage
go test -cover ./...

# Verbose
go test -v ./...
```

---

## 8. Common Errors & Fixes

| Error | Cause | Fix |
|-------|-------|-----|
| `cannot use "user" as ChatMessageRole` | String vs typed string | Use `schemas.ChatMessageRoleUser` |
| `req.TextCompletionRequest.Prompt undefined` | Wrong field name | Use `Input.PromptStr` |
| `params.MaxTokens undefined` | Wrong field name | Use `MaxCompletionTokens` |
| `cannot convert string to ModelProvider` | Type mismatch | Use `schemas.ModelProvider("name")` |
| `account.New undefined` | Wrong constructor | Use `account.NewEnhancedAccount(nil)` |
| `chunk.ChatResponse undefined` | BifrostStream embedded types | Use `chunk.BifrostChatResponse` |
| `chunk.Error undefined` | BifrostStream embedded types | Use `chunk.BifrostError` |
| `pkce.CodeVerifier undefined` | Wrong PKCE field name | Use `pkce.Verifier` |
| `store.Get returns 1 value` | API changed | `token := store.Get(key)` not `token, ok` |
| `illegal character U+005C` | Heredoc escaping `\!` | Use `sed -i '' 's/\\!/!/g'` to fix |

### BifrostStream Gotcha

```go
// BifrostStream uses embedded types - access via full type name
type BifrostStream struct {
    *BifrostTextCompletionResponse
    *BifrostChatResponse           // ← Access via chunk.BifrostChatResponse
    *BifrostResponsesStreamResponse
    *BifrostError                  // ← Access via chunk.BifrostError
}

// ❌ WRONG
if chunk.ChatResponse != nil { ... }
if chunk.Error != nil { ... }

// ✅ CORRECT
if chunk.BifrostChatResponse != nil { ... }
if chunk.BifrostError != nil { ... }
```

---

## 9. OAuth Integration Patterns

### PKCE Flow (CLIProxyAPI)

```go
// Generate PKCE codes
pkce, _ := oauthproxy.GeneratePKCE()
// pkce.Verifier  - 43+ char random string
// pkce.Challenge - SHA256(Verifier), base64url encoded

// Claude OAuth
claude := oauthproxy.NewClaudeOAuth()
authURL := claude.GetAuthURL(pkce, "state-string")
// User visits authURL, authorizes, gets redirected with ?code=...

// Exchange code for token
token, _ := claude.ExchangeCode(authCode, pkce)

// Store token
store := oauthproxy.NewTokenStore("/path/tokens.json")
store.Set("claude", &oauthproxy.StoredToken{
    AccessToken:  token.AccessToken,
    RefreshToken: token.RefreshToken,
    ExpiresAt:    time.Now().Add(time.Duration(token.ExpiresIn) * time.Second),
})
```

### agentapi Integration

```go
// SSE event streaming
client := agentcli.NewClient(config)
events, _ := client.SubscribeToEvents(ctx)
for event := range events {
    switch event.Type {
    case "status_changed":
        // Handle status change
    case "content_delta":
        // Handle content update
    }
}

// TUI capture
screen, _ := client.GetScreen(ctx)
fmt.Println(screen.Text)
```

---

## 10. Development Workflow

### Adding a Feature

1. **Research**: Check existing patterns in codebase
2. **Plan**: Identify files to modify, interfaces to implement
3. **Implement**: Write code following existing patterns
4. **Test**: Write unit tests, run `go test ./...`
5. **Verify**: Run `go build ./...` and `go vet ./...`

### Before Committing

```bash
go mod tidy
go build ./...
go vet ./...
go test ./...
```

---

## Quick Reference

| Command | Purpose |
|---------|---------|
| `go build ./...` | Build all packages |
| `go test ./...` | Run all tests |
| `go vet ./...` | Static analysis |
| `go mod tidy` | Clean dependencies |
| `go run ./cmd/bifrost-enhanced` | Run server |

| Type | Import |
|------|--------|
| `schemas.Plugin` | `github.com/maximhq/bifrost/core/schemas` |
| `schemas.Account` | `github.com/maximhq/bifrost/core/schemas` |
| `schemas.BifrostRequest` | `github.com/maximhq/bifrost/core/schemas` |
| `schemas.ChatMessageRole` | `github.com/maximhq/bifrost/core/schemas` |


## Child-Agent and Delegation Policy
- Use child agents liberally for scoped discovery, audits, multi-repo scans, and implementation planning before direct parent-agent edits.
- Prefer delegating high-context or high-churn tasks to subagents, and keep parent-agent changes focused on integration and finalization.
- Reserve parent-agent direct writes for the narrowest, final decision layer.

## CI Completeness Policy

- Always evaluate and fix ALL CI check failures on a PR, including pre-existing failures inherited from main.
- Never dismiss a CI failure as "pre-existing" or "unrelated to our changes" — if it fails on the PR, fix it in the PR.
- This includes: build, lint, test, docs build, security scanning (CodeQL), code review gates (CodeRabbit), workflow guard checks, and any other CI jobs.
- When a failure is caused by infrastructure outside the branch (e.g., rate limits, external service outages), implement or improve automated retry/bypass mechanisms in CI workflows.
- After fixing CI failures, verify locally where possible (build, vet, tests) before pushing.

## Phenotype Git and Delivery Workflow Protocol <!-- PHENOTYPE_GIT_DELIVERY_PROTOCOL -->

- Use branch-based delivery with pull requests; do not rely on direct default-branch writes where rulesets apply.
- Prefer stacked PRs for multi-part changes so each PR is small, reviewable, and independently mergeable.
- Keep PRs linear and scoped: one concern per PR, explicit dependency order for stacks, and clear migration steps.
- Enforce CI and required checks strictly: do not merge until all required checks and policy gates are green.
- Resolve all review threads and substantive PR comments before merge; do not leave unresolved reviewer feedback.
- Follow repository coding standards and best practices (typing, tests, lint, docs, security) before requesting merge.
- Rebase or restack to keep branches current with target branch and to avoid stale/conflicting stacks.
- When a ruleset or merge policy blocks progress, surface the blocker explicitly and adapt the plan (for example: open PR path, restack, or split changes).

## Phenotype Org Cross-Project Reuse Protocol <!-- PHENOTYPE_SHARED_REUSE_PROTOCOL -->

- Treat this repository as part of the broader Phenotype organization project collection, not an isolated codebase.
- During research and implementation, actively identify code that is sharable, modularizable, splittable, or decomposable for reuse across repositories.
- When reusable logic is found, prefer extraction into existing shared modules/projects first; if none fit, propose creating a new shared module/project.
- Include a  section in plans with candidate code, target shared location, impacted repos, and migration order.
- For cross-repo moves or ownership-impacting extractions, ask the user for confirmation on destination and rollout, then bake that into the execution plan.
- Execute forward-only migrations: extract shared code, update all callers, and remove duplicated local implementations.

## Phenotype Long-Term Stability and Non-Destructive Change Protocol <!-- PHENOTYPE_LONGTERM_STABILITY_PROTOCOL -->

- Optimize for long-term platform value over short-term convenience; choose durable solutions even when implementation complexity is higher.
- Classify proposed changes as `quick_fix` or `stable_solution`; prefer `stable_solution` unless an incident response explicitly requires a temporary fix.
- Do not use deletions/reversions as the default strategy; prefer targeted edits, forward fixes, and incremental hardening.
- Prefer moving obsolete or superseded material into `.archive/` over destructive removal when retention is operationally useful.
- Prefer clean manual merges, explicit conflict resolution, and auditable history over forceful rewrites, force merges, or history-destructive workflows.
- Prefer completing unused stubs into production-quality implementations when they represent intended product direction; avoid leaving stubs ignored indefinitely.
- Do not merge any PR while any check is failing, including non-required checks, unless the user gives explicit exception approval.
- When proposing a quick fix, include a scheduled follow-up path to a stable solution in the same plan.

## Worktree Discipline

- Feature work goes in `.worktrees/<topic>/`
- Legacy `PROJECT-wtrees/` and `repo-wtrees/` roots are for migration only and must not receive new work.
- Canonical repository remains on `main` for final integration and verification.
