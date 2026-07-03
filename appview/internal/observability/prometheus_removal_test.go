package observability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrometheusRemoval(t *testing.T) {
	root := moduleRoot(t)
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
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "internal/observability/prometheus_removal_test.go" {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(body)
		for _, forbidden := range []string{
			"github.com/prometheus/",
			"prometheus.",
			"promhttp.",
			"MetricsHandler",
		} {
			if strings.Contains(text, forbidden) {
				violations = append(violations, rel+" contains "+forbidden)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan repository: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("Prometheus runtime references remain:\n%s", strings.Join(violations, "\n"))
	}
}
