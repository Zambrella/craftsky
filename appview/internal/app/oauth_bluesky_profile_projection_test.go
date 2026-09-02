package app

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

type recordingBlueskyProfileEventHandler struct {
	events []tap.Event
}

func TestOAuthBlueskyProfileProjectionIsIdempotentWithCanonicalReplay(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, `
		CREATE TABLE bluesky_profiles (
			did TEXT PRIMARY KEY,
			display_name TEXT,
			description TEXT,
			avatar_cid TEXT,
			avatar_mime TEXT,
			banner_cid TEXT,
			banner_mime TEXT,
			record_cid TEXT NOT NULL,
			indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);`)
	handler := index.NewBlueskyProfile(pool)
	projector := oauthBlueskyProfileProjection{handler: handler}
	did := syntax.DID("did:plc:oauth-replay")
	cid := syntax.CID("bafyreioauthreplay")
	record := map[string]any{
		"displayName": "Alice",
		"description": "Textile maker",
		"avatar": map[string]any{
			"ref": map[string]any{"$link": "bafkavatar"}, "mimeType": "image/jpeg",
		},
		"banner": map[string]any{
			"ref": map[string]any{"$link": "bafkbanner"}, "mimeType": "image/png",
		},
	}

	if err := projector.ProjectBlueskyProfile(context.Background(), did, cid, record); err != nil {
		t.Fatalf("direct projection: %v", err)
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := handler.Handle(context.Background(), tap.Event{
		URI: syntax.ATURI("at://" + did.String() + "/app.bsky.actor.profile/self"),
		CID: cid, DID: did, Collection: "app.bsky.actor.profile", Rkey: "self",
		Action: "create", Record: raw,
	}); err != nil {
		t.Fatalf("Tap/indexer replay: %v", err)
	}

	var count int
	var displayName, description, avatarCID, bannerCID, recordCID string
	err = pool.QueryRow(context.Background(), `
		SELECT count(*) OVER (), display_name, description, avatar_cid, banner_cid, record_cid
		FROM bluesky_profiles WHERE did = $1`, did).
		Scan(&count, &displayName, &description, &avatarCID, &bannerCID, &recordCID)
	if err != nil {
		t.Fatalf("read projection: %v", err)
	}
	if count != 1 || displayName != "Alice" || description != "Textile maker" ||
		avatarCID != "bafkavatar" || bannerCID != "bafkbanner" || recordCID != cid.String() {
		t.Fatalf("projection = count:%d display:%q description:%q avatar:%q banner:%q cid:%q",
			count, displayName, description, avatarCID, bannerCID, recordCID)
	}
}

func (h *recordingBlueskyProfileEventHandler) Handle(_ context.Context, event tap.Event) error {
	h.events = append(h.events, event)
	return nil
}

func TestOAuthBlueskyProfileProjectionUsesCanonicalTapEvent(t *testing.T) {
	t.Parallel()
	handler := &recordingBlueskyProfileEventHandler{}
	projector := oauthBlueskyProfileProjection{handler: handler}
	did := syntax.DID("did:plc:oauth-projection")
	cid := syntax.CID("bafyreioauthprofile")
	record := map[string]any{
		"displayName": "Alice",
		"description": "Textile maker",
		"avatar": map[string]any{
			"ref":      map[string]any{"$link": "bafkavatar"},
			"mimeType": "image/jpeg",
		},
	}

	if err := projector.ProjectBlueskyProfile(context.Background(), did, cid, record); err != nil {
		t.Fatalf("ProjectBlueskyProfile: %v", err)
	}
	if len(handler.events) != 1 {
		t.Fatalf("events = %d, want 1", len(handler.events))
	}
	event := handler.events[0]
	if event.DID != did || event.CID != cid || event.Collection != "app.bsky.actor.profile" ||
		event.Rkey != "self" || event.Action != "create" ||
		event.URI != syntax.ATURI("at://did:plc:oauth-projection/app.bsky.actor.profile/self") {
		t.Fatalf("canonical event = %+v", event)
	}
	var projected map[string]any
	if err := json.Unmarshal(event.Record, &projected); err != nil {
		t.Fatalf("unmarshal projected record: %v", err)
	}
	if projected["displayName"] != "Alice" || projected["description"] != "Textile maker" {
		t.Fatalf("projected record = %#v", projected)
	}
}

func TestOAuthCraftskyProfileProjectionUsesCanonicalTapEvent(t *testing.T) {
	t.Parallel()
	handler := &recordingBlueskyProfileEventHandler{}
	projector := oauthCraftskyProfileProjection{handler: handler}
	did := syntax.DID("did:plc:oauth-craftsky-projection")
	cid := syntax.CID("bafyreioauthcraftskyprofile")
	record := map[string]any{
		"$type":  "social.craftsky.actor.profile",
		"crafts": []string{"knitting"},
	}

	if err := projector.ProjectCraftskyProfile(context.Background(), did, cid, record); err != nil {
		t.Fatalf("ProjectCraftskyProfile: %v", err)
	}
	if len(handler.events) != 1 {
		t.Fatalf("events = %d, want 1", len(handler.events))
	}
	event := handler.events[0]
	if event.DID != did || event.CID != cid || event.Collection != "social.craftsky.actor.profile" ||
		event.Rkey != "self" || event.Action != "create" ||
		event.URI != syntax.ATURI("at://did:plc:oauth-craftsky-projection/social.craftsky.actor.profile/self") {
		t.Fatalf("canonical event = %+v", event)
	}
}
