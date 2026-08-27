package index

import (
	"os"
	"strings"
	"testing"
)

func TestIndexersDoNotWriteFollowerGrowthAnalytics(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read index package: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		contents, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		lower := strings.ToLower(string(contents))
		if strings.Contains(lower, "followergrowth") || strings.Contains(lower, "follower_growth") {
			t.Errorf("%s crosses the follower-growth snapshot boundary", name)
		}
	}
}
