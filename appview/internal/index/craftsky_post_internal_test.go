package index

import (
	"encoding/json"
	"testing"

	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/ipfs/go-cid"

	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
)

func TestFlattenImages_IncludesSizeAndAspectRatio(t *testing.T) {
	t.Parallel()
	refCID, err := cid.Parse("bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a")
	if err != nil {
		t.Fatalf("parse cid: %v", err)
	}
	flat := flattenImages([]*craftskylex.FeedPost_Image{
		{
			Image: &lexutil.LexBlob{Ref: lexutil.LexLink(refCID), MimeType: "image/jpeg", Size: 253496},
			Alt:   strPtr("project photo"),
			AspectRatio: &craftskylex.FeedPost_AspectRatio{
				Width:  919,
				Height: 2000,
			},
		},
	})
	if len(flat) != 1 {
		t.Fatalf("len(flat) = %d, want 1", len(flat))
	}
	if got := flat[0]["cid"]; got != "bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a" {
		t.Fatalf("cid = %v", got)
	}
	if got := flat[0]["mime"]; got != "image/jpeg" {
		t.Fatalf("mime = %v", got)
	}
	if got := flat[0]["alt"]; got != "project photo" {
		t.Fatalf("alt = %v", got)
	}
	if got := any(flat[0]["size"]); got == nil {
		t.Fatalf("size missing in flattened image: %+v", flat[0])
	}
	aspectRaw := any(flat[0]["aspectRatio"])
	aspect, ok := aspectRaw.(map[string]any)
	if !ok {
		t.Fatalf("aspectRatio = %T %v, want map", aspectRaw, aspectRaw)
	}
	if aspect["width"] != int64(919) || aspect["height"] != int64(2000) {
		t.Fatalf("aspectRatio = %+v", aspect)
	}
}

func TestExtractProjectForIndex_ProjectnessRequiresCommonCraftType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "no project", raw: `{"text":"general"}`, want: false},
		{name: "empty project", raw: `{"project":{}}`, want: false},
		{name: "common without craft type", raw: `{"project":{"common":{}}}`, want: false},
		{name: "common with craft type", raw: `{"project":{"common":{"craftType":"social.craftsky.feed.defs#knitting"}}}`, want: true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			project, err := extractProjectForIndex(json.RawMessage(tc.raw))
			if err != nil {
				t.Fatalf("extractProjectForIndex: %v", err)
			}
			got := project != nil
			if got != tc.want {
				t.Fatalf("project present = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestExtractProjectForIndex_PreservesKnownAndUnknownDetails(t *testing.T) {
	t.Parallel()

	known, err := extractProjectForIndex(json.RawMessage(`{
		"project": {
			"common": {"craftType":"social.craftsky.feed.defs#knitting", "tags":[" FairIsle "]},
			"details": {
				"$type":"social.craftsky.project.knitting#details",
				"projectType":"sweater",
				"needleSizeMm":"4.5",
				"gauge":{"stitches":22,"measurement":10,"unit":"cm"}
			}
		}
	}`))
	if err != nil {
		t.Fatalf("extract known: %v", err)
	}
	if known == nil || known.Details.Type != "social.craftsky.project.knitting#details" {
		t.Fatalf("known details type = %+v", known)
	}
	if got := jsonString(known.Details.Map, "projectType"); got != "sweater" {
		t.Fatalf("projectType = %v, want sweater", got)
	}
	if got := jsonRaw(known.Details.Map, "gauge"); got == nil {
		t.Fatalf("gauge raw missing")
	}

	unknown, err := extractProjectForIndex(json.RawMessage(`{
		"project": {
			"common": {"craftType":"social.craftsky.feed.defs#future"},
			"details": {"$type":"social.craftsky.project.future#details", "newField":"kept"}
		}
	}`))
	if err != nil {
		t.Fatalf("extract unknown: %v", err)
	}
	if unknown == nil || unknown.Details.Type != "social.craftsky.project.future#details" || len(unknown.RawDetails) == 0 {
		t.Fatalf("unknown details not preserved: %+v", unknown)
	}
}

func TestFlattenImages_DefaultsMissingAltToEmptyString(t *testing.T) {
	t.Parallel()
	refCID, err := cid.Parse("bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a")
	if err != nil {
		t.Fatalf("parse cid: %v", err)
	}
	flat := flattenImages([]*craftskylex.FeedPost_Image{
		{
			Image: &lexutil.LexBlob{Ref: lexutil.LexLink(refCID), MimeType: "image/jpeg", Size: 253496},
		},
	})
	if len(flat) != 1 {
		t.Fatalf("len(flat) = %d, want 1", len(flat))
	}
	if got := flat[0]["alt"]; got != "" {
		t.Fatalf("alt = %v, want empty string", got)
	}
}

func strPtr(s string) *string { return &s }
