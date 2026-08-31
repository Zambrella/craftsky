package auth_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
)

func TestAuthRequestAdmissionSerializesTheGlobalPendingCapacity(t *testing.T) {
	pool := withAuthSchema(t)
	config := testStoreConfig()
	config.PendingAuthRequestCapacity = 3
	store := auth.NewPostgresAuthStore(pool, config)

	const attempts = 12
	owners := make([]syntax.DID, attempts)
	for index := range owners {
		owners[index] = syntax.DID(fmt.Sprintf("did:plc:capacity-%02d", index))
		seedAuthOwner(t, pool, owners[index])
	}

	start := make(chan struct{})
	results := make(chan error, attempts)
	var workers sync.WaitGroup
	for index, owner := range owners {
		workers.Add(1)
		go func(index int, owner syntax.DID) {
			defer workers.Done()
			<-start
			results <- saveAdmissionAuthRequest(store, owner, index)
		}(index, owner)
	}
	close(start)
	workers.Wait()
	close(results)

	var admitted, rejected int
	for err := range results {
		switch {
		case err == nil:
			admitted++
		case errors.Is(err, auth.ErrAuthRequestCapacity):
			rejected++
		default:
			t.Fatalf("unexpected SaveAuthRequestInfo error: %v", err)
		}
	}
	if admitted != config.PendingAuthRequestCapacity || rejected != attempts-config.PendingAuthRequestCapacity {
		t.Fatalf("admitted=%d rejected=%d, want %d/%d", admitted, rejected, config.PendingAuthRequestCapacity, attempts-config.PendingAuthRequestCapacity)
	}

	var pending int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM oauth_auth_requests
		WHERE request_state IN ('ready','exchange_started','exchange_ambiguous')
	`).Scan(&pending); err != nil {
		t.Fatal(err)
	}
	if pending != config.PendingAuthRequestCapacity {
		t.Fatalf("pending rows=%d, want hard capacity %d", pending, config.PendingAuthRequestCapacity)
	}
}

func TestRegistrationReservationSharesHardCapacityWithEveryAuthRequestPurpose(t *testing.T) {
	pool := withAuthSchema(t)
	config := testStoreConfig()
	config.PendingAuthRequestCapacity = 1
	store := auth.NewPostgresAuthStore(pool, config)
	ctx := context.Background()

	reservation, err := store.ReserveAuthRequestCapacity(ctx)
	if err != nil {
		t.Fatalf("reserve registration capacity before provider work: %v", err)
	}
	var reservations int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM oauth_auth_request_reservations`).Scan(&reservations); err != nil {
		t.Fatalf("count durable reservations: %v", err)
	}
	if reservations != 1 {
		t.Fatalf("durable reservations=%d, want 1 before provider work", reservations)
	}

	loginOwner := syntax.DID("did:plc:reservation-login")
	deletionOwner := syntax.DID("did:plc:reservation-deletion")
	seedAuthOwner(t, pool, loginOwner)
	seedOwnerLifecycle(t, pool, deletionOwner, "deletion_pending")
	if err := saveAdmissionAuthRequest(store, loginOwner, 90); !errors.Is(err, auth.ErrAuthRequestCapacity) {
		t.Fatalf("login admission error=%v, want capacity exhausted", err)
	}
	deletionCtx := auth.WithAccountDeletionAuthRequestAuthority(
		ctx, deletionOwner, 1, 1,
		uuid.MustParse("10000000-0000-4000-8000-000000000006"), "device-reservation-deletion",
	)
	if err := store.SaveAuthRequestInfo(deletionCtx, oauth.AuthRequestData{
		State: "reservation-deletion", RequestURI: "urn:request:reservation-deletion",
	}); !errors.Is(err, auth.ErrAuthRequestCapacity) {
		t.Fatalf("deletion admission error=%v, want capacity exhausted", err)
	}
	if _, err := store.ReserveAuthRequestCapacity(ctx); !errors.Is(err, auth.ErrAuthRequestCapacity) {
		t.Fatalf("second reservation error=%v, want capacity exhausted", err)
	}

	registrationCtx := auth.WithRegistrationAuthRequest(
		ctx, "https://bsky.social", "https://auth.bsky.app",
		auth.HandoffVerifiedLink, "device-reservation-registration", "",
	)
	if err := store.SaveRegistrationAuthRequest(
		registrationCtx,
		reservation.ID,
		oauth.AuthRequestData{
			State: "reservation-registration", RequestURI: "urn:request:reservation-registration",
		},
	); err != nil {
		t.Fatalf("consume registration reservation: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM oauth_auth_request_reservations`).Scan(&reservations); err != nil {
		t.Fatalf("count consumed reservations: %v", err)
	}
	if reservations != 0 {
		t.Fatalf("durable reservations=%d, want 0 after registration persistence", reservations)
	}
	if err := store.ReleaseAuthRequestCapacity(ctx, reservation.ID); err != nil {
		t.Fatalf("idempotently release consumed reservation: %v", err)
	}
	if err := saveAdmissionAuthRequest(store, loginOwner, 91); !errors.Is(err, auth.ErrAuthRequestCapacity) {
		t.Fatalf("login admission after reservation consumption error=%v, want capacity exhausted", err)
	}
}

func TestAbandonedRegistrationExpiresAndStaleExchangeConvergesAfterRestart(t *testing.T) {
	pool := withAuthSchema(t)
	config := testStoreConfig()
	config.PendingAuthRequestCapacity = 2
	config.AuthRequestReservationExpiry = time.Minute
	config.AuthRequestExchangeExpiry = 2 * time.Minute
	config.AuthRequestExpiry = 30 * time.Minute
	config.AuthRequestTerminalRetention = time.Hour
	storeBeforeRestart := auth.NewPostgresAuthStore(pool, config)
	ctx := context.Background()

	expiredReservation, err := storeBeforeRestart.ReserveAuthRequestCapacity(ctx)
	if err != nil {
		t.Fatalf("reserve abandoned registration: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE oauth_auth_request_reservations
		SET created_at=now()-interval '3 minutes',expires_at=now()-interval '2 minutes'
		WHERE id=$1
	`, expiredReservation.ID); err != nil {
		t.Fatal(err)
	}
	replacementReservation, err := storeBeforeRestart.ReserveAuthRequestCapacity(ctx)
	if err != nil {
		t.Fatalf("expired reservation blocked replacement: %v", err)
	}

	registrationContext := auth.WithRegistrationAuthRequest(
		ctx, "https://bsky.social", "https://auth.bsky.app",
		auth.HandoffVerifiedLink, "device-abandoned-registration", "",
	)
	readyRequest := oauth.AuthRequestData{
		State: "abandoned-registration-ready", RequestURI: "urn:request:abandoned-registration-ready",
	}
	if err := storeBeforeRestart.SaveRegistrationAuthRequest(
		registrationContext, replacementReservation.ID, readyRequest,
	); err != nil {
		t.Fatalf("save registration ready request: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE oauth_auth_requests SET created_at=now()-interval '31 minutes' WHERE state=$1
	`, readyRequest.State); err != nil {
		t.Fatal(err)
	}
	if _, err := storeBeforeRestart.BeginRegistrationExchange(ctx, readyRequest.State); !errors.Is(err, auth.ErrAuthRequestState) {
		t.Fatalf("expired registration begin error=%v, want stale-state rejection", err)
	}

	for _, state := range []string{"stale-registration-held", "stale-registration-lost-token"} {
		reservation, reserveErr := storeBeforeRestart.ReserveAuthRequestCapacity(ctx)
		if reserveErr != nil {
			t.Fatalf("reserve %s: %v", state, reserveErr)
		}
		if saveErr := storeBeforeRestart.SaveRegistrationAuthRequest(
			registrationContext,
			reservation.ID,
			oauth.AuthRequestData{State: state, RequestURI: "urn:request:" + state},
		); saveErr != nil {
			t.Fatalf("save %s: %v", state, saveErr)
		}
		attemptID, beginErr := storeBeforeRestart.BeginRegistrationExchange(ctx, state)
		if beginErr != nil {
			t.Fatalf("begin %s: %v", state, beginErr)
		}
		if state == "stale-registration-held" {
			if _, insertErr := pool.Exec(ctx, `
				INSERT INTO oauth_unverified_credentials(
					request_state,data,status,eligible_at,expires_at
				) VALUES($1,'{}','held',now()+interval '1 hour',now()+interval '2 hours')
			`, state); insertErr != nil {
				t.Fatal(insertErr)
			}
		}
		if _, updateErr := pool.Exec(ctx, `
			UPDATE oauth_auth_requests
			SET exchange_started_at=now()-interval '3 minutes',consumed_at=now()-interval '3 minutes'
			WHERE state=$1 AND exchange_attempt_id=$2
		`, state, attemptID); updateErr != nil {
			t.Fatal(updateErr)
		}
	}

	storeAfterRestart := auth.NewPostgresAuthStore(pool, config)
	stats, err := storeAfterRestart.ReconcileStaleRegistrationExchanges(ctx, 10)
	if err != nil {
		t.Fatalf("reconcile stale registrations after restart: %v", err)
	}
	if stats.CleanupPending != 1 || stats.Ambiguous != 1 {
		t.Fatalf("reconciliation stats=%+v, want one cleanup-pending and one ambiguous", stats)
	}

	var heldRequestState, heldCredentialStatus string
	if err := pool.QueryRow(ctx, `
		SELECT request.request_state,credential.status
		FROM oauth_auth_requests request
		JOIN oauth_unverified_credentials credential ON credential.request_state=request.state
		WHERE request.state='stale-registration-held'
	`).Scan(&heldRequestState, &heldCredentialStatus); err != nil {
		t.Fatal(err)
	}
	if heldRequestState != string(auth.AuthRequestCleanupPending) || heldCredentialStatus != "pending" {
		t.Fatalf("held stale exchange converged to %s/%s, want cleanup_pending/pending", heldRequestState, heldCredentialStatus)
	}
	metadata, err := storeAfterRestart.LoadAuthRequestMetadata(ctx, "stale-registration-lost-token")
	if err != nil {
		t.Fatal(err)
	}
	if metadata.RequestState != auth.AuthRequestExchangeAmbiguous {
		t.Fatalf("stale exchange without quarantine state=%s, want exchange_ambiguous", metadata.RequestState)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE oauth_auth_requests
		SET exchange_finished_at=now()-interval '31 minutes'
		WHERE state='stale-registration-lost-token'
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := storeAfterRestart.ReserveAuthRequestCapacity(ctx); err != nil {
		t.Fatalf("bounded registration ambiguity still consumed capacity: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		DELETE FROM oauth_unverified_credentials WHERE request_state='stale-registration-held';
		UPDATE oauth_auth_requests
		SET request_state='revoked',consumed_at=now()-interval '2 hours'
		WHERE state='stale-registration-held';
		UPDATE oauth_auth_requests
		SET exchange_finished_at=now()-interval '2 hours'
		WHERE state='stale-registration-lost-token'
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := storeAfterRestart.SweepAuthRequests(ctx, 10); err != nil {
		t.Fatalf("sweep terminal registration evidence: %v", err)
	}

	var requests, reservations, owners, sessions int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM oauth_auth_requests`).Scan(&requests); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM oauth_auth_request_reservations`).Scan(&reservations); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM owner_lifecycles`).Scan(&owners); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM oauth_sessions`).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if requests != 0 || reservations != 1 || owners != 0 || sessions != 0 {
		t.Fatalf("durable residue requests=%d reservations=%d owners=%d sessions=%d, want 0/1/0/0", requests, reservations, owners, sessions)
	}
}

func TestAuthRequestAdmissionDeletesExpiredReadyRowsBeforeCapacityCheck(t *testing.T) {
	pool := withAuthSchema(t)
	config := testStoreConfig()
	config.PendingAuthRequestCapacity = 1
	store := auth.NewPostgresAuthStore(pool, config)

	firstOwner := syntax.DID("did:plc:expired-capacity-first")
	secondOwner := syntax.DID("did:plc:expired-capacity-second")
	seedAuthOwner(t, pool, firstOwner)
	seedAuthOwner(t, pool, secondOwner)
	if err := saveAdmissionAuthRequest(store, firstOwner, 1); err != nil {
		t.Fatalf("save first auth request: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE oauth_auth_requests SET created_at=now()-interval '2 hours'
		WHERE state='capacity-state-01'
	`); err != nil {
		t.Fatal(err)
	}

	if err := saveAdmissionAuthRequest(store, secondOwner, 2); err != nil {
		t.Fatalf("expired ready row blocked replacement admission: %v", err)
	}

	var states []string
	rows, err := pool.Query(context.Background(), `SELECT state FROM oauth_auth_requests ORDER BY state`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var state string
		if err := rows.Scan(&state); err != nil {
			t.Fatal(err)
		}
		states = append(states, state)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(states) != 1 || states[0] != "capacity-state-02" {
		t.Fatalf("remaining auth request states=%v, want only replacement", states)
	}
}

func TestSweepAuthRequestsIsBoundedAndPreservesInFlightAndAmbiguousEvidence(t *testing.T) {
	pool := withAuthSchema(t)
	config := testStoreConfig()
	config.PendingAuthRequestCapacity = 20
	config.AuthRequestTerminalRetention = 24 * time.Hour
	store := auth.NewPostgresAuthStore(pool, config)

	states := []string{
		"sweep-ready",
		"sweep-exchange-failed",
		"sweep-consumed",
		"sweep-revoked",
		"sweep-exchange-started",
		"sweep-exchange-ambiguous",
		"sweep-recent-consumed",
	}
	for index, state := range states {
		owner := syntax.DID(fmt.Sprintf("did:plc:sweep-%02d", index))
		seedAuthOwner(t, pool, owner)
		requestContext := auth.WithLoginAuthRequest(
			context.Background(), owner, 1, 1, auth.HandoffVerifiedLink,
			fmt.Sprintf("device-sweep-%02d", index), "",
		)
		if err := store.SaveAuthRequestInfo(requestContext, oauth.AuthRequestData{
			State: state, RequestURI: "urn:request:" + state,
		}); err != nil {
			t.Fatalf("save %s: %v", state, err)
		}
	}

	failedAttempt, err := store.BeginExchange(context.Background(), "sweep-exchange-failed")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MarkExchangeFailed(context.Background(), "sweep-exchange-failed", failedAttempt); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteAuthRequestInfo(context.Background(), "sweep-consumed"); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteAuthRequestInfo(context.Background(), "sweep-recent-consumed"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE oauth_auth_requests
		SET request_state='revoked',consumed_at=now()
		WHERE state='sweep-revoked'
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.BeginExchange(context.Background(), "sweep-exchange-started"); err != nil {
		t.Fatal(err)
	}
	ambiguousAttempt, err := store.BeginExchange(context.Background(), "sweep-exchange-ambiguous")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MarkExchangeAmbiguous(context.Background(), "sweep-exchange-ambiguous", ambiguousAttempt); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE oauth_auth_requests
		SET created_at=now()-interval '48 hours',
		    consumed_at=CASE WHEN consumed_at IS NULL THEN NULL ELSE now()-interval '48 hours' END,
		    exchange_started_at=CASE WHEN exchange_started_at IS NULL THEN NULL ELSE now()-interval '48 hours' END,
		    exchange_finished_at=CASE WHEN exchange_finished_at IS NULL THEN NULL ELSE now()-interval '48 hours' END
		WHERE state<>'sweep-recent-consumed'
	`); err != nil {
		t.Fatal(err)
	}

	first, err := store.SweepAuthRequests(context.Background(), 2)
	if err != nil {
		t.Fatalf("first bounded sweep: %v", err)
	}
	if first.Deleted != 2 {
		t.Fatalf("first sweep deleted=%d, want batch limit 2", first.Deleted)
	}
	second, err := store.SweepAuthRequests(context.Background(), 10)
	if err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if second.Deleted != 2 {
		t.Fatalf("second sweep deleted=%d, want remaining 2", second.Deleted)
	}
	if second.Pending != 2 || second.OldestPendingCreatedAt == nil {
		t.Fatalf("pending sweep metrics=%+v, want two retained pending evidence rows and oldest age input", second)
	}

	rows, err := pool.Query(context.Background(), `SELECT state FROM oauth_auth_requests ORDER BY state`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var remaining []string
	for rows.Next() {
		var state string
		if err := rows.Scan(&state); err != nil {
			t.Fatal(err)
		}
		remaining = append(remaining, state)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	want := []string{"sweep-exchange-ambiguous", "sweep-exchange-started", "sweep-recent-consumed"}
	if fmt.Sprint(remaining) != fmt.Sprint(want) {
		t.Fatalf("remaining states=%v, want %v", remaining, want)
	}
}

func TestProviderRegistrationPreservesLoginAndDeletionAmbiguityRetentionAndCapacity(t *testing.T) {
	pool := withAuthSchema(t)
	config := testStoreConfig()
	config.PendingAuthRequestCapacity = 3
	config.AuthRequestExpiry = 30 * time.Minute
	config.AuthRequestTerminalRetention = time.Hour
	store := auth.NewPostgresAuthStore(pool, config)
	ctx := context.Background()

	loginOwner := syntax.DID("did:plc:retained-login-ambiguity")
	deletionOwner := syntax.DID("did:plc:retained-deletion-ambiguity")
	seedAuthOwner(t, pool, loginOwner)
	seedOwnerLifecycle(t, pool, deletionOwner, "deletion_pending")
	loginContext := auth.WithLoginAuthRequest(
		ctx, loginOwner, 1, 1, auth.HandoffVerifiedLink, "device-retained-login", "",
	)
	deletionContext := auth.WithAccountDeletionAuthRequestAuthority(
		ctx, deletionOwner, 1, 1,
		uuid.MustParse("10000000-0000-4000-8000-000000000016"), "device-retained-deletion",
	)
	for _, request := range []struct {
		context context.Context
		state   string
	}{
		{context: loginContext, state: "retained-login-ambiguity"},
		{context: deletionContext, state: "retained-deletion-ambiguity"},
	} {
		if err := store.SaveAuthRequestInfo(request.context, oauth.AuthRequestData{
			State: request.state, RequestURI: "urn:request:" + request.state,
		}); err != nil {
			t.Fatalf("save %s: %v", request.state, err)
		}
		attemptID, err := store.BeginExchange(ctx, request.state)
		if err != nil {
			t.Fatalf("begin %s: %v", request.state, err)
		}
		if err := store.MarkExchangeAmbiguous(ctx, request.state, attemptID); err != nil {
			t.Fatalf("mark %s ambiguous: %v", request.state, err)
		}
	}
	if _, err := pool.Exec(ctx, `
		UPDATE oauth_auth_requests
		SET created_at=now()-interval '48 hours',
		    consumed_at=now()-interval '48 hours',
		    exchange_started_at=now()-interval '48 hours',
		    exchange_finished_at=now()-interval '48 hours'
		WHERE state IN ('retained-login-ambiguity','retained-deletion-ambiguity')
	`); err != nil {
		t.Fatal(err)
	}

	reservation, err := store.ReserveAuthRequestCapacity(ctx)
	if err != nil {
		t.Fatalf("reserve registration alongside retained ambiguity: %v", err)
	}
	if _, err := store.ReserveAuthRequestCapacity(ctx); !errors.Is(err, auth.ErrAuthRequestCapacity) {
		t.Fatalf("fourth pending unit error=%v, want capacity exhausted", err)
	}
	stats, err := store.SweepAuthRequests(ctx, 10)
	if err != nil {
		t.Fatalf("sweep mixed-purpose pending state: %v", err)
	}
	if stats.Deleted != 0 || stats.Pending != 3 {
		t.Fatalf("mixed-purpose sweep stats=%+v, want zero deleted and three pending", stats)
	}

	rows, err := pool.Query(ctx, `
		SELECT purpose,request_state FROM oauth_auth_requests ORDER BY purpose
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var retained []string
	for rows.Next() {
		var purpose auth.OAuthPurpose
		var state auth.AuthRequestState
		if err := rows.Scan(&purpose, &state); err != nil {
			t.Fatal(err)
		}
		retained = append(retained, string(purpose)+":"+string(state))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	want := []string{"accountDeletion:exchange_ambiguous", "login:exchange_ambiguous"}
	if fmt.Sprint(retained) != fmt.Sprint(want) {
		t.Fatalf("retained ambiguity=%v, want %v", retained, want)
	}

	if err := store.ReleaseAuthRequestCapacity(ctx, reservation.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReserveAuthRequestCapacity(ctx); err != nil {
		t.Fatalf("released registration reservation did not restore one slot: %v", err)
	}
}

func saveAdmissionAuthRequest(store *auth.PostgresAuthStore, owner syntax.DID, index int) error {
	ctx := auth.WithLoginAuthRequest(
		context.Background(), owner, 1, 1, auth.HandoffVerifiedLink,
		fmt.Sprintf("device-capacity-%02d", index), "",
	)
	return store.SaveAuthRequestInfo(ctx, oauth.AuthRequestData{
		State:      fmt.Sprintf("capacity-state-%02d", index),
		RequestURI: fmt.Sprintf("urn:request:capacity-%02d", index),
	})
}
