package instagram

import (
	"context"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestPostgresInstagramSuggestionEligibilityPolicyFailsClosedWithoutRelationshipSafety(t *testing.T) {
	pool := testdb.WithSchema(t, eligibilityPolicySchema)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 17, 0, 0, 0, time.UTC)
	seedEligibilityPolicyFacts(t, pool, now)
	request := SuggestionEligibilityRequest{
		ImporterDID:      syntax.DID("did:plc:synthetic-alice"),
		TargetDID:        syntax.DID("did:plc:synthetic-bob"),
		ImportedUsername: "synthetic.bob",
		Direction:        DirectionFollowing,
	}

	closed := NewPostgresInstagramSuggestionEligibilityPolicy(pool, nil, func() time.Time { return now })
	decision, err := closed.Evaluate(ctx, EligibilityAtMatch, request)
	if err != nil {
		t.Fatalf("closed evaluate: %v", err)
	}
	if decision.Eligible || decision.Reason != EligibilitySafetyUnavailable {
		t.Fatalf("closed decision=%+v", decision)
	}

	open := NewPostgresInstagramSuggestionEligibilityPolicy(pool, staticRelationshipSafety{facts: RelationshipSafetyFacts{Available: true}}, func() time.Time { return now })
	for _, stage := range AllEligibilityStages() {
		decision, err := open.Evaluate(ctx, stage, request)
		if err != nil {
			t.Fatalf("stage %s: %v", stage, err)
		}
		if !decision.Eligible || decision.Reason != EligibilityAllowed {
			t.Fatalf("stage %s decision=%+v", stage, decision)
		}
	}
}

func TestPostgresInstagramSuggestionEligibilityPolicyAppliesSafetyAndCurrentFacts(t *testing.T) {
	pool := testdb.WithSchema(t, eligibilityPolicySchema)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 17, 30, 0, 0, time.UTC)
	seedEligibilityPolicyFacts(t, pool, now)
	request := SuggestionEligibilityRequest{
		ImporterDID:      syntax.DID("did:plc:synthetic-alice"),
		TargetDID:        syntax.DID("did:plc:synthetic-bob"),
		ImportedUsername: "synthetic.bob",
		Direction:        DirectionFollowing,
	}

	blocked := NewPostgresInstagramSuggestionEligibilityPolicy(pool, staticRelationshipSafety{facts: RelationshipSafetyFacts{Available: true, TargetBlocksImporter: true}}, func() time.Time { return now })
	decision, err := blocked.Evaluate(ctx, EligibilityAtAccept, request)
	if err != nil || decision.Eligible || decision.Reason != EligibilityRelationshipSafety {
		t.Fatalf("blocked decision=%+v err=%v", decision, err)
	}

	if _, err := pool.Exec(ctx, `INSERT INTO atproto_follows (did, subject_did) VALUES ($1, $2)`, request.ImporterDID, request.TargetDID); err != nil {
		t.Fatal(err)
	}
	open := NewPostgresInstagramSuggestionEligibilityPolicy(pool, staticRelationshipSafety{facts: RelationshipSafetyFacts{Available: true}}, func() time.Time { return now })
	decision, err = open.Evaluate(ctx, EligibilityAtAccept, request)
	if err != nil || decision.Eligible || decision.Reason != EligibilityAlreadyFollowing {
		t.Fatalf("already-following decision=%+v err=%v", decision, err)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_profiles WHERE did = $1`, request.TargetDID); err != nil {
		t.Fatal(err)
	}
	decision, err = open.Evaluate(ctx, EligibilityAtList, request)
	if err != nil || decision.Eligible || decision.Reason != EligibilityMembership {
		t.Fatalf("departed decision=%+v err=%v", decision, err)
	}
}

type staticRelationshipSafety struct {
	facts RelationshipSafetyFacts
}

func (s staticRelationshipSafety) RelationshipSafety(context.Context, syntax.DID, syntax.DID) (RelationshipSafetyFacts, error) {
	return s.facts, nil
}

const eligibilityPolicySchema = `
	CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
	CREATE TABLE instagram_account_links (
		id UUID PRIMARY KEY,
		owner_did TEXT NOT NULL,
		state TEXT NOT NULL,
		username_normalized TEXT,
		discoverable BOOLEAN NOT NULL,
		conflict_pending BOOLEAN NOT NULL,
		verified_at TIMESTAMPTZ NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL
	);
	CREATE TABLE atproto_follows (did TEXT NOT NULL, subject_did TEXT NOT NULL);
	CREATE TABLE moderation_outputs (
		id TEXT PRIMARY KEY,
		source_did TEXT NOT NULL,
		subject_type TEXT NOT NULL,
		subject_did TEXT NOT NULL,
		value TEXT NOT NULL,
		action TEXT NOT NULL,
		expires_at TIMESTAMPTZ,
		indexed_at TIMESTAMPTZ NOT NULL
	);`

func seedEligibilityPolicyFacts(t *testing.T, pool *pgxpool.Pool, now time.Time) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles (did) VALUES ('did:plc:synthetic-alice'), ('did:plc:synthetic-bob')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_account_links (
			id, owner_did, state, username_normalized, discoverable,
			conflict_pending, verified_at, updated_at
		) VALUES ($1, 'did:plc:synthetic-bob', 'active', 'synthetic.bob', true, false, $2, $2)
	`, uuid.New(), now); err != nil {
		t.Fatal(err)
	}
}
