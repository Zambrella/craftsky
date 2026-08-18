package auth_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

func TestHandoffExchangeIsHashOnlyReplayableUntilConfirmation(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	storeConfig := testStoreConfig()
	storeConfig.OwnerLifecycles = owners
	oauthStore := auth.NewPostgresAuthStore(pool, storeConfig)
	children, err := auth.NewCraftskySessionStoreWithConfig(pool, auth.CraftskySessionConfig{
		Inactivity: 30 * 24 * time.Hour, ActivityWriteInterval: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	handoffs, err := auth.NewHandoffService(auth.HandoffServiceOptions{
		Pool: pool, Owners: owners, Sessions: children,
		ExchangeTTL: 5 * time.Minute, ConfirmationTTL: 2 * time.Minute,
		ReceiptKey: []byte("0123456789abcdef0123456789abcdef"), ReceiptKeyVersion: 1,
		Now: time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}

	owner := syntax.DID("did:plc:handoff-owner")
	state := "handoff-parent-state"
	var code string
	err = owners.WithOnboardingAuth(context.Background(), owner, func(authCtx context.Context, authority ownerlifecycle.Lifecycle) error {
		requestContext := auth.WithLoginAuthRequest(
			authCtx, owner, authority.Generation, authority.AuthEpoch,
			auth.HandoffVerifiedLink, "device-handoff", "",
		)
		request := oauth.AuthRequestData{
			State: state, RequestURI: "urn:request:handoff-parent", AuthServerURL: "https://auth.example.com",
		}
		if err := oauthStore.SaveAuthRequestInfo(requestContext, request); err != nil {
			return err
		}
		attemptID, err := oauthStore.BeginExchange(authCtx, state)
		if err != nil {
			return err
		}
		attempt := auth.CallbackAttempt{
			State: state, AttemptID: attemptID, Owner: owner,
			OwnerGeneration: authority.Generation, AuthEpoch: authority.AuthEpoch,
			Purpose: auth.LoginOAuthPurpose,
		}
		callbackContext := auth.WithCallbackAttempt(authCtx, attempt)
		if err := oauthStore.SaveSession(callbackContext, validOAuthSession(owner, state)); err != nil {
			return err
		}
		code, err = handoffs.CreateExchange(callbackContext, attempt, syntax.Handle("alice.example"), "device-handoff")
		return err
	})
	if err != nil {
		t.Fatalf("finalize callback handoff: %v", err)
	}
	if code == "" {
		t.Fatal("empty handoff code")
	}
	var rawCodeCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM oauth_handoff_exchanges WHERE encode(code_hash,'hex')=$1
	`, code).Scan(&rawCodeCount); err != nil {
		t.Fatal(err)
	}
	if rawCodeCount != 0 {
		t.Fatal("raw handoff code was durably persisted")
	}

	if _, err := handoffs.Exchange(context.Background(), code, "wrong-device"); !errors.Is(err, auth.ErrHandoffInvalid) {
		t.Fatalf("wrong-device exchange = %v", err)
	}
	first, err := handoffs.Exchange(context.Background(), code, "device-handoff")
	if err != nil {
		t.Fatalf("first exchange: %v", err)
	}
	second, err := handoffs.Exchange(context.Background(), code, "device-handoff")
	if err != nil {
		t.Fatalf("replayed exchange: %v", err)
	}
	if first != second || first.Token == "" || first.ReceiptID == uuid.Nil || first.DID != owner || first.Handle != "alice.example" {
		t.Fatalf("replayed handoff mismatch: first=%+v second=%+v", first, second)
	}
	var childrenCount, receiptsCount int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM craftsky_sessions`).Scan(&childrenCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM oauth_handoff_receipts`).Scan(&receiptsCount); err != nil {
		t.Fatal(err)
	}
	if childrenCount != 1 || receiptsCount != 1 {
		t.Fatalf("handoff replay created children=%d receipts=%d", childrenCount, receiptsCount)
	}
	if _, err := children.Lookup(context.Background(), first.Token); !errors.Is(err, auth.ErrCraftskySessionNotFound) {
		t.Fatalf("pending child authenticated before confirmation: %v", err)
	}

	if _, err := owners.Transition(context.Background(), ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: 1, To: ownerlifecycle.StateActive, Reason: "profileCreated",
	}); err != nil {
		t.Fatal(err)
	}
	if err := handoffs.Confirm(context.Background(), first.Token, first.ReceiptID, "device-handoff"); err != nil {
		t.Fatalf("confirm handoff: %v", err)
	}
	if err := handoffs.Confirm(context.Background(), first.Token, first.ReceiptID, "device-handoff"); err != nil {
		t.Fatalf("idempotent confirm: %v", err)
	}
	info, err := children.Lookup(context.Background(), first.Token)
	if err != nil || info.DID != owner || info.SessionID != state {
		t.Fatalf("confirmed child lookup = %+v, %v", info, err)
	}
	var codeHash, ciphertext, nonce []byte
	if err := pool.QueryRow(context.Background(), `
		SELECT exchange.code_hash,receipt.ciphertext,receipt.nonce
		FROM oauth_handoff_exchanges exchange
		JOIN oauth_handoff_receipts receipt ON receipt.exchange_id=exchange.id
	`).Scan(&codeHash, &ciphertext, &nonce); err != nil {
		t.Fatal(err)
	}
	if codeHash != nil || ciphertext != nil || nonce != nil {
		t.Fatal("confirmation retained recoverable code or receipt secrets")
	}
	tokenHash := sha256.Sum256([]byte(first.Token))
	var active bool
	if err := pool.QueryRow(context.Background(), `
		SELECT child.lifecycle_state='active' AND parent.lifecycle_state='active'
		FROM craftsky_sessions child
		JOIN oauth_sessions parent ON parent.account_did=child.account_did AND parent.session_id=child.oauth_session_id
		WHERE child.token_hash=$1
	`, tokenHash[:]).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if !active {
		t.Fatal("confirmation did not atomically activate parent and child")
	}
}

func newAuthOwnerStore(t *testing.T, pool *pgxpool.Pool) *ownerlifecycle.Store {
	t.Helper()
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	store, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	return store
}
