#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

current=$(cat VERSION)
IFS='.' read -r major minor patch <<< "$current"
patch=$((patch + 1))
new="$major.$minor.$patch"
tag="v$new"

# The three files that carry the version. They are written before the check
# runs, because the check verifies they agree with VERSION.
versioned=(VERSION example/main.go clihelp.go)

# A failed bump used to leave the new version written into all three with
# nothing committed. The next run then bumped again from there and skipped a
# version outright — and since the failure was silent, there was nothing to
# suggest looking. Put them back, whatever fails, up until the commit lands.
# Only the two generated directories, and only if they were clean when we
# started: reverting docs/ wholesale would throw away hand-written edits to
# docs/completion.md and its neighbours, which live in the same tree.
generated=(docs/clihelp docs/mail_cli_fake)
generated_were_clean=1
[ -n "$(git status --porcelain -- "${generated[@]}")" ] && generated_were_clean=0

restore() {
    git checkout -- "${versioned[@]}" 2>/dev/null || true
    [ "$generated_were_clean" = 1 ] && git checkout -- "${generated[@]}" 2>/dev/null || true
}
trap restore ERR

log=$(mktemp -t clihelp-bump-XXXXXX)

echo "$new" > VERSION
ruby -pi -e "sub(/Version:\s+\"[^\"]+\"/, %Q{Version:        \"$new\"})" example/main.go
ruby -pi -e "sub(/const Version = \"[^\"]+\"/, %Q{const Version = \"$new\"})" clihelp.go

# The generated documentation embeds App.Version, so it is stale the moment the
# version changes. Nothing used to regenerate it at release, which left the docs
# one version behind every time. Regenerate before the check, so the check sees
# the tree that is about to be committed.
if ! CLIHELP_GEN=1 CLIHELP_NO_AUTO_COMPLETION=1 go run ./example > "$log" 2>&1; then
    restore
    trap - ERR
    echo "bump aborted at $new: regenerating docs/clihelp failed. Version files restored." >&2
    tail -n 25 "$log" >&2
    exit 1
fi
if ! CLIHELP_GEN=1 CLIHELP_NO_AUTO_COMPLETION=1 go run ./example/mail_cli_fake >> "$log" 2>&1; then
    restore
    trap - ERR
    echo "bump aborted at $new: regenerating docs/mail_cli_fake failed. Version files restored." >&2
    tail -n 25 "$log" >&2
    exit 1
fi

# The check's output went to /dev/null, so a release that failed here reported
# nothing but "Error 1" — no failing step, no test name, nothing to act on.
if ! bash tools/check.sh > "$log" 2>&1; then
    restore
    trap - ERR
    echo "bump aborted at $new: tools/check.sh failed. Version files restored." >&2
    echo "full output: $log" >&2
    echo >&2
    tail -n 25 "$log" >&2
    exit 1
fi

git add "${versioned[@]}" "${generated[@]}"
git commit -q -m "chore: bump version to $new"
# Past here the version is committed; restoring the files would empty the very
# commit just made, so the trap comes off and each step reports for itself.
trap - ERR

if ! git tag -a "$tag" -m "Release $tag"; then
    echo "bump: the version commit landed, but tag $tag could not be created." >&2
    echo "Fix the tag, then push both by hand." >&2
    exit 1
fi

if ! git push --quiet; then
    echo "bump: committed and tagged $tag locally, but the branch did not push." >&2
    echo "Run: git push && git push origin $tag" >&2
    exit 1
fi

if ! git push --quiet origin "$tag"; then
    echo "bump: the branch pushed but tag $tag did not." >&2
    echo "Run: git push origin $tag" >&2
    exit 1
fi

# Prime account-wide Go module cache (~/.go/pkg/mod). Best effort: the release
# is already published, so a cold cache is not a failure.
GOPROXY=direct go install github.com/sarielhp/clihelp/example@"$tag" > /dev/null 2>&1 || true

rm -f "$log"
echo "Success $new (commit+tag+push+cached)"
