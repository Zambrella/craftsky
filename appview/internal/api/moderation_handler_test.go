package api_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
)

type moderationInserterStub struct {
	result *api.ModerationInsertResult
	err    error
	calls  int
	key    string
}

func (stub *moderationInserterStub) InsertOutput(
	_ context.Context,
	key string,
	_ api.ModerationOutputInput,
) (*api.ModerationInsertResult, error) {
	stub.calls++
	stub.key = key
	return stub.result, stub.err
}

func TestDevModerationHandlerRequiresValidIdempotencyKeyBeforePersistence(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		key       string
		wantError string
	}{
		{name: "missing", wantError: "missing_idempotency_key"},
		{name: "too short", key: "short", wantError: "invalid_idempotency_key"},
		{name: "non printable", key: "moderation-key-\u007f", wantError: "invalid_idempotency_key"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			store := &moderationInserterStub{}
			handler := api.DevModerationOzoneEventsHandler(
				"dev-token",
				api.ModerationRequestConfig{
					DefaultSourceDID:  "did:plc:labeler",
					TrustedSourceDIDs: []string{"did:plc:labeler"},
				},
				store,
				slog.New(slog.NewTextHandler(io.Discard, nil)),
			)
			req := httptest.NewRequest(http.MethodPost, "/v1/dev/moderation/ozone-events", strings.NewReader(`{
				"subject":{"type":"account","did":"did:plc:target"},
				"value":"hide","action":"negate"
			}`))
			req.Header.Set("X-Craftsky-Dev-Moderation-Token", "dev-token")
			if testCase.key != "" {
				req.Header.Set("Idempotency-Key", testCase.key)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), testCase.wantError) {
				t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
			}
			if store.calls != 0 {
				t.Fatalf("store calls = %d, want 0", store.calls)
			}
		})
	}
}

func TestDevModerationHandlerReplaysSuccessAndMapsKeyConflict(t *testing.T) {
	const key = "moderation-handler-0001"
	newRequest := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/v1/dev/moderation/ozone-events", strings.NewReader(`{
			"subject":{"type":"account","did":"did:plc:target"},
			"value":"hide","action":"negate"
		}`))
		req.Header.Set("X-Craftsky-Dev-Moderation-Token", "dev-token")
		req.Header.Set("Idempotency-Key", key)
		return req
	}
	config := api.ModerationRequestConfig{
		DefaultSourceDID:  "did:plc:labeler",
		TrustedSourceDIDs: []string{"did:plc:labeler"},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	successStore := &moderationInserterStub{result: &api.ModerationInsertResult{
		OutputID: "output-one", Status: "indexed", Replayed: true,
	}}
	success := httptest.NewRecorder()
	api.DevModerationOzoneEventsHandler("dev-token", config, successStore, logger).ServeHTTP(success, newRequest())
	if success.Code != http.StatusCreated || successStore.key != key ||
		success.Body.String() != "{\"outputId\":\"output-one\",\"status\":\"indexed\"}\n" {
		t.Fatalf("success = %d %s key=%q", success.Code, success.Body.String(), successStore.key)
	}

	conflictStore := &moderationInserterStub{err: api.ErrModerationIdempotencyKeyConflict}
	conflict := httptest.NewRecorder()
	api.DevModerationOzoneEventsHandler("dev-token", config, conflictStore, logger).ServeHTTP(conflict, newRequest())
	if conflict.Code != http.StatusConflict || !strings.Contains(conflict.Body.String(), "idempotency_key_conflict") {
		t.Fatalf("conflict = %d %s", conflict.Code, conflict.Body.String())
	}
	if !errors.Is(conflictStore.err, api.ErrModerationIdempotencyKeyConflict) {
		t.Fatal("conflict stub lost sentinel")
	}
}
