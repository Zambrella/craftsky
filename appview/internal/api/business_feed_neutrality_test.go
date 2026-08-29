package api_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

const businessNeutralityDDL = `
CREATE TABLE craftsky_account_types (
    owner_did TEXT PRIMARY KEY,
    account_type TEXT NOT NULL CHECK (account_type IN ('regular', 'business'))
);

CREATE TABLE craftsky_business_profiles (
    owner_did TEXT PRIMARY KEY,
    uri TEXT NOT NULL UNIQUE,
    cid TEXT NOT NULL,
    raw_record JSONB NOT NULL,
    source_revision TEXT NOT NULL
);
`

type feedNeutralitySnapshot struct {
	itemKeys []string
	cursors  []string
}

func TestBusinessFeedNeutrality(t *testing.T) {
	assertBusinessFeedNeutrality(t)
}

func assertBusinessFeedNeutrality(t *testing.T) {
	t.Helper()
	pool := testdb.WithSchema(t, timelineStoreDDL+businessNeutralityDDL)
	ctx := context.Background()
	for _, did := range []string{"did:plc:viewer", "did:plc:alice", "did:plc:bob"} {
		seedMember(t, pool, did)
	}
	seedFollow(t, pool, "did:plc:viewer", "did:plc:alice", "follow-alice")
	seedFollow(t, pool, "did:plc:viewer", "did:plc:bob", "follow-bob")
	base := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	seedPost(t, pool, "did:plc:alice", "newest", "same neutral fixture", base.Add(2*time.Minute))
	seedPost(t, pool, "did:plc:bob", "middle", "same neutral fixture", base.Add(time.Minute))
	seedPost(t, pool, "did:plc:alice", "oldest", "same neutral fixture", base)

	store := api.NewPostStore(pool)
	before := captureFeedNeutralitySnapshot(t, store, "did:plc:viewer")

	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_account_types(owner_did, account_type)
		VALUES ('did:plc:alice', 'business'), ('did:plc:bob', 'regular');
		INSERT INTO craftsky_business_profiles(owner_did, uri, cid, raw_record, source_revision)
		VALUES
			('did:plc:alice', 'at://did:plc:alice/social.craftsky.business.profile/self', 'bafyalicebusiness',
			 '{"$type":"social.craftsky.business.profile","businessTypes":["dyer"],"offerings":["yarn"]}', '3mfeedalice'),
			('did:plc:bob', 'at://did:plc:bob/social.craftsky.business.profile/self', 'bafybobbusiness',
			 '{"$type":"social.craftsky.business.profile","businessTypes":["teacher"],"offerings":["classes"]}', '3mfeedbob')
	`); err != nil {
		t.Fatalf("change account type and taxonomy: %v", err)
	}

	after := captureFeedNeutralitySnapshot(t, store, "did:plc:viewer")
	if !slices.Equal(after.itemKeys, before.itemKeys) {
		t.Fatalf("feed identities after business changes = %v, want unchanged %v", after.itemKeys, before.itemKeys)
	}
	if !slices.Equal(after.cursors, before.cursors) {
		t.Fatalf("feed cursors after business changes = %v, want unchanged %v", after.cursors, before.cursors)
	}
}

func captureFeedNeutralitySnapshot(t *testing.T, store *api.PostStore, viewerDID string) feedNeutralitySnapshot {
	t.Helper()
	var snapshot feedNeutralitySnapshot
	cursor := ""
	for {
		items, next, err := store.ListTimeline(context.Background(), viewerDID, 2, cursor)
		if err != nil {
			t.Fatalf("list timeline cursor %q: %v", cursor, err)
		}
		for _, item := range items {
			snapshot.itemKeys = append(snapshot.itemKeys, item.ItemKey)
		}
		snapshot.cursors = append(snapshot.cursors, next)
		if next == "" {
			return snapshot
		}
		cursor = next
	}
}
