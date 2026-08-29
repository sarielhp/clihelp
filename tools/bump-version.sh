#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

current=$(cat VERSION)
IFS='.' read -r major minor patch <<< "$current"
patch=$((patch + 1))
new="$major.$minor.$patch"
tag="v$new"
echo "$new" > VERSION
ruby -pi -e "sub(/Version:\s+\"[^\"]+\"/, %Q{Version:        \"$new\"})" example/main.go

bash tools/check.sh > /dev/null 2>&1

git add VERSION example/main.go > /dev/null 2>&1
git commit -m "chore: bump version to $new" > /dev/null 2>&1
git tag -a "$tag" -m "Release $tag" > /dev/null 2>&1
git push > /dev/null 2>&1
git push origin "$tag" > /dev/null 2>&1

# Prime account-wide Go module cache (~/.go/pkg/mod)
GOPROXY=direct go install github.com/sarielhp/clihelp/example@"$tag" > /dev/null 2>&1 || true

echo "Success $new (commit+tag+push+cached)"
