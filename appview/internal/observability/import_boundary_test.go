package observability

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestSentryImportBoundary(t *testing.T) {
	root := moduleRoot(t)
	allowedProductionPrefixes := []string{
		filepath.Join(root, "internal", "observability") + string(filepath.Separator),
	}
	allowedTestPrefixes := []string{
		filepath.Join(root, "internal", "observability") + string(filepath.Separator),
		filepath.Join(root, "internal", "middleware") + string(filepath.Separator),
		filepath.Join(root, "internal", "tap") + string(filepath.Separator),
		filepath.Join(root, "internal", "api") + string(filepath.Separator),
	}

	var violations []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		importsSentry, err := fileImports(path, "github.com/getsentry/sentry-go")
		if err != nil {
			return err
		}
		if !importsSentry {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, "_test.go") {
			if hasPathPrefix(path, allowedTestPrefixes) {
				return nil
			}
			violations = append(violations, rel+" imports sentry-go from an unapproved test package")
			return nil
		}
		if hasPathPrefix(path, allowedProductionPrefixes) {
			return nil
		}
		violations = append(violations, rel+" imports sentry-go outside internal/observability")
		return nil
	})
	if err != nil {
		t.Fatalf("scan imports: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("sentry-go import boundary violations:\n%s", strings.Join(violations, "\n"))
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatal("could not find go.mod")
		}
		wd = parent
	}
}

func fileImports(path, importPath string) (bool, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(file.Imports, func(spec *ast.ImportSpec) bool {
		return strings.Trim(spec.Path.Value, `"`) == importPath
	}), nil
}

func hasPathPrefix(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
