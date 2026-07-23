package instagram

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/testdb"
)

func TestSuggestionMatcherCreatesOnlyEligibleFollowingSupportBeforePrivacyFinalization(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000023_instagram_migration.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(migration))
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 21, 0, 0, 0, time.UTC)
	alice := syntax.DID("did:plc:synthetic-alice")
	bob := syntax.DID("did:plc:synthetic-bob")
	insertVerifiedImportOwner(t, pool, alice, "synthetic.alice", now)
	seedSuggestionLink(t, pool, bob, "synthetic.bob", now)
	store := NewSuggestionStore(pool)
	policy := matcherPolicy{decision: EligibilityDecision{Eligible: true, Reason: EligibilityAllowed}}
	matcher := NewSuggestionMatcher(pool, store, policy, func() time.Time { return now })
	importStore := NewImportStore(pool)
	service, err := NewImportService(ImportServiceOptions{
		Repository: importStore, Matcher: matcher, Now: func() time.Time { return now },
		NewID: func() uuid.UUID { return uuid.MustParse("00000000-0000-0000-0000-000000000271") },
	})
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateImport(ctx, alice, ImportSourceInstagramJSON, []ImportEntry{
		{Username: "synthetic.bob"},
		{Username: "unmatched.synthetic"},
	})
	if err != nil {
		t.Fatalf("create import: %v", err)
	}
	if created.InitialSuggestionCount != 1 || created.Counts.Following != 2 {
		t.Fatalf("created=%+v", created)
	}
	var suggestions, supports, handles int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_follow_suggestions WHERE importer_did = $1 AND target_did = $2 AND state = 'pending'`, alice, bob).Scan(&suggestions); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_suggestion_sources`).Scan(&supports); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_graph_handles`).Scan(&handles); err != nil {
		t.Fatal(err)
	}
	if suggestions != 1 || supports != 1 || handles != 2 {
		t.Fatalf("suggestions=%d supports=%d handles=%d", suggestions, supports, handles)
	}
	var username string
	var matched bool
	if err := pool.QueryRow(ctx, `SELECT username_normalized, matched FROM instagram_graph_handles WHERE matched`).Scan(&username, &matched); err != nil {
		t.Fatal(err)
	}
	if username != "synthetic.bob" || !matched {
		t.Fatalf("retained username=%q matched=%t", username, matched)
	}
}

func TestSuggestionMatcherFailsClosedAndRetainsGraphForFutureReconciliation(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000023_instagram_migration.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(migration))
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 21, 30, 0, 0, time.UTC)
	alice := syntax.DID("did:plc:synthetic-alice")
	bob := syntax.DID("did:plc:synthetic-bob")
	insertVerifiedImportOwner(t, pool, alice, "synthetic.alice", now)
	seedSuggestionLink(t, pool, bob, "synthetic.bob", now)
	matcher := NewSuggestionMatcher(pool, NewSuggestionStore(pool), matcherPolicy{decision: EligibilityDecision{Reason: EligibilitySafetyUnavailable}}, func() time.Time { return now })
	service, err := NewImportService(ImportServiceOptions{Repository: NewImportStore(pool), Matcher: matcher, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateImport(ctx, alice, ImportSourceManual, []ImportEntry{{Username: "synthetic.bob"}})
	if err != nil {
		t.Fatal(err)
	}
	if created.InitialSuggestionCount != 0 {
		t.Fatalf("suggestion count=%d", created.InitialSuggestionCount)
	}
	var handles, suggestions int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_graph_handles`).Scan(&handles); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_follow_suggestions`).Scan(&suggestions); err != nil {
		t.Fatal(err)
	}
	if handles != 1 || suggestions != 0 {
		t.Fatalf("handles=%d suggestions=%d", handles, suggestions)
	}
}

type matcherPolicy struct{ decision EligibilityDecision }

func (p matcherPolicy) Evaluate(context.Context, EligibilityStage, SuggestionEligibilityRequest) (EligibilityDecision, error) {
	return p.decision, nil
}

var _ InstagramSuggestionEligibilityPolicy = matcherPolicy{}
