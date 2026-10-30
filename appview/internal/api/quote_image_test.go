package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	"social.craftsky/appview/internal/lexicon/schema"
)

const quoteImageTestCID = "bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"

func TestQuoteImagePublicationRoundTrip(t *testing.T) {
	req, err := DecodePostCreate(strings.NewReader(`{"text":"look here","sponsored":false,"embed":{"quote":{"uri":"at://did:plc:bob/social.craftsky.feed.post/target","cid":"` + quoteImageTestCID + `"},"imageCid":"` + quoteImageTestCID + `"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePostCreate(req); err != nil {
		t.Fatal(err)
	}
	record, err := lexiconRecordBodyAt(req, nil, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.ValidateCraftskyRecord(raw, "social.craftsky.feed.post"); err != nil {
		t.Fatal(err)
	}
	var decoded craftskylex.FeedPost
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	embed := decoded.Embed.FeedPost_QuoteEmbed
	if embed.ImageCid == nil || *embed.ImageCid != quoteImageTestCID {
		t.Fatalf("lost CID: %+v", embed)
	}
	// An edit must reconstruct the validation model without dropping selection.
	edited, err := postEditCandidate(raw)
	if err != nil || edited.Embed.ImageCID == nil || *edited.Embed.ImageCID != quoteImageTestCID {
		t.Fatalf("edit lost selection: %+v, %v", edited, err)
	}
	// Generated CBOR is also the persisted PDS representation.
	var cbor strings.Builder
	if err := decoded.MarshalCBOR(&cbor); err != nil {
		t.Fatal(err)
	}
	var reread craftskylex.FeedPost
	if err := reread.UnmarshalCBOR(strings.NewReader(cbor.String())); err != nil {
		t.Fatal(err)
	}
	if *reread.Embed.FeedPost_QuoteEmbed.ImageCid != quoteImageTestCID {
		t.Fatal("CBOR lost image selection")
	}
}

func TestQuoteImageValidation(t *testing.T) {
	for _, tc := range []struct {
		name, value  string
		quote, valid bool
	}{
		{"canonical", quoteImageTestCID, true, true},
		{"empty", "", true, false},
		{"invalid", "not-a-cid", true, false},
		{"noncanonical", strings.ToUpper(quoteImageTestCID), true, false},
		{"without quote", quoteImageTestCID, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := PostCreateRequest{Text: "quote", Embed: &EmbedRequest{ImageCID: &tc.value}}
			if tc.quote {
				req.Embed.Quote = &StrongRef{URI: "at://did:plc:bob/social.craftsky.feed.post/target", CID: quoteImageTestCID}
			}
			if err := ValidatePostCreate(req); (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
		})
	}
}

type quoteImageReader struct{ row *QuoteViewRow }

func (r quoteImageReader) QuoteViewRows(context.Context, []ResponseStrongRef) (map[string]*QuoteViewRow, error) {
	return map[string]*QuoteViewRow{"at://did:plc:bob/social.craftsky.feed.post/target": r.row}, nil
}

type quoteImageResolver struct{}

func (quoteImageResolver) ResolveHandle(context.Context, syntax.DID) (syntax.Handle, error) {
	return "bob.example", nil
}
func (quoteImageResolver) ResolveDID(context.Context, syntax.Handle) (syntax.DID, error) {
	return "did:plc:bob", nil
}

func TestQuoteImageHydrationResolvesEachQuoteIndependently(t *testing.T) {
	target := "at://did:plc:bob/social.craftsky.feed.post/target"
	targetRow := &PostRow{URI: target, DID: "did:plc:bob", Images: json.RawMessage(`[{"cid":"first","mime":"image/jpeg","alt":"first"},{"cid":"selected","mime":"image/png","alt":"selected alt","aspectRatio":{"width":4,"height":3}}]`)}
	for _, state := range []string{"visible", "hidden", "blocked", "muted", "unavailable"} {
		t.Run(state, func(t *testing.T) {
			var responses []*PostResponse
			for _, cid := range []string{"selected", "first", "removed", ""} {
				raw, _ := json.Marshal(map[string]any{"$type": "social.craftsky.feed.post#quoteEmbed", "imageCid": cid})
				responses = append(responses, BuildPostResponse(&PostRow{DID: "did:plc:alice", QuoteURI: &target, RawEmbed: raw}, "alice.example"))
			}
			if err := attachQuoteViews(context.Background(), quoteImageReader{&QuoteViewRow{State: state, Post: targetRow}}, quoteImageResolver{}, responses); err != nil {
				t.Fatal(err)
			}
			if state != "visible" {
				for _, response := range responses {
					if response.QuoteView.Post != nil {
						t.Fatal("hidden quote leaked media")
					}
				}
				return
			}
			for i, want := range []string{"selected", "first", "first", "first"} {
				if got := responses[i].QuoteView.Post.Images[0].CID; got != want {
					t.Fatalf("quote %d image=%s, want %s", i, got, want)
				}
			}
			selected := responses[0].QuoteView.Post.Images[0]
			if selected.Alt != "selected alt" || selected.AspectRatio.Width != 4 || selected.Thumb == "" {
				t.Fatalf("lost image metadata: %+v", selected)
			}
			if got := BuildQuoteView(&QuoteViewRow{State: "visible", Post: targetRow}, "bob.example").Post.Images[0].CID; got != "first" {
				t.Fatal("mutated original image order")
			}
		})
	}
	// A reorder still resolves by CID, rather than the old image index.
	targetRow.Images = json.RawMessage(`[{"cid":"selected","mime":"image/png","alt":"selected alt"},{"cid":"first","mime":"image/jpeg","alt":"first"}]`)
	view := BuildQuoteView(&QuoteViewRow{State: "visible", Post: targetRow}, "bob.example")
	preferQuoteImage(view, "first")
	if view.Post.Images[0].CID != "first" {
		t.Fatal("reorder changed selected image")
	}
}
