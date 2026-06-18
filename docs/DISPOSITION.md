# Disposition — bifrost (Koosha fork)

**Status:** Active (Wave H / G17)  
**Disposition:** DYNAMIC-KEEP → phenotype-gateway (`packages/bifrost`)  
**Registry row:** `gw-bifrost` in [disposition-index.json](https://github.com/KooshaPari/phenotype-registry/blob/main/registry/disposition-index.json)  
**Upstream:** [maximhq/bifrost](https://github.com/maximhq/bifrost)  
**Vendor policy:** [`VENDOR_PIN.md`](./VENDOR_PIN.md)  
**Local delta:** [`LOCAL_DELTA.md`](./LOCAL_DELTA.md)

---

## Assessment

| Module / area | Disposition | Target | Rationale |
|---------------|-------------|--------|-----------|
| Koosha fork `main` | DYNAMIC-KEEP | bifrost (this repo) | Vendor-tagged inference gateway; not upstream feature sprawl |
| Upstream maximhq/bifrost | TRACK | Quarterly sync | Baseline `transports/v1.5.15`; no ~300-remote merge |
| Koosha-local deltas | DYNAMIC-KEEP | `feat/bifrost-local-delta` | phenotype-gateway parity only |
| phenotype-gateway submodule | ABSORB | `third_party/bifrost` | Pin SHA after local delta merge |
| argis plugin plane | PEER | argis-extensions | MCP/SLM plugins — not bifrost core |

---

## Vendor pin (G17)

| Field | Value |
|-------|-------|
| koosha_vendor_tag | `phenotype/vendor-2026-06` |
| koosha_vendor_sha | `677c1ae21` |
| local_delta_branch | `feat/bifrost-local-delta` |
| PR | bifrost#6 (LOCAL_DELTA), bifrost#7 (vendor tag) |

Do **not** merge upstream remote sprawl. Cherry-pick Koosha deltas only.

---

## Stack

| Tier | Language | Justification |
|------|----------|---------------|
| Core | Go | Upstream bifrost inference gateway |
| Edge | Python sidecars (argis) | Research/prompt sidecars — owned by argis-extensions |

---

## verify

```bash
go build ./...
# Smoke: phenotype-gateway spikes/go/bifrost/
```

---

## Branch hygiene

Target ≤5 remotes: `main`, `feat/bifrost-local-delta`, one active `feat/*` lane. See [`BRANCH_INVENTORY.md`](./BRANCH_INVENTORY.md), [`BRANCH_PRUNE.md`](./BRANCH_PRUNE.md).

---

## FSM

| Field | Value |
|-------|-------|
| Wave | H |
| fsm | `done` |
| relocated_date | 2026-06-18 |
