package tool

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Runtime inventory tools must never import os/exec (design: no host binaries).
func TestRuntimeToolsDoNotImportOSExec(t *testing.T) {
	files := []string{"ps.go", "ps_grep.go", "ss.go", "ss_grep.go"}
	for _, name := range files {
		path := filepath.Join(".", name)
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, src, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if p == "os/exec" {
				t.Errorf("%s imports os/exec", name)
			}
		}
		// Also forbid obvious exec helpers in the AST body.
		f, err = parser.ParseFile(fset, path, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "exec" {
				t.Errorf("%s references exec.%s", name, sel.Sel.Name)
			}
			return true
		})
	}
}
