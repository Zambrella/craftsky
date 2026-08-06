package api_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

func TestProfilePinQueriesUseBoundedIndexedAccess(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000035_profile_pins.up.sql")
	if err != nil {
		t.Fatalf("read profile pin migration: %v", err)
	}
	pool := testdb.WithSchema(t, postStoreDDL+string(migration))
	ctx := context.Background()
	seedMember(t, pool, "did:plc:alice")
	uri := seedPost(t, pool, "did:plc:alice", "pin", "Pinned", time.Now())
	if _, err := pool.Exec(ctx, `
		INSERT INTO profile_pins (
			owner_did, slot, post_uri, state_token, created_at, updated_at
		)
		VALUES (
			'did:plc:alice', 'standard', $1,
			'00000000-0000-4000-8000-000000000001', now(), now()
		)
	`, uri); err != nil {
		t.Fatalf("seed pin: %v", err)
	}
	if _, err := pool.Exec(ctx, `SET enable_seqscan = off`); err != nil {
		t.Fatalf("disable sequential scans: %v", err)
	}
	rows, err := pool.Query(ctx, `
		EXPLAIN (COSTS OFF)
		SELECT p.uri
		FROM profile_pins pin
		JOIN craftsky_posts p ON p.uri = pin.post_uri
		WHERE pin.owner_did = 'did:plc:alice'
		  AND pin.slot = 'standard'
	`)
	if err != nil {
		t.Fatalf("explain profile pin list read: %v", err)
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("scan explain line: %v", err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate explain: %v", err)
	}
	plan := strings.Join(lines, "\n")
	if !strings.Contains(plan, "profile_pins_pkey") {
		t.Fatalf("plan does not use owner/slot primary-key index:\n%s", plan)
	}
	if !strings.Contains(plan, "craftsky_posts_pkey") {
		t.Fatalf("plan does not use post URI index:\n%s", plan)
	}
}

func TestProfilePinHandlerCallsStayFixedAsPageSizeGrows(t *testing.T) {
	for _, itemCount := range []int{1, 10, 50} {
		itemCount := itemCount
		t.Run(twoDigits(itemCount), func(t *testing.T) {
			base := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
			rows := make([]*api.PostRow, 0, itemCount)
			for index := 0; index < itemCount; index++ {
				rows = append(rows, testPostRow(
					"did:plc:alice",
					"row-"+twoDigits(index),
					"Post",
					base.Add(-time.Duration(index)*time.Minute),
				))
			}
			store := &countingProfilePostStore{fakePostStore: fakePostStore{listRows: rows}}
			pinReader := &countingProfilePinReader{}
			handler := api.ListPostsByAuthorHandler(
				store,
				fakeResolver{handleFor: "alice.example"},
				nilLogger(),
				pinReader,
			)
			response := requestProfilePinPage(t, handler, "did:plc:alice", itemCount, "")
			if response.Code != 200 {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if pinReader.calls != 1 || store.listCalls != 1 || store.engagementCalls != 1 {
				t.Fatalf(
					"itemCount=%d calls pin/list/engagement = %d/%d/%d",
					itemCount,
					pinReader.calls,
					store.listCalls,
					store.engagementCalls,
				)
			}
		})
	}
}

type countingProfilePinReader struct {
	calls int
}

func (reader *countingProfilePinReader) ReadProfileListPin(
	context.Context,
	syntax.DID,
	syntax.DID,
	api.ProfilePinSlot,
	[]string,
) (*api.ProfileListPin, error) {
	reader.calls++
	return nil, nil
}

type countingProfilePostStore struct {
	fakePostStore
	listCalls int
}

func (store *countingProfilePostStore) ListByAuthor(
	_ context.Context,
	_ string,
	_ int,
	_ string,
) ([]*api.PostRow, string, error) {
	store.listCalls++
	return store.listRows, store.listCursor, store.listErr
}
