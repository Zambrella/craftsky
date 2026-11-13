package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/testdb"
)

type checkoutFake struct {
	mu             sync.Mutex
	requests       []CheckoutRequest
	session        CheckoutSession
	failNext       bool
	portalCustomer string
}

func (f *checkoutFake) CreateCheckout(_ context.Context, r CheckoutRequest) (CheckoutSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, r)
	if f.session.ID == "" {
		f.session = CheckoutSession{ID: "cs_live_checkout", URL: "https://checkout.stripe.com/c/pay/cs_live_checkout#opaque", Status: "open", LiveMode: true, EnvironmentKnown: true, Managed: true, StripeTaxLiability: true, AppUserID: r.CustomerID.String(), ExpiresAt: time.Now().Add(time.Hour)}
	}
	if f.failNext {
		f.failNext = false
		return CheckoutSession{}, errors.New("synthetic uncertain transport")
	}
	return f.session, nil
}
func (f *checkoutFake) GetCheckout(context.Context, string) (CheckoutSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.session, nil
}
func (f *checkoutFake) CreateManagement(_ context.Context, customer, _ string) (string, error) {
	f.portalCustomer = customer
	return "https://billing.stripe.com/p/session/test_portal#opaque", nil
}
func checkoutFixture(t *testing.T) (*Store, *pgxpool.Pool, *checkoutFake, syntax.DID, BillingAccount, time.Time) {
	t.Helper()
	old, err := os.ReadFile("../../migrations/000076_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../migrations/000125_stripe_web_checkouts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(old)+`ALTER TABLE provider_subscriptions ADD COLUMN pending_mapped_tier TEXT;`+string(migration)+`CREATE TABLE moderation_account_standings(owner_did TEXT PRIMARY KEY,effective_suspended BOOLEAN NOT NULL DEFAULT false);`)
	store := NewStore(pool)
	fake := &checkoutFake{}
	did := syntax.DID("did:plc:checkout")
	account, _, err := store.EnsureAccount(t.Context(), did)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := pool.Exec(t.Context(), `UPDATE billing_accounts SET reconciled_at=$2 WHERE id=$1`, account.ID, now); err != nil {
		t.Fatal(err)
	}
	if err := store.SetWebCheckoutProvider(fake, WebCheckoutConfig{AccountID: "acct_merchant", PlusPriceID: "price_plus", BusinessPriceID: "price_business", SuccessURL: "https://app.example/settings/subscriptions", CancelURL: "https://app.example/settings/subscriptions", BillingReturnURL: "https://app.example/settings/subscriptions"}); err != nil {
		t.Fatal(err)
	}
	return store, pool, fake, did, account, now
}
func TestStripeCheckoutRecoversUncertainOutcomeWithOriginalArguments(t *testing.T) {
	store, pool, fake, did, account, now := checkoutFixture(t)
	fake.failNext = true
	if _, err := store.WebCheckoutURL(t.Context(), did, TierPlus, now); err == nil {
		t.Fatal("expected uncertain failure")
	}
	// Another process/configuration deployment must recover the persisted request,
	// not dispatch a second tier or another URL under a new key.
	another := NewStore(pool)
	if err := another.SetWebCheckoutProvider(fake, store.checkoutConfig); err != nil {
		t.Fatal(err)
	}
	if _, err := another.WebCheckoutURL(t.Context(), did, TierBusiness, now); !errors.Is(err, ErrWebCheckoutPending) {
		t.Fatalf("different tier=%v", err)
	}
	if len(fake.requests) != 2 || fake.requests[0] != fake.requests[1] || fake.requests[0].CustomerID != account.RevenueCatAppUserID {
		t.Fatalf("idempotent request mismatch: calls=%d", len(fake.requests))
	}
	raw, err := store.WebCheckoutURL(t.Context(), did, TierPlus, now)
	if err != nil || raw != fake.session.URL {
		t.Fatalf("resume err=%v", err)
	}
	if len(fake.requests) != 2 {
		t.Fatal("known session was recreated")
	}
}
func TestStripeCheckoutConcurrentTiersShareOnePayableSession(t *testing.T) {
	store, _, fake, did, _, now := checkoutFixture(t)
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, tier := range []Tier{TierPlus, TierBusiness} {
		go func() { <-start; _, err := store.WebCheckoutURL(context.Background(), did, tier, now); results <- err }()
	}
	close(start)
	var allowed, pending int
	for range 2 {
		err := <-results
		if err == nil {
			allowed++
		} else if errors.Is(err, ErrWebCheckoutPending) {
			pending++
		} else {
			t.Fatal(err)
		}
	}
	if allowed != 1 || pending != 1 || len(fake.requests) != 1 {
		t.Fatalf("allowed=%d pending=%d creates=%d", allowed, pending, len(fake.requests))
	}
}
func TestStripeCheckoutRequiresObservedExpirationAndFreshProviderState(t *testing.T) {
	store, pool, fake, did, account, now := checkoutFixture(t)
	if _, err := store.WebCheckoutURL(t.Context(), did, TierPlus, now); err != nil {
		t.Fatal(err)
	}
	fake.session.ExpiresAt = now.Add(-time.Second)
	if _, err := store.WebCheckoutURL(t.Context(), did, TierBusiness, now); !errors.Is(err, ErrWebCheckoutPending) {
		t.Fatalf("local expiry allowed replacement=%v", err)
	}
	if len(fake.requests) != 1 {
		t.Fatal("second session before observed expiration")
	}
	fake.session.Status = "expired"
	fake.session.ID = "cs_live_checkout"
	fake.session.ExpiresAt = now.Add(time.Hour)
	if _, err := store.WebCheckoutURL(t.Context(), did, TierBusiness, now); err == nil {
		t.Fatal("expired create result accepted")
	}
	if len(fake.requests) != 2 {
		t.Fatal("observed expired did not replace operation")
	}
	// Completed checkout cannot disappear merely because RevenueCat's first
	// snapshot is still empty.
	fake.session.Status = "complete"
	fake.session.SubscriptionAppUserID = account.RevenueCatAppUserID.String()
	fake.session.CustomerID = "cus_right"
	if _, err := store.WebCheckoutURL(t.Context(), did, TierPlus, now); !errors.Is(err, ErrSubscriptionAlreadyExists) {
		t.Fatalf("complete empty snapshot=%v", err)
	}
	fake.session.SubscriptionCanceled = true
	if _, err := pool.Exec(t.Context(), `UPDATE billing_accounts SET requested_generation=1 WHERE id=$1`, account.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.ResolveWebCheckoutForDeletion(t.Context(), did, now); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM billing_web_checkouts`).Scan(&count); err != nil || count != 1 {
		t.Fatal("stale snapshot cleared completed operation")
	}
	if _, err := pool.Exec(t.Context(), `UPDATE billing_accounts SET reconciled_generation=requested_generation,reconciled_at=$2 WHERE id=$1`, account.ID, now); err != nil {
		t.Fatal(err)
	}
	if err := store.ResolveWebCheckoutForDeletion(t.Context(), did, now); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM billing_web_checkouts`).Scan(&count); err != nil || count != 0 {
		t.Fatal("resolved cancellation not cleared")
	}
}
func TestStripeCheckoutRejectsUnverifiedStateAndUnsafeSessions(t *testing.T) {
	store, pool, fake, did, account, now := checkoutFixture(t)
	if _, err := pool.Exec(t.Context(), `UPDATE billing_accounts SET reconciled_at=NULL WHERE id=$1`, account.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.WebCheckoutURL(t.Context(), did, TierPlus, now); !errors.Is(err, ErrSubscriptionReconciliationRequired) {
		t.Fatalf("unverified state=%v", err)
	}
	if len(fake.requests) != 0 {
		t.Fatal("provider called before reconciliation")
	}
	if _, err := pool.Exec(t.Context(), `UPDATE billing_accounts SET reconciled_at=$2 WHERE id=$1`, account.ID, now); err != nil {
		t.Fatal(err)
	}
	base := CheckoutSession{ID: "cs_live_checkout", Status: "open", Managed: true, StripeTaxLiability: true, LiveMode: true, EnvironmentKnown: true, AppUserID: account.RevenueCatAppUserID.String(), ExpiresAt: now.Add(time.Hour), URL: "https://checkout.stripe.com/c/pay/cs_live_checkout"}
	for _, test := range []struct {
		name   string
		change func(*CheckoutSession)
	}{
		{"managed payments absent", func(s *CheckoutSession) { s.Managed = false }},
		{"Stripe tax liability absent", func(s *CheckoutSession) { s.StripeTaxLiability = false }},
		{"environment absent", func(s *CheckoutSession) { s.EnvironmentKnown = false }},
		{"wrong environment", func(s *CheckoutSession) { s.LiveMode = false }},
		{"wrong customer metadata", func(s *CheckoutSession) { s.AppUserID = uuid.NewString() }},
		{"wrong subscription metadata", func(s *CheckoutSession) { s.Status = "complete"; s.SubscriptionAppUserID = uuid.NewString() }},
		{"wrong session environment prefix", func(s *CheckoutSession) { s.ID = "cs_test_checkout" }},
		{"unsafe destination", func(s *CheckoutSession) { s.URL = "https://checkout.stripe.com.evil.example/c/pay/cs_live_checkout" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake.session = base
			test.change(&fake.session)
			if _, err := store.WebCheckoutURL(t.Context(), did, TierPlus, now); !errors.Is(err, ErrWebCheckoutUnavailable) {
				t.Fatalf("unsafe provider response accepted: %v", err)
			}
		})
	}
}
func TestStripeManagementUsesOnlyVerifiedOwnerCustomer(t *testing.T) {
	store, _, fake, did, account, now := checkoutFixture(t)
	if _, err := store.WebManagementURL(t.Context(), did, now); !errors.Is(err, ErrWebManagementUnavailable) {
		t.Fatal("unmapped management allowed")
	}
	if _, err := store.WebCheckoutURL(t.Context(), did, TierPlus, now); err != nil {
		t.Fatal(err)
	}
	fake.session.Status = "complete"
	fake.session.CustomerID = "cus_right"
	fake.session.SubscriptionAppUserID = account.RevenueCatAppUserID.String()
	if _, err := store.WebManagementURL(t.Context(), did, now); err != nil {
		t.Fatal(err)
	}
	if fake.portalCustomer != "cus_right" {
		t.Fatal("wrong portal customer")
	}
	fake.session.AppUserID = uuid.NewString()
	if _, err := store.WebManagementURL(t.Context(), did, now); !errors.Is(err, ErrWebManagementUnavailable) {
		t.Fatal("wrong owner session allowed")
	}
}
func TestStripeUnknownCheckoutCannotReplayAfterIdempotencyRetention(t *testing.T) {
	store, pool, fake, did, account, now := checkoutFixture(t)
	fake.failNext = true
	_, _ = store.WebCheckoutURL(t.Context(), did, TierPlus, now)
	if _, err := pool.Exec(t.Context(), `UPDATE billing_web_checkouts SET created_at=$2 WHERE billing_account_id=$1`, account.ID, now.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.WebCheckoutURL(t.Context(), did, TierPlus, now); !errors.Is(err, ErrWebCheckoutUnavailable) {
		t.Fatalf("old uncertain operation=%v", err)
	}
	if len(fake.requests) != 1 {
		t.Fatal("old idempotency key replayed")
	}
}

func TestStripePendingFlagSurvivesDisabledConfigurationAndBothEnvironments(t *testing.T) {
	store, pool, _, did, account, now := checkoutFixture(t)
	if _, err := store.WebCheckoutURL(t.Context(), did, TierPlus, now); err != nil {
		t.Fatal(err)
	}
	disabled := NewStoreForEnvironment(pool, "sandbox")
	state, err := disabled.OwnerState(t.Context(), did, now)
	if err != nil || !state.WebCheckoutPending || state.WebCheckoutAvailable {
		t.Fatalf("disabled sandbox pending=%t available=%t err=%v", state.WebCheckoutPending, state.WebCheckoutAvailable, err)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO billing_web_checkouts(billing_account_id,environment,operation_id,tier,stripe_account_id,price_id,success_url,cancel_url,created_at) VALUES($1,'sandbox',$2,'plus','acct_sandbox','price_plus','https://app.example/success','https://app.example/cancel',$3)`, account.ID, uuid.New(), now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `DELETE FROM billing_web_checkouts WHERE environment='production'`); err != nil {
		t.Fatal(err)
	}
	state, err = store.OwnerState(t.Context(), did, now)
	if err != nil || !state.WebCheckoutPending {
		t.Fatal("remaining other environment lost pending guard")
	}
	if _, err := pool.Exec(t.Context(), `DELETE FROM billing_web_checkouts`); err != nil {
		t.Fatal(err)
	}
	state, err = store.OwnerState(t.Context(), did, now)
	if err != nil || state.WebCheckoutPending {
		t.Fatal("resolved pending flag was retained")
	}
}
func TestStripeCheckoutRechecksLifecycleBeforeDispatch(t *testing.T) {
	store, pool, fake, did, _, now := checkoutFixture(t)
	if _, err := pool.Exec(t.Context(), `CREATE OR REPLACE FUNCTION appview_owner_is_active(candidate_did TEXT) RETURNS BOOLEAN LANGUAGE SQL AS $$ SELECT false $$`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.WebCheckoutURL(t.Context(), did, TierPlus, now); err == nil {
		t.Fatal("departed owner started checkout")
	}
	if len(fake.requests) != 0 {
		t.Fatal("provider called for departed owner")
	}
}

type standingBlockedCheckout struct {
	*checkoutFake
	entered chan struct{}
	release chan struct{}
}

func (f *standingBlockedCheckout) CreateCheckout(ctx context.Context, r CheckoutRequest) (CheckoutSession, error) {
	close(f.entered)
	select {
	case <-f.release:
	case <-ctx.Done():
		return CheckoutSession{}, ctx.Err()
	}
	return f.checkoutFake.CreateCheckout(ctx, r)
}
func TestStripeCheckoutStandingFenceSerializesSuspension(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprint(existing), func(t *testing.T) {
			store, pool, fake, did, _, now := checkoutFixture(t)
			if existing {
				if _, err := pool.Exec(t.Context(), `INSERT INTO moderation_account_standings(owner_did) VALUES($1)`, did); err != nil {
					t.Fatal(err)
				}
			}
			blocked := &standingBlockedCheckout{checkoutFake: fake, entered: make(chan struct{}), release: make(chan struct{})}
			store.checkoutProvider = blocked
			purchase := make(chan error, 1)
			go func() { _, err := store.WebCheckoutURL(context.Background(), did, TierPlus, now); purchase <- err }()
			select {
			case <-blocked.entered:
			case <-time.After(3 * time.Second):
				t.Fatal("checkout did not reach provider")
			}
			suspension := make(chan error, 1)
			go func() {
				tx, err := pool.Begin(context.Background())
				if err != nil {
					suspension <- err
					return
				}
				defer tx.Rollback(context.Background())
				// Moderation's standing transitions ensure the row then lock/update it.
				if _, err = tx.Exec(context.Background(), `INSERT INTO moderation_account_standings(owner_did) VALUES($1) ON CONFLICT DO NOTHING`, did); err == nil {
					_, err = tx.Exec(context.Background(), `UPDATE moderation_account_standings SET effective_suspended=true WHERE owner_did=$1`, did)
				}
				if err == nil {
					err = tx.Commit(context.Background())
				}
				suspension <- err
			}()
			select {
			case err := <-suspension:
				t.Fatalf("suspension crossed the provider effect: %v", err)
			case <-time.After(100 * time.Millisecond):
			}
			close(blocked.release)
			select {
			case err := <-purchase:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("checkout did not finish")
			}
			select {
			case err := <-suspension:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("suspension did not finish")
			}
			if _, err := store.WebCheckoutURL(t.Context(), did, TierPlus, now); !errors.Is(err, ErrWebCheckoutUnavailable) {
				t.Fatalf("suspended resume=%v", err)
			}
			if len(fake.requests) != 1 {
				t.Fatal("suspended owner dispatched new checkout")
			}
		})
	}
}

func TestStripeSuspendedOwnerCanResolveRetainedCheckout(t *testing.T) {
	store, pool, fake, did, _, now := checkoutFixture(t)
	if _, err := store.WebCheckoutURL(t.Context(), did, TierPlus, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `UPDATE moderation_account_standings SET effective_suspended=true WHERE owner_did=$1`, did); err != nil {
		t.Fatal(err)
	}
	fake.session.Status = "expired"
	if err := store.ResolveWebCheckoutForDeletion(t.Context(), did, now); err != nil {
		t.Fatal(err)
	}
	state, err := store.OwnerState(t.Context(), did, now)
	if err != nil || state.WebCheckoutPending {
		t.Fatalf("suspended retained checkout pending=%t err=%v", state.WebCheckoutPending, err)
	}
	if len(fake.requests) != 1 {
		t.Fatal("retained resolver created another checkout")
	}
}

func TestStripeStandingDenialOnlyClearsNewUndispatchedOperation(t *testing.T) {
	for _, previous := range []string{"none", "known", "uncertain"} {
		t.Run(previous, func(t *testing.T) {
			store, pool, fake, did, account, now := checkoutFixture(t)
			if previous != "none" {
				fake.failNext = previous == "uncertain"
				_, err := store.WebCheckoutURL(t.Context(), did, TierPlus, now)
				if (err != nil) != (previous == "uncertain") {
					t.Fatalf("prepare %s checkout: %v", previous, err)
				}
			}
			before := len(fake.requests)
			if _, err := pool.Exec(t.Context(), `INSERT INTO moderation_account_standings(owner_did,effective_suspended) VALUES($1,true) ON CONFLICT(owner_did) DO UPDATE SET effective_suspended=true`, did); err != nil {
				t.Fatal(err)
			}
			if _, err := store.WebCheckoutURL(t.Context(), did, TierPlus, now); !errors.Is(err, ErrWebCheckoutUnavailable) {
				t.Fatalf("suspended checkout=%v", err)
			}
			var count int
			if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM billing_web_checkouts WHERE billing_account_id=$1`, account.ID).Scan(&count); err != nil {
				t.Fatal(err)
			}
			state, err := store.OwnerState(t.Context(), did, now)
			wantRetained := previous != "none"
			wantCount := 0
			if wantRetained {
				wantCount = 1
			}
			if err != nil || count != wantCount || state.WebCheckoutPending != wantRetained || len(fake.requests) != before {
				t.Fatalf("previous=%s operations=%d pending=%t creates=%d before=%d err=%v", previous, count, state.WebCheckoutPending, len(fake.requests), before, err)
			}
		})
	}
}

func TestStripeCheckoutUsesOnePoolConnectionForEveryOwnerOperation(t *testing.T) {
	store, pool, fake, did, account, now := checkoutFixture(t)
	config := pool.Config()
	config.MaxConns = 1
	config.MinConns = 0
	single, err := pgxpool.NewWithConfig(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(single.Close)
	store.pool = single
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if _, err := store.WebCheckoutURL(ctx, did, TierPlus, now); err != nil {
		t.Fatalf("create with one connection: %v", err)
	}
	if _, err := store.WebCheckoutURL(ctx, did, TierPlus, now); err != nil {
		t.Fatalf("resume with one connection: %v", err)
	}
	fake.session.Status = "complete"
	fake.session.SubscriptionAppUserID = account.RevenueCatAppUserID.String()
	fake.session.CustomerID = "cus_right"
	if _, err := store.WebManagementURL(ctx, did, now); err != nil {
		t.Fatalf("management with one connection: %v", err)
	}
	fake.session.SubscriptionCanceled = true
	if err := store.ResolveWebCheckoutForDeletion(ctx, did, now); err != nil {
		t.Fatalf("cleanup with one connection: %v", err)
	}
	if _, err := single.Exec(ctx, `UPDATE billing_accounts SET reconciled_generation=requested_generation,reconciled_at=$2 WHERE id=$1`, account.ID, now); err != nil {
		t.Fatal(err)
	}
	if err := store.ResolveWebCheckoutForDeletion(ctx, did, now); err != nil {
		t.Fatalf("reconciled cleanup with one connection: %v", err)
	}
	state, err := store.OwnerState(ctx, did, now)
	if err != nil || state.WebCheckoutPending || fake.portalCustomer != "cus_right" {
		t.Fatalf("completed owner operations pending=%t err=%v", state.WebCheckoutPending, err)
	}
}

func TestStripeCanceledCheckoutRequiresSnapshotGenerationAfterCancellation(t *testing.T) {
	store, pool, fake, did, account, now := checkoutFixture(t)
	if _, err := store.WebCheckoutURL(t.Context(), did, TierPlus, now); err != nil {
		t.Fatal(err)
	}
	fake.session.Status = "complete"
	fake.session.SubscriptionAppUserID = account.RevenueCatAppUserID.String()
	fake.session.SubscriptionCanceled = true
	// Even an otherwise fresh empty snapshot cannot release a completed payment
	// until a generation requested after verified cancellation is reconciled.
	if err := store.ResolveWebCheckoutForDeletion(t.Context(), did, now); err != nil {
		t.Fatal(err)
	}
	var requested, target int64
	if err := pool.QueryRow(t.Context(), `SELECT account.requested_generation,checkout.terminal_reconciliation_generation FROM billing_accounts account JOIN billing_web_checkouts checkout ON checkout.billing_account_id=account.id WHERE account.id=$1`, account.ID).Scan(&requested, &target); err != nil || target < 1 || requested != target {
		t.Fatalf("post-cancellation generation requested=%d target=%d err=%v", requested, target, err)
	}
	// Simulate a pre-cancellation request whose empty snapshot finishes later.
	if _, err := pool.Exec(t.Context(), `UPDATE billing_accounts SET reconciled_at=$2 WHERE id=$1`, account.ID, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := store.ResolveWebCheckoutForDeletion(t.Context(), did, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	state, err := store.OwnerState(t.Context(), did, now)
	if err != nil || !state.WebCheckoutPending || state.RequestedGeneration != target || state.ReconciledGeneration >= target {
		t.Fatalf("old generation released checkout pending=%t requested=%d reconciled=%d err=%v", state.WebCheckoutPending, state.RequestedGeneration, state.ReconciledGeneration, err)
	}
	if _, err := pool.Exec(t.Context(), `UPDATE billing_accounts SET reconciled_generation=$2,reconciled_at=$3 WHERE id=$1`, account.ID, target, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := store.ResolveWebCheckoutForDeletion(t.Context(), did, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	state, err = store.OwnerState(t.Context(), did, now)
	if err != nil || state.WebCheckoutPending {
		t.Fatalf("reconciled terminal checkout retained pending=%t err=%v", state.WebCheckoutPending, err)
	}
}
