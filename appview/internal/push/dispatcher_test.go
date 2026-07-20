package push

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"social.craftsky/appview/internal/testdb"
)

type blockingSender struct {
	mu       sync.Mutex
	results  []ProviderResult
	requests []SendRequest
	started  chan int
	release  chan struct{}
}

type advancingSender struct {
	now      *time.Time
	advance  time.Duration
	requests []SendRequest
}

type queueObserver struct {
	pending int
	age     time.Duration
}

type privacyObserver struct{ values []string }

func (o *privacyObserver) ObservePushDelivery(platform, result string) {
	o.values = append(o.values, platform, result)
}
func (o *privacyObserver) ObservePushQueue(pending int, age time.Duration) {
	o.values = append(o.values, fmt.Sprint(pending), age.String())
}

type sentinelFailureSender struct{ sentinel string }

func (s sentinelFailureSender) Send(context.Context, SendRequest) (ProviderResult, error) {
	return ProviderResult{Class: ResultRetryable}, errors.New("provider failure included " + s.sentinel)
}

type contextBlockingSender struct {
	request  SendRequest
	deadline time.Time
}

func (s *contextBlockingSender) Send(ctx context.Context, request SendRequest) (ProviderResult, error) {
	s.request = request
	s.deadline, _ = ctx.Deadline()
	<-ctx.Done()
	return ProviderResult{Class: ResultRetryable}, ctx.Err()
}

func (*queueObserver) ObservePushDelivery(string, string) {}
func (o *queueObserver) ObservePushQueue(pending int, age time.Duration) {
	o.pending = pending
	o.age = age
}

func (s *advancingSender) Send(_ context.Context, request SendRequest) (ProviderResult, error) {
	s.requests = append(s.requests, request)
	*s.now = s.now.Add(s.advance)
	return ProviderResult{Class: ResultSuccess}, nil
}

func (s *blockingSender) Send(_ context.Context, request SendRequest) (ProviderResult, error) {
	s.mu.Lock()
	call := len(s.requests)
	s.requests = append(s.requests, request)
	result := ProviderResult{Class: ResultSuccess}
	if call < len(s.results) {
		result = s.results[call]
	}
	s.mu.Unlock()
	s.started <- call
	<-s.release
	return result, nil
}

func (s *blockingSender) requestCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

type scriptedSender struct {
	mu       sync.Mutex
	results  []ProviderResult
	requests []SendRequest
}

func (s *scriptedSender) Send(_ context.Context, request SendRequest) (ProviderResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, request)
	if len(s.results) == 0 {
		return ProviderResult{Class: ResultSuccess}, nil
	}
	result := s.results[0]
	s.results = s.results[1:]
	if result.Class == ResultRetryable {
		return result, errors.New("provider unavailable")
	}
	return result, nil
}

func (s *scriptedSender) requestCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

func TestDispatcherIT001ProjectsCanonicalRoutingFacts(t *testing.T) {
	pool := dispatcherPool(t)
	seedDelivery(t, pool, "pending", time.Now().Add(6*time.Hour))
	const sourceURI = "at://did:plc:actor/social.craftsky.feed.post/source"
	const subjectURI = "at://did:plc:recipient/social.craftsky.feed.post/subject"
	const rootURI = "at://did:plc:root/social.craftsky.feed.post/root"
	if _, err := pool.Exec(
		context.Background(),
		`UPDATE notification_events SET source_uri=$1,subject_uri=$2,root_uri=$3 WHERE id='00000000-0000-0000-0000-000000000001'`,
		sourceURI,
		subjectURI,
		rootURI,
	); err != nil {
		t.Fatal(err)
	}

	sender := &scriptedSender{}
	now := time.Now().UTC()
	dispatcher := NewDispatcher(pool, sender, DispatcherOptions{Now: func() time.Time { return now }})
	if n, err := dispatcher.ProcessBatch(context.Background(), "worker"); err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if len(sender.requests) != 1 {
		t.Fatalf("send requests = %d", len(sender.requests))
	}

	request := sender.requests[0]
	if request.RoutingFacts.ActorDID.String() != "did:plc:actor" ||
		request.RoutingFacts.SourceURI.String() != sourceURI ||
		request.RoutingFacts.SubjectURI.String() != subjectURI ||
		request.RoutingFacts.RootURI.String() != rootURI {
		t.Fatalf("routing facts = %#v", request.RoutingFacts)
	}
	if request.Token != "secret-token" ||
		request.Platform != "ios" ||
		request.AccountSubscriptionID != "30000000-0000-0000-0000-000000000001" ||
		request.ActorDisplayName != "Alice" ||
		request.TTL <= 0 {
		t.Fatalf("unchanged send metadata = %#v", request)
	}
}

func TestDispatcherSuppressesRelationshipProtectedDeliveryBeforeProviderSend(t *testing.T) {
	for _, setup := range []struct {
		name string
		sql  string
	}{
		{name: "mute", sql: `INSERT INTO actor_mutes(owner_did,subject_did) VALUES('did:plc:viewer','did:plc:actor')`},
		{name: "inbound block", sql: `INSERT INTO atproto_blocks(uri,blocker_did,subject_did) VALUES('at://did:plc:actor/app.bsky.graph.block/r1','did:plc:actor','did:plc:viewer')`},
	} {
		t.Run(setup.name, func(t *testing.T) {
			pool := dispatcherPool(t)
			seedDelivery(t, pool, "pending", time.Now().Add(6*time.Hour))
			if _, err := pool.Exec(context.Background(), setup.sql); err != nil {
				t.Fatal(err)
			}
			sender := &scriptedSender{}
			dispatcher := NewDispatcher(pool, sender, DispatcherOptions{})
			if n, err := dispatcher.ProcessBatch(context.Background(), "worker"); err != nil || n != 0 {
				t.Fatalf("n=%d err=%v", n, err)
			}
			if sender.requestCount() != 0 {
				t.Fatalf("provider sends = %d, want 0", sender.requestCount())
			}
		})
	}
}

func TestDispatcherIT012ProjectsTargetContentRole(t *testing.T) {
	tests := []struct {
		name       string
		category   string
		subjectURI string
		quotedURI  string
		want       ContentRole
	}{
		{"root post", "like", "at://did:plc:viewer/social.craftsky.feed.post/root", "", ContentRolePost},
		{"direct comment", "like", "at://did:plc:viewer/social.craftsky.feed.post/comment", "", ContentRoleComment},
		{"nested reply", "reply", "at://did:plc:viewer/social.craftsky.feed.post/reply", "", ContentRoleReply},
		{"quoted target", "quote", "at://did:plc:actor/social.craftsky.feed.post/quote", "at://did:plc:viewer/social.craftsky.feed.post/comment", ContentRoleComment},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool := dispatcherPool(t)
			seedDelivery(t, pool, "pending", time.Now().Add(6*time.Hour))
			if _, err := pool.Exec(context.Background(), `
				INSERT INTO craftsky_posts(uri,reply_root_uri,reply_parent_uri) VALUES
				('at://did:plc:viewer/social.craftsky.feed.post/root',NULL,NULL),
				('at://did:plc:viewer/social.craftsky.feed.post/comment','at://did:plc:viewer/social.craftsky.feed.post/root','at://did:plc:viewer/social.craftsky.feed.post/root'),
				('at://did:plc:viewer/social.craftsky.feed.post/reply','at://did:plc:viewer/social.craftsky.feed.post/root','at://did:plc:viewer/social.craftsky.feed.post/comment')
			`); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(
				context.Background(),
				`UPDATE notification_events SET category=$1,subject_uri=$2,quoted_uri=NULLIF($3,'') WHERE id='00000000-0000-0000-0000-000000000001'`,
				test.category,
				test.subjectURI,
				test.quotedURI,
			); err != nil {
				t.Fatal(err)
			}

			sender := &scriptedSender{}
			now := time.Now().UTC()
			dispatcher := NewDispatcher(pool, sender, DispatcherOptions{Now: func() time.Time { return now }})
			if n, err := dispatcher.ProcessBatch(context.Background(), "worker"); err != nil || n != 1 {
				t.Fatalf("n=%d err=%v", n, err)
			}
			if got := sender.requests[0].RoutingFacts.TargetRole; got != test.want {
				t.Fatalf("target role = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDispatcherRetriesThenSucceedsAndDoesNotResendSuccess(t *testing.T) {
	pool := dispatcherPool(t)
	seedDelivery(t, pool, "pending", time.Now().Add(6*time.Hour))
	sender := &scriptedSender{results: []ProviderResult{{Class: ResultRetryable}, {Class: ResultSuccess}}}
	now := time.Now().UTC()
	dispatcher := NewDispatcher(pool, sender, DispatcherOptions{BatchSize: 10, LeaseDuration: time.Minute, Now: func() time.Time { return now }, Jitter: func() float64 { return 0 }})
	if n, err := dispatcher.ProcessBatch(context.Background(), "worker-1"); err != nil || n != 1 {
		t.Fatalf("first n=%d err=%v", n, err)
	}
	now = now.Add(2 * time.Second)
	if n, err := dispatcher.ProcessBatch(context.Background(), "worker-1"); err != nil || n != 1 {
		t.Fatalf("second n=%d err=%v", n, err)
	}
	if n, err := dispatcher.ProcessBatch(context.Background(), "worker-1"); err != nil || n != 0 {
		t.Fatalf("third n=%d err=%v", n, err)
	}
	var status string
	var attempts int
	if err := pool.QueryRow(context.Background(), `SELECT status,attempts FROM push_deliveries`).Scan(&status, &attempts); err != nil {
		t.Fatal(err)
	}
	if status != "succeeded" || attempts != 2 || len(sender.requests) != 2 {
		t.Fatalf("status=%s attempts=%d sends=%d", status, attempts, len(sender.requests))
	}
}

func TestDispatcherExpiresWithoutSendingAndInvalidTokenDeactivatesInstallation(t *testing.T) {
	t.Run("expired", func(t *testing.T) {
		pool := dispatcherPool(t)
		seedDelivery(t, pool, "pending", time.Now().Add(-time.Minute))
		sender := &scriptedSender{}
		d := NewDispatcher(pool, sender, DispatcherOptions{Now: time.Now, BatchSize: 10, LeaseDuration: time.Minute})
		if n, err := d.ProcessBatch(context.Background(), "w"); err != nil || n != 1 {
			t.Fatalf("n=%d err=%v", n, err)
		}
		var status string
		_ = pool.QueryRow(context.Background(), `SELECT status FROM push_deliveries`).Scan(&status)
		if status != "expired" || len(sender.requests) != 0 {
			t.Fatalf("status=%s sends=%d", status, len(sender.requests))
		}
	})
	t.Run("invalid token", func(t *testing.T) {
		pool := dispatcherPool(t)
		seedDelivery(t, pool, "pending", time.Now().Add(6*time.Hour))
		sender := &scriptedSender{results: []ProviderResult{{Class: ResultInvalidToken}}}
		d := NewDispatcher(pool, sender, DispatcherOptions{Now: time.Now, BatchSize: 10, LeaseDuration: time.Minute})
		if _, err := d.ProcessBatch(context.Background(), "w"); err != nil {
			t.Fatal(err)
		}
		var active bool
		_ = pool.QueryRow(context.Background(), `SELECT active FROM push_installations`).Scan(&active)
		if active {
			t.Fatal("installation still active")
		}
	})
}

func TestDispatchersClaimOneDeliveryOnceAndRecoverExpiredLease(t *testing.T) {
	pool := dispatcherPool(t)
	seedDelivery(t, pool, "pending", time.Now().Add(6*time.Hour))
	sender := &scriptedSender{}
	d1 := NewDispatcher(pool, sender, DispatcherOptions{Now: time.Now, BatchSize: 1, LeaseDuration: time.Minute})
	d2 := NewDispatcher(pool, sender, DispatcherOptions{Now: time.Now, BatchSize: 1, LeaseDuration: time.Minute})
	var wg sync.WaitGroup
	wg.Add(2)
	errs := make(chan error, 2)
	go func() { defer wg.Done(); _, err := d1.ProcessBatch(context.Background(), "one"); errs <- err }()
	go func() { defer wg.Done(); _, err := d2.ProcessBatch(context.Background(), "two"); errs <- err }()
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(sender.requests) != 1 {
		t.Fatalf("concurrent sends=%d", len(sender.requests))
	}

	if _, err := pool.Exec(context.Background(), `UPDATE push_deliveries SET status='leased',sent_at=NULL,lease_owner='crashed',lease_expires_at=now()-interval '1 minute'`); err != nil {
		t.Fatal(err)
	}
	if _, err := d1.ProcessBatch(context.Background(), "recovery"); err != nil {
		t.Fatal(err)
	}
	if len(sender.requests) != 2 {
		t.Fatalf("lease recovery sends=%d", len(sender.requests))
	}
}

func TestDispatcherDoesNotSendClaimedDeliveryCancelledBeforeItsTurn(t *testing.T) {
	pool := dispatcherPool(t)
	deadline := time.Now().Add(6 * time.Hour)
	seedDelivery(t, pool, "pending", deadline)
	seedSecondDelivery(t, pool, deadline)
	sender := &blockingSender{started: make(chan int, 2), release: make(chan struct{}, 2)}
	d := NewDispatcher(pool, sender, DispatcherOptions{Now: time.Now, BatchSize: 2, LeaseDuration: time.Minute})

	done := make(chan error, 1)
	go func() { _, err := d.ProcessBatch(context.Background(), "worker"); done <- err }()
	if call := <-sender.started; call != 0 {
		t.Fatalf("first call = %d", call)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE push_deliveries SET status='cancelled',lease_owner=NULL,lease_expires_at=NULL WHERE id='40000000-0000-0000-0000-000000000002'`); err != nil {
		t.Fatal(err)
	}
	sender.release <- struct{}{}
	select {
	case <-sender.started:
		sender.release <- struct{}{}
	case <-time.After(500 * time.Millisecond):
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := sender.requestCount(); got != 1 {
		t.Fatalf("send calls = %d, want 1", got)
	}
	var status string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM push_deliveries WHERE id='40000000-0000-0000-0000-000000000002'`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "cancelled" {
		t.Fatalf("cancelled delivery status = %s", status)
	}
}

func TestDispatcherStaleWorkerCannotOverwriteRecoveredSuccess(t *testing.T) {
	pool := dispatcherPool(t)
	seedDelivery(t, pool, "pending", time.Now().Add(6*time.Hour))
	now := time.Now().UTC().Add(time.Second)
	staleSender := &blockingSender{results: []ProviderResult{{Class: ResultPermanentFailure}}, started: make(chan int, 1), release: make(chan struct{}, 1)}
	stale := NewDispatcher(pool, staleSender, DispatcherOptions{Now: func() time.Time { return now }, BatchSize: 1, LeaseDuration: time.Minute})

	done := make(chan error, 1)
	go func() { _, err := stale.ProcessBatch(context.Background(), "appview"); done <- err }()
	<-staleSender.started
	now = now.Add(2 * time.Minute)
	freshSender := &blockingSender{started: make(chan int, 1), release: make(chan struct{}, 1)}
	fresh := NewDispatcher(pool, freshSender, DispatcherOptions{Now: func() time.Time { return now }, BatchSize: 1, LeaseDuration: time.Minute})
	freshDone := make(chan error, 1)
	go func() { _, err := fresh.ProcessBatch(context.Background(), "appview"); freshDone <- err }()
	<-freshSender.started
	staleSender.release <- struct{}{}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	freshSender.release <- struct{}{}
	if err := <-freshDone; err != nil {
		t.Fatal(err)
	}
	var status, owner sql.NullString
	if err := pool.QueryRow(context.Background(), `SELECT status,lease_owner FROM push_deliveries`).Scan(&status, &owner); err != nil {
		t.Fatal(err)
	}
	if status.String != "succeeded" || owner.Valid {
		t.Fatalf("status=%s owner=%v", status.String, owner)
	}
}

func TestDispatcherCannotFinalizeAfterItsLeaseExpires(t *testing.T) {
	pool := dispatcherPool(t)
	seedDelivery(t, pool, "pending", time.Now().Add(6*time.Hour))
	now := time.Now().UTC().Add(time.Second)
	sender := &blockingSender{started: make(chan int, 1), release: make(chan struct{}, 1)}
	d := NewDispatcher(pool, sender, DispatcherOptions{Now: func() time.Time { return now }, BatchSize: 1, LeaseDuration: time.Minute})
	done := make(chan error, 1)
	go func() { _, err := d.ProcessBatch(context.Background(), "appview"); done <- err }()
	<-sender.started
	now = now.Add(2 * time.Minute)
	sender.release <- struct{}{}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	var status string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM push_deliveries`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status == "succeeded" {
		t.Fatal("provider result finalized after lease expiry")
	}
}

func TestDispatcherInvalidTokenResultCannotDeactivateRotatedInstallation(t *testing.T) {
	pool := dispatcherPool(t)
	seedDelivery(t, pool, "pending", time.Now().Add(6*time.Hour))
	sender := &blockingSender{results: []ProviderResult{{Class: ResultInvalidToken}}, started: make(chan int, 1), release: make(chan struct{}, 1)}
	d := NewDispatcher(pool, sender, DispatcherOptions{Now: time.Now, BatchSize: 1, LeaseDuration: time.Minute})

	done := make(chan error, 1)
	go func() { _, err := d.ProcessBatch(context.Background(), "worker"); done <- err }()
	<-sender.started
	if _, err := pool.Exec(context.Background(), `UPDATE push_installations SET fcm_token='rotated-token',updated_at=now() WHERE id='10000000-0000-0000-0000-000000000001'`); err != nil {
		t.Fatal(err)
	}
	sender.release <- struct{}{}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	var active bool
	var token, status string
	if err := pool.QueryRow(context.Background(), `SELECT i.active,i.fcm_token,d.status FROM push_installations i JOIN push_account_subscriptions s ON s.installation_id=i.id JOIN push_deliveries d ON d.account_subscription_id=s.id`).Scan(&active, &token, &status); err != nil {
		t.Fatal(err)
	}
	if !active || token != "rotated-token" || status == "permanent_failure" {
		t.Fatalf("active=%v token=%s status=%s", active, token, status)
	}
}

func TestDispatcherSuccessForOldTokenCannotFinalizeAfterRotation(t *testing.T) {
	pool := dispatcherPool(t)
	seedDelivery(t, pool, "pending", time.Now().Add(6*time.Hour))
	sender := &blockingSender{started: make(chan int, 1), release: make(chan struct{}, 1)}
	d := NewDispatcher(pool, sender, DispatcherOptions{Now: time.Now, BatchSize: 1, LeaseDuration: time.Minute})
	done := make(chan error, 1)
	go func() { _, err := d.ProcessBatch(context.Background(), "worker"); done <- err }()
	<-sender.started
	if _, err := pool.Exec(context.Background(), `UPDATE push_installations SET fcm_token='rotated-token',updated_at=now() WHERE id='10000000-0000-0000-0000-000000000001'`); err != nil {
		t.Fatal(err)
	}
	sender.release <- struct{}{}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	var status string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM push_deliveries`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status == "succeeded" {
		t.Fatal("old-token result finalized after token rotation")
	}
}

func TestDispatcherRechecksDeadlineBeforeEachSend(t *testing.T) {
	pool := dispatcherPool(t)
	base := time.Now().UTC().Add(time.Second)
	seedDelivery(t, pool, "pending", base.Add(time.Hour))
	seedSecondDelivery(t, pool, base.Add(time.Second))
	sender := &advancingSender{now: &base, advance: 2 * time.Second}
	d := NewDispatcher(pool, sender, DispatcherOptions{Now: func() time.Time { return base }, BatchSize: 2, LeaseDuration: time.Minute})
	if n, err := d.ProcessBatch(context.Background(), "worker"); err != nil || n != 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if len(sender.requests) != 1 {
		t.Fatalf("send calls=%d, want 1", len(sender.requests))
	}
	if sender.requests[0].TTL <= 0 || sender.requests[0].TTL > time.Hour {
		t.Fatalf("first TTL=%s", sender.requests[0].TTL)
	}
	var status string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM push_deliveries WHERE id='40000000-0000-0000-0000-000000000002'`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "expired" {
		t.Fatalf("second status=%s, want expired", status)
	}
}

func TestDispatcherBoundsInFlightSendByAbsoluteDeadline(t *testing.T) {
	pool := dispatcherPool(t)
	deadline := time.Now().UTC().Add(250 * time.Millisecond)
	seedDelivery(t, pool, "pending", deadline)
	sender := &contextBlockingSender{}
	d := NewDispatcher(pool, sender, DispatcherOptions{Now: time.Now, BatchSize: 1, LeaseDuration: time.Minute, SendTimeout: time.Second})
	started := time.Now()
	if n, err := d.ProcessBatch(context.Background(), "appview"); err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	elapsed := time.Since(started)
	if elapsed >= 600*time.Millisecond {
		t.Fatalf("send ran %s, beyond absolute deadline", elapsed)
	}
	if sender.deadline.After(deadline.Add(25 * time.Millisecond)) {
		t.Fatalf("provider context deadline=%s delivery deadline=%s", sender.deadline, deadline)
	}
	if sender.request.TTL <= 0 || sender.request.TTL > deadline.Sub(started)+50*time.Millisecond {
		t.Fatalf("provider TTL=%s", sender.request.TTL)
	}
	var status string
	var updatedAt time.Time
	if err := pool.QueryRow(context.Background(), `SELECT status,updated_at FROM push_deliveries`).Scan(&status, &updatedAt); err != nil {
		t.Fatal(err)
	}
	if status != "expired" || updatedAt.Before(deadline.Add(-25*time.Millisecond)) {
		t.Fatalf("status=%s updatedAt=%s deadline=%s", status, updatedAt, deadline)
	}
}

func TestDispatcherRunRecoversFromTransientStoreFailure(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE bluesky_profiles(did TEXT PRIMARY KEY,display_name TEXT,avatar_cid TEXT);
		CREATE TABLE craftsky_posts(uri TEXT PRIMARY KEY,reply_root_uri TEXT,reply_parent_uri TEXT);
		CREATE TABLE actor_mutes(owner_did TEXT NOT NULL, subject_did TEXT NOT NULL, PRIMARY KEY(owner_did, subject_did));
		CREATE TABLE atproto_blocks(uri TEXT PRIMARY KEY, blocker_did TEXT NOT NULL, subject_did TEXT NOT NULL);
	`)
	sender := &scriptedSender{}
	d := NewDispatcher(pool, sender, DispatcherOptions{Now: time.Now, BatchSize: 1, LeaseDuration: time.Minute})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- d.Run(ctx, 10*time.Millisecond, "worker") }()

	select {
	case err := <-done:
		t.Fatalf("worker exited on transient store failure: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	seedDelivery(t, pool, "pending", time.Now().Add(6*time.Hour))
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && sender.requestCount() == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if sender.requestCount() != 1 {
		t.Fatalf("send calls=%d after store recovery", sender.requestCount())
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Run error=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not stop after cancellation")
	}
}

func TestDispatcherObservesPersistedQueueDepthAndOldestAge(t *testing.T) {
	pool := dispatcherPool(t)
	base := time.Now().UTC().Add(time.Second)
	seedDelivery(t, pool, "pending", base.Add(6*time.Hour))
	seedSecondDelivery(t, pool, base.Add(6*time.Hour))
	if _, err := pool.Exec(context.Background(), `UPDATE push_deliveries SET created_at=$1`, base.Add(-2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	observer := &queueObserver{}
	d := NewDispatcher(pool, &scriptedSender{}, DispatcherOptions{Now: func() time.Time { return base }, BatchSize: 1, LeaseDuration: time.Minute, Observer: observer})
	if _, err := d.ProcessBatch(context.Background(), "worker"); err != nil {
		t.Fatal(err)
	}
	if observer.pending != 2 || observer.age < 2*time.Minute || observer.age >= 2*time.Minute+time.Second {
		t.Fatalf("pending=%d age=%s", observer.pending, observer.age)
	}
}

func TestDispatcherTelemetryNeverExposesProviderSentinels(t *testing.T) {
	pool := dispatcherPool(t)
	seedDelivery(t, pool, "pending", time.Now().Add(6*time.Hour))
	const sentinel = "SENTINEL_SECRET_FCM_TOKEN"
	if _, err := pool.Exec(context.Background(), `UPDATE push_installations SET fcm_token=$1`, sentinel); err != nil {
		t.Fatal(err)
	}
	observer := &privacyObserver{}
	d := NewDispatcher(pool, sentinelFailureSender{sentinel: sentinel}, DispatcherOptions{Now: time.Now, BatchSize: 1, LeaseDuration: time.Minute, Observer: observer})
	if _, err := d.ProcessBatch(context.Background(), "worker"); err != nil {
		t.Fatal(err)
	}
	var resultClass string
	if err := pool.QueryRow(context.Background(), `SELECT provider_result_class FROM push_deliveries`).Scan(&resultClass); err != nil {
		t.Fatal(err)
	}
	observable := strings.Join(append(observer.values, resultClass), " ")
	if strings.Contains(observable, sentinel) || strings.Contains(observable, "did:plc:") {
		t.Fatalf("telemetry leaked sentinel: %s", observable)
	}
}

func dispatcherPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := testdb.WithSchema(t, `
		CREATE TABLE bluesky_profiles(did TEXT PRIMARY KEY,display_name TEXT,avatar_cid TEXT);
		CREATE TABLE craftsky_posts(uri TEXT PRIMARY KEY,reply_root_uri TEXT,reply_parent_uri TEXT);
		CREATE TABLE actor_mutes(owner_did TEXT NOT NULL, subject_did TEXT NOT NULL, PRIMARY KEY(owner_did, subject_did));
		CREATE TABLE atproto_blocks(uri TEXT PRIMARY KEY, blocker_did TEXT NOT NULL, subject_did TEXT NOT NULL);
	`)
	migration, err := os.ReadFile("../../migrations/000021_appview_notifications.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	return pool
}

func seedDelivery(t *testing.T, pool *pgxpool.Pool, status string, deadline time.Time) {
	t.Helper()
	statements := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO bluesky_profiles(did,display_name) VALUES('did:plc:actor','Alice')`, nil},
		{`INSERT INTO notification_events(id,recipient_did,actor_did,category,subject_key,source_uri,source_cid,source_rkey,eligibility_scope,recipient_followed_actor,push_enabled_snapshot,state,first_activity_at,activity_at,indexed_at,initial_push_evaluated_at) VALUES('00000000-0000-0000-0000-000000000001','did:plc:viewer','did:plc:actor','like','subject','source','cid','r','everyone',false,true,'active',now(),now(),now(),now())`, nil},
		{`INSERT INTO push_installations(id,device_id,platform,fcm_token) VALUES('10000000-0000-0000-0000-000000000001','device','ios','secret-token')`, nil},
		{`INSERT INTO push_account_subscriptions(id,installation_id,account_did,routing_id) VALUES('20000000-0000-0000-0000-000000000001','10000000-0000-0000-0000-000000000001','did:plc:viewer','30000000-0000-0000-0000-000000000001')`, nil},
		{`INSERT INTO push_deliveries(id,notification_id,account_subscription_id,status,next_attempt_at,deadline_at) VALUES('40000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000001',$1,now(),$2)`, []any{status, deadline}},
	}
	for _, statement := range statements {
		if _, err := pool.Exec(context.Background(), statement.sql, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
}

func seedSecondDelivery(t *testing.T, pool *pgxpool.Pool, deadline time.Time) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO notification_events(id,recipient_did,actor_did,category,subject_key,source_uri,source_cid,source_rkey,eligibility_scope,recipient_followed_actor,push_enabled_snapshot,state,first_activity_at,activity_at,indexed_at,initial_push_evaluated_at)
		VALUES('00000000-0000-0000-0000-000000000002','did:plc:viewer','did:plc:actor','like','subject-2','source-2','cid-2','r-2','everyone',false,true,'active',now(),now(),now(),now())`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO push_deliveries(id,notification_id,account_subscription_id,status,next_attempt_at,deadline_at)
		VALUES('40000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000002','20000000-0000-0000-0000-000000000001','pending',now(),$1)`, deadline); err != nil {
		t.Fatal(err)
	}
}
