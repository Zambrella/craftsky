package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/testdb"
)

const pdsMigrationAuthSchema = `
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
	CREATE TABLE oauth_sessions (
		account_did TEXT NOT NULL,
		session_id TEXT NOT NULL,
		data JSONB NOT NULL,
		lifecycle_state TEXT NOT NULL,
		owner_generation BIGINT NOT NULL,
		auth_epoch BIGINT NOT NULL,
		row_version BIGINT NOT NULL,
		absolute_expires_at TIMESTAMPTZ NOT NULL,
		revocation_requested_at TIMESTAMPTZ,
		cleanup_next_attempt_at TIMESTAMPTZ,
		cleanup_lease_token UUID,
		cleanup_lease_expires_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL,
		PRIMARY KEY (account_did, session_id)
	);
	CREATE TABLE craftsky_sessions (
		token_hash BYTEA PRIMARY KEY,
		account_did TEXT NOT NULL,
		oauth_session_id TEXT NOT NULL,
		lifecycle_state TEXT NOT NULL,
		auth_epoch BIGINT NOT NULL,
		last_seen_at TIMESTAMPTZ NOT NULL,
		idle_expires_at TIMESTAMPTZ NOT NULL,
		revoked_at TIMESTAMPTZ
	);
`

type pdsMigrationAuthService struct {
	owner     syntax.DID
	sessionID string
}

func (service pdsMigrationAuthService) Authenticate(context.Context, string) (auth.AuthInfo, error) {
	return auth.AuthInfo{DID: service.owner, SessionID: service.sessionID}, nil
}

type currentPDSMigrationMember struct{}

func (currentPDSMigrationMember) IsCurrentMember(context.Context, syntax.DID) (bool, error) {
	return true, nil
}

type pdsMigrationEndpointValidator struct{}

func (pdsMigrationEndpointValidator) ValidateOrigin(_ context.Context, raw string) (*url.URL, error) {
	return url.Parse(raw)
}

func (pdsMigrationEndpointValidator) ValidateOAuthEndpoint(_ context.Context, _, raw string) (*url.URL, error) {
	return url.Parse(raw)
}

type pdsMigrationAuthorityVerifier struct {
	current auth.OAuthAuthority
	err     error
}

func (verifier pdsMigrationAuthorityVerifier) ResolveCurrent(context.Context, syntax.DID) (auth.OAuthAuthority, error) {
	return verifier.current, verifier.err
}

type coordinatorProfileEffects struct {
	coordinator     *auth.OAuthSessionCoordinator
	owner           syntax.DID
	sessionID       string
	requestID       string
	operationCalled bool
}

func (effects *coordinatorProfileEffects) ResolveExpectedOwners(
	_ context.Context,
	ownerGeneration int64,
	_ []syntax.DID,
) ([]ownerlifecycle.ExpectedOwner, error) {
	return []ownerlifecycle.ExpectedOwner{{Owner: effects.owner, Generation: ownerGeneration}}, nil
}

func (effects *coordinatorProfileEffects) ReadRecord(
	ctx context.Context,
	request pdseffects.ReadRecordRequest,
	_ any,
) (syntax.CID, error) {
	effects.requestID = middleware.GetRunID(ctx)
	err := effects.coordinator.WithActiveEffectSession(
		ctx,
		request.ExpectedOwners,
		effects.owner,
		effects.sessionID,
		func(context.Context, *oauth.ClientSession) error {
			effects.operationCalled = true
			return nil
		},
	)
	return "", err
}

func (*coordinatorProfileEffects) PutRecord(context.Context, pdseffects.PutRecordRequest) (pdseffects.RecordResult, error) {
	return pdseffects.RecordResult{}, errors.New("unexpected profile write")
}

func (*coordinatorProfileEffects) DeleteRecord(context.Context, pdseffects.DeleteRecordRequest) (pdseffects.RecordResult, error) {
	return pdseffects.RecordResult{}, errors.New("unexpected profile delete")
}

func (*coordinatorProfileEffects) UploadBlob(context.Context, pdseffects.UploadBlobRequest) (*auth.UploadedBlob, error) {
	return nil, errors.New("unexpected profile blob upload")
}

func TestPutProfileMapsOnlyConfirmedStaleAuthorityToExpiredSession(t *testing.T) {
	pool := testdb.WithSchema(t, pdsMigrationAuthSchema)
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	owners, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	store := auth.NewPostgresAuthStore(pool, auth.StoreConfig{
		EndpointValidator: pdsMigrationEndpointValidator{},
	})
	config := oauth.NewPublicConfig(
		"https://appview.example/oauth/client-metadata.json",
		"https://appview.example/oauth/callback",
		[]string{"atproto"},
	)

	seedParent := func(owner syntax.DID, sessionID string, tokenHash []byte, pds, issuer string) {
		t.Helper()
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO owner_lifecycles(
				owner_did,state,generation,auth_epoch,transition_reason,
				transitioned_at,created_at,updated_at
			) VALUES($1,'active',1,1,'test',now(),now(),now())
			ON CONFLICT (owner_did) DO NOTHING
		`, owner); err != nil {
			t.Fatal(err)
		}
		data := oauth.ClientSessionData{
			AccountDID: owner, SessionID: sessionID, HostURL: pds, AuthServerURL: issuer,
			AuthServerTokenEndpoint:      issuer + "/oauth/token",
			AuthServerRevocationEndpoint: issuer + "/oauth/revoke",
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO oauth_sessions(
				account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
				row_version,absolute_expires_at,created_at,updated_at
			) VALUES($1,$2,$3,'active',1,1,1,now()+interval '1 day',now(),now())
		`, owner, sessionID, data); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO craftsky_sessions(
				token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,
				last_seen_at,idle_expires_at
			) VALUES($1,$2,$3,'active',1,now(),now()+interval '1 day')
		`, tokenHash, owner, sessionID); err != nil {
			t.Fatal(err)
		}
	}

	owner := syntax.DID("did:plc:migrating")
	seedParent(owner, "stale-parent", []byte("stale-child"), "https://pds-a.example", "https://issuer-a.example")
	seedParent(owner, "current-parent", []byte("current-child"), "https://pds-b.example", "https://issuer-b.example")

	newHandler := func(sessionID string, verifier pdsMigrationAuthorityVerifier) (http.Handler, *coordinatorProfileEffects) {
		coordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
			App:   &oauth.ClientApp{Client: http.DefaultClient, Config: &config},
			Store: store, Owners: owners, AuthorityVerifier: verifier,
			OperationTimeout: 5 * time.Second,
		})
		if err != nil {
			t.Fatal(err)
		}
		effects := &coordinatorProfileEffects{coordinator: coordinator, owner: owner, sessionID: sessionID}
		handler := api.PutMeProfileHandler(
			&fakeStore{}, fakeResolver{},
			func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
				return effects, nil
			},
			api.DefaultMediaLimits(), nilLogger(),
		)
		handler = middleware.CurrentMember(currentPDSMigrationMember{}, nilLogger(), owners)(handler)
		handler = middleware.Authenticated(
			pdsMigrationAuthService{owner: owner, sessionID: sessionID},
			nilLogger(), middleware.DevAuthPolicy{Mode: middleware.DevAuthDisabled},
		)(handler)
		return middleware.Logging(nilLogger())(handler), effects
	}

	t.Run("confirmed stale authority", func(t *testing.T) {
		handler, effects := newHandler("stale-parent", pdsMigrationAuthorityVerifier{current: auth.OAuthAuthority{
			DID: owner, PDSOrigin: "https://pds-b.example", IssuerOrigin: "https://issuer-b.example",
		}})
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(`{"displayName":"Migrated"}`))
		request.Header.Set("Authorization", "Bearer stale-child")

		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if len(body) != 3 || body["error"] != "pds_session_expired" || body["message"] == "" ||
			body["requestId"] == "" || body["requestId"] != effects.requestID {
			t.Fatalf("error envelope = %#v, context request ID = %q", body, effects.requestID)
		}
		if effects.operationCalled {
			t.Fatal("credential-bearing profile effect ran after stale authority was confirmed")
		}
		var staleParent, staleChild, currentParent, currentChild string
		if err := pool.QueryRow(context.Background(), `
			SELECT
				(SELECT lifecycle_state FROM oauth_sessions WHERE account_did=$1 AND session_id='stale-parent'),
				(SELECT lifecycle_state FROM craftsky_sessions WHERE token_hash=$2),
				(SELECT lifecycle_state FROM oauth_sessions WHERE account_did=$1 AND session_id='current-parent'),
				(SELECT lifecycle_state FROM craftsky_sessions WHERE token_hash=$3)
		`, owner, []byte("stale-child"), []byte("current-child")).Scan(
			&staleParent, &staleChild, &currentParent, &currentChild,
		); err != nil {
			t.Fatal(err)
		}
		if staleParent != "revocation_pending" || staleChild != "revoked" ||
			currentParent != "active" || currentChild != "active" {
			t.Fatalf("session states = %q/%q, %q/%q", staleParent, staleChild, currentParent, currentChild)
		}
	})

	t.Run("transient authority failure", func(t *testing.T) {
		transientErr := errors.New("DID resolution timed out")
		handler, effects := newHandler("current-parent", pdsMigrationAuthorityVerifier{err: transientErr})
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(`{"displayName":"Retry"}`))
		request.Header.Set("Authorization", "Bearer current-child")

		handler.ServeHTTP(recorder, request)

		var body envelope.Error
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if recorder.Code == http.StatusUnauthorized || body.Error == "pds_session_expired" {
			t.Fatalf("transient failure mapped to expired session: status=%d body=%+v", recorder.Code, body)
		}
		if effects.operationCalled {
			t.Fatal("profile effect ran while authority was unverified")
		}
		var parentState string
		if err := pool.QueryRow(context.Background(), `
			SELECT lifecycle_state FROM oauth_sessions
			WHERE account_did=$1 AND session_id='current-parent'
		`, owner).Scan(&parentState); err != nil {
			t.Fatal(err)
		}
		if parentState != "active" {
			t.Fatalf("transient authority failure changed parent state to %q", parentState)
		}
	})
}
