package index

import (
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
