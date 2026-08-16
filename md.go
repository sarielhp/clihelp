package clihelp

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// markdownFormatVersion is baked into the output hash so that a future change
// to the markdown layout forces regeneration even when the usage tree is
// otherwise unchanged.
const markdownFormatVersion = 1

// markdownHashName is the ignore-on-git sidecar file that doubles as both the
// "generation is enabled here" flag and the record of the last generated state.
const markdownHashName = ".clihelp-hash"

// envGenDocs, when set, forces generation and bootstraps the hash file. It is
// the explicit developer opt-in that a deployed environment will not provide.
const envGenDocs = "CLIHELP_GEN"

// defaultMarkdownDir is used when MarkdownOptions.Dir is empty.
const defaultMarkdownDir = "docs/clihelp"

// MarkdownOptions controls markdown help-page generation.
type MarkdownOptions struct {
	// Dir is the directory that receives the generated pages. When empty it
	// defaults to "docs/clihelp".
	Dir string
}

// RenderMarkdown generates GitHub-friendly markdown help pages for a. It owns
// exactly the directory Dir: it writes one .md file per command plus an
// index.md, prunes orphaned .md files it produced earlier (safe, it is the sole
// owner of that directory), and never touches files outside it.
//
// Generation is gated so a deployed binary (which never sets the CLIHELP_GEN
// environment variable) stays silent. The on-disk hash file both enables
// generation and records the last generated state; when the usage tree is
// unchanged the pass is a no-op. changed reports whether any generation pass
// ran. A suggestion for committing the pages is printed to stderr only when
// changed is true.
//
// Additive helper: the markdown materialized in Dir is not tracked by git (the
// per-app dotfile is gitignored), so committing and pushing the generated pages
// to the repository is a separate, ordinary `git add`/`commit`/`push` step.
func RenderMarkdown(a *App, o MarkdownOptions) (changed bool, err error) {
	dir := o.Dir
	if dir == "" {
		dir = defaultMarkdownDir
	}

	pages, err := renderMarkdownPages(a)
	if err != nil {
		return false, err
	}
	newHash := markdownHash(pages)

	force := os.Getenv(envGenDocs) != ""
	if !force {
		stored, err := os.ReadFile(filepath.Join(dir, markdownHashName))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return false, nil // not bootstrapped / deployed
			}
			return false, err
		}
		if string(stored) == newHash {
			return false, nil // up to date
		}
	}

	if err := writeMarkdownPages(dir, pages); err != nil {
		return false, err
	}
	hashPath := filepath.Join(dir, markdownHashName)
	if err := os.WriteFile(hashPath, []byte(newHash), 0o644); err != nil {
		return false, err
	}
	ensureHashIgnored(dir, hashPath)
	fmt.Fprintf(os.Stderr,
		"help pages generated under %s/\nNext step (choose one):\n"+
			"  manual:    git add -A %s && git commit && git push\n"+
			"  automated: add the above to a release workflow\n", dir, dir)
	return true, nil
}

// cmdNode carries a command together with the full path leading to it.
type cmdNode struct {
	path []string
	cmd  Command
}

// collectNodes gathers every command (including nested subcommands) into a flat
// list, walking both the top-level command tree and the shortcut commands.
func collectNodes(a *App) []cmdNode {
	var out []cmdNode
	for _, c := range a.Commands {
		collect(c, nil, &out)
	}
	for _, s := range a.Shortcuts {
		collect(s, nil, &out)
	}
	return out
}

func collect(c Command, prefix []string, out *[]cmdNode) {
	path := append(append([]string{}, prefix...), c.Name)
	*out = append(*out, cmdNode{path: path, cmd: c})
	for _, sub := range c.Subcommands {
		collect(sub, path, out)
	}
}

// renderMarkdownPages materializes every page in memory, keyed by its
// directory-relative filename (e.g. "index.md", "config-set-time.md").
func renderMarkdownPages(a *App) (map[string]string, error) {
	pages := map[string]string{"index.md": renderIndex(a)}
	for _, n := range collectNodes(a) {
		pages[markdownRelFile(n.path)] = renderCommandPage(a, n)
	}
	return pages, nil
}

// markdownHash returns a deterministic content hash over every page. The app
// version participates implicitly through the index page content, so bumping
// the version invalidates the cache even when no command changed.
func markdownHash(pages map[string]string) string {
	keys := make([]string, 0, len(pages))
	for k := range pages {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	h := sha256.New()
	fmt.Fprintf(h, "clihelp-md-v%d\n", markdownFormatVersion)
	for _, k := range keys {
		fmt.Fprintf(h, "== %s ==\n%s\n", k, pages[k])
	}
	return hex.EncodeToString(h.Sum(nil))
}

// markdownSlug converts an arbitrary command name into a safe path segment.
func markdownSlug(seg string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(seg) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			dash = false
		} else if !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// markdownRelFile maps a command path to its flat filename (e.g. "config set"
// -> "config-set.md"). Slug segments are joined with '-' so nesting does not
// require subdirectories.
func markdownRelFile(path []string) string {
	if len(path) == 0 {
		return "index.md"
	}
	segs := make([]string, len(path))
	for i, p := range path {
		segs[i] = markdownSlug(p)
	}
	return strings.Join(segs, "-") + ".md"
}

// mdInline escapes text for use inside a markdown paragraph or list item.
func mdInline(s string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`*`, `\*`,
		`_`, `\_`,
		"`", "\\`",
		"[", "\\[",
		"]", "\\]",
		"<", "\\<",
	).Replace(strings.TrimSpace(s))
}

// mdCode wraps s as inline code, escaping backticks so they cannot break out.
func mdCode(s string) string { return "`" + strings.ReplaceAll(s, "`", "\\`") + "`" }

// renderIndex renders the top-level application overview page.
func renderIndex(a *App) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", mdInline(a.Name))
	if a.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", a.Description)
	}

	if len(a.Commands) > 0 {
		b.WriteString("## Commands\n\n| Command | Description |\n|---------|-------------|\n")
		for _, c := range a.Commands {
			desc := c.Description
			if desc == "" {
				desc = "—"
			}
			fmt.Fprintf(&b, "| [%s](%s) | %s |\n", mdInline(displayName(c)), markdownSlug(c.Name)+".md", desc)
		}
		b.WriteString("\n")
	}

	if len(a.Shortcuts) > 0 {
		b.WriteString("## Shortcut Commands\n\n| Command | Description |\n|---------|-------------|\n")
		for _, s := range a.Shortcuts {
			desc := s.Description
			if desc == "" {
				desc = "—"
			}
			fmt.Fprintf(&b, "| [%s](%s) | %s |\n", mdInline(displayName(s)), markdownSlug(s.Name)+".md", desc)
		}
		b.WriteString("\n")
	}

	if len(a.GlobalFlags) > 0 {
		b.WriteString("## Global Flags\n\n| Flag | Description |\n|------|-------------|\n")
		for _, f := range a.GlobalFlags {
			fmt.Fprintf(&b, "| %s | %s |\n", mdCode(f.Flags), f.Description)
		}
		b.WriteString("\n")
	}

	if a.Version != "" {
		fmt.Fprintf(&b, "## Version\n\n%s\n\n", mdInline(a.Version))
	}
	if a.ConfigPath != "" {
		fmt.Fprintf(&b, "## Config file location\n\n%s\n\n", mdCode(a.ConfigPath))
	}
	if a.GlobalNote != "" {
		b.WriteString("## About\n\n")
		fmt.Fprintf(&b, "%s\n\n", a.GlobalNote)
	}
	return b.String()
}

// renderCommandPage renders the detailed page for a single command.
func renderCommandPage(a *App, n cmdNode) string {
	cmd := n.cmd
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", mdInline(title(&cmd)))
	if cmd.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", cmd.Description)
	}

	if cmd.UsageLine != "" {
		b.WriteString("## Usage\n\n```\n")
		b.WriteString(cmd.UsageLine)
		b.WriteString("\n```\n\n")
	}

	if subs := subcommandEntries(&cmd); len(subs) > 0 {
		b.WriteString("## Subcommands\n\n| Command | Description |\n|---------|-------------|\n")
		for _, s := range subs {
			file := ""
			for i := range cmd.Subcommands {
				if cmd.Subcommands[i].Name == s.Name {
					p := append(append([]string{}, n.path...), s.Name)
					file = markdownRelFile(p)
					break
				}
			}
			desc := s.Description
			if desc == "" {
				desc = "—"
			}
			if file != "" {
				fmt.Fprintf(&b, "| [%s](%s) | %s |\n", mdInline(s.Name), file, desc)
			} else {
				fmt.Fprintf(&b, "| %s | %s |\n", mdInline(s.Name), desc)
			}
		}
		b.WriteString("\n")
	}

	if len(cmd.Parameters) > 0 {
		b.WriteString("## Parameters\n\n| Parameter | Description |\n|-----------|-------------|\n")
		for _, p := range cmd.Parameters {
			fmt.Fprintf(&b, "| %s | %s |\n", mdCode(p.Name), p.Description)
		}
		b.WriteString("\n")
	}

	if len(cmd.Options) > 0 {
		b.WriteString("## Flags\n\n| Flag | Description |\n|------|-------------|\n")
		for _, f := range cmd.Options {
			fmt.Fprintf(&b, "| %s | %s |\n", mdCode(f.Flags), f.Description)
		}
		b.WriteString("\n")
	}

	if len(cmd.Examples) > 0 {
		b.WriteString("## Examples\n\n")
		for _, ex := range cmd.Examples {
			line := fmt.Sprintf("- %s", mdCode(ex.Line))
			if ex.Description != "" {
				line += " — " + ex.Description
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	for _, note := range cmd.Notes {
		if note.Heading != "" {
			fmt.Fprintf(&b, "## %s\n\n", mdInline(note.Heading))
		}
		fmt.Fprintf(&b, "%s\n\n", note.Text)
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

// writeMarkdownPages writes every page under dir and prunes stale .md pages
// that no longer correspond to a command (dir is generator-owned).
func writeMarkdownPages(dir string, pages map[string]string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for name, content := range pages {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if _, ok := pages[e.Name()]; ok {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// ensureHashIgnored keeps the hash sidecar out of git so it is never committed
// or pushed. It asks git rather than parsing .gitignore itself, because only git
// knows its own ignore semantics (patterns, negation, nesting, global excludes).
// Outside a git work tree it silently does nothing.
func ensureHashIgnored(dir, hashPath string) {
	cmd := exec.Command("git", "check-ignore", "-q", hashPath)
	if err := cmd.Run(); err == nil {
		return // already ignored
	} else {
		var ee *exec.ExitError
		if !errors.As(err, &ee) || ee.ExitCode() != 1 {
			return // git unavailable / not a repo
		}
	}

	gi := filepath.Join(dir, ".gitignore")
	if data, err := os.ReadFile(gi); err == nil && bytes.Contains(data, []byte(filepath.Base(hashPath))) {
		return
	}
	f, err := os.OpenFile(gi, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "\n%s\n", filepath.Base(hashPath))
}
