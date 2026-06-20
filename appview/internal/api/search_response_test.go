package api_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
)

func TestSearchPostPageResponseOmitsPopularityScore(t *testing.T) {
	body, err := json.Marshal(api.SearchPostPageResponse{
		Hashtag: "sock",
		Items: []*api.PostResponse{{
			URI:         "at://did:plc:alice/social.craftsky.feed.post/aaa",
			CID:         "bafycid",
			Rkey:        "aaa",
			Text:        "sock update",
			Tags:        []string{"sock"},
			LikeCount:   3,
			RepostCount: 1,
			ReplyCount:  2,
			CreatedAt:   time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC),
			IndexedAt:   time.Date(2026, 6, 19, 12, 1, 0, 0, time.UTC),
			Author: api.PostAuthor{
				DID:    "did:plc:alice",
				Handle: "alice.craftsky.social",
			},
		}},
		Cursor: "opaque",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	jsonBody := string(body)
	for _, want := range []string{`"hashtag":"sock"`, `"items"`, `"likeCount":3`, `"repostCount":1`, `"replyCount":2`, `"cursor":"opaque"`} {
		if !strings.Contains(jsonBody, want) {
			t.Fatalf("body = %s, want containing %s", jsonBody, want)
		}
	}
	if strings.Contains(jsonBody, "popularityScore") {
		t.Fatalf("body = %s, must not expose popularityScore", jsonBody)
	}
}

func TestBuildProfileSearchSummaryIncludesAvatar(t *testing.T) {
	displayName := "Alice"
	description := "Knitter"
	avatarCID := "bafavatar"
	avatarMime := "image/jpeg"

	got := api.BuildProfileSearchSummary(api.ProfileSearchRow{
		DID:               "did:plc:alice",
		Handle:            "alice.craftsky.social",
		DisplayName:       &displayName,
		Description:       &description,
		AvatarCID:         &avatarCID,
		AvatarMime:        &avatarMime,
		IsCraftskyProfile: true,
		ViewerIsFollowing: true,
	})

	wantAvatar := "https://cdn.bsky.app/img/avatar/plain/did:plc:alice/bafavatar@jpeg"
	if got.Avatar == nil || *got.Avatar != wantAvatar {
		t.Fatalf("avatar = %v, want %q", got.Avatar, wantAvatar)
	}
	if !got.ViewerIsFollowing || !got.IsCraftskyProfile || got.DisplayName == nil || *got.DisplayName != displayName {
		t.Fatalf("summary = %+v, want profile fields preserved", got)
	}
}
