# Bifrost — Workspace Operating Notes

## Project Overview

Bifrost is a Phenotype ecosystem infrastructure component (Flake/Nix-based provisioning and deployment).

- **Stack**: Nix/Flakes + Make
- **Primary**: `flake.nix`, `Makefile`, `pulse.yaml`

## Build & Test

```bash
nix build      # Build the flake
make help      # Show available targets
nix flake check  # Validate flake outputs
```

## Code Quality

- Run `nix fmt` before commit
- `nix flake check` for validation
- No ad-hoc shell scripts — use Nix or Make

## Governance

- Reference: `~/.claude/CLAUDE.md` (global baseline)
- Reference: `AgilePlus/CLAUDE.md` (Phenotype org governance)
- CLAUDE.md required for all Phenotype org repos
