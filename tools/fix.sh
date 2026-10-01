#!/usr/bin/env bash
# Repair what tools/check.sh reports: format the sources and tidy the module.
# This rewrites files; check.sh does not.
set -euo pipefail
cd "$(dirname "$0")/.."
export PATH="$HOME/.go/bin:$HOME/go/bin:$PATH"

echo "=== Formatting ==="
gofmt -s -w .

echo "=== Tidy ==="
go mod tidy
