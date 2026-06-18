# Koosha local delta — bifrost

**Vendor policy:** [VENDOR_PIN.md](./VENDOR_PIN.md)  
**Branch:** `feat/bifrost-local-delta`  
**Submodule pin:** `677c1ae` (phenotype-gateway `third_party/bifrost` — bump from `f9cec7bb`)

## Upstream baseline

| Field | Value |
|-------|-------|
| maximhq tag | `transports/v1.5.15` |
| Vendor policy | See [VENDOR_PIN.md](./VENDOR_PIN.md) |

## Local delta inventory (initial)

| Area | Status | Notes |
|------|--------|-------|
| phenotype-gateway parity | in_progress | GATEWAY_FEATURE_PARITY + submodule pin |
| docs/VENDOR_PIN.md | done | #5 — baseline tag `transports/v1.5.15` (H12) |
| docs/LOCAL_DELTA.md | done | #6 scaffold; inventory updated H12 |

## Workflow

1. Select upstream baseline tag → record in `VENDOR_PIN.md`
2. `git diff <upstream-tag>..main` on Koosha fork → extract local commits
3. Cherry-pick local commits onto `feat/bifrost-local-delta`
4. Bump phenotype-gateway submodule SHA after gate passes

## Gate

- `go build ./...`
- Smoke test against phenotype-gateway `spikes/go/bifrost/`
