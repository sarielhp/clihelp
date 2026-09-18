package clihelp

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeFileAtomically writes data to a temporary file beside path and renames it
// into place.
//
// The contract, stated so it can be relied on: the replacement is atomic, so a
// reader sees either the old file or the new one; a symlink is followed rather
// than replaced; the file's permission bits are preserved by the caller passing
// them in; the data and the directory entry are flushed before it returns; and a
// clean error return leaves nothing behind. Ownership, ACLs and hard links are
// *not* preserved — a rename installs a new inode — and an interrupt between the
// write and the rename can leave one .tmp-* sibling.
func writeFileAtomically(path string, data []byte, mode os.FileMode) error {
	if refuseForeignOwner(path) {
		return fmt.Errorf("refusing to write %q as root: it belongs to another user", path)
	}

	// Write through a symlink rather than over it. rename(2) replaces the link
	// itself, and ~/.zshrc is routinely a link into a dotfiles repo: replacing it
	// detaches the repo without a word, and git status stays clean. Resolving
	// also keeps the temp file beside the real file, so the rename stays within
	// one filesystem.
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}

	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp) // no-op once the rename has succeeded

	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	// Durability, not just visibility: the rename can reach disk before the data
	// blocks do, and the file at the other end of this is the user's shell
	// startup file. A machine that loses power mid-install should not bring back
	// an empty ~/.bashrc.
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp, mode); err != nil {
		return err
	}
	preserveOwner(path, tmp)
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	if dir, err := os.Open(filepath.Dir(path)); err == nil {
		_ = dir.Sync()
		dir.Close()
	}
	return nil
}
