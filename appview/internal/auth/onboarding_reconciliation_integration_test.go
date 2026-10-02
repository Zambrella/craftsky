package auth_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/testdb"
)

type recoveryPDS struct {
	auth.PDSClient
	auth.RepositoryCommandPDSClient
	auth.PDSRecordLister
	record           map[string]any
	readErr          error
	writeErr         error
	applyCalls       int
	concurrentRecord map[string]any
}

func (p *recoveryPDS) GetRecord(_ context.Context, _ syntax.DID, collection, _ string, out any) (string, error) {
	if collection == "app.bsky.actor.profile" {
		return "", auth.ErrRecordNotFound
	}
	if p.readErr != nil {
		return "", p.readErr
	}
	if p.record == nil {
		return "", auth.ErrRecordNotFound
	}
	raw, _ := json.Marshal(p.record)
	return "profile-cid", json.Unmarshal(raw, out)
}

func (p *recoveryPDS) LatestCommit(context.Context, syntax.DID) (syntax.CID, error) {
	return "repository-head", nil
}

func (p *recoveryPDS) ApplyWrites(_ context.Context, _ syntax.DID, _ syntax.CID, writes []auth.RepositoryWrite) error {
	p.applyCalls++
	if p.concurrentRecord != nil {
		p.record = p.concurrentRecord
		p.concurrentRecord = nil
		return auth.ErrRepositorySwapConflict
	}
	if p.writeErr != nil {
		return p.writeErr
	}
	if p.record != nil || len(writes) != 1 || writes[0].Action != "create" {
		return errors.New("unsafe profile write")
	}
	p.record = writes[0].Record.(map[string]any)
	return nil
}

type recoveryWriter struct {
	commands *pdscommands.OnboardingProfileService
}

func (w recoveryWriter) PutOnboardingProfile(ctx context.Context, client auth.PDSClient, request auth.OnboardingProfileWrite) (syntax.CID, error) {
	cid, err := w.commands.PutProfile(ctx, client, request.Owner, request.OwnerGeneration, request.Record)
	if errors.Is(err, pdscommands.ErrOnboardingProfileChanged) {
		err = errors.Join(auth.ErrProfileCreationConflict, err)
	}
	return cid, err
}

type recoveryFixture struct {
	pool       *pgxpool.Pool
	owners     *ownerlifecycle.Store
	store      *auth.PostgresAuthStore
	children   *auth.CraftskySessionStore
	handoffs   *auth.HandoffService
	flow       *auth.OAuthFlowService
	pds        *recoveryPDS
	writer     recoveryWriter
	owner      syntax.DID
	state      string
	exchanges  int
	departures int
	removeErr  error
	reconciler *auth.OnboardingReconciler
}

func newRecoveryFixture(t *testing.T) *recoveryFixture {
	t.Helper()
	ctx := context.Background()
	f := &recoveryFixture{pool: testdb.WithMigratedSchema(t), owner: "did:plc:missing-profile-recovery", state: "recovery-callback"}
	// Exercise recovery with only one connection: all fenced local work must
	// reuse it rather than acquire another pool connection.
	poolConfig := f.pool.Config()
	poolConfig.MaxConns = 1
	var poolErr error
	f.pool, poolErr = pgxpool.NewWithConfig(ctx, poolConfig)
	if poolErr != nil {
		t.Fatal(poolErr)
	}
	t.Cleanup(f.pool.Close)
	f.owners = newAuthOwnerStore(t, f.pool)
	config := testStoreConfig()
	config.OwnerLifecycles = f.owners
	f.store = auth.NewPostgresAuthStore(f.pool, config)
	var err error
	f.children, err = auth.NewCraftskySessionStoreWithConfig(f.pool, auth.CraftskySessionConfig{
		Inactivity: 24 * time.Hour, ActivityWriteInterval: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := auth.NewSessionLifecycleService(auth.SessionLifecycleOptions{Pool: f.pool, Owners: f.owners, Sessions: f.children})
	if err != nil {
		t.Fatal(err)
	}
	commandStore, err := pdscommands.NewStore(pdscommands.StoreConfig{Pool: f.pool, Lifecycles: f.owners})
	if err != nil {
		t.Fatal(err)
	}
	commands, err := pdscommands.NewOnboardingProfileService(commandStore, f.owners)
	if err != nil {
		t.Fatal(err)
	}
	f.writer = recoveryWriter{commands}
	f.pds = &recoveryPDS{}
	// Yesterday's onboarding succeeded; its accepted command must stay intact.
	err = f.owners.WithOnboardingAuth(ctx, f.owner, func(authCtx context.Context, authority ownerlifecycle.Lifecycle) error {
		_, err := commands.PutProfile(authCtx, f.pds, f.owner, authority.Generation,
			map[string]any{"$type": "social.craftsky.actor.profile", "crafts": []string{}})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.owners.Transition(ctx, ownerlifecycle.TransitionRequest{Owner: f.owner, ExpectedGeneration: 1, To: ownerlifecycle.StateActive, Reason: "profileActivated"})
	if err != nil {
		t.Fatal(err)
	}
	f.pds.record = nil
	f.pds.applyCalls = 0
	oldHash := sha256.Sum256([]byte("old-bearer"))
	for _, statement := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO craftsky_profiles(did,crafts,record_cid) VALUES($1,ARRAY['knitting'],'old-profile-cid')`, []any{f.owner}},
		{`INSERT INTO oauth_sessions(account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,absolute_expires_at)
		 VALUES($1,'old-parent','{}','active',2,1,now()+interval '1 day')`, []any{f.owner}},
		{`INSERT INTO craftsky_sessions(token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,last_seen_at,idle_expires_at)
		 VALUES($1,$2,'old-parent','active',1,now(),now()+interval '1 day')`, []any{oldHash[:], f.owner}},
	} {
		if _, err := f.pool.Exec(ctx, statement.sql, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	key, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	request := oauth.AuthRequestData{
		State: f.state, AccountDID: &f.owner, Scopes: []string{"atproto"}, RequestURI: "urn:request:recovery",
		AuthServerURL: "https://auth.example.com", AuthServerTokenEndpoint: "https://auth.example.com/oauth/token",
		PKCEVerifier: "verifier", DPoPPrivateKeyMultibase: key.Multibase(),
	}
	err = f.owners.WithExistingAuth(ctx, f.owner, func(authCtx context.Context, authority ownerlifecycle.Lifecycle) error {
		return f.store.SaveAuthRequestInfo(auth.WithLoginAuthRequest(authCtx, f.owner, authority.Generation, authority.AuthEpoch,
			"https://pds.example.com", "https://auth.example.com", auth.HandoffVerifiedLink, "recovery-device", ""), request)
	})
	if err != nil {
		t.Fatal(err)
	}
	pending := func(ctx context.Context, attempt auth.CallbackAttempt) (auth.PDSClient, error) {
		if _, err := f.store.ResumePendingOnboardingSession(ctx, attempt); err != nil {
			return nil, err
		}
		return f.pds, nil
	}
	noop := func(context.Context, pgx.Tx, ownerlifecycle.Lifecycle, ownerlifecycle.Lifecycle) error { return nil }
	reconciler, err := auth.NewOnboardingReconciler(f.owners, sessions, pending, noop,
		func(ctx context.Context, tx pgx.Tx, before, after ownerlifecycle.Lifecycle) error {
			f.departures++
			if f.removeErr != nil {
				return f.removeErr
			}
			_, err := tx.Exec(ctx, `DELETE FROM craftsky_profiles WHERE did=$1`, after.Owner)
			return err
		})
	if err != nil {
		t.Fatal(err)
	}
	oauthConfig := oauth.NewPublicConfig("https://appview.example/oauth/client-metadata.json", "https://appview.example/oauth/callback", []string{"atproto"})
	f.reconciler = reconciler
	app := &oauth.ClientApp{Config: &oauthConfig, Resolver: oauth.NewResolver(),
		Dir: &callbackDirectory{identity: &identity.Identity{DID: f.owner, Handle: "recover.example"}},
		Client: &http.Client{Transport: callbackRoundTripFunc(func(request *http.Request) (*http.Response, error) {
			f.exchanges++
			body, _ := json.Marshal(oauth.TokenResponse{Subject: f.owner.String(), Scope: "atproto", AccessToken: "fresh-access", RefreshToken: "fresh-refresh"})
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body))), Request: request}, nil
		})},
	}
	f.flow, err = auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{App: app, Store: f.store, Owners: f.owners, OnboardingReconciler: reconciler,
		AuthorityVerifier: &fakeOAuthAuthorityVerifier{authority: auth.OAuthAuthority{DID: f.owner, PDSOrigin: "https://pds.example.com", IssuerOrigin: "https://auth.example.com"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	f.handoffs, err = auth.NewHandoffService(auth.HandoffServiceOptions{Pool: f.pool, Owners: f.owners, Sessions: f.children,
		RepositoryJobs: auth.RepositoryJobTxEnqueuerFunc(func(context.Context, pgx.Tx, syntax.DID, auth.RepositoryJobKind) error { return nil }),
		ExchangeTTL:    2 * time.Minute, ConfirmationTTL: time.Minute, ReceiptKey: []byte("0123456789abcdef0123456789abcdef"), ReceiptKeyVersion: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *recoveryFixture) complete(finalize auth.OAuthCallbackFinalizer) error {
	return f.flow.CompleteCallback(context.Background(), url.Values{"state": {f.state}, "iss": {"https://auth.example.com"}, "code": {"single-use-code"}}, finalize)
}

func TestOAuthMissingProfileReconciliationCompletesSameSignIn(t *testing.T) {
	f := newRecoveryFixture(t)
	var code string
	err := f.complete(func(ctx context.Context, result auth.OAuthCallbackResult) error {
		if result.Attempt.OwnerGeneration != 3 || result.Attempt.AuthEpoch != 2 ||
			result.Metadata.OwnerGeneration != 3 || result.Metadata.AuthEpoch != 2 {
			t.Fatalf("unrebound callback: %+v", result.Attempt)
		}
		if _, err := f.store.ResumePendingOnboardingSession(ctx, result.Attempt); err != nil {
			return err
		}
		if err := auth.InitializeProfileAndIdentityCache(ctx, f.pds, result.Attempt, f.writer, nil, nil, nil, nil); err != nil {
			return err
		}
		var err error
		code, err = f.handoffs.CreateExchange(ctx, result.Attempt, result.Handle, "recovery-device")
		return err
	})
	if err != nil {
		t.Fatalf("recover callback: %v", err)
	}
	if f.exchanges != 1 || f.departures != 1 || f.pds.applyCalls != 1 {
		t.Fatalf("exchanges/departures/writes=%d/%d/%d", f.exchanges, f.departures, f.pds.applyCalls)
	}
	var oldParent string
	if err := f.pool.QueryRow(context.Background(), `SELECT lifecycle_state FROM oauth_sessions WHERE session_id='old-parent'`).Scan(&oldParent); err != nil {
		t.Fatal(err)
	}
	if oldParent != "revocation_pending" {
		t.Fatalf("old parent survived: %s", oldParent)
	}
	if _, err := f.children.Lookup(context.Background(), "old-bearer"); !errors.Is(err, auth.ErrCraftskySessionNotFound) {
		t.Fatalf("old bearer survived: %v", err)
	}
	var accepted int
	if err := f.pool.QueryRow(context.Background(), `SELECT count(*) FROM pds_commands WHERE operation_kind='oauth_onboarding_profile' AND state='accepted'`).Scan(&accepted); err != nil {
		t.Fatal(err)
	}
	if accepted != 2 {
		t.Fatalf("accepted journal generations=%d, want old and new", accepted)
	}
	exchange, err := f.handoffs.Exchange(context.Background(), code, "recovery-device")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.owners.Transition(context.Background(), ownerlifecycle.TransitionRequest{Owner: f.owner, ExpectedGeneration: 3, To: ownerlifecycle.StateActive, Reason: "profileActivated"}); err != nil {
		t.Fatal(err)
	}
	if err = f.handoffs.Confirm(context.Background(), exchange.Token, exchange.ReceiptID, "recovery-device"); err != nil {
		t.Fatal(err)
	}
	if _, err = f.children.Lookup(context.Background(), exchange.Token); err != nil {
		t.Fatalf("new bearer rejected: %v", err)
	}
}

func TestOAuthMissingProfileReconciliationReadFailuresDoNotDepart(t *testing.T) {
	for _, readErr := range []error{context.DeadlineExceeded, errors.New("provider permission denied")} {
		t.Run(readErr.Error(), func(t *testing.T) {
			f := newRecoveryFixture(t)
			f.pds.readErr = readErr
			finalized := false
			err := f.complete(func(context.Context, auth.OAuthCallbackResult) error { finalized = true; return nil })
			if !errors.Is(err, readErr) || finalized || f.departures != 0 || f.pds.applyCalls != 0 {
				t.Fatalf("error=%v finalized=%v departures=%d", err, finalized, f.departures)
			}
			authority, err := f.owners.Get(context.Background(), f.owner)
			if err != nil || authority.State != ownerlifecycle.StateActive || authority.Generation != 2 || authority.AuthEpoch != 1 {
				t.Fatalf("authority=%+v error=%v", authority, err)
			}
		})
	}
}

func TestOAuthMissingProfileReconciliationCompensatesReboundCredential(t *testing.T) {
	f := newRecoveryFixture(t)
	want := errors.New("finalization failed after departure")
	err := f.complete(func(context.Context, auth.OAuthCallbackResult) error { return want })
	if !errors.Is(err, want) || errors.Is(err, auth.ErrCallbackAttemptInvalid) {
		t.Fatalf("compensation error=%v", err)
	}
	var state string
	var generation, epoch int64
	if err := f.pool.QueryRow(context.Background(), `SELECT lifecycle_state,owner_generation,auth_epoch FROM oauth_sessions WHERE session_id=$1`, f.state).Scan(&state, &generation, &epoch); err != nil {
		t.Fatal(err)
	}
	if state != "revocation_pending" || generation != 3 || epoch != 2 {
		t.Fatalf("rebound parent=%s/%d/%d", state, generation, epoch)
	}
}

func TestOAuthMissingProfileReconciliationPreservesConcurrentProfile(t *testing.T) {
	f := newRecoveryFixture(t)
	f.pds.concurrentRecord = map[string]any{"$type": "social.craftsky.actor.profile", "crafts": []string{"crochet"}}
	err := f.complete(func(ctx context.Context, result auth.OAuthCallbackResult) error {
		return auth.InitializeProfileAndIdentityCache(ctx, f.pds, result.Attempt, f.writer, nil, nil, nil, nil)
	})
	if err != nil {
		t.Fatalf("concurrent profile: %v", err)
	}
	raw, _ := json.Marshal(f.pds.record)
	if f.pds.applyCalls != 1 || !strings.Contains(string(raw), "crochet") {
		t.Fatalf("concurrent profile overwritten: %s, writes=%d", raw, f.pds.applyCalls)
	}
}

func TestOAuthMissingProfileReconciliationNeverRepeatsUncertainWrite(t *testing.T) {
	f := newRecoveryFixture(t)
	f.pds.writeErr = errors.New("response lost")
	var recovered auth.OAuthCallbackResult
	err := f.complete(func(ctx context.Context, result auth.OAuthCallbackResult) error {
		recovered = result
		return auth.InitializeProfileAndIdentityCache(ctx, f.pds, result.Attempt, f.writer, nil, nil, nil, nil)
	})
	if !errors.Is(err, pdscommands.ErrOnboardingProfileUnresolved) {
		t.Fatalf("uncertain write=%v", err)
	}
	f.pds.writeErr = nil
	err = f.owners.WithExistingAuth(context.Background(), f.owner, func(ctx context.Context, _ ownerlifecycle.Lifecycle) error {
		_, err := f.writer.commands.PutProfile(ctx, f.pds, f.owner, recovered.Attempt.OwnerGeneration, map[string]any{"$type": "social.craftsky.actor.profile", "crafts": []string{}})
		return err
	})
	if !errors.Is(err, pdscommands.ErrOnboardingProfileUnresolved) || f.pds.applyCalls != 1 {
		t.Fatalf("repeated uncertain write: %v writes=%d", err, f.pds.applyCalls)
	}
}

func TestOAuthMissingProfileReconciliationLeavesPresentProfileAndSessionsIntact(t *testing.T) {
	f := newRecoveryFixture(t)
	f.pds.record = map[string]any{"$type": "social.craftsky.actor.profile", "crafts": []string{"knitting"}}
	err := f.complete(func(ctx context.Context, result auth.OAuthCallbackResult) error {
		if result.Attempt.OwnerGeneration != 2 || result.Attempt.AuthEpoch != 1 {
			t.Fatalf("present profile callback rebound: %+v", result.Attempt)
		}
		return auth.InitializeProfileAndIdentityCache(ctx, f.pds, result.Attempt, f.writer, nil, nil, nil, nil)
	})
	if err != nil || f.departures != 0 || f.pds.applyCalls != 0 {
		t.Fatalf("present profile recovery=%v departures=%d writes=%d", err, f.departures, f.pds.applyCalls)
	}
	if _, err := f.children.Lookup(context.Background(), "old-bearer"); err != nil {
		t.Fatalf("existing session revoked: %v", err)
	}
}

func TestOAuthMissingProfileReconciliationRollsBackOnParticipantFailure(t *testing.T) {
	f := newRecoveryFixture(t)
	f.removeErr = errors.New("membership cleanup unavailable")
	err := f.complete(func(context.Context, auth.OAuthCallbackResult) error {
		t.Fatal("failed reconciliation reached finalization")
		return nil
	})
	if !errors.Is(err, f.removeErr) || errors.Is(err, auth.ErrCallbackAttemptInvalid) {
		t.Fatalf("rollback error=%v", err)
	}
	authority, err := f.owners.Get(context.Background(), f.owner)
	if err != nil || authority.State != ownerlifecycle.StateActive || authority.Generation != 2 || authority.AuthEpoch != 1 {
		t.Fatalf("partially committed departure: %+v, %v", authority, err)
	}
	if _, err := f.children.Lookup(context.Background(), "old-bearer"); err != nil {
		t.Fatalf("rollback revoked old bearer: %v", err)
	}
}

func TestOAuthMissingProfileReconciliationRejectsStaleAttemptAfterRebinding(t *testing.T) {
	f := newRecoveryFixture(t)
	err := f.complete(func(ctx context.Context, result auth.OAuthCallbackResult) error {
		result.Attempt.OwnerGeneration--
		result.Attempt.AuthEpoch--
		_, _, err := f.reconciler.Reconcile(auth.WithCallbackAttempt(ctx, result.Attempt), result)
		if !errors.Is(err, auth.ErrCallbackAttemptInvalid) {
			t.Fatalf("stale attempt accepted: %v", err)
		}
		return nil
	})
	if err != nil || f.departures != 1 {
		t.Fatalf("stale attempt changed recovery: %v, departures=%d", err, f.departures)
	}
}

func TestOAuthMissingProfileReconciliationExcludesDeletionAndTerminalOwners(t *testing.T) {
	for _, state := range []ownerlifecycle.State{ownerlifecycle.StateDeletionPending, ownerlifecycle.StateDeleting, ownerlifecycle.StateTerminal} {
		t.Run(string(state), func(t *testing.T) {
			f := newRecoveryFixture(t)
			_, err := f.owners.Transition(context.Background(), ownerlifecycle.TransitionRequest{
				Owner: f.owner, ExpectedGeneration: 2, To: ownerlifecycle.StateDeletionPending, Reason: "deletionIntent",
			})
			if err != nil {
				t.Fatal(err)
			}
			if state == ownerlifecycle.StateDeleting {
				_, err = f.owners.Transition(context.Background(), ownerlifecycle.TransitionRequest{Owner: f.owner, ExpectedGeneration: 3, To: state, Reason: "deletionAccepted"})
			}
			if state == ownerlifecycle.StateTerminal {
				_, err = f.owners.Terminalize(context.Background(), ownerlifecycle.TerminalizeRequest{Owner: f.owner, Reason: "identityDeleted"})
			}
			if err != nil {
				t.Fatal(err)
			}
			err = f.complete(func(context.Context, auth.OAuthCallbackResult) error {
				t.Fatal("ineligible callback finalized")
				return nil
			})
			if err == nil || f.departures != 0 || f.exchanges != 0 || f.pds.applyCalls != 0 {
				t.Fatalf("ineligible recovery=%v exchanges=%d departures=%d writes=%d", err, f.exchanges, f.departures, f.pds.applyCalls)
			}
		})
	}
}
