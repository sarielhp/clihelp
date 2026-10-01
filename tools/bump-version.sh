#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# Start only from a tree whose tracked files are all committed. restore() below
# runs `git checkout` on the three version files, which silently throws away an
# uncommitted edit to any of them (clihelp.go is the library's core file), and a
# successful bump commits those files whole, so an unrelated edit would ship
# inside a tagged, pushed release. Untracked files cannot be touched by either,
# so they do not block a release.
if [ -n "$(git status --porcelain --untracked-files=no)" ]; then
    echo "bump: tracked files have uncommitted changes; commit or stash them first:" >&2
    git status --short --untracked-files=no >&2
    exit 1
fi

current=$(cat VERSION)
IFS='.' read -r major minor patch <<< "$current"
patch=$((patch + 1))
new="$major.$minor.$patch"
tag="v$new"

# The files that carry the version, and the changelog that records it. They are written before the check
# runs, because the check verifies they agree with VERSION.
versioned=(VERSION example/main.go clihelp.go CHANGES.md)

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
# An interrupt is a failure too. Only ERR used to restore, so Ctrl-C (or a
# killed terminal) left the new version written into every file and the next
# run bumped again from there, skipping a version — exactly what this function
# exists to prevent.
trap 'restore; trap - ERR INT TERM HUP; echo "bump interrupted; version files restored." >&2; exit 130' INT TERM HUP

log=$(mktemp -t clihelp-bump-XXXXXX)

echo "$new" > VERSION
ruby -pi -e "sub(/Version:\s+\"[^\"]+\"/, %Q{Version:        \"$new\"})" example/main.go
ruby -pi -e "sub(/const Version = \"[^\"]+\"/, %Q{const Version = \"$new\"})" clihelp.go

# Promote the changelog: what has accumulated under [Unreleased] becomes this
# release's section, dated today, and [Unreleased] starts empty again. Nothing in
# the release path used to touch CHANGES.md, so five tagged releases had no
# heading at all. A release with nothing to record is refused, and the files go
# back as they were.
if ! NEW="$new" DAY="$(date +%F)" ruby -e '
  text = File.read("CHANGES.md")
  head = /^## \[Unreleased\]\n/
  m = head.match(text) or exit 2
  rest = m.post_match
  body = rest.split(/^## \[/, 2).first
  exit 3 if body.strip.empty?
  File.write("CHANGES.md", m.pre_match + m[0] + "\n## [#{ENV["NEW"]}] - #{ENV["DAY"]}\n" + rest)
'; then
    restore
    trap - ERR INT TERM HUP
    echo "bump aborted at $new: CHANGES.md needs a non-empty '## [Unreleased]' section to promote. Version files restored." >&2
    exit 1
fi

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
# commit just made, so the traps come off and each step reports for itself.
trap - ERR INT TERM HUP

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
