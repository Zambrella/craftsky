package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
)

// IT-017 / NFR-005 / AC-031: secret canaries traverse real migration paths,
// while every emitted sink retains only bounded operational context.
func TestPDSMigrationProductionPathsAreCrossSinkSecretFree(t *testing.T) {
	ctx := context.Background()
	pool := phase10MigrationPool(t)
	owner := syntax.DID("did:plc:phase21telemetry")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'telemetry fixture',now(),now(),now())
	`, owner); err != nil {
		t.Fatal(err)
	}

	const (
		accessCanary       = "migration-access-token-canary"
		refreshCanary      = "migration-refresh-token-canary"
		dpopProofCanary    = "migration-dpop-proof-canary"
		craftskyCanary     = "migration-craftsky-bearer-canary"
		deletionHashCanary = "migration-confirmation-hash-canary"
		rawSessionCanary   = "migration-session-json-canary"
	)
	privateKey, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	dpopKeyCanary := privateKey.Multibase()
	canaries := []string{
		accessCanary, refreshCanary, dpopKeyCanary, dpopProofCanary,
		craftskyCanary, deletionHashCanary, rawSessionCanary,
	}

	var localLogs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&localLogs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	metrics := observability.NewInMemoryMetricRecorder()
	observerLogs := &crossSinkLogRecorder{}
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{
		Env: "test", SentryDSN: "https://public@example.invalid/1",
		SentryTransport: transport, TracingEnabled: true, TracesSampleRate: 1,
		LogsEnabled: true, MetricRecorder: metrics, LogSink: observerLogs, Logger: logger,
	})

	owners := phase10OwnerStore(t, pool)
	authStore, oauthApp := phase10AuthStore(t, pool, owners)
	baseSession := oauth.ClientSessionData{
		AccountDID: owner, HostURL: "https://pds-b.example", AuthServerURL: "https://issuer-b.example",
		AuthServerTokenEndpoint:      "https://issuer-b.example/oauth/token",
		AuthServerRevocationEndpoint: "https://issuer-b.example/oauth/revoke",
		Scopes:                       []string{"atproto"}, AccessToken: accessCanary, RefreshToken: refreshCanary,
		DPoPPrivateKeyMultibase: dpopKeyCanary,
	}
	for _, sessionID := range []string{"success-parent", "transient-parent"} {
		data := baseSession
		data.SessionID = sessionID
		crossSinkInsertParent(t, pool, data)
	}
	staleSession := baseSession
	staleSession.SessionID = "stale-parent"
	staleSession.HostURL = "https://pds-a.example"
	staleSession.AuthServerURL = "https://issuer-a.example"
	staleSession.AuthServerTokenEndpoint = "https://issuer-a.example/oauth/token"
	staleSession.AuthServerRevocationEndpoint = "https://issuer-a.example/oauth/revoke"
	crossSinkInsertParent(t, pool, staleSession)
	childHash := sha256.Sum256([]byte(craftskyCanary))
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_sessions(
			token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,
			last_seen_at,idle_expires_at
		) VALUES($1,$2,'stale-parent','active',1,now(),now()+interval '1 day')
	`, childHash[:], owner); err != nil {
		t.Fatal(err)
	}

	currentAuthority := auth.OAuthAuthority{
		DID: owner, PDSOrigin: "https://pds-b.example", IssuerOrigin: "https://issuer-b.example",
	}
	newCoordinator := func(verifier auth.OAuthAuthorityVerifier) *auth.OAuthSessionCoordinator {
		coordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
			App: oauthApp, Store: authStore, Owners: owners, AuthorityVerifier: verifier,
			Observer: observer, OperationTimeout: time.Second,
		})
		if err != nil {
			t.Fatal(err)
		}
		return coordinator
	}
	successCoordinator := newCoordinator(crossSinkAuthorityVerifier{authority: currentAuthority})
	var successReached bool
	if err := successCoordinator.WithActiveSession(ctx, owner, "success-parent", func(_ context.Context, session *oauth.ClientSession) error {
		successReached = true
		if session.Data.AccessToken != accessCanary || session.Data.RefreshToken != refreshCanary ||
			session.Data.DPoPPrivateKeyMultibase != dpopKeyCanary {
			t.Fatalf("success path did not receive credential canaries")
		}
		return nil
	}); err != nil || !successReached {
		t.Fatalf("matching authority path reached=%t err=%v", successReached, err)
	}

	staleCoordinator := newCoordinator(crossSinkAuthorityVerifier{authority: currentAuthority})
	transientPathError := errors.New(strings.Join([]string{
		dpopProofCanary, deletionHashCanary, `{"accessToken":"` + rawSessionCanary + `"}`,
	}, " "))
	transientCoordinator := newCoordinator(crossSinkAuthorityVerifier{err: transientPathError})
	authService := &crossSinkAuthService{owner: owner, sessions: map[string]string{
		craftskyCanary: "stale-parent", "migration-transient-bearer": "transient-parent",
	}}
	var effects []*crossSinkEffects
	factory := pdseffects.ExecutorFactory(func(_ context.Context, requestOwner syntax.DID, sessionID string) (pdseffects.EffectExecutor, error) {
		coordinator := staleCoordinator
		if sessionID == "transient-parent" {
			coordinator = transientCoordinator
		}
		effect := &crossSinkEffects{coordinator: coordinator, owner: requestOwner, sessionID: sessionID}
		effects = append(effects, effect)
		return effect, nil
	})
	endpoint := api.PutBusinessProfileHandler(factory)
	endpoint = middleware.CurrentMember(crossSinkCurrentMember{}, logger, owners)(endpoint)
	endpoint = middleware.Authenticated(authService, logger, middleware.DevAuthPolicy{Mode: middleware.DevAuthDisabled})(endpoint)
	mux := http.NewServeMux()
	mux.Handle("PUT /v1/business/profile", endpoint)
	handler := middleware.Logging(logger)(middleware.HTTPMetrics(observer)(mux))

	staleResponse := crossSinkRequest(t, handler, craftskyCanary)
	transientResponse := crossSinkRequest(t, handler, "migration-transient-bearer")
	if staleResponse.Code != http.StatusUnauthorized {
		t.Fatalf("stale status=%d body=%s", staleResponse.Code, staleResponse.Body.String())
	}
	if transientResponse.Code < http.StatusInternalServerError || transientResponse.Code == http.StatusUnauthorized {
		t.Fatalf("transient status=%d body=%s", transientResponse.Code, transientResponse.Body.String())
	}
	if len(effects) != 2 || effects[0].operationCalled || effects[1].operationCalled {
		t.Fatalf("credential-bearing API effects ran: %#v", effects)
	}
	if len(authService.seenTokens) != 2 || authService.seenTokens[0] != craftskyCanary {
		t.Fatalf("authenticated bearer canary did not reach production middleware: %#v", authService.seenTokens)
	}
	apiBodies := []json.RawMessage{
		append(json.RawMessage(nil), staleResponse.Body.Bytes()...),
		append(json.RawMessage(nil), transientResponse.Body.Bytes()...),
	}
	var staleBody map[string]any
	if err := json.Unmarshal(apiBodies[0], &staleBody); err != nil {
		t.Fatal(err)
	}
	requestID, _ := staleBody["requestId"].(string)
	if len(staleBody) != 3 || staleBody["error"] != "pds_session_expired" || requestID == "" ||
		!crossSinkLogHasRunID(localLogs.String(), requestID) {
		t.Fatalf("stale API correlation body=%#v logs=%s", staleBody, localLogs.String())
	}

	ingestionStore, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	ingestionService, err := ingestion.NewService(ingestion.ServiceConfig{
		Store: ingestionStore, Lifecycles: owners,
		ProfileParticipant: func(context.Context, pgx.Tx, ownerlifecycle.Lifecycle, ownerlifecycle.Lifecycle) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := newTransactionalIndexerDispatcherWithActorDeletion(pool, logger, nil, nil, nil)
	repair, err := ingestion.NewRepositoryRepair(ingestion.RepositoryRepairConfig{
		Store: ingestionStore, Ingestor: ingestionService, Projector: dispatcher.Project,
	})
	if err != nil {
		t.Fatal(err)
	}
	repairPathError := errors.New("repository unavailable " + strings.Join(canaries, " "))
	var repairPathCalled bool
	directory := repositorySnapshotDirectoryFunc(func(context.Context, syntax.DID) (*identity.Identity, error) {
		repairPathCalled = true
		return nil, repairPathError
	})
	fetcher, err := ingestion.NewRepositorySnapshotFetcher(directory, http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	if err := ingestionStore.EnqueueRepositoryJob(ctx, owner, ingestion.RepositoryJobPDSReconcile); err != nil {
		t.Fatal(err)
	}
	worker, err := ingestion.NewRepositoryWorker(ingestion.RepositoryWorkerConfig{
		Store:    ingestionStore,
		Handler:  newTapRepositoryJobHandler(phase10RepositoryTracker{}, fetcher, dispatcher, repair, observer),
		WorkerID: "phase21-cross-sink", PollInterval: time.Second, LeaseDuration: time.Minute,
		BatchSize: 1, BackoffMin: time.Second, BackoffMax: time.Minute,
		Logger: logger, Observer: observer,
	})
	if err != nil {
		t.Fatal(err)
	}
	processed, repairErr := worker.RunOnce(ctx)
	if processed != 1 || repairErr == nil || !repairPathCalled {
		t.Fatalf("production repair path called=%t processed=%d err=%v", repairPathCalled, processed, repairErr)
	}

	callbackState := "phase21-callback-cleanup"
	err = owners.WithExistingAuth(ctx, owner, func(authCtx context.Context, authority ownerlifecycle.Lifecycle) error {
		callbackRequestCtx := auth.WithLoginAuthRequest(
			authCtx, owner, authority.Generation, authority.AuthEpoch,
			staleSession.HostURL, staleSession.AuthServerURL,
			auth.HandoffVerifiedLink, "phase21-device", "",
		)
		if err := authStore.SaveAuthRequestInfo(callbackRequestCtx, oauth.AuthRequestData{
			State: callbackState, RequestURI: "urn:request:" + callbackState,
			AuthServerURL: staleSession.AuthServerURL,
		}); err != nil {
			return err
		}
		callbackAttempt, err := authStore.BeginExchange(authCtx, callbackState)
		if err != nil {
			return err
		}
		callbackSession := staleSession
		callbackSession.SessionID = callbackState
		if err := authStore.QuarantineCallbackCredential(authCtx, callbackState, callbackAttempt, callbackSession, time.Now()); err != nil {
			return err
		}
		return authStore.MarkCallbackCredentialForCleanup(authCtx, callbackState, callbackAttempt)
	})
	if err != nil {
		t.Fatal(err)
	}

	revoker := &crossSinkCredentialRevoker{err: errors.New(
		"cleanup unavailable " + dpopProofCanary + " " + deletionHashCanary + " " + rawSessionCanary,
	)}
	cleanup, err := auth.NewOAuthRevocationProcessor(auth.OAuthRevocationProcessorOptions{
		Pool: pool, Revoker: revoker, Now: time.Now, NewLeaseToken: uuid.New,
		BatchSize: 2, LeaseDuration: time.Minute, OperationTimeout: 10 * time.Second,
		MaxAttempts: 3, BaseBackoff: time.Second, MaxBackoff: time.Minute,
		MaxCredentialRetention: 24 * time.Hour, Observer: observer,
	})
	if err != nil {
		t.Fatal(err)
	}
	processed, err = cleanup.ProcessBatch(ctx)
	if err != nil || processed != 2 {
		t.Fatalf("production cleanup path processed=%d err=%v", processed, err)
	}
	if len(revoker.sessions) != 2 {
		t.Fatalf("cleanup path did not receive credential canaries: %#v", revoker.sessions)
	}
	seenCleanupSessions := map[string]bool{}
	for _, session := range revoker.sessions {
		if session.AccessToken != accessCanary || session.RefreshToken != refreshCanary ||
			session.DPoPPrivateKeyMultibase != dpopKeyCanary {
			t.Fatalf("cleanup path did not receive credential canaries: %#v", revoker.sessions)
		}
		seenCleanupSessions[session.SessionID] = true
	}
	if !seenCleanupSessions["stale-parent"] || !seenCleanupSessions[callbackState] {
		t.Fatalf("parent and callback cleanup paths were not both exercised: %#v", seenCleanupSessions)
	}

	if !observer.Flush(time.Second) {
		t.Fatal("flush production migration telemetry")
	}
	for _, call := range metrics.Calls() {
		if err := observability.ValidateMetricCall(call); err != nil {
			t.Fatalf("invalid migration metric %#v: %v", call, err)
		}
	}
	crossSinkRequireMetric(t, metrics.Calls(), "craftsky_appview_authority_verifications_total", map[string]string{
		"operation": "session_select", "result": "success", "reason": "none",
	})
	crossSinkRequireMetric(t, metrics.Calls(), "craftsky_appview_authority_verifications_total", map[string]string{
		"operation": "session_select", "result": "mismatch", "reason": "pds_changed",
	})
	crossSinkRequireMetric(t, metrics.Calls(), "craftsky_appview_authority_verifications_total", map[string]string{
		"operation": "session_select", "result": "error", "reason": "resolve_failed",
	})
	crossSinkRequireMetric(t, metrics.Calls(), "craftsky_appview_repository_repairs_total", map[string]string{
		"job_kind": "pds_reconcile", "result": "retry", "reason": "source_unavailable",
	})
	crossSinkRequireMetric(t, metrics.Calls(), "craftsky_appview_repository_snapshot_verifications_total", map[string]string{
		"result": "error", "reason": "source_unavailable",
	})
	for _, credentialKind := range []string{"parent", "callback"} {
		crossSinkRequireMetric(t, metrics.Calls(), "craftsky_appview_oauth_cleanups_total", map[string]string{
			"credential_kind": credentialKind, "result": "retry", "reason": "dependency_unavailable",
		})
	}
	for name, want := range map[string]int{
		"craftsky_appview_authority_verifications_total":                     3,
		"craftsky_appview_authority_verification_duration_seconds":           3,
		"craftsky_appview_repository_repairs_total":                          1,
		"craftsky_appview_repository_repair_duration_seconds":                1,
		"craftsky_appview_repository_repair_attempt":                         1,
		"craftsky_appview_repository_snapshot_verifications_total":           1,
		"craftsky_appview_repository_snapshot_verification_duration_seconds": 1,
		"craftsky_appview_oauth_cleanups_total":                              2,
		"craftsky_appview_oauth_cleanup_duration_seconds":                    2,
		"craftsky_appview_oauth_cleanup_attempt":                             2,
	} {
		crossSinkRequireMetricCount(t, metrics.Calls(), name, want)
	}
	captured, err := json.Marshal(struct {
		Metrics      []observability.MetricCall
		LocalLogs    string
		ObserverLogs []crossSinkLogEvent
		SentryEvents []*sentry.Event
		APIBodies    []json.RawMessage
	}{metrics.Calls(), localLogs.String(), observerLogs.events, transport.Events(), apiBodies})
	if err != nil {
		t.Fatal(err)
	}
	output := string(captured)
	for _, canary := range canaries {
		if strings.Contains(output, canary) {
			t.Fatalf("secret canary %q escaped into production telemetry: %s", canary, output)
		}
	}
	for _, bounded := range []string{
		"session_select", "success", "none", "pds_changed", "resolve_failed",
		"pds_reconcile", "source_unavailable", "oauth_cleanup", "dependency_unavailable", "requestId", "run_id",
	} {
		if !strings.Contains(output, bounded) {
			t.Fatalf("bounded field %q missing from production telemetry: %s", bounded, output)
		}
	}
	var transactionCount int
	var correlatedErrorFound bool
	expectedTraceIDs := map[string]bool{}
	for _, effect := range effects {
		if effect.traceID == "" || effect.spanID == "" {
			t.Fatalf("production API path lacks trace correlation: %#v", effect)
		}
		expectedTraceIDs[effect.traceID] = true
	}
	for _, event := range transport.Events() {
		if event.Type == "transaction" && event.Transaction == "PUT /v1/business/profile" {
			traceID := fmt.Sprint(event.Contexts["trace"]["trace_id"])
			if !expectedTraceIDs[traceID] {
				t.Fatalf("transaction trace ID %q does not match a production API path: %#v", traceID, event.Contexts)
			}
			transactionCount++
		}
		if event.Level == sentry.LevelError && event.Tags["sentry_trace_id"] == effects[1].traceID &&
			event.Tags["sentry_span_id"] == effects[1].spanID {
			correlatedErrorFound = true
		}
	}
	if transactionCount != 2 || !correlatedErrorFound {
		t.Fatalf("Sentry production traces/events transactionCount=%d correlatedError=%t events=%#v", transactionCount, correlatedErrorFound, transport.Events())
	}
}

type crossSinkAuthorityVerifier struct {
	authority auth.OAuthAuthority
	err       error
}

func (verifier crossSinkAuthorityVerifier) ResolveCurrent(context.Context, syntax.DID) (auth.OAuthAuthority, error) {
	return verifier.authority, verifier.err
}

type crossSinkAuthService struct {
	owner      syntax.DID
	sessions   map[string]string
	seenTokens []string
}

func (service *crossSinkAuthService) Authenticate(_ context.Context, token string) (auth.AuthInfo, error) {
	service.seenTokens = append(service.seenTokens, token)
	sessionID, ok := service.sessions[token]
	if !ok {
		return auth.AuthInfo{}, auth.ErrAuthTokenInvalid
	}
	return auth.AuthInfo{DID: service.owner, SessionID: sessionID}, nil
}

type crossSinkCurrentMember struct{}

func (crossSinkCurrentMember) IsCurrentMember(context.Context, syntax.DID) (bool, error) {
	return true, nil
}

type crossSinkEffects struct {
	coordinator     *auth.OAuthSessionCoordinator
	owner           syntax.DID
	sessionID       string
	operationCalled bool
	traceID         string
	spanID          string
}

func (effects *crossSinkEffects) ResolveExpectedOwners(
	_ context.Context,
	ownerGeneration int64,
	_ []syntax.DID,
) ([]ownerlifecycle.ExpectedOwner, error) {
	return []ownerlifecycle.ExpectedOwner{{Owner: effects.owner, Generation: ownerGeneration}}, nil
}

func (effects *crossSinkEffects) ReadRecord(
	ctx context.Context,
	request pdseffects.ReadRecordRequest,
	_ any,
) (syntax.CID, error) {
	effects.traceID, effects.spanID = observability.TraceIDs(ctx)
	err := effects.coordinator.WithActiveEffectSession(
		ctx, request.ExpectedOwners, effects.owner, effects.sessionID,
		func(context.Context, *oauth.ClientSession) error {
			effects.operationCalled = true
			return nil
		},
	)
	return "", err
}

func (*crossSinkEffects) PutRecord(context.Context, pdseffects.PutRecordRequest) (pdseffects.RecordResult, error) {
	return pdseffects.RecordResult{}, errors.New("unexpected profile write")
}

func (*crossSinkEffects) DeleteRecord(context.Context, pdseffects.DeleteRecordRequest) (pdseffects.RecordResult, error) {
	return pdseffects.RecordResult{}, errors.New("unexpected profile delete")
}

func (*crossSinkEffects) UploadBlob(context.Context, pdseffects.UploadBlobRequest) (*auth.UploadedBlob, error) {
	return nil, errors.New("unexpected profile blob upload")
}

type crossSinkCredentialRevoker struct {
	err      error
	sessions []oauth.ClientSessionData
}

func (revoker *crossSinkCredentialRevoker) RevokeSession(_ context.Context, data oauth.ClientSessionData) error {
	revoker.sessions = append(revoker.sessions, data)
	return revoker.err
}

type crossSinkLogEvent struct {
	Message string
	Attrs   observability.EventContext
}

type crossSinkLogRecorder struct {
	events []crossSinkLogEvent
}

func (recorder *crossSinkLogRecorder) Emit(_ context.Context, _ slog.Level, message string, attrs observability.EventContext) {
	recorder.events = append(recorder.events, crossSinkLogEvent{Message: message, Attrs: attrs})
}

func crossSinkInsertParent(t *testing.T, pool *pgxpool.Pool, data oauth.ClientSessionData) {
	t.Helper()
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			row_version,absolute_expires_at,created_at,updated_at
		) VALUES($1,$2,$3,'active',1,1,1,now()+interval '1 day',now(),now())
	`, data.AccountDID, data.SessionID, raw); err != nil {
		t.Fatal(err)
	}
}

func crossSinkRequest(t *testing.T, handler http.Handler, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/v1/business/profile", strings.NewReader(`{}`))
	request.Header.Set("Authorization", "Bearer "+bearer)
	request.Header.Set("If-Match", "*")
	handler.ServeHTTP(recorder, request)
	return recorder
}

func crossSinkLogHasRunID(logs, want string) bool {
	for _, line := range strings.Split(strings.TrimSpace(logs), "\n") {
		var event map[string]any
		if json.Unmarshal([]byte(line), &event) == nil && event["run_id"] == want {
			return true
		}
	}
	return false
}

func crossSinkRequireMetric(t *testing.T, calls []observability.MetricCall, name string, attrs map[string]string) {
	t.Helper()
	for _, call := range calls {
		if call.Name == name && reflect.DeepEqual(call.Attributes, attrs) {
			return
		}
	}
	t.Fatalf("metric %q missing attributes %#v in %#v", name, attrs, calls)
}

func crossSinkRequireMetricCount(t *testing.T, calls []observability.MetricCall, name string, want int) {
	t.Helper()
	count := 0
	for _, call := range calls {
		if call.Name == name {
			count++
		}
	}
	if count != want {
		t.Fatalf("metric %q count=%d, want %d in %#v", name, count, want, calls)
	}
}
