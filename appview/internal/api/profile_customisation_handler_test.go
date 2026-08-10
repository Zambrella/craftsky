package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
)

type fakeProfileCustomisationWriter struct {
	owner syntax.DID
	value api.ProfileCustomisation
	err   error
	calls int
}

func (f *fakeProfileCustomisationWriter) Put(
	_ context.Context,
	owner syntax.DID,
	value api.ProfileCustomisation,
) (api.ProfileCustomisation, error) {
	f.owner = owner
	f.value = value
	f.calls++
	return value, f.err
}

func TestPutProfileCustomisationHandlerReturnsAuthoritativeValue(t *testing.T) {
	t.Parallel()

	store := &fakeProfileCustomisationWriter{}
	handler := api.PutProfileCustomisationHandler(store)
	req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me/customisation", strings.NewReader(
		`{"colour":"teal","profileBorder":"thick","profileBackground":"x2"}`,
	))
	req = req.WithContext(middleware.WithDID(req.Context(), syntax.DID("did:plc:alice")))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	if store.owner != syntax.DID("did:plc:alice") || store.calls != 1 {
		t.Fatalf("store owner/calls = %q/%d, want authenticated Alice/1", store.owner, store.calls)
	}
	var got api.ProfileCustomisation
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got != store.value {
		t.Fatalf("response = %+v, want %+v", got, store.value)
	}
}

func TestPutProfileCustomisationHandlerMapsValidationAndStoreErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		storeError error
		wantStatus int
		wantCode   string
		wantCalls  int
	}{
		{
			name:       "unsupported value",
			body:       `{"colour":"#fff","profileBorder":"medium","profileBackground":"none"}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "validation_failed",
		},
		{
			name:       "missing field",
			body:       `{"colour":"cobalt","profileBorder":"medium"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "unexpected field",
			body:       `{"colour":"cobalt","profileBorder":"medium","profileBackground":"none","url":"https://example.com"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "unexpected_field",
		},
		{
			name:       "malformed body",
			body:       `{"colour":`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "malformed_body",
		},
		{
			name:       "store failure",
			body:       `{"colour":"cobalt","profileBorder":"medium","profileBackground":"none"}`,
			storeError: errors.New("database unavailable"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
			wantCalls:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := &fakeProfileCustomisationWriter{err: tt.storeError}
			handler := api.PutProfileCustomisationHandler(store)
			req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me/customisation", strings.NewReader(tt.body))
			req = req.WithContext(middleware.WithDID(req.Context(), syntax.DID("did:plc:alice")))
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, req)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, tt.wantStatus, response.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body["error"] != tt.wantCode {
				t.Fatalf("error = %v, want %q", body["error"], tt.wantCode)
			}
			if store.calls != tt.wantCalls {
				t.Fatalf("store calls = %d, want %d", store.calls, tt.wantCalls)
			}
		})
	}
}
