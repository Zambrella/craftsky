package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBodyLimitDefaultJSONRejectsOversizedBeforeHandler(t *testing.T) {
	const limit = int64(1024 * 1024)
	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	})
	handler := BodyLimit(BodyLimitConfig{DefaultJSONBytes: limit}, BodyDefaultJSON, nil)(next)

	req := httptest.NewRequest(http.MethodPost, "/v1/posts", strings.NewReader(strings.Repeat("a", int(limit)+1)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if handlerCalled {
		t.Fatal("handler was called for oversized body")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "request_body_too_large") || !strings.Contains(rec.Body.String(), "request body exceeds the configured limit") {
		t.Fatalf("body = %q, want request_body_too_large envelope", rec.Body.String())
	}
}

func TestBodyLimitDefaultJSONAllowsAtLimit(t *testing.T) {
	const limit = int64(1024 * 1024)
	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		if int64(len(body)) != limit {
			t.Fatalf("body len = %d, want %d", len(body), limit)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	handler := BodyLimit(BodyLimitConfig{DefaultJSONBytes: limit}, BodyDefaultJSON, nil)(next)

	req := httptest.NewRequest(http.MethodPost, "/v1/posts", strings.NewReader(strings.Repeat("a", int(limit))))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Fatal("handler was not called for body at limit")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", rec.Code, rec.Body.String())
	}
}

func TestBodyLimitUploadUsesUploadOverride(t *testing.T) {
	const defaultLimit = int64(10)
	const uploadLimit = int64(20)

	t.Run("allows body over default but within upload override", func(t *testing.T) {
		handlerCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			if len(body) != 15 {
				t.Fatalf("body len = %d, want 15", len(body))
			}
			w.WriteHeader(http.StatusNoContent)
		})
		handler := BodyLimit(BodyLimitConfig{DefaultJSONBytes: defaultLimit, UploadBytes: uploadLimit}, BodyUpload, nil)(next)

		req := httptest.NewRequest(http.MethodPost, "/v1/blobs/images", strings.NewReader(strings.Repeat("a", 15)))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if !handlerCalled {
			t.Fatal("handler was not called for upload body within override")
		}
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204; body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("rejects body over upload override", func(t *testing.T) {
		handlerCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})
		handler := BodyLimit(BodyLimitConfig{DefaultJSONBytes: defaultLimit, UploadBytes: uploadLimit}, BodyUpload, nil)(next)

		req := httptest.NewRequest(http.MethodPost, "/v1/blobs/images", strings.NewReader(strings.Repeat("a", 21)))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if handlerCalled {
			t.Fatal("handler was called for upload body over override")
		}
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want 413; body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "request_body_too_large") {
			t.Fatalf("body = %q, want request_body_too_large", rec.Body.String())
		}
	})
}

func TestBodyLimitNoBodyRejectsNonEmptyBodies(t *testing.T) {
	for _, tc := range []struct {
		name       string
		body       string
		wantStatus int
		wantCalled bool
	}{
		{name: "absent", body: "", wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "empty whitespace", body: "   ", wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "non-empty", body: "{}", wantStatus: http.StatusBadRequest, wantCalled: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handlerCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusNoContent)
			})
			handler := BodyLimit(BodyLimitConfig{DefaultJSONBytes: 10}, BodyNoBody, nil)(next)

			var body io.Reader
			if tc.name != "absent" {
				body = strings.NewReader(tc.body)
			}
			req := httptest.NewRequest(http.MethodGet, "/v1/whoami", body)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if handlerCalled != tc.wantCalled {
				t.Fatalf("handlerCalled = %v, want %v", handlerCalled, tc.wantCalled)
			}
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if !tc.wantCalled && !strings.Contains(rec.Body.String(), "request_body_not_allowed") {
				t.Fatalf("body = %q, want request_body_not_allowed", rec.Body.String())
			}
		})
	}
}
