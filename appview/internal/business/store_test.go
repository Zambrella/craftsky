package business

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestStoreAccountType(t *testing.T) {
	migration, err := testdb.ReadMigration("000076_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(migration))
	store := NewStore(pool)
	alice := syntax.DID("did:plc:alice")
	ctx := context.Background()
	got, err := store.ReadAccountType(ctx, alice)
	if err != nil || got != AccountTypeRegular {
		t.Fatalf("unassigned = %q, %v", got, err)
	}
	for _, statement := range []string{
		`INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id) VALUES ('10000000-0000-4000-8000-000000000001','did:plc:owner','20000000-0000-4000-8000-000000000001')`,
		`INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,mapped_tier,accepted_generation) VALUES ('30000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000001','project','subscription','business','app','app_store','production','active',true,'business',1)`,
		`INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at) VALUES ('30000000-0000-4000-8000-000000000001','business','did:plc:alice',now())`,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	got, err = store.ReadAccountType(ctx, alice)
	if err != nil || got != AccountTypeBusiness {
		t.Fatalf("licensed = %q, %v", got, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE provider_subscriptions SET gives_access=false WHERE id='30000000-0000-4000-8000-000000000001'`); err != nil {
		t.Fatal(err)
	}
	got, err = store.ReadAccountType(ctx, alice)
	if err != nil || got != AccountTypeRegular {
		t.Fatalf("lapsed = %q, %v", got, err)
	}
}

type accountTypeQueryTracer struct {
	mu      sync.Mutex
	queries []string
}

func (tracer *accountTypeQueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	tracer.mu.Lock()
	tracer.queries = append(tracer.queries, data.SQL)
	tracer.mu.Unlock()
	return ctx
}

func (*accountTypeQueryTracer) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func (tracer *accountTypeQueryTracer) reset() {
	tracer.mu.Lock()
	tracer.queries = nil
	tracer.mu.Unlock()
}

func (tracer *accountTypeQueryTracer) snapshot() []string {
	tracer.mu.Lock()
	defer tracer.mu.Unlock()
	return append([]string(nil), tracer.queries...)
}

func TestStoreReadAccountTypesUsesOneSetBasedQuery(t *testing.T) {
	migration, err := testdb.ReadMigration("000076_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(migration))
	for _, statement := range []string{
		`INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id) VALUES ('10000000-0000-4000-8000-000000000001','did:plc:owner','20000000-0000-4000-8000-000000000001')`,
		`INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,mapped_tier,accepted_generation) VALUES ('30000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000001','project','subscription','business','app','app_store','production','active',true,'business',1)`,
		`INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at) VALUES ('30000000-0000-4000-8000-000000000001','business','did:plc:summary00',now())`,
	} {
		if _, err := pool.Exec(context.Background(), statement); err != nil {
			t.Fatal(err)
		}
	}

	tracer := &accountTypeQueryTracer{}
	config := pool.Config().Copy()
	config.ConnConfig.Tracer = tracer
	tracedPool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("create traced pool: %v", err)
	}
	t.Cleanup(tracedPool.Close)
	store := NewStore(tracedPool)

	for _, count := range []int{1, 50} {
		t.Run(fmt.Sprintf("dids_%d", count), func(t *testing.T) {
			dids := make([]syntax.DID, count)
			for index := range dids {
				dids[index] = syntax.DID(fmt.Sprintf("did:plc:summary%02d", index))
			}
			tracer.reset()
			values, err := store.ReadAccountTypes(context.Background(), dids)
			if err != nil {
				t.Fatalf("ReadAccountTypes: %v", err)
			}
			if values["did:plc:summary00"] != AccountTypeBusiness {
				t.Fatalf("stored account type = %q, want business", values["did:plc:summary00"])
			}
			queries := tracer.snapshot()
			if len(queries) != 1 {
				t.Fatalf("SQL query count = %d, want 1: %v", len(queries), queries)
			}
			if strings.Contains(queries[0], "craftsky_account_types") || !strings.Contains(queries[0], "ANY($1)") {
				t.Fatalf("account-type query is not set-based: %s", queries[0])
			}
		})
	}
}
