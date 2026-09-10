package api_test

import (
	"reflect"
	"testing"

	"social.craftsky/appview/internal/api"
)

func TestProfileCustomisationCatalogueIsClosedAndStable(t *testing.T) {
	t.Parallel()

	assertCatalogue(t, "colours", api.ProfileColourCatalogue, []string{
		"cobalt",
		"orchid",
		"rose",
		"amber",
		"lime",
		"teal",
		"ink",
	})
	assertCatalogue(t, "backgrounds", api.ProfileBackgroundCatalogue, []string{
		"none",
		"bayerdark",
		"cubedark",
		"dotcrossdark",
		"scallopdark",
		"skewdark",
		"x2",
	})

	wantDefault := api.ProfileCustomisation{
		Colour:     "cobalt",
		Background: "none",
	}
	if got := api.DefaultProfileCustomisation; got != wantDefault {
		t.Fatalf("default customisation = %+v, want %+v", got, wantDefault)
	}
}

func TestProfileCustomisationCatalogueFallsBackPerField(t *testing.T) {
	t.Parallel()

	got := api.EffectiveProfileCustomisation(api.ProfileCustomisation{
		Colour:     "future-colour",
		Background: "cubedark",
	})
	want := api.ProfileCustomisation{
		Colour:     "cobalt",
		Background: "cubedark",
	}
	if got != want {
		t.Fatalf("effective customisation = %+v, want %+v", got, want)
	}
}

func assertCatalogue(t *testing.T, name string, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s catalogue = %v, want %v", name, got, want)
	}

	seen := make(map[string]struct{}, len(got))
	for _, key := range got {
		if _, ok := seen[key]; ok {
			t.Fatalf("%s catalogue contains duplicate key %q", name, key)
		}
		seen[key] = struct{}{}
	}
}
