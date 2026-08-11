package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/accountdeletion"
	"social.craftsky/appview/internal/app"
)

func TestAccountDeletionStatusRoutesUseOnlyMatchingRestrictedCredential(t *testing.T) {
	t.Parallel()

	wantPolicies := map[string]accountdeletion.StatusAction{
		"GET /v1/account-deletions/{jobId}":         accountdeletion.StatusRead,
		"POST /v1/account-deletions/{jobId}/retry":  accountdeletion.StatusRetry,
		"POST /v1/account-deletions/{jobId}/reauth": accountdeletion.StatusStartReauthentication,
	}
	for _, policy := range V1RoutePolicies(app.EnvDev, app.Config{Env: app.EnvDev}) {
		key := policy.Method + " " + policy.PathPattern
		want, ok := wantPolicies[key]
		if !ok {
			continue
		}
		if policy.AuthRequired || policy.CurrentMemberRequired || policy.DeletionStatusAction != want {
			t.Fatalf("%s policy = %+v", key, policy)
		}
		delete(wantPolicies, key)
	}
	if len(wantPolicies) != 0 {
		t.Fatalf("missing deletion status policies: %v", wantPolicies)
	}

	jobID := uuid.MustParse("10000000-0000-0000-0000-000000000021")
	service := &recordingStatusRouteService{jobID: jobID, owner: syntax.DID("did:plc:alice")}
	deps := testDeps()
	deps.AccountDeletionStatus = service
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	for _, target := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/account-deletions/" + jobID.String()},
		{http.MethodPost, "/v1/account-deletions/" + jobID.String() + "/retry"},
		{http.MethodPost, "/v1/account-deletions/" + jobID.String() + "/reauth"},
	} {
		response := serveStatusRequest(mux, target.method, target.path, "Bearer ordinary-token")
		if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), `"error":"invalid_deletion_status"`) {
			t.Fatalf("ordinary credential %s status = %d body = %s", target.path, response.Code, response.Body.String())
		}
	}

	response := serveStatusRequest(mux, http.MethodGet, "/v1/account-deletions/"+jobID.String(), "DeletionStatus valid-status")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"phase":"removingCraftskyRecords"`) || strings.Contains(response.Body.String(), "oauth-session") {
		t.Fatalf("status response = %d body = %s", response.Code, response.Body.String())
	}
	response = serveStatusRequest(mux, http.MethodPost, "/v1/account-deletions/"+jobID.String()+"/retry", "DeletionStatus valid-status")
	if response.Code != http.StatusAccepted || service.retryCalls != 1 {
		t.Fatalf("retry response = %d calls = %d body = %s", response.Code, service.retryCalls, response.Body.String())
	}
	response = serveStatusRequest(mux, http.MethodPost, "/v1/account-deletions/"+jobID.String()+"/reauth", "DeletionStatus valid-status")
	if response.Code != http.StatusOK || service.reauthCalls != 1 || !strings.Contains(response.Body.String(), `"authUrl":"https://auth.invalid/reauth"`) {
		t.Fatalf("reauth response = %d calls = %d body = %s", response.Code, service.reauthCalls, response.Body.String())
	}

	crossJob := "10000000-0000-0000-0000-000000000022"
	response = serveStatusRequest(mux, http.MethodGet, "/v1/account-deletions/"+crossJob, "DeletionStatus valid-status")
	if response.Code != http.StatusUnauthorized || service.statusCalls != 1 {
		t.Fatalf("cross-job response = %d statusCalls = %d body = %s", response.Code, service.statusCalls, response.Body.String())
	}
}

func serveStatusRequest(handler http.Handler, method, path, authorization string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, nil)
	request.Header.Set("Authorization", authorization)
	request.Header.Set("X-Craftsky-Device-Id", "alice-phone")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

type recordingStatusRouteService struct {
	jobID       uuid.UUID
	owner       syntax.DID
	statusCalls int
	retryCalls  int
	reauthCalls int
}

func (service *recordingStatusRouteService) AuthorizeStatusRoute(_ context.Context, token string, jobID uuid.UUID, deviceID string, _ accountdeletion.StatusAction) (accountdeletion.StatusGrant, error) {
	if token != "valid-status" || jobID != service.jobID || deviceID != "alice-phone" {
		return accountdeletion.StatusGrant{}, accountdeletion.ErrStatusUnauthorized
	}
	return accountdeletion.StatusGrant{JobID: jobID.String(), Owner: service.owner, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func (service *recordingStatusRouteService) GetStatus(context.Context, uuid.UUID, syntax.DID) (accountdeletion.DeletionStatusView, error) {
	service.statusCalls++
	return accountdeletion.ProjectDeletionStatus(service.jobID.String(), accountdeletion.StatusActive, accountdeletion.PhaseRemovingCraftskyRecords, false), nil
}

func (service *recordingStatusRouteService) Retry(context.Context, uuid.UUID, syntax.DID) (accountdeletion.DeletionStatusView, error) {
	service.retryCalls++
	return accountdeletion.ProjectDeletionStatus(service.jobID.String(), accountdeletion.StatusActive, accountdeletion.PhaseRemovingCraftskyRecords, false), nil
}

func (service *recordingStatusRouteService) StartReauthentication(context.Context, uuid.UUID, syntax.DID) (accountdeletion.ReauthenticationStart, error) {
	service.reauthCalls++
	return accountdeletion.ReauthenticationStart{AuthURL: "https://auth.invalid/reauth", ExpiresAt: time.Now().Add(10 * time.Minute)}, nil
}

var _ accountdeletion.StatusRouteService = (*recordingStatusRouteService)(nil)
