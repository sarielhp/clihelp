// Package doc provides markdown documentation site generation for clihelp applications.
package doc

import (
	"context"
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
	"time"

	"github.com/sarielhp/clihelp"
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
func RenderMarkdown(a *clihelp.App, o MarkdownOptions) (changed bool, err error) {
	dir := o.Dir
	if dir == "" {
		dir = defaultMarkdownDir
	}

	pages, err := renderMarkdownPages(a)
	if err != nil {
		return false, err
	}
	newHash := markdownHash(pages)

	hashPath := filepath.Join(dir, markdownHashName)
	previous, readErr := readMarkdownState(hashPath)
	if readErr != nil && !errors.Is(readErr, fs.ErrNotExist) {
		return false, readErr
	}

	if os.Getenv(envGenDocs) == "" {
		if errors.Is(readErr, fs.ErrNotExist) {
			return false, nil // not bootstrapped / deployed
		}
		// The hash records what was generated, not what survived: a page deleted
		// by hand used to leave the hash matching and the page missing for good.
		if previous.hash == newHash && pagesPresent(dir, pages) {
			return false, nil // up to date
		}
	}

	if err := writeMarkdownPages(dir, pages, previous.pages); err != nil {
		return false, err
	}
	if err := writeMarkdownState(hashPath, markdownState{hash: newHash, pages: pageNames(pages)}); err != nil {
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
	cmd  clihelp.Command
}

// collectNodes gathers every command (including nested subcommands) into a flat
// list, walking both the top-level command tree and the shortcut commands.
func collectNodes(a *clihelp.App) []cmdNode {
	var out []cmdNode
	for _, c := range a.Commands {
		collect(c, nil, &out)
	}
	for _, s := range a.Shortcuts {
		collect(s, nil, &out)
	}
	return out
}

func collect(c clihelp.Command, prefix []string, out *[]cmdNode) {
	if c.Hidden {
		return
	}
	path := append(append([]string{}, prefix...), c.Name)
	*out = append(*out, cmdNode{path: path, cmd: c})
	for _, sub := range c.Subcommands {
		collect(sub, path, out)
	}
}

// renderMarkdownPages materializes every page in memory, keyed by its
// directory-relative filename (e.g. "index.md", "config-set-time.md").
func renderMarkdownPages(a *clihelp.App) (map[string]string, error) {
	pages := map[string]string{
		"index.md": renderIndex(a),
		"nav.md":   renderNav(a),
	}
	for _, n := range collectNodes(a) {
		file := markdownRelFile(n.path)
		if _, dup := pages[file]; dup {
			return nil, fmt.Errorf("markdown page collision: %q (%s) duplicates an existing page; rename conflicting commands", file, strings.Join(n.path, " "))
		}
		pages[file] = renderCommandPage(a, n)
	}
	return pages, nil
}

// markdownHash returns a deterministic content hash over every page. The app
// version participates implicitly through the index page content, so bumping
// the version invalidates the cache even when no command changed.
func markdownHash(pages map[string]string) string {
	return markdownHashOf(pages, markdownFormatVersion)
}

// markdownHashOf takes the version as an argument so that its contribution can
// be asserted: with the constant inlined, the only way to check that raising it
// regenerates anything was to raise it.
func markdownHashOf(pages map[string]string, version int) string {
	keys := make([]string, 0, len(pages))
	for k := range pages {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	h := sha256.New()
	fmt.Fprintf(h, "clihelp-md-v%d\n", version)
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
		"|", "\\|", // Escape pipe for table cells
	).Replace(strings.TrimSpace(s))
}

// mdCode wraps s as inline code. A backslash is not an escape inside a code
// span, so a backtick cannot be escaped: the span is fenced with a longer run of
// backticks than any inside it, as CommonMark prescribes, and padded with spaces
// when the content itself starts or ends with one. A newline cannot appear in an
// inline span at all, so it becomes a space; a caller that wants a block writes
// one itself.
func mdCode(s string) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", " "), "\n", " ")
	fence := "`"
	for strings.Contains(s, fence) {
		fence += "`"
	}
	pad := ""
	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		pad = " "
	}
	return fence + pad + s + pad + fence
}

// mdTableCode renders s as inline code inside a table cell. GitHub splits a row
// on "|" before it parses the cells, so a pipe must be escaped even inside a
// code span.
func mdTableCode(s string) string { return mdTableCell(mdCode(s)) }

// mdTableCell escapes text for use inside a markdown table cell.
func mdTableCell(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}

// pageMeta carries Jekyll front-matter fields for a generated page.
type pageMeta struct {
	title       string
	hasChildren bool
	parent      string
}

// yamlQuote returns s wrapped in YAML single quotes, doubling any embedded
// single quote so the value cannot break out of the front-matter field.
func yamlQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// pageHeader renders Jekyll front matter from meta.
func pageHeader(m pageMeta) string {
	var b strings.Builder
	fmt.Fprintf(&b, "---\ntitle: %s\n", yamlQuote(m.title))
	if m.hasChildren {
		fmt.Fprintf(&b, "has_children: true\n")
	}
	if m.parent != "" {
		fmt.Fprintf(&b, "parent: %s\n", yamlQuote(m.parent))
	}
	b.WriteString("---\n\n")
	return b.String()
}

// renderNav renders the page showing the full command tree as a nested list.
func renderNav(a *clihelp.App) string {
	var b strings.Builder
	b.WriteString(pageHeader(pageMeta{title: a.Name + " — Navigation"}))
	fmt.Fprintf(&b, "# %s — Navigation\n\n", mdInline(a.Name))
	if len(a.Commands) > 0 {
		b.WriteString("## Commands\n\n")
		for _, c := range a.Commands {
			if c.Hidden {
				continue
			}
			renderNavNode(&b, c, []string{c.Name}, 0)
		}
		b.WriteString("\n")
	}
	if len(a.Shortcuts) > 0 {
		b.WriteString("## Shortcut Commands\n\n")
		for _, s := range a.Shortcuts {
			if s.Hidden {
				continue
			}
			renderNavNode(&b, s, []string{s.Name}, 0)
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

func renderNavNode(b *strings.Builder, c clihelp.Command, path []string, depth int) {
	indent := strings.Repeat("  ", depth)
	file := markdownRelFile(path)
	fmt.Fprintf(b, "%s- [%s](%s)", indent, mdInline(displayName(c)), file)
	if c.Description != "" {
		fmt.Fprintf(b, " — %s", c.Description)
	}
	b.WriteString("\n")
	for _, sub := range c.Subcommands {
		if sub.Hidden {
			continue
		}
		subPath := append(append([]string{}, path...), sub.Name)
		renderNavNode(b, sub, subPath, depth+1)
	}
}

func renderIndexCommandTable(b *strings.Builder, heading string, cmds []clihelp.Command) {
	var visible []clihelp.Command
	for _, c := range cmds {
		if !c.Hidden {
			visible = append(visible, c)
		}
	}
	if len(visible) == 0 {
		return
	}
	fmt.Fprintf(b, "## %s\n\n| Command | Description |\n|---------|-------------|\n", heading)
	for _, c := range visible {
		desc := c.Description
		if desc == "" {
			desc = "—"
		}
		fmt.Fprintf(b, "| [%s](%s) | %s |\n", mdInline(displayName(c)), markdownSlug(c.Name)+".md", mdTableCell(desc))
	}
	b.WriteString("\n")
}

func renderIndexGlobalFlags(b *strings.Builder, a *clihelp.App) {
	var globalFlags []clihelp.Option
	for _, f := range a.PersistentOptions {
		if !f.Hidden {
			globalFlags = append(globalFlags, f)
		}
	}
	for _, f := range a.GlobalFlags {
		if !f.Hidden {
			globalFlags = append(globalFlags, f)
		}
	}
	if len(globalFlags) == 0 {
		return
	}
	b.WriteString("## Global Flags\n\n| Flag | Description |\n|------|-------------|\n")
	for _, f := range globalFlags {
		desc := f.Description
		if f.DefaultText != "" {
			desc = desc + " (default: " + f.DefaultText + ")"
		}
		fmt.Fprintf(b, "| %s | %s |\n", mdTableCode(f.Flags), mdTableCell(desc))
	}
	b.WriteString("\n")
}

func renderMarkdownExamples(b *strings.Builder, examples []clihelp.Example) {
	if len(examples) == 0 {
		return
	}
	b.WriteString("## Examples\n\n")
	for _, ex := range examples {
		line := fmt.Sprintf("- %s", mdCode(ex.Line))
		if ex.Description != "" {
			line += " — " + ex.Description
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")
}

// renderIndex renders the top-level application overview page.
func renderIndex(a *clihelp.App) string {
	var b strings.Builder
	b.WriteString(pageHeader(pageMeta{title: a.Name, hasChildren: true}))
	fmt.Fprintf(&b, "# %s\n\n", mdInline(a.Name))
	if a.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", a.Description)
	}

	renderIndexCommandTable(&b, "Commands", a.Commands)
	renderIndexCommandTable(&b, "Shortcut Commands", a.Shortcuts)
	renderIndexGlobalFlags(&b, a)
	renderMarkdownExamples(&b, a.Examples)

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

// firstWord is the command name at the head of a subcommand entry, without the
// arguments or the alias list that may follow it.
func firstWord(s string) string {
	if i := strings.IndexAny(s, " \t("); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

func findSubcommandFile(cmd *clihelp.Command, path []string, name string) string {
	for i := range cmd.Subcommands {
		if cmd.Subcommands[i].Name == name {
			p := append(append([]string{}, path...), cmd.Subcommands[i].Name)
			return markdownRelFile(p)
		}
		for _, alias := range cmd.Subcommands[i].Aliases {
			if alias == name {
				p := append(append([]string{}, path...), cmd.Subcommands[i].Name)
				return markdownRelFile(p)
			}
		}
	}
	return ""
}

func renderCommandSubcommandsTable(b *strings.Builder, cmd *clihelp.Command, path []string) {
	subs := subcommandEntries(cmd)
	if len(subs) == 0 {
		return
	}
	b.WriteString("## Subcommands\n\n| Command | Description |\n|---------|-------------|\n")
	for _, s := range subs {
		// An explicit entry is written as it is typed — "add <email>" — so the
		// page it links to is found by the command word alone.
		file := findSubcommandFile(cmd, path, firstWord(s.Name))
		desc := s.Description
		if desc == "" {
			desc = "—"
		}
		if file != "" {
			fmt.Fprintf(b, "| [%s](%s) | %s |\n", mdInline(s.Name), file, mdTableCell(desc))
		} else {
			fmt.Fprintf(b, "| %s | %s |\n", mdInline(s.Name), mdTableCell(desc))
		}
	}
	b.WriteString("\n")
}

func renderCommandParametersTable(b *strings.Builder, params []clihelp.Param) {
	if len(params) == 0 {
		return
	}
	b.WriteString("## Parameters\n\n| Parameter | Description |\n|-----------|-------------|\n")
	for _, p := range params {
		fmt.Fprintf(b, "| %s | %s |\n", mdTableCode(p.Name), mdTableCell(p.Description))
	}
	b.WriteString("\n")
}

func renderCommandFlagsTable(b *strings.Builder, options []clihelp.Option) {
	if len(options) == 0 {
		return
	}
	b.WriteString("## Flags\n\n| Flag | Description |\n|------|-------------|\n")
	for _, f := range options {
		desc := f.Description
		if f.DefaultText != "" {
			desc = desc + " (default: " + f.DefaultText + ")"
		}
		fmt.Fprintf(b, "| %s | %s |\n", mdTableCode(f.Flags), mdTableCell(desc))
	}
	b.WriteString("\n")
}

// renderCommandPage renders the detailed page for a single command.
func renderCommandPage(a *clihelp.App, n cmdNode) string {
	cmd := n.cmd
	fullTitle := a.Name + " " + strings.Join(n.path, " ")

	meta := pageMeta{title: fullTitle}
	if len(cmd.Subcommands) > 0 {
		meta.hasChildren = true
	}
	if len(n.path) > 1 {
		meta.parent = a.Name + " " + strings.Join(n.path[:len(n.path)-1], " ")
	}

	var b strings.Builder
	b.WriteString(pageHeader(meta))
	fmt.Fprintf(&b, "# %s\n\n", mdInline(fullTitle))
	desc := cmd.LongDescription
	if desc == "" {
		desc = cmd.Description
	}
	if desc != "" {
		fmt.Fprintf(&b, "%s\n\n", desc)
	}

	if cmd.UsageLine != "" {
		b.WriteString("## Usage\n\n```\n")
		b.WriteString(cmd.UsageLine)
		b.WriteString("\n```\n\n")
	}

	renderCommandSubcommandsTable(&b, &cmd, n.path)
	renderCommandParametersTable(&b, cmd.Parameters)
	renderCommandFlagsTable(&b, a.CollectOptions(n.path, &cmd))
	renderMarkdownExamples(&b, cmd.Examples)

	for _, note := range cmd.Notes {
		if note.Heading != "" {
			fmt.Fprintf(&b, "## %s\n\n", mdInline(note.Heading))
		}
		if note.Raw {
			trimmed := strings.Trim(note.Text, "\r\n")
			if !strings.HasPrefix(strings.TrimSpace(trimmed), "```") {
				fmt.Fprintf(&b, "```\n%s\n```\n\n", trimmed)
			} else {
				fmt.Fprintf(&b, "%s\n\n", trimmed)
			}
		} else {
			fmt.Fprintf(&b, "%s\n\n", note.Text)
		}
	}

	fmt.Fprintf(&b, "---\n\n[↑ %s](index.md) — [nav](nav.md)\n", mdInline(a.Name))

	return strings.TrimRight(b.String(), "\n") + "\n"
}

// markdownState is the sidecar's contents: the hash of the last generated pages
// and the names of those pages. The names are what makes pruning safe — only a
// file this generator wrote is a candidate for deletion.
type markdownState struct {
	hash  string
	pages []string
}

func readMarkdownState(path string) (markdownState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return markdownState{}, err
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	st := markdownState{hash: lines[0]}
	for _, name := range lines[1:] {
		if name != "" {
			st.pages = append(st.pages, name)
		}
	}
	return st, nil
}

func writeMarkdownState(path string, st markdownState) error {
	var b strings.Builder
	b.WriteString(st.hash)
	for _, name := range st.pages {
		b.WriteString("\n")
		b.WriteString(name)
	}
	b.WriteString("\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func pageNames(pages map[string]string) []string {
	names := make([]string, 0, len(pages))
	for name := range pages {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// pagesPresent reports whether every page the stored hash stands for is still on
// disk.
func pagesPresent(dir string, pages map[string]string) bool {
	for name := range pages {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return false
		}
	}
	return true
}

// writeMarkdownPages writes every page under dir and prunes the pages of an
// earlier pass that no longer correspond to a command.
//
// Only a page listed in previous is removed. Deleting every .md the pass did not
// write meant that pointing Dir at a directory that already held documentation
// destroyed it on the first run.
func writeMarkdownPages(dir string, pages map[string]string, previous []string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for name, content := range pages {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}

	for _, name := range previous {
		if _, ok := pages[name]; ok {
			continue
		}
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}

// gitLookupTimeout bounds the one external command this package runs.
const gitLookupTimeout = 3 * time.Second

// ensureHashIgnored keeps the hash sidecar out of git so it is never committed
// or pushed. It asks git rather than parsing .gitignore itself, because only git
// knows its own ignore semantics (patterns, negation, nesting, global excludes).
// Outside a git work tree it silently does nothing.
func ensureHashIgnored(dir, hashPath string) {
	// git is another program, and this one runs inside the caller's. A repository
	// on an unreachable mount, or a git that stops for credentials, would
	// otherwise hang documentation generation with nothing on screen. The
	// deadline kills git; WaitDelay is what bounds the wait when git has forked
	// and its child still holds the pipe — the same pair the pager and the manual
	// page lookup need, and this was the one place without either.
	ctx, cancel := context.WithTimeout(context.Background(), gitLookupTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "check-ignore", "-q", hashPath)
	cmd.WaitDelay = time.Second
	runErr := cmd.Run()
	if runErr == nil {
		return // already ignored
	}
	var ee *exec.ExitError
	if !errors.As(runErr, &ee) || ee.ExitCode() != 1 {
		return // git unavailable / not a repo
	}

	gi := filepath.Join(dir, ".gitignore")
	if data, err := os.ReadFile(gi); err == nil && hasIgnoreRule(string(data), filepath.Base(hashPath)) {
		return
	}
	f, err := os.OpenFile(gi, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "\n%s\n", filepath.Base(hashPath))
}

// hasIgnoreRule reports whether .gitignore already carries name as a rule of its
// own. A substring test matched "!.clihelp-hash", a rule that un-ignores the
// file, and then declined to add the rule that would ignore it.
func hasIgnoreRule(contents, name string) bool {
	for _, line := range strings.Split(contents, "\n") {
		if strings.TrimSpace(line) == name {
			return true
		}
	}
	return false
}

func displayName(c clihelp.Command) string {
	return c.Name
}

// subcommandEntries is clihelp's own list, not a second opinion about it.
//
// A copy used to live here, and it had lost the branch that prefers an explicit
// SubcommandEntries — the field whose purpose is documenting subcommands the
// tree does not carry — so the generated pages disagreed with `--help`.
func subcommandEntries(cmd *clihelp.Command) []clihelp.Param {
	return clihelp.SubcommandList(*cmd)
}
