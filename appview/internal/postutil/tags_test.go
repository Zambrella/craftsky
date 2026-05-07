// appview/internal/postutil/tags_test.go
package postutil_test

import (
	"reflect"
	"testing"

	appbsky "github.com/bluesky-social/indigo/api/bsky"

	"social.craftsky/appview/internal/postutil"
)

func TestExtractTags_NilFacets(t *testing.T) {
	got := postutil.ExtractTags(nil)
	if !reflect.DeepEqual(got, []string{}) {
		t.Fatalf("want empty slice, got %#v", got)
	}
}

func TestExtractTags_LowercasesTrimsAndDedupes(t *testing.T) {
	facets := []*appbsky.RichtextFacet{
		{Features: []*appbsky.RichtextFacet_Features_Elem{
			{RichtextFacet_Tag: &appbsky.RichtextFacet_Tag{Tag: "  Knitting  "}},
		}},
		{Features: []*appbsky.RichtextFacet_Features_Elem{
			{RichtextFacet_Tag: &appbsky.RichtextFacet_Tag{Tag: "knitting"}},
			{RichtextFacet_Tag: &appbsky.RichtextFacet_Tag{Tag: "Shawl"}},
		}},
	}
	got := postutil.ExtractTags(facets)
	want := []string{"knitting", "shawl"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %#v, got %#v", want, got)
	}
}

func TestExtractTags_IgnoresNonTagFeaturesAndEmpty(t *testing.T) {
	facets := []*appbsky.RichtextFacet{
		{Features: []*appbsky.RichtextFacet_Features_Elem{
			{RichtextFacet_Mention: &appbsky.RichtextFacet_Mention{Did: "did:plc:abc"}},
			{RichtextFacet_Tag: &appbsky.RichtextFacet_Tag{Tag: ""}},
			{RichtextFacet_Tag: &appbsky.RichtextFacet_Tag{Tag: "   "}},
		}},
		nil,
	}
	got := postutil.ExtractTags(facets)
	if !reflect.DeepEqual(got, []string{}) {
		t.Fatalf("want empty slice, got %#v", got)
	}
}
