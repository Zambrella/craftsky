package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
)

func TestSavedPostStatusFromMutationOutcome(t *testing.T) {
	for _, test := range []struct {
		name   string
		result api.SaveMutationResult
		want   int
	}{
		{name: "created", result: api.SaveMutationResult{Created: true}, want: http.StatusCreated},
		{name: "unchanged existing", result: api.SaveMutationResult{}, want: http.StatusOK},
		{name: "moved existing", result: api.SaveMutationResult{Changed: true}, want: http.StatusOK},
		{name: "unfiled existing", result: api.SaveMutationResult{Changed: true}, want: http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := api.SaveMutationHTTPStatus(test.result); got != test.want {
				t.Fatalf("status = %d, want %d", got, test.want)
			}
		})
	}
}

func TestSavedPostResponseSerializesExactPostAndViewerState(t *testing.T) {
	folderID := "opaque-folder"
	savedAt := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	item := api.SavedPostItem{
		Post: &api.PostResponse{
			URI:                 "at://did:plc:bob/social.craftsky.feed.post/reply",
			ViewerHasSaved:      true,
			ViewerSavedFolderID: &folderID,
			Reply: &api.ResponseReply{
				Root:   api.ResponseStrongRef{URI: "at://did:plc:bob/social.craftsky.feed.post/root", CID: "root-cid"},
				Parent: api.ResponseStrongRef{URI: "at://did:plc:bob/social.craftsky.feed.post/parent", CID: "parent-cid"},
			},
		},
		SavedAt:  savedAt,
		FolderID: &folderID,
	}
	raw, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal saved item: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode saved item: %v", err)
	}
	if decoded["savedAt"] != savedAt.Format(time.RFC3339) || decoded["folderId"] != folderID {
		t.Fatalf("save metadata = %#v", decoded)
	}
	post, ok := decoded["post"].(map[string]any)
	if !ok {
		t.Fatalf("post shape = %#v", decoded["post"])
	}
	if post["uri"] != item.Post.URI || post["viewerHasSaved"] != true || post["viewerSavedFolderId"] != folderID {
		t.Fatalf("post viewer state = %#v", post)
	}
	if _, leaked := post["viewerSavedFolderName"]; leaked {
		t.Fatalf("post leaked folder name: %#v", post)
	}
	reply, ok := post["reply"].(map[string]any)
	if !ok || reply["root"].(map[string]any)["uri"] != item.Post.Reply.Root.URI || reply["parent"].(map[string]any)["uri"] != item.Post.Reply.Parent.URI {
		t.Fatalf("reply context = %#v", post["reply"])
	}

	unsavedRaw, err := json.Marshal(api.PostResponse{})
	if err != nil {
		t.Fatalf("marshal unsaved post: %v", err)
	}
	var unsaved map[string]any
	if err := json.Unmarshal(unsavedRaw, &unsaved); err != nil {
		t.Fatalf("decode unsaved post: %v", err)
	}
	if unsaved["viewerHasSaved"] != false {
		t.Fatalf("unsaved viewerHasSaved = %#v", unsaved["viewerHasSaved"])
	}
	if value, present := unsaved["viewerSavedFolderId"]; !present || value != nil {
		t.Fatalf("unsaved viewerSavedFolderId = %#v, present=%v", value, present)
	}
}
