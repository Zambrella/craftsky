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

	"social.craftsky/appview/internal/accountdeletion"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
)

type didBoundDeletionService struct {
	owner  syntax.DID
	accept accountdeletion.AcceptParams
}

func (service *didBoundDeletionService) CreateIntent(_ context.Context, params accountdeletion.CreateIntentParams) (accountdeletion.IntentResult, error) {
	if params.Owner != service.owner {
		return accountdeletion.IntentResult{}, errors.New("wrong owner")
	}
	return accountdeletion.IntentResult{
		JobID:           "10000000-0000-4000-8000-000000000013",
		AuthURL:         "https://auth.example/authorize",
		ConfirmationDID: service.owner,
	}, nil
}

func (*didBoundDeletionService) CancelIntent(context.Context, string, syntax.DID) error {
	return nil
}

func (service *didBoundDeletionService) Accept(_ context.Context, params accountdeletion.AcceptParams) error {
	service.accept = params
	if params.ConfirmationDID != service.owner {
		return accountdeletion.ErrConfirmationDIDMismatch
	}
	return nil
}

func TestAccountDeletionIdentityContractIgnoresHandleState(t *testing.T) {
	owner := syntax.DID("did:plc:alicefullidentifier")
	for _, handleState := range []string{"valid", "stale", "invalid", "absent"} {
		t.Run(handleState, func(t *testing.T) {
			service := &didBoundDeletionService{owner: owner}
			handler := api.CreateAccountDeletionIntentHandler(service)
			request := httptest.NewRequest(http.MethodPost, "/v1/account-deletion/intents", nil)
			ctx := middleware.WithDID(request.Context(), owner)
			ctx = middleware.WithDeviceID(ctx, "device-alice")
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request.WithContext(ctx))

			if response.Code != http.StatusCreated {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["confirmationDid"] != owner.String() {
				t.Fatalf("confirmationDid = %v, want %s", body["confirmationDid"], owner)
			}
			if _, exists := body["confirmationHandle"]; exists {
				t.Fatal("response retained confirmationHandle")
			}
		})
	}
}

func TestAccountDeletionAcceptRequiresExactFullDID(t *testing.T) {
	owner := syntax.DID("did:plc:alicefullidentifier")
	service := &didBoundDeletionService{owner: owner}
	handler := api.AcceptAccountDeletionHandler(service)

	request := httptest.NewRequest(http.MethodPost, "/v1/account-deletions/10000000-0000-4000-8000-000000000013",
		strings.NewReader(`{"reauthProof":"proof","confirmationDid":"did:plc:alicefullidentifieq"}`))
	request.SetPathValue("jobId", "10000000-0000-4000-8000-000000000013")
	ctx := middleware.WithDID(request.Context(), owner)
	request = request.WithContext(ctxkeys.WithRunID(ctx, "request-013"))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	var body envelope.Error
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error != "confirmation_did_mismatch" || body.Message == "" || body.RequestID == "" {
		t.Fatalf("error envelope = %+v", body)
	}
}
