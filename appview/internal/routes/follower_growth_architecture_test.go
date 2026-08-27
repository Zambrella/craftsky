package routes

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestFollowerGrowthProductionDependenciesStayPrivateAndNonInfluential(t *testing.T) {
	t.Parallel()

	allowed := []string{
		"api/follower_growth.go",
		"app/deps.go",
		"app/deps_content.go",
		"app/routes_adapter.go",
		"followergrowth/period.go",
		"followergrowth/series.go",
		"followergrowth/store.go",
		"followergrowth/worker.go",
		"observability/follower_growth.go",
		"observability/metric_recorder.go",
		"ownerlifecycle/terminal_inventory.go",
		"ownerlifecycle/terminal_purge_processor.go",
		"routes/dependencies.go",
		"routes/routes.go",
		"routes/routes_profile_notification.go",
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
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source := strings.ToLower(string(raw))
		referencesGrowth := strings.Contains(source, "followergrowth") || strings.Contains(source, "follower_growth")
		if referencesGrowth && !slices.Contains(allowed, relative) {
			t.Errorf("unapproved follower-growth production dependency in %s", relative)
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
