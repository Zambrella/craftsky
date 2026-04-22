package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeviceID_AcceptsValidHeaderAndInjectsCtx(t *testing.T) {
	var seen string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := GetDeviceID(r.Context())
		seen = id
		w.WriteHeader(http.StatusOK)
	})
	h := DeviceID(nil, discardLogger())(next)

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("X-Craftsky-Device-Id", "2c3f6a1e-0b4d-4cf5-9aa1-f0b4a9c9e1b3")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if seen != "2c3f6a1e-0b4d-4cf5-9aa1-f0b4a9c9e1b3" {
		t.Errorf("ctx device id = %q, want the sent value", seen)
	}
}

func TestDeviceID_MissingHeaderReturns400Envelope(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not run")
	})
	h := DeviceID(nil, discardLogger())(next)

	req := httptest.NewRequest("GET", "/x", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not json: %v", err)
	}
	if body["error"] != "missing_device_id" {
		t.Errorf("error = %v, want missing_device_id", body["error"])
	}
}

func TestDeviceID_EmptyHeaderReturns400(t *testing.T) {
	h := DeviceID(nil, discardLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not run")
	}))

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("X-Craftsky-Device-Id", "")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestDeviceID_TooLongHeaderReturns400(t *testing.T) {
	h := DeviceID(nil, discardLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not run")
	}))

	long := make([]byte, 257)
	for i := range long {
		long[i] = 'a'
	}
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("X-Craftsky-Device-Id", string(long))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

type fakeToucher struct {
	calls []struct{ did, sid, devID string }
}

func (f *fakeToucher) TouchDeviceID(ctx context.Context, did, sid, devID string) error {
	f.calls = append(f.calls, struct{ did, sid, devID string }{did, sid, devID})
	return nil
}

func TestDeviceID_FiresTouchWhenSessionInCtx(t *testing.T) {
	tt := &fakeToucher{}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := DeviceID(tt, discardLogger())(next)

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("X-Craftsky-Device-Id", "dev-abc")
	ctx := req.Context()
	ctx = WithDID(ctx, "did:plc:alice")
	ctx = WithOAuthSessionID(ctx, "sess-1")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if len(tt.calls) != 1 {
		t.Fatalf("TouchDeviceID called %d times, want 1", len(tt.calls))
	}
	got := tt.calls[0]
	if got.did != "did:plc:alice" || got.sid != "sess-1" || got.devID != "dev-abc" {
		t.Errorf("call = %+v, want did=alice sid=sess-1 devID=abc", got)
	}
}

func TestDeviceID_SkipsTouchWhenNoSession(t *testing.T) {
	tt := &fakeToucher{}
	h := DeviceID(tt, discardLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("X-Craftsky-Device-Id", "dev-abc")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if len(tt.calls) != 0 {
		t.Errorf("TouchDeviceID called %d times on unauthed request, want 0", len(tt.calls))
	}
}
