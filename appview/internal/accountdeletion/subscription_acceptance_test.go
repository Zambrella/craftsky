package accountdeletion_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/accountdeletion"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
)

type subscriptionDeletionService struct{}

func (subscriptionDeletionService) CreateIntent(context.Context, accountdeletion.CreateIntentParams) (accountdeletion.IntentResult, error) {
	return accountdeletion.IntentResult{
		JobID: "10000000-0000-4000-8000-000000000051",
		Warning: &accountdeletion.DeletionWarning{
			Code:    "provider_billing_not_canceled",
			Message: "Deleting your CraftSky account does not cancel provider billing and may end CraftSky access.",
		},
	}, nil
}

func (subscriptionDeletionService) CancelIntent(context.Context, string, syntax.DID) error {
	return nil
}
func (subscriptionDeletionService) Accept(context.Context, accountdeletion.AcceptParams) error {
	return accountdeletion.ErrProviderBillingMustBeResolved
}

func TestSubscriptionAwareAccountDeletionWarnsAndBlocksGenerically(t *testing.T) {
	service := subscriptionDeletionService{}
	owner := syntax.DID("did:plc:subscription-deletion-owner")
	intentRequest := httptest.NewRequest(http.MethodPost, "/v1/account-deletion/intents", nil)
	intentContext := middleware.WithDID(intentRequest.Context(), owner)
	intentContext = middleware.WithDeviceID(intentContext, "deletion-device")
	intentResponse := httptest.NewRecorder()
	api.CreateAccountDeletionIntentHandler(service).ServeHTTP(intentResponse, intentRequest.WithContext(intentContext))
	if intentResponse.Code != http.StatusCreated {
		t.Fatalf("intent status = %d; body=%s", intentResponse.Code, intentResponse.Body.String())
	}
	var intentBody map[string]any
	if err := json.Unmarshal(intentResponse.Body.Bytes(), &intentBody); err != nil {
		t.Fatal(err)
	}
	warning, ok := intentBody["warning"].(map[string]any)
	if !ok || warning["code"] != "provider_billing_not_canceled" {
		t.Fatalf("deletion warning = %#v", intentBody["warning"])
	}

	acceptRequest := httptest.NewRequest(http.MethodPost, "/v1/account-deletions/10000000-0000-4000-8000-000000000051", strings.NewReader(`{"reauthProof":"proof","confirmationDid":"did:plc:subscription-deletion-owner"}`))
	acceptRequest.SetPathValue("jobId", "10000000-0000-4000-8000-000000000051")
	acceptRequest = acceptRequest.WithContext(middleware.WithDID(acceptRequest.Context(), owner))
	acceptResponse := httptest.NewRecorder()
	api.AcceptAccountDeletionHandler(service).ServeHTTP(acceptResponse, acceptRequest)
	if acceptResponse.Code != http.StatusConflict {
		t.Fatalf("accept status = %d; body=%s", acceptResponse.Code, acceptResponse.Body.String())
	}
	var errorBody map[string]any
	if err := json.Unmarshal(acceptResponse.Body.Bytes(), &errorBody); err != nil {
		t.Fatal(err)
	}
	if errorBody["error"] != "provider_billing_must_be_resolved" || errorBody["message"] != "Resolve provider billing and refresh subscription status before deleting this account." {
		t.Fatalf("blocking error = %#v", errorBody)
	}
}
