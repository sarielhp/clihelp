package clihelp

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// A helper copied into a subpackage drifts. This test makes that a decision
// rather than an accident.
//
// When doc/ and tree/ were split out, four unexported helpers were duplicated
// instead of shared, and three of them diverged — two into real defects: a width
// measurement that could not see OSC 8 hyperlinks, and a sentence truncator that
// returned newlines into a reflow assuming one line. Nothing failed. Both
// packages compiled, both test suites passed, and the divergence was found two
// months later by regenerating documentation.
//
// So: any unexported function declared both in this package and in a subpackage
// has to be named below, with a reason. That is deliberately annoying. The point
// is that the next person to copy a helper has to write down why, in a place a
// reviewer will read, instead of the copy sitting there looking like code.
//
// Exported names are not checked: clihelp.Options and tree.Options are different
// types in different packages, and clihelp.Render and tree.Render are different
// entry points. Those are the package boundary working as intended.
var sharedHelperExceptions = map[string]string{
	"firstSentence":     "tree/tree.go: one-line wrapper around clihelp.FirstSentence; tree/drift_test.go asserts it agrees",
	"subcommandEntries": "doc/md.go: one-line wrapper around clihelp.SubcommandList; doc/drift_test.go asserts it agrees",
	"visualLen":         "tree/tree.go: one-line wrapper around clihelp.VisualWidth; tree/drift_test.go asserts it agrees",
}

func TestNoUnexplainedHelperCopiesInSubpackages(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	rootFuncs := unexportedFuncs(t, root)
	if len(rootFuncs) == 0 {
		t.Fatal("parsed no unexported functions in the root package; the test is broken, not the code")
	}

	subpackages, err := filepath.Glob(filepath.Join(root, "*", ""))
	if err != nil {
		t.Fatal(err)
	}

	for _, dir := range subpackages {
		info, statErr := os.Stat(dir)
		if statErr != nil || !info.IsDir() {
			continue
		}
		switch filepath.Base(dir) {
		case "doc", "tree": // the library's own subpackages
		default:
			continue // example apps, docs, tools, review, prompts, .git
		}

		for _, complaint := range unexplainedCopies(rootFuncs, unexportedFuncs(t, dir), sharedHelperExceptions) {
			t.Error(complaint)
		}
	}

	// A stale exception is its own small lie: it says a copy exists when it does
	// not, so the next reader trusts the list and stops looking.
	for name := range sharedHelperExceptions {
		if _, ok := rootFuncs[name]; !ok {
			t.Errorf("sharedHelperExceptions lists %q, which the root package no longer declares", name)
		}
	}
}

// unexplainedCopies is the comparison, separated from the parsing so that it can
// be tested directly. Driving it through a real copied helper would mean putting
// a defect in the tree to see the guard fire, which is a poor trade.
func unexplainedCopies(root, sub, exceptions map[string]string) []string {
	var out []string
	names := make([]string, 0, len(sub))
	for name := range sub {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if _, ok := root[name]; !ok {
			continue
		}
		if reason, allowed := exceptions[name]; allowed {
			if strings.TrimSpace(reason) == "" {
				out = append(out, name+" is listed as a shared helper with no reason given")
			}
			continue
		}
		out = append(out, name+" is declared in both the root package ("+root[name]+
			") and "+sub[name]+".\nEither call the root package's version, or add it to "+
			"sharedHelperExceptions with a reason.\nA copy that nobody wrote a reason for is how "+
			"tree's width measurement stopped understanding hyperlinks.")
	}
	return out
}

// unexportedFuncs maps each unexported top-level function name in dir to the
// file it was found in. Methods are skipped: a method belongs to its receiver's
// type, so the same name on two different types is not a copy.
func unexportedFuncs(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	fset := token.NewFileSet()
	// One file at a time rather than parser.ParseDir, which is deprecated as of
	// Go 1.25 and makes staticcheck — and therefore tools/check.sh — fail.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if parseErr != nil {
			t.Fatalf("parsing %s: %v", name, parseErr)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name == nil {
				continue
			}
			if fname := fn.Name.Name; !ast.IsExported(fname) {
				out[fname] = name
			}
		}
	}
	return out
}

// The exception list is read by whoever the guard fires on, so keep it in order.
//
// This has to read the source: ranging a map gives Go's deliberately randomised
// order, so the order as written is not observable from inside the test. The
// first version of this ranged the map and used t.Logf, which meant it could not
// fail and was not checking anything.
func TestSharedHelperExceptionsAreSorted(t *testing.T) {
	src, err := os.ReadFile("crosspackage_test.go")
	if err != nil {
		t.Fatal(err)
	}
	const marker = "sharedHelperExceptions = map[string]string{"
	body := string(src)
	start := strings.Index(body, marker)
	if start < 0 {
		t.Fatalf("could not find %q; this test needs updating", marker)
	}
	body = body[start+len(marker):]
	end := strings.Index(body, "\n}")
	if end < 0 {
		t.Fatal("could not find the end of the exception list")
	}

	var keys []string
	for _, line := range strings.Split(body[:end], "\n") {
		if i := strings.Index(line, `"`); i >= 0 {
			if j := strings.Index(line[i+1:], `"`); j >= 0 {
				keys = append(keys, line[i+1:i+1+j])
			}
		}
	}
	if len(keys) != len(sharedHelperExceptions) {
		t.Fatalf("read %d keys from the source but the map has %d", len(keys), len(sharedHelperExceptions))
	}
	if !sort.StringsAreSorted(keys) {
		sorted := append([]string(nil), keys...)
		sort.Strings(sorted)
		t.Errorf("sharedHelperExceptions is not sorted:\n  got  %v\n  want %v", keys, sorted)
	}
}

// The guard has to fire on a copy, stay quiet on an explained one, and complain
// about an exception with no reason. Checked here rather than by copying a
// helper into a subpackage for real.
func TestUnexplainedCopiesDetection(t *testing.T) {
	root := map[string]string{"visualLen": "format.go", "wrapWidth": "format.go"}

	for _, tt := range []struct {
		name       string
		sub        map[string]string
		exceptions map[string]string
		want       int
	}{
		{"an unexplained copy is reported", map[string]string{"wrapWidth": "tree.go"}, nil, 1},
		{"an explained copy is allowed", map[string]string{"visualLen": "tree.go"},
			map[string]string{"visualLen": "wrapper, asserted in drift_test.go"}, 0},
		{"an exception with no reason is reported", map[string]string{"visualLen": "tree.go"},
			map[string]string{"visualLen": "   "}, 1},
		{"a name only the subpackage has is fine", map[string]string{"renderTreeNode": "tree.go"}, nil, 0},
		{"two copies are both reported", map[string]string{"visualLen": "tree.go", "wrapWidth": "tree.go"}, nil, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := unexplainedCopies(root, tt.sub, tt.exceptions); len(got) != tt.want {
				t.Errorf("got %d complaints, want %d: %v", len(got), tt.want, got)
			}
		})
	}
}
