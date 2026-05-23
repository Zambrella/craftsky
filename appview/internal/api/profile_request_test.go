// appview/internal/api/profile_request_test.go
package api_test

import (
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
)

func TestDecodeProfilePut_HappyPath(t *testing.T) {
	t.Parallel()
	body := `{"displayName":"Alice","description":"textile","crafts":["sewing"]}`
	req, err := api.DecodeProfilePut(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if req.DisplayName == nil || *req.DisplayName != "Alice" {
		t.Errorf("displayName = %v", req.DisplayName)
	}
	if req.Crafts == nil || len(req.Crafts) != 1 || req.Crafts[0] != "sewing" {
		t.Errorf("crafts = %v", req.Crafts)
	}
}

func TestDecodeProfilePut_AcceptsAvatarAndBannerBlobs(t *testing.T) {
	t.Parallel()
	req, err := api.DecodeProfilePut(strings.NewReader(`{
		"avatar":{"$type":"blob","ref":{"$link":"bafavatar"},"mimeType":"image/jpeg","size":1},
		"banner":{"$type":"blob","ref":{"$link":"bafbanner"},"mimeType":"image/png","size":2}
	}`))
	if err != nil {
		t.Fatalf("DecodeProfilePut: %v", err)
	}
	if !req.Avatar.Present || req.Avatar.Blob == nil {
		t.Fatalf("avatar = %+v", req.Avatar)
	}
	if !req.Banner.Present || req.Banner.Blob == nil {
		t.Fatalf("banner = %+v", req.Banner)
	}
}

func TestDecodeProfilePut_AcceptsNullAvatarClear(t *testing.T) {
	t.Parallel()
	req, err := api.DecodeProfilePut(strings.NewReader(`{"avatar":null}`))
	if err != nil {
		t.Fatalf("DecodeProfilePut: %v", err)
	}
	if !req.Avatar.Present || req.Avatar.Blob != nil {
		t.Fatalf("avatar = %+v", req.Avatar)
	}
}

func TestValidateProfilePut_OversizeDisplayName(t *testing.T) {
	t.Parallel()
	dn := strings.Repeat("x", 641) // 641 bytes > 640.
	req := api.ProfilePutRequest{DisplayName: &dn}
	err := api.ValidateProfilePut(req)
	var fe *api.FieldError
	if !asFieldErr(err, &fe) {
		t.Fatalf("want FieldError; got %v", err)
	}
	if _, ok := fe.Fields["displayName"]; !ok {
		t.Errorf("fields = %v", fe.Fields)
	}
}

func TestValidateProfilePut_TooManyCrafts(t *testing.T) {
	t.Parallel()
	crafts := make([]string, 11)
	for i := range crafts {
		crafts[i] = "a"
	}
	req := api.ProfilePutRequest{Crafts: crafts}
	err := api.ValidateProfilePut(req)
	var fe *api.FieldError
	if !asFieldErr(err, &fe) || fe.Fields["crafts"] == "" {
		t.Fatalf("want FieldError on crafts; got %v", err)
	}
}

func TestValidateProfilePut_RejectsUnsupportedAvatarMIME(t *testing.T) {
	t.Parallel()
	req := api.ProfilePutRequest{Avatar: api.ProfileImageUpdate{
		Present: true,
		Blob: map[string]any{
			"$type":    "blob",
			"ref":      map[string]any{"$link": "bafavatar"},
			"mimeType": "image/gif",
			"size":     1,
		},
	}}
	err := api.ValidateProfilePut(req)
	var fe *api.FieldError
	if !asFieldErr(err, &fe) || fe.Fields["avatar.mimeType"] == "" {
		t.Fatalf("want FieldError on avatar.mimeType; got %v", err)
	}
}

// asFieldErr is a tiny helper mirroring errors.As for our concrete type.
func asFieldErr(err error, out **api.FieldError) bool {
	if err == nil {
		return false
	}
	if fe, ok := err.(*api.FieldError); ok {
		*out = fe
		return true
	}
	return false
}
