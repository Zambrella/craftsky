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
		"ink",
		"terracotta",
		"denim",
		"forest",
		"plum",
		"seafoam",
		"dusty-pink",
		"chocolate",
	})
	assertCatalogue(t, "backgrounds", api.ProfileBackgroundCatalogue, []string{
		"none",
		"bayerdark",
		"cubedark",
		"dotcrossdark",
		"scallopdark",
		"skewdark",
		"x2",
		"craft-sewing",
		"craft-knitting",
		"craft-crochet",
		"craft-quilting",
		"craft-embroidery",
		"animated-flow",
		"animated-waves",
		"animated-weave",
		"animated-grain",
		"animated-marble",
		"animated-textile",
		"animated-ribbons",
		"animated-silk",
		"animated-bloom",
		"animated-sunbeams",
		"animated-lagoon",
		"animated-nebula",
		"animated-prism",
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

func TestRetiredProfileCustomisationChoicesFallBackIndependently(t *testing.T) {
	t.Parallel()
	got := api.EffectiveProfileCustomisation(api.ProfileCustomisation{Colour: "teal", Background: "x2"})
	if got != (api.ProfileCustomisation{Colour: "cobalt", Background: "x2"}) {
		t.Fatalf("retired colour fallback = %+v", got)
	}
	for _, background := range []string{"animated-dust", "animated-satin", "animated-opal"} {
		got = api.EffectiveProfileCustomisation(api.ProfileCustomisation{Colour: "seafoam", Background: background})
		if got != (api.ProfileCustomisation{Colour: "seafoam", Background: "none"}) {
			t.Fatalf("retired background %q fallback = %+v", background, got)
		}
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
