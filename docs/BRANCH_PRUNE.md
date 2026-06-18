# Branch prune ledger — G17 (2026-06-18)

Wave 17 vendor pin hygiene. Pre-audit: **~339** remote branches (upstream mirror sprawl).

## Keep (target ≤5)

| Branch | Role |
|--------|------|
| `main` | Canonical vendor pin + tag `phenotype/vendor-2026-06` |
| `feat/bifrost-local-delta` | Koosha-local delta port lane |
| `feat/wave17-g17-vendor-pin` | G17 docs + prune PR (delete after merge) |

## Delete policy

Removed categories (no unique Koosha value post-vendor-pin):

- Upstream dated feature branches (`01-03-*`, `04-*`, etc.)
- `graphite-base/*` stack branches
- `snyk-*` / `coderabbitai/*` bot branches
- `gh-pages`, `v1.*` release mirrors, WIP/draft branches
- Stale `chore/*` and `feat/wave-h4-*` superseded by G17

## Post-merge

After PR merge, expect **2** remotes: `main`, `feat/bifrost-local-delta`.
