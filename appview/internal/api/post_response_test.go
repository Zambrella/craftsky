// appview/internal/api/post_response_test.go
package api_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
)

func ptrStr(s string) *string { return &s }

func baseRow() *api.PostRow {
	return &api.PostRow{
		URI:       "at://did:plc:alice/social.craftsky.feed.post/rk1",
		DID:       "did:plc:alice",
		Rkey:      "rk1",
		CID:       "bafycid",
		Text:      "hello",
		Tags:      []string{},
		CreatedAt: time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC),
		IndexedAt: time.Date(2026, 5, 4, 12, 0, 1, 0, time.UTC),
	}
}

func TestBuildPostResponse_MinimalPost(t *testing.T) {
	t.Parallel()
	resp := api.BuildPostResponse(baseRow(), syntax.Handle("alice.example"))
	if resp.URI != "at://did:plc:alice/social.craftsky.feed.post/rk1" {
		t.Errorf("uri = %q", resp.URI)
	}
	if resp.Author.DID != "did:plc:alice" || resp.Author.Handle != "alice.example" {
		t.Errorf("author = %+v", resp.Author)
	}
	if resp.Rkey != "rk1" {
		t.Errorf("rkey = %q", resp.Rkey)
	}
	if resp.CID != "bafycid" {
		t.Errorf("cid = %q", resp.CID)
	}
	if resp.Text != "hello" {
		t.Errorf("text = %q", resp.Text)
	}
	if !resp.CreatedAt.Equal(baseRow().CreatedAt) {
		t.Errorf("createdAt = %v", resp.CreatedAt)
	}
	if !resp.IndexedAt.Equal(baseRow().IndexedAt) {
		t.Errorf("indexedAt = %v", resp.IndexedAt)
	}
	if resp.Reply != nil {
		t.Errorf("expected nil reply, got %+v", resp.Reply)
	}
	if resp.Quote != nil {
		t.Errorf("expected nil quote, got %+v", resp.Quote)
	}
	if resp.Author.DisplayName != nil {
		t.Errorf("expected nil displayName")
	}
	// Tags must serialise as []
	b, _ := json.Marshal(resp.Tags)
	if string(b) != "[]" {
		t.Errorf("tags = %s", b)
	}
}

func TestBuildPostResponse_WithReplyAndQuote(t *testing.T) {
	t.Parallel()
	row := baseRow()
	row.ReplyRootURI = ptrStr("at://did:plc:bob/social.craftsky.feed.post/r1")
	row.ReplyRootCID = ptrStr("bafyR1")
	row.ReplyParentURI = ptrStr("at://did:plc:bob/social.craftsky.feed.post/r2")
	row.ReplyParentCID = ptrStr("bafyR2")
	row.QuoteURI = ptrStr("at://did:plc:carol/social.craftsky.feed.post/q1")
	row.QuoteCID = ptrStr("bafyQ1")

	resp := api.BuildPostResponse(row, syntax.Handle("alice.example"))
	if resp.Reply == nil || resp.Reply.Root.URI != *row.ReplyRootURI {
		t.Errorf("reply: %+v", resp.Reply)
	}
	if resp.Reply.Parent.URI != *row.ReplyParentURI {
		t.Errorf("reply.parent: %+v", resp.Reply.Parent)
	}
	if resp.Quote == nil || resp.Quote.URI != *row.QuoteURI {
		t.Errorf("quote: %+v", resp.Quote)
	}
}

func TestBuildPostResponse_WithAuthorDisplayFields(t *testing.T) {
	t.Parallel()
	row := baseRow()
	row.AuthorDisplayName = ptrStr("Alice")
	row.AuthorAvatarCID = ptrStr("bafyAvatar")

	resp := api.BuildPostResponse(row, syntax.Handle("alice.example"))
	if resp.Author.DisplayName == nil || *resp.Author.DisplayName != "Alice" {
		t.Errorf("displayName = %v", resp.Author.DisplayName)
	}
	if resp.Author.AvatarCID == nil || *resp.Author.AvatarCID != "bafyAvatar" {
		t.Errorf("avatarCID = %v", resp.Author.AvatarCID)
	}
}

func TestBuildPostResponse_FacetsPassThrough(t *testing.T) {
	t.Parallel()
	row := baseRow()
	row.Facets = json.RawMessage(`[{"index":{"byteStart":0,"byteEnd":5}}]`)
	resp := api.BuildPostResponse(row, syntax.Handle("alice.example"))
	if string(resp.Facets) != string(row.Facets) {
		t.Errorf("facets = %s", resp.Facets)
	}
}

func TestBuildPostResponse_OneSidedReplyPointer_DropsReply(t *testing.T) {
	t.Parallel()
	row := baseRow()
	// Only the root pointer is set — parent is nil. The lexicon requires
	// both, so the response must drop the reply rather than emit a
	// half-populated object.
	row.ReplyRootURI = ptrStr("at://did:plc:bob/social.craftsky.feed.post/r1")
	row.ReplyRootCID = ptrStr("bafyR1")

	resp := api.BuildPostResponse(row, syntax.Handle("alice.example"))
	if resp.Reply != nil {
		t.Errorf("expected nil reply when parent pointer is missing, got %+v", resp.Reply)
	}
}
