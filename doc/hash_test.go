package doc

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// markdownHash is the cache key: RenderMarkdown compares it against the value
// stored beside the pages and rewrites nothing when they agree. Determinism was
// asserted; sensitivity was not, and sensitivity is the whole job. Each mutation
// below — dropping the format version, dropping the page names — left the suite
// green while making the cache miss a change it exists to catch.
func TestMarkdownHashIsSensitiveToEveryInput(t *testing.T) {
	base := map[string]string{
		"index.md":  "the index",
		"build.md":  "the build page",
		"config.md": "the config page",
	}
	want := markdownHash(base)

	t.Run("content", func(t *testing.T) {
		changed := clonePages(base)
		changed["build.md"] += " with an extra sentence"
		if markdownHash(changed) == want {
			t.Error("a changed page did not change the hash")
		}
	})

	t.Run("a renamed page", func(t *testing.T) {
		// The same content under a different name. Without the name in the hash
		// a rename is invisible, so the old file is never replaced.
		changed := clonePages(base)
		delete(changed, "build.md")
		changed["compile.md"] = base["build.md"]
		if markdownHash(changed) == want {
			t.Error("renaming a page did not change the hash")
		}
	})

	t.Run("two pages swapping content", func(t *testing.T) {
		changed := clonePages(base)
		changed["build.md"], changed["config.md"] = base["config.md"], base["build.md"]
		if markdownHash(changed) == want {
			t.Error("two pages exchanging content did not change the hash")
		}
	})

	t.Run("a new page", func(t *testing.T) {
		changed := clonePages(base)
		changed["deploy.md"] = "the deploy page"
		if markdownHash(changed) == want {
			t.Error("an added page did not change the hash")
		}
	})

	t.Run("a removed page", func(t *testing.T) {
		changed := clonePages(base)
		delete(changed, "config.md")
		if markdownHash(changed) == want {
			t.Error("a removed page did not change the hash")
		}
	})
}

// The format version is in the hash so that upgrading clihelp regenerates every
// page: the templates change, the pages must follow, and nothing else in the
// input would have moved.
func TestMarkdownHashCoversTheFormatVersion(t *testing.T) {
	pages := map[string]string{"index.md": "same content either way"}
	if markdownHashOf(pages, 1) == markdownHashOf(pages, 2) {
		t.Error("the format version does not affect the hash, so raising markdownFormatVersion would regenerate nothing")
	}
	if markdownHash(pages) != markdownHashOf(pages, markdownFormatVersion) {
		t.Error("markdownHash does not use markdownFormatVersion")
	}
}

func clonePages(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// The one external command this package runs is bounded. A repository on an
// unreachable mount, or a git that stops for credentials, would otherwise hang
// documentation generation with nothing on screen — and killing git is not
// enough on its own when git has forked and its child still holds the pipe.
func TestGitLookupIsBounded(t *testing.T) {
	dir := t.TempDir()
	stub := filepath.Join(dir, "bin")
	if err := os.MkdirAll(stub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stub, "git"), []byte("#!/bin/sh\nsleep 30\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stub+string(os.PathListSeparator)+os.Getenv("PATH"))

	start := time.Now()
	ensureHashIgnored(dir, filepath.Join(dir, ".clihelp-md-hash"))
	if elapsed := time.Since(start); elapsed > gitLookupTimeout+2*time.Second {
		t.Errorf("the git lookup ran for %v against a %v deadline", elapsed, gitLookupTimeout)
	}
}
