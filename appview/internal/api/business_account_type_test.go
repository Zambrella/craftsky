package api_test

import (
	"context"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

type businessLifecycleReader map[syntax.DID]ownerlifecycle.Lifecycle

func (r businessLifecycleReader) Get(_ context.Context, did syntax.DID) (ownerlifecycle.Lifecycle, error) {
	return r[did], nil
}

func TestBusinessAccountTypeIsReadOnlyAndFollowsAssignedAccess(t *testing.T) {
	pool := testdb.WithSchema(t, subscriptionSchema(t))
	store := business.NewStore(pool)
	ctx := context.Background()
	owner := syntax.DID("did:plc:owner")
	beneficiary := syntax.DID("did:plc:beneficiary")
	seedBusinessTestLicense(t, pool, beneficiary)
	for _, tc := range []struct {
		did  syntax.DID
		want business.AccountType
	}{
		{owner, business.AccountTypeRegular}, {beneficiary, business.AccountTypeBusiness},
	} {
		got, err := store.ReadAccountType(ctx, tc.did)
		if err != nil || got != tc.want {
			t.Fatalf("%s type = %s, err=%v", tc.did, got, err)
		}
	}
	var oldTable any
	if err := pool.QueryRow(ctx, `SELECT to_regclass('craftsky_account_types')`).Scan(&oldTable); err != nil || oldTable != nil {
		t.Fatalf("legacy table = %v, error=%v", oldTable, err)
	}
}
