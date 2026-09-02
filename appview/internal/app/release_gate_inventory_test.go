package app

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestReleaseGatesUseFixturesInsteadOfLiveBluesky(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	liveOriginPattern := regexp.MustCompile(`https://(?:bsky\.social|auth\.bsky\.app|api\.bsky\.app|public\.api\.bsky\.app)`)
	accountCreationMethod := "com.atproto.server." + "createAccount"

	wantLiveOriginInventory := map[string]int{
		"appview/internal/app/config_test.go|https://bsky.social":                           1,
		"appview/internal/auth/auth_request_admission_test.go|https://auth.bsky.app":        2,
		"appview/internal/auth/auth_request_admission_test.go|https://bsky.social":          2,
		"appview/internal/auth/store_test.go|https://auth.bsky.app":                         3,
		"appview/internal/auth/store_test.go|https://bsky.social":                           10,
		"appview/internal/db/provider_registration_migration_test.go|https://auth.bsky.app": 4,
		"appview/internal/db/provider_registration_migration_test.go|https://bsky.social":   4,
	}
	gotLiveOriginInventory := make(map[string]int)

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
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read release inventory source %s: %v", path, err)
		}
		relative, err := filepath.Rel(repositoryRoot, path)
		if err != nil {
			t.Fatalf("make inventory path relative: %v", err)
		}
		relative = filepath.ToSlash(relative)
		if relative == "appview/internal/app/release_gate_inventory_test.go" {
			continue
		}
		if strings.Contains(string(raw), accountCreationMethod) {
			t.Errorf("%s invokes the live account-creation XRPC", relative)
		}
		for _, origin := range liveOriginPattern.FindAllString(string(raw), -1) {
			gotLiveOriginInventory[relative+"|"+origin]++
		}
	}

	if diff := inventoryDiff(wantLiveOriginInventory, gotLiveOriginInventory); diff != "" {
		t.Fatalf("live Bluesky test-origin inventory changed; only reviewed inert config/persistence values are allowed:\n%s", diff)
	}

	assertFileContains(t, filepath.Join(repositoryRoot, "scripts", "appview-check"),
		"go test -p 1 -count=1 -json ./...",
		"go test -p 1 -count=1 -race -json ./...",
	)
	assertFileContains(t, filepath.Join(repositoryRoot, "justfile"),
		"app-test *ARGS:",
		"flutter test {{ARGS}}",
	)
	assertFileContains(t, "federated_real_flow_integration_test.go",
		`realFlowPDSOrigin       = "https://pds.real-flow.test"`,
		`realFlowSecondPDSOrigin = "https://pds-second.real-flow.test"`,
		`realFlowAuthOrigin      = "https://auth.real-flow.test"`,
		`return nil, errors.New("unknown test hostname")`,
	)
}

func inventoryDiff(want, got map[string]int) string {
	keys := make(map[string]struct{}, len(want)+len(got))
	for key := range want {
		keys[key] = struct{}{}
	}
	for key := range got {
		keys[key] = struct{}{}
	}
	ordered := make([]string, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Strings(ordered)

	var differences []string
	for _, key := range ordered {
		if want[key] != got[key] {
			differences = append(differences, key+": want "+strconv.Itoa(want[key])+", got "+strconv.Itoa(got[key]))
		}
	}
	return strings.Join(differences, "\n")
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
