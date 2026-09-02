package api_test

import (
	"context"
	"reflect"
	"testing"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

type searchNeutralityRow struct {
	DID           string
	FollowedRank  int
	RelevanceRank int
	HandleLower   string
}

type searchNeutralitySnapshot struct {
	rows    []searchNeutralityRow
	cursors []string
}

func TestBusinessSearchNeutrality(t *testing.T) {
	assertBusinessSearchNeutrality(t)
}

func assertBusinessSearchNeutrality(t *testing.T) {
	t.Helper()
	pool := testdb.WithSchema(t, searchStoreDDL+businessNeutralityDDL)
	ctx := context.Background()
	for _, did := range []string{"did:plc:viewer", "did:plc:alice", "did:plc:bob", "did:plc:carol"} {
		seedMember(t, pool, did)
	}
	seedSearchIdentity(t, pool, "did:plc:viewer", "viewer.test", "Viewer", "")
	seedSearchIdentity(t, pool, "did:plc:alice", "alice-maker.test", "Craft Maker", "maker profile")
	seedSearchIdentity(t, pool, "did:plc:bob", "bob-maker.test", "Craft Maker", "maker profile")
	seedSearchIdentity(t, pool, "did:plc:carol", "carol-maker.test", "Craft Maker", "maker profile")
	seedFollow(t, pool, "did:plc:viewer", "did:plc:carol", "follow-carol")

	store := api.NewSearchStore(pool, nil)
	before := captureSearchNeutralitySnapshot(t, store, "did:plc:viewer")

	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_account_types(owner_did, account_type)
		VALUES ('did:plc:alice', 'business'), ('did:plc:bob', 'regular'), ('did:plc:carol', 'business');
		INSERT INTO craftsky_business_profiles(owner_did, uri, cid, raw_record, source_revision)
		VALUES
			('did:plc:alice', 'at://did:plc:alice/social.craftsky.business.profile/self', 'bafysearchalice',
			 '{"$type":"social.craftsky.business.profile","businessTypes":["dyer"],"offerings":["fiber"]}', '3msearchalice'),
			('did:plc:bob', 'at://did:plc:bob/social.craftsky.business.profile/self', 'bafysearchbob',
			 '{"$type":"social.craftsky.business.profile","businessTypes":["teacher"],"offerings":["classes"]}', '3msearchbob'),
			('did:plc:carol', 'at://did:plc:carol/social.craftsky.business.profile/self', 'bafysearchcarol',
			 '{"$type":"social.craftsky.business.profile","businessTypes":["pattern-designer"],"offerings":["patterns"]}', '3msearchcarol')
	`); err != nil {
		t.Fatalf("change account type and taxonomy: %v", err)
	}

	after := captureSearchNeutralitySnapshot(t, store, "did:plc:viewer")
	if !reflect.DeepEqual(after.rows, before.rows) {
		t.Fatalf("search rank/order after business changes = %#v, want unchanged %#v", after.rows, before.rows)
	}
	if !reflect.DeepEqual(after.cursors, before.cursors) {
		t.Fatalf("search cursors after business changes = %v, want unchanged %v", after.cursors, before.cursors)
	}
}

func captureSearchNeutralitySnapshot(t *testing.T, store *api.SearchStore, viewerDID string) searchNeutralitySnapshot {
	t.Helper()
	var snapshot searchNeutralitySnapshot
	cursor := ""
	for {
		rows, next, err := store.SearchProfiles(context.Background(), viewerDID, api.ProfileSearchRequest{
			Query:  "maker",
			Limit:  2,
			Cursor: cursor,
		})
		if err != nil {
			t.Fatalf("search profiles cursor %q: %v", cursor, err)
		}
		for _, row := range rows {
			snapshot.rows = append(snapshot.rows, searchNeutralityRow{
				DID:           row.DID,
				FollowedRank:  row.FollowedRank,
				RelevanceRank: row.RelevanceRank,
				HandleLower:   row.HandleLower,
			})
		}
		snapshot.cursors = append(snapshot.cursors, next)
		if next == "" {
			return snapshot
		}
		cursor = next
	}
}
