package subscriptions

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/testdb"
)

func TestAcceptRevenueCatEventDeduplicatesSanitizedWorkTransactionally(t *testing.T) {
	accounts, err := os.ReadFile("../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	events, err := os.ReadFile("../../migrations/000070_revenuecat_events.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(accounts)+string(events))
	store := NewStore(pool)
	account, _, err := store.EnsureAccount(context.Background(), syntax.DID("did:plc:webhook-owner"))
	if err != nil {
		t.Fatal(err)
	}
	event := RevenueCatEvent{ID: "event-stable", Type: "UNKNOWN_FUTURE_EVENT", RevenueCatAppUserID: account.RevenueCatAppUserID, OccurredAt: time.UnixMilli(1788955200123)}

	const deliveries = 8
	results := make(chan bool, deliveries)
	errorsCh := make(chan error, deliveries)
	var ready sync.WaitGroup
	ready.Add(deliveries)
	start := make(chan struct{})
	for range deliveries {
		go func() {
			ready.Done()
			<-start
			accepted, err := store.AcceptRevenueCatEvent(context.Background(), event, time.Now())
			if err != nil {
				errorsCh <- err
				return
			}
			results <- accepted
		}()
	}
	ready.Wait()
	close(start)
	acceptedCount := 0
	for range deliveries {
		select {
		case err := <-errorsCh:
			t.Fatalf("concurrent event acceptance: %v", err)
		case accepted := <-results:
			if accepted {
				acceptedCount++
			}
		}
	}
	if acceptedCount != 1 {
		t.Fatalf("new event acceptances = %d, want 1", acceptedCount)
	}
	var eventCount int
	var generation int64
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM revenuecat_events`).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT requested_generation FROM billing_accounts WHERE id=$1`, account.ID).Scan(&generation); err != nil {
		t.Fatal(err)
	}
	if eventCount != 1 || generation != 1 {
		t.Fatalf("durable event/work = count %d generation %d, want 1/1", eventCount, generation)
	}

	retry := RevenueCatEvent{ID: "event-retry", Type: strings.Repeat("x", 129), RevenueCatAppUserID: account.RevenueCatAppUserID}
	if _, err := store.AcceptRevenueCatEvent(context.Background(), retry, time.Now()); err == nil {
		t.Fatal("invalid event unexpectedly committed")
	}
	retry.Type = "RENEWAL"
	if accepted, err := store.AcceptRevenueCatEvent(context.Background(), retry, time.Now()); err != nil || !accepted {
		t.Fatalf("retry after failed persistence = accepted %t, error %v", accepted, err)
	}

	for _, prohibited := range []string{"raw_body", "payload", "receipt", "secret", "customer_id"} {
		var exists bool
		if err := pool.QueryRow(context.Background(), `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema=current_schema() AND table_name='revenuecat_events' AND column_name=$1
			)
		`, prohibited).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if exists {
			t.Fatalf("prohibited persisted event column %q exists", prohibited)
		}
	}
}
