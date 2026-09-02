package business_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/observability"
)

func TestBusinessObservabilityExcludesAuthoredValuesAndUsesBoundedAttributes(t *testing.T) {
	canaries := []string{
		"https://uri-canary.example/private?order=alpha#detail",
		"mailto:email-canary@example.com",
		"free-text-canary-hours-and-service-area",
		"title-canary-hand-dyed-yarn",
		"918273.45",
		"location-canary-locality",
	}
	owner := syntax.DID("did:plc:business-observability-canary")
	profileURI := syntax.ATURI("at://" + owner.String() + "/social.craftsky.business.profile/self")
	eventURI := syntax.ATURI("at://" + owner.String() + "/social.craftsky.business.event/3msprivacy001")

	profileRaw, err := json.Marshal(map[string]any{
		"$type":       "social.craftsky.business.profile",
		"hoursNote":   canaries[2],
		"serviceArea": canaries[2],
		"location":    map[string]any{"country": "GB", "locality": canaries[5]},
		"primaryAction": map[string]any{
			"type": "email", "destination": canaries[1],
		},
		"products": []map[string]any{{
			"title": canaries[3], "uri": canaries[0],
			"price": map[string]any{"amount": canaries[4], "currency": "USD"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := business.HydrateProfile(profileRaw); err != nil {
		t.Fatalf("hydrate profile: %v", err)
	}
	if err := business.ValidateWebDestination(canaries[0]); err != nil {
		t.Fatalf("validate web destination: %v", err)
	}
	if err := business.ValidateMailtoDestination(canaries[1]); err != nil {
		t.Fatalf("validate mail destination: %v", err)
	}

	var logs bytes.Buffer
	metrics := observability.NewInMemoryMetricRecorder()
	observer := observability.New(observability.Config{
		Env: "test", TracingEnabled: true, LogsEnabled: true, MetricRecorder: metrics,
		Logger: slog.New(slog.NewJSONHandler(&logs, nil)),
	})

	traceCtx, root := observer.StartSpan(context.Background(), observability.SpanContext{
		Operation: "business.security", Component: "business",
	})
	observer.Log(traceCtx, slog.LevelInfo, "business record processed", observability.EventContext{
		"component": "business", "operation": "business.hydrate", "result": "success",
		"destination": canaries[0], "email": canaries[1], "title": canaries[3],
		"price": canaries[4], "location": canaries[5], "did": owner, "uri": eventURI,
	})
	observer.ObserveIndexerHandled("social.craftsky.business.profile", nil, time.Millisecond)
	observer.ObserveIndexerHandled("social.craftsky.business.event", nil, time.Millisecond)

	wrapped := observer.WrapPDSFactory(func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
		return businessTelemetryPDS{}, nil
	})
	client, err := wrapped(traceCtx, owner, "session-canary")
	if err != nil {
		t.Fatalf("wrap PDS client: %v", err)
	}
	var record map[string]any
	if _, err := client.GetRecord(traceCtx, owner, "social.craftsky.business.profile", "self", &record); err != nil {
		t.Fatal(err)
	}
	if err := client.PutRecord(traceCtx, owner, "social.craftsky.business.profile", "self", json.RawMessage(profileRaw)); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteRecord(traceCtx, owner, "social.craftsky.business.profile", "self"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetRecord(traceCtx, owner, "social.craftsky.business.event", "3msprivacy001", &record); err != nil {
		t.Fatal(err)
	}
	if err := client.PutRecord(traceCtx, owner, "social.craftsky.business.event", "3msprivacy001", map[string]any{"name": canaries[3], "eventUri": canaries[0]}); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteRecord(traceCtx, owner, "social.craftsky.business.event", "3msprivacy001"); err != nil {
		t.Fatal(err)
	}
	root.Finish("success")
	traceAttributes := observability.SanitizeEventContext(observability.EventContext{
		"component": "business", "operation": "business.hydrate", "result": "success",
		"destination": canaries[0], "email": canaries[1], "title": canaries[3],
		"price": canaries[4], "location": canaries[5], "did": owner, "uri": eventURI,
	})

	calls := metrics.Calls()
	for _, call := range calls {
		if err := observability.ValidateMetricCall(call); err != nil {
			t.Fatalf("metric call failed validation: %v; call=%#v", err, call)
		}
	}
	for _, want := range []struct{ key, value string }{
		{"nsid", "social.craftsky.business.profile"},
		{"nsid", "social.craftsky.business.event"},
		{"operation", "business.profile.get"},
		{"operation", "business.profile.put"},
		{"operation", "business.profile.delete"},
		{"operation", "business.event.get"},
		{"operation", "business.event.put"},
		{"operation", "business.event.delete"},
	} {
		if !businessMetricAttributeExists(calls, want.key, want.value) {
			t.Errorf("metrics missing bounded %s=%q: %#v", want.key, want.value, calls)
		}
	}

	telemetry, err := json.Marshal(struct {
		Logs            string
		Metrics         []observability.MetricCall
		TraceAttributes observability.EventContext
	}{logs.String(), calls, traceAttributes})
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range append(canaries, owner.String(), profileURI.String(), eventURI.String()) {
		if strings.Contains(string(telemetry), forbidden) {
			t.Fatalf("telemetry contains prohibited business value %q: %s", forbidden, telemetry)
		}
	}
}

type businessTelemetryPDS struct{}

func (businessTelemetryPDS) GetRecord(context.Context, syntax.DID, string, string, any) (string, error) {
	return "bafy-safe-cid", nil
}

func (businessTelemetryPDS) PutRecord(context.Context, syntax.DID, string, string, any) error {
	return nil
}

func (businessTelemetryPDS) CreateRecord(context.Context, syntax.DID, string, any) (syntax.ATURI, syntax.CID, error) {
	return "", "", nil
}

func (businessTelemetryPDS) DeleteRecord(context.Context, syntax.DID, string, string) error {
	return nil
}

func (businessTelemetryPDS) UploadBlob(context.Context, string, []byte) (*auth.UploadedBlob, error) {
	return &auth.UploadedBlob{}, nil
}

func businessMetricAttributeExists(calls []observability.MetricCall, key, value string) bool {
	for _, call := range calls {
		if call.Attributes[key] == value {
			return true
		}
	}
	return false
}
