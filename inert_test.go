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

// A test that cannot fail is worse than no test: it occupies the place where the
// real one would go, and it reports success forever.
//
// Four were found here at once. A test named "GlobalFlagsBound" logged the two
// values it existed to check and asserted only that Execute returned no error,
// so the flags could have stopped binding to their targets entirely. A fuzzer's
// comment said "a second bind of the same spec must be refused, not panic" and
// then discarded both results, so every name-collision check in options.go could
// have been deleted with a million executions still green. PrintError was called
// twice and its output never looked at — it also printed into the real stderr,
// which is why "Error: sample error" appeared in the middle of every run. And a
// test that every help flag "actually runs" checked only the error, so a flag
// that quietly printed nothing passed as working.
//
// Two earlier ones set the precedent: TestSharedHelperExceptionsAreSorted used
// t.Logf where it meant t.Errorf and could not fail — once fixed it immediately
// caught that the list it guards was unsorted — and an assertion that no escapes
// were emitted was vacuous in a process where fatih/color had already turned
// colour off.
//
// So: every test function must contain something that can fail it. The exception
// list is deliberately awkward to add to.
var inertTestExceptions = map[string]string{}

// assertionPrefixes name the helpers this suite fails through, beyond t.Error
// and t.Fatal themselves.
var assertionPrefixes = []string{"assert", "require", "must", "expect", "Assert"}

func TestEveryTestCanFail(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dirs := []string{root}
	matches, err := filepath.Glob(filepath.Join(root, "*"))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range matches {
		if info, statErr := os.Stat(m); statErr == nil && info.IsDir() {
			switch filepath.Base(m) {
			case "doc", "tree", "example", "clihelptest":
				dirs = append(dirs, m)
			}
		}
	}

	var inert []string
	checked := 0
	for _, dir := range dirs {
		names, funcs, helpers := testFuncsIn(t, dir)
		checked += len(names)
		for _, name := range names {
			if _, ok := inertTestExceptions[name]; ok {
				continue
			}
			if !canFail(funcs[name], helpers, map[string]bool{}) {
				inert = append(inert, filepath.Base(dir)+":"+name)
			}
		}
	}
	if checked == 0 {
		t.Fatal("parsed no test functions at all; this guard is broken, not the suite")
	}
	sort.Strings(inert)
	for _, name := range inert {
		t.Errorf("%s contains nothing that can fail it: no t.Error/t.Fatal and no "+
			"assertion helper. Give it an assertion, or add it to inertTestExceptions "+
			"with a reason.", name)
	}

	for name := range inertTestExceptions {
		found := false
		for _, dir := range dirs {
			if names, _, _ := testFuncsIn(t, dir); contains(names, name) {
				found = true
			}
		}
		if !found {
			t.Errorf("inertTestExceptions lists %q, which no longer exists", name)
		}
	}
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

// testFuncsIn parses every _test.go file in dir and returns the top-level Test
// and Fuzz functions, by name. TestMain is skipped: it orchestrates a run rather
// than asserting anything, and this package's own reports through os.Exit.
func testFuncsIn(t *testing.T, dir string) ([]string, map[string]*ast.FuncDecl, map[string]*ast.FuncDecl) {
	t.Helper()
	fset := token.NewFileSet()
	out := map[string]*ast.FuncDecl{}
	all := map[string]*ast.FuncDecl{}
	var names []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, 0)
		if parseErr != nil {
			t.Fatalf("parsing %s: %v", e.Name(), parseErr)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Body == nil {
				continue
			}
			name := fn.Name.Name
			all[name] = fn
			if name == "TestMain" || !(strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Fuzz")) {
				continue
			}
			names = append(names, name)
			out[name] = fn
		}
	}
	sort.Strings(names)
	return names, out, all
}

// canFail reports whether anything in fn's body can fail the test: a call to
// t.Error/t.Fatal (under any receiver name, including a subtest's), a call to a
// helper whose name says it asserts, or a call to another function in this
// package's test files that can itself fail. The last is what lets a test read
// as three named steps without this guard calling it inert.
func canFail(fn *ast.FuncDecl, helpers map[string]*ast.FuncDecl, seen map[string]bool) bool {
	if fn == nil || seen[fn.Name.Name] {
		return false
	}
	seen[fn.Name.Name] = true
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || found {
			return !found
		}
		switch sel := call.Fun.(type) {
		case *ast.SelectorExpr:
			switch sel.Sel.Name {
			case "Error", "Errorf", "Fatal", "Fatalf":
				found = true
			default:
				if hasAssertionPrefix(sel.Sel.Name) {
					found = true
				}
			}
		case *ast.Ident:
			if hasAssertionPrefix(sel.Name) || canFail(helpers[sel.Name], helpers, seen) {
				found = true
			}
		}
		return !found
	})
	return found
}

func hasAssertionPrefix(name string) bool {
	for _, p := range assertionPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}
