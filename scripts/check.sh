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

echo "=== Version Drift ==="
want=$(cat VERSION)
have=$(grep -oP 'Version:\s+"\K[^"]+' example/main.go || true)
if [ -z "$have" ]; then
    echo "ERROR: could not find a Version literal in example/main.go" >&2
    exit 1
fi
if [ "$want" != "$have" ]; then
    echo "ERROR: example/main.go Version=$have but VERSION says $want" >&2
    exit 1
fi

echo "=== Test ==="
go test -timeout 30s ./...

echo "=== Build Example ==="
go build -o /dev/null ./example

echo ""
echo "All checks passed."
