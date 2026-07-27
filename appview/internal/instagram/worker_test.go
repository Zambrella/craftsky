package instagram

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/integrations/instagrammeta"
)

func TestWebhookWorkerRedeemsChecksMembershipLooksUpAndCompletesBeforeOptionalReply(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	events := make([]string, 0, 8)
	work := syntheticClaimedWebhookWork(now)
	queue := &fakeWebhookQueue{claimed: []WebhookWork{work}, events: &events}
	redeemer := &fakeWebhookRedeemer{
		redemption: WebhookRedemption{
			AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000101"),
			OwnerDID:  syntax.DID("did:plc:synthetic-owner"),
		},
		events: &events,
	}
	membership := &fakeWebhookMembership{current: true, events: &events}
	meta := &fakeMetaClient{username: "synthetic.crafter", events: &events}
	worker, err := NewWebhookWorker(queue, redeemer, membership, meta, WebhookWorkerOptions{
		BatchSize:   1,
		Now:         func() time.Time { return now },
		ReplyText:   "Synthetic verification reply.",
		ReplyWindow: 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewWebhookWorker: %v", err)
	}

	processed, err := worker.ProcessBatch(context.Background())
	if err != nil {
		t.Fatalf("ProcessBatch: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}
	wantEvents := []string{"claim", "redeem", "membership", "lookup", "candidate", "complete", "reply"}
	if fmt.Sprint(events) != fmt.Sprint(wantEvents) {
		t.Fatalf("events = %v, want %v", events, wantEvents)
	}
	if queue.completed != work.ID || redeemer.candidateUsername != "synthetic.crafter" || meta.replyRecipient != work.SenderIGSID {
		t.Fatal("worker did not persist candidate/complete/reply expected work")
	}
	diagnostic := fmt.Sprintf("worker=%v/%+v/%#v work=%v", worker, worker, worker, work)
	for _, private := range []string{work.SenderIGSID, "synthetic.crafter", meta.replyRecipient} {
		if strings.Contains(diagnostic, private) {
			t.Fatalf("worker diagnostic leaked %q: %s", private, diagnostic)
		}
	}
}

func TestWebhookWorkerRetriesTransientProviderFailureWithFixedBackoff(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	events := make([]string, 0, 8)
	work := syntheticClaimedWebhookWork(now)
	queue := &fakeWebhookQueue{claimed: []WebhookWork{work}, events: &events}
	redeemer := &fakeWebhookRedeemer{
		redemption: WebhookRedemption{
			AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000102"),
			OwnerDID:  syntax.DID("did:plc:synthetic-owner"),
		},
		events: &events,
	}
	membership := &fakeWebhookMembership{current: true, events: &events}
	meta := &fakeMetaClient{
		lookupErr: fakeProviderError{kind: instagrammeta.ProviderErrorRateLimited, retryAfter: 3 * time.Second},
		events:    &events,
	}
	worker, err := NewWebhookWorker(queue, redeemer, membership, meta, WebhookWorkerOptions{
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewWebhookWorker: %v", err)
	}

	processed, err := worker.ProcessBatch(context.Background())
	if err != nil {
		t.Fatalf("ProcessBatch: %v", err)
	}
	if processed != 1 || queue.retried != work.ID || !queue.nextRetry.Equal(now.Add(3*time.Second)) {
		t.Fatalf("retry result = processed %d id %s next %s", processed, queue.retried, queue.nextRetry)
	}
	wantEvents := []string{"claim", "redeem", "membership", "lookup", "retry"}
	if fmt.Sprint(events) != fmt.Sprint(wantEvents) {
		t.Fatalf("events = %v, want %v", events, wantEvents)
	}
}

func TestWebhookWorkerIgnoresUnavailableChallengeWithoutProviderCall(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	events := make([]string, 0, 4)
	work := syntheticClaimedWebhookWork(now)
	queue := &fakeWebhookQueue{claimed: []WebhookWork{work}, events: &events}
	redeemer := &fakeWebhookRedeemer{redeemErr: ErrInstagramResourceNotFound, events: &events}
	worker, err := NewWebhookWorker(
		queue,
		redeemer,
		&fakeWebhookMembership{current: true, events: &events},
		&fakeMetaClient{events: &events},
		WebhookWorkerOptions{Now: func() time.Time { return now }},
	)
	if err != nil {
		t.Fatalf("NewWebhookWorker: %v", err)
	}
	processed, err := worker.ProcessBatch(context.Background())
	if err != nil {
		t.Fatalf("ProcessBatch: %v", err)
	}
	if processed != 1 || queue.ignored != work.ID || queue.terminalReason != WebhookReasonChallengeUnavailable {
		t.Fatalf("ignore result = processed %d ignored %s reason %q", processed, queue.ignored, queue.terminalReason)
	}
	wantEvents := []string{"claim", "redeem", "ignore"}
	if fmt.Sprint(events) != fmt.Sprint(wantEvents) {
		t.Fatalf("events = %v, want %v", events, wantEvents)
	}
}

func TestWebhookWorkerRateLimitsInvalidRedemptionsAndMetaLookupsWithoutProviderCall(t *testing.T) {
	t.Parallel()

	t.Run("invalid redemption excess is terminally ignored", func(t *testing.T) {
		t.Parallel()
		now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
		events := make([]string, 0, 5)
		work := syntheticClaimedWebhookWork(now)
		queue := &fakeWebhookQueue{claimed: []WebhookWork{work}, events: &events}
		limiter := &fakeWebhookIdentifierLimiter{
			decision: RateLimitDecision{Allowed: false, RetryAfter: time.Minute},
			events:   &events,
		}
		worker, err := NewWebhookWorker(
			queue,
			&fakeWebhookRedeemer{redeemErr: ErrInstagramResourceNotFound, events: &events},
			&fakeWebhookMembership{current: true, events: &events},
			&fakeMetaClient{events: &events},
			WebhookWorkerOptions{Now: func() time.Time { return now }, RateLimiter: limiter},
		)
		if err != nil {
			t.Fatalf("NewWebhookWorker: %v", err)
		}
		processed, err := worker.ProcessBatch(context.Background())
		if err != nil || processed != 1 || queue.terminalReason != WebhookReasonRateLimited {
			t.Fatalf("result = processed %d err %v reason %q", processed, err, queue.terminalReason)
		}
		wantEvents := []string{"claim", "redeem", "limit", "ignore"}
		if fmt.Sprint(events) != fmt.Sprint(wantEvents) || limiter.scope != RateLimitInvalidRedemptionIGSID || limiter.identifier != work.SenderIGSID {
			t.Fatalf("events/limit = %v scope %q identifier match %t", events, limiter.scope, limiter.identifier == work.SenderIGSID)
		}
	})

	t.Run("Meta lookup excess retries without provider call", func(t *testing.T) {
		t.Parallel()
		now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
		events := make([]string, 0, 7)
		work := syntheticClaimedWebhookWork(now)
		queue := &fakeWebhookQueue{claimed: []WebhookWork{work}, events: &events}
		limiter := &fakeWebhookIdentifierLimiter{
			decision: RateLimitDecision{Allowed: false, RetryAfter: 3 * time.Second},
			events:   &events,
		}
		worker, err := NewWebhookWorker(
			queue,
			&fakeWebhookRedeemer{
				redemption: WebhookRedemption{AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000120"), OwnerDID: syntax.DID("did:plc:synthetic-owner")},
				events:     &events,
			},
			&fakeWebhookMembership{current: true, events: &events},
			&fakeMetaClient{username: "must.not.be.looked.up", events: &events},
			WebhookWorkerOptions{Now: func() time.Time { return now }, RateLimiter: limiter},
		)
		if err != nil {
			t.Fatalf("NewWebhookWorker: %v", err)
		}
		processed, err := worker.ProcessBatch(context.Background())
		if err != nil || processed != 1 || !queue.nextRetry.Equal(now.Add(3*time.Second)) {
			t.Fatalf("result = processed %d err %v retry %s", processed, err, queue.nextRetry)
		}
		wantEvents := []string{"claim", "redeem", "membership", "limit", "retry"}
		if fmt.Sprint(events) != fmt.Sprint(wantEvents) || limiter.scope != RateLimitMetaLookupIGSID || limiter.identifier != work.SenderIGSID {
			t.Fatalf("events/limit = %v scope %q identifier match %t", events, limiter.scope, limiter.identifier == work.SenderIGSID)
		}
	})
}

func TestWebhookWorkerInactivatesDepartedOwnerAndRetriesMembershipFailure(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name       string
		membership *fakeWebhookMembership
		wantEvents []string
		wantIgnore bool
		wantRetry  bool
	}{
		{
			name:       "departed",
			membership: &fakeWebhookMembership{current: false},
			wantEvents: []string{"claim", "redeem", "membership", "full-inactivate", "inactivate", "ignore"},
			wantIgnore: true,
		},
		{
			name:       "membership unavailable",
			membership: &fakeWebhookMembership{err: errors.New("synthetic membership storage failure")},
			wantEvents: []string{"claim", "redeem", "membership", "retry"},
			wantRetry:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
			events := make([]string, 0, 8)
			work := syntheticClaimedWebhookWork(now)
			queue := &fakeWebhookQueue{claimed: []WebhookWork{work}, events: &events}
			redeemer := &fakeWebhookRedeemer{
				redemption: WebhookRedemption{
					AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000103"),
					OwnerDID:  syntax.DID("did:plc:synthetic-owner"),
				},
				events: &events,
			}
			test.membership.events = &events
			worker, err := NewWebhookWorker(queue, redeemer, test.membership, &fakeMetaClient{events: &events}, WebhookWorkerOptions{Now: func() time.Time { return now }})
			if err != nil {
				t.Fatalf("NewWebhookWorker: %v", err)
			}
			processed, err := worker.ProcessBatch(context.Background())
			if err != nil {
				t.Fatalf("ProcessBatch: %v", err)
			}
			if processed != 1 || (queue.ignored != uuid.Nil) != test.wantIgnore || (queue.retried != uuid.Nil) != test.wantRetry {
				t.Fatalf("result = processed %d ignored %s retried %s", processed, queue.ignored, queue.retried)
			}
			if test.wantIgnore && queue.terminalReason != WebhookReasonMembershipInactive {
				t.Fatalf("terminal reason = %q", queue.terminalReason)
			}
			if fmt.Sprint(events) != fmt.Sprint(test.wantEvents) {
				t.Fatalf("events = %v, want %v", events, test.wantEvents)
			}
		})
	}
}

func TestWebhookWorkerBoundsAndRedactsTerminalProviderFailures(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name       string
		kind       instagrammeta.ProviderErrorKind
		attempts   int
		wantCode   AttemptRetryCode
		wantReason WebhookTerminalReason
	}{
		{name: "invalid profile", kind: instagrammeta.ProviderErrorInvalidResponse, attempts: 1, wantCode: RetryInvalidProfileResponse, wantReason: WebhookReasonInvalidProfile},
		{name: "authentication", kind: instagrammeta.ProviderErrorAuthentication, attempts: 1, wantCode: RetryProfileLookupUnavailable, wantReason: WebhookReasonProviderPermanent},
		{name: "transient exhaustion", kind: instagrammeta.ProviderErrorTransient, attempts: WebhookMaxAttempts, wantCode: RetryProfileLookupUnavailable, wantReason: WebhookReasonMaxAttempts},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
			events := make([]string, 0, 8)
			work := syntheticClaimedWebhookWork(now)
			work.Attempts = test.attempts
			queue := &fakeWebhookQueue{claimed: []WebhookWork{work}, events: &events}
			redeemer := &fakeWebhookRedeemer{
				redemption: WebhookRedemption{
					AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000104"),
					OwnerDID:  syntax.DID("did:plc:synthetic-owner"),
				},
				events: &events,
			}
			worker, err := NewWebhookWorker(
				queue,
				redeemer,
				&fakeWebhookMembership{current: true, events: &events},
				&fakeMetaClient{lookupErr: fakeProviderError{kind: test.kind}, events: &events},
				WebhookWorkerOptions{Now: func() time.Time { return now }},
			)
			if err != nil {
				t.Fatalf("NewWebhookWorker: %v", err)
			}
			processed, err := worker.ProcessBatch(context.Background())
			if err != nil {
				t.Fatalf("ProcessBatch: %v", err)
			}
			if processed != 1 || queue.failed != work.ID || redeemer.retryCode != test.wantCode || queue.terminalReason != test.wantReason {
				t.Fatalf("terminal result = processed %d failed %s code %q reason %q", processed, queue.failed, redeemer.retryCode, queue.terminalReason)
			}
			wantEvents := []string{"claim", "redeem", "membership", "lookup", "reject", "fail"}
			if fmt.Sprint(events) != fmt.Sprint(wantEvents) {
				t.Fatalf("events = %v, want %v", events, wantEvents)
			}
		})
	}
}

func TestWebhookWorkerDoesNotProcessAtFifteenMinuteAgeBoundary(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 19, 12, 15, 0, 0, time.UTC)
	events := make([]string, 0, 4)
	work := syntheticClaimedWebhookWork(now)
	work.ProcessingStartedAt = now.Add(-WebhookMaxProcessingAge)
	queue := &fakeWebhookQueue{claimed: []WebhookWork{work}, events: &events}
	worker, err := NewWebhookWorker(
		queue,
		&fakeWebhookRedeemer{events: &events},
		&fakeWebhookMembership{current: true, events: &events},
		&fakeMetaClient{events: &events},
		WebhookWorkerOptions{Now: func() time.Time { return now }},
	)
	if err != nil {
		t.Fatalf("NewWebhookWorker: %v", err)
	}
	processed, err := worker.ProcessBatch(context.Background())
	if err != nil {
		t.Fatalf("ProcessBatch: %v", err)
	}
	if processed != 1 || queue.failed != work.ID || queue.terminalReason != WebhookReasonMaxAge {
		t.Fatalf("age result = processed %d failed %s reason %q", processed, queue.failed, queue.terminalReason)
	}
	wantEvents := []string{"claim", "fail"}
	if fmt.Sprint(events) != fmt.Sprint(wantEvents) {
		t.Fatalf("events = %v, want %v", events, wantEvents)
	}
}

func TestWebhookWorkerHandlesCandidateValidationAndStorageFailures(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name         string
		candidateErr error
		wantEvents   []string
		wantFailed   bool
		wantRetry    bool
	}{
		{
			name:         "invalid profile response",
			candidateErr: ErrInvalidInstagramUsername,
			wantEvents:   []string{"claim", "redeem", "membership", "lookup", "candidate", "reject", "fail"},
			wantFailed:   true,
		},
		{
			name:         "candidate storage unavailable",
			candidateErr: errors.New("synthetic candidate storage failure"),
			wantEvents:   []string{"claim", "redeem", "membership", "lookup", "candidate", "retry"},
			wantRetry:    true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
			events := make([]string, 0, 8)
			work := syntheticClaimedWebhookWork(now)
			queue := &fakeWebhookQueue{claimed: []WebhookWork{work}, events: &events}
			redeemer := &fakeWebhookRedeemer{
				redemption: WebhookRedemption{
					AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000105"),
					OwnerDID:  syntax.DID("did:plc:synthetic-owner"),
				},
				candidateErr: test.candidateErr,
				events:       &events,
			}
			worker, err := NewWebhookWorker(
				queue,
				redeemer,
				&fakeWebhookMembership{current: true, events: &events},
				&fakeMetaClient{username: "synthetic.crafter", events: &events},
				WebhookWorkerOptions{Now: func() time.Time { return now }},
			)
			if err != nil {
				t.Fatalf("NewWebhookWorker: %v", err)
			}
			processed, err := worker.ProcessBatch(context.Background())
			if err != nil {
				t.Fatalf("ProcessBatch: %v", err)
			}
			if processed != 1 || (queue.failed != uuid.Nil) != test.wantFailed || (queue.retried != uuid.Nil) != test.wantRetry {
				t.Fatalf("result = processed %d failed %s retried %s", processed, queue.failed, queue.retried)
			}
			if fmt.Sprint(events) != fmt.Sprint(test.wantEvents) {
				t.Fatalf("events = %v, want %v", events, test.wantEvents)
			}
		})
	}
}

func TestWebhookWorkerCancellationDoesNotBecomeProviderFailure(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name        string
		cancelFirst bool
		lookupErr   error
		wantEvents  []string
	}{
		{name: "before claim", cancelFirst: true, wantEvents: nil},
		{name: "provider call", lookupErr: context.Canceled, wantEvents: []string{"claim", "redeem", "membership", "lookup"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			events := make([]string, 0, 6)
			work := syntheticClaimedWebhookWork(now)
			queue := &fakeWebhookQueue{claimed: []WebhookWork{work}, events: &events}
			redeemer := &fakeWebhookRedeemer{
				redemption: WebhookRedemption{
					AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000106"),
					OwnerDID:  syntax.DID("did:plc:synthetic-owner"),
				},
				events: &events,
			}
			worker, err := NewWebhookWorker(
				queue,
				redeemer,
				&fakeWebhookMembership{current: true, events: &events},
				&fakeMetaClient{lookupErr: test.lookupErr, events: &events},
				WebhookWorkerOptions{Now: func() time.Time { return now }},
			)
			if err != nil {
				t.Fatalf("NewWebhookWorker: %v", err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			if test.cancelFirst {
				cancel()
			} else {
				defer cancel()
			}
			processed, err := worker.ProcessBatch(ctx)
			if !errors.Is(err, context.Canceled) || processed != 0 {
				t.Fatalf("ProcessBatch cancellation = (%d, %v)", processed, err)
			}
			if queue.failed != uuid.Nil || queue.retried != uuid.Nil || queue.completed != uuid.Nil {
				t.Fatal("cancellation changed durable provider outcome")
			}
			if fmt.Sprint(events) != fmt.Sprint(test.wantEvents) {
				t.Fatalf("events = %v, want %v", events, test.wantEvents)
			}
		})
	}
}

func TestWebhookWorkerReplyIsWindowBoundOptionalAndAfterDurableCompletion(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name        string
		eventAge    time.Duration
		completeErr error
		replyErr    error
		wantReply   bool
		wantError   bool
	}{
		{name: "inside window optional failure", eventAge: WebhookMaxReplyWindow - time.Nanosecond, replyErr: errors.New("synthetic reply failure"), wantReply: true},
		{name: "at window boundary", eventAge: WebhookMaxReplyWindow},
		{name: "future event", eventAge: -time.Second},
		{name: "completion failed", eventAge: time.Minute, completeErr: errors.New("synthetic completion failure"), wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
			events := make([]string, 0, 8)
			work := syntheticClaimedWebhookWork(now)
			work.EventAt = now.Add(-test.eventAge)
			queue := &fakeWebhookQueue{claimed: []WebhookWork{work}, events: &events, completeErr: test.completeErr}
			redeemer := &fakeWebhookRedeemer{
				redemption: WebhookRedemption{
					AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000107"),
					OwnerDID:  syntax.DID("did:plc:synthetic-owner"),
				},
				events: &events,
			}
			meta := &fakeMetaClient{username: "synthetic.crafter", replyErr: test.replyErr, events: &events}
			worker, err := NewWebhookWorker(
				queue,
				redeemer,
				&fakeWebhookMembership{current: true, events: &events},
				meta,
				WebhookWorkerOptions{
					Now:         func() time.Time { return now },
					ReplyText:   "Synthetic reply.",
					ReplyWindow: WebhookMaxReplyWindow,
				},
			)
			if err != nil {
				t.Fatalf("NewWebhookWorker: %v", err)
			}
			processed, err := worker.ProcessBatch(context.Background())
			if (err != nil) != test.wantError {
				t.Fatalf("ProcessBatch = (%d, %v), want error %t", processed, err, test.wantError)
			}
			gotReply := meta.replyRecipient != ""
			if gotReply != test.wantReply {
				t.Fatalf("reply called = %t, want %t; events %v", gotReply, test.wantReply, events)
			}
			if gotReply {
				completeIndex, replyIndex := -1, -1
				for index, event := range events {
					if event == "complete" {
						completeIndex = index
					}
					if event == "reply" {
						replyIndex = index
					}
				}
				if completeIndex < 0 || replyIndex <= completeIndex {
					t.Fatalf("reply was not after durable completion: %v", events)
				}
			}
		})
	}
}

func TestWebhookWorkerDoesNotFinalizeAfterLeaseExpiresDuringProviderCall(t *testing.T) {
	t.Parallel()

	clock := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	events := make([]string, 0, 8)
	work := syntheticClaimedWebhookWork(clock)
	queue := &fakeWebhookQueue{claimed: []WebhookWork{work}, events: &events}
	redeemer := &fakeWebhookRedeemer{
		redemption: WebhookRedemption{
			AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000108"),
			OwnerDID:  syntax.DID("did:plc:synthetic-owner"),
		},
		events: &events,
	}
	meta := &fakeMetaClient{
		username: "synthetic.crafter",
		events:   &events,
		onLookup: func() { clock = work.LeaseExpiresAt },
	}
	worker, err := NewWebhookWorker(
		queue,
		redeemer,
		&fakeWebhookMembership{current: true, events: &events},
		meta,
		WebhookWorkerOptions{Now: func() time.Time { return clock }},
	)
	if err != nil {
		t.Fatalf("NewWebhookWorker: %v", err)
	}
	processed, err := worker.ProcessBatch(context.Background())
	if !errors.Is(err, ErrWebhookLeaseLost) || processed != 0 {
		t.Fatalf("ProcessBatch expired lease = (%d, %v)", processed, err)
	}
	if queue.completed != uuid.Nil || queue.failed != uuid.Nil || queue.retried != uuid.Nil || redeemer.candidateUsername != "" {
		t.Fatal("expired worker changed candidate or durable queue state")
	}
	wantEvents := []string{"claim", "redeem", "membership", "lookup"}
	if fmt.Sprint(events) != fmt.Sprint(wantEvents) {
		t.Fatalf("events = %v, want %v", events, wantEvents)
	}
}

func TestWebhookWorkerProcessesDurableWorkAndClearsSensitiveFields(t *testing.T) {
	store := newWebhookTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	input := syntheticWebhookItem(0xe1, 0xf1, "synthetic-durable-sender", now.Add(-time.Minute))
	if inserted, err := store.EnqueueWebhookWork(ctx, []instagrammeta.WorkItem{input}, now); err != nil || inserted != 1 {
		t.Fatalf("enqueue = (%d, %v)", inserted, err)
	}
	events := make([]string, 0, 6)
	redeemer := &fakeWebhookRedeemer{
		redemption: WebhookRedemption{
			AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000109"),
			OwnerDID:  syntax.DID("did:plc:synthetic-owner"),
		},
		events: &events,
	}
	worker, err := NewWebhookWorker(
		store,
		redeemer,
		&fakeWebhookMembership{current: true, events: &events},
		&fakeMetaClient{username: "synthetic.durable", events: &events},
		WebhookWorkerOptions{Now: func() time.Time { return now }},
	)
	if err != nil {
		t.Fatalf("NewWebhookWorker: %v", err)
	}
	processed, err := worker.ProcessBatch(ctx)
	if err != nil || processed != 1 {
		t.Fatalf("ProcessBatch = (%d, %v)", processed, err)
	}
	var id uuid.UUID
	if err := store.pool.QueryRow(ctx, `SELECT id FROM instagram_webhook_work`).Scan(&id); err != nil {
		t.Fatalf("select work ID: %v", err)
	}
	assertWebhookWorkRow(t, store, id, WebhookWorkCompleted, false)
	if redeemer.candidateUsername != "synthetic.durable" {
		t.Fatalf("candidate username = %q", redeemer.candidateUsername)
	}
}

func TestWebhookWorkerConfigurationKeepsFixedSafetyMaxima(t *testing.T) {
	t.Parallel()

	if WebhookWorkerCount != 4 || WebhookLeaseDuration != time.Minute || WebhookMaxAttempts != 5 || WebhookMaxProcessingAge != 15*time.Minute {
		t.Fatalf("fixed worker limits changed: workers=%d lease=%s attempts=%d age=%s", WebhookWorkerCount, WebhookLeaseDuration, WebhookMaxAttempts, WebhookMaxProcessingAge)
	}
	events := make([]string, 0)
	queue := &fakeWebhookQueue{events: &events}
	redeemer := &fakeWebhookRedeemer{events: &events}
	membership := &fakeWebhookMembership{events: &events}
	meta := &fakeMetaClient{events: &events}
	for name, create := range map[string]func() (*WebhookWorker, error){
		"nil queue": func() (*WebhookWorker, error) {
			return NewWebhookWorker(nil, redeemer, membership, meta, WebhookWorkerOptions{})
		},
		"batch too large": func() (*WebhookWorker, error) {
			return NewWebhookWorker(queue, redeemer, membership, meta, WebhookWorkerOptions{BatchSize: instagrammeta.MaxSupportedEvents + 1})
		},
		"reply window too large": func() (*WebhookWorker, error) {
			return NewWebhookWorker(queue, redeemer, membership, meta, WebhookWorkerOptions{ReplyText: "synthetic", ReplyWindow: WebhookMaxReplyWindow + time.Nanosecond})
		},
		"identifier limit too large": func() (*WebhookWorker, error) {
			return NewWebhookWorker(queue, redeemer, membership, meta, WebhookWorkerOptions{RateLimiter: &fakeWebhookIdentifierLimiter{}, MetaLookupsPerIGSIDPerHour: WebhookMetaLookupIGSIDLimit + 1})
		},
		"retry policy above maximum": func() (*WebhookWorker, error) {
			return NewWebhookWorker(queue, redeemer, membership, meta, WebhookWorkerOptions{RetryPolicy: WebhookRetryPolicy{
				MaxAttempts:      WebhookMaxAttempts + 1,
				InitialBackoff:   WebhookInitialBackoff,
				MaxBackoff:       WebhookMaxBackoff,
				MaxProcessingAge: WebhookMaxProcessingAge,
			}})
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := create(); err == nil {
				t.Fatal("NewWebhookWorker accepted unsafe configuration")
			}
		})
	}
}

type fakeWebhookQueue struct {
	claimed        []WebhookWork
	claimErr       error
	events         *[]string
	completed      uuid.UUID
	ignored        uuid.UUID
	failed         uuid.UUID
	retried        uuid.UUID
	nextRetry      time.Time
	completeErr    error
	ignoreErr      error
	failErr        error
	retryErr       error
	terminalReason WebhookTerminalReason
}

func (q *fakeWebhookQueue) ClaimWebhookWork(_ context.Context, _ int, _ time.Time) ([]WebhookWork, error) {
	*q.events = append(*q.events, "claim")
	return append([]WebhookWork(nil), q.claimed...), q.claimErr
}

func (q *fakeWebhookQueue) CompleteWebhookWork(_ context.Context, id, _ uuid.UUID, _ time.Time, _ WebhookTerminalReason) error {
	*q.events = append(*q.events, "complete")
	q.completed = id
	return q.completeErr
}

func (q *fakeWebhookQueue) IgnoreWebhookWork(_ context.Context, id, _ uuid.UUID, _ time.Time, reason WebhookTerminalReason) error {
	*q.events = append(*q.events, "ignore")
	q.ignored = id
	q.terminalReason = reason
	return q.ignoreErr
}

func (q *fakeWebhookQueue) FailWebhookWork(_ context.Context, id, _ uuid.UUID, _ time.Time, reason WebhookTerminalReason) error {
	*q.events = append(*q.events, "fail")
	q.failed = id
	q.terminalReason = reason
	return q.failErr
}

func (q *fakeWebhookQueue) RetryWebhookWork(_ context.Context, id, _ uuid.UUID, next, _ time.Time) error {
	*q.events = append(*q.events, "retry")
	q.retried = id
	q.nextRetry = next
	return q.retryErr
}

type fakeWebhookRedeemer struct {
	redemption        WebhookRedemption
	redeemErr         error
	candidateErr      error
	inactivateErr     error
	rejectErr         error
	candidateUsername string
	retryCode         AttemptRetryCode
	events            *[]string
}

func (r *fakeWebhookRedeemer) RedeemWebhookChallenge(_ context.Context, _ WebhookRedemptionRequest) (WebhookRedemption, error) {
	*r.events = append(*r.events, "redeem")
	return r.redemption, r.redeemErr
}

func (r *fakeWebhookRedeemer) SetWebhookCandidate(_ context.Context, _ uuid.UUID, username string, _ time.Time) error {
	*r.events = append(*r.events, "candidate")
	r.candidateUsername = username
	return r.candidateErr
}

func (r *fakeWebhookRedeemer) InactivateWebhookOwner(_ context.Context, _ uuid.UUID, _ syntax.DID, _ time.Time) error {
	*r.events = append(*r.events, "inactivate")
	return r.inactivateErr
}

func (r *fakeWebhookRedeemer) RejectWebhookAttempt(_ context.Context, _ uuid.UUID, retryCode AttemptRetryCode, _ time.Time) error {
	*r.events = append(*r.events, "reject")
	r.retryCode = retryCode
	return r.rejectErr
}

type fakeWebhookMembership struct {
	current bool
	err     error
	events  *[]string
}

type fakeWebhookIdentifierLimiter struct {
	decision   RateLimitDecision
	err        error
	scope      RateLimitScope
	identifier string
	events     *[]string
}

func (l *fakeWebhookIdentifierLimiter) AllowIdentifier(_ context.Context, scope RateLimitScope, identifier []byte, _ time.Duration, _ int) (RateLimitDecision, error) {
	if l.events != nil {
		*l.events = append(*l.events, "limit")
	}
	l.scope = scope
	l.identifier = string(identifier)
	return l.decision, l.err
}

func (m *fakeWebhookMembership) IsCurrentMember(_ context.Context, _ syntax.DID) (bool, error) {
	*m.events = append(*m.events, "membership")
	return m.current, m.err
}

func (m *fakeWebhookMembership) InactivateMembership(_ context.Context, _ syntax.DID) error {
	*m.events = append(*m.events, "full-inactivate")
	return nil
}

type fakeMetaClient struct {
	username       string
	lookupErr      error
	replyErr       error
	replyRecipient string
	events         *[]string
	onLookup       func()
}

type fakeProviderError struct {
	kind       instagrammeta.ProviderErrorKind
	retryAfter time.Duration
}

func (e fakeProviderError) Error() string                         { return "synthetic provider failure" }
func (e fakeProviderError) Kind() instagrammeta.ProviderErrorKind { return e.kind }
func (e fakeProviderError) RetryAfter() time.Duration             { return e.retryAfter }

func (c *fakeMetaClient) LookupUsername(_ context.Context, _ string) (string, error) {
	*c.events = append(*c.events, "lookup")
	if c.onLookup != nil {
		c.onLookup()
	}
	return c.username, c.lookupErr
}

func (c *fakeMetaClient) SendReply(_ context.Context, recipient, _ string) error {
	*c.events = append(*c.events, "reply")
	c.replyRecipient = recipient
	return c.replyErr
}

func syntheticClaimedWebhookWork(now time.Time) WebhookWork {
	item := syntheticWebhookItem(0xb1, 0xc1, "synthetic-worker-sender", now.Add(-time.Minute))
	return WebhookWork{
		ID:                  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		MessageIDDigest:     item.MessageIDDigest,
		SenderIGSID:         item.SenderIGSID,
		OfficialAccountID:   item.OfficialAccountID,
		ChallengeDigest:     item.ChallengeDigest,
		EventAt:             item.EventAt,
		Status:              WebhookWorkProcessing,
		Attempts:            1,
		ProcessingStartedAt: now,
		LeaseToken:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		LeaseExpiresAt:      now.Add(WebhookLeaseDuration),
		CreatedAt:           now,
	}
}
