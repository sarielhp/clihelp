package clihelp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Symlinking ~/.zshrc into a git repo is the usual dotfile layout — stow,
// chezmoi, yadm, or a hand-rolled ln -s. rename(2) replaces the link, not its
// target, so an install used to detach the repo silently: the file the user
// edits from then on is not the file their shell reads, and git status stays
// clean.
func TestWriteThroughASymlinkedDotfile(t *testing.T) {
	home := sandboxHome(t)
	repo := filepath.Join(home, "dotfiles")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(repo, "zshrc")
	if err := os.WriteFile(real, []byte("export EDITOR=vi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, ".zshrc")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}

	if _, err := installShellIntegration(installApp(), "zsh", true); err != nil {
		t.Fatal(err)
	}

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the symlink was replaced by a regular file, orphaning the dotfiles repo")
	}
	body, _ := os.ReadFile(real)
	if !strings.Contains(string(body), "shell integration") {
		t.Errorf("the block did not reach the file the symlink names:\n%s", body)
	}
	if !strings.Contains(string(body), "export EDITOR=vi") {
		t.Errorf("the user's own content was lost")
	}
}

func TestWriteFileAtomicallyKeepsMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f")
	if err := writeFileAtomically(path, []byte("a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
	// And nothing is left behind.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("a temporary file was left behind: %s", e.Name())
		}
	}
}

// An atomic replace installs a new inode, so anything carried across has to be
// carried deliberately. Mode was; ownership was not, and the auto path runs
// before command dispatch on every invocation, so `sudo -E myapp anything` was
// enough to take a user's own startup file away from them.
func TestWriteFileAtomicallyPreservesGroup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rc")
	if err := os.WriteFile(path, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	beforeUID, beforeGID := ownerOf(t, before)

	if err := writeFileAtomically(path, []byte("replaced\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	afterUID, afterGID := ownerOf(t, after)
	if beforeUID != afterUID || beforeGID != afterGID {
		t.Errorf("ownership changed across the replace: %d:%d -> %d:%d", beforeUID, beforeGID, afterUID, afterGID)
	}
}
