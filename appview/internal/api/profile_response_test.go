// appview/internal/api/profile_response_test.go
package api_test

import (
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
)

func strPtr(s string) *string { return &s }

func TestBuildProfileResponse_FullRow(t *testing.T) {
	t.Parallel()
	row := &api.ProfileRow{
		DID:         "did:plc:xyz",
		Crafts:      []string{"knitting", "sewing"},
		CreatedAt:   time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC),
		DisplayName: strPtr("Alice"),
		Description: strPtr("textile person"),
		AvatarCID:   strPtr("bafav"),
		AvatarMime:  strPtr("image/jpeg"),
		BannerCID:   strPtr("bafbn"),
		BannerMime:  strPtr("image/png"),
	}
	out := api.BuildProfileResponse(row, "alice.example", true)
	if out.DID != "did:plc:xyz" || out.Handle != "alice.example" {
		t.Errorf("did/handle mismatch: %+v", out)
	}
	if out.DisplayName == nil || *out.DisplayName != "Alice" {
		t.Errorf("displayName = %v", out.DisplayName)
	}
	if out.Avatar == nil ||
		*out.Avatar != "https://cdn.bsky.app/img/avatar/plain/did:plc:xyz/bafav@jpeg" {
		t.Errorf("avatar = %v", out.Avatar)
	}
	if out.Banner == nil ||
		*out.Banner != "https://cdn.bsky.app/img/banner/plain/did:plc:xyz/bafbn@png" {
		t.Errorf("banner = %v", out.Banner)
	}
	if out.CreatedAt == nil {
		t.Errorf("createdAt should be present for GET")
	}
}

func TestBuildProfileResponse_UnknownMimeOmitsAvatar(t *testing.T) {
	t.Parallel()
	row := &api.ProfileRow{
		DID:        "did:plc:xyz",
		Crafts:     []string{},
		CreatedAt:  time.Now(),
		AvatarCID:  strPtr("baf"),
		AvatarMime: strPtr("image/tiff"), // not in supported set.
	}
	out := api.BuildProfileResponse(row, "h", true)
	if out.Avatar != nil {
		t.Errorf("avatar should be omitted; got %v", *out.Avatar)
	}
}

func TestBuildProfileResponse_NoCreatedAtForPut(t *testing.T) {
	t.Parallel()
	row := &api.ProfileRow{
		DID:       "did:plc:x",
		Crafts:    []string{},
		CreatedAt: time.Now(),
	}
	out := api.BuildProfileResponse(row, "h", false)
	if out.CreatedAt != nil {
		t.Errorf("createdAt should be omitted for PUT; got %v", *out.CreatedAt)
	}
}

func TestBuildProfileResponse_EmptyCraftsStaysArray(t *testing.T) {
	t.Parallel()
	row := &api.ProfileRow{DID: "did:plc:x", Crafts: nil, CreatedAt: time.Now()}
	out := api.BuildProfileResponse(row, "h", true)
	if out.Crafts == nil {
		t.Fatal("crafts must never be nil (must serialise as [])")
	}
	if len(out.Crafts) != 0 {
		t.Errorf("crafts = %v", out.Crafts)
	}
}
