package clihelp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every destination this library writes to is a name a user or a distribution
// legitimately uses. "Does not carry our marker" means "someone else wrote it",
// never "stale".
func TestNothingOverwritesAFileClihelpDidNotWrite(t *testing.T) {
	mine := "# hand written, not by clihelp\nalias x='y'\n"

	t.Run("auto-install leaves a foreign completion script alone", func(t *testing.T) {
		home := sandboxHome(t)
		autoEnv(t)
		script := filepath.Join(home, ".local", "share", "bash-completion", "completions", "myapp")
		writeFixture(t, script, mine)

		app := installApp()
		app.AutoInstallCompletion = true
		TestExecute(app, []string{"build"}).AssertNoError(t)

		if got, _ := os.ReadFile(script); string(got) != mine {
			t.Errorf("an ordinary program run overwrote a hand-written completion script:\n%s", got)
		}
	})

	t.Run("install refuses to overwrite a foreign fish drop-in", func(t *testing.T) {
		home := sandboxHome(t)
		drop := filepath.Join(home, ".config", "fish", "conf.d", "myapp.fish")
		writeFixture(t, drop, mine)

		_, err := InstallShellIntegration(installApp(), "fish", true)
		if err == nil {
			t.Errorf("install overwrote a file clihelp never wrote")
		}
		if got, _ := os.ReadFile(drop); string(got) != mine {
			t.Errorf("the user's file was modified:\n%s", got)
		}
	})

	t.Run("uninstall leaves a foreign fish drop-in alone", func(t *testing.T) {
		home := sandboxHome(t)
		drop := filepath.Join(home, ".config", "fish", "conf.d", "myapp.fish")
		writeFixture(t, drop, mine)

		if _, err := UninstallShellIntegration(installApp(), "fish"); err != nil {
			t.Fatal(err)
		}
		if got, _ := os.ReadFile(drop); string(got) != mine {
			t.Errorf("uninstall removed or changed a file clihelp never wrote")
		}
	})

	t.Run("install and uninstall still work on our own fish drop-in", func(t *testing.T) {
		home := sandboxHome(t)
		if _, err := InstallShellIntegration(installApp(), "fish", true); err != nil {
			t.Fatal(err)
		}
		drop := filepath.Join(home, ".config", "fish", "conf.d", "myapp.fish")
		if _, err := os.Stat(drop); err != nil {
			t.Fatalf("our own drop-in was not written: %v", err)
		}
		if _, err := InstallShellIntegration(installApp(), "fish", true); err != nil {
			t.Errorf("re-installing over our own drop-in was refused: %v", err)
		}
		if _, err := UninstallShellIntegration(installApp(), "fish"); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(drop); !os.IsNotExist(err) {
			t.Errorf("uninstall left our own drop-in behind")
		}
	})
}

// The markers delimit a block. A file that merely mentions the marker text — in
// a comment, in a quoted string — contains no block at all.
func TestBlockEditingMatchesWholeLinesOnly(t *testing.T) {
	begin, end := "# >>> B >>>", "# <<< E <<<"
	block := begin + "\nsource x\n" + end + "\n"

	for _, tt := range []struct {
		name string
		in   string
		keep []string
	}{
		{"mentions the marker in a comment",
			"# I removed the \"" + begin + "\" block\nexport EDITOR=vi\nalias ll='ls -l'\n",
			[]string{"export EDITOR=vi", "alias ll='ls -l'"}},
		{"marker inside a quoted string",
			"echo \"" + begin + "\"\nexport EDITOR=vi\n",
			[]string{"export EDITOR=vi"}},
		{"begin without end",
			begin + "\nhalf a block\nexport EDITOR=vi\n",
			[]string{"export EDITOR=vi"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := upsertBlock(tt.in, block, begin, end)
			if err != nil {
				return // refusing is an acceptable answer; destroying is not
			}
			for _, want := range tt.keep {
				if !strings.Contains(out, want) {
					t.Errorf("user line %q was deleted:\n%s", want, out)
				}
			}
		})
	}
}

// Installing and then uninstalling gives the file back byte for byte — the
// promise in docs/completion.md. The one normalisation is a missing trailing
// newline, which install adds and uninstall cannot know to remove; every
// well-formed startup file ends in one.
func TestBlockRoundTripsExactly(t *testing.T) {
	begin, end := "# >>> B >>>", "# <<< E <<<"
	block := begin + "\nsource x\n" + end + "\n"
	for _, in := range []string{"", "a\n", "a\nb\n", "# comment only\n", "a\n\n\nb\n"} {
		up, _, err := upsertBlock(in, block, begin, end)
		if err != nil {
			t.Fatalf("upsertBlock(%q): %v", in, err)
		}
		down, _, err := removeBlock(up, begin, end)
		if err != nil {
			t.Fatalf("removeBlock: %v", err)
		}
		if down != in {
			t.Errorf("round trip changed the file:\n in: %q\nout: %q", in, down)
		}
	}
}

func TestRemoveBlockRemovesEveryCopy(t *testing.T) {
	begin, end := "# >>> B >>>", "# <<< E <<<"
	block := begin + "\nsource x\n" + end + "\n"
	in := "a\n" + block + "b\n" + block + "c\n"
	out, changed, err := removeBlock(in, begin, end)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || strings.Contains(out, begin) {
		t.Errorf("a duplicate block survived:\n%s", out)
	}
}

func writeFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
