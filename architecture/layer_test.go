package architecture_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestDomainLayerHasNoOuterDependencies(t *testing.T) {
	imports := listPackageImports(t)

	assertNoImports(t, imports, "core/domain/", []string{
		"core/models",
		"core/repositories",
		"core/infrastructure/repositories",
		"core/infrastructure",
		"core/application",
		"core/routes",
		"core/services",
		"github.com/gofiber",
		"gorm.io/",
		"net/http",
	})
}

func TestApplicationUsecasesDependOnPortsNotInfrastructure(t *testing.T) {
	imports := listPackageImports(t)

	assertNoImports(t, imports, "core/application/usecases", []string{
		"core/repositories",
		"core/infrastructure/repositories",
		"core/infrastructure",
		"core/routes",
		"core/infrastructure/db",
		"core/infrastructure/socket",
		"github.com/gofiber",
		"gorm.io/",
		"net/http",
	})
}

func TestInterfaceHandlersDoNotReachRepositories(t *testing.T) {
	imports := listPackageImports(t)

	assertNoImports(t, imports, "core/routes/handlers", []string{
		"core/repositories",
		"core/infrastructure/repositories",
		"core/infrastructure",
		"gorm.io/",
	})
}

func TestLegacyServicesLayerIsNotUsed(t *testing.T) {
	imports := listPackageImports(t)

	for pkg, pkgImports := range imports {
		if strings.HasPrefix(pkg, "core/services") {
			t.Fatalf("legacy services layer still exists as package %s", pkg)
		}
		for _, imported := range pkgImports {
			if strings.HasPrefix(imported, "core/services") {
				t.Fatalf("%s imports legacy services layer dependency %s", pkg, imported)
			}
		}
	}
}

func listPackageImports(t *testing.T) map[string][]string {
	t.Helper()

	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}}{{range .Imports}}\n\t{{.}}{{end}}\n--END--", "./...")
	cmd.Dir = ".."
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list imports: %v", err)
	}

	result := make(map[string][]string)
	var current string
	for _, line := range strings.Split(string(output), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		if line == "--END--" {
			current = ""
			continue
		}

		if !strings.HasPrefix(line, "\t") {
			current = strings.TrimSpace(line)
			result[current] = nil
			continue
		}

		if current != "" {
			result[current] = append(result[current], strings.TrimSpace(line))
		}
	}

	return result
}

func assertNoImports(t *testing.T, imports map[string][]string, packagePrefix string, forbidden []string) {
	t.Helper()

	for pkg, pkgImports := range imports {
		if !strings.HasPrefix(pkg, packagePrefix) {
			continue
		}
		for _, imported := range pkgImports {
			for _, bad := range forbidden {
				if strings.HasPrefix(imported, bad) {
					t.Fatalf("%s imports forbidden dependency %s", pkg, imported)
				}
			}
		}
	}
}
