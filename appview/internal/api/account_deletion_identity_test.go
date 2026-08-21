package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/accountdeletion"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

type identityUnavailableDeletionService struct{}

func (identityUnavailableDeletionService) CreateIntent(context.Context, accountdeletion.CreateIntentParams) (accountdeletion.IntentResult, error) {
	return accountdeletion.IntentResult{}, accountdeletion.ErrIdentityUnavailable
}

func (identityUnavailableDeletionService) CancelIntent(context.Context, string, syntax.DID) error {
	return nil
}

func (identityUnavailableDeletionService) Accept(context.Context, accountdeletion.AcceptParams) error {
	return nil
}

func TestCreateAccountDeletionIntentIdentityFailureIsRetryable(t *testing.T) {
	handler := api.CreateAccountDeletionIntentHandler(identityUnavailableDeletionService{})
	request := httptest.NewRequest(http.MethodPost, "/v1/account-deletion/intents", nil)
	ctx := middleware.WithDID(request.Context(), syntax.DID("did:plc:alice"))
	ctx = middleware.WithDeviceID(ctx, "device-alice")
	request = request.WithContext(ctx)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusServiceUnavailable, response.Body.String())
	}
	var body envelope.Error
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error != "identity_unavailable" {
		t.Fatalf("error = %q, want identity_unavailable", body.Error)
	}
}
