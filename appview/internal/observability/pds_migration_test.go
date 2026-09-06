package observability

import (
	"context"
	"testing"
	"time"
)

func TestMigrationMetricsAreBoundedAndSecretFree(t *testing.T) {
	recorder := NewInMemoryMetricRecorder()
	observer := New(Config{MetricRecorder: recorder})

	observer.ObserveAuthorityVerification("session_select", "mismatch", "issuer_changed", 25*time.Millisecond)
	observer.ObserveOAuthMetadataCache(context.Background(), "protected_resource", "hit")
	observer.ObserveOAuthMetadataCache(context.Background(), "protected_resource", "miss")
	observer.ObserveOAuthMetadataCache(context.Background(), "authorization_server", "coalesced")
	observer.ObserveOAuthMetadataCache(context.Background(), "authorization_server", "error")
	observer.ObserveOAuthCleanup(context.Background(), "parent", "retry", "dependency_unavailable", time.Second, 2)
	observer.ObserveRepositoryRepair("pds_reconcile", "retry", "signature_invalid", 2*time.Second, 3)
	observer.ObserveRepositoryRepairQueue(7, 16*time.Minute, 5, true)
	observer.ObserveRepositorySnapshotVerification("error", "signature_invalid", 150*time.Millisecond)

	calls := recorder.Calls()
	if len(calls) != 18 {
		t.Fatalf("metric call count = %d, want 18", len(calls))
	}
	for _, call := range calls {
		if err := ValidateMetricCall(call); err != nil {
			t.Fatalf("invalid migration metric %#v: %v", call, err)
		}
	}
	assertMetricAttribute(t, calls, "craftsky_appview_authority_verifications_total", "result", "mismatch")
	assertMetricAttribute(t, calls, "craftsky_appview_oauth_metadata_cache_requests_total", "stage", "protected_resource")
	for _, result := range []string{"hit", "miss", "coalesced", "error"} {
		assertMetricAttribute(t, calls, "craftsky_appview_oauth_metadata_cache_requests_total", "result", result)
	}
	assertMetricAttribute(t, calls, "craftsky_appview_oauth_cleanups_total", "reason", "dependency_unavailable")
	assertMetricAttribute(t, calls, "craftsky_appview_repository_repairs_total", "reason", "signature_invalid")
	assertMetricAttribute(t, calls, "craftsky_appview_repository_repair_alert", "alert", "true")
	assertMetricAttribute(t, calls, "craftsky_appview_repository_snapshot_verifications_total", "result", "error")
}

func TestMigrationMetricsReplaceUnboundedValues(t *testing.T) {
	recorder := NewInMemoryMetricRecorder()
	observer := New(Config{MetricRecorder: recorder})
	secret := "Bearer oauth-token did:plc:private"

	observer.ObserveAuthorityVerification(secret, secret, secret, 0)
	observer.ObserveOAuthMetadataCache(context.Background(), secret, secret)
	observer.ObserveOAuthCleanup(context.Background(), secret, secret, secret, 0, -1)
	observer.ObserveRepositoryRepair(secret, secret, secret, 0, -1)
	observer.ObserveRepositorySnapshotVerification(secret, secret, 0)

	for _, call := range recorder.Calls() {
		if err := ValidateMetricCall(call); err != nil {
			t.Fatalf("unbounded migration value escaped sanitization in %#v: %v", call, err)
		}
		for _, value := range call.Attributes {
			if value != "unknown" {
				t.Fatalf("attribute value = %q, want bounded unknown", value)
			}
		}
	}
}

func assertMetricAttribute(t *testing.T, calls []MetricCall, name, key, want string) {
	t.Helper()
	for _, call := range calls {
		if call.Name == name && call.Attributes[key] == want {
			return
		}
	}
	t.Fatalf("metric %q missing %s=%q in %#v", name, key, want, calls)
}
