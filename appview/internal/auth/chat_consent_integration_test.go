package auth_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"social.craftsky/appview/internal/auth"
)

// IT-015 / FR-001, RULE-004, NFR-001 / AC-002, AC-019.
func TestChatConsentHandoffEnablesOnlyAfterConfirmation(t *testing.T) {
	f := newChatConsentFixture(t)
	if _, err := f.pool.Exec(t.Context(), `INSERT INTO chat_sync_state(owner_did,phase,cursor,safe_state,next_due_at,failure_count) VALUES($1,'ready','saved-consent-cursor','authorizationNeeded',now()+interval '5 minutes',2)`, f.owner); err != nil {
		t.Fatal(err)
	}
	var code string
	err := f.complete(func(ctx context.Context, result auth.OAuthCallbackResult) error {
		var err error
		code, err = f.handoffs.CreateExchange(ctx, result.Attempt, result.Handle, "recovery-device")
		return err
	})
	if err != nil {
		t.Fatalf("chat callback: %v", err)
	}
	if f.departures != 0 || f.pds.applyCalls != 0 {
		t.Fatal("chat consent ran profile onboarding")
	}
	var enabled bool
	if err := f.pool.QueryRow(t.Context(), `SELECT enabled FROM chat_accounts WHERE owner_did=$1`, f.owner).Scan(&enabled); err != nil || enabled {
		t.Fatalf("enabled before confirmation: %t %v", enabled, err)
	}
	exchange, err := f.handoffs.Exchange(t.Context(), code, "recovery-device")
	if err != nil {
		t.Fatal(err)
	}
	source := sha256.Sum256([]byte("old-bearer"))
	if exchange.Purpose != auth.ChatOAuthPurpose || exchange.SourceTokenHash != base64.RawURLEncoding.EncodeToString(source[:]) {
		t.Fatal("chat handoff does not identify the exact retained source session")
	}
	if err := f.handoffs.Confirm(t.Context(), exchange.Token, exchange.ReceiptID, "recovery-device"); err != nil {
		t.Fatal(err)
	}
	var push bool
	var generation int64
	if err := f.pool.QueryRow(t.Context(), `SELECT enabled,generation,push_enabled FROM chat_accounts WHERE owner_did=$1`, f.owner).Scan(&enabled, &generation, &push); err != nil || !enabled || generation != 2 || push {
		t.Fatalf("confirmed connection %t/%d/%t: %v", enabled, generation, push, err)
	}
	if f.repositoryJobs != 0 {
		t.Fatal("optional consent queued unrelated repository onboarding work")
	}
	var resumed bool
	if err := f.pool.QueryRow(t.Context(), `SELECT cursor='saved-consent-cursor' AND phase='ready' AND failure_count=0 AND recovery_id IS NOT NULL AND next_due_at<=now() FROM chat_sync_state WHERE owner_did=$1`, f.owner).Scan(&resumed); err != nil || !resumed {
		t.Fatalf("confirmed consent did not resume retained progress: %v", err)
	}
	if _, err := f.children.Lookup(t.Context(), "old-bearer"); err != nil {
		t.Fatalf("base child lost: %v", err)
	}
	if err := f.handoffs.Confirm(t.Context(), exchange.Token, exchange.ReceiptID, "recovery-device"); err != nil {
		t.Fatalf("repeat confirmation: %v", err)
	}
}

// IT-015 / RULE-004: missing purpose metadata cannot turn optional consent
// into an ordinary login that bypasses the initiating-session binding.
func TestChatHandoffFailsClosedIfPurposeMetadataDisappears(t *testing.T) {
	for _, afterExchange := range []bool{false, true} {
		t.Run(map[bool]string{false: "exchange", true: "confirmation"}[afterExchange], func(t *testing.T) {
			f := newChatConsentFixture(t)
			var code string
			if err := f.complete(func(ctx context.Context, result auth.OAuthCallbackResult) error {
				var err error
				code, err = f.handoffs.CreateExchange(ctx, result.Attempt, result.Handle, "recovery-device")
				return err
			}); err != nil {
				t.Fatal(err)
			}
			var exchange auth.HandoffExchangeResult
			if afterExchange {
				var err error
				exchange, err = f.handoffs.Exchange(t.Context(), code, "recovery-device")
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, err := f.pool.Exec(t.Context(), `DELETE FROM oauth_auth_requests WHERE state=$1`, f.state); err != nil {
				t.Fatal(err)
			}
			var err error
			if afterExchange {
				err = f.handoffs.Confirm(t.Context(), exchange.Token, exchange.ReceiptID, "recovery-device")
			} else {
				_, err = f.handoffs.Exchange(t.Context(), code, "recovery-device")
			}
			if !errors.Is(err, auth.ErrHandoffInvalid) {
				t.Fatal("missing purpose binding was accepted")
			}
		})
	}
}

// IT-015 / FR-001, RULE-004: PAR requests the fixed bundle and binds the
// authenticated child; it never accepts an arbitrary owner from the browser.
func TestChatConsentStartBindsAuthenticatedChild(t *testing.T) {
	f := newRecoveryFixture(t)
	if _, err := f.pool.Exec(t.Context(), `UPDATE craftsky_sessions SET last_device_id='recovery-device' WHERE account_did=$1`, f.owner); err != nil {
		t.Fatal(err)
	}
	var requested url.Values
	client := &http.Client{Transport: callbackRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{}`
		status := http.StatusOK
		switch r.URL.Path {
		case "/.well-known/oauth-protected-resource":
			body = `{"authorization_servers":["https://auth.example.com"]}`
		case "/.well-known/oauth-authorization-server":
			body = `{"issuer":"https://auth.example.com","authorization_endpoint":"https://auth.example.com/authorize","token_endpoint":"https://auth.example.com/token","pushed_authorization_request_endpoint":"https://auth.example.com/par","revocation_endpoint":"https://auth.example.com/revoke","response_types_supported":["code"],"grant_types_supported":["authorization_code","refresh_token"],"code_challenge_methods_supported":["S256"],"token_endpoint_auth_methods_supported":["none","private_key_jwt"],"token_endpoint_auth_signing_alg_values_supported":["ES256"],"scopes_supported":["atproto"],"dpop_signing_alg_values_supported":["ES256"],"authorization_response_iss_parameter_supported":true,"require_pushed_authorization_requests":true,"client_id_metadata_document_supported":true}`
		case "/par":
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			requested, err = url.ParseQuery(string(raw))
			if err != nil {
				return nil, err
			}
			body = `{"request_uri":"urn:synthetic:chat","expires_in":600}`
			status = http.StatusCreated
		default:
			return nil, errors.New("unexpected request")
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	cfg := oauth.NewPublicConfig("https://appview.example/oauth-client-metadata.json", "https://appview.example/oauth/callback", []string{"atproto"})
	dir := &callbackDirectory{identity: &identity.Identity{DID: f.owner, Handle: "recover.example", Services: map[string]identity.ServiceEndpoint{"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: "https://pds.example.com"}}}}
	flow, err := auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{App: &oauth.ClientApp{Config: &cfg, Client: client, Resolver: &oauth.Resolver{Client: client}, Dir: dir}, Store: f.store, Owners: f.owners, AuthorityVerifier: &fakeOAuthAuthorityVerifier{authority: auth.OAuthAuthority{DID: f.owner, PDSOrigin: "https://pds.example.com", IssuerOrigin: "https://auth.example.com"}}})
	if err != nil {
		t.Fatal(err)
	}
	starter, ok := any(flow).(interface {
		StartChatAuthorization(context.Context, auth.AuthInfo, string, auth.HandoffMode, string, string, *bool) (string, error)
	})
	if !ok {
		t.Fatal("optional messaging consent has no authenticated admission boundary")
	}
	push := false
	redirect, err := starter.StartChatAuthorization(t.Context(), auth.AuthInfo{DID: f.owner, SessionID: "old-parent"}, "old-bearer", auth.HandoffVerifiedLink, "", "recovery-device", &push)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(redirect, "https://auth.example.com/authorize?") {
		t.Fatal("invalid authorization redirect")
	}
	if requested.Get("login_hint") != f.owner.String() {
		t.Fatal("PAR not bound to owner DID")
	}
	for _, scope := range append([]string{"atproto"}, auth.OptionalChatScopes()...) {
		if !strings.Contains(" "+requested.Get("scope")+" ", " "+scope+" ") {
			t.Fatal("missing fixed scope")
		}
	}
	metadata, err := f.store.LoadAuthRequestMetadata(t.Context(), requested.Get("state"))
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte("old-bearer"))
	if metadata.Purpose != auth.ChatOAuthPurpose || metadata.ChatParentSessionID != "old-parent" || string(metadata.ChatChildTokenHash) != string(hash[:]) || metadata.ChatGeneration != 1 || metadata.ChatPushEnabled == nil || *metadata.ChatPushEnabled {
		t.Fatal("consent authority not durably captured")
	}
	var raw []byte
	if err := f.pool.QueryRow(t.Context(), `SELECT data FROM oauth_auth_requests WHERE state=$1`, requested.Get("state")).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var request oauth.AuthRequestData
	if json.Unmarshal(raw, &request) != nil || len(request.Scopes) != len(auth.OptionalChatScopes())+1 {
		t.Fatal("requested profile not durable")
	}
	if _, err := starter.StartChatAuthorization(t.Context(), auth.AuthInfo{DID: f.owner, SessionID: "old-parent"}, "wrong-child", auth.HandoffVerifiedLink, "", "recovery-device", nil); err == nil {
		t.Fatal("wrong child authorized")
	}
}

func newChatConsentFixture(t *testing.T) *recoveryFixture {
	t.Helper()
	f := newRecoveryFixture(t, auth.OptionalChatScopes()...)
	hash := sha256.Sum256([]byte("old-bearer"))
	if _, err := f.pool.Exec(t.Context(), `UPDATE craftsky_sessions SET last_device_id='recovery-device' WHERE token_hash=$1`, hash[:]); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(t.Context(), `INSERT INTO chat_accounts(owner_did) VALUES($1)`, f.owner); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(t.Context(), `UPDATE oauth_auth_requests SET purpose='chat',
		chat_parent_session_id='old-parent',chat_child_token_hash=$2,chat_generation=1,chat_push_enabled=false WHERE state=$1`, f.state, hash[:]); err != nil {
		t.Fatal(err)
	}
	return f
}

// IT-015 / FR-001, FR-016, RULE-004: stale consent cannot regain authority.
func TestChatConsentRejectsStaleInitiatingAuthority(t *testing.T) {
	for _, test := range []struct{ name, sql string }{
		{"disconnect", `UPDATE chat_accounts SET generation=generation+1,enabled=false WHERE owner_did=$1`},
		{"child logout", `UPDATE craftsky_sessions SET lifecycle_state='revoked',revoked_at=now() WHERE account_did=$1`},
		{"device changed", `UPDATE craftsky_sessions SET last_device_id='other-device' WHERE account_did=$1`},
		{"idle expiry", `UPDATE craftsky_sessions SET created_at=now()-interval '2 days',last_seen_at=now()-interval '1 day',idle_expires_at=now()-interval '1 minute' WHERE account_did=$1`},
		{"consent expiry", `UPDATE oauth_auth_requests SET created_at=now()-interval '1 hour' WHERE owner_did=$1`},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newChatConsentFixture(t)
			if _, err := f.pool.Exec(t.Context(), test.sql, f.owner); err != nil {
				t.Fatal(err)
			}
			err := f.complete(func(context.Context, auth.OAuthCallbackResult) error { t.Fatal("stale callback finalized"); return nil })
			if !errors.Is(err, auth.ErrOAuthFlowInvalid) || f.exchanges != 0 {
				t.Fatalf("stale consent exchanged %d tokens; err=%v", f.exchanges, err)
			}
		})
	}
}

// IT-015: a superseded consent is a definite invalid receipt, so the client
// can discard it instead of retrying a transient server failure forever.
func TestChatConsentStaleGenerationConfirmationIsInvalidHandoff(t *testing.T) {
	f := newChatConsentFixture(t)
	var code string
	if err := f.complete(func(ctx context.Context, result auth.OAuthCallbackResult) error {
		var err error
		code, err = f.handoffs.CreateExchange(ctx, result.Attempt, result.Handle, "recovery-device")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	exchange, err := f.handoffs.Exchange(t.Context(), code, "recovery-device")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(t.Context(), `UPDATE chat_accounts SET generation=generation+1,enabled=false WHERE owner_did=$1`, f.owner); err != nil {
		t.Fatal(err)
	}
	if err := f.handoffs.Confirm(t.Context(), exchange.Token, exchange.ReceiptID, "recovery-device"); !errors.Is(err, auth.ErrHandoffInvalid) {
		t.Fatalf("stale chat confirmation error type=%T classification=%v", err, errors.Is(err, auth.ErrHandoffInvalid))
	}
	var enabled bool
	if err := f.pool.QueryRow(t.Context(), `SELECT enabled FROM chat_accounts WHERE owner_did=$1`, f.owner).Scan(&enabled); err != nil || enabled {
		t.Fatal("stale consent re-enabled messaging")
	}
	if _, err := f.children.Lookup(t.Context(), "old-bearer"); err != nil {
		t.Fatal("stale optional consent damaged base access")
	}
}
