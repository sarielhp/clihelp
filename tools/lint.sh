#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export PATH="$HOME/go/bin:$PATH"

echo "=== Vet ==="
go vet ./...

echo "=== Staticcheck ==="
staticcheck ./...
