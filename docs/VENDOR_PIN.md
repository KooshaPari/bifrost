# Vendor pin — bifrost

**Upstream:** https://github.com/maximhq/bifrost  
**Koosha fork:** KooshaPari/bifrost

## Policy (Wave H4 / G17)

- Do **not** merge upstream feature-branch sprawl (~300 remotes)
- Pin Koosha vendor tag `phenotype/vendor-2026-06` on fork `main`
- Port **Koosha-local deltas only** to `feat/bifrost-local-delta`
- phenotype-gateway absorbs via submodule → `third_party/bifrost`

## Baseline (G17 — 2026-06-18)

| Field | Value |
|-------|-------|
| koosha_vendor_tag | `phenotype/vendor-2026-06` |
| koosha_vendor_sha | `677c1ae21` (main @ tag) |
| upstream_remote | `maximhq/bifrost` |
| upstream_track | `dev` branch / `transports/v1.5.x` tags (quarterly sync) |
| upstream_tag_at_pin | `transports/v1.5.15` (reference; fork not rebased this wave) |
| local_delta_branch | `feat/bifrost-local-delta` |
| submodule_pin | Update phenotype-gateway `third_party/bifrost` after merge |

## Quarterly sync

1. `git fetch upstream` (add `maximhq/bifrost` if missing)
2. Diff `phenotype/vendor-*` tag vs upstream `dev` HEAD
3. Cherry-pick Koosha deltas onto new vendor tag branch
4. Re-tag `phenotype/vendor-YYYY-MM` on merged `main`
5. Bump phenotype-gateway submodule SHA

## Branch hygiene (G17)

Target ≤5 remotes: `main`, `feat/bifrost-local-delta`, optional active `feat/*` lane only.

See [BRANCH_INVENTORY.md](./BRANCH_INVENTORY.md) and [BRANCH_PRUNE.md](./BRANCH_PRUNE.md).
