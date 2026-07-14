package notifications

import "testing"

func TestEffectivePreferencesDefaultEveryCategory(t *testing.T) {
	got, err := ResolvePreferences(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(Categories()) {
		t.Fatalf("got %d preferences, want %d", len(got), len(Categories()))
	}
	for _, category := range Categories() {
		want := Preference{Scope: Everyone, PushEnabled: true}
		if got[category] != want {
			t.Errorf("%s = %+v, want %+v", category, got[category], want)
		}
	}
}

func TestResolvePreferencesMergesPartialPatch(t *testing.T) {
	persisted := map[Category]Preference{
		Like: {Scope: PeopleIFollow, PushEnabled: false},
	}
	pushOff := false
	got, err := ResolvePreferences(persisted, map[Category]PreferencePatch{
		Follow: {PushEnabled: &pushOff},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got[Like] != persisted[Like] {
		t.Fatalf("omitted persisted like changed to %+v", got[Like])
	}
	if got[Follow] != (Preference{Scope: Everyone, PushEnabled: false}) {
		t.Fatalf("follow = %+v", got[Follow])
	}
	if got[Quote] != (Preference{Scope: Everyone, PushEnabled: true}) {
		t.Fatalf("unpersisted quote = %+v", got[Quote])
	}
}

func TestResolvePreferencesRejectsInvalidPatchWithoutReturningPartialState(t *testing.T) {
	invalidScope := Scope("nobody")
	got, err := ResolvePreferences(nil, map[Category]PreferencePatch{
		Like:                {Scope: &invalidScope},
		Category("unknown"): {},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if got != nil {
		t.Fatalf("invalid patch returned partial state: %v", got)
	}
}
