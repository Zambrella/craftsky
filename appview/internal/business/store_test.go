package business

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestStoreAccountType(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000061_business_account_types.up.sql")
	if err != nil {
		t.Fatalf("read account type migration: %v", err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE owner_lifecycles (
			owner_did TEXT PRIMARY KEY, state TEXT NOT NULL, generation BIGINT NOT NULL,
			auth_epoch BIGINT NOT NULL, transition_reason TEXT NOT NULL,
			transitioned_at TIMESTAMPTZ NOT NULL, terminal_at TIMESTAMPTZ,
			purge_completed_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		);
	`+string(migration))
	store := NewStore(pool)
	alice := syntax.DID("did:plc:alice")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at)
		VALUES ($1,'active',1,1,'test',now(),now(),now())
	`, alice); err != nil {
		t.Fatalf("seed owner lifecycle: %v", err)
	}
	ctx := ownerlifecycle.WithExpectedGeneration(context.Background(), 1)

	got, err := store.ReadAccountType(ctx, alice)
	if err != nil {
		t.Fatalf("read missing account type: %v", err)
	}
	if got != AccountTypeRegular {
		t.Fatalf("missing account type = %q, want %q", got, AccountTypeRegular)
	}
	assertAccountTypeRowCount(t, pool, alice, 0)

	if err := store.PutAccountType(ctx, alice, AccountTypeBusiness); err != nil {
		t.Fatalf("put business account type: %v", err)
	}
	got, err = store.ReadAccountType(ctx, alice)
	if err != nil {
		t.Fatalf("read business account type: %v", err)
	}
	if got != AccountTypeBusiness {
		t.Fatalf("stored account type = %q, want %q", got, AccountTypeBusiness)
	}
	assertAccountTypeRowCount(t, pool, alice, 1)

	if err := store.PutAccountType(ctx, alice, AccountTypeRegular); err != nil {
		t.Fatalf("put regular account type: %v", err)
	}
	got, err = store.ReadAccountType(ctx, alice)
	if err != nil {
		t.Fatalf("read regular account type: %v", err)
	}
	if got != AccountTypeRegular {
		t.Fatalf("updated account type = %q, want %q", got, AccountTypeRegular)
	}
	assertAccountTypeRowCount(t, pool, alice, 1)

	if err := store.PutAccountType(ctx, alice, AccountType("pro")); !errors.Is(err, ErrInvalidAccountType) {
		t.Fatalf("put invalid account type error = %v, want ErrInvalidAccountType", err)
	}
	got, err = store.ReadAccountType(ctx, alice)
	if err != nil {
		t.Fatalf("read after invalid account type: %v", err)
	}
	if got != AccountTypeRegular {
		t.Fatalf("account type after invalid put = %q, want %q", got, AccountTypeRegular)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE craftsky_account_types SET account_type = 'pro' WHERE owner_did = $1
	`, alice); err == nil {
		t.Fatal("database accepted invalid account type")
	}

	if err := store.DeleteAccountType(ctx, alice); err != nil {
		t.Fatalf("delete account type: %v", err)
	}
	if err := store.DeleteAccountType(ctx, alice); err != nil {
		t.Fatalf("repeat account type deletion: %v", err)
	}
	assertAccountTypeRowCount(t, pool, alice, 0)
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
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_account_types (
			owner_did TEXT PRIMARY KEY,
			account_type TEXT NOT NULL CHECK (account_type IN ('regular', 'business'))
		)
	`)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_account_types(owner_did, account_type)
		VALUES ('did:plc:summary00', 'business')
	`); err != nil {
		t.Fatalf("seed account type: %v", err)
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
			if !strings.Contains(queries[0], "craftsky_account_types") || !strings.Contains(queries[0], "ANY($1)") {
				t.Fatalf("account-type query is not set-based: %s", queries[0])
			}
		})
	}
}

func assertAccountTypeRowCount(t *testing.T, pool *pgxpool.Pool, did syntax.DID, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM craftsky_account_types WHERE owner_did = $1
	`, did).Scan(&got); err != nil {
		t.Fatalf("count account type rows: %v", err)
	}
	if got != want {
		t.Fatalf("account type row count = %d, want %d", got, want)
	}
}
