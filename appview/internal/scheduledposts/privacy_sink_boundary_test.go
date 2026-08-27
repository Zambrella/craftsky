package scheduledposts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Scheduled worker lifecycles receive only OperationalObserver. Their concrete
// observer methods emit bounded metrics; logs, traces, and Sentry events remain
// impossible unless this package gains a direct sink dependency.
func TestIR017ScheduledWorkerTelemetryIsMetricsOnlyByConstruction(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	forbidden := []string{
		"github.com/getsentry/sentry-go",
		`"log/slog"`,
		".CaptureError(",
		".StartSpan(",
		".Log(",
		".AddBreadcrumb(",
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatal(err)
		}
		for _, symbol := range forbidden {
			if strings.Contains(string(body), symbol) {
				t.Fatalf("scheduled worker production file %s gained forbidden telemetry sink %q", name, symbol)
			}
		}
	}
}
