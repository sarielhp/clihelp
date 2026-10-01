#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

if [[ -n $(git status --porcelain) ]]; then
    # Errors stay visible: with stderr discarded, a failed checkpoint (no git
    # identity, a hook, a locked index) exited 1 having said nothing at all.
    git add -A
    git commit -q -m "wip: checkpoint [$(date +'%H:%M:%S')]" --no-verify
    echo "Checkpoint saved."
fi
