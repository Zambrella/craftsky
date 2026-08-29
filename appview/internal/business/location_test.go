package business

import (
	"errors"
	"strings"
	"testing"
)

func TestLocationValidation(t *testing.T) {
	for input, want := range map[string]string{
		"US": "US",
		"gb": "GB",
		"ax": "AX",
		"ZW": "ZW",
	} {
		got, err := NormalizeCountry(input)
		if err != nil || got != want {
			t.Errorf("NormalizeCountry(%q) = (%q, %v), want (%q, nil)", input, got, err, want)
		}
	}
	for _, input := range []string{"", "U", "USA", "XK", "ZZ", "u1", " us "} {
		if _, err := NormalizeCountry(input); !errors.Is(err, ErrInvalidLocation) {
			t.Errorf("NormalizeCountry(%q) error = %v, want ErrInvalidLocation", input, err)
		}
	}
	if err := ValidateText(TextFieldLocality, strings.Repeat("a", 100)); err != nil {
		t.Fatalf("locality grapheme boundary rejected: %v", err)
	}
	if err := ValidateText(TextFieldLocality, strings.Repeat("a", 101)); !errors.Is(err, ErrInvalidText) {
		t.Fatalf("oversized locality error = %v, want ErrInvalidText", err)
	}
}

func TestIndependentLocationHydration(t *testing.T) {
	locality := "Edinburgh"
	got := HydrateIndependentLocation("gb", &locality)
	if got == nil || got.Country != "GB" || got.Locality == nil || *got.Locality != locality {
		t.Fatalf("valid location = %+v", got)
	}

	oversized := strings.Repeat("a", 101)
	got = HydrateIndependentLocation("US", &oversized)
	if got == nil || got.Country != "US" || got.Locality != nil {
		t.Fatalf("location with oversized locality = %+v, want country only", got)
	}

	for _, country := range []string{"ZZ", "XK", "USA"} {
		if got := HydrateIndependentLocation(country, &locality); got != nil {
			t.Errorf("invalid country %q hydrated as %+v", country, got)
		}
	}
}
