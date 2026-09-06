package video_test

import (
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/video"
)

func TestPlaybackURLBuilderUsesConfiguredTemplatesAndPathEscaping(t *testing.T) {
	t.Parallel()
	builder, err := video.NewPlaybackURLBuilder(video.PlaybackConfig{
		PlaylistTemplate:  "https://media.example/watch/{did}/{cid}/master.m3u8",
		ThumbnailTemplate: "https://media.example/watch/{did}/{cid}/poster.jpg",
	})
	if err != nil {
		t.Fatalf("NewPlaybackURLBuilder: %v", err)
	}
	playlist, thumbnail := builder.URLs(syntax.DID("did:web:example.com:user"), syntax.CID(testVideoCID))
	if playlist != "https://media.example/watch/did:web:example.com:user/"+testVideoCID+"/master.m3u8" ||
		thumbnail != "https://media.example/watch/did:web:example.com:user/"+testVideoCID+"/poster.jpg" {
		t.Fatalf("URLs = %q %q", playlist, thumbnail)
	}
	if strings.Contains(playlist, "blob") || strings.Contains(playlist, "mp4") {
		t.Fatalf("raw fallback leaked into %q", playlist)
	}
}

func TestPlaybackURLBuilderDefaultsAndRejectsUnsafeConfiguration(t *testing.T) {
	t.Parallel()
	builder, err := video.NewPlaybackURLBuilder(video.PlaybackConfig{})
	if err != nil {
		t.Fatalf("defaults: %v", err)
	}
	playlist, thumbnail := builder.URLs("did:plc:alice", syntax.CID(testVideoCID))
	if playlist != "https://video.bsky.app/watch/did:plc:alice/"+testVideoCID+"/playlist.m3u8" ||
		thumbnail != "https://video.bsky.app/watch/did:plc:alice/"+testVideoCID+"/thumbnail.jpg" {
		t.Fatalf("default URLs = %q %q", playlist, thumbnail)
	}
	for _, template := range []string{"http://media.example/{did}/{cid}", "https://media.example/{did}", "https://user:pass@media.example/{did}/{cid}"} {
		if _, err := video.NewPlaybackURLBuilder(video.PlaybackConfig{PlaylistTemplate: template}); err == nil {
			t.Fatalf("unsafe template accepted: %q", template)
		}
	}
}
