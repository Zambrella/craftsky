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

func TestDecodeProfilePut_RejectsAvatar(t *testing.T) {
	t.Parallel()
	_, err := api.DecodeProfilePut(strings.NewReader(`{"avatar":"blob:..."}`))
	var fe *api.FieldError
	if err == nil || !asFieldErr(err, &fe) {
		t.Fatalf("want FieldError; got %v", err)
	}
	if _, ok := fe.Fields["avatar"]; !ok {
		t.Errorf("fields = %v", fe.Fields)
	}
}

func TestDecodeProfilePut_RejectsBanner(t *testing.T) {
	t.Parallel()
	_, err := api.DecodeProfilePut(strings.NewReader(`{"banner":"blob:..."}`))
	var fe *api.FieldError
	if err == nil || !asFieldErr(err, &fe) {
		t.Fatal("want FieldError")
	}
	if _, ok := fe.Fields["banner"]; !ok {
		t.Errorf("fields = %v", fe.Fields)
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
