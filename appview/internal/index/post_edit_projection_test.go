package index_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
)

// IT-010 / IT-007: readers receive edit metadata belonging to the version that
// passed the durable projection and current-CID visibility path.
func TestPostEditProjectionExposesCurrentMetadata(t *testing.T) {
	f := newNotificationPipeline(t)
	event := notificationEvent(101, "did:plc:actor", "3aaaaaaaaaaa5", "")
	event.CID = "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"
	event.Live = true
	event.Record = json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"published edit","sponsored":false,"createdAt":"2019-01-01T00:00:00Z","editedAt":"2020-01-02T12:00:00Z"}`)
	f.ingest(event)
	f.drain()
	handler := api.GetPostHandler(api.NewPostStore(f.pool), postEditProjectionResolver{}, testLogger())
	request := httptest.NewRequest(http.MethodGet, "/v1/posts/did:plc:actor/3aaaaaaaaaaa5", nil)
	request.SetPathValue("did", "did:plc:actor")
	request.SetPathValue("rkey", "3aaaaaaaaaaa5")
	request = request.WithContext(middleware.WithDID(request.Context(), event.DID))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var wire map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if _, present := wire["editTimeSource"]; present {
		t.Fatal("edit timestamps must not expose provenance modes")
	}
	var result struct {
		Text                           string
		EditedAt                       *time.Time
		EditTimeSource, RecordRevision string
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	want := time.Date(2020, 1, 2, 12, 0, 0, 0, time.UTC)
	if result.Text != "published edit" || result.EditedAt == nil || !result.EditedAt.Equal(want) || result.RecordRevision != event.Rev.String() {
		t.Fatalf("current response lost edit metadata: %s", response.Body.String())
	}
}

// IT-007 / IT-010: an ahead-of-projection source cannot relabel the older body.
func TestPostEditMetadataStaysWithServingCIDWhileSourceIsAhead(t *testing.T) {
	f := newNotificationPipeline(t)
	event := notificationEvent(101, "did:plc:actor", "3aaaaaaaaaaa5", "")
	event.CID = "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"
	event.Record = json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"visible version","sponsored":false,"createdAt":"2019-01-01T00:00:00Z","editedAt":"2020-01-02T12:00:00Z"}`)
	f.ingest(event)
	f.drain()
	event.ID = 102
	event.Action = "update"
	event.CID = "bafyreif7dhriwzuzrl3gsrxjo3mzxt5jdls7yo3jwoashfx74og6n3ck4m"
	event.Rev = "3aaaaaaaaaaa3"
	event.Record = json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"pending version","sponsored":false,"createdAt":"2019-01-01T00:00:00Z","editedAt":"2021-01-02T12:00:00Z"}`)
	f.ingest(event)
	read := func() struct {
		Text, RecordRevision string
		EditedAt             *time.Time
	} {
		t.Helper()
		handler := api.GetPostHandler(api.NewPostStore(f.pool), postEditProjectionResolver{}, testLogger())
		request := httptest.NewRequest(http.MethodGet, "/v1/posts/did:plc:actor/3aaaaaaaaaaa5", nil)
		request.SetPathValue("did", "did:plc:actor")
		request.SetPathValue("rkey", "3aaaaaaaaaaa5")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request.WithContext(middleware.WithDID(request.Context(), event.DID)))
		if response.Code != 200 {
			t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
		var result struct {
			Text, RecordRevision string
			EditedAt             *time.Time
		}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	before := read()
	if before.Text != "visible version" || before.RecordRevision != "3aaaaaaaaaaa2" || before.EditedAt == nil || !before.EditedAt.Equal(time.Date(2020, 1, 2, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("source metadata leaked onto older body: %+v", before)
	}
	f.drain()
	after := read()
	if after.Text != "pending version" || after.RecordRevision != "3aaaaaaaaaaa3" || after.EditedAt == nil || !after.EditedAt.Equal(time.Date(2021, 1, 2, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("new serving version lost its metadata: %+v", after)
	}
}

// IT-011 / FR-008: a quote's stale strong-reference CID does not prevent current
// original content/metadata hydration or change the quoting author's own post.
func TestPostEditQuotePreviewExposesLatestOriginalMetadata(t *testing.T) {
	f := newNotificationPipeline(t)
	target := notificationEvent(101, "did:plc:actor", "3aaaaaaaaaaa5", "")
	target.CID = "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"
	target.Record = json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"latest original","sponsored":false,"createdAt":"2019-01-01T00:00:00Z","editedAt":"2020-01-02T12:00:00Z"}`)
	f.ingest(target)
	quote := notificationEvent(102, "did:plc:recipient", "3aaaaaaaaaaa6", quoteExtra(target.URI.String()))
	quote.CID = "bafyreif7dhriwzuzrl3gsrxjo3mzxt5jdls7yo3jwoashfx74og6n3ck4m"
	f.ingest(quote)
	f.drain()
	handler := api.GetPostHandler(api.NewPostStore(f.pool), postEditProjectionResolver{}, testLogger())
	request := httptest.NewRequest(http.MethodGet, "/v1/posts/did:plc:recipient/3aaaaaaaaaaa6", nil)
	request.SetPathValue("did", "did:plc:recipient")
	request.SetPathValue("rkey", "3aaaaaaaaaaa6")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(middleware.WithDID(request.Context(), quote.DID)))
	if response.Code != 200 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var result struct {
		Text      string
		EditedAt  *time.Time
		QuoteView struct {
			State string
			Post  struct {
				Text, EditTimeSource, RecordRevision string
				EditedAt                             *time.Time
			}
		}
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	preview := result.QuoteView.Post
	want := time.Date(2020, 1, 2, 12, 0, 0, 0, time.UTC)
	if result.Text != "hello recipient" || result.EditedAt != nil || result.QuoteView.State != "visible" || preview.Text != "latest original" || preview.EditedAt == nil || !preview.EditedAt.Equal(want) || preview.RecordRevision != target.Rev.String() {
		t.Fatalf("quote metadata/identity mismatch: %s", response.Body.String())
	}
}

type postEditProjectionResolver struct{}

func (postEditProjectionResolver) ResolveHandle(_ context.Context, did syntax.DID) (syntax.Handle, error) {
	if did == "did:plc:recipient" {
		return "recipient.example", nil
	}
	return "actor.example", nil
}

func (postEditProjectionResolver) ResolveDID(_ context.Context, handle syntax.Handle) (syntax.DID, error) {
	if handle == "recipient.example" {
		return "did:plc:recipient", nil
	}
	return "did:plc:actor", nil
}

// IT-014: malformed declarations remain subject to strict Lexicon validation.
func TestPostEditMalformedExternalDeclarationIsNotProjected(t *testing.T) {
	for _, declaration := range []string{`"not-a-timestamp"`, `42`} {
		t.Run(declaration, func(t *testing.T) {
			f := newNotificationPipeline(t)
			event := notificationEvent(950, "did:plc:actor", "3aaaaaaaaaaa5", "")
			f.ingest(event)
			f.drain()
			event.ID++
			event.Action = "update"
			event.CID = "cid-malformed-edit-time"
			event.Rev = "3aaaaaaaaaaa3"
			event.Record = json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"external correction","sponsored":false,"createdAt":"2026-05-04T12:00:00Z","editedAt":` + declaration + `}`)
			f.ingest(event)
			f.drain()
			var structural, semantic string
			if err := f.pool.QueryRow(context.Background(), `SELECT structural_validation_status,semantic_validation_status FROM tap_source_records WHERE uri=$1`, event.URI).Scan(&structural, &semantic); err != nil {
				t.Fatal(err)
			}
			if structural != "invalid" || semantic != "invalid" {
				t.Fatalf("malformed declaration accepted: validation=%s/%s", structural, semantic)
			}
			f.assertCount(`SELECT count(*) FROM craftsky_posts`, 0)
		})
	}
}

// IT-014: external edit timestamps are declarations, never inferred times.
func TestPostEditExternalTimestampUsesOnlyCurrentDeclaration(t *testing.T) {
	for _, extra := range []string{"", `,"editedAt":"2099-01-01T00:00:00Z"`} {
		t.Run(extra, func(t *testing.T) {
			f := newNotificationPipeline(t)
			event := notificationEvent(960, "did:plc:actor", "3aaaaaaaaaaa5", "")
			f.ingest(event)
			f.drain()
			event.ID++
			event.Action = "update"
			event.CID = "cid-external-edited"
			event.Rev = "3aaaaaaaaaaa3"
			event.Record = json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"external correction","sponsored":false,"createdAt":"2026-05-04T12:00:00Z"` + extra + `}`)
			f.ingest(event)
			f.drain()
			var text string
			var at *time.Time
			if err := f.pool.QueryRow(context.Background(), `SELECT text,edited_at FROM craftsky_posts WHERE uri=$1`, event.URI).Scan(&text, &at); err != nil {
				t.Fatal(err)
			}
			if text != "external correction" || (extra == "" && at != nil) || (extra != "" && (at == nil || at.Year() != 2099)) {
				t.Fatalf("declaration=%q text=%q timestamp=%v", extra, text, at)
			}
		})
	}
}
