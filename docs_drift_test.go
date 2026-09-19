package clihelp

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Documentation that names something the library does not have teaches it
// confidently, and an agent following it writes code that does not compile.
//
// This is not hypothetical: "clihelp.PrintError(err)" appeared in six documents,
// eight times, for the whole life of the project. It is a method on App. Anyone
// — person or agent — who copied it got a build error and no clue why, because
// the documentation said otherwise. Three other documents named functions that
// had been removed.
//
// So every "clihelp.X" in the prose must be a package-level name, and every
// "App.X" / "Command.X" / … must be a field or method of that type.
var docDriftExceptions = map[string]string{
	"Context.Done": "ctx.Context is an embedded context.Context; Done is its method",
	"Context.Err":  "ctx.Context is an embedded context.Context; Err is its method",
}

// documentedTypes are the types whose members the prose refers to by name.
var documentedTypes = []string{
	"App", "Command", "Option", "Options", "Theme", "Note", "Param", "Context", "Example",
}

func TestDocumentationNamesThingsThatExist(t *testing.T) {
	pkgLevel, methods, fields := exportedSurface(t)
	if len(pkgLevel) == 0 {
		t.Fatal("parsed no exported names; this guard is broken, not the documentation")
	}

	files, err := filepath.Glob("docs/*.md")
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, "README.md", "llms.txt")

	pkgRef := regexp.MustCompile(`\bclihelp\.([A-Z]\w*)`)
	typeRefs := map[string]*regexp.Regexp{}
	for _, name := range documentedTypes {
		typeRefs[name] = regexp.MustCompile(`\b` + name + `\.([A-Z]\w*)`)
	}

	checked := 0
	for _, file := range files {
		body, readErr := os.ReadFile(file)
		if readErr != nil {
			continue // a document that is not there is not a drift
		}
		for i, line := range strings.Split(string(body), "\n") {
			for _, m := range pkgRef.FindAllStringSubmatch(line, -1) {
				checked++
				if _, ok := docDriftExceptions["clihelp."+m[1]]; ok {
					continue
				}
				if pkgLevel[m[1]] {
					continue
				}
				why := "no such exported name"
				if methods[m[1]] {
					why = "a method, not a package-level function — write it on the value"
				}
				t.Errorf("%s:%d names clihelp.%s: %s", file, i+1, m[1], why)
			}
			for typeName, re := range typeRefs {
				for _, m := range re.FindAllStringSubmatch(line, -1) {
					checked++
					ref := typeName + "." + m[1]
					if _, ok := docDriftExceptions[ref]; ok {
						continue
					}
					if fields[typeName][m[1]] || methods[m[1]] {
						continue
					}
					t.Errorf("%s:%d names %s, which is no field or method of %s", file, i+1, ref, typeName)
				}
			}
		}
	}
	if checked == 0 {
		t.Error("no documented identifiers were checked at all; the patterns have stopped matching")
	}

	for ref := range docDriftExceptions {
		if !strings.Contains(ref, ".") {
			t.Errorf("docDriftExceptions key %q is not a qualified reference", ref)
		}
	}
}

// exportedSurface reads the package's own source for what it actually offers.
func exportedSurface(t *testing.T) (pkgLevel, methods map[string]bool, fields map[string]map[string]bool) {
	t.Helper()
	pkgLevel, methods = map[string]bool{}, map[string]bool{}
	fields = map[string]map[string]bool{}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(fset, name, nil, 0)
		if parseErr != nil {
			t.Fatalf("parsing %s: %v", name, parseErr)
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if !d.Name.IsExported() {
					continue
				}
				if d.Recv != nil {
					methods[d.Name.Name] = true
				} else {
					pkgLevel[d.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if !s.Name.IsExported() {
							continue
						}
						pkgLevel[s.Name.Name] = true
						st, ok := s.Type.(*ast.StructType)
						if !ok {
							continue
						}
						fields[s.Name.Name] = map[string]bool{}
						for _, f := range st.Fields.List {
							for _, n := range f.Names {
								if n.IsExported() {
									fields[s.Name.Name][n.Name] = true
								}
							}
						}
					case *ast.ValueSpec:
						for _, n := range s.Names {
							if n.IsExported() {
								pkgLevel[n.Name] = true
							}
						}
					}
				}
			}
		}
	}
	names := make([]string, 0, len(pkgLevel))
	for n := range pkgLevel {
		names = append(names, n)
	}
	sort.Strings(names)
	return pkgLevel, methods, fields
}
