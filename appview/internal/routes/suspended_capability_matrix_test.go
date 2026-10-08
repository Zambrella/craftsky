package routes

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/moderation"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/subscriptions"
	"social.craftsky/appview/internal/testdb"
)

type suspendedBodyProbe struct {
	reads int
}

func (body *suspendedBodyProbe) Read([]byte) (int, error) {
	body.reads++
	return 0, io.EOF
}

func (*suspendedBodyProbe) Close() error { return nil }

type suspendedRouteEffects struct {
	handlerCalls    map[string]int
	pdsFactoryCalls int
}

type suspendedRouteReader struct {
	owner syntax.DID
	calls int
}

type recordingRouteEffectExecutor struct {
	resolvedGenerations []int64
	deleteRequests      []pdseffects.DeleteRecordRequest
}

type recordingRouteAddressedCommands struct {
	requests []pdscommands.AddressedDeleteCommandRequest
}

func (*recordingRouteAddressedCommands) Put(
	context.Context,
	pdscommands.AddressedPutCommandRequest,
) (pdscommands.CommandResult, error) {
	return pdscommands.CommandResult{}, errors.New("unexpected addressed put")
}

func (commands *recordingRouteAddressedCommands) Delete(
	_ context.Context,
	request pdscommands.AddressedDeleteCommandRequest,
) (pdscommands.CommandResult, error) {
	commands.requests = append(commands.requests, request)
	return pdscommands.CommandResult{TerminalResult: pdscommands.TerminalResult{
		State: pdscommands.CommandAccepted, HTTPStatus: http.StatusNoContent,
	}}, nil
}

func (executor *recordingRouteEffectExecutor) ResolveExpectedOwners(
	_ context.Context,
	generation int64,
	_ []syntax.DID,
) ([]ownerlifecycle.ExpectedOwner, error) {
	executor.resolvedGenerations = append(executor.resolvedGenerations, generation)
	return []ownerlifecycle.ExpectedOwner{{Owner: "did:plc:suspended", Generation: generation}}, nil
}

func (*recordingRouteEffectExecutor) ReadRecord(context.Context, pdseffects.ReadRecordRequest, any) (syntax.CID, error) {
	panic("unexpected ReadRecord call")
}

func (*recordingRouteEffectExecutor) PutRecord(context.Context, pdseffects.PutRecordRequest) (pdseffects.RecordResult, error) {
	panic("unexpected PutRecord call")
}

func (executor *recordingRouteEffectExecutor) DeleteRecord(
	_ context.Context,
	request pdseffects.DeleteRecordRequest,
) (pdseffects.RecordResult, error) {
	executor.deleteRequests = append(executor.deleteRequests, request)
	return pdseffects.RecordResult{}, nil
}

func (*recordingRouteEffectExecutor) UploadBlob(context.Context, pdseffects.UploadBlobRequest) (*auth.UploadedBlob, error) {
	panic("unexpected UploadBlob call")
}

func (reader *suspendedRouteReader) IsSuspended(_ context.Context, owner syntax.DID) (bool, error) {
	reader.owner = owner
	reader.calls++
	return true, nil
}

var routeParameterPattern = regexp.MustCompile(`\{[^}]+\}`)

func TestSuspendedAccountInvokesEveryRegisteredAuthenticatedRoute(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000076_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
		INSERT INTO craftsky_profiles (did) VALUES ('did:plc:suspended');
	`+string(migration))
	for _, statement := range []string{
		`INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id) VALUES ('10000000-0000-4000-8000-000000000001','did:plc:suspended','20000000-0000-4000-8000-000000000001')`,
		`INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,mapped_tier,accepted_generation) VALUES ('30000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000001','project','subscription','business','app','app_store','production','active',true,'business',1)`,
		`INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at) VALUES ('30000000-0000-4000-8000-000000000001','business','did:plc:suspended',now())`,
	} {
		if _, err := pool.Exec(context.Background(), statement); err != nil {
			t.Fatal(err)
		}
	}
	reader := &suspendedRouteReader{}
	effects := &suspendedRouteEffects{handlerCalls: make(map[string]int)}
	deps := testDeps()
	deps.Config = Config{Env: EnvProd, AllowedOrigins: []string{"*"}}
	deps.DB = pool
	deps.Subscriptions = subscriptions.NewStore(pool)
	deps.AuthService = &auth.MockAuthService{DefaultDID: "did:plc:suspended"}
	deps.SuspensionReader = reader
	deps.routeHandlerDecorator = func(policy RoutePolicy, _ http.Handler) http.Handler {
		key := policyKey(policy.Method, policy.PathPattern)
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			effects.handlerCalls[key]++
			w.WriteHeader(http.StatusNoContent)
		})
	}
	deps.NewBlobEffects = func(context.Context, syntax.DID, string) (api.BlobEffectExecutor, error) {
		effects.pdsFactoryCalls++
		return nil, nil
	}

	policies := V1RoutePolicies(EnvProd, deps.Config)
	catalogue, err := NewV1Catalogue(policies)
	if err != nil {
		t.Fatalf("construct catalogue: %v", err)
	}
	policyMux := NewPolicyMux(http.NewServeMux(), catalogue)
	AddRoutes(context.Background(), policyMux, deps)
	if err := policyMux.Validate(); err != nil {
		t.Fatalf("validate registered routes: %v", err)
	}
	handler := catalogue.RoutingHandler(policyMux)

	tested := make(map[string]struct{})
	for _, policy := range policies {
		if policy.AccessClass != AccessCurrentMember && policy.AccessClass != AccessAuthenticatedRecovery {
			continue
		}
		key := policyKey(policy.Method, policy.PathPattern)
		if _, registered := policyMux.registered[key]; !registered {
			t.Fatalf("authenticated policy %s was not registered", key)
		}
		tested[key] = struct{}{}
		t.Run(key, func(t *testing.T) {
			beforeHandler := effects.handlerCalls[key]
			beforeReader := reader.calls
			beforePDS := effects.pdsFactoryCalls
			request, body := suspendedRouteRequest(policy)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if policy.SuspensionClass == SuspensionDenied {
				if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), `"error":"account_suspended"`) {
					t.Fatalf("status/body = %d/%s, want suspended denial", response.Code, response.Body.String())
				}
				if effects.handlerCalls[key] != beforeHandler || effects.pdsFactoryCalls != beforePDS {
					t.Fatal("denied route reached a handler or PDS boundary")
				}
				if reader.calls != beforeReader+1 || reader.owner != syntax.DID("did:plc:suspended") {
					t.Fatalf("suspension lookups = %d owner = %q", reader.calls-beforeReader, reader.owner)
				}
				if body != nil && body.reads != 0 {
					t.Fatalf("denied route read request body %d times", body.reads)
				}
				return
			}
			if response.Code != http.StatusNoContent || effects.handlerCalls[key] != beforeHandler+1 {
				t.Fatalf("allowed route status/handler delta = %d/%d, want 204/1", response.Code, effects.handlerCalls[key]-beforeHandler)
			}
			if reader.calls != beforeReader {
				t.Fatalf("allowed route performed %d suspension lookups", reader.calls-beforeReader)
			}
		})
	}

	for key := range policyMux.registered {
		policy := catalogue.policies[key]
		if policy.AccessClass != AccessCurrentMember && policy.AccessClass != AccessAuthenticatedRecovery {
			continue
		}
		if _, ok := tested[key]; !ok {
			t.Errorf("registered authenticated route was not invoked: %s", key)
		}
	}
}

func TestSuspendedStoredStandingEnforcesRealHandlersAtPDSBoundary(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
		CREATE TABLE owner_lifecycles (
			owner_did TEXT PRIMARY KEY,
			state TEXT NOT NULL,
			generation BIGINT NOT NULL,
			auth_epoch BIGINT NOT NULL,
			transition_reason TEXT NOT NULL,
			transitioned_at TIMESTAMPTZ NOT NULL,
			terminal_at TIMESTAMPTZ,
			purge_completed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		);
		CREATE TABLE moderation_account_standings (
			owner_did TEXT PRIMARY KEY,
			active_strike_count INTEGER NOT NULL DEFAULT 0,
			threshold_suspended BOOLEAN NOT NULL DEFAULT false,
			severe_suspended BOOLEAN NOT NULL DEFAULT false,
			effective_suspended BOOLEAN GENERATED ALWAYS AS
				(threshold_suspended OR severe_suspended) STORED,
			revision BIGINT NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		INSERT INTO craftsky_profiles (did) VALUES ('did:plc:suspended');
		INSERT INTO owner_lifecycles (
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES ('did:plc:suspended','active',7,1,'test',now(),now(),now());
		INSERT INTO moderation_account_standings (
			owner_did,active_strike_count,threshold_suspended
		) VALUES ('did:plc:suspended',3,true);
	`)
	executor := &recordingRouteEffectExecutor{}
	addressed := &recordingRouteAddressedCommands{}
	factoryCalls := 0
	deps := testDeps()
	deps.Config = Config{Env: EnvProd, AllowedOrigins: []string{"*"}}
	deps.DB = pool
	deps.AuthService = &auth.MockAuthService{DefaultDID: "did:plc:suspended"}
	deps.OwnerLifecycles = newRouteOwnerLifecycleStore(t, pool)
	deps.SuspensionReader = moderation.NewStore(pool)
	deps.PDSAddressedCommands = addressed
	deps.NewBlobEffects = func(_ context.Context, owner syntax.DID, sessionID string) (api.BlobEffectExecutor, error) {
		factoryCalls++
		if owner != syntax.DID("did:plc:suspended") || sessionID != "" {
			t.Fatalf("PDS factory owner/session = %q/%q", owner, sessionID)
		}
		return executor, nil
	}
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	remove := httptest.NewRequest(
		http.MethodDelete,
		"/v1/posts/did:plc:suspended/post1",
		nil,
	)
	remove.Header.Set("Authorization", "Bearer suspended-session")
	remove.Header.Set("X-Craftsky-Device-Id", "suspended-device")
	remove.Header.Set("Idempotency-Key", "018f4d5c-7a61-7d40-a1a2-777777777777")
	remove.Header.Set("If-Match", "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq")
	removed := httptest.NewRecorder()
	mux.ServeHTTP(removed, remove)
	if removed.Code != http.StatusNoContent {
		t.Fatalf("owner removal status/body = %d/%s, want 204", removed.Code, removed.Body.String())
	}
	if factoryCalls != 0 {
		t.Fatalf("legacy PDS factory calls = %d, want 0", factoryCalls)
	}
	if len(addressed.requests) != 1 {
		t.Fatalf("addressed delete requests = %d, want 1", len(addressed.requests))
	}
	deleted := addressed.requests[0]
	if deleted.Owner != syntax.DID("did:plc:suspended") ||
		deleted.OwnerGeneration != 7 ||
		deleted.URI != syntax.ATURI("at://did:plc:suspended/social.craftsky.feed.post/post1") {
		t.Fatalf("addressed delete request = %+v", deleted)
	}

	deniedBody := &suspendedBodyProbe{}
	create := httptest.NewRequest(http.MethodPost, "/v1/posts", deniedBody)
	create.ContentLength = 2
	create.Header.Set("Authorization", "Bearer suspended-session")
	create.Header.Set("X-Craftsky-Device-Id", "suspended-device")
	create.Header.Set("Content-Type", "application/json")
	denied := httptest.NewRecorder()
	mux.ServeHTTP(denied, create)
	if denied.Code != http.StatusForbidden || !strings.Contains(denied.Body.String(), `"error":"account_suspended"`) {
		t.Fatalf("public mutation status/body = %d/%s, want suspended denial", denied.Code, denied.Body.String())
	}
	if deniedBody.reads != 0 {
		t.Fatalf("denied public mutation read its body %d times", deniedBody.reads)
	}
	if factoryCalls != 0 || len(addressed.requests) != 1 {
		t.Fatalf("denied public mutation reached PDS boundary: legacyFactory=%d addressedDeletes=%d", factoryCalls, len(addressed.requests))
	}
}

func suspendedRouteRequest(policy RoutePolicy) (*http.Request, *suspendedBodyProbe) {
	path := routeParameterPattern.ReplaceAllString(policy.PathPattern, "fixture")
	var body *suspendedBodyProbe
	var request *http.Request
	if policy.BodyKind == BodyNoBody {
		request = httptest.NewRequest(policy.Method, path, nil)
	} else {
		body = &suspendedBodyProbe{}
		request = httptest.NewRequest(policy.Method, path, body)
		request.ContentLength = 2
		if policy.BodyKind == BodyDefaultJSON {
			request.Header.Set("Content-Type", "application/json")
		}
	}
	request.Header.Set("Authorization", "Bearer suspended-session")
	request.Header.Set("X-Craftsky-Device-Id", "suspended-device")
	return request, body
}

func TestEveryAuthenticatedV1MutationHasExactSuspensionClassification(t *testing.T) {
	wantAllowed := []string{
		"DELETE /v1/account-deletion/intents/{jobId}",
		"DELETE /v1/billing/licenses/{licenseId}/assignment",
		"DELETE /v1/events/{did}/{rkey}",
		"DELETE /v1/migrations/instagram/account",
		"DELETE /v1/migrations/instagram/imports/{importId}",
		"DELETE /v1/migrations/instagram/suggestions/{suggestionId}",
		"DELETE /v1/migrations/instagram/verifications/{verificationId}",
		"DELETE /v1/notifications/devices/{accountSubscriptionId}",
		"DELETE /v1/posts/{did}/{rkey}",
		"DELETE /v1/posts/{did}/{rkey}/pin",
		"DELETE /v1/posts/{did}/{rkey}/saves",
		"DELETE /v1/profiles/me/business",
		"DELETE /v1/profiles/{handleOrDid}/blocks",
		"DELETE /v1/profiles/{handleOrDid}/mutes",
		"DELETE /v1/saved-post-folders/{folderId}",
		"DELETE /v1/scheduled-post-media/{mediaId}",
		"DELETE /v1/scheduled-posts/{id}",
		"DELETE /v1/search/recent/{id}",
		"PATCH /v1/notifications/preferences",
		"PATCH /v1/saved-post-folders/{folderId}",
		"POST /v1/account-deletion/intents",
		"POST /v1/account-deletions/{jobId}",
		"POST /v1/auth/logout",
		"POST /v1/billing/reconciliation",
		"POST /v1/events/{did}/{rkey}/reports",
		"POST /v1/languages/preferences/initialize",
		"POST /v1/notifications/devices",
		"POST /v1/notifications/seen",
		"POST /v1/posts/{did}/{rkey}/reports",
		"POST /v1/posts/{did}/{rkey}/saves",
		"POST /v1/profiles/{handleOrDid}/blocks",
		"POST /v1/profiles/{handleOrDid}/mutes",
		"POST /v1/profiles/{handleOrDid}/reports",
		"POST /v1/saved-post-folders",
		"POST /v1/search/recent",
		"PUT /v1/billing/account",
		"PUT /v1/billing/licenses/{licenseId}/assignment",
		"PUT /v1/languages/preferences",
	}
	wantDenied := []string{
		"DELETE /v1/posts/{did}/{rkey}/likes",
		"DELETE /v1/posts/{did}/{rkey}/reposts",
		"DELETE /v1/profiles/{handleOrDid}/follows",
		"PATCH /v1/migrations/instagram/imports/{importId}",
		"PATCH /v1/migrations/instagram/settings",
		"POST /v1/blobs/images",
		"POST /v1/blobs/videos/authorization",
		"POST /v1/events",
		"POST /v1/link-previews",
		"POST /v1/migrations/instagram/imports",
		"POST /v1/migrations/instagram/suggestions/{suggestionId}/accept",
		"POST /v1/migrations/instagram/verifications",
		"POST /v1/migrations/instagram/verifications/{verificationId}/confirm",
		"POST /v1/onboarding/completion",
		"POST /v1/posts",
		"POST /v1/posts/{did}/{rkey}/likes",
		"POST /v1/posts/{did}/{rkey}/reposts",
		"POST /v1/profiles/{handleOrDid}/follows",
		"POST /v1/scheduled-posts",
		"POST /v1/scheduled-posts/{id}/publication",
		"PUT /v1/events/{did}/{rkey}",
		"PUT /v1/posts/{did}/{rkey}/pin",
		"PUT /v1/profiles/me",
		"PUT /v1/profiles/me/business",
		"PUT /v1/profiles/me/customisation",
		"PUT /v1/scheduled-post-media/{mediaId}",
		"PUT /v1/scheduled-posts/{id}",
	}
	slices.Sort(wantAllowed)
	slices.Sort(wantDenied)
	gotAllowedMatrix := make([]string, 0, len(retainedSuspendedMutations))
	for key := range retainedSuspendedMutations {
		gotAllowedMatrix = append(gotAllowedMatrix, key)
	}
	gotDeniedMatrix := make([]string, 0, len(deniedSuspendedMutations))
	for key := range deniedSuspendedMutations {
		gotDeniedMatrix = append(gotDeniedMatrix, key)
	}
	slices.Sort(gotAllowedMatrix)
	slices.Sort(gotDeniedMatrix)
	if !slices.Equal(gotAllowedMatrix, wantAllowed) {
		t.Errorf("retained suspended mutation matrix\n got: %q\nwant: %q", gotAllowedMatrix, wantAllowed)
	}
	if !slices.Equal(gotDeniedMatrix, wantDenied) {
		t.Errorf("denied suspended mutation matrix\n got: %q\nwant: %q", gotDeniedMatrix, wantDenied)
	}

	var gotAllowed, gotDenied []string
	for _, policy := range V1RoutePolicies(EnvProd, Config{}) {
		if !policy.SuspensionClass.Valid() {
			t.Errorf("%s %s has invalid suspension class %q", policy.Method, policy.PathPattern, policy.SuspensionClass)
		}
		if policy.Method == http.MethodGet && policy.SuspensionClass != SuspensionAllowed {
			t.Errorf("read route %s must remain available", policy.PathPattern)
		}
		if policy.Method == http.MethodGet || policy.AccessClass == AccessAnonymous || policy.AccessClass == AccessModerator {
			continue
		}
		key := policyKey(policy.Method, policy.PathPattern)
		switch policy.SuspensionClass {
		case SuspensionAllowed:
			gotAllowed = append(gotAllowed, key)
		case SuspensionDenied:
			gotDenied = append(gotDenied, key)
		}
	}
	slices.Sort(gotAllowed)
	slices.Sort(gotDenied)
	if !slices.Equal(gotAllowed, wantAllowed) {
		t.Errorf("allowed suspended mutations\n got: %q\nwant: %q", gotAllowed, wantAllowed)
	}
	if !slices.Equal(gotDenied, wantDenied) {
		t.Errorf("denied suspended mutations\n got: %q\nwant: %q", gotDenied, wantDenied)
	}
}

func TestEveryConfiguredV1RouteHasValidSuspensionClass(t *testing.T) {
	configs := []struct {
		env Environment
		cfg Config
	}{
		{env: EnvProd, cfg: Config{}},
		{env: EnvProd, cfg: Config{ModerationAdminEnabled: true}},
		{env: EnvDev, cfg: Config{EnableDevModeration: true, DevModerationToken: "token"}},
	}
	for _, config := range configs {
		for _, policy := range V1RoutePolicies(config.env, config.cfg) {
			if !policy.SuspensionClass.Valid() {
				t.Errorf("%s %s has invalid suspension class %q", policy.Method, policy.PathPattern, policy.SuspensionClass)
			}
		}
	}
}

func TestUnclassifiedAuthenticatedMutationFailsCatalogueConstruction(t *testing.T) {
	policy := RoutePolicy{
		Method:      http.MethodPost,
		PathPattern: "/v1/future-mutation",
		RateClass:   RateClassWrite,
		BodyKind:    BodyDefaultJSON,
		AccessClass: AccessCurrentMember,
	}
	policy.SuspensionClass = suspensionClassFor(policy)

	if policy.SuspensionClass != SuspensionUnspecified {
		t.Fatalf("unclassified mutation class = %q, want unspecified", policy.SuspensionClass)
	}
	if _, err := NewV1Catalogue([]RoutePolicy{policy}); err == nil {
		t.Fatal("catalogue accepted an unclassified authenticated mutation")
	}
}

func TestSuspensionClassFailsClosed(t *testing.T) {
	if (SuspensionClass(0)).Valid() || SuspensionUnspecified.AllowedWhenSuspended() {
		t.Fatal("zero suspension class must be invalid and denied")
	}
	if !SuspensionAllowed.AllowedWhenSuspended() || SuspensionDenied.AllowedWhenSuspended() {
		t.Fatal("suspension class allow decision is incorrect")
	}
}
