package notifications

import (
	"slices"
	"testing"
)

func TestCategoriesExposeExactlyTheApprovedWireValues(t *testing.T) {
	want := []Category{Like, Follow, Reply, Mention, Quote, Repost, EverythingElse, InstagramMatch, Moderation}

	if got := Categories(); !slices.Equal(got, want) {
		t.Fatalf("Categories() = %v, want %v", got, want)
	}

	for _, category := range want {
		if !category.Valid() {
			t.Errorf("approved category %q is not valid", category)
		}
	}

	if Category("likeViaRepost").Valid() || Category("repostViaRepost").Valid() {
		t.Fatal("via-repost attribution must not be a notification category")
	}
	if EverythingElse.HasProducer() {
		t.Fatal("everythingElse must remain reserved without a producer")
	}
}

func TestModerationCategoryIsFixedScopeAndProduced(t *testing.T) {
	if !Moderation.Valid() || !Moderation.HasProducer() || !Moderation.FixedScope() {
		t.Fatal("moderation category must be valid, produced, and fixed-scope")
	}
	peopleIFollow := PeopleIFollow
	if _, err := ResolvePreferences(nil, map[Category]PreferencePatch{Moderation: {Scope: &peopleIFollow}}); err == nil {
		t.Fatal("moderation scope patch must be rejected")
	}
}

func TestCategoriesReturnsAnIndependentSlice(t *testing.T) {
	got := Categories()
	got[0] = Category("changed")

	if Categories()[0] != Like {
		t.Fatal("callers must not be able to mutate the category registry")
	}
}
