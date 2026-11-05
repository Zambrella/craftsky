package api

import (
	"encoding/json"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type fixedPlaybackBuilder struct{}

func (fixedPlaybackBuilder) URLs(did syntax.DID, blobCID syntax.CID) (string, string) {
	return "https://media.example/" + did.String() + "/" + blobCID.String() + "/index.m3u8",
		"https://media.example/" + did.String() + "/" + blobCID.String() + "/poster.jpg"
}

func TestConfiguredPlaybackReachesPostAndSearchHydration(t *testing.T) {
	row := &PostRow{
		DID:      "did:plc:alice",
		Rkey:     "post1",
		RawEmbed: json.RawMessage(`{"$type":"app.bsky.embed.video","video":{"ref":{"$link":"bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"},"mimeType":"video/mp4","size":123}}`),
	}
	builder := fixedPlaybackBuilder{}
	sources := []struct {
		name   string
		source any
	}{
		{name: "post store", source: NewPostStoreWithPlayback(nil, nil, builder)},
		{name: "search store", source: NewSearchStoreWithPlayback(nil, nil, builder)},
	}

	for _, test := range sources {
		t.Run(test.name, func(t *testing.T) {
			response := buildPostResponse(row, "alice.example", test.source)
			if response.Video == nil {
				t.Fatal("video was not hydrated")
			}
			if got, want := response.Video.Playlist, "https://media.example/did:plc:alice/bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a/index.m3u8"; got != want {
				t.Fatalf("playlist = %q, want %q", got, want)
			}
		})
	}
}

type hostedPlaybackBuilder struct{ fixedPlaybackBuilder }

func (hostedPlaybackBuilder) URLsWithEmbed(did syntax.DID, blobCID syntax.CID) (string, string, string) {
	playlist, thumbnail := (fixedPlaybackBuilder{}).URLs(did, blobCID)
	return playlist, thumbnail, "https://player.mediadelivery.net/embed/123/aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa"
}

func TestHostedPlaybackMetadataIsAdditive(t *testing.T) {
	row := &PostRow{DID: "did:plc:alice", Rkey: "post1", RawEmbed: json.RawMessage(`{"$type":"app.bsky.embed.video","video":{"ref":{"$link":"bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"},"mimeType":"video/mp4","size":123}}`)}
	standard := BuildPostResponse(row, "alice.test", fixedPlaybackBuilder{})
	if standard.Video == nil || standard.Video.EmbedURL != "" {
		t.Fatal("unexpected hosted destination on standard provider")
	}
	hosted := BuildPostResponse(row, "alice.test", hostedPlaybackBuilder{})
	if hosted.Video == nil || hosted.Video.EmbedURL == "" || hosted.Video.Playlist != standard.Video.Playlist {
		t.Fatal("hosted destination lost direct playback fallback")
	}
	raw, err := json.Marshal(hosted.Video)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["embedUrl"] != hosted.Video.EmbedURL {
		t.Fatal("missing camelCase hosted destination")
	}
}
