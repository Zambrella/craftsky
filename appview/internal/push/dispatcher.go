package push

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/notifications"
)

type DispatcherOptions struct {
	BatchSize     int
	LeaseDuration time.Duration
	Now           func() time.Time
	Jitter        func() float64
	SendTimeout   time.Duration
	Observer      DispatcherObserver
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

func NewDispatcher(pool *pgxpool.Pool, sender Sender, options DispatcherOptions) *Dispatcher {
	if options.BatchSize <= 0 {
		options.BatchSize = 100
	}
	if options.BatchSize > 500 {
		options.BatchSize = 500
	}
	if options.LeaseDuration <= 0 {
		options.LeaseDuration = time.Minute
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Jitter == nil {
		options.Jitter = func() float64 { return 0 }
	}
	if options.SendTimeout <= 0 {
		options.SendTimeout = 10 * time.Second
	}
	return &Dispatcher{pool: pool, sender: sender, options: options}
}

type claimedDelivery struct {
	id, notificationID, subscriptionID, installationID uuid.UUID
	category                                           notifications.Category
	routingID, token, platform                         string
	leaseToken                                         string
	actorName                                          sql.NullString
	attempts                                           int
	deadline                                           time.Time
}

// claim reserves the next due deliveries in one transaction. Expired leases
// are first made retryable, then FOR UPDATE SKIP LOCKED divides available rows
// between concurrent dispatcher instances without making them wait on each
// other's selections.
//
// The worker label is currently retained at the call boundary, but each claim
// uses a unique UUID lease token as its real owner. That token fences stale
// workers from finalizing a row after another worker has recovered it.
func (d *Dispatcher) claim(ctx context.Context, _ string) ([]claimedDelivery, error) {
	now := d.options.Now().UTC()
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// A worker may crash or lose its database connection after claiming work.
	// Once its lease expires, return those rows to retry immediately.
	if _, err := tx.Exec(ctx, `UPDATE push_deliveries SET status='retry',lease_owner=NULL,lease_expires_at=NULL,next_attempt_at=$1,updated_at=$1 WHERE status='leased' AND lease_expires_at<=$1`, now); err != nil {
		return nil, err
	}

	// Only due deliveries with a currently active account subscription and
	// installation are eligible. SKIP LOCKED lets another dispatcher claim a
	// different batch instead of blocking on these rows.
	rows, err := tx.Query(ctx, `
		SELECT d.id,d.notification_id,d.account_subscription_id,i.id,n.category,s.routing_id,i.fcm_token,i.platform,b.display_name,d.attempts,d.deadline_at
		FROM push_deliveries d JOIN notification_events n ON n.id=d.notification_id JOIN push_account_subscriptions s ON s.id=d.account_subscription_id JOIN push_installations i ON i.id=s.installation_id LEFT JOIN bluesky_profiles b ON b.did=n.actor_did
		WHERE d.status IN ('pending','retry') AND d.next_attempt_at<=$1 AND s.active AND i.active ORDER BY d.next_attempt_at,d.id FOR UPDATE OF d SKIP LOCKED LIMIT $2`, now, d.options.BatchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []claimedDelivery
	for rows.Next() {
		var item claimedDelivery
		if err := rows.Scan(&item.id, &item.notificationID, &item.subscriptionID, &item.installationID, &item.category, &item.routingID, &item.token, &item.platform, &item.actorName, &item.attempts, &item.deadline); err != nil {
			return nil, err
		}
		// Attempts counts claims, not confirmed sends. It is incremented before
		// provider work, so a crashed or later-invalidated claim still counts.
		// A fresh token uniquely identifies this particular claim.
		item.attempts++
		item.leaseToken = uuid.NewString()
		out = append(out, item)
	}

	// Persist every lease before committing so no selected row can be returned
	// to the caller without its ownership being durable.
	for _, item := range out {
		if _, err := tx.Exec(ctx, `UPDATE push_deliveries SET status='leased',attempts=$2,lease_owner=$3,lease_expires_at=$4,updated_at=$5 WHERE id=$1`, item.id, item.attempts, item.leaseToken, now.Add(d.options.LeaseDuration), now); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
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
	items, err := d.claim(ctx, worker)
	if err != nil {
		return 0, err
	}
	for _, item := range items {
		now := d.options.Now().UTC()

		// State can change after a batch is claimed: a notification may be
		// retracted, a device may be removed, its token may rotate, or this lease
		// may expire and be recovered. Never send unless this exact claim still
		// owns the current active delivery.
		owned, err := d.ownsCurrentDelivery(ctx, item, now)
		if err != nil {
			return 0, err
		}
		if !owned {
			continue
		}

		// TTL is always the time remaining until the original absolute delivery
		// deadline. A retry does not start a fresh delivery window.
		ttl, ok := ProviderTTL(now, item.deadline)
		if !ok {
			if _, err := d.pool.Exec(ctx, `UPDATE push_deliveries d SET status='expired',lease_owner=NULL,lease_expires_at=NULL,updated_at=$6 WHERE d.id=$1 AND d.status='leased' AND d.lease_owner=$2 AND d.account_subscription_id=$3 AND d.lease_expires_at>$6 AND EXISTS(SELECT 1 FROM push_account_subscriptions s JOIN push_installations i ON i.id=s.installation_id WHERE s.id=$3 AND s.active AND i.id=$4 AND i.active AND i.fcm_token=$5)`, item.id, item.leaseToken, item.subscriptionID, item.installationID, item.token, now); err != nil {
				return 0, err
			}
			continue
		}

		// Do not let a provider call run past the delivery deadline, even when the
		// configured send timeout is longer than the remaining TTL.
		attemptTimeout := d.options.SendTimeout
		if ttl < attemptTimeout {
			attemptTimeout = ttl
		}
		sendCtx, cancel := context.WithTimeout(ctx, attemptTimeout)
		result, sendErr := d.sender.Send(sendCtx, SendRequest{Token: item.token, NotificationID: item.notificationID.String(), Category: item.category, AccountSubscriptionID: item.routingID, ActorDisplayName: item.actorName.String, Platform: item.platform, TTL: ttl})
		cancel()
		now = d.options.Now().UTC()

		// Every state-changing query below repeats the lease, subscription,
		// installation, and token checks. These are compare-and-set guards: if
		// ownership changed while the provider call was in flight, a stale result
		// cannot overwrite the newer state.
		switch result.Class {
		case ResultSuccess:
			_, err = d.pool.Exec(ctx, `UPDATE push_deliveries d SET status='succeeded',sent_at=$6,provider_result_class='success',lease_owner=NULL,lease_expires_at=NULL,updated_at=$6 WHERE d.id=$1 AND d.status='leased' AND d.lease_owner=$2 AND d.account_subscription_id=$3 AND d.lease_expires_at>$6 AND EXISTS(SELECT 1 FROM push_account_subscriptions s JOIN push_installations i ON i.id=s.installation_id WHERE s.id=$3 AND s.active AND i.id=$4 AND i.active AND i.fcm_token=$5)`, item.id, item.leaseToken, item.subscriptionID, item.installationID, item.token, now)
		case ResultRetryable:
			// Retry this delivery independently with bounded exponential backoff.
			// If no retry fits before its deadline, expire it instead.
			next, retry := NextRetry(now, item.deadline, item.attempts, d.options.Jitter())
			if !retry {
				_, err = d.pool.Exec(ctx, `UPDATE push_deliveries d SET status='expired',provider_result_class='retryable',lease_owner=NULL,lease_expires_at=NULL,updated_at=$6 WHERE d.id=$1 AND d.status='leased' AND d.lease_owner=$2 AND d.account_subscription_id=$3 AND d.lease_expires_at>$6 AND EXISTS(SELECT 1 FROM push_account_subscriptions s JOIN push_installations i ON i.id=s.installation_id WHERE s.id=$3 AND s.active AND i.id=$4 AND i.active AND i.fcm_token=$5)`, item.id, item.leaseToken, item.subscriptionID, item.installationID, item.token, now)
			} else {
				_, err = d.pool.Exec(ctx, `UPDATE push_deliveries d SET status='retry',next_attempt_at=$6,provider_result_class='retryable',lease_owner=NULL,lease_expires_at=NULL,updated_at=$7 WHERE d.id=$1 AND d.status='leased' AND d.lease_owner=$2 AND d.account_subscription_id=$3 AND d.lease_expires_at>$7 AND EXISTS(SELECT 1 FROM push_account_subscriptions s JOIN push_installations i ON i.id=s.installation_id WHERE s.id=$3 AND s.active AND i.id=$4 AND i.active AND i.fcm_token=$5)`, item.id, item.leaseToken, item.subscriptionID, item.installationID, item.token, next, now)
			}
		case ResultInvalidToken:
			// Invalidating a token affects the installation and all account
			// subscriptions routed through it, so perform that cleanup atomically.
			err = d.invalidate(ctx, item, now)
		default:
			_, err = d.pool.Exec(ctx, `UPDATE push_deliveries d SET status='permanent_failure',provider_result_class='permanent',lease_owner=NULL,lease_expires_at=NULL,updated_at=$6 WHERE d.id=$1 AND d.status='leased' AND d.lease_owner=$2 AND d.account_subscription_id=$3 AND d.lease_expires_at>$6 AND EXISTS(SELECT 1 FROM push_account_subscriptions s JOIN push_installations i ON i.id=s.installation_id WHERE s.id=$3 AND s.active AND i.id=$4 AND i.active AND i.fcm_token=$5)`, item.id, item.leaseToken, item.subscriptionID, item.installationID, item.token, now)
		}
		if err != nil {
			return 0, err
		}
		if d.options.Observer != nil {
			d.options.Observer.ObservePushDelivery(item.platform, string(result.Class))
		}
		// Sender implementations translate raw provider errors into ResultClass.
		// State transitions and telemetry use that safe classification rather
		// than persisting or exposing provider error details.
		_ = sendErr
	}
	return len(items), nil
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
			JOIN push_account_subscriptions s ON s.id=d.account_subscription_id
			JOIN push_installations i ON i.id=s.installation_id
			WHERE d.id=$1 AND d.status='leased' AND d.lease_owner=$2 AND d.lease_expires_at>$6
			  AND s.id=$3 AND s.active
			  AND i.id=$4 AND i.active AND i.fcm_token=$5
		)`, item.id, item.leaseToken, item.subscriptionID, item.installationID, item.token, now).Scan(&owned)
	return owned, err
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
			WHERE d.id=$1 AND d.status='leased' AND d.lease_owner=$2 AND d.lease_expires_at>$6
			  AND s.id=$3 AND s.active
			  AND i.id=$4 AND i.active AND i.fcm_token=$5
			FOR UPDATE OF d, i
		)`, item.id, item.leaseToken, item.subscriptionID, item.installationID, item.token, now).Scan(&owned); err != nil {
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
