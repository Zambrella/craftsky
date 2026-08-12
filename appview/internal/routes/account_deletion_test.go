package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/accountdeletion"
	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/testdb"
)

func TestAccountDeletionAcceptanceRouteIsAuthenticatedOwnerScopedAndStrict(t *testing.T) {
	t.Parallel()

	wantPolicies := map[string]BodyKind{
		"POST /v1/account-deletion/intents":  BodyNoBody,
		"POST /v1/account-deletions/{jobId}": BodyDefaultJSON,
	}
	for _, policy := range V1RoutePolicies(app.EnvDev, app.Config{Env: app.EnvDev}) {
		key := policy.Method + " " + policy.PathPattern
		bodyKind, ok := wantPolicies[key]
		if !ok {
			continue
		}
		if !policy.AuthRequired || !policy.CurrentMemberRequired || policy.RateClass != RateClassWrite || policy.BodyKind != bodyKind {
			t.Fatalf("%s policy = %+v", key, policy)
		}
		delete(wantPolicies, key)
	}
	if len(wantPolicies) != 0 {
		t.Fatalf("missing account deletion policies: %v", wantPolicies)
	}

	owner := syntax.DID("did:plc:alice")
	pool := testdb.WithSchema(t, `CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);`)
	if _, err := pool.Exec(context.Background(), `INSERT INTO craftsky_profiles(did) VALUES($1)`, owner); err != nil {
		t.Fatal(err)
	}
	service := &recordingAccountDeletionService{}
	deps := testDeps()
	deps.DB = pool
	deps.AuthService = &auth.MockAuthService{DefaultDID: owner}
	deps.AccountDeletion = service
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	jobID := "10000000-0000-0000-0000-000000000001"
	path := "/v1/account-deletions/" + jobID
	response := serveAccountDeletionRequest(mux, path, `{"reauthProof":"proof","confirmationHandle":"alice.test"}`, "", "device-alice", "status-token")
	if response.Code != http.StatusUnauthorized || service.acceptCalls != 0 {
		t.Fatalf("unauthorized status = %d calls = %d body = %s", response.Code, service.acceptCalls, response.Body.String())
	}

	for _, body := range []string{
		`{"reauthProof":`,
		`{"reauthProof":"proof","confirmationHandle":"alice.test","targetDid":"did:plc:bob"}`,
		`{"reauthProof":"","confirmationHandle":"alice.test"}`,
	} {
		response = serveAccountDeletionRequest(mux, path, body, "bearer", "device-alice", "status-token")
		assertCanonicalDeletionError(t, response, http.StatusBadRequest, "invalid_request")
	}
	if service.acceptCalls != 0 {
		t.Fatalf("invalid requests mutated service %d times", service.acceptCalls)
	}

	response = serveAccountDeletionRequest(mux, path, `{"reauthProof":"stale","confirmationHandle":"alice.test"}`, "bearer", "device-alice", "status-token")
	assertCanonicalDeletionError(t, response, http.StatusUnauthorized, "reauthentication_required")
	if service.last.Owner != owner {
		t.Fatalf("service owner = %q, want authenticated Alice", service.last.Owner)
	}

	validBody := `{"reauthProof":"proof","confirmationHandle":"alice.test"}`
	response = serveAccountDeletionRequest(mux, path, validBody, "bearer", "device-alice", "status-token")
	if response.Code != http.StatusAccepted || response.Body.Len() != 0 {
		t.Fatalf("valid acceptance status = %d body = %s", response.Code, response.Body.String())
	}
	response = serveAccountDeletionRequest(mux, path, validBody, "bearer", "device-alice", "status-token")
	if response.Code != http.StatusAccepted || service.acceptedJobs[jobID] != 2 {
		t.Fatalf("duplicate acceptance status = %d calls = %d body = %s", response.Code, service.acceptedJobs[jobID], response.Body.String())
	}
	if service.last.Owner != owner {
		t.Fatalf("derived acceptance scope = %+v", service.last)
	}

	for _, removed := range []struct {
		method     string
		path       string
		bearer     string
		wantStatus int
	}{
		{method: http.MethodGet, path: path, bearer: "bearer", wantStatus: http.StatusMethodNotAllowed},
		{method: http.MethodPost, path: path + "/retry", bearer: "bearer", wantStatus: http.StatusNotFound},
		{method: http.MethodPost, path: path + "/reauth", bearer: "bearer", wantStatus: http.StatusNotFound},
		// The static recovery route is gone. The generic acceptance pattern
		// may parse "recover" as a job ID, but a former status credential can
		// no longer authorize it as a recovery endpoint.
		{method: http.MethodPost, path: "/v1/account-deletions/recover", wantStatus: http.StatusUnauthorized},
	} {
		request := httptest.NewRequest(removed.method, removed.path, strings.NewReader(`{}`))
		if removed.bearer != "" {
			request.Header.Set("Authorization", "Bearer "+removed.bearer)
		}
		request.Header.Set("X-Craftsky-Device-Id", "device-alice")
		request.Header.Set("X-Craftsky-Deletion-Status", "former-status-token")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != removed.wantStatus {
			t.Fatalf("removed endpoint %s %s status=%d want=%d body=%s", removed.method, removed.path, response.Code, removed.wantStatus, response.Body.String())
		}
	}
}

func serveAccountDeletionRequest(mux http.Handler, path, body, bearer, deviceID, statusCapability string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	request.Header.Set("X-Craftsky-Device-Id", deviceID)
	request.Header.Set("X-Craftsky-Deletion-Status", statusCapability)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

func assertCanonicalDeletionError(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("error status = %d, want %d; body = %s", response.Code, status, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error envelope: %v; body = %s", err, response.Body.String())
	}
	if body["error"] != code || body["message"] == nil || body["requestId"] == nil {
		t.Fatalf("non-canonical error envelope = %#v", body)
	}
	if _, exists := body["request_id"]; exists {
		t.Fatalf("snake_case leaked into error envelope = %#v", body)
	}
}

type recordingAccountDeletionService struct {
	acceptCalls  int
	last         accountdeletion.AcceptParams
	acceptedJobs map[string]int
}

func (service *recordingAccountDeletionService) CreateIntent(context.Context, accountdeletion.CreateIntentParams) (accountdeletion.IntentResult, error) {
	return accountdeletion.IntentResult{
		JobID:     "10000000-0000-0000-0000-000000000099",
		AuthURL:   "https://auth.invalid/authorize",
		ExpiresAt: time.Date(2026, 8, 10, 15, 10, 0, 0, time.UTC),
	}, nil
}

func (service *recordingAccountDeletionService) CancelIntent(context.Context, string, syntax.DID) error {
	return nil
}

func (service *recordingAccountDeletionService) Accept(_ context.Context, params accountdeletion.AcceptParams) error {
	service.acceptCalls++
	service.last = params
	if params.ReauthProof == "stale" {
		return accountdeletion.ErrReauthenticationRequired
	}
	if service.acceptedJobs == nil {
		service.acceptedJobs = make(map[string]int)
	}
	service.acceptedJobs[params.JobID]++
	return nil
}

var _ accountdeletion.Service = (*recordingAccountDeletionService)(nil)
