package repositories

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestRepositoriesUseGORMBuildersInsteadOfRawStatements(t *testing.T) {
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	statement := regexp.MustCompile(`(?i)\b(SELECT\s+|INSERT\s+INTO\s+|UPDATE\s+\w+\s+SET\s+|DELETE\s+FROM\s+|WITH\s+RECURSIVE\s+)`)
	for _, entry := range files {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		positions := token.NewFileSet()
		file, err := parser.ParseFile(positions, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch value := node.(type) {
			case *ast.CallExpr:
				if method, ok := value.Fun.(*ast.SelectorExpr); ok && (method.Sel.Name == "Raw" || method.Sel.Name == "Exec" || method.Sel.Name == "QueryContext" || method.Sel.Name == "ExecContext") {
					t.Errorf("%s: build persistence queries through GORM, found %s", positions.Position(value.Pos()), method.Sel.Name)
				}
			case *ast.BasicLit:
				if value.Kind == token.STRING {
					literal, err := strconv.Unquote(value.Value)
					if err == nil && statement.MatchString(literal) {
						t.Errorf("%s: handwritten SQL statement; use a GORM subquery", positions.Position(value.Pos()))
					}
				}
			}
			return true
		})
	}
}
