package clihelp

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// checkFixture is a throwaway module holding tools/check.sh and tools/fix.sh and
// just enough else for the gate to pass: a VERSION, the two files the version
// drift check reads, and a stub staticcheck. HOME points into the fixture so
// nothing outside it is read.
func checkFixture(t *testing.T) (root string, env []string) {
	t.Helper()
	for _, tool := range []string{"go", "gofmt", "bash"} {
		if _, err := exec.LookPath(tool); err != nil {
			skipMissingTool(t, tool+" not found")
		}
	}
	root = t.TempDir()
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join("tools", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	files := map[string]string{
		"tools/check.sh":       read("check.sh"),
		"tools/fix.sh":         read("fix.sh"),
		"tools/audit.sh":       read("audit.sh"),
		"tools/audit_lines.rb": read("audit_lines.rb"),
		"go.mod":               "module example.com/fixture\n\ngo 1.22\n",
		"VERSION":              "0.0.1\n",
		"clihelp.go":           "package fixture\n\nconst Version = \"0.0.1\"\n",
		"example/main.go":      "package main\n\nvar app = struct {\n\tName    string\n\tVersion string\n}{\n\tName:    \"x\",\n\tVersion: \"0.0.1\",\n}\n\nfunc main() { _ = app }\n",
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
	stubs := filepath.Join(root, ".stubs")
	if err := os.MkdirAll(stubs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stubs, "staticcheck"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cache, _ := exec.Command("go", "env", "GOCACHE").Output()
	env = append(withoutMakeState(os.Environ()),
		"HOME="+filepath.Join(root, ".home"),
		"GOCACHE="+strings.TrimSpace(string(cache)),
		"GOFLAGS=-mod=mod",
		"PATH="+stubs+string(os.PathListSeparator)+os.Getenv("PATH"))
	return root, env
}

// snapshot hashes every file under root except the stub and home directories, so
// a test can say "the check changed nothing".
func snapshot(t *testing.T, root string) string {
	t.Helper()
	var names []string
	sums := map[string][32]byte{}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if strings.HasPrefix(rel, ".stubs") || strings.HasPrefix(rel, ".home") {
			return nil
		}
		b, _ := os.ReadFile(path)
		names = append(names, rel)
		sums[rel] = sha256.Sum256(b)
		return nil
	})
	sort.Strings(names)
	var out strings.Builder
	for _, n := range names {
		fmt.Fprintf(&out, "%s %x\n", n, sums[n])
	}
	return out.String()
}

func runScript(root string, env []string, script string) (failed bool, output string) {
	cmd := exec.Command("bash", script)
	cmd.Dir = root
	cmd.Env = env
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	err := cmd.Run()
	return err != nil, buf.String()
}

// A "check" that rewrites the tree is a fix wearing a check's name: it
// reformatted and re-tidied files behind the caller's back, a release left those
// rewrites uncommitted after the tag, and a gate run to learn the state of the
// tree changed it. The check reports; fix.sh repairs.
func TestCheckReportsButNeverRewrites(t *testing.T) {
	root, env := checkFixture(t)
	if failed, out := runScript(root, env, "tools/check.sh"); failed {
		t.Fatalf("check.sh failed on a clean fixture:\n%s", out)
	}

	bad := filepath.Join(root, "bad.go")
	if err := os.WriteFile(bad, []byte("package fixture\n\nfunc   Add(a,b int)int{return a+b}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, root)
	failed, out := runScript(root, env, "tools/check.sh")
	if !failed {
		t.Fatal("check.sh passed with a mis-formatted file")
	}
	if !strings.Contains(out, "bad.go") || !strings.Contains(out, "fix") {
		t.Errorf("the failure should name the file and the remedy:\n%s", out)
	}
	if after := snapshot(t, root); after != before {
		t.Errorf("check.sh changed the tree:\n--- before\n%s--- after\n%s", before, after)
	}

	// And the remedy works: fix.sh formats it, after which the check passes.
	if failed, out := runScript(root, env, "tools/fix.sh"); failed {
		t.Fatalf("fix.sh failed:\n%s", out)
	}
	if listed, _ := exec.Command("gofmt", "-s", "-l", root).Output(); len(bytes.TrimSpace(listed)) != 0 {
		t.Errorf("fix.sh left files unformatted:\n%s", listed)
	}
	if failed, out := runScript(root, env, "tools/check.sh"); failed {
		t.Errorf("check.sh still fails after fix.sh:\n%s", out)
	}
}

// The size and nesting limits in AGENTS.md used to be enforced by a script that
// measured only length (and could be fooled by a brace in a string). go-audit,
// the standards tool, measures length, nesting depth and branch count; this runs
// it so that `go test` — and so every gate — holds the repository to them.
func TestGuidelinesAreEnforced(t *testing.T) {
	tool, err := exec.LookPath("go-audit")
	if err != nil {
		const home = "/home/sariel/prog/standards/go/bin/go-audit"
		if _, statErr := os.Stat(home); statErr != nil {
			t.Skip("go-audit is not installed; the size and nesting guidelines are not enforced on this machine")
		}
		tool = home
	}
	t.Run("the repository conforms", func(t *testing.T) {
		out, err := exec.Command(tool, "--strict", "-q", ".").CombinedOutput()
		if err != nil {
			t.Errorf("go-audit reports violations:\n%s", out)
		}
	})
	t.Run("it can fail", func(t *testing.T) {
		dir := t.TempDir()
		deep := "package p\n\nfunc F(a, b, c, d, e, f bool) int {\n" +
			"\tif a {\n\t\tif b {\n\t\t\tif c {\n\t\t\t\tif d {\n\t\t\t\t\tif e {\n\t\t\t\t\t\tif f {\n\t\t\t\t\t\t\treturn 1\n" +
			"\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t}\n\treturn 0\n}\n"
		if err := os.WriteFile(filepath.Join(dir, "deep.go"), []byte(deep), 0o644); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(tool, "--strict", "-q", dir)
		if out, err := cmd.CombinedOutput(); err == nil {
			t.Errorf("go-audit accepted a function nested six deep:\n%s", out)
		}
	})
}
