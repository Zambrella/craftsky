package api_test

import (
	"errors"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
)

func TestSavedPostFolderNameValidation(t *testing.T) {
	hundred := strings.Repeat("界", 100)
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "one character", input: "A", want: "A"},
		{name: "trims surrounding whitespace", input: "  Ideas \t", want: "Ideas"},
		{name: "preserves accepted casing", input: "IDEAS", want: "IDEAS"},
		{name: "allows duplicate display value", input: "Ideas", want: "Ideas"},
		{name: "allows emoji", input: "🧶 Ideas", want: "🧶 Ideas"},
		{name: "allows punctuation", input: "WIPs & maybes!", want: "WIPs & maybes!"},
		{name: "allows one hundred unicode characters", input: hundred, want: hundred},
		{name: "rejects empty", input: "", wantErr: true},
		{name: "rejects whitespace", input: " \t\n ", wantErr: true},
		{name: "rejects one hundred and one unicode characters", input: hundred + "界", wantErr: true},
		{name: "rejects slash", input: "Ideas/Next", wantErr: true},
		{name: "rejects backslash", input: `Ideas\Next`, wantErr: true},
		{name: "rejects newline", input: "Ideas\nNext", wantErr: true},
		{name: "rejects tab", input: "Ideas\tNext", wantErr: true},
		{name: "rejects nul", input: "Ideas\x00Next", wantErr: true},
		{name: "rejects unicode control", input: "Ideas\u0085Next", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := api.NormalizeSavedPostFolderName(tt.input)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("NormalizeSavedPostFolderName: %v", err)
				}
				if got != tt.want {
					t.Fatalf("normalized name = %q, want %q", got, tt.want)
				}
				return
			}
			var fieldErr *api.FieldError
			if !errors.As(err, &fieldErr) {
				t.Fatalf("want *FieldError, got %v", err)
			}
			if fieldErr.Code != "validation_failed" {
				t.Fatalf("error code = %q, want validation_failed", fieldErr.Code)
			}
			if fieldErr.Fields["name"] == "" {
				t.Fatalf("name field error missing: %#v", fieldErr.Fields)
			}
		})
	}
}
