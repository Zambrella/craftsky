package api_test

import (
	"errors"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
)

func TestSavedPostCursorRoundTripAndCompatibility(t *testing.T) {
	savedAt := time.Date(2026, 7, 20, 12, 34, 56, 789, time.UTC)
	uri := "at://did:plc:bob/social.craftsky.feed.post/one"
	cursor, err := api.EncodeSavedPostCursor(api.SavedPostScopeFolder, "folder-opaque", api.SavedPostSortNewest, savedAt, uri)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	payload, err := envelope.DecodeCursor(cursor)
	if err != nil {
		t.Fatalf("decode raw payload: %v", err)
	}
	if _, ok := payload["ownerDid"]; ok {
		t.Fatalf("cursor encoded ownerDid: %#v", payload)
	}
	if payload["kind"] != "savedPost" || payload["scope"] != "folder" || payload["sort"] != "newest" {
		t.Fatalf("cursor payload = %#v", payload)
	}

	decoded, err := api.DecodeSavedPostCursor(cursor, api.SavedPostScopeFolder, "folder-opaque", api.SavedPostSortNewest)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !decoded.SavedAt.Equal(savedAt) || decoded.URI != uri {
		t.Fatalf("decoded = %+v", decoded)
	}

	for _, test := range []struct {
		name     string
		cursor   string
		scope    api.SavedPostScope
		folderID string
		sort     api.SavedPostSort
	}{
		{name: "malformed", cursor: "not-base64", scope: api.SavedPostScopeFolder, folderID: "folder-opaque", sort: api.SavedPostSortNewest},
		{name: "cross scope", cursor: cursor, scope: api.SavedPostScopeAll, sort: api.SavedPostSortNewest},
		{name: "cross folder", cursor: cursor, scope: api.SavedPostScopeFolder, folderID: "other-folder", sort: api.SavedPostSortNewest},
		{name: "cross sort", cursor: cursor, scope: api.SavedPostScopeFolder, folderID: "folder-opaque", sort: api.SavedPostSortOldest},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := api.DecodeSavedPostCursor(test.cursor, test.scope, test.folderID, test.sort)
			if !errors.Is(err, envelope.ErrInvalidCursor) {
				t.Fatalf("decode error = %v, want invalid cursor", err)
			}
		})
	}
}

func TestSavedPostCursorSupportsAllAndUnfiledScopes(t *testing.T) {
	for _, scope := range []api.SavedPostScope{api.SavedPostScopeAll, api.SavedPostScopeUnfiled} {
		cursor, err := api.EncodeSavedPostCursor(scope, "", api.SavedPostSortOldest, time.Date(2026, 7, 20, 1, 2, 3, 0, time.UTC), "at://did:plc:a/social.craftsky.feed.post/x")
		if err != nil {
			t.Fatalf("encode %s: %v", scope, err)
		}
		decoded, err := api.DecodeSavedPostCursor(cursor, scope, "", api.SavedPostSortOldest)
		if err != nil || decoded.URI == "" {
			t.Fatalf("decode %s = %+v, %v", scope, decoded, err)
		}
	}
}

func TestSavedPostCursorFolderDistinguishesDuplicateNamesByOpaqueID(t *testing.T) {
	ids := []string{"opaque-a", "opaque-b", "opaque-c"}
	for _, id := range ids {
		cursor, err := api.EncodeSavedPostFolderCursor("ideas", id)
		if err != nil {
			t.Fatalf("encode %s: %v", id, err)
		}
		decoded, err := api.DecodeSavedPostFolderCursor(cursor)
		if err != nil {
			t.Fatalf("decode %s: %v", id, err)
		}
		if decoded.FoldedName != "ideas" || decoded.FolderID != id {
			t.Fatalf("decoded = %+v, want ideas/%s", decoded, id)
		}
	}

	wrongKind, err := envelope.EncodeCursor(map[string]any{
		"kind":       "savedPost",
		"foldedName": "ideas",
		"folderId":   "opaque-a",
	})
	if err != nil {
		t.Fatalf("encode wrong kind: %v", err)
	}
	if _, err := api.DecodeSavedPostFolderCursor(wrongKind); !errors.Is(err, envelope.ErrInvalidCursor) {
		t.Fatalf("wrong-kind error = %v, want invalid cursor", err)
	}
}
