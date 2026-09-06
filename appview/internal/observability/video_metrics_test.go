package observability

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestVideoMetricsUseOnlyBoundedLabels(t *testing.T) {
	t.Parallel()
	recorder := NewInMemoryMetricRecorder()
	observer := New(Config{MetricRecorder: recorder})
	observer.ObserveVideoOperation(context.Background(), "did:plc:alice/job-secret", "https://video.example/cid-secret", "token-secret", time.Second)
	observer.ObserveVideoOperation(context.Background(), "verification", "rejected", "owner_mismatch", 2*time.Second)

	captured := fmt.Sprint(recorder.Calls())
	for _, secret := range []string{"did:plc:alice", "job-secret", "video.example", "cid-secret", "token-secret"} {
		if strings.Contains(captured, secret) {
			t.Fatalf("metrics leaked %q: %s", secret, captured)
		}
	}
	if !strings.Contains(captured, "craftsky_appview_video_operations_total") || !strings.Contains(captured, "verification") || !strings.Contains(captured, "rejected") || !strings.Contains(captured, "owner_mismatch") {
		t.Fatalf("bounded video metrics missing: %s", captured)
	}
}
