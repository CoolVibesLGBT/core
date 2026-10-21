package architecture_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestRepositoryAndServiceErrorsAreCentralized(t *testing.T) {
	for _, directory := range []string{"../application/ports", "../application/usecases", "../infrastructure/repositories"} {
		entries, err := os.ReadDir(directory)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			path := filepath.Join(directory, entry.Name())
			positions := token.NewFileSet()
			file, err := parser.ParseFile(positions, path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			imports := make(map[string]string)
			for _, spec := range file.Imports {
				imported, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					t.Fatal(err)
				}
				name := filepath.Base(imported)
				if spec.Name != nil {
					name = spec.Name.Name
				}
				imports[name] = imported
			}
			if entry.Name() != "errors.go" {
				for _, declaration := range file.Decls {
					general, ok := declaration.(*ast.GenDecl)
					if !ok || (general.Tok != token.VAR && general.Tok != token.CONST) {
						continue
					}
					for _, spec := range general.Specs {
						for _, name := range spec.(*ast.ValueSpec).Names {
							if strings.HasPrefix(name.Name, "Err") || strings.HasPrefix(name.Name, "err") {
								t.Errorf("%s: move %s to errors.go", positions.Position(name.Pos()), name.Name)
							}
						}
					}
				}
			}
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, ok := selector.X.(*ast.Ident)
				if !ok {
					return true
				}
				if imports[pkg.Name] == "errors" && selector.Sel.Name == "New" {
					t.Errorf("%s: register error messages in constants/errors.go", positions.Position(call.Pos()))
				}
				if imports[pkg.Name] == "fmt" && selector.Sel.Name == "Errorf" {
					// Message-free wrapping may add context to an existing domain or application error.
					literal, ok := call.Args[0].(*ast.BasicLit)
					if !ok || (literal.Value != `"%w: %w"` && literal.Value != `"%w: %s"`) {
						t.Errorf("%s: register error message templates in constants/errors.go", positions.Position(call.Pos()))
					}
				}
				return true
			})
		}
	}
}
