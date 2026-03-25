#!/usr/bin/env bash
set -euo pipefail

# Quality gate: build, vet, and test the Go project.

cmd="${1:-verify}"

case "$cmd" in
  verify)
    echo "==> go vet ./..."
    go vet ./...
    echo "==> go build ./..."
    go build ./...
    echo "==> go test ./..."
    go test ./...
    echo "Quality gate passed."
    ;;
  *)
    echo "Usage: $0 {verify}" >&2
    exit 1
    ;;
esac
