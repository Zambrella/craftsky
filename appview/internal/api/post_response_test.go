// appview/internal/api/post_response_test.go
package api_test

import (
	"encoding/json"
	"net/url"
	"strings"
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
	if resp.LikeCount != 0 || resp.RepostCount != 0 || resp.ReplyCount != 0 || resp.ViewerHasLiked || resp.ViewerHasReposted || resp.ViewerHasReplied {
		t.Errorf("engagement defaults = %+v", resp)
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

func TestBuildPostResponse_JSONIncludesEngagementFields(t *testing.T) {
	t.Parallel()
	resp := api.BuildPostResponse(baseRow(), syntax.Handle("alice.example"))
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(data)
	for _, key := range []string{`"likeCount":0`, `"repostCount":0`, `"replyCount":0`, `"viewerHasLiked":false`, `"viewerHasReposted":false`, `"viewerHasReplied":false`} {
		if !strings.Contains(got, key) {
			t.Fatalf("missing %s in %s", key, got)
		}
	}
}

func TestBuildPostResponse_IncludesProjectForProjectRows(t *testing.T) {
	t.Parallel()
	title := "Hitchhiker Shawl"
	row := baseRow()
	row.Project = &api.Project{
		Common: api.ProjectCommon{
			CraftType: "social.craftsky.feed.defs#knitting",
			Title:     &title,
		},
		Details: json.RawMessage(`{"$type":"social.craftsky.project.knitting#details","projectType":"shawl"}`),
	}

	resp := api.BuildPostResponse(row, syntax.Handle("alice.example"))
	if resp.Project == nil || resp.Project.Common.Title == nil || *resp.Project.Common.Title != title {
		t.Fatalf("project = %+v", resp.Project)
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), `"project"`) || !strings.Contains(string(data), `"craftType"`) {
		t.Fatalf("project/camelCase fields missing from %s", data)
	}
	if strings.Contains(string(data), "craft_type") {
		t.Fatalf("snake_case project field leaked in %s", data)
	}
}

func TestBuildPostResponse_OmitsProjectForGeneralRows(t *testing.T) {
	t.Parallel()
	resp := api.BuildPostResponse(baseRow(), syntax.Handle("alice.example"))
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), `"project"`) {
		t.Fatalf("general post response should omit project: %s", data)
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

func TestBuildPostResponse_ImageURLsKnownAndUnknownMIME(t *testing.T) {
	t.Parallel()
	row := baseRow()
	row.Images = json.RawMessage(`[
		{"cid":"bafkjpeg","mime":"image/jpeg","size":123,"alt":"jpg"},
		{"cid":"bafktiff","mime":"image/tiff","size":456,"alt":"tiff"}
	]`)

	resp := api.BuildPostResponse(row, syntax.Handle("alice.example"))
	if len(resp.Images) != 2 {
		t.Fatalf("len(images) = %d, want 2", len(resp.Images))
	}
	if resp.Images[0].Thumb == "" || resp.Images[0].Fullsize == "" {
		t.Fatalf("known mime urls missing: %+v", resp.Images[0])
	}
	if resp.Images[1].Thumb != "" || resp.Images[1].Fullsize != "" {
		t.Fatalf("unknown mime urls must be omitted: %+v", resp.Images[1])
	}
}

func TestBuildPostResponse_ImageMetadataIncludesAspectRatioAndSize(t *testing.T) {
	t.Parallel()
	row := baseRow()
	row.Images = json.RawMessage(`[
		{"cid":"bafkimage","mime":"image/jpeg","size":253496,"alt":"project photo","aspectRatio":{"width":919,"height":2000}}
	]`)

	resp := api.BuildPostResponse(row, syntax.Handle("alice.example"))
	if len(resp.Images) != 1 {
		t.Fatalf("len(images) = %d, want 1", len(resp.Images))
	}
	img := resp.Images[0]
	if img.CID != "bafkimage" || img.MIME != "image/jpeg" || img.Size != 253496 || img.Alt != "project photo" {
		t.Fatalf("image metadata = %+v", img)
	}
	if img.AspectRatio == nil || img.AspectRatio.Width != 919 || img.AspectRatio.Height != 2000 {
		t.Fatalf("aspectRatio = %+v", img.AspectRatio)
	}
}

func TestBuildPostResponse_DevMediaCIDUsesLocalDevMediaURL(t *testing.T) {
	t.Setenv("CRAFTSKY_DEV_MEDIA_BASE_URL", "http://example.test/media/")
	row := baseRow()
	row.Images = json.RawMessage(`[
		{"cid":"devmedia:knit-cardigan-moss","mime":"image/jpeg","alt":"project photo"}
	]`)

	resp := api.BuildPostResponse(row, syntax.Handle("alice.example"))
	if len(resp.Images) != 1 {
		t.Fatalf("len(images) = %d, want 1", len(resp.Images))
	}
	got := resp.Images[0].Fullsize
	if got != "http://example.test/media/knit-cardigan-moss" || resp.Images[0].Thumb != got {
		t.Fatalf("dev media urls = thumb %q fullsize %q", resp.Images[0].Thumb, got)
	}
	if _, err := url.ParseRequestURI(got); err != nil {
		t.Fatalf("dev media URL is not parseable: %v", err)
	}
}

func TestBuildPostResponse_ModerationWarningMetadataIsGeneric(t *testing.T) {
	t.Parallel()
	row := baseRow()
	row.ModerationWarningKind = ptrStr("post")

	resp := api.BuildPostResponse(row, syntax.Handle("alice.example"))
	if resp.Moderation == nil || resp.Moderation.WarningKind != "post" {
		t.Fatalf("moderation = %+v, want post warning", resp.Moderation)
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	moderation, ok := body["moderation"].(map[string]any)
	if !ok {
		t.Fatalf("moderation missing or wrong type in %s", data)
	}
	if len(moderation) != 1 || moderation["warningKind"] != "post" {
		t.Fatalf("moderation payload = %#v, want only warningKind", moderation)
	}
	for _, forbidden := range []string{"raw unsafe reason fixture", "internalReason", "sourceDid", "outputId", "reportCount"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("moderation response leaked %q in %s", forbidden, data)
		}
	}
}

func TestBuildPostResponse_OmitsModerationWhenUnwarned(t *testing.T) {
	t.Parallel()
	resp := api.BuildPostResponse(baseRow(), syntax.Handle("alice.example"))
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), "moderation") {
		t.Fatalf("unwarned post must omit moderation metadata: %s", data)
	}
}
