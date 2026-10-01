package clihelp

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// commitFixture is a throwaway repository holding tools/commit.sh, the real
// .gitignore, and a stub quality gate that only records that it ran — so the
// script's decisions can be observed without a 30-second gate or a real commit
// to the project.
func commitFixture(t *testing.T) string {
	t.Helper()
	for _, tool := range []string{"git", "bash"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not installed", tool)
		}
	}
	root := t.TempDir()
	script, err := os.ReadFile(filepath.Join("tools", "commit.sh"))
	if err != nil {
		t.Fatal(err)
	}
	ignore, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"tools/commit.sh": string(script),
		"tools/check.sh":  "#!/bin/sh\ntouch .gate-ran\necho check >> .order\n",
		"tools/fix.sh":    "#!/bin/sh\necho fix >> .order\n",
		".gitignore":      string(ignore) + ".gate-ran\n.order\n",
		"tracked.txt":     "one\n",
	}
	for name, body := range files {
		full := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@example.com"},
		{"config", "user.name", "t"},
		{"config", "core.excludesFile", "/dev/null"},
		{"add", "."},
		{"commit", "-q", "-m", "init"},
	} {
		git(t, root, args...)
	}
	return root
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// runCommit runs the script and reports its exit status, stdout and stderr.
func runCommit(root string, env []string, msg ...string) (failed bool, stdout, stderr string) {
	cmd := exec.Command("bash", append([]string{"tools/commit.sh"}, msg...)...)
	cmd.Dir = root
	cmd.Env = append(withoutMakeState(os.Environ()), env...)
	var o, e bytes.Buffer
	cmd.Stdout, cmd.Stderr = &o, &e
	err := cmd.Run()
	return err != nil, o.String(), e.String()
}

func gateRan(root string) bool {
	_, err := os.Stat(filepath.Join(root, ".gate-ran"))
	return err == nil
}

func write(t *testing.T, root, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A message that cannot be committed used to be discovered after the whole gate
// had run: ARGS= (one empty argument) passes the "no arguments" check, takes
// thirty seconds, stages everything, and only then fails inside git. And
// AGENTS.md requires conventional commits, which nothing enforced.
func TestCommitRefusesABadMessageBeforeTheGate(t *testing.T) {
	for _, tt := range []struct{ name, msg string }{
		{"empty", ""},
		{"only spaces", "   "},
		{"not conventional", "update some things"},
		{"unknown type", "wibble: something"},
		{"no description", "feat:"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := commitFixture(t)
			write(t, root, "tracked.txt", "two\n")
			failed, _, stderr := runCommit(root, nil, tt.msg)
			if !failed {
				t.Error("commit.sh accepted the message")
			}
			if gateRan(root) {
				t.Error("the gate ran before the message was checked")
			}
			if staged := git(t, root, "diff", "--cached", "--name-only"); staged != "" {
				t.Errorf("a refused message left files staged:\n%s", staged)
			}
			if !strings.Contains(stderr, "commit.sh") {
				t.Errorf("the refusal is not attributed to commit.sh:\n%s", stderr)
			}
		})
	}
}

func TestCommitAcceptsConventionalMessages(t *testing.T) {
	for _, msg := range []string{
		"feat: add a thing", "fix(render): wrap early", "docs: x", "test: y", "chore: z",
		"refactor: w", "feat!: break it", "fix: has a body\n\nsecond paragraph",
	} {
		t.Run(strings.SplitN(msg, "\n", 2)[0], func(t *testing.T) {
			root := commitFixture(t)
			write(t, root, "tracked.txt", "two\n")
			failed, stdout, stderr := runCommit(root, nil, msg)
			if failed {
				t.Fatalf("commit.sh refused %q:\n%s", msg, stderr)
			}
			if !strings.HasPrefix(stdout, "Success ") {
				t.Errorf("no success line: %q", stdout)
			}
			if subject := strings.TrimSpace(git(t, root, "log", "-1", "--format=%s")); subject != strings.SplitN(msg, "\n", 2)[0] {
				t.Errorf("committed subject %q", subject)
			}
		})
	}
}

// `git add -A` committed whatever happened to be lying in the tree — a scratch
// file, a key, a .env — with no word of it, in a script whose whole output on
// success is one line. New files now have to be named: staged first, or
// admitted with COMMIT_ADD_UNTRACKED=1.
func TestCommitDoesNotSweepUpUntrackedFiles(t *testing.T) {
	root := commitFixture(t)
	write(t, root, "tracked.txt", "two\n")
	write(t, root, "scratch.txt", "notes\n")

	failed, _, stderr := runCommit(root, nil, "fix: change tracked")
	if !failed {
		t.Fatal("commit.sh went ahead with an untracked file present")
	}
	if !strings.Contains(stderr, "scratch.txt") || !strings.Contains(stderr, "COMMIT_ADD_UNTRACKED") {
		t.Errorf("the refusal must name the file and the way forward:\n%s", stderr)
	}
	if gateRan(root) {
		t.Error("the gate ran although the commit was already going to be refused")
	}
	if tracked := git(t, root, "ls-files"); strings.Contains(tracked, "scratch.txt") {
		t.Errorf("the untracked file was added:\n%s", tracked)
	}
	if log := git(t, root, "log", "--oneline"); strings.Count(log, "\n") != 1 {
		t.Errorf("a commit was made:\n%s", log)
	}
}

func TestCommitAdmitsNewFilesOnlyWhenAsked(t *testing.T) {
	t.Run("opt-in", func(t *testing.T) {
		root := commitFixture(t)
		write(t, root, "new.txt", "new\n")
		failed, _, stderr := runCommit(root, []string{"COMMIT_ADD_UNTRACKED=1"}, "feat: add new.txt")
		if failed {
			t.Fatalf("commit.sh refused the opt-in:\n%s", stderr)
		}
		if shown := git(t, root, "show", "--stat", "--format=", "HEAD"); !strings.Contains(shown, "new.txt") {
			t.Errorf("new.txt is not in the commit:\n%s", shown)
		}
	})
	t.Run("staged by the author", func(t *testing.T) {
		root := commitFixture(t)
		write(t, root, "new.txt", "new\n")
		git(t, root, "add", "new.txt")
		failed, _, stderr := runCommit(root, nil, "feat: add new.txt")
		if failed {
			t.Fatalf("a file the author staged was refused:\n%s", stderr)
		}
		if shown := git(t, root, "show", "--stat", "--format=", "HEAD"); !strings.Contains(shown, "new.txt") {
			t.Errorf("new.txt is not in the commit:\n%s", shown)
		}
	})
}

func TestCommitStagesChangesAndDeletionsToTrackedFiles(t *testing.T) {
	root := commitFixture(t)
	write(t, root, "second.txt", "s\n")
	git(t, root, "add", "second.txt")
	git(t, root, "commit", "-q", "-m", "chore: second")
	write(t, root, "tracked.txt", "changed\n")
	if err := os.Remove(filepath.Join(root, "second.txt")); err != nil {
		t.Fatal(err)
	}
	if failed, _, stderr := runCommit(root, nil, "fix: edit and delete"); failed {
		t.Fatalf("commit.sh failed:\n%s", stderr)
	}
	shown := git(t, root, "show", "--stat", "--format=", "HEAD")
	if !strings.Contains(shown, "tracked.txt") || !strings.Contains(shown, "second.txt") {
		t.Errorf("an edit and a deletion should both be committed:\n%s", shown)
	}
	if status := git(t, root, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("the tree is not clean afterwards:\n%s", status)
	}
}

// Files that look like credentials must not be one `COMMIT_ADD_UNTRACKED=1` away
// from a public repository.
func TestGitignoreKeepsCredentialShapedFilesOut(t *testing.T) {
	root := commitFixture(t)
	for _, name := range []string{".env", ".env.local", "server.pem", "deploy.key", "id_rsa", "id_rsa.pub", "merge.orig"} {
		write(t, root, name, "secret\n")
	}
	write(t, root, "tracked.txt", "two\n")
	failed, _, stderr := runCommit(root, []string{"COMMIT_ADD_UNTRACKED=1"}, "fix: change tracked")
	if failed {
		t.Fatalf("commit.sh failed (a credential-shaped file was probably listed as untracked):\n%s", stderr)
	}
	files := git(t, root, "ls-files")
	for _, name := range []string{".env", ".env.local", "server.pem", "deploy.key", "id_rsa", "id_rsa.pub", "merge.orig"} {
		if strings.Contains(files, name+"\n") {
			t.Errorf("%s was committed", name)
		}
	}
}

// commit.sh used to rely on check.sh to reformat the tree as a side effect.
// The check is read-only now, so the script repairs first (tools/fix.sh) and only
// then checks; otherwise a mis-formatted edit would fail the commit it should
// have quietly cleaned up.
func TestCommitRepairsBeforeItChecks(t *testing.T) {
	root := commitFixture(t)
	write(t, root, "tracked.txt", "two\n")
	if failed, _, stderr := runCommit(root, nil, "fix: change tracked"); failed {
		t.Fatalf("commit.sh failed:\n%s", stderr)
	}
	order, err := os.ReadFile(filepath.Join(root, ".order"))
	if err != nil {
		t.Fatal(err)
	}
	if string(order) != "fix\ncheck\n" {
		t.Errorf("order of the repair and the check = %q, want fix then check", order)
	}
}
