# Koosha local delta — bifrost

**Vendor policy:** [VENDOR_PIN.md](./VENDOR_PIN.md)  
**Branch:** `feat/bifrost-local-delta`  
**Submodule pin:** `f9cec7bb` (phenotype-gateway `third_party/bifrost`)

## Scope

Port **Koosha-local changes only** — do not merge maximhq/bifrost feature-branch sprawl.

## Local delta inventory (initial)

| Area | Status | Notes |
|------|--------|-------|
| phenotype-gateway parity | pending | Map bifrost features in GATEWAY_FEATURE_PARITY |
| Authvault integration | pending | Cross-cutting auth via Rust crates git pin |
| MCP / guardrails | upstream | Track vendor tag updates |
| Koosha deploy hooks | TBD | Diff vs `maximhq/bifrost` tag when selected |

## Workflow

1. Select upstream baseline tag → record in `VENDOR_PIN.md`
2. `git diff <upstream-tag>..main` on Koosha fork → extract local commits
3. Cherry-pick local commits onto `feat/bifrost-local-delta`
4. Bump phenotype-gateway submodule SHA after gate passes

## Gate

- `go build ./...`
- Smoke test against phenotype-gateway `spikes/go/bifrost/`
