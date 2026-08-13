#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "=== Formatting ==="
gofmt -s -w .

echo "=== Tidy ==="
go mod tidy

echo "=== Vet ==="
go vet ./...

echo "=== Staticcheck ==="
staticcheck ./...

echo "=== Test ==="
go test -timeout 30s ./...

echo "=== Build Example ==="
go build -o /dev/null ./example

echo ""
echo "All checks passed."
