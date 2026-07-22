package architecture_test

import (
	"os/exec"
	"sort"
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
		"core/adapters/inbound",
		"core/services",
		"github.com/gofiber",
		"gorm.io/",
		"net/http",
	})
}

func TestModerationDomainUsesOnlyStandardLibrary(t *testing.T) {
	imports := listPackageImports(t)
	moderationImports, ok := imports["core/domain/moderation"]
	if !ok {
		t.Fatal("core/domain/moderation package is missing")
	}

	cmd := exec.Command("go", "list", "std")
	cmd.Dir = ".."
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list std: %v", err)
	}
	standard := make(map[string]struct{})
	for _, pkg := range strings.Fields(string(output)) {
		standard[pkg] = struct{}{}
	}

	for _, imported := range moderationImports {
		if _, ok := standard[imported]; !ok {
			t.Fatalf("core/domain/moderation must stay pure; found non-standard dependency %s", imported)
		}
	}
}

func TestApplicationUsecasesDependOnPortsNotInfrastructure(t *testing.T) {
	imports := listPackageImports(t)

	assertNoImports(t, imports, "core/application/usecases", []string{
		"core/repositories",
		"core/infrastructure/repositories",
		"core/infrastructure",
		"core/adapters/inbound",
		"core/infrastructure/db",
		"core/infrastructure/socket",
		"github.com/gofiber",
		"gorm.io/",
		"mime/multipart",
		"net/http",
	})
}

func TestApplicationPortsHaveNoTransportOrInfrastructureDependencies(t *testing.T) {
	imports := listPackageImports(t)

	assertNoImports(t, imports, "core/application/ports", []string{
		"core/infrastructure",
		"core/adapters/inbound",
		"core/repositories",
		"core/services",
		"github.com/gofiber",
		"gorm.io/",
		"mime/multipart",
		"net/http",
	})
}

func TestInboundAdaptersDoNotDependOnInfrastructure(t *testing.T) {
	imports := listPackageImports(t)

	assertNoImports(t, imports, "core/adapters/inbound", []string{
		"core/infrastructure",
		"core/infrastructure/repositories",
		"core/repositories",
		"gorm.io",
	})
}

func TestRepositoryAdaptersHaveNoInboundTransportDependencies(t *testing.T) {
	imports := listPackageImports(t)

	assertNoImports(t, imports, "core/infrastructure/repositories", []string{
		"core/adapters/inbound",
		"core/middleware",
		"core/router",
		"core/routes",
		"github.com/gofiber",
		"mime/multipart",
		"net/http",
	})
}

func TestApplicationTypesHaveNoTransportOrInfrastructureDependencies(t *testing.T) {
	imports := listPackageImports(t)

	assertNoImports(t, imports, "core/application/types", []string{
		"core/adapters/inbound",
		"core/infrastructure",
		"core/middleware",
		"core/repositories",
		"core/router",
		"core/routes",
		"core/utils",
		"github.com/gofiber",
		"gorm.io",
		"mime/multipart",
		"net/http",
	})
}

func TestApplicationTypesDependencyClosureStaysPersistenceFree(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "core/application/types")
	cmd.Dir = ".."
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list application types dependencies: %v", err)
	}

	forbidden := []string{
		"core/models",
		"core/infrastructure",
		"core/adapters/inbound",
		"github.com/gofiber",
		"gorm.io",
	}
	var violations []string
	for _, dependency := range strings.Fields(string(output)) {
		for _, root := range forbidden {
			if isPackageOrSubpackage(dependency, root) {
				violations = append(violations, dependency)
				break
			}
		}
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		t.Fatalf("core/application/types dependency closure reaches persistence or transport:\n%s", strings.Join(violations, "\n"))
	}
}

func TestApplicationUsecasesDoNotDependOnHelpers(t *testing.T) {
	imports := listPackageImports(t)

	assertNoImports(t, imports, "core/application/usecases", []string{
		"core/helpers",
	})
}

func TestBackgroundWorkersDependOnPortsNotInfrastructure(t *testing.T) {
	imports := listPackageImports(t)

	assertNoImports(t, imports, "core/workers", []string{
		"core/infrastructure",
		"core/repositories",
		"gorm.io",
	})
}

func TestOnlyBootstrapMayImportInboundAdaptersFromInfrastructure(t *testing.T) {
	imports := listPackageImports(t)
	if _, ok := imports["core/infrastructure/bootstrap"]; !ok {
		t.Fatal("core/infrastructure/bootstrap package is missing")
	}

	foundInfrastructure := false
	var violations []string
	for pkg, pkgImports := range imports {
		if !isPackageOrSubpackage(pkg, "core/infrastructure") {
			continue
		}
		foundInfrastructure = true
		if isPackageOrSubpackage(pkg, "core/infrastructure/bootstrap") {
			continue
		}
		for _, imported := range pkgImports {
			if isPackageOrSubpackage(imported, "core/adapters/inbound") {
				violations = append(violations, pkg+" imports "+imported)
			}
		}
	}
	if !foundInfrastructure {
		t.Fatal("core/infrastructure package tree is missing")
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		t.Fatalf("only core/infrastructure/bootstrap may compose inbound adapters:\n%s", strings.Join(violations, "\n"))
	}
}

func TestInterfaceHandlersDoNotReachRepositories(t *testing.T) {
	imports := listPackageImports(t)

	assertNoImports(t, imports, "core/adapters/inbound/http/routes/handlers", []string{
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

	foundPackage := false
	var violations []string
	for pkg, pkgImports := range imports {
		if !isPackageOrSubpackage(pkg, packagePrefix) {
			continue
		}
		foundPackage = true
		for _, imported := range pkgImports {
			for _, bad := range forbidden {
				if isPackageOrSubpackage(imported, bad) {
					violations = append(violations, pkg+" imports "+imported)
					break
				}
			}
		}
	}
	if !foundPackage {
		t.Fatalf("architecture guard package tree %s is missing", strings.TrimSuffix(packagePrefix, "/"))
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		t.Fatalf("%s has forbidden dependencies:\n%s", strings.TrimSuffix(packagePrefix, "/"), strings.Join(violations, "\n"))
	}
}

func isPackageOrSubpackage(pkg, root string) bool {
	root = strings.TrimSuffix(root, "/")
	return pkg == root || strings.HasPrefix(pkg, root+"/")
}
