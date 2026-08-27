package followergrowth

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSnapshotsAreAbsentFromOrdinaryRetention(t *testing.T) {
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := entry.Name()
		if entry.IsDir() || !strings.Contains(name, "retention") || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lower := strings.ToLower(string(contents))
		if strings.Contains(lower, "follower_growth_snapshots") || strings.Contains(lower, "followergrowth") {
			t.Errorf("ordinary retention source %s references follower growth snapshots", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan retention sources: %v", err)
	}
}
