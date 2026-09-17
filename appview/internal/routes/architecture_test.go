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

	function := parsedFunction(t, "routes_profile_notification.go", "registerProfileRelationshipRoutes")
	for constructor, resolverArgument := range map[string]int{
		"FollowProfileHandler":           2,
		"UnfollowProfileHandler":         2,
		"MuteProfileHandler":             2,
		"UnmuteProfileHandler":           2,
		"BlockProfileHandler":            2,
		"UnblockProfileHandler":          2,
		"NewProfileReportTargetResolver": 1,
	} {
		assertCallArgument(t, function, constructor, resolverArgument, "routes.authoritativeResolver")
	}
	assertCallArgument(t, function, "GetProfileHandler", 2, "routes.handleResolver")
}

// REG-004: interaction list reads stay on AppView and cannot invoke PDS effects.
func TestPostInteractionReadRoutesUseOnlyReadDependencies(t *testing.T) {
	t.Parallel()

	function := parsedFunction(t, "routes_scheduled_post.go", "registerPostRoutes")
	wantArguments := map[string][]string{
		"ListPostLikesHandler":   {"routes.postStore", "routes.logger"},
		"ListPostRepostsHandler": {"routes.postStore", "routes.logger"},
		"ListPostQuotesHandler":  {"routes.postStore", "routes.handleResolver", "routes.logger", "routes.languages"},
	}
	for constructor, want := range wantArguments {
		call := uniqueCall(t, function, constructor)
		if len(call.Args) != len(want) {
			t.Errorf("%s arguments = %d, want %d read dependencies", constructor, len(call.Args), len(want))
			continue
		}
		for index, argument := range want {
			if got := selectorName(call.Args[index]); got != argument {
				t.Errorf("%s argument %d = %q, want %q", constructor, index, got, argument)
			}
		}
	}
}

func parsedFunction(t *testing.T, path, name string) *ast.FuncDecl {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == name {
			return function
		}
	}
	t.Fatalf("%s is missing function %s", path, name)
	return nil
}

func assertCallArgument(t *testing.T, function *ast.FuncDecl, constructor string, index int, want string) {
	t.Helper()
	call := uniqueCall(t, function, constructor)
	if len(call.Args) <= index {
		t.Fatalf("%s has %d arguments, need resolver argument %d", constructor, len(call.Args), index)
	}
	if got := selectorName(call.Args[index]); got != want {
		t.Errorf("%s resolver = %q, want %q", constructor, got, want)
	}
}

func uniqueCall(t *testing.T, function *ast.FuncDecl, name string) *ast.CallExpr {
	t.Helper()
	var matches []*ast.CallExpr
	ast.Inspect(function.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if ok && selectorName(call.Fun) == "api."+name {
			matches = append(matches, call)
		}
		return true
	})
	if len(matches) != 1 {
		t.Fatalf("%s contains %d calls to api.%s, want 1", function.Name.Name, len(matches), name)
	}
	return matches[0]
}

func selectorName(expression ast.Expr) string {
	switch expression := expression.(type) {
	case *ast.Ident:
		return expression.Name
	case *ast.SelectorExpr:
		prefix := selectorName(expression.X)
		if prefix == "" {
			return expression.Sel.Name
		}
		return prefix + "." + expression.Sel.Name
	case *ast.ParenExpr:
		return selectorName(expression.X)
	default:
		return ""
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
