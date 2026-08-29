package accountdeletion

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestCraftskyRecordCollections(t *testing.T) {
	t.Parallel()

	want := recordCollectionsFromLexicons(t, "../../../lexicon/social/craftsky")
	got := CraftskyRecordCollections()

	gotSorted := append([]syntax.NSID(nil), got...)
	sort.Slice(gotSorted, func(i, j int) bool { return gotSorted[i] < gotSorted[j] })
	if !reflect.DeepEqual(gotSorted, want) {
		t.Fatalf("CraftskyRecordCollections() = %v, want exact Lexicon record set %v", gotSorted, want)
	}

	wantTail := []syntax.NSID{
		"social.craftsky.business.event",
		"social.craftsky.business.profile",
		"social.craftsky.actor.profile",
	}
	if len(got) < len(wantTail) || !reflect.DeepEqual(got[len(got)-len(wantTail):], wantTail) {
		t.Fatalf("CraftskyRecordCollections() tail = %v, want events, declaration, membership", got)
	}

	seen := make(map[syntax.NSID]struct{}, len(got))
	for _, collection := range got {
		if _, exists := seen[collection]; exists {
			t.Fatalf("CraftskyRecordCollections() contains duplicate %q", collection)
		}
		seen[collection] = struct{}{}
	}
}

func recordCollectionsFromLexicons(t *testing.T, root string) []syntax.NSID {
	t.Helper()

	var collections []syntax.NSID
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var lexicon struct {
			ID   syntax.NSID `json:"id"`
			Defs map[string]struct {
				Type string `json:"type"`
			} `json:"defs"`
		}
		if err := json.Unmarshal(raw, &lexicon); err != nil {
			return err
		}
		if lexicon.Defs["main"].Type == "record" {
			collections = append(collections, lexicon.ID)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk CraftSky Lexicons: %v", err)
	}

	sort.Slice(collections, func(i, j int) bool { return collections[i] < collections[j] })
	return collections
}
