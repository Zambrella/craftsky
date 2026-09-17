package routes

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestFollowerGrowthProductionDependenciesStayPrivateAndNonInfluential(t *testing.T) {
	t.Parallel()

	allowedPackages := map[string]bool{
		"api":            true,
		"app":            true,
		"followergrowth": true,
		"observability":  true,
		"ownerlifecycle": true,
		"routes":         true,
	}
	protectedBoundaries := []string{
		"feed", "timeline", "rank", "recommend", "search", "discover", "moderation", "advert",
	}

	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		relative := filepath.ToSlash(strings.TrimPrefix(path, "../"))
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		referencesGrowth := referencesFollowerGrowth(file)
		ownerPackage := strings.Split(relative, "/")[0]
		if referencesGrowth && !allowedPackages[ownerPackage] {
			t.Errorf("unapproved follower-growth production dependency in package %s (%s)", ownerPackage, relative)
		}
		for _, boundary := range protectedBoundaries {
			if strings.Contains(strings.ToLower(relative), boundary) && referencesGrowth {
				t.Errorf("follower growth crossed %s boundary in %s", boundary, relative)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("inventory production dependencies: %v", err)
	}

	for _, absentSubsystem := range []string{"advertising", "recommendation"} {
		if _, err := os.Stat(filepath.Join("..", absentSubsystem)); err == nil || !os.IsNotExist(err) {
			t.Errorf("%s subsystem now exists; add it explicitly to the follower-growth non-interference inventory", absentSubsystem)
		}
	}
}

func referencesFollowerGrowth(file *ast.File) bool {
	for _, imported := range file.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err == nil && strings.HasSuffix(path, "/internal/followergrowth") {
			return true
		}
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.Ident:
			found = found || strings.Contains(strings.ToLower(node.Name), "followergrowth")
		case *ast.BasicLit:
			if node.Kind == token.STRING {
				value, err := strconv.Unquote(node.Value)
				found = found || err == nil && strings.Contains(strings.ToLower(value), "follower_growth")
			}
		}
		return !found
	})
	return found
}
