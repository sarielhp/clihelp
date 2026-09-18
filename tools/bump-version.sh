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
restore() { git checkout -- "${versioned[@]}" 2>/dev/null || true; }
trap restore ERR

log=$(mktemp -t clihelp-bump-XXXXXX)

echo "$new" > VERSION
ruby -pi -e "sub(/Version:\s+\"[^\"]+\"/, %Q{Version:        \"$new\"})" example/main.go
ruby -pi -e "sub(/const Version = \"[^\"]+\"/, %Q{const Version = \"$new\"})" clihelp.go

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

git add "${versioned[@]}"
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
