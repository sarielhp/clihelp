#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
export PATH="$HOME/.go/bin:$HOME/go/bin:$PATH"

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
lib=$(grep -oP 'const Version = "\K[^"]+' clihelp.go || true)
if [ "$want" != "$lib" ]; then
    echo "ERROR: clihelp.go const Version=$lib but VERSION says $want" >&2
    exit 1
fi

# A shell, man or sh that is missing must fail the gate, not skip its tests in
# silence; set CLIHELP_REQUIRE_SHELLS=0 to run on a machine without them.
export CLIHELP_REQUIRE_SHELLS="${CLIHELP_REQUIRE_SHELLS:-1}"

echo "=== Test (race) ==="
go test -race -timeout 300s ./...

# Ambiguous-width characters (a bullet, an em dash, an ellipsis) are two columns
# under East Asian width rules. The library measures what it draws, so layout must
# hold there too; one leg of the suite proves it.
echo "=== Test (East Asian widths) ==="
RUNEWIDTH_EASTASIAN=1 go test -count=1 -timeout 300s .

echo "=== Build Example ==="
go build -o /dev/null ./example

echo ""
echo "All checks passed."
