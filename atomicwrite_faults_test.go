package clihelp

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The contract on writeFileAtomically was a comment, not a checked property:
// every syscall in it can fail, and none of those failures was reachable from a
// test while os was called directly. Each one now has to leave the original file
// exactly as it was, return the error rather than swallow it, and leave no
// .tmp-* sibling behind in the user's home directory.

var errInjected = errors.New("injected failure")

// failingFile fails at whichever step the test names.
type failingFile struct {
	tempFile
	failWrite, failSync, failClose bool
}

func (f *failingFile) Write(p []byte) (int, error) {
	if f.failWrite {
		return 0, errInjected
	}
	return f.tempFile.Write(p)
}

func (f *failingFile) Sync() error {
	if f.failSync {
		return errInjected
	}
	return f.tempFile.Sync()
}

func (f *failingFile) Close() error {
	if f.failClose {
		_ = f.tempFile.Close()
		return errInjected
	}
	return f.tempFile.Close()
}

func TestAtomicWriteFailurePaths(t *testing.T) {
	const original = "the file as the user left it\n"

	for _, tt := range []struct {
		name    string
		breakIt func(ops *fileOps)
	}{
		{"CreateTemp fails", func(ops *fileOps) {
			ops.createTemp = func(string, string) (tempFile, error) { return nil, errInjected }
		}},
		{"Write fails", func(ops *fileOps) {
			inner := ops.createTemp
			ops.createTemp = func(d, p string) (tempFile, error) {
				f, err := inner(d, p)
				return &failingFile{tempFile: f, failWrite: true}, err
			}
		}},
		{"Sync fails", func(ops *fileOps) {
			inner := ops.createTemp
			ops.createTemp = func(d, p string) (tempFile, error) {
				f, err := inner(d, p)
				return &failingFile{tempFile: f, failSync: true}, err
			}
		}},
		{"Close fails", func(ops *fileOps) {
			inner := ops.createTemp
			ops.createTemp = func(d, p string) (tempFile, error) {
				f, err := inner(d, p)
				return &failingFile{tempFile: f, failClose: true}, err
			}
		}},
		{"Chmod fails", func(ops *fileOps) {
			ops.chmod = func(string, os.FileMode) error { return errInjected }
		}},
		{"Rename fails", func(ops *fileOps) {
			ops.rename = func(string, string) error { return errInjected }
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".bashrc")
			if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
				t.Fatal(err)
			}

			ops := realFileOps()
			tt.breakIt(&ops)
			err := writeFileAtomicallyWith(ops, path, []byte("replacement\n"), 0o644)

			if !errors.Is(err, errInjected) {
				t.Fatalf("the failure was not reported: %v", err)
			}
			got, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("the original file is gone: %v", readErr)
			}
			if string(got) != original {
				t.Errorf("a failed write changed the file:\n%q", got)
			}
			assertNoTempLeftBehind(t, dir)
		})
	}
}

// The success path has the same obligation about temporary files, and it is
// worth checking that the seam did not quietly stop doing the real work.
func TestAtomicWriteSucceedsThroughTheSeam(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".bashrc")
	if err := os.WriteFile(path, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var syncedDir string
	ops := realFileOps()
	inner := ops.syncDir
	ops.syncDir = func(name string) { syncedDir = name; inner(name) }

	if err := writeFileAtomicallyWith(ops, path, []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "new\n" {
		t.Fatalf("the write did not land: %q (%v)", got, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode is %v, want 0600", info.Mode().Perm())
	}
	if syncedDir != dir {
		t.Errorf("the containing directory was not flushed: %q", syncedDir)
	}
	assertNoTempLeftBehind(t, dir)
}

func assertNoTempLeftBehind(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("a temporary file was left in the user's directory: %s", e.Name())
		}
	}
}
