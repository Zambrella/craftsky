package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/eligibility"
)

type recordingEligibilityReviewer struct{ review eligibility.Review }

func (reviewer *recordingEligibilityReviewer) Review(_ context.Context, review eligibility.Review) (eligibility.Status, error) {
	reviewer.review = review
	return eligibility.Status{State: review.State, Appealable: review.State == eligibility.StateRestricted, AppealGuidance: review.AppealGuidance}, nil
}

func TestAgeEligibilityReviewHandlerUsesAuthenticatedAdministratorAndSupportsReversal(t *testing.T) {
	reviewer := &recordingEligibilityReviewer{}
	now := time.Date(2030, 9, 22, 12, 0, 0, 0, time.UTC)
	handler := AgeEligibilityReviewHandler(reviewer, func() time.Time { return now })

	request := httptest.NewRequest(http.MethodPost, "/v1/admin/eligibility/reviews", strings.NewReader(`{
		"accountDid":"did:plc:member","state":"eligible","reason":"appeal upheld"
	}`))
	request = request.WithContext(ctxkeys.WithModerator(request.Context(), ctxkeys.Moderator{
		ActorID: "eligibility-admin", Role: "safetyAdministrator",
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if reviewer.review.AccountDID != "did:plc:member" || reviewer.review.ReviewerID != "eligibility-admin" || reviewer.review.State != eligibility.StateEligible || !reviewer.review.ReviewedAt.Equal(now) {
		t.Fatalf("review=%+v", reviewer.review)
	}
}

func TestAgeEligibilityReviewHandlerRejectsUnprivilegedModerator(t *testing.T) {
	handler := AgeEligibilityReviewHandler(&recordingEligibilityReviewer{}, time.Now)
	request := httptest.NewRequest(http.MethodPost, "/v1/admin/eligibility/reviews", strings.NewReader(`{"accountDid":"did:plc:member","state":"eligible","reason":"appeal upheld"}`))
	request = request.WithContext(ctxkeys.WithModerator(request.Context(), ctxkeys.Moderator{ActorID: "helper", Role: "helper"}))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
