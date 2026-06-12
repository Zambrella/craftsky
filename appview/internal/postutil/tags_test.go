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

func TestExtractTagsForText_ValidatesByteRanges(t *testing.T) {
	t.Parallel()

	facets := []*appbsky.RichtextFacet{
		{
			Index: &appbsky.RichtextFacet_ByteSlice{ByteStart: 3, ByteEnd: 7},
			Features: []*appbsky.RichtextFacet_Features_Elem{
				{RichtextFacet_Tag: &appbsky.RichtextFacet_Tag{Tag: "Tag"}},
			},
		},
		{
			Index: &appbsky.RichtextFacet_ByteSlice{ByteStart: 0, ByteEnd: 99},
			Features: []*appbsky.RichtextFacet_Features_Elem{
				{RichtextFacet_Tag: &appbsky.RichtextFacet_Tag{Tag: "out-of-range"}},
			},
		},
		{
			Features: []*appbsky.RichtextFacet_Features_Elem{
				{RichtextFacet_Tag: &appbsky.RichtextFacet_Tag{Tag: "missing-index"}},
			},
		},
	}

	got := postutil.ExtractTagsForText("hi #Tag", facets)
	want := []string{"tag"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractTagsForText = %#v, want %#v", got, want)
	}

	emptyText := postutil.ExtractTagsForText("", facets)
	if !reflect.DeepEqual(emptyText, []string{}) {
		t.Fatalf("ExtractTagsForText empty text = %#v, want empty slice", emptyText)
	}
}

func TestMergeTags_LowercasesTrimsDedupesAndPreservesFirstSeenOrder(t *testing.T) {
	t.Parallel()

	got := postutil.MergeTags(
		[]string{"FairIsle", "wip"},
		[]string{" fairisle ", "WIP", "linen", ""},
	)
	want := []string{"fairisle", "wip", "linen"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MergeTags = %#v, want %#v", got, want)
	}
}

func TestMergeTags_ReturnsNonNilEmptySlice(t *testing.T) {
	t.Parallel()

	got := postutil.MergeTags(nil, []string{" ", ""})
	if !reflect.DeepEqual(got, []string{}) {
		t.Fatalf("MergeTags = %#v, want empty slice", got)
	}
}

func TestExtractMentionDIDs_TrimsDedupesAndIgnoresNonMentions(t *testing.T) {
	t.Parallel()

	facets := []*appbsky.RichtextFacet{
		{Features: []*appbsky.RichtextFacet_Features_Elem{
			{RichtextFacet_Mention: &appbsky.RichtextFacet_Mention{Did: " did:plc:alice "}},
			{RichtextFacet_Tag: &appbsky.RichtextFacet_Tag{Tag: "ignored"}},
		}},
		{Features: []*appbsky.RichtextFacet_Features_Elem{
			{RichtextFacet_Mention: &appbsky.RichtextFacet_Mention{Did: "did:plc:alice"}},
			{RichtextFacet_Mention: &appbsky.RichtextFacet_Mention{Did: "did:plc:bob"}},
			{RichtextFacet_Mention: &appbsky.RichtextFacet_Mention{Did: "   "}},
		}},
		nil,
	}

	got := postutil.ExtractMentionDIDs(facets)
	want := []string{"did:plc:alice", "did:plc:bob"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractMentionDIDs = %#v, want %#v", got, want)
	}
}

func TestExtractMentionDIDsForText_ValidatesByteRanges(t *testing.T) {
	t.Parallel()

	facets := []*appbsky.RichtextFacet{
		{
			Index: &appbsky.RichtextFacet_ByteSlice{ByteStart: 0, ByteEnd: 6},
			Features: []*appbsky.RichtextFacet_Features_Elem{
				{RichtextFacet_Mention: &appbsky.RichtextFacet_Mention{Did: "did:plc:alice"}},
			},
		},
		{
			Index: &appbsky.RichtextFacet_ByteSlice{ByteStart: 0, ByteEnd: 99},
			Features: []*appbsky.RichtextFacet_Features_Elem{
				{RichtextFacet_Mention: &appbsky.RichtextFacet_Mention{Did: "did:plc:mallory"}},
			},
		},
		{
			Index: &appbsky.RichtextFacet_ByteSlice{ByteStart: 0, ByteEnd: 0},
			Features: []*appbsky.RichtextFacet_Features_Elem{
				{RichtextFacet_Mention: &appbsky.RichtextFacet_Mention{Did: "did:plc:empty"}},
			},
		},
	}

	got := postutil.ExtractMentionDIDsForText("@alice", facets)
	want := []string{"did:plc:alice"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractMentionDIDsForText = %#v, want %#v", got, want)
	}
}

func TestMergeMentionDIDs_DedupesAndReturnsNonNil(t *testing.T) {
	t.Parallel()

	got := postutil.MergeMentionDIDs(
		[]string{" did:plc:alice ", ""},
		[]string{"did:plc:alice", "did:plc:bob"},
	)
	want := []string{"did:plc:alice", "did:plc:bob"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MergeMentionDIDs = %#v, want %#v", got, want)
	}

	empty := postutil.MergeMentionDIDs(nil, []string{" "})
	if !reflect.DeepEqual(empty, []string{}) {
		t.Fatalf("MergeMentionDIDs empty = %#v, want empty slice", empty)
	}
}
