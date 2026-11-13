package accountdeletion

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/subscriptions"
	"social.craftsky/appview/internal/testdb"
)

func TestDeletionCannotOrphanAnUnresolvedStripeCheckout(t *testing.T) {
	accounts, err := os.ReadFile("../../migrations/000076_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(accounts)+stripeCheckoutDeletionDDL(t))
	ctx := context.Background()
	now := time.Now().UTC()
	participant := subscriptions.NewDeletionParticipant()
	for _, scenario := range []struct {
		name, environment string
		sessionID         any
	}{
		{"unknown create outcome", "sandbox", nil},
		{"retained payable session", "production", "cs_live_pending"},
		{"sandbox checkout also blocks deletion", "sandbox", "cs_test_pending"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			owner := syntax.DID("did:plc:" + uuid.NewString())
			accountID := uuid.New()
			if _, err := pool.Exec(ctx, `INSERT INTO billing_accounts
				(id,owner_did,revenuecat_app_user_id,requested_generation,reconciled_generation,deletion_requested_generation,reconciled_at)
				VALUES($1,$2,$3,1,1,1,$4)`, accountID, owner, uuid.New(), now); err != nil {
				t.Fatal(err)
			}
			// Even an old local timestamp cannot prove the remote session expired.
			if _, err := pool.Exec(ctx, `INSERT INTO billing_web_checkouts
				(billing_account_id,environment,operation_id,tier,stripe_account_id,price_id,success_url,cancel_url,session_id,created_at)
				VALUES($1,$2,$3,'plus','acct_test','price_plus','https://example.com/billing','https://example.com/billing',$4,$5)`,
				accountID, scenario.environment, uuid.New(), scenario.sessionID, now.Add(-72*time.Hour)); err != nil {
				t.Fatal(err)
			}
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			closed, err := participant.ConfirmDeletion(ctx, tx, owner, now)
			_ = tx.Rollback(ctx)
			if closed || !errors.Is(err, subscriptions.ErrProviderBillingMustBeResolved) {
				t.Fatalf("unresolved checkout allowed deletion: closed=%t err=%v", closed, err)
			}
			var state string
			var pending int
			if err := pool.QueryRow(ctx, `SELECT state,(SELECT count(*) FROM billing_web_checkouts WHERE billing_account_id=$1)
				FROM billing_accounts WHERE id=$1`, accountID).Scan(&state, &pending); err != nil || state != "active" || pending != 1 {
				t.Fatalf("blocked deletion changed billing: state=%s pending=%d err=%v", state, pending, err)
			}
		})
	}
}
