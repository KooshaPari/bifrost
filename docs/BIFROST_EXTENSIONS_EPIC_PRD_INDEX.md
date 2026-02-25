# Bifrost‑Extensions – Epic / PRD Index

**Purpose:** map Bifrost‑Extensions gateway/plugins to the unified
Epics/PRDs.

- Backlog: `plans/012-epics-backlog.md`
- Architecture: `docs/reference/UNIFIED_ARCHITECTURE.md`
- Epic PRDs: `docs/unified-specifications/E*.md`

---

## 1. Epic Mapping for Bifrost‑Extensions

| Epic | Name                                   | Canonical PRD                                                                 | Local Scope (bifrost-extensions)                         |
|------|----------------------------------------|-------------------------------------------------------------------------------|----------------------------------------------------------|
| E2   | Bifrost Gateway Integration            | `../../docs/unified-specifications/E2_BIFROST_GATEWAY_PRD.md`                | `server/`, `api/`, `config/`, `infra/`                   |
| E6   | Embedding Pipeline & Semantic Fast-Path| `../../docs/unified-specifications/E6_EMBEDDING_PIPELINE_PRD.md`             | `plugins/embeddings/`, `plugins/contextfolding/`         |
| E7   | Governance & Cost Engine               | `../../docs/unified-specifications/E7_GOVERNANCE_COST_PRD.md`                | `costengine/`, `auth/`, governance‑related plugins       |
| E11  | Observability & Telemetry              | `../../docs/unified-specifications/E11_OBSERVABILITY_TELEMETRY_PRD.md`       | `infra/`, `performance/`, tests under `tests/performance/` |
| E4   | SLM Portfolio                          | `../../docs/unified-specifications/E4_SLM_PORTFOLIO_PRD.md`                  | `slm-server/`, `slm/`                                    |

---

## 2. Local Documentation Anchors

- Project README: `bifrost-extensions/README.md`
- Docs index: `bifrost-extensions/docs/README.md`
- AI/ML architecture: `bifrost-extensions/docs/AI_ML_ARCHITECTURE.md`

Use this as the bridge from **Epics/PRDs → concrete gateway/plugins code**.

