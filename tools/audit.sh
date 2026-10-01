#!/usr/bin/env bash
# Hold the sources to the size, nesting and complexity limits in AGENTS.md.
#
# go-audit (the standards tool) measures function length, nesting depth and
# branch count, and file length; it is used when it is installed. Without it the
# Ruby script is the fallback, and it measures length only.
set -euo pipefail
cd "$(dirname "$0")/.."

audit="$(command -v go-audit || true)"
if [ -z "$audit" ] && [ -x /home/sariel/prog/standards/go/bin/go-audit ]; then
    audit=/home/sariel/prog/standards/go/bin/go-audit
fi

if [ -n "$audit" ]; then
    exec "$audit" --strict -q .
fi

echo "go-audit is not installed: falling back to tools/audit_lines.rb, which checks length only (no nesting or complexity)." >&2
if command -v ruby > /dev/null; then
    exec ./tools/audit_lines.rb
fi
echo "ruby is not installed either: the size limits were not checked." >&2
