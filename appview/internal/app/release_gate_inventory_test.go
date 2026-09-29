package app

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestReleaseGatesUseFixturesInsteadOfLiveBluesky(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate release inventory source")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", "..", ".."))
	liveOriginPattern := regexp.MustCompile(`https://(?:bsky\.social|auth\.bsky\.app|api\.bsky\.app|public\.api\.bsky\.app)`)
	accountCreationMethod := "com.atproto.server." + "createAccount"

	allowedLiveOrigins := map[string]map[string]bool{
		"appview/internal/app/config_test.go":                         {"https://bsky.social": true},
		"appview/internal/auth/auth_request_admission_test.go":        {"https://auth.bsky.app": true, "https://bsky.social": true},
		"appview/internal/auth/store_test.go":                         {"https://auth.bsky.app": true, "https://bsky.social": true},
		"appview/internal/db/provider_registration_migration_test.go": {"https://auth.bsky.app": true, "https://bsky.social": true},
	}

	paths := []string{
		filepath.Join(repositoryRoot, "justfile"),
		filepath.Join(repositoryRoot, "scripts", "appview-check"),
	}
	for _, root := range []string{
		filepath.Join(repositoryRoot, "appview"),
		filepath.Join(repositoryRoot, "app", "test"),
		filepath.Join(repositoryRoot, ".github", "workflows"),
	} {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if os.IsNotExist(walkErr) && path == root {
					return fs.SkipDir
				}
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".dart") ||
				strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml") {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("inventory automated tests under %s: %v", root, err)
		}
	}

	for _, path := range paths {
		relative, err := filepath.Rel(repositoryRoot, path)
		if err != nil {
			t.Fatalf("make inventory path relative: %v", err)
		}
		relative = filepath.ToSlash(relative)
		if relative == "appview/internal/app/release_gate_inventory_test.go" {
			continue
		}
		values := sourceConstantStrings(t, path)
		for _, value := range values {
			if strings.Contains(value, accountCreationMethod) {
				t.Errorf("%s contains the live account-creation XRPC", relative)
			}
			for _, origin := range liveOriginPattern.FindAllString(value, -1) {
				if !allowedLiveOrigins[relative][origin] {
					t.Errorf("%s contains unreviewed live Bluesky origin %s", relative, origin)
				}
			}
		}
	}

	for path, origins := range allowedLiveOrigins {
		for origin := range origins {
			if !sourceContainsConstant(t, filepath.Join(repositoryRoot, filepath.FromSlash(path)), origin) {
				t.Errorf("reviewed inert origin %s is no longer owned by %s; update the semantic inventory", origin, path)
			}
		}
	}

	assertReleaseTestCommands(t, filepath.Join(repositoryRoot, "scripts", "appview-check"))
	assertFlutterTestRecipe(t, repositoryRoot)
	assertFederatedFixtureIsHermetic(t, filepath.Join(filepath.Dir(sourceFile), "federated_real_flow_fixture_test.go"))
}

func TestLocalReleaseAndVersionWiringIsPresent(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	assertFileContains(t, filepath.Join(repositoryRoot, "scripts", "release"),
		"create_release",
		"push_release",
		"git push --atomic",
		"appview/CHANGELOG.md",
		"app/CHANGELOG.md",
	)
	assertFileContains(t, filepath.Join(repositoryRoot, "justfile"),
		"release-create-appview VERSION NOTES=\"\":",
		"release-create-app VERSION NOTES=\"\":",
		"release-push STREAM TAG:",
		"appview-deploy TAG:",
	)
	assertFileContains(t, filepath.Join(repositoryRoot, "scripts", "appview-deploy"),
		"RENDER_API_KEY",
		"commitId",
		"tap.connected",
	)
	assertFileContains(t, filepath.Join(repositoryRoot, ".github", "workflows", "backend-ci.yml"),
		"pull_request:",
		"name: PR checks",
		"flutter analyze",
		"./scripts/appview-check",
	)
	assertFileContains(t, filepath.Join(repositoryRoot, "appview", "Dockerfile"),
		"internal/buildinfo.version",
		"grep -Eq '^(0|[1-9][0-9]*)",
		"adduser -S -s /bin/sh -G app app",
	)
}

func sourceConstantStrings(t *testing.T, path string) []string {
	t.Helper()
	if filepath.Ext(path) != ".go" {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		return []string{string(raw)}
	}
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	constants := make(map[string]string)
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.CONST {
			continue
		}
		for _, specification := range general.Specs {
			valueSpec := specification.(*ast.ValueSpec)
			for index, name := range valueSpec.Names {
				if index < len(valueSpec.Values) {
					if value, ok := constantString(valueSpec.Values[index], constants); ok {
						constants[name.Name] = value
					}
				}
			}
		}
	}
	var values []string
	ast.Inspect(file, func(node ast.Node) bool {
		expression, ok := node.(ast.Expr)
		if ok {
			if value, constant := constantString(expression, constants); constant {
				values = append(values, value)
			}
		}
		return true
	})
	return values
}

func constantString(expression ast.Expr, constants map[string]string) (string, bool) {
	switch expression := expression.(type) {
	case *ast.BasicLit:
		if expression.Kind != token.STRING {
			return "", false
		}
		value, err := strconv.Unquote(expression.Value)
		return value, err == nil
	case *ast.BinaryExpr:
		if expression.Op != token.ADD {
			return "", false
		}
		left, leftOK := constantString(expression.X, constants)
		right, rightOK := constantString(expression.Y, constants)
		return left + right, leftOK && rightOK
	case *ast.Ident:
		value, ok := constants[expression.Name]
		return value, ok
	case *ast.ParenExpr:
		return constantString(expression.X, constants)
	default:
		return "", false
	}
}

func sourceContainsConstant(t *testing.T, path, want string) bool {
	t.Helper()
	for _, value := range sourceConstantStrings(t, path) {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func assertFileContains(t *testing.T, path string, fragments ...string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for _, fragment := range fragments {
		if !strings.Contains(string(raw), fragment) {
			t.Errorf("%s does not contain required release inventory fragment %q", path, fragment)
		}
	}
}

func assertReleaseTestCommands(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read release script: %v", err)
	}
	var ordinary, race bool
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 || fields[0] != "go" || fields[1] != "test" {
			continue
		}
		flags := make(map[string]bool)
		for _, field := range fields[2:] {
			flags[field] = true
		}
		if !flags["-p"] || !flags["1"] || !flags["-count=1"] || !flags["-json"] || !flags["./..."] {
			continue
		}
		if flags["-race"] {
			race = true
		} else {
			ordinary = true
		}
	}
	if !ordinary || !race {
		t.Fatalf("release script must run serial uncached full tests with and without race; ordinary=%t race=%t", ordinary, race)
	}
}

func assertFlutterTestRecipe(t *testing.T, repositoryRoot string) {
	t.Helper()
	justPath, err := exec.LookPath("just")
	if err != nil {
		t.Log("just is unavailable; the release gate owns semantic recipe validation")
		return
	}
	command := exec.Command(justPath, "--justfile", filepath.Join(repositoryRoot, "justfile"), "--dump", "--dump-format", "json")
	command.Dir = repositoryRoot
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("load just recipes: %v: %s", err, output)
	}
	var dump struct {
		Recipes map[string]struct {
			Body       [][]any `json:"body"`
			Parameters []struct {
				Name string `json:"name"`
				Kind string `json:"kind"`
			} `json:"parameters"`
		} `json:"recipes"`
	}
	if err := json.Unmarshal(output, &dump); err != nil {
		t.Fatalf("decode just recipes: %v", err)
	}
	recipe, ok := dump.Recipes["app-test"]
	if !ok {
		t.Fatal("justfile is missing app-test recipe")
	}
	if len(recipe.Parameters) != 1 || recipe.Parameters[0].Name != "ARGS" || recipe.Parameters[0].Kind != "star" {
		t.Fatalf("app-test parameters = %+v, want variadic ARGS", recipe.Parameters)
	}
	encodedBody, _ := json.Marshal(recipe.Body)
	if !strings.Contains(string(encodedBody), "flutter test") || !strings.Contains(string(encodedBody), `"ARGS"`) {
		t.Fatalf("app-test recipe does not forward ARGS to flutter test: %s", encodedBody)
	}
}

func assertFederatedFixtureIsHermetic(t *testing.T, path string) {
	t.Helper()
	for _, origin := range []string{
		"https://pds.real-flow.test",
		"https://pds-second.real-flow.test",
		"https://auth.real-flow.test",
	} {
		if !sourceContainsConstant(t, path, origin) {
			t.Errorf("federated fixture is missing controlled origin %s", origin)
		}
	}
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse federated fixture: %v", err)
	}
	var rejectsUnknownHosts bool
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "New" {
			return true
		}
		if value, ok := constantString(call.Args[0], nil); ok && value == "unknown test hostname" {
			rejectsUnknownHosts = true
		}
		return true
	})
	if !rejectsUnknownHosts {
		t.Fatal("federated fixture resolver no longer rejects unknown test hostnames")
	}
}
