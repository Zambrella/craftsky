package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type videoQuoteReader struct{ quoteImageReader }

func (videoQuoteReader) PostPlaybackURLBuilder() PlaybackURLBuilder { return fixedPlaybackBuilder{} }

func TestQuoteVideoPreviewUsesConfiguredPlaybackAndVisibility(t *testing.T) {
	target := "at://did:plc:bob/social.craftsky.feed.post/target"
	row := &PostRow{URI: target, DID: "did:plc:bob", Rkey: "target", RawEmbed: json.RawMessage(`{"$type":"app.bsky.embed.video","video":{"ref":{"$link":"bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"},"mimeType":"video/mp4","size":123},"alt":"Knitting in progress","aspectRatio":{"width":9,"height":16}}`)}
	response := &PostResponse{Quote: &ResponseStrongRef{URI: target}}
	store := videoQuoteReader{quoteImageReader{&QuoteViewRow{State: "visible", Post: row}}}
	if err := attachQuoteViews(context.Background(), store, quoteImageResolver{}, []*PostResponse{response}); err != nil {
		t.Fatal(err)
	}
	preview := response.QuoteView.Post
	if preview == nil || preview.Video == nil {
		t.Fatal("video omitted from quote preview")
	}
	if !strings.HasPrefix(preview.Video.Thumbnail, "https://media.example/did:plc:bob/") || preview.Video.Alt != "Knitting in progress" || preview.Video.AspectRatio.Width != 9 {
		t.Fatal("incorrect video preview")
	}
	for _, state := range []string{"hidden", "blocked", "muted", "unavailable"} {
		view := BuildQuoteView(&QuoteViewRow{State: state, Post: row}, "bob.test", fixedPlaybackBuilder{})
		if view.Post != nil {
			t.Fatalf("%s quote leaked video preview", state)
		}
	}
}
