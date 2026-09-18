//go:build unix

package clihelp

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// lockFile has one job: only one holder inside the critical section at a time.
//
// It did not do it. flock is held on an inode, not on a name, and the release
// function unlinked the lock file — so a waiter already blocked on the old inode
// could acquire it at the moment a newcomer's O_CREATE produced a fresh inode
// and locked that one instead. Two writers, each certain it held the lock.
//
// The symptom was whole blocks disappearing from a startup file while every
// caller reported success, and it only appeared under load, which is why it
// showed up in a full -race run and never in the test on its own. The counter
// below makes it deterministic: a lost update is a lost increment.
func TestLockFileSerializesItsHolders(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, ".bashrc")
	counter := filepath.Join(dir, "counter")
	if err := os.WriteFile(counter, []byte("0"), 0o600); err != nil {
		t.Fatal(err)
	}

	const workers, rounds = 12, 40
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for r := 0; r < rounds; r++ {
				unlock, err := lockFile(target)
				if err != nil {
					t.Errorf("lockFile: %v", err)
					return
				}
				// A read-modify-write with a real gap in the middle, which is
				// what install does with the startup file.
				raw, readErr := os.ReadFile(counter)
				if readErr != nil {
					unlock()
					t.Errorf("read: %v", readErr)
					return
				}
				n, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
				if writeErr := os.WriteFile(counter, []byte(strconv.Itoa(n+1)), 0o600); writeErr != nil {
					unlock()
					t.Errorf("write: %v", writeErr)
					return
				}
				unlock()
			}
		}()
	}
	wg.Wait()

	raw, err := os.ReadFile(counter)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
	if want := workers * rounds; got != want {
		t.Errorf("%d of %d updates were lost: two holders were inside the lock at once", want-got, want)
	}
}
