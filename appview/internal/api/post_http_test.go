package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

func authedReq(method, path, body, did string) *http.Request {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, reader)
	ctx := middleware.WithDID(request.Context(), syntax.DID(did))
	ctx = middleware.WithOwnerGeneration(ctx, 1)
	return request.WithContext(ctx)
}

func authedPostPathReq(method, path, body, did string) *http.Request {
	request := authedReq(method, path, body, did)
	parts := strings.Split(strings.Trim(path, "/"), "/")
	request.SetPathValue("did", parts[2])
	request.SetPathValue("rkey", parts[3])
	return request
}

func decodeResponseJSON[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(recorder.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode response JSON: %v; body=%q", err, recorder.Body.String())
	}
	return value
}

func assertAPIError(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) envelope.Error {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, status, recorder.Body.String())
	}
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
		t.Fatalf("error response=%+v, want code %q and non-empty message", body, code)
	}
	return body
}

func TestAuthedReqPreservesBody(t *testing.T) {
	t.Parallel()
	const body = `{"text":"missing sponsored"}`
	request := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	got, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	if string(got) != body {
		t.Fatalf("request body = %q, want exact fixture %q", got, body)
	}
}
