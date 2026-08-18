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
