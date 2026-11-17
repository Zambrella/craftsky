package subscriptions

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WebPaywall struct {
	APIKey    string `json:"apiKey"`
	AppUserID string `json:"appUserId"`
	Offering  string `json:"offering"`
}

func (p WebPaywall) String() string             { return "WebPaywall{private:[REDACTED]}" }
func (p WebPaywall) GoString() string           { return p.String() }
func (p WebPaywall) Format(f fmt.State, _ rune) { _, _ = io.WriteString(f, p.String()) }

var revenueCatStripeKey = regexp.MustCompile(`^strp_(?:sb_)?[A-Za-z0-9]+$`)

// ValidWebPaywallKey prevents configuring a live SDK against sandbox access (or vice versa).
func ValidWebPaywallKey(key, environment string) bool {
	return revenueCatStripeKey.MatchString(key) &&
		(environment == "production" || environment == "sandbox") &&
		strings.HasPrefix(key, "strp_sb_") == (environment == "sandbox")
}

func (s *Store) PrepareWebPaywall(ctx context.Context, did syntax.DID, tier Tier, now time.Time) (WebPaywall, error) {
	var result WebPaywall
	key := s.checkoutConfig.RevenueCatWebKey
	if !tier.Paid() || !ValidWebPaywallKey(key, s.accessEnvironment) {
		return result, ErrWebCheckoutUnavailable
	}
	err := s.withCheckoutFence(ctx, did, false, func(conn *pgxpool.Conn) error {
		state, err := s.ownerStateUsing(ctx, conn, did, now)
		if err != nil {
			return err
		}
		if state.WebCheckoutPending {
			return ErrWebCheckoutPending
		}
		if err := checkoutPreflight(state, now); err != nil {
			return err
		}
		result = WebPaywall{APIKey: key, AppUserID: state.RevenueCatAppUserID.String(), Offering: string(tier)}
		return nil
	})
	return result, err
}
