package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

func serveScheduledPostRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	body string,
	owner syntax.DID,
) *httptest.ResponseRecorder {
	t.Helper()
	request := scheduledPostRequest(method, path, body, owner)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func serveScheduledPostPathRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	id string,
	body string,
	owner syntax.DID,
) *httptest.ResponseRecorder {
	t.Helper()
	request := scheduledPostRequest(method, "/v1/scheduled-posts/"+id, body, owner)
	request.SetPathValue("id", id)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func scheduledPostRequest(method, path, body string, owner syntax.DID) *http.Request {
	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}
	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Content-Type", "application/json")
	return request.WithContext(middleware.WithDID(request.Context(), owner))
}

func decodeResponseJSON[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(recorder.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode response JSON: %v; body=%q", err, recorder.Body.String())
	}
	return value
}

func assertScheduledPostError(t *testing.T, recorder *httptest.ResponseRecorder, code string) {
	t.Helper()
	body := decodeResponseJSON[envelope.Error](t, recorder)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &fields); err != nil {
		t.Fatalf("decode error envelope fields: %v", err)
	}
	for _, key := range []string{"error", "message", "requestId"} {
		if _, ok := fields[key]; !ok {
			t.Fatalf("error envelope missing %q: %s", key, recorder.Body.String())
		}
	}
	if body.Error != code || body.Message == "" {
		t.Fatalf("error response=%+v, want code %q and standard envelope", body, code)
	}
}

func TestScheduledPostRequestPreservesBody(t *testing.T) {
	t.Parallel()
	const body = `{"payload":{"text":"missing sponsored"}}`
	request := scheduledPostRequest(http.MethodPost, "/v1/scheduled-posts", body, "did:plc:alice")
	got, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	if string(got) != body {
		t.Fatalf("request body = %q, want exact fixture %q", got, body)
	}
}
