package clihelp

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

const defaultChangelog = "# Changelog\n\n## [Unreleased]\n\n### Added\n- a thing\n\n## [0.0.1] - 2026-01-01\n\n### Added\n- an older thing\n"

// bumpFixture is a throwaway repository holding tools/bump-version.sh and the
// three files it rewrites, with a stub `go` first on PATH so the script can
// never build, push or tag anything real. It returns the repository root and
// the PATH to run the script with.
func bumpFixture(t *testing.T, goStub, changelog string) (root, path string) {
	t.Helper()
	for _, tool := range []string{"git", "bash", "ruby"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not installed", tool)
		}
	}
	root = t.TempDir()
	script, err := os.ReadFile(filepath.Join("tools", "bump-version.sh"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"tools/bump-version.sh":       string(script),
		"VERSION":                     "0.0.1\n",
		"example/main.go":             "package main\n\nvar app = App{\n\tName:                  \"x\",\n\tVersion:               \"0.0.1\",\n}\n",
		"clihelp.go":                  "package clihelp\n\nconst Version = \"0.0.1\"\n",
		"docs/clihelp/index.md":       "docs\n",
		"docs/mail_cli_fake/index.md": "docs\n",
		"CHANGES.md":                  changelog,
		"tools/check.sh":              "#!/bin/sh\nexit 0\n",
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
	remote := filepath.Join(t.TempDir(), "remote.git")
	bin := filepath.Join(root, ".stubs")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "go"), []byte("#!/bin/sh\n"+goStub+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@example.com"},
		{"config", "user.name", "t"},
		{"config", "core.excludesFile", "/dev/null"},
		{"add", "tools", "VERSION", "example", "clihelp.go", "docs", "CHANGES.md"},
		{"commit", "-q", "-m", "init"},
		{"branch", "-M", "master"},
		{"init", "-q", "--bare", remote},
		{"remote", "add", "origin", remote},
		{"push", "-q", "-u", "origin", "master"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return root, bin + string(os.PathListSeparator) + os.Getenv("PATH")
}

func bumpCommand(root, path string) *exec.Cmd {
	cmd := exec.Command("bash", "tools/bump-version.sh")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PATH="+path, "HOME="+root, "GIT_CONFIG_NOSYSTEM=1")
	return cmd
}

func fileHas(t *testing.T, root, name, want string) bool {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Contains(string(body), want)
}

// A failed bump used to run `git checkout` on clihelp.go and example/main.go
// whatever state they were in, so an uncommitted edit to either was deleted; a
// successful one committed it into the tagged release. Either way the author
// lost control of what shipped. The script has to refuse to start instead.
func TestBumpRefusesATreeWithUncommittedChanges(t *testing.T) {
	root, path := bumpFixture(t, "exit 1", defaultChangelog)
	edit := "// precious uncommitted work\n"
	full := filepath.Join(root, "clihelp.go")
	body, _ := os.ReadFile(full)
	if err := os.WriteFile(full, append(body, edit...), 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	cmd := bumpCommand(root, path)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Errorf("bump succeeded on a tree with uncommitted changes")
	}
	if !fileHas(t, root, "clihelp.go", "precious uncommitted work") {
		t.Errorf("bump deleted an uncommitted edit; stderr:\n%s", stderr.String())
	}
	if !fileHas(t, root, "VERSION", "0.0.1") {
		t.Errorf("bump rewrote VERSION on a refused run")
	}
	if !strings.Contains(stderr.String(), "uncommitted") {
		t.Errorf("the refusal does not say why:\n%s", stderr.String())
	}
}

// Interrupting a bump used to leave VERSION and the files that carry it
// rewritten — only the ERR trap restored them — so the next run bumped again
// from there and skipped a version. The script's own comment promises the
// opposite.
func TestBumpRestoresTheVersionFilesWhenInterrupted(t *testing.T) {
	root, path := bumpFixture(t, "sleep 30", defaultChangelog)
	cmd := bumpCommand(root, path)
	// Its own process group, so the signal reaches the stub as a terminal's
	// Ctrl-C would reach a foreground job.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	// Wait for the rewrite to land, then interrupt.
	deadline := time.Now().Add(10 * time.Second)
	for !fileHas(t, root, "VERSION", "0.0.2") {
		if time.Now().After(deadline) {
			t.Fatal("the script never reached the version rewrite")
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		t.Fatal("the script did not exit after SIGTERM")
	}

	status := exec.Command("git", "status", "--porcelain", "--untracked-files=no")
	status.Dir = root
	out, err := status.Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(bytes.TrimSpace(out)) != 0 {
		t.Errorf("an interrupted bump left the tree modified:\n%s", out)
	}
}

// The changelog kept its own [Unreleased] heading through every release: tags
// v0.3.45 to v0.3.48 exist and CHANGES.md has no heading for any of them, because
// nothing in the release path touched it. A bump now promotes the heading — the
// new version and the day, with the old text beneath it and an empty
// [Unreleased] above — and the commit that lands carries the change.
func TestBumpPromotesTheChangelog(t *testing.T) {
	root, path := bumpFixture(t, "exit 0", defaultChangelog)
	var stderr bytes.Buffer
	cmd := bumpCommand(root, path)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("bump failed: %v\n%s", err, stderr.String())
	}
	body, err := os.ReadFile(filepath.Join(root, "CHANGES.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := "## [Unreleased]\n\n## [0.0.2] - " + time.Now().Format("2006-01-02") + "\n\n### Added\n- a thing\n"
	if !strings.Contains(string(body), want) {
		t.Errorf("CHANGES.md was not promoted; want it to contain\n%q\ngot\n%s", want, body)
	}
	if !strings.Contains(string(body), "## [0.0.1] - 2026-01-01") {
		t.Errorf("the older release heading was lost:\n%s", body)
	}
	show := exec.Command("git", "show", "--stat", "--format=", "HEAD")
	show.Dir = root
	out, _ := show.Output()
	if !strings.Contains(string(out), "CHANGES.md") {
		t.Errorf("the version commit does not include CHANGES.md:\n%s", out)
	}
}

// A release with nothing under [Unreleased] has no changelog entry to promote,
// and shipping it would put an empty section in the history. Refuse, and leave
// every file as it was.
func TestBumpRefusesAnEmptyChangelog(t *testing.T) {
	empty := "# Changelog\n\n## [Unreleased]\n\n## [0.0.1] - 2026-01-01\n\n### Added\n- an older thing\n"
	root, path := bumpFixture(t, "exit 0", empty)
	var stderr bytes.Buffer
	cmd := bumpCommand(root, path)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Error("bump succeeded with nothing under [Unreleased]")
	}
	if !fileHas(t, root, "VERSION", "0.0.1") {
		t.Error("VERSION was left rewritten after a refused bump")
	}
	body, _ := os.ReadFile(filepath.Join(root, "CHANGES.md"))
	if string(body) != empty {
		t.Errorf("CHANGES.md was modified by a refused bump:\n%s", body)
	}
	if !strings.Contains(stderr.String(), "Unreleased") {
		t.Errorf("the refusal does not mention [Unreleased]:\n%s", stderr.String())
	}
}

// The version literal in example/main.go sits among keys that gofmt aligns, and
// the bump rewrote it with a fixed run of spaces. The tree was no longer
// gofmt-clean, which the old check hid by reformatting it after the fact and
// leaving that rewrite uncommitted after the tag; the read-only check refuses it.
// The bump has to change the number and nothing else.
func TestBumpKeepsTheVersionLiteralAligned(t *testing.T) {
	root, path := bumpFixture(t, "exit 0", defaultChangelog)
	var stderr bytes.Buffer
	cmd := bumpCommand(root, path)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("bump failed: %v\n%s", err, stderr.String())
	}
	body, err := os.ReadFile(filepath.Join(root, "example", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	const want = "\tVersion:               \"0.0.2\",\n"
	if !strings.Contains(string(body), want) {
		t.Errorf("the version literal lost its alignment; want a line %q in\n%s", want, body)
	}
}
