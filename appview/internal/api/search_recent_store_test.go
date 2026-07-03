package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

const recentSearchStoreDDL = `
CREATE TABLE craftsky_recent_searches (
    id TEXT PRIMARY KEY,
    viewer_did TEXT NOT NULL,
    search_type TEXT NOT NULL CHECK (search_type IN ('query', 'hashtag', 'profile', 'post', 'project')),
    display_label TEXT NOT NULL,
    normalized_payload JSONB NOT NULL,
    normalized_payload_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (viewer_did, search_type, normalized_payload_hash)
);
`

func recentReq(t *testing.T, body string) api.SaveRecentSearchRequest {
	t.Helper()
	req := httptest.NewRequest("POST", "/v1/search/recent", bytes.NewBufferString(body))
	parsed, err := api.DecodeSaveRecentSearchRequest(req)
	if err != nil {
		t.Fatalf("DecodeSaveRecentSearchRequest: %v", err)
	}
	return parsed
}

func TestSearchStore_RecentSearchLifecycleDedupesPrunesAndHardDeletes(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, recentSearchStoreDDL)
	ctx := context.Background()
	store := api.NewSearchStore(pool, nil)
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)

	query := recentReq(t, `{"type":"query","displayLabel":"Alpaca socks","payload":{"q":" Alpaca socks "}}`)
	queryRow, err := store.SaveRecentSearch(ctx, "did:plc:alice", query, now.Add(-time.Minute))
	if err != nil {
		t.Fatalf("SaveRecentSearch query: %v", err)
	}
	var queryPayload map[string]string
	if err := json.Unmarshal(queryRow.NormalizedPayload, &queryPayload); err != nil {
		t.Fatalf("decode query payload: %v", err)
	}
	if queryRow.Type != "query" || queryPayload["q"] != "Alpaca socks" || len(queryPayload) != 1 {
		t.Fatalf("query row = %+v payload=%s", queryRow, queryRow.NormalizedPayload)
	}

	first := recentReq(t, `{"type":"hashtag","displayLabel":"#Sock","payload":{"tag":"#Sock"}}`)
	row, err := store.SaveRecentSearch(ctx, "did:plc:alice", first, now)
	if err != nil {
		t.Fatalf("SaveRecentSearch first: %v", err)
	}
	if row.DisplayLabel != "#Sock" || row.ID == "" {
		t.Fatalf("saved row = %+v", row)
	}
	dup := recentReq(t, `{"type":"hashtag","displayLabel":"#SOCK latest","payload":{"tag":"sock"}}`)
	refreshed, err := store.SaveRecentSearch(ctx, "did:plc:alice", dup, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("SaveRecentSearch duplicate: %v", err)
	}
	if refreshed.ID != row.ID || refreshed.DisplayLabel != "#Sock" || !refreshed.UpdatedAt.After(row.UpdatedAt) {
		t.Fatalf("refreshed = %+v, original = %+v; want same ID, original label, newer updatedAt", refreshed, row)
	}
	if _, err := store.SaveRecentSearch(ctx, "did:plc:bob", first, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("SaveRecentSearch bob: %v", err)
	}
	for i := 0; i < 51; i++ {
		req := recentReq(t, `{"type":"post","displayLabel":"post","payload":{"q":"query-`+string(rune('a'+i))+`"}}`)
		if _, err := store.SaveRecentSearch(ctx, "did:plc:alice", req, now.Add(time.Duration(i+3)*time.Minute)); err != nil {
			t.Fatalf("SaveRecentSearch prune %d: %v", i, err)
		}
	}
	aliceRows, err := store.ListRecentSearches(ctx, "did:plc:alice")
	if err != nil {
		t.Fatalf("ListRecentSearches alice: %v", err)
	}
	if len(aliceRows) != 50 {
		t.Fatalf("alice rows len = %d, want 50", len(aliceRows))
	}
	bobRows, err := store.ListRecentSearches(ctx, "did:plc:bob")
	if err != nil {
		t.Fatalf("ListRecentSearches bob: %v", err)
	}
	if len(bobRows) != 1 || bobRows[0].DisplayLabel != "#Sock" {
		t.Fatalf("bob rows = %+v", bobRows)
	}
	if err := store.DeleteRecentSearch(ctx, "did:plc:alice", bobRows[0].ID); err != nil {
		t.Fatalf("DeleteRecentSearch not-owned: %v", err)
	}
	bobRows, _ = store.ListRecentSearches(ctx, "did:plc:bob")
	if len(bobRows) != 1 {
		t.Fatalf("not-owned delete removed bob row: %+v", bobRows)
	}
	deleteID := aliceRows[0].ID
	if err := store.DeleteRecentSearch(ctx, "did:plc:alice", deleteID); err != nil {
		t.Fatalf("DeleteRecentSearch owned: %v", err)
	}
	if err := store.DeleteRecentSearch(ctx, "did:plc:alice", deleteID); err != nil {
		t.Fatalf("DeleteRecentSearch idempotent: %v", err)
	}
	aliceRows, _ = store.ListRecentSearches(ctx, "did:plc:alice")
	for _, row := range aliceRows {
		if row.ID == deleteID {
			t.Fatalf("hard-deleted row %s still listed", deleteID)
		}
	}
}
