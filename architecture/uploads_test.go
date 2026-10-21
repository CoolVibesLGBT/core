package architecture_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestUploadsUseOneMediaTypeAndConstructor(t *testing.T) {
	canonical := filepath.Clean("../domain/media/upload.go")
	adapter := filepath.Clean("../adapters/media/upload.go")
	definitions := 0
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		positions := token.NewFileSet()
		file, err := parser.ParseFile(positions, path, nil, 0)
		if err != nil {
			return err
		}
		imports := make(map[string]string)
		for _, spec := range file.Imports {
			imported, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			name := filepath.Base(imported)
			if spec.Name != nil {
				name = spec.Name.Name
			}
			imports[name] = imported
		}
		mediaType := func(expression ast.Expr) bool {
			if name, ok := expression.(*ast.Ident); ok {
				return name.Name == "UploadedFile"
			}
			selector, ok := expression.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "UploadedFile" {
				return false
			}
			pkg, ok := selector.X.(*ast.Ident)
			return ok && imports[pkg.Name] == "core/domain/media"
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.TypeSpec:
				if strings.HasSuffix(strings.ToLower(node.Name.Name), "uploadedfile") || mediaType(node.Type) {
					definitions++
					if filepath.Clean(path) != canonical || node.Name.Name != "UploadedFile" {
						t.Errorf("%s: reuse core/domain/media.UploadedFile instead of defining another upload type", positions.Position(node.Pos()))
					}
				}
			case *ast.CompositeLit:
				if mediaType(node.Type) && filepath.Clean(path) != canonical {
					t.Errorf("%s: construct uploads through the shared media constructor", positions.Position(node.Pos()))
				}
			case *ast.CallExpr:
				selector, ok := node.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "NewUploadedFile" {
					break
				}
				pkg, ok := selector.X.(*ast.Ident)
				if ok && imports[pkg.Name] == "core/domain/media" && !strings.HasSuffix(path, "_test.go") && filepath.Clean(path) != adapter {
					t.Errorf("%s: adapt upload sources in core/adapters/media", positions.Position(node.Pos()))
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if definitions != 1 {
		t.Fatalf("upload type definitions = %d, want 1", definitions)
	}
}
