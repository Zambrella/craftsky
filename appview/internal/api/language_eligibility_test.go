package api_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

func TestTimelineLanguageEligibilityUsesExactTagsRepostSubjectsAndQuoteOuters(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	ctx := context.Background()
	for _, did := range []string{"did:plc:viewer", "did:plc:alice", "did:plc:bob"} {
		seedMember(t, pool, did)
	}
	seedFollow(t, pool, "did:plc:viewer", "did:plc:alice", "follow-alice")
	base := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)

	frenchSubject := seedPost(t, pool, "did:plc:bob", "french-subject", "French subject", base.Add(5*time.Minute))
	repost := seedInteraction(t, pool, "repost", "did:plc:alice", "repost-french", frenchSubject, false)
	englishOuter := seedQuotePost(t, pool, "did:plc:alice", "english-outer", "English outer", frenchSubject, "bafyfrench", base.Add(3*time.Minute))
	frenchOuter := seedQuotePost(t, pool, "did:plc:alice", "french-outer", "French outer", englishOuter, "bafyenglish", base.Add(2*time.Minute))
	baseFrench := seedPost(t, pool, "did:plc:alice", "base-fr", "Base French", base.Add(time.Minute))
	regionalFrench := seedPost(t, pool, "did:plc:alice", "regional-fr", "Regional French", base)

	if _, err := pool.Exec(ctx, `
		UPDATE craftsky_posts
		SET langs = CASE uri
			WHEN $1 THEN ARRAY['fr']::text[]
			WHEN $2 THEN ARRAY['en']::text[]
			WHEN $3 THEN ARRAY['fr']::text[]
			WHEN $4 THEN ARRAY['fr']::text[]
			WHEN $5 THEN ARRAY['fr-CA']::text[]
			ELSE langs
		END
		WHERE uri IN ($1, $2, $3, $4, $5)
	`, frenchSubject, englishOuter, frenchOuter, baseFrench, regionalFrench); err != nil {
		t.Fatalf("seed languages: %v", err)
	}

	store := api.NewPostStore(pool)
	items, _, err := store.ListTimelineWithLanguages(ctx, "did:plc:viewer", []string{"fr"}, 20, "")
	if err != nil {
		t.Fatalf("ListTimelineWithLanguages: %v", err)
	}
	got := timelineURIs(items)
	if !slices.Contains(got, baseFrench) {
		t.Fatalf("exact base tag post missing from %v", got)
	}
	if slices.Contains(got, regionalFrench) {
		t.Fatalf("regional tag matched base tag in %v", got)
	}
	if slices.Contains(got, englishOuter) {
		t.Fatalf("quote eligibility used its French quoted record instead of English outer in %v", got)
	}
	if !slices.Contains(got, frenchOuter) {
		t.Fatalf("French outer quote was excluded because its quoted record is English: %v", got)
	}
	foundFrenchRepost := false
	for _, item := range items {
		if item.Repost != nil && item.Repost.URI == repost {
			foundFrenchRepost = true
		}
	}
	if !foundFrenchRepost {
		t.Fatalf("French repost subject did not match selected French: %v", got)
	}

	englishItems, _, err := store.ListTimelineWithLanguages(ctx, "did:plc:viewer", []string{"en"}, 20, "")
	if err != nil {
		t.Fatalf("English ListTimelineWithLanguages: %v", err)
	}
	if !slices.Contains(timelineURIs(englishItems), englishOuter) {
		t.Fatalf("eligible English outer quote missing: %v", timelineURIs(englishItems))
	}
	for _, item := range englishItems {
		if item.Repost != nil && item.Repost.URI == repost {
			t.Fatalf("English filter included French repost subject: %v", timelineURIs(englishItems))
		}
	}
}
