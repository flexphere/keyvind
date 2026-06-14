package keyvind

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// These tests guard the core's design invariants mechanically, so a drift like
// "an editor concept leaked into the parser" fails `make all` instead of slipping
// through review. They are tripwires, not proofs — see CLAUDE.md for the full
// (semantic) invariants a reviewer still has to check.

// editorTerms are concepts the core must never know about: it discriminates
// commands from key sequences and stops there. Text editing, selection, cursors,
// and specific editor-mode semantics belong to the host. (Modes themselves are
// fine — they are arbitrary user-defined namespaces, so "mode" is not listed.)
var editorTerms = []string{
	"selection", "visual", "insert", "cursor", "undo", "redo",
	"yank", "paste", "scroll", "highlight", "clipboard", "viewport",
}

// coreFiles parses the non-test .go files of the core package (the repo root).
func coreFiles(t *testing.T) []*ast.File {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		t.Fatal("no core .go files found")
	}
	return files
}

func editorTermIn(s string) string {
	low := strings.ToLower(s)
	for _, term := range editorTerms {
		if strings.Contains(low, term) {
			return term
		}
	}
	return ""
}

// TestCoreHasNoEditorVocabulary fails if any declared identifier or string
// literal in the core carries editor vocabulary — the exact shape of the
// "selection mode" leak. It inspects names, not comments, so prose may still
// mention the host's selection.
func TestCoreHasNoEditorVocabulary(t *testing.T) {
	report := func(kind, name string) {
		if term := editorTermIn(name); term != "" {
			t.Errorf("core %s %q contains editor term %q — the core must not know editor concepts", kind, name, term)
		}
	}
	for _, f := range coreFiles(t) {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				report("identifier", x.Name.Name)
			case *ast.TypeSpec:
				report("identifier", x.Name.Name)
			case *ast.ValueSpec:
				for _, id := range x.Names {
					report("identifier", id.Name)
				}
			case *ast.Field:
				for _, id := range x.Names {
					report("field", id.Name)
				}
			case *ast.BasicLit:
				if x.Kind == token.STRING {
					report("string literal", x.Value)
				}
			}
			return true
		})
	}
}

// TestCoreImportsOnlyStdlib fails if the core imports any non-standard-library
// package, keeping it framework-agnostic and dependency-free.
func TestCoreImportsOnlyStdlib(t *testing.T) {
	for _, f := range coreFiles(t) {
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			first := path
			if i := strings.IndexByte(path, '/'); i >= 0 {
				first = path[:i]
			}
			if strings.Contains(first, ".") {
				t.Errorf("core imports non-stdlib package %q — the core must stay dependency-free", path)
			}
		}
	}
}
