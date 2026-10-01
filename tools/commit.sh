#!/usr/bin/env bash
# Check the message, repair formatting, run the quality gate, then stage and commit. Silent on
# success; on failure it prints the output of the step that failed to stderr and
# exits non-zero.
#
# The message must be a conventional commit (feat:, fix:, docs:, test:, chore:,
# refactor:, perf:, build:, ci:, style:, revert:, optionally scoped or marked
# breaking), and it is checked before the gate so a message that cannot be used
# costs nothing.
#
# Tracked files are staged as they are (edits and deletions). A new, untracked
# file is never swept up by accident: stage it yourself with `git add`, or admit
# every untracked file with COMMIT_ADD_UNTRACKED=1.
#
# COMMIT_TRAILER, when set, is appended to the message as its own paragraph, e.g.
#   COMMIT_TRAILER="Co-Authored-By: Name <name@example.com>" make commit ARGS="..."
set -euo pipefail
cd "$(dirname "$0")/.."

msg="$*"
if [ -z "${msg//[[:space:]]/}" ]; then
  echo "commit.sh: a commit message is required (usage: $0 <commit-message>)" >&2
  exit 1
fi

subject="${msg%%$'\n'*}"
conventional='^(feat|fix|docs|test|chore|refactor|perf|build|ci|style|revert)(\([^)]+\))?!?: .*[^[:space:]]'
if ! [[ "$subject" =~ $conventional ]]; then
  echo "commit.sh: the message must be a conventional commit, e.g. \"fix: wrap early\" (got: $subject)" >&2
  exit 1
fi

if [ "${COMMIT_ADD_UNTRACKED:-}" != 1 ]; then
  untracked="$(git ls-files --others --exclude-standard)"
  if [ -n "$untracked" ]; then
    {
      echo "commit.sh: untracked files are present, and would be left out:"
      sed 's/^/  /' <<< "$untracked"
      echo "Stage the ones you mean with 'git add <file>', or commit all of them with COMMIT_ADD_UNTRACKED=1."
    } >&2
    exit 1
  fi
fi

log="$(mktemp)"
trap 'rm -f "$log"' EXIT

run() {
  if ! "$@" > "$log" 2>&1; then
    cat "$log" >&2
    echo "commit.sh: failed: $*" >&2
    exit 1
  fi
}

run bash tools/fix.sh
run bash tools/check.sh
if [ "${COMMIT_ADD_UNTRACKED:-}" = 1 ]; then
  run git add -A
else
  run git add -u
fi
if git diff --cached --quiet; then
  echo "commit.sh: nothing to commit" >&2
  exit 1
fi
if [ -n "${COMMIT_TRAILER:-}" ]; then
  run git commit -m "$msg" -m "$COMMIT_TRAILER"
else
  run git commit -m "$msg"
fi

echo "Success $msg"
