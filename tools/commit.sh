#!/usr/bin/env bash
# Quality gate, then stage and commit. Silent on success; on failure it prints
# the output of the step that failed to stderr and exits non-zero.
set -euo pipefail
cd "$(dirname "$0")/.."

if [ $# -eq 0 ]; then
  echo "Usage: $0 <commit-message>" >&2
  exit 1
fi

msg="$*"

log="$(mktemp)"
trap 'rm -f "$log"' EXIT

run() {
  if ! "$@" > "$log" 2>&1; then
    cat "$log" >&2
    echo "commit.sh: failed: $*" >&2
    exit 1
  fi
}

run bash tools/check.sh
run git add -A
run git commit -m "$msg"

echo "Success $msg"
