package subscriptions

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/ownerlifecycle"
)

var ErrWebCheckoutUnavailable = errors.New("web checkout unavailable")
var ErrSubscriptionReconciliationRequired = errors.New("subscription reconciliation required")
var ErrSubscriptionAlreadyExists = errors.New("subscription already exists")
var ErrWebCheckoutPending = errors.New("another subscription checkout is pending")
var ErrWebManagementUnavailable = errors.New("web subscription management unavailable")

// This identifies the only failed dispatch with a proven absent remote effect.
var errCheckoutStandingDenied = fmt.Errorf("checkout standing denied: %w", ErrWebCheckoutUnavailable)

type CheckoutRequest struct {
	OperationID uuid.UUID
	CustomerID  uuid.UUID
	AccountID   string
	PriceID     string
	SuccessURL  string
	CancelURL   string
}
type CheckoutSession struct {
	ID, URL, Status, CustomerID, AppUserID, SubscriptionAppUserID                 string
	LiveMode, EnvironmentKnown, Managed, StripeTaxLiability, SubscriptionCanceled bool
	ExpiresAt                                                                     time.Time
}

func (s CheckoutSession) String() string             { return "CheckoutSession{private:[REDACTED]}" }
func (s CheckoutSession) GoString() string           { return s.String() }
func (s CheckoutSession) Format(f fmt.State, _ rune) { _, _ = io.WriteString(f, s.String()) }

type CheckoutProvider interface {
	CreateCheckout(context.Context, CheckoutRequest) (CheckoutSession, error)
	GetCheckout(context.Context, string) (CheckoutSession, error)
	CreateManagement(context.Context, string, string) (string, error)
}
type WebCheckoutConfig struct{ AccountID, PlusPriceID, BusinessPriceID, SuccessURL, CancelURL, BillingReturnURL string }

func (c WebCheckoutConfig) String() string             { return "WebCheckoutConfig{private:[REDACTED]}" }
func (c WebCheckoutConfig) GoString() string           { return c.String() }
func (c WebCheckoutConfig) Format(f fmt.State, _ rune) { _, _ = io.WriteString(f, c.String()) }

var stripePriceID = regexp.MustCompile(`^price_[A-Za-z0-9]+$`)
var stripeAccountID = regexp.MustCompile(`^acct_[A-Za-z0-9]+$`)
var stripeCheckoutPath = regexp.MustCompile(`^/c/pay/(cs_(?:test_|live_)[A-Za-z0-9]+)$`)
var stripeManagementPath = regexp.MustCompile(`^/p/session/[A-Za-z0-9_-]+$`)

// ValidateWebCheckoutConfig permits only complete configurations. Empty means disabled.
func ValidateWebCheckoutConfig(c WebCheckoutConfig) error {
	if !stripeAccountID.MatchString(c.AccountID) || !stripePriceID.MatchString(c.PlusPriceID) || !stripePriceID.MatchString(c.BusinessPriceID) || c.PlusPriceID == c.BusinessPriceID {
		return errors.New("Stripe checkout requires an account and two distinct price identifiers")
	}
	for _, raw := range []string{c.SuccessURL, c.CancelURL, c.BillingReturnURL} {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || u.Opaque != "" || u.RawPath != "" {
			return errors.New("Stripe return URLs must be fixed HTTPS URLs without credentials, query or fragment")
		}
	}
	return nil
}
func (s *Store) SetWebCheckoutProvider(provider CheckoutProvider, config WebCheckoutConfig) error {
	if provider == nil {
		return errors.New("Stripe checkout provider is required")
	}
	if err := ValidateWebCheckoutConfig(config); err != nil {
		return err
	}
	s.checkoutProvider = provider
	s.checkoutConfig = config
	return nil
}
func ValidStripeManagementURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host == "billing.stripe.com" && u.User == nil && u.RawQuery == "" && u.RawPath == "" && u.Opaque == "" && stripeManagementPath.MatchString(u.Path)
}
func validCheckoutSession(session CheckoutSession, customer uuid.UUID, environment string) bool {
	if session.AppUserID != customer.String() || !session.EnvironmentKnown || session.LiveMode != (environment == "production") || !session.Managed || !session.StripeTaxLiability {
		return false
	}
	if session.Status != "open" && session.Status != "complete" && session.Status != "expired" {
		return false
	}
	if (session.Status == "complete" || session.SubscriptionAppUserID != "") && session.SubscriptionAppUserID != customer.String() {
		return false
	}
	return stripeCheckoutPath.MatchString("/c/pay/"+session.ID) && strings.HasPrefix(session.ID, "cs_test_") == !session.LiveMode
}
func checkoutURL(session CheckoutSession) (string, error) {
	u, err := url.Parse(session.URL)
	if err != nil || u.Scheme != "https" || u.Host != "checkout.stripe.com" || u.User != nil || u.RawQuery != "" || u.RawPath != "" || u.Opaque != "" {
		return "", ErrWebCheckoutUnavailable
	}
	match := stripeCheckoutPath.FindStringSubmatch(u.Path)
	if len(match) != 2 || match[1] != session.ID {
		return "", ErrWebCheckoutUnavailable
	}
	return u.String(), nil
}

type checkoutOperation struct {
	CheckoutRequest
	Tier                             Tier
	SessionID                        *string
	CreatedAt                        time.Time
	TerminalReconciliationGeneration *int64
}

func (s *Store) readCheckoutOperation(ctx context.Context, conn *pgxpool.Conn, accountID uuid.UUID) (checkoutOperation, error) {
	var op checkoutOperation
	err := conn.QueryRow(ctx, `SELECT operation_id,tier,stripe_account_id,price_id,success_url,cancel_url,session_id,created_at,terminal_reconciliation_generation FROM billing_web_checkouts WHERE billing_account_id=$1 AND environment=$2`, accountID, s.accessEnvironment).Scan(&op.OperationID, &op.Tier, &op.AccountID, &op.PriceID, &op.SuccessURL, &op.CancelURL, &op.SessionID, &op.CreatedAt, &op.TerminalReconciliationGeneration)
	return op, err
}
func (s *Store) deleteCheckoutOperation(ctx context.Context, conn *pgxpool.Conn, accountID uuid.UUID) error {
	_, err := conn.Exec(ctx, `DELETE FROM billing_web_checkouts WHERE billing_account_id=$1 AND environment=$2`, accountID, s.accessEnvironment)
	return err
}

// withCheckoutFence follows the canonical owner-effect lock order, then holds a
// per-owner checkout lock through provider I/O. The durable operation commits
// before dispatch and survives cancellation; a separate bounded transaction
// keeps moderation standing stable only while the provider effect runs.
func (s *Store) withCheckoutFence(ctx context.Context, did syntax.DID, allowDeletionPending bool, callback func(*pgxpool.Conn) error) (result error) {
	if s == nil || s.pool == nil || did == "" {
		return ErrWebCheckoutUnavailable
	}
	acquireCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := s.pool.Acquire(acquireCtx)
	if err != nil {
		return fmt.Errorf("acquire checkout connection: %w", err)
	}
	ownerKey, err := ownerlifecycle.FenceKey(did)
	if err != nil {
		conn.Release()
		return err
	}
	digest := sha256.Sum256([]byte("social.craftsky.checkout.v1\x00" + did.String()))
	key := int64(binary.BigEndian.Uint64(digest[:8]))
	ownerLocked, checkoutLocked := false, false
	defer func() {
		if conn == nil {
			return
		}
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		unlock := func(query string, key int64) bool {
			var ok bool
			err := conn.QueryRow(cleanup, query, key).Scan(&ok)
			return err == nil && ok
		}
		clean := true
		if checkoutLocked {
			clean = unlock(`SELECT pg_advisory_unlock($1)`, key)
		}
		if ownerLocked {
			clean = unlock(`SELECT pg_advisory_unlock_shared($1)`, ownerKey) && clean
		}
		if !clean {
			discardSubscriptionFence(conn)
			result = errors.Join(result, errors.New("release checkout fence failed"))
		} else {
			conn.Release()
		}
	}()
	if _, err := conn.Exec(acquireCtx, `SELECT pg_advisory_lock_shared($1)`, ownerKey); err != nil {
		discardSubscriptionFence(conn)
		conn = nil
		return fmt.Errorf("acquire checkout owner fence: %w", err)
	}
	ownerLocked = true
	if _, err := conn.Exec(acquireCtx, `SELECT pg_advisory_lock($1)`, key); err != nil {
		discardSubscriptionFence(conn)
		conn = nil
		return fmt.Errorf("acquire checkout operation fence: %w", err)
	}
	checkoutLocked = true
	var active bool
	if err := conn.QueryRow(ctx, `SELECT appview_owner_is_active($1)`, did).Scan(&active); err != nil {
		return fmt.Errorf("read checkout owner: %w", err)
	}
	if !active && allowDeletionPending {
		if err := conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM owner_lifecycles WHERE owner_did=$1 AND state='deletion_pending')`, did).Scan(&active); err != nil {
			return err
		}
	}
	if !active {
		return ownerlifecycle.ErrOwnerNotActive
	}
	if expected, ok := ownerlifecycle.ExpectedGeneration(ctx); ok && !allowDeletionPending {
		var current bool
		if err := conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM owner_lifecycles WHERE owner_did=$1 AND generation=$2 AND state='active')`, did, expected).Scan(&current); err != nil {
			return err
		}
		if !current {
			return ownerlifecycle.ErrGenerationChanged
		}
	}
	return callback(conn)
}
func checkoutPreflight(state BillingState, now time.Time) error {
	if state.ReconciliationStale || state.ReconciledAt == nil || now.Sub(*state.ReconciledAt) > 5*time.Minute || state.ReconciledAt.After(now.Add(time.Minute)) {
		return ErrSubscriptionReconciliationRequired
	}
	for _, sub := range state.Subscriptions {
		if subscriptionBlocksCheckout(sub, now) {
			return ErrSubscriptionAlreadyExists
		}
	}
	for _, license := range state.Licenses {
		if license.Anomaly != "none" && license.Anomaly != "" {
			return ErrSubscriptionAlreadyExists
		}
	}
	return nil
}

// withCheckoutStanding linearizes remote effects with moderation transitions.
// Only the moderation row is held for at most 30 seconds. The durable checkout
// operation is already committed, and no pooled state read occurs in callback.
func (s *Store) withCheckoutStanding(ctx context.Context, conn *pgxpool.Conn, did syntax.DID, callback func(context.Context) error) error {
	effectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	tx, err := conn.Begin(effectCtx)
	if err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	if _, err := tx.Exec(effectCtx, `INSERT INTO moderation_account_standings(owner_did) VALUES($1) ON CONFLICT DO NOTHING`, did); err != nil {
		return err
	}
	var suspended bool
	if err := tx.QueryRow(effectCtx, `SELECT effective_suspended FROM moderation_account_standings WHERE owner_did=$1 FOR SHARE`, did).Scan(&suspended); err != nil {
		return err
	}
	if suspended {
		return errCheckoutStandingDenied
	}
	if err := callback(effectCtx); err != nil {
		return err
	}
	return tx.Commit(effectCtx)
}
func (s *Store) recoverCheckout(ctx context.Context, conn *pgxpool.Conn, did syntax.DID, state BillingState, op checkoutOperation, now time.Time) (session CheckoutSession, resolved bool, err error) {
	err = s.withCheckoutStanding(ctx, conn, did, func(effectCtx context.Context) error {
		session, resolved, err = s.recoverCheckoutEffect(effectCtx, conn, state, op, now)
		return err
	})
	return session, resolved, err
}
func (s *Store) recoverCheckoutEffect(ctx context.Context, conn *pgxpool.Conn, state BillingState, op checkoutOperation, now time.Time) (CheckoutSession, bool, error) {
	if op.AccountID != s.checkoutConfig.AccountID {
		return CheckoutSession{}, false, ErrWebCheckoutUnavailable
	}
	op.CustomerID = state.RevenueCatAppUserID
	var session CheckoutSession
	var err error
	if op.SessionID == nil {
		// Stripe retains idempotency results for at least 24 hours. Never recreate an
		// unresolved operation after that guarantee can lapse.
		if now.Sub(op.CreatedAt) >= 23*time.Hour {
			return session, false, ErrWebCheckoutUnavailable
		}
		session, err = s.checkoutProvider.CreateCheckout(ctx, op.CheckoutRequest)
	} else {
		session, err = s.checkoutProvider.GetCheckout(ctx, *op.SessionID)
	}
	if err != nil {
		return session, false, fmt.Errorf("resolve Stripe checkout: %w", err)
	}
	if !validCheckoutSession(session, state.RevenueCatAppUserID, s.accessEnvironment) {
		return session, false, ErrWebCheckoutUnavailable
	}
	if op.SessionID != nil && session.ID != *op.SessionID {
		return session, false, ErrWebCheckoutUnavailable
	}
	if op.SessionID == nil {
		if _, err := conn.Exec(ctx, `UPDATE billing_web_checkouts SET session_id=$3 WHERE billing_account_id=$1 AND environment=$2`, state.BillingAccountID, s.accessEnvironment, session.ID); err != nil {
			return session, false, err
		}
	}
	if session.Status == "expired" {
		return session, true, s.deleteCheckoutOperation(ctx, conn, state.BillingAccountID)
	}
	if session.Status == "complete" && session.SubscriptionCanceled {
		if op.TerminalReconciliationGeneration == nil {
			// Queue a new generation only after observing provider cancellation.
			// The atomic statement works inside the standing transaction and in
			// the no-create resolver, without acquiring another connection.
			var target int64
			err := conn.QueryRow(ctx, `
				WITH requested AS (
					UPDATE billing_accounts
					SET requested_generation=requested_generation+1,
						reconciliation_requested_at=$3,next_attempt_at=$3,updated_at=$3
					WHERE id=$1 AND state='active'
					RETURNING requested_generation
				)
				UPDATE billing_web_checkouts
				SET terminal_reconciliation_generation=(SELECT requested_generation FROM requested)
				WHERE billing_account_id=$1 AND environment=$2
				RETURNING terminal_reconciliation_generation
			`, state.BillingAccountID, s.accessEnvironment, now).Scan(&target)
			return session, false, err
		}
		if state.ReconciledGeneration >= *op.TerminalReconciliationGeneration && checkoutPreflight(state, now) == nil {
			return session, true, s.deleteCheckoutOperation(ctx, conn, state.BillingAccountID)
		}
	}
	return session, false, nil
}
func (s *Store) WebCheckoutURL(ctx context.Context, did syntax.DID, tier Tier, now time.Time) (destination string, err error) {
	if s.checkoutProvider == nil || !tier.Paid() {
		return "", ErrWebCheckoutUnavailable
	}
	err = s.withCheckoutFence(ctx, did, false, func(conn *pgxpool.Conn) error {
		state, err := s.ownerStateUsing(ctx, conn, did)
		if err != nil {
			return err
		}
		op, err := s.readCheckoutOperation(ctx, conn, state.BillingAccountID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if err == nil {
			session, resolved, err := s.recoverCheckout(ctx, conn, did, state, op, now)
			if err != nil {
				return err
			}
			if !resolved {
				if session.Status == "complete" {
					return ErrSubscriptionAlreadyExists
				}
				if err := checkoutPreflight(state, now); err != nil {
					return err
				}
				if op.Tier != tier {
					return ErrWebCheckoutPending
				}
				if !session.ExpiresAt.After(now) {
					return ErrWebCheckoutUnavailable
				}
				destination, err = checkoutURL(session)
				return err
			}
		}
		if err := checkoutPreflight(state, now); err != nil {
			return err
		}
		price := s.checkoutConfig.PlusPriceID
		if tier == TierBusiness {
			price = s.checkoutConfig.BusinessPriceID
		}
		op = checkoutOperation{CheckoutRequest: CheckoutRequest{OperationID: uuid.New(), CustomerID: state.RevenueCatAppUserID, AccountID: s.checkoutConfig.AccountID, PriceID: price, SuccessURL: s.checkoutConfig.SuccessURL, CancelURL: s.checkoutConfig.CancelURL}, Tier: tier, CreatedAt: now}
		if _, err := conn.Exec(ctx, `INSERT INTO billing_web_checkouts(billing_account_id,environment,operation_id,tier,stripe_account_id,price_id,success_url,cancel_url,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, state.BillingAccountID, s.accessEnvironment, op.OperationID, tier, op.AccountID, price, op.SuccessURL, op.CancelURL, now); err != nil {
			return err
		}
		session, _, err := s.recoverCheckout(ctx, conn, did, state, op, now)
		if err != nil {
			if errors.Is(err, errCheckoutStandingDenied) {
				// This operation was just inserted and the standing guard proved
				// no provider dispatch occurred. Previous uncertain operations
				// never enter this branch and must remain recoverable.
				cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				return errors.Join(err, s.deleteCheckoutOperation(cleanup, conn, state.BillingAccountID))
			}
			return err
		}
		if session.Status == "complete" {
			return ErrSubscriptionAlreadyExists
		}
		if session.Status != "open" || !session.ExpiresAt.After(now) {
			return ErrWebCheckoutUnavailable
		}
		destination, err = checkoutURL(session)
		return err
	})
	return destination, err
}

// ResolveWebCheckoutForDeletion never creates an operation. An unknown remote
// outcome can only be recovered with its original persisted idempotency key.
func (s *Store) ResolveWebCheckoutForDeletion(ctx context.Context, did syntax.DID, now time.Time) error {
	if s.checkoutProvider == nil {
		return nil
	}
	return s.withCheckoutFence(ctx, did, true, func(conn *pgxpool.Conn) error {
		state, err := s.ownerStateUsing(ctx, conn, did)
		if err != nil {
			return err
		}
		op, err := s.readCheckoutOperation(ctx, conn, state.BillingAccountID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if op.SessionID == nil {
			return nil
		}
		_, _, err = s.recoverCheckoutEffect(ctx, conn, state, op, now)
		return err
	})
}
func (s *Store) WebManagementURL(ctx context.Context, did syntax.DID, now time.Time) (destination string, err error) {
	if s.checkoutProvider == nil {
		return "", ErrWebManagementUnavailable
	}
	err = s.withCheckoutFence(ctx, did, false, func(conn *pgxpool.Conn) error {
		state, err := s.ownerStateUsing(ctx, conn, did)
		if err != nil {
			return err
		}
		op, err := s.readCheckoutOperation(ctx, conn, state.BillingAccountID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrWebManagementUnavailable
		}
		if err != nil {
			return err
		}
		if op.SessionID == nil || op.AccountID != s.checkoutConfig.AccountID {
			return ErrWebManagementUnavailable
		}
		return s.withCheckoutStanding(ctx, conn, did, func(effectCtx context.Context) error {
			session, err := s.checkoutProvider.GetCheckout(effectCtx, *op.SessionID)
			if err != nil {
				return fmt.Errorf("resolve Stripe management customer: %w", err)
			}
			if !validCheckoutSession(session, state.RevenueCatAppUserID, s.accessEnvironment) || session.ID != *op.SessionID || session.Status != "complete" || session.CustomerID == "" {
				return ErrWebManagementUnavailable
			}
			destination, err = s.checkoutProvider.CreateManagement(effectCtx, session.CustomerID, s.checkoutConfig.BillingReturnURL)
			if err == nil && !ValidStripeManagementURL(destination) {
				return ErrWebManagementUnavailable
			}
			return err
		})
	})
	return destination, err
}

func subscriptionBlocksCheckout(sub BillingSubscription, now time.Time) bool {
	if sub.PendingPayment || sub.GivesAccess || sub.Anomaly != "none" || (sub.CurrentPeriodEndsAt != nil && sub.CurrentPeriodEndsAt.After(now)) || (sub.EndsAt != nil && sub.EndsAt.After(now)) {
		return true
	}
	switch sub.Status {
	case "expired", "refunded", "revoked":
		return false
	default:
		return true
	}
}

// selectAccountSubscription chooses the sole legacy projection for this DID.
// Active subscriptions take precedence over history. Ambiguous concurrent
// subscriptions retain a projection for support visibility but grant no access.
func selectAccountSubscription(subs []ProviderSubscriptionSnapshot, catalog *Catalog) string {
	var selected *ProviderSubscriptionSnapshot
	for i := range subs {
		candidate := &subs[i]
		if _, ok := catalog.Authorize(catalog.projectID, candidate.Environment, candidate.ProductID); !ok {
			continue
		}
		if selected == nil || (candidate.GivesAccess && !selected.GivesAccess) || (candidate.GivesAccess == selected.GivesAccess && laterSubscription(*candidate, *selected)) {
			selected = candidate
		}
	}
	if selected == nil {
		return ""
	}
	return selected.ID
}

func laterSubscription(a, b ProviderSubscriptionSnapshot) bool {
	for _, pair := range [][2]*time.Time{{a.CurrentPeriodEndsAt, b.CurrentPeriodEndsAt}, {a.StartsAt, b.StartsAt}} {
		if pair[0] != nil && pair[1] == nil {
			return true
		}
		if pair[0] == nil && pair[1] != nil {
			return false
		}
		if pair[0] != nil && !pair[0].Equal(*pair[1]) {
			return pair[0].After(*pair[1])
		}
	}
	return strings.Compare(a.ID, b.ID) > 0
}
