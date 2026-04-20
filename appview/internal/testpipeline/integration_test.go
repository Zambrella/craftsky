package testpipeline_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testpipeline"
)

// TestPipelineEndToEnd wires the real Dispatcher → Indexer → Postgres →
// Handler chain, pushes a synthetic Tap event in one end, and asserts the
// record comes back out of GET /test/feed. If this test fails, the
// indexing half of the pipeline is broken. (The WS consumer → Dispatcher
// leg is covered by existing tap package tests.)
func TestPipelineEndToEnd(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)

	dispatcher := index.NewDispatcher(index.NotImplemented{})
	dispatcher.Register("social.craftsky.test.post", testpipeline.NewIndexer(pool))

	rec, _ := json.Marshal(map[string]any{
		"text":      "end-to-end",
		"createdAt": "2026-04-19T10:00:00Z",
	})
	ev := tap.Event{
		URI:        "at://did:plc:e2e/social.craftsky.test.post/3kxzzz",
		CID:        "bafyzzz",
		DID:        "did:plc:e2e",
		Collection: "social.craftsky.test.post",
		Rkey:       "3kxzzz",
		Action:     "create",
		Record:     rec,
	}
	if err := dispatcher.Handle(context.Background(), ev); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/test/feed", nil)
	w := httptest.NewRecorder()
	testpipeline.NewHandler(pool).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", w.Code)
	}
	var body struct {
		Posts []struct {
			URI  string `json:"uri"`
			Text string `json:"text"`
		} `json:"posts"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, p := range body.Posts {
		if p.URI == ev.URI && p.Text == "end-to-end" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("posted record not in feed: %+v", body.Posts)
	}
}
