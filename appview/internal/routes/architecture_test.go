package routes

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestAddRoutesDelegatesToNarrowCapabilityRegistrars(t *testing.T) {
	t.Parallel()

	want := map[string]bool{
		"registerPublicOperationsRoutes":    false,
		"registerPublicOAuthRoutes":         false,
		"registerAuthRoutes":                false,
		"registerSearchRoutes":              false,
		"registerAccountDeletionRoutes":     false,
		"registerMigrationRoutes":           false,
		"registerProfileRelationshipRoutes": false,
		"registerNotificationRoutes":        false,
		"registerScheduledPostRoutes":       false,
		"registerPostRoutes":                false,
	}
	for _, name := range productionRouteFiles(t) {
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		if file.Name.Name != "routes" {
			t.Fatalf("production route file %s belongs to package %s", name, file.Name.Name)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil {
				continue
			}
			if function.Name.Name == "AddRoutes" {
				ast.Inspect(function.Body, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok {
						return true
					}
					selector, ok := call.Fun.(*ast.SelectorExpr)
					if ok && selector.Sel.Name == "Handle" {
						t.Errorf("AddRoutes registers a handler directly; delegate registration to a capability registrar")
					}
					return true
				})
			}
			if _, expected := want[function.Name.Name]; !expected {
				continue
			}
			want[function.Name.Name] = true
			ast.Inspect(function.Type.Params, func(node ast.Node) bool {
				identifier, ok := node.(*ast.Ident)
				if ok && identifier.Name == "Dependencies" {
					t.Errorf("%s accepts aggregate Dependencies; use a capability bundle", function.Name.Name)
				}
				return true
			})
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("missing capability registrar %s", name)
		}
	}
}

func TestProductionRoutesDoNotImportAppComposition(t *testing.T) {
	t.Parallel()

	for _, name := range productionRouteFiles(t) {
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imported := range file.Imports {
			path, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				t.Fatalf("unquote import in %s: %v", name, err)
			}
			if strings.HasSuffix(path, "/internal/app") {
				t.Errorf("production route file %s imports root composition package %s", name, path)
			}
		}
	}
}

func TestHandleTargetedMutationsUseAuthoritativeIdentityResolver(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("routes_profile_notification.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	for _, constructor := range []string{
		"FollowProfileHandler", "UnfollowProfileHandler",
		"MuteProfileHandler", "UnmuteProfileHandler",
		"BlockProfileHandler", "UnblockProfileHandler",
		"NewProfileReportTargetResolver",
	} {
		needle := constructor + "("
		start := strings.Index(source, needle)
		if start < 0 {
			t.Errorf("missing mutation constructor %s", constructor)
			continue
		}
		end := strings.Index(source[start:], "))")
		if end < 0 {
			end = min(len(source)-start, 600)
		}
		call := source[start : start+end]
		if !strings.Contains(call, "routes.authoritativeResolver") {
			t.Errorf("%s is not wired to routes.authoritativeResolver", constructor)
		}
	}
	if !strings.Contains(source, "GetProfileHandler(routes.profileStore, routes.handleResolver") {
		t.Error("profile display route no longer uses the cached handle resolver")
	}
}

func productionRouteFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		matched, err := build.Default.MatchFile(".", name)
		if err != nil {
			t.Fatalf("evaluate build constraints for %s: %v", name, err)
		}
		if matched {
			files = append(files, name)
		}
	}
	return files
}
