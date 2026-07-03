package observability

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestValidateMetricCallRejectsForbiddenAndHighCardinalityAttributes(t *testing.T) {
	call := MetricCall{
		Name:  "craftsky_appview_http_requests_total",
		Kind:  MetricKindCounter,
		Value: 1,
		Attributes: map[string]string{
			"method":        "GET",
			"route_pattern": "/v1/posts/did:plc:raw?cursor=secret",
			"run_id":        "run-123",
			"token":         "secret-token",
			"email":         "alice@example.com",
		},
	}

	err := ValidateMetricCall(call)
	if err == nil {
		t.Fatal("ValidateMetricCall returned nil, want validation error")
	}
	msg := err.Error()
	for _, want := range []string{"route_pattern", "run_id", "token", "email"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("validation error %q missing %q", msg, want)
		}
	}
}

func TestMetricRecorderNormalizesUnsafeRuntimeAttributes(t *testing.T) {
	recorder := NewInMemoryMetricRecorder()

	recorder.HTTPRequestFinished(context.Background(), "TRACE", "/v1/posts/did:plc:raw?cursor=secret", 777, time.Millisecond, 12)
	recorder.DBOperation(context.Background(), "SELECT * FROM users WHERE did='did:plc:raw'", "/v1/search/posts?q=secret", "did:plc:raw", time.Millisecond)

	for _, call := range recorder.Calls() {
		if err := ValidateMetricCall(call); err != nil {
			t.Fatalf("normalized metric call failed validation: %v; call=%#v", err, call)
		}
		for _, value := range call.Attributes {
			if strings.Contains(value, "did:") || strings.Contains(value, "secret") || strings.Contains(value, "SELECT") {
				t.Fatalf("metric call retained unsafe runtime value: %#v", call)
			}
		}
	}
}
