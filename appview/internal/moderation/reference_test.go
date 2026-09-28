package moderation

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestCaseReferenceRoundTrip(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("c73bcdcc-2669-4bf6-81d3-e4ae73fb11fd")
	want := "MOD-c73bcdcc-2669-4bf6-81d3-e4ae73fb11fd"

	formatted, err := FormatCaseReference(id)
	if err != nil {
		t.Fatalf("FormatCaseReference() error = %v, want nil", err)
	}
	if formatted != want {
		t.Fatalf("FormatCaseReference() = %q, want %q", formatted, want)
	}

	for _, raw := range []string{want, strings.ToUpper(want)} {
		reference, err := ParseCaseReference(raw)
		if err != nil {
			t.Fatalf("ParseCaseReference(%q) error = %v, want nil", raw, err)
		}
		if reference.String() != want {
			t.Fatalf("ParseCaseReference(%q).String() = %q, want %q", raw, reference, want)
		}
		if reference.UUID() != id {
			t.Fatalf("ParseCaseReference(%q).UUID() = %s, want %s", raw, reference.UUID(), id)
		}
	}
}

func TestCaseReferenceRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{
		"",
		"c73bcdcc-2669-4bf6-81d3-e4ae73fb11fd",
		"CASE-c73bcdcc-2669-4bf6-81d3-e4ae73fb11fd",
		"MOD-not-a-uuid",
		"MOD-00000000-0000-0000-0000-000000000000",
		"MOD-6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	} {
		raw := raw
		t.Run(raw, func(t *testing.T) {
			t.Parallel()

			if _, err := ParseCaseReference(raw); !errors.Is(err, ErrInvalidCaseReference) {
				t.Fatalf("ParseCaseReference(%q) error = %v, want ErrInvalidCaseReference", raw, err)
			}
		})
	}
}

func TestFormatCaseReferenceRejectsNonV4UUID(t *testing.T) {
	t.Parallel()

	for _, id := range []uuid.UUID{
		uuid.Nil,
		uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
	} {
		if _, err := FormatCaseReference(id); !errors.Is(err, ErrInvalidCaseReference) {
			t.Fatalf("FormatCaseReference(%s) error = %v, want ErrInvalidCaseReference", id, err)
		}
	}
}
