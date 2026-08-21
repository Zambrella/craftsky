package api_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestProfilePinObservabilityIsBoundedAndRedacted(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000035_profile_pins.up.sql")
	if err != nil {
		t.Fatalf("read profile pin migration: %v", err)
	}
	pool := testdb.WithSchema(t, profilePinStoreTestDDL+string(migration))
	recorder := observability.NewInMemoryMetricRecorder()
	observer := observability.New(observability.Config{MetricRecorder: recorder})
	store := api.NewProfilePinStore(pool, api.ProfilePinStoreOptions{Observer: observer})
	ctx := ownerlifecycle.WithExpectedGeneration(context.Background(), 1)
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	uriB := syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/standard-b")

	if _, err := store.Pin(ctx, alice, alice, syntax.RecordKey("standard-a")); err != nil {
		t.Fatalf("pin: %v", err)
	}
	if _, err := store.Pin(ctx, alice, alice, syntax.RecordKey("standard-b")); err != nil {
		t.Fatalf("replace: %v", err)
	}
	if _, err := store.Pin(ctx, alice, alice, syntax.RecordKey("standard-b")); err != nil {
		t.Fatalf("same-target pin: %v", err)
	}
	if _, err := store.Unpin(ctx, alice, uriB); err != nil {
		t.Fatalf("unpin: %v", err)
	}
	if _, err := store.Unpin(ctx, alice, uriB); err != nil {
		t.Fatalf("stale unpin: %v", err)
	}
	if _, err := store.Pin(ctx, alice, bob, syntax.RecordKey("other")); err == nil {
		t.Fatal("cross-owner pin unexpectedly succeeded")
	}
	if _, err := pool.Exec(ctx, `DROP TABLE profile_pins`); err != nil {
		t.Fatalf("drop profile_pins for internal error: %v", err)
	}
	if _, err := store.Pin(ctx, alice, alice, syntax.RecordKey("standard-a")); err == nil {
		t.Fatal("pin with missing table unexpectedly succeeded")
	}

	var calls []observability.MetricCall
	for _, call := range recorder.Calls() {
		if call.Name == "craftsky_appview_profile_pin_operation_duration_seconds" {
			calls = append(calls, call)
		}
	}
	if len(calls) != 7 {
		t.Fatalf("profile pin metric calls = %d, want 7: %#v", len(calls), calls)
	}
	allowed := map[string]map[string]bool{
		"operation":   {"pin": true, "replace": true, "unpin": true},
		"slot":        {"standard": true, "project": true, "unknown": true},
		"result":      {"success": true, "noop": true, "rejected": true, "error": true},
		"error_class": {"none": true, "forbidden": true, "not_found": true, "policy": true, "store": true},
	}
	for _, call := range calls {
		if err := observability.ValidateMetricCall(call); err != nil {
			t.Fatalf("invalid metric call: %v; call=%#v", err, call)
		}
		if len(call.Attributes) != len(allowed) {
			t.Fatalf("unexpected metric dimensions: %#v", call.Attributes)
		}
		for key, values := range allowed {
			if !values[call.Attributes[key]] {
				t.Fatalf("unbounded %s=%q in %#v", key, call.Attributes[key], call)
			}
		}
	}
	raw, err := json.Marshal(calls)
	if err != nil {
		t.Fatalf("marshal calls: %v", err)
	}
	diagnostics := string(raw)
	for _, forbidden := range []string{
		"did:plc:",
		"at://",
		"standard-a",
		"standard-b",
		"state_token",
		"created_at",
		"updated_at",
	} {
		if strings.Contains(diagnostics, forbidden) {
			t.Fatalf("telemetry leaked %q: %s", forbidden, diagnostics)
		}
	}
}
