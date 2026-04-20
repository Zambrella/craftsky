package testpipeline_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testpipeline"
)

func TestHandler_EmptyTableReturnsEmptyArray(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	h := testpipeline.NewHandler(pool)

	req := httptest.NewRequest(http.MethodGet, "/test/feed", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rec.Code)
	}
	var body struct {
		Posts []any `json:"posts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Posts == nil {
		t.Error("posts field should be non-nil empty array, not null")
	}
	if len(body.Posts) != 0 {
		t.Errorf("posts: got %d items want 0", len(body.Posts))
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("content-type: got %q", ct)
	}
}

func TestHandler_ReturnsReverseChronological(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	ix := testpipeline.NewIndexer(pool)
	h := testpipeline.NewHandler(pool)

	for _, row := range []struct {
		uri, text, ts string
	}{
		{"at://did:plc:a/social.craftsky.test.post/1", "first", "2026-04-19T10:00:00Z"},
		{"at://did:plc:a/social.craftsky.test.post/2", "second", "2026-04-19T11:00:00Z"},
		{"at://did:plc:a/social.craftsky.test.post/3", "third", "2026-04-19T12:00:00Z"},
	} {
		rec, _ := json.Marshal(map[string]any{"text": row.text, "createdAt": row.ts})
		ev := tapEvent(row.uri, "bafy", "did:plc:a", "create", rec)
		if err := ix.Handle(context.Background(), ev); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/test/feed", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var body struct {
		Posts []struct {
			Text string `json:"text"`
		} `json:"posts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Posts) != 3 {
		t.Fatalf("got %d posts want 3", len(body.Posts))
	}
	if body.Posts[0].Text != "third" || body.Posts[2].Text != "first" {
		t.Errorf("order wrong: %+v", body.Posts)
	}
}

// All records share createdAt; order among ties is unspecified. This
// test only checks count, not order — don't "fix" it.
func TestHandler_LimitRespected(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	ix := testpipeline.NewIndexer(pool)
	h := testpipeline.NewHandler(pool)

	for i := 0; i < 5; i++ {
		rec, _ := json.Marshal(map[string]any{"text": "x", "createdAt": "2026-04-19T10:00:00Z"})
		ev := tapEvent(
			fmt.Sprintf("at://did:plc:a/social.craftsky.test.post/%d", i),
			"bafy", "did:plc:a", "create", rec,
		)
		if err := ix.Handle(context.Background(), ev); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/test/feed?limit=2", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var body struct {
		Posts []any `json:"posts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Posts) != 2 {
		t.Errorf("got %d want 2", len(body.Posts))
	}
}

// Seeds 201 rows so the clamp is actually observable in the response.
// Without the clamp the handler would return 201; with the clamp it
// returns 200.
func TestHandler_LimitClampedTo200(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	ix := testpipeline.NewIndexer(pool)
	h := testpipeline.NewHandler(pool)

	for i := 0; i < 201; i++ {
		rec, _ := json.Marshal(map[string]any{"text": "x", "createdAt": "2026-04-19T10:00:00Z"})
		ev := tapEvent(
			fmt.Sprintf("at://did:plc:a/social.craftsky.test.post/%d", i),
			"bafy", "did:plc:a", "create", rec,
		)
		if err := ix.Handle(context.Background(), ev); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/test/feed?limit=999", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200 (limit should clamp, not 400)", rec.Code)
	}
	var body struct {
		Posts []any `json:"posts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Posts) != 200 {
		t.Errorf("got %d posts, want 200 (clamp)", len(body.Posts))
	}
}

func TestHandler_InvalidLimitReturns400(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	h := testpipeline.NewHandler(pool)

	for _, q := range []string{"abc", "-1", "0"} {
		req := httptest.NewRequest(http.MethodGet, "/test/feed?limit="+q, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("limit=%q: got %d want 400", q, rec.Code)
		}
	}
}

// tapEvent is a tiny test helper kept local to avoid exporting a helper
// that will be deleted with the package.
func tapEvent(uri, cid, did, action string, record []byte) tap.Event {
	return tap.Event{
		URI: uri, CID: cid, DID: did,
		Collection: "social.craftsky.test.post",
		Action:     action, Record: record,
	}
}
