package envelope_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"social.craftsky/appview/internal/api/envelope"
)

func TestWriteError_WritesCanonicalShape(t *testing.T) {
	rec := httptest.NewRecorder()
	envelope.WriteError(rec, 422, "validation_failed", "bad input", "req-123", nil)

	if rec.Code != 422 {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	if body["error"] != "validation_failed" {
		t.Errorf("error = %v, want validation_failed", body["error"])
	}
	if body["message"] != "bad input" {
		t.Errorf("message = %v, want bad input", body["message"])
	}
	if body["requestId"] != "req-123" {
		t.Errorf("requestId = %v, want req-123", body["requestId"])
	}
	if _, ok := body["fields"]; ok {
		t.Errorf("fields should be omitted when nil; got %v", body["fields"])
	}
}

func TestWriteError_IncludesFieldsWhenProvided(t *testing.T) {
	rec := httptest.NewRecorder()
	envelope.WriteError(rec, 422, "validation_failed", "bad input", "req-1", map[string]string{
		"text":     "exceeds max length",
		"material": "unknown code",
	})

	var body struct {
		Fields map[string]string `json:"fields"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	if body.Fields["text"] != "exceeds max length" {
		t.Errorf("fields.text = %q", body.Fields["text"])
	}
	if body.Fields["material"] != "unknown code" {
		t.Errorf("fields.material = %q", body.Fields["material"])
	}
}
