# bifrost-extensions Repository Audit Report

**Date:** 2026-03-02  
**Repository:** /Users/kooshapari/CodeProjects/Phenotype/repos/bifrost-extensions  
**Go Version:** 1.24.3 (primary), 1.23.0 (slm-manager)

---

## 1. Code Metrics

### Lines of Code (LoC)

| Metric | Value |
|--------|-------|
| **Total Go Files** | 130 |
| **Total Lines** | 23,712 |
| **Average LOC per File** | ~182 |

### Top 10 Largest Files

| Rank | File | Lines | Category |
|------|------|-------|----------|
| 1 | `slm-manager/ui.go` | 1,171 | **CRITICAL** |
| 2 | `slm-manager/app.go` | 674 | **HIGH** |
| 3 | `slm-manager/providers.go` | 517 | **HIGH** |
| 4 | `plugins/promptadapter/registry.go` | 504 | **HIGH** |
| 5 | `infra/neo4j/multitenant.go` | 432 | **MEDIUM** |
| 6 | `providers/agentcli/client.go` | 408 | **MEDIUM** |
| 7 | `plugins/learning/tiered.go` | 398 | **MEDIUM** |
| 8 | `slm-manager/api.go` | 388 | **MEDIUM** |
| 9 | `plugins/learning/model_profile.go` | 344 | **MEDIUM** |
| 10 | `slm-manager/services.go` | 341 | **MEDIUM** |

---

## 2. Code Complexity Analysis

### Files Over 500 Lines (High Complexity)

Only 4 files exceed 500 lines:

1. **`slm-manager/ui.go` (1,171 lines)** — CRITICAL VIOLATION
   - GUI widget factory (Fyne framework)
   - Monolithic widget composition
   - Should be split into feature-specific UI modules
   - Recommendation: Break into `dashboard.go`, `providers.go`, `logs.go`, `settings.go`

2. **`slm-manager/app.go` (674 lines)** — HIGH VIOLATION
   - Core application state and lifecycle management
   - Mixes UI coordination with business logic
   - Candidates for extraction: lifecycle management, model profile handling, config loading

3. **`slm-manager/providers.go` (517 lines)** — HIGH VIOLATION
   - Provider registry and instantiation
   - Multiple responsibilities: registry, factory, config validation
   - Should extract registry pattern to separate module

4. **`plugins/promptadapter/registry.go` (504 lines)** — BORDERLINE
   - Model behavior profile registry
   - Well-structured but dense pattern registration
   - Acceptable if functions are simple; refactor if complex functions present

---

## 3. `go vet` Results

### Critical Findings (Build-Blocking)

**Status:** ❌ BLOCKING — 46+ errors preventing build

#### Missing go.sum Entries (Infrastructure)
- `github.com/go-chi/chi/v5`
- `github.com/go-chi/cors`
- `github.com/google/uuid`
- `github.com/hatchet-dev/hatchet/pkg/client` and `/pkg/worker`
- `github.com/jackc/pgx/v5`
- `github.com/nats-io/nats.go`
- `github.com/neo4j/neo4j-go-driver/v5`
- `github.com/redis/go-redis/v9`
- `github.com/spf13/viper`
- `golang.org/x/net/http2` and `/http2/h2c`
- `github.com/bytedance/sonic`
- `github.com/valyala/fasthttp`

**Fix:** Run `go mod tidy` to regenerate go.sum

#### Broken Local Replacements (Dependency Management)
| Replacement | Status | Issue |
|------------|--------|-------|
| `github.com/maximhq/bifrost/core` | ❌ BROKEN | Missing `../../bifrost/core` directory |
| `github.com/KooshaPari/phenotype-go-config` | ❌ BROKEN | Missing `../../../template-commons/phenotype-go-config` |
| `github.com/KooshaPari/phenotype-go-middleware` | ❌ BROKEN | Missing `../../../template-commons/phenotype-go-middleware` |
| `github.com/coder/agentapi` | ❌ BROKEN | Missing `../../agentapi-plusplus` |
| `github.com/kooshapari/CLIProxyAPI/v7` | ❌ BROKEN | Missing `../../CLIProxyAPI` |

**Impact:** Cannot build or test until worktree structure is restored or replacements are corrected.

#### Location: Files Affected
- `account/account.go`
- `api/rest_handlers.go`, `api/server.go`
- `api/connect/embeddings_service.go`, `api/connect/server.go`
- `api/graphql/server.go`, `api/graphql/gen/generated.go`
- `api/graphql/resolvers/subscription.go`
- `db/db.go`
- `plugins/voyage/plugin.go`
- `cmd/bifrost/cli/server.go`, `cmd/bifrost/cli/config.go`
- `config/config.go`
- `infra/hatchet/client.go`
- `infra/nats/client.go`
- `infra/neo4j/client.go`
- `infra/redis/client.go`
- `internal/config/config.go`
- `internal/middleware/middleware.go`
- `slm-server/updater.go`

---

## 4. Hexagonal Architecture Assessment

### Structure Review

The codebase exhibits a **partial, inconsistent hexagonal architecture:**

#### Positive Signs
| Layer | Evidence | Quality |
|-------|----------|---------|
| **Ports** | `api/` (REST, GraphQL, Connect), `providers/`, `wrappers/` | ✓ Good separation |
| **Adapters** | `infra/` (Neo4j, Redis, NATS, Hatchet, Upstash) | ✓ Well-organized |
| **Plugins** | Modular plugin system in `plugins/` | ✓ Extensible |
| **Commands** | CLI separation in `cmd/`, internal routing in `internal/cli` | ✓ Clear entry points |

#### Issues & Violations
| Layer | Issue | Severity |
|-------|-------|----------|
| **Core Logic** | No explicit `core/` or `domain/` layer; logic scattered across `services/`, `infra/`, `plugins/` | 🔴 HIGH |
| **Cross-Cutting** | `internal/middleware/` depends on broken `phenotype-go-middleware` replacement | 🔴 HIGH |
| **UI Coupling** | `slm-manager/ui.go` directly imports and orchestrates many infra adapters (violates port/adapter) | 🔴 HIGH |
| **Service Layer** | `services/` exists but underutilized; most logic in `plugins/` and `infra/` | 🟠 MEDIUM |
| **Database** | `db/` provides adapters but not abstracted behind clear repository interfaces | 🟠 MEDIUM |

### Violation Examples

**Example 1: Tight Adapter Coupling in slm-manager**
```go
// slm-manager/ui.go imports directly from infra
// Should inject via ports/interfaces, not direct imports
```

**Example 2: Missing Domain Layer**
- Model profiles in `plugins/learning/model_profile.go`
- Provider definitions scattered across `providers/`, `infra/`, `plugins/`
- No unified domain model or interface contracts

**Example 3: Plugin Architecture Inconsistency**
- Some plugins implement interface patterns (`promptadapter/transforms.go`)
- Others directly reference other packages (`voyage/plugin.go` → raw HTTP calls)

---

## 5. Maintainability Issues

### Critical (Must Fix)

| Issue | Files | Impact |
|-------|-------|--------|
| **Broken go.mod replacements** | Root & slm-manager | Cannot build, test, or deploy |
| **Missing go.sum entries** | All API, infra, plugins | Dependency resolution fails |
| **Monolithic UI file (1,171 lines)** | `slm-manager/ui.go` | 20% of UI code in single file; testing, debugging, maintenance nightmare |
| **Multiple main packages (19)** | Across `cmd/`, `slm-manager/`, `slm-server/` | Unclear entry points; binaries not well-documented |

### High Priority

| Issue | Count | Risk |
|-------|-------|------|
| **Packages over 500 LOC** | 4 files | Hard to test, understand, modify |
| **Missing interface contracts** | ~30% of packages | No formal port definitions; tight coupling |
| **Init() functions** | 6 files | Hidden initialization order dependencies |
| **Unused directories** | `PROJECT-wtrees/`, `bifrost-extensions-wtrees/` (empty) | Git clutter; confusing worktree discipline |

### Medium Priority

| Issue | Scope | Recommendation |
|-------|-------|-----------------|
| **Two go.mod files** | Root + slm-manager | Decide: monorepo with internal modules or separate workspaces |
| **Scattered config loading** | `config/`, `internal/config/` | Consolidate configuration strategy |
| **Weak service layer** | `services/` only 3 packages | Expand or remove; clarify boundaries |
| **Plugin registry patterns** | `plugins/promptadapter/registry.go` | Formalize plugin interface contract |

---

## 6. Structural Recommendations

### Immediate (Blocking Delivery)

1. **Restore Local Dependencies**
   ```bash
   # Either:
   # Option A: Fix worktree structure
   # mkdir -p ../../bifrost/core ../../agentapi-plusplus ../../CLIProxyAPI ../../../template-commons/phenotype-go-{config,middleware}
   
   # Option B: Use published versions instead of local replacements
   go get github.com/maximhq/bifrost/core@latest
   go get github.com/kooshapari/phenotype-go-config@latest
   ```

2. **Run `go mod tidy`**
   ```bash
   go mod tidy
   cd slm-manager && go mod tidy
   ```

3. **Verify build**
   ```bash
   go build ./...
   go vet ./...
   ```

### Short-term (Hygiene)

4. **Break Down `slm-manager/ui.go` (1,171 lines)**
   - Extract dashboard UI → `slm-manager/ui/dashboard.go`
   - Extract providers UI → `slm-manager/ui/providers.go`
   - Extract logs UI → `slm-manager/ui/logs.go`
   - Extract settings UI → `slm-manager/ui/settings.go`
   - Keep coordinator in `slm-manager/ui.go`
   - **Goal:** Each file <300 lines

5. **Refactor `slm-manager/app.go` (674 lines)**
   - Extract lifecycle → `slm-manager/lifecycle.go`
   - Extract model management → `slm-manager/models.go`
   - Keep app structure in `slm-manager/app.go`
   - **Goal:** <400 lines

6. **Document and Enforce Worktree Discipline**
   - Remove empty `PROJECT-wtrees/` and `bifrost-extensions-wtrees/` directories
   - Per CLAUDE.md: use `bifrost-extensions-wtrees/<topic>` for feature branches only
   - Ensure canonical `bifrost-extensions/` stays on `main`

### Medium-term (Architecture)

7. **Define Explicit Ports & Adapters**
   - Create `internal/domain/` with core interfaces
   - Create `internal/ports/` for inbound/outbound contracts
   - Refactor `infra/` packages to implement ports
   - Example:
     ```go
     // internal/ports/neo4j.go
     type GraphDB interface { Query(ctx context.Context, q string) (...) }
     
     // infra/neo4j/client.go
     var _ ports.GraphDB = (*Client)(nil)
     ```

8. **Consolidate Configuration**
   - Merge `config/` and `internal/config/` strategies
   - Use viper for all config (already in dependencies)
   - Define configuration interfaces in `internal/ports/config.go`

9. **Strengthen Service Layer**
   - Move plugin orchestration logic from `plugins/` to `services/`
   - Define service interfaces in `internal/ports/services.go`
   - Example: `services/promptadapter/service.go` wraps `plugins/promptadapter/`

10. **Decide on Monorepo vs. Multi-module**
    - If staying monorepo: use subdirectory modules (`slm-manager` as a sub-module is OK)
    - If splitting: extract `slm-manager` to separate repository
    - Document decision in `ARCHITECTURE_PRINCIPLES.md`

---

## 7. Quick Fixes Checklist

- [ ] Run `go mod tidy` (root) and `go mod tidy` (slm-manager/)
- [ ] Restore or replace broken local dependencies
- [ ] Verify `go vet ./...` passes (0 errors)
- [ ] Verify `go build ./...` completes
- [ ] Remove empty `PROJECT-wtrees/` and `bifrost-extensions-wtrees/` directories
- [ ] Add `.golangci.yml` linting configuration (currently empty)
- [ ] Verify no circular imports: `go list -json ./... | jq '.Imports[] | select(. | contains("bifrost-extensions"))' | sort -u`

---

## Summary

| Category | Status | Severity |
|----------|--------|----------|
| **Build** | ❌ Broken | 🔴 CRITICAL |
| **Code Complexity** | ⚠️ 4 hot spots | 🟠 HIGH |
| **Architecture** | ⚠️ Partial hexagonal | 🟠 HIGH |
| **Dependencies** | ❌ Misaligned | 🔴 CRITICAL |
| **Documentation** | ✓ Extensive | ✓ GOOD |
| **Testing** | ❓ Unknown (blocked by build) | ? |

**Recommendation:** Unblock build immediately (go mod tidy + dependency restore), then tackle code complexity and architecture consolidation in parallel.

