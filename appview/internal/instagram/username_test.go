package instagram

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeInstagramUsername(t *testing.T) {
	t.Parallel()

	for input, want := range map[string]string{
		"Alice.Crafts":          "alice.crafts",
		" @Alice_Crafts9 ":      "alice_crafts9",
		"2synthetic":            "2synthetic",
		strings.Repeat("a", 30): strings.Repeat("a", 30),
	} {
		got, err := NormalizeInstagramUsername(input)
		if err != nil {
			t.Errorf("NormalizeInstagramUsername(%q): %v", input, err)
			continue
		}
		if got != want {
			t.Errorf("NormalizeInstagramUsername(%q) = %q, want %q", input, got, want)
		}
	}

	for _, input := range []string{
		"",
		"   ",
		"@@alice",
		"alice smith",
		"alice@example",
		"alice/crafts",
		"álïce",
		"Ａｌｉｃｅ",
		strings.Repeat("a", 31),
	} {
		if got, err := NormalizeInstagramUsername(input); !errors.Is(err, ErrInvalidInstagramUsername) {
			t.Errorf("NormalizeInstagramUsername(%q) = %q, %v; want invalid username", input, got, err)
		}
	}
}

func TestNormalizeImportEntriesIsDeterministicAndDeduplicated(t *testing.T) {
	t.Parallel()

	got, err := NormalizeImportEntries([]ImportEntry{
		{Username: " Alice.Crafts "},
		{Username: "@alice.crafts"},
		{Username: "ALICE.CRAFTS"},
		{Username: "bob_9"},
	})
	if err != nil {
		t.Fatalf("NormalizeImportEntries: %v", err)
	}
	want := []ImportEntry{
		{Username: "alice.crafts"},
		{Username: "bob_9"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalized entries = %#v, want %#v", got, want)
	}
}

func TestNormalizeImportEntriesEnforcesDeduplicatedLimit(t *testing.T) {
	t.Parallel()

	duplicates := make([]ImportEntry, MaxImportEntries+20)
	for i := range duplicates {
		duplicates[i] = ImportEntry{Username: "same"}
	}
	got, err := NormalizeImportEntries(duplicates)
	if err != nil || len(got) != 1 {
		t.Fatalf("duplicates normalized to %d entries with %v, want one", len(got), err)
	}

	tooMany := make([]ImportEntry, 0, MaxImportEntries+1)
	for i := 0; i <= MaxImportEntries; i++ {
		tooMany = append(tooMany, ImportEntry{Username: syntheticUsername(i)})
	}
	if _, err := NormalizeImportEntries(tooMany); !errors.Is(err, ErrTooManyImportEntries) {
		t.Fatalf("too-many error = %v, want ErrTooManyImportEntries", err)
	}
}

func syntheticUsername(value int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz"
	if value == 0 {
		return "a"
	}
	var reversed []byte
	for value > 0 {
		reversed = append(reversed, alphabet[value%len(alphabet)])
		value /= len(alphabet)
	}
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	return "synthetic_" + string(reversed)
}
