package api

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/profilelinks"
	"social.craftsky/appview/internal/testdb"
)

func TestLinkEditsPreserveUnknownFieldsAndReconcileAcceptedWrites(t *testing.T) {
	owner := syntax.DID("did:plc:owner")
	before := json.RawMessage(`{"$type":"app.bsky.actor.link","url":"https://example.com/","title":"Before","createdAt":"2026-10-27T12:00:00Z","future":{"n":9007199254740993},"labels":{"$type":"com.atproto.label.defs#selfLabels","values":[{"val":"porn"}]}}`)
	beforeCID, _ := profilelinks.RecordCID(before)
	input := ProfileLinkInput{Rkey: "3m5abc2345672", CID: beforeCID, URL: "https://example.com/", Title: "After", CreatedAt: "2026-10-27T12:00:00Z"}
	saved, refs, err := buildProfileLinkRecords(owner, []ProfileLinkInput{input}, []pdscommands.AuthoritativeRecord{{CID: beforeCID, Record: before}})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]json.RawMessage
	if json.Unmarshal(saved[0], &decoded) != nil || string(decoded["future"]) != `{"n":9007199254740993}` || len(decoded["labels"]) == 0 {
		t.Fatal("extension or self labels lost")
	}
	// Lost-response retries use the original expected CID but converge on the
	// already-written content rather than rejecting a successfully applied save.
	if _, _, err := buildProfileLinkRecords(owner, []ProfileLinkInput{input}, []pdscommands.AuthoritativeRecord{{CID: refs[0].CID, Record: saved[0]}}); err != nil {
		t.Fatal(err)
	}
	input.Title = "Conflicting edit"
	if _, _, err := buildProfileLinkRecords(owner, []ProfileLinkInput{input}, []pdscommands.AuthoritativeRecord{{CID: refs[0].CID, Record: saved[0]}}); !errors.Is(err, pdscommands.ErrRepositoryConflict) {
		t.Fatal("overwrote a newer edit")
	}
}

func TestProfileLinksOrderVisibilityAndOwnerEditor(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:links")
	a := "at://did:plc:links/app.bsky.actor.link/3m5abc2345672"
	b := "at://did:plc:links/app.bsky.actor.link/3m5abc2345673"
	c := "at://did:plc:links/app.bsky.actor.link/3m5abc2345674"
	refs, _ := json.Marshal([]profilelinks.StrongRef{{URI: syntax.ATURI(b), CID: "cid-b"}, {URI: syntax.ATURI(a), CID: "cid-a"}, {URI: syntax.ATURI(c), CID: "cid-c"}})
	if _, err := pool.Exec(ctx, `INSERT INTO bluesky_profiles(did,record_cid,links) VALUES($1,'profile-cid',$2)`, owner, refs); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ uri, url, labels string }{{a, "https://example.com/a", ""}, {b, "https://example.com/b", ""}, {c, "https://example.com/c", `,"labels":{"$type":"com.atproto.label.defs#selfLabels","values":[{"val":"porn"}]}`}} {
		raw := `{"$type":"app.bsky.actor.link","url":"` + tc.url + `","createdAt":"2026-10-27T12:00:00Z"` + tc.labels + `}`
		if _, err := pool.Exec(ctx, `INSERT INTO bluesky_profile_links(uri,did,cid,record)VALUES($1,$2,'record-cid',$3)`, tc.uri, owner, raw); err != nil {
			t.Fatal(err)
		}
	}
	store := NewProfileStore(pool)
	public, editable, err := store.readProfileLinks(ctx, owner, owner.String())
	if err != nil {
		t.Fatal(err)
	}
	if len(public) != 2 || public[0].URI.String() != b || public[1].URI.String() != a || len(editable) != 3 {
		t.Fatalf("public=%+v editable=%+v", public, editable)
	}
	visitor, editor, err := store.readProfileLinks(ctx, owner, "did:plc:visitor")
	if err != nil || len(visitor) != 2 || len(editor) != 0 {
		t.Fatalf("visitor editor leak: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state)VALUES($1,'profile_link','record-cid','blocked')`, b); err != nil {
		t.Fatal(err)
	}
	public, editable, err = store.readProfileLinks(ctx, owner, owner.String())
	if err != nil || len(public) != 1 || len(editable) != 3 {
		t.Fatalf("pending scan lost editor data: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO moderation_outputs(id,source_did,subject_type,subject_did,subject_collection,subject_rkey,subject_uri,value,action)VALUES('link-warning','did:plc:moderator','profileLink',$1,'app.bsky.actor.link','3m5abc2345672',$2,'warn','apply')`, owner, a); err != nil {
		t.Fatal(err)
	}
	public, editable, err = store.readProfileLinks(ctx, owner, owner.String())
	if err != nil || len(public) != 0 || len(editable) != 3 {
		t.Fatalf("warning must hide the pill but preserve editing: public=%+v editable=%+v err=%v", public, editable, err)
	}
}

func TestConsoleProfileLinkPreviewHonorsCapturedVersionAndOmitsIcon(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:link-preview")
	uri := "at://" + owner.String() + "/app.bsky.actor.link/3m5abc2345672"
	raw := `{"$type":"app.bsky.actor.link","url":"https://example.com/shop","title":"Shop","createdAt":"2026-10-27T12:00:00Z"}`
	if _, err := pool.Exec(ctx, `INSERT INTO bluesky_profile_links(uri,did,cid,record)VALUES($1,$2,'current-cid',$3)`, uri, owner, raw); err != nil {
		t.Fatal(err)
	}
	view, err := readConsoleSubject(ctx, pool, "profileLink", owner, uri, "current-cid", nil)
	if err != nil || view.State != "available" || view.Title != "Shop" || view.Text != "https://example.com/shop" || view.MediaAvailable || len(view.Images) > 0 {
		t.Fatalf("view=%+v err=%v", view, err)
	}
	changed, err := readConsoleSubject(ctx, pool, "profileLink", owner, uri, "older-cid", nil)
	if err != nil || changed.State != "unavailable" || changed.Version.Status != "changed" || changed.Text != "" {
		t.Fatalf("changed view=%+v err=%v", changed, err)
	}
}
