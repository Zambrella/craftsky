package api_test

import (
	"errors"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
)

func TestProfileCustomisationRequestAcceptsCompleteSupportedValue(t *testing.T) {
	t.Parallel()

	got, err := api.DecodeProfileCustomisationPut(strings.NewReader(
		`{"colour":"teal","profileBorder":"thick","profileBackground":"cubedark"}`,
	))
	if err != nil {
		t.Fatalf("DecodeProfileCustomisationPut: %v", err)
	}
	want := api.ProfileCustomisation{
		Colour:     "teal",
		Border:     "thick",
		Background: "cubedark",
	}
	if got != want {
		t.Fatalf("request = %+v, want %+v", got, want)
	}
}

func TestProfileCustomisationRequestRejectsInvalidBodies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		wantCode  string
		wantField string
	}{
		{
			name:      "missing colour",
			body:      `{"profileBorder":"medium","profileBackground":"none"}`,
			wantCode:  "invalid_request",
			wantField: "colour",
		},
		{
			name:      "missing border",
			body:      `{"colour":"cobalt","profileBackground":"none"}`,
			wantCode:  "invalid_request",
			wantField: "profileBorder",
		},
		{
			name:      "missing background",
			body:      `{"colour":"cobalt","profileBorder":"medium"}`,
			wantCode:  "invalid_request",
			wantField: "profileBackground",
		},
		{
			name:      "unknown field",
			body:      `{"colour":"cobalt","profileBorder":"medium","profileBackground":"none","textureUrl":"https://example.com/tile.png"}`,
			wantCode:  "unexpected_field",
			wantField: "textureUrl",
		},
		{
			name:      "non-string colour",
			body:      `{"colour":123,"profileBorder":"medium","profileBackground":"none"}`,
			wantCode:  "invalid_request",
			wantField: "colour",
		},
		{
			name:      "nested background resource",
			body:      `{"colour":"cobalt","profileBorder":"medium","profileBackground":{"asset":"https://example.com/tile.png"}}`,
			wantCode:  "invalid_request",
			wantField: "profileBackground",
		},
		{
			name:      "arbitrary hex colour",
			body:      `{"colour":"#ff00ff","profileBorder":"medium","profileBackground":"none"}`,
			wantCode:  "validation_failed",
			wantField: "colour",
		},
		{
			name:      "URL colour",
			body:      `{"colour":"https://example.com/colour","profileBorder":"medium","profileBackground":"none"}`,
			wantCode:  "validation_failed",
			wantField: "colour",
		},
		{
			name:      "unsupported border",
			body:      `{"colour":"cobalt","profileBorder":"none","profileBackground":"none"}`,
			wantCode:  "validation_failed",
			wantField: "profileBorder",
		},
		{
			name:      "unsupported background",
			body:      `{"colour":"cobalt","profileBorder":"medium","profileBackground":"future-texture"}`,
			wantCode:  "validation_failed",
			wantField: "profileBackground",
		},
		{
			name:      "malformed JSON",
			body:      `{"colour":`,
			wantCode:  "malformed_body",
			wantField: "_",
		},
		{
			name:      "duplicate field",
			body:      `{"colour":"cobalt","colour":"teal","profileBorder":"medium","profileBackground":"none"}`,
			wantCode:  "malformed_body",
			wantField: "_",
		},
		{
			name:      "trailing JSON value",
			body:      `{"colour":"cobalt","profileBorder":"medium","profileBackground":"none"}{}`,
			wantCode:  "malformed_body",
			wantField: "_",
		},
		{
			name:      "oversized body",
			body:      `{"colour":"cobalt","profileBorder":"medium","profileBackground":"` + strings.Repeat("x", 4096) + `"}`,
			wantCode:  "malformed_body",
			wantField: "_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := api.DecodeProfileCustomisationPut(strings.NewReader(tt.body))
			var fieldErr *api.FieldError
			if !errors.As(err, &fieldErr) {
				t.Fatalf("error = %v, want *FieldError", err)
			}
			if fieldErr.Code != tt.wantCode {
				t.Fatalf("error code = %q, want %q", fieldErr.Code, tt.wantCode)
			}
			if fieldErr.Fields[tt.wantField] == "" {
				t.Fatalf("fields = %v, want error for %q", fieldErr.Fields, tt.wantField)
			}
		})
	}
}
