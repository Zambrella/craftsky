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
		{name: "empty whitespace", body: "   ", wantStatus: http.StatusBadRequest, wantCalled: false},
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

func TestBodyLimitUnknownLengthStreamsThroughMaxBytesReader(t *testing.T) {
	t.Parallel()

	const limit = int64(8)
	probe := &countingBodyReader{reader: strings.NewReader("1234567890123456")}
	handlerCalled := false
	var requestClosed bool
	handler := BodyLimit(BodyLimitConfig{DefaultJSONBytes: limit}, BodyDefaultJSON, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		_, err := io.ReadAll(r.Body)
		if err == nil {
			t.Fatal("ReadAll() error = nil, want MaxBytesError")
		}
		http.Error(w, "handler-mapped-malformed-body", http.StatusBadRequest)
		requestClosed = r.Close
	}))
	req := httptest.NewRequest(http.MethodPost, "/v1/test", probe)
	req.ContentLength = -1
	req.TransferEncoding = []string{"chunked"}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if !handlerCalled {
		t.Fatal("handler was not called for unknown-length streaming body")
	}
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413; body=%s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "handler-mapped") {
		t.Fatalf("body = %q, handler response was not suppressed", recorder.Body.String())
	}
	if !requestClosed || recorder.Header().Get("Connection") != "close" {
		t.Fatalf("boundary rejection did not disable connection reuse: request.Close=%v headers=%v", requestClosed, recorder.Header())
	}
	if probe.bytesRead > limit+1 {
		t.Fatalf("underlying bytes read = %d, want at most %d", probe.bytesRead, limit+1)
	}
}

func TestBodyPrecheckDoesNotReadUnknownLength(t *testing.T) {
	t.Parallel()

	probe := &countingBodyReader{reader: strings.NewReader("123456789")}
	handler := BodyPrecheck(BodyLimitConfig{DefaultJSONBytes: 8}, BodyDefaultJSON, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "/v1/test", probe)
	req.ContentLength = -1
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
	if probe.bytesRead != 0 {
		t.Fatalf("bytes read = %d, want 0", probe.bytesRead)
	}
}

func TestBodyPrecheckUnsupportedMediaDetachesUnreadBody(t *testing.T) {
	t.Parallel()

	probe := &countingBodyReader{reader: strings.NewReader("plain text")}
	handler := BodyPrecheck(BodyLimitConfig{DefaultJSONBytes: 1024}, BodyDefaultJSON, nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler called for unsupported media type")
	}))
	request := httptest.NewRequest(http.MethodPost, "/v1/posts", probe)
	request.Header.Set("Content-Type", "text/plain")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want 415; body=%s", recorder.Code, recorder.Body.String())
	}
	if probe.bytesRead != 0 {
		t.Fatalf("body bytes read = %d, want 0", probe.bytesRead)
	}
	if request.Body != http.NoBody || !request.Close || recorder.Header().Get("Connection") != "close" {
		t.Fatalf("unread body was not detached: body=%T request.Close=%t headers=%v", request.Body, request.Close, recorder.Header())
	}
}

type countingBodyReader struct {
	reader    io.Reader
	bytesRead int64
}

func (reader *countingBodyReader) Read(buffer []byte) (int, error) {
	read, err := reader.reader.Read(buffer)
	reader.bytesRead += int64(read)
	return read, err
}
