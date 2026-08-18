package push

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/notifications"
)

type DispatcherOptions struct {
	BatchSize          int
	Concurrency        int
	LeaseDuration      time.Duration
	FinalizationMargin time.Duration
	Now                func() time.Time
	Jitter             func() float64
	SendTimeout        time.Duration
	Observer           DispatcherObserver
	LifecycleFence     DeliveryLifecycleFence
}

const (
	defaultPushBatchSize          = 100
	maxPushBatchSize              = 500
	defaultPushConcurrency        = 4
	maxPushConcurrency            = 64
	defaultPushLeaseDuration      = time.Minute
	defaultPushSendTimeout        = 10 * time.Second
	defaultPushFinalizationMargin = 5 * time.Second
)

var (
	ErrInvalidDispatcherOptions  = errors.New("invalid push dispatcher options")
	ErrDeliveryLifecycleInactive = errors.New("push delivery lifecycle is inactive")
)

// DeliveryLifecycleFence holds the recipient and optional actor owner fences
// through the last eligibility check, provider call, and token-fenced local
// finalization. Implementations canonicalize DIDs and fail with
// ErrDeliveryLifecycleInactive when any required owner is no longer active.
type DeliveryLifecycleFence interface {
	WithActiveOwners(context.Context, []syntax.DID, func(context.Context) error) error
}

// DispatcherObserver receives aggregate, privacy-safe queue and delivery
// outcomes. Provider errors and device tokens are deliberately not exposed.
type DispatcherObserver interface {
	ObservePushDelivery(string, string)
	ObservePushQueue(int, time.Duration)
}

// Dispatcher polls the durable push_deliveries outbox and hands due deliveries
// to a provider-specific Sender. Database leases allow multiple dispatchers to
// work concurrently without normally sending the same row at the same time.
type Dispatcher struct {
	pool    *pgxpool.Pool
	sender  Sender
	options DispatcherOptions
}

func NewDispatcherValidated(
	pool *pgxpool.Pool,
	sender Sender,
	options DispatcherOptions,
) (*Dispatcher, error) {
	if options.BatchSize == 0 {
		options.BatchSize = defaultPushBatchSize
	}
	if options.BatchSize < 1 || options.BatchSize > maxPushBatchSize {
		return nil, fmt.Errorf("%w: PUSH_BATCH_SIZE must be between 1 and %d", ErrInvalidDispatcherOptions, maxPushBatchSize)
	}
	if options.Concurrency == 0 {
		options.Concurrency = min(defaultPushConcurrency, options.BatchSize)
	}
	if options.Concurrency < 1 || options.Concurrency > maxPushConcurrency ||
		options.Concurrency > options.BatchSize {
		return nil, fmt.Errorf("%w: PUSH_CONCURRENCY must be between 1 and min(PUSH_BATCH_SIZE, %d)", ErrInvalidDispatcherOptions, maxPushConcurrency)
	}
	if options.LeaseDuration == 0 {
		options.LeaseDuration = defaultPushLeaseDuration
	}
	if options.LeaseDuration < 0 {
		return nil, fmt.Errorf("%w: PUSH_LEASE_DURATION must be positive", ErrInvalidDispatcherOptions)
	}
	if options.SendTimeout == 0 {
		options.SendTimeout = defaultPushSendTimeout
	}
	if options.SendTimeout < 0 {
		return nil, fmt.Errorf("%w: PUSH_SEND_TIMEOUT must be positive", ErrInvalidDispatcherOptions)
	}
	if options.FinalizationMargin == 0 {
		options.FinalizationMargin = defaultPushFinalizationMargin
	}
	if options.FinalizationMargin < 0 {
		return nil, fmt.Errorf("%w: PUSH_FINALIZATION_MARGIN must be positive", ErrInvalidDispatcherOptions)
	}
	if options.LeaseDuration <= options.SendTimeout ||
		options.FinalizationMargin >= options.LeaseDuration-options.SendTimeout {
		return nil, fmt.Errorf("%w: PUSH_LEASE_DURATION must exceed PUSH_SEND_TIMEOUT + PUSH_FINALIZATION_MARGIN", ErrInvalidDispatcherOptions)
	}
	if options.LifecycleFence == nil {
		return nil, fmt.Errorf("%w: owner lifecycle fence is required", ErrInvalidDispatcherOptions)
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Jitter == nil {
		options.Jitter = func() float64 { return 0 }
	}
	return &Dispatcher{pool: pool, sender: sender, options: options}, nil
}

func attemptDeadline(
	now time.Time,
	deliveryDeadline time.Time,
	leaseExpiresAt time.Time,
	sendTimeout time.Duration,
	finalizationMargin time.Duration,
) (time.Time, bool) {
	deadline := now.Add(sendTimeout)
	if deliveryDeadline.Before(deadline) {
		deadline = deliveryDeadline
	}
	leaseDeadline := leaseExpiresAt.Add(-finalizationMargin)
	if leaseDeadline.Before(deadline) {
		deadline = leaseDeadline
	}
	return deadline, deadline.After(now)
}

type claimedDelivery struct {
	id, notificationID, subscriptionID, installationID uuid.UUID
	category                                           notifications.Category
	recipientDID                                       syntax.DID
	actorDID                                           *syntax.DID
	sourceURI                                          syntax.ATURI
	subjectURI, rootURI                                sql.NullString
	targetRole                                         ContentRole
	routingID, token, platform                         string
	leaseToken                                         string
	actorName                                          sql.NullString
	attempts                                           int
	deadline, leaseExpiresAt                           time.Time
}

// claimOne reserves one due delivery in a transaction. FOR UPDATE SKIP LOCKED
// divides available rows between concurrent dispatcher instances without
// making them wait on each other's selections.
//
// The worker label is currently retained at the call boundary, but each claim
// uses a unique UUID lease token as its real owner. That token fences stale
// workers from finalizing a row after another worker has recovered it.
func (d *Dispatcher) claimOne(ctx context.Context, _ string) ([]claimedDelivery, error) {
	now := d.options.Now().UTC()
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Only due deliveries with a currently active account subscription and
	// installation are eligible. SKIP LOCKED lets another dispatcher claim a
	// different row instead of blocking on this one.
	rows, err := tx.Query(ctx, `
		SELECT d.id,d.notification_id,d.account_subscription_id,i.id,n.category,
			n.recipient_did,n.actor_did,COALESCE(n.source_uri,''),n.subject_uri,n.root_uri,
			CASE
				WHEN target.uri IS NULL OR target.reply_root_uri IS NULL THEN 'post'
				WHEN target.reply_parent_uri = target.reply_root_uri THEN 'comment'
				ELSE 'reply'
			END,
			s.routing_id,i.fcm_token,i.platform,b.display_name,
			d.attempts,d.deadline_at
		FROM push_deliveries d
		JOIN notification_events n ON n.id=d.notification_id
		JOIN push_account_subscriptions s ON s.id=d.account_subscription_id
		JOIN push_installations i ON i.id=s.installation_id
		LEFT JOIN bluesky_profiles b ON b.did=n.actor_did
		LEFT JOIN craftsky_posts target ON target.uri = CASE WHEN n.category='quote' THEN n.quoted_uri ELSE n.subject_uri END
		WHERE d.status IN ('pending','retry')
		  AND d.next_attempt_at<=$1
		  AND n.state='active'
		  AND s.active AND i.active
		  AND NOT appview_owner_is_terminal(n.recipient_did)
		  AND NOT appview_owner_is_terminal(n.actor_did)
		  AND NOT EXISTS (
			SELECT 1 FROM actor_mutes mute
			WHERE mute.owner_did = n.recipient_did AND mute.subject_did = n.actor_did
		  )
		  AND NOT EXISTS (
			SELECT 1 FROM atproto_blocks block
			WHERE (block.blocker_did = n.recipient_did AND block.subject_did = n.actor_did)
			   OR (block.blocker_did = n.actor_did AND block.subject_did = n.recipient_did)
		  )
		ORDER BY d.next_attempt_at,d.id FOR UPDATE OF d SKIP LOCKED LIMIT 1`, now)
	if err != nil {
		return nil, err
	}
	out, err := scanClaimedDeliveries(rows)
	if err != nil {
		return nil, err
	}

	// Persist the lease before committing so no selected row can be returned to
	// the caller without its ownership and exact expiry being durable.
	for index := range out {
		out[index].leaseExpiresAt = now.Add(d.options.LeaseDuration)
		if err := tx.QueryRow(ctx, `UPDATE push_deliveries SET status='leased',attempts=$2,lease_owner=$3,lease_expires_at=$4,updated_at=$5 WHERE id=$1 RETURNING lease_expires_at`, out[index].id, out[index].attempts, out[index].leaseToken, out[index].leaseExpiresAt, now).Scan(&out[index].leaseExpiresAt); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

// recoverExpiredLeases returns work abandoned by a crashed worker to retry.
func (d *Dispatcher) recoverExpiredLeases(ctx context.Context, now time.Time) error {
	_, err := d.pool.Exec(ctx, `UPDATE push_deliveries SET status='retry',lease_owner=NULL,lease_expires_at=NULL,next_attempt_at=$1,updated_at=$1 WHERE status='leased' AND lease_expires_at<=$1`, now)
	return err
}

type claimedDeliveryRows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close()
}

func scanClaimedDeliveries(rows claimedDeliveryRows) ([]claimedDelivery, error) {
	defer rows.Close()

	type rawClaim struct {
		item         claimedDelivery
		recipientDID string
		actorDID     sql.NullString
	}
	var raw []rawClaim
	for rows.Next() {
		var claim rawClaim
		if err := rows.Scan(
			&claim.item.id, &claim.item.notificationID, &claim.item.subscriptionID,
			&claim.item.installationID, &claim.item.category, &claim.recipientDID,
			&claim.actorDID, &claim.item.sourceURI, &claim.item.subjectURI,
			&claim.item.rootURI, &claim.item.targetRole, &claim.item.routingID,
			&claim.item.token, &claim.item.platform, &claim.item.actorName,
			&claim.item.attempts, &claim.item.deadline,
		); err != nil {
			return nil, fmt.Errorf("scan push delivery claim: %w", err)
		}
		raw = append(raw, claim)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate push delivery claims: %w", err)
	}

	out := make([]claimedDelivery, 0, len(raw))
	for _, claim := range raw {
		recipientDID, actorDID, err := decodeClaimDIDs(
			claim.recipientDID,
			claim.actorDID,
		)
		if err != nil {
			return nil, err
		}
		claim.item.recipientDID = recipientDID
		claim.item.actorDID = actorDID
		// Attempts counts claims, not confirmed sends. It is incremented before
		// provider work, so a crashed or later-invalidated claim still counts.
		// A fresh token uniquely identifies this particular claim.
		claim.item.attempts++
		claim.item.leaseToken = uuid.NewString()
		out = append(out, claim.item)
	}
	return out, nil
}

func decodeClaimDIDs(
	recipient string,
	actor sql.NullString,
) (syntax.DID, *syntax.DID, error) {
	recipientDID, err := syntax.ParseDID(recipient)
	if err != nil {
		return "", nil, fmt.Errorf("decode push recipient DID: %w", err)
	}
	if !actor.Valid {
		return recipientDID, nil, nil
	}
	actorDID, err := syntax.ParseDID(actor.String)
	if err != nil {
		return "", nil, fmt.Errorf("decode push actor DID: %w", err)
	}
	return recipientDID, &actorDID, nil
}

// ProcessBatch claims and processes up to BatchSize deliveries. Provider
// outcomes are handled per delivery and normally do not become returned
// errors: retryable results are rescheduled, while terminal results are
// persisted. Returned errors generally mean queue state could not be read or
// updated reliably.
func (d *Dispatcher) ProcessBatch(ctx context.Context, worker string) (int, error) {
	// Record queue health before claiming so monitoring includes work that may
	// be leased by this or another dispatcher.
	if d.options.Observer != nil {
		pending, oldestAge, err := d.queueStats(ctx, d.options.Now().UTC())
		if err != nil {
			return 0, err
		}
		d.options.Observer.ObservePushQueue(pending, oldestAge)
	}
	now := d.options.Now().UTC()
	if err := d.recoverExpiredLeases(ctx, now); err != nil {
		return 0, err
	}
	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()

	var nextTicket atomic.Int64
	var processed atomic.Int64
	var workers sync.WaitGroup
	firstError := make(chan error, 1)
	for workerIndex := 0; workerIndex < d.options.Concurrency; workerIndex++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				if workerCtx.Err() != nil {
					return
				}
				ticket := int(nextTicket.Add(1))
				if ticket > d.options.BatchSize {
					return
				}
				items, err := d.claimOne(workerCtx, worker)
				if err != nil {
					select {
					case firstError <- err:
						cancelWorkers()
					default:
					}
					return
				}
				if len(items) == 0 {
					return
				}
				processed.Add(1)
				if err := d.processClaim(workerCtx, items[0]); err != nil {
					select {
					case firstError <- err:
						cancelWorkers()
					default:
					}
					return
				}
			}
		}()
	}
	workers.Wait()
	select {
	case err := <-firstError:
		return int(processed.Load()), err
	default:
		return int(processed.Load()), nil
	}
}

// processClaim owns exactly one active worker slot. No delivery is leased
// before a worker enters this capability.
func (d *Dispatcher) processClaim(ctx context.Context, item claimedDelivery) error {
	owners := []syntax.DID{item.recipientDID}
	if item.actorDID != nil {
		owners = append(owners, *item.actorDID)
	}
	err := d.options.LifecycleFence.WithActiveOwners(ctx, owners, func(fenceCtx context.Context) error {
		return d.processClaimFenced(fenceCtx, item)
	})
	if errors.Is(err, ErrDeliveryLifecycleInactive) {
		return d.cancelLifecycleInvalid(ctx, item, d.options.Now().UTC())
	}
	return err
}

func (d *Dispatcher) processClaimFenced(ctx context.Context, item claimedDelivery) error {
	now := d.options.Now().UTC()
	var actorDID syntax.DID
	if item.actorDID != nil {
		actorDID = *item.actorDID
	}

	// State can change after a batch is claimed: a notification may be
	// retracted, a device may be removed, its token may rotate, or this lease
	// may expire and be recovered. Never send unless this exact claim still
	// owns the current active delivery.
	owned, err := d.ownsCurrentDelivery(ctx, item, now)
	if err != nil {
		return err
	}
	if !owned {
		return nil
	}
	// TTL is always the time remaining until the original absolute delivery
	// deadline. A retry does not start a fresh delivery window.
	ttl, ok := ProviderTTL(now, item.deadline)
	if !ok {
		if _, err := d.pool.Exec(ctx, `UPDATE push_deliveries d SET status='expired',lease_owner=NULL,lease_expires_at=NULL,updated_at=$6 WHERE d.id=$1 AND d.status='leased' AND d.lease_owner=$2 AND d.account_subscription_id=$3 AND d.lease_expires_at=$7 AND d.lease_expires_at>$6 AND EXISTS(SELECT 1 FROM push_account_subscriptions s JOIN push_installations i ON i.id=s.installation_id WHERE s.id=$3 AND s.active AND i.id=$4 AND i.active AND i.fcm_token=$5)`, item.id, item.leaseToken, item.subscriptionID, item.installationID, item.token, now, item.leaseExpiresAt); err != nil {
			return err
		}
		return nil
	}

	// The provider call must finish before both the logical delivery deadline
	// and the exact persisted lease expiry, leaving the configured database
	// finalization margin untouched.
	providerDeadline, usefulWindow := attemptDeadline(
		now,
		item.deadline,
		item.leaseExpiresAt,
		d.options.SendTimeout,
		d.options.FinalizationMargin,
	)
	if !usefulWindow {
		return d.releaseInsufficientWindow(ctx, item, now)
	}
	sendCtx, cancel := context.WithDeadline(ctx, providerDeadline)
	result, sendErr := d.sender.Send(sendCtx, SendRequest{
		Token:                 item.token,
		Category:              item.category,
		AccountSubscriptionID: item.routingID,
		RoutingFacts: RoutingFacts{
			ActorDID:       actorDID,
			SourceURI:      item.sourceURI,
			SubjectURI:     syntax.ATURI(item.subjectURI.String),
			RootURI:        syntax.ATURI(item.rootURI.String),
			TargetRole:     item.targetRole,
			NotificationID: item.notificationID.String(),
		},
		ActorDisplayName: item.actorName.String,
		Platform:         item.platform,
		Semantics:        DeliveryUniqueEvent,
		TTL:              ttl,
	})
	cancel()
	now = d.options.Now().UTC()

	// Every state-changing query below repeats the lease, subscription,
	// installation, and token checks. These are compare-and-set guards: if
	// ownership changed while the provider call was in flight, a stale result
	// cannot overwrite the newer state.
	switch result.Class {
	case ResultSuccess:
		_, err = d.pool.Exec(ctx, `UPDATE push_deliveries d SET status='succeeded',sent_at=$6,provider_result_class='success',lease_owner=NULL,lease_expires_at=NULL,updated_at=$6 WHERE d.id=$1 AND d.status='leased' AND d.lease_owner=$2 AND d.account_subscription_id=$3 AND d.lease_expires_at=$7 AND d.lease_expires_at>$6 AND EXISTS(SELECT 1 FROM push_account_subscriptions s JOIN push_installations i ON i.id=s.installation_id WHERE s.id=$3 AND s.active AND i.id=$4 AND i.active AND i.fcm_token=$5)`, item.id, item.leaseToken, item.subscriptionID, item.installationID, item.token, now, item.leaseExpiresAt)
	case ResultRetryable:
		// Retry this delivery independently with bounded exponential backoff.
		// If no retry fits before its deadline, expire it instead.
		next, retry := NextRetry(now, item.deadline, item.attempts, d.options.Jitter())
		if !retry {
			_, err = d.pool.Exec(ctx, `UPDATE push_deliveries d SET status='expired',provider_result_class='retryable',lease_owner=NULL,lease_expires_at=NULL,updated_at=$6 WHERE d.id=$1 AND d.status='leased' AND d.lease_owner=$2 AND d.account_subscription_id=$3 AND d.lease_expires_at=$7 AND d.lease_expires_at>$6 AND EXISTS(SELECT 1 FROM push_account_subscriptions s JOIN push_installations i ON i.id=s.installation_id WHERE s.id=$3 AND s.active AND i.id=$4 AND i.active AND i.fcm_token=$5)`, item.id, item.leaseToken, item.subscriptionID, item.installationID, item.token, now, item.leaseExpiresAt)
		} else {
			_, err = d.pool.Exec(ctx, `UPDATE push_deliveries d SET status='retry',next_attempt_at=$6,provider_result_class='retryable',lease_owner=NULL,lease_expires_at=NULL,updated_at=$7 WHERE d.id=$1 AND d.status='leased' AND d.lease_owner=$2 AND d.account_subscription_id=$3 AND d.lease_expires_at=$8 AND d.lease_expires_at>$7 AND EXISTS(SELECT 1 FROM push_account_subscriptions s JOIN push_installations i ON i.id=s.installation_id WHERE s.id=$3 AND s.active AND i.id=$4 AND i.active AND i.fcm_token=$5)`, item.id, item.leaseToken, item.subscriptionID, item.installationID, item.token, next, now, item.leaseExpiresAt)
		}
	case ResultInvalidToken:
		// Invalidating a token affects the installation and all account
		// subscriptions routed through it, so perform that cleanup atomically.
		err = d.invalidate(ctx, item, now)
	default:
		_, err = d.pool.Exec(ctx, `UPDATE push_deliveries d SET status='permanent_failure',provider_result_class='permanent',lease_owner=NULL,lease_expires_at=NULL,updated_at=$6 WHERE d.id=$1 AND d.status='leased' AND d.lease_owner=$2 AND d.account_subscription_id=$3 AND d.lease_expires_at=$7 AND d.lease_expires_at>$6 AND EXISTS(SELECT 1 FROM push_account_subscriptions s JOIN push_installations i ON i.id=s.installation_id WHERE s.id=$3 AND s.active AND i.id=$4 AND i.active AND i.fcm_token=$5)`, item.id, item.leaseToken, item.subscriptionID, item.installationID, item.token, now, item.leaseExpiresAt)
	}
	if err != nil {
		return err
	}
	if d.options.Observer != nil {
		d.options.Observer.ObservePushDelivery(item.platform, string(result.Class))
	}
	// Sender implementations translate raw provider errors into ResultClass.
	// State transitions and telemetry use that safe classification rather
	// than persisting or exposing provider error details.
	_ = sendErr
	return nil
}

func (d *Dispatcher) cancelLifecycleInvalid(
	ctx context.Context,
	item claimedDelivery,
	now time.Time,
) error {
	_, err := d.pool.Exec(ctx, `
		UPDATE push_deliveries
		SET status='cancelled',lease_owner=NULL,lease_expires_at=NULL,updated_at=$4
		WHERE id=$1 AND status='leased' AND lease_owner=$2
		  AND lease_expires_at=$3 AND lease_expires_at>$4
	`, item.id, item.leaseToken, item.leaseExpiresAt, now)
	return err
}

// queueStats reports all outstanding work, including currently leased rows,
// and the age of the oldest row for backlog monitoring.
func (d *Dispatcher) queueStats(ctx context.Context, now time.Time) (int, time.Duration, error) {
	var pending int
	var oldestSeconds float64
	err := d.pool.QueryRow(ctx, `
		SELECT count(*)::int,
		       COALESCE(EXTRACT(EPOCH FROM ($1::timestamptz - min(created_at))), 0)::float8
		FROM push_deliveries
		WHERE status IN ('pending','retry','leased')
	`, now).Scan(&pending, &oldestSeconds)
	if oldestSeconds < 0 {
		oldestSeconds = 0
	}
	return pending, time.Duration(oldestSeconds * float64(time.Second)), err
}

// ownsCurrentDelivery verifies that this exact lease is still live and that
// its routing information has not been cancelled or rotated since claim time.
func (d *Dispatcher) ownsCurrentDelivery(ctx context.Context, item claimedDelivery, now time.Time) (bool, error) {
	var owned bool
	err := d.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM push_deliveries d
			JOIN notification_events n ON n.id=d.notification_id
			JOIN push_account_subscriptions s ON s.id=d.account_subscription_id
			JOIN push_installations i ON i.id=s.installation_id
			WHERE d.id=$1 AND d.status='leased' AND d.lease_owner=$2
			  AND d.lease_expires_at=$7 AND d.lease_expires_at>$6
			  AND n.state='active'
			  AND NOT appview_owner_is_terminal(n.recipient_did)
			  AND NOT appview_owner_is_terminal(n.actor_did)
			  AND s.id=$3 AND s.active AND s.account_did=n.recipient_did
			  AND i.id=$4 AND i.active AND i.fcm_token=$5
			  AND NOT EXISTS (
				SELECT 1 FROM actor_mutes mute
				WHERE mute.owner_did = n.recipient_did AND mute.subject_did = n.actor_did
			  )
			  AND NOT EXISTS (
				SELECT 1 FROM atproto_blocks block
				WHERE (block.blocker_did = n.recipient_did AND block.subject_did = n.actor_did)
				   OR (block.blocker_did = n.actor_did AND block.subject_did = n.recipient_did)
			  )
		)`, item.id, item.leaseToken, item.subscriptionID, item.installationID, item.token, now, item.leaseExpiresAt).Scan(&owned)
	return owned, err
}

func (d *Dispatcher) releaseInsufficientWindow(
	ctx context.Context,
	item claimedDelivery,
	now time.Time,
) error {
	_, err := d.pool.Exec(ctx, `
		UPDATE push_deliveries
		SET status='retry', next_attempt_at=$4, lease_owner=NULL,
			lease_expires_at=NULL, updated_at=$4
		WHERE id=$1 AND status='leased' AND lease_owner=$2
		  AND lease_expires_at=$3 AND lease_expires_at>$4
	`, item.id, item.leaseToken, item.leaseExpiresAt, now)
	return err
}

// invalidate deactivates an installation after the provider rejects its token.
// It locks and rechecks the current token before changing anything so an
// in-flight result for an old token cannot deactivate a newly rotated token.
func (d *Dispatcher) invalidate(ctx context.Context, item claimedDelivery, now time.Time) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var owned bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM push_deliveries d
			JOIN push_account_subscriptions s ON s.id=d.account_subscription_id
			JOIN push_installations i ON i.id=s.installation_id
			WHERE d.id=$1 AND d.status='leased' AND d.lease_owner=$2
			  AND d.lease_expires_at=$7 AND d.lease_expires_at>$6
			  AND s.id=$3 AND s.active
			  AND i.id=$4 AND i.active AND i.fcm_token=$5
			FOR UPDATE OF d, i
		)`, item.id, item.leaseToken, item.subscriptionID, item.installationID, item.token, now, item.leaseExpiresAt).Scan(&owned); err != nil {
		return err
	}
	if !owned {
		// Another operation changed the lease, installation, subscription, or
		// token while the provider call was in flight. The stale result is ignored.
		return tx.Commit(ctx)
	}

	// One physical installation may route notifications for several signed-in
	// accounts. An invalid FCM token therefore deactivates every subscription
	// attached to that installation and cancels their outstanding deliveries.
	if _, err := tx.Exec(ctx, `UPDATE push_installations SET active=false,deactivated_at=$2,updated_at=$2 WHERE id=$1 AND active AND fcm_token=$3`, item.installationID, now, item.token); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE push_account_subscriptions SET active=false,deactivated_at=$2,updated_at=$2 WHERE installation_id=$1`, item.installationID, now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE push_deliveries SET status=CASE WHEN id=$2 THEN 'permanent_failure' ELSE 'cancelled' END,provider_result_class=CASE WHEN id=$2 THEN 'invalidToken' ELSE provider_result_class END,lease_owner=NULL,lease_expires_at=NULL,updated_at=$3 WHERE account_subscription_id IN(SELECT id FROM push_account_subscriptions WHERE installation_id=$1) AND status IN('pending','retry','leased')`, item.installationID, item.id, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (d *Dispatcher) Run(
	ctx context.Context,
	poll time.Duration,
	worker string,
) error {
	// Use a one-second poll interval when the supplied value is invalid.
	if poll <= 0 {
		poll = time.Second
	}

	// Create the ticker used for normal polling after successful batch runs.
	// The first batch runs immediately, before waiting for this ticker.
	ticker := time.NewTicker(poll)
	defer ticker.Stop()

	// Consecutive ProcessBatch errors use exponential backoff, starting
	// with the normal poll interval and capped at 30 seconds.
	errorDelay := poll
	const maxErrorDelay = 30 * time.Second

	// Continue processing until the context is cancelled or reaches
	// its deadline.
	for {
		// ProcessBatch returning nil includes the case where there was
		// simply no work available.
		// Individual Firebase outcomes are handled per delivery, so a returned
		// error more often means queue/storage state could not be read or updated.
		if _, err := d.ProcessBatch(ctx, worker); err != nil {
			// Check whether the dispatcher context has been cancelled.
			//
			// This does not inspect whether err itself is a context error.
			// It checks the current state of ctx.
			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Treat the ProcessBatch error as transient and wait before
			// trying another batch.
			timer := time.NewTimer(errorDelay)

			select {
			case <-ctx.Done():
				// The timer has not necessarily fired, so release it.
				timer.Stop()

				// Because ctx.Done() has fired, ctx.Err() will be
				// context.Canceled or context.DeadlineExceeded.
				return ctx.Err()

			case <-timer.C:
				// The retry delay has elapsed.
			}

			// Increase the delay for the next consecutive error.
			errorDelay *= 2
			if errorDelay > maxErrorDelay {
				errorDelay = maxErrorDelay
			}

			// Retry ProcessBatch without waiting for the normal ticker.
			continue
		}

		// A successful ProcessBatch resets the error backoff, even when
		// the batch contained no deliveries.
		errorDelay = poll

		select {
		case <-ctx.Done():
			// ctx.Err() is guaranteed to be non-nil after Done closes.
			return ctx.Err()

		case <-ticker.C:
			// The next normal polling point has arrived.
		}
	}
}
