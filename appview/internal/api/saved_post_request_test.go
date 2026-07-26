package api_test

import (
	"errors"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
)

func TestSavedPostRequestDecodesFolderAssignmentTriState(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantPresent bool
		wantID      *string
		wantCode    string
	}{
		{name: "no body", body: "", wantPresent: false},
		{name: "whitespace body", body: " \t\n", wantPresent: false},
		{name: "empty object", body: `{}`, wantPresent: false},
		{name: "explicit null", body: `{"folderId":null}`, wantPresent: true},
		{name: "folder value", body: `{"folderId":"opaque-folder"}`, wantPresent: true, wantID: stringPointer("opaque-folder")},
		{name: "unknown field", body: `{"folderId":null,"ownerDid":"did:plc:bob"}`, wantCode: "unexpected_field"},
		{name: "malformed", body: `{"folderId":`, wantCode: "malformed_body"},
		{name: "wrong type", body: `{"folderId":42}`, wantCode: "malformed_body"},
		{name: "trailing json", body: `{"folderId":null}{}`, wantCode: "malformed_body"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := api.DecodeSavePostRequest(strings.NewReader(tt.body))
			if tt.wantCode != "" {
				var fieldErr *api.FieldError
				if !errors.As(err, &fieldErr) {
					t.Fatalf("want *FieldError, got %v", err)
				}
				if fieldErr.Code != tt.wantCode {
					t.Fatalf("error code = %q, want %q", fieldErr.Code, tt.wantCode)
				}
				return
			}
			if err != nil {
				t.Fatalf("DecodeSavePostRequest: %v", err)
			}
			if got.Present != tt.wantPresent {
				t.Fatalf("Present = %v, want %v", got.Present, tt.wantPresent)
			}
			switch {
			case got.ID == nil && tt.wantID == nil:
			case got.ID == nil || tt.wantID == nil:
				t.Fatalf("ID = %v, want %v", got.ID, tt.wantID)
			case *got.ID != *tt.wantID:
				t.Fatalf("ID = %q, want %q", *got.ID, *tt.wantID)
			}
		})
	}
}

func stringPointer(value string) *string {
	return &value
}
