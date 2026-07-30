package languages

import (
	"hash/fnv"
	"slices"
	"strings"
	"testing"
)

func TestUT008V1BaseLanguageCatalogueMatchesPinnedSnapshot(t *testing.T) {
	tags := make([]string, 0, len(supportedBaseLanguageTags))
	for tag := range supportedBaseLanguageTags {
		tags = append(tags, tag)
	}
	slices.Sort(tags)

	if got, want := len(tags), 184; got != want {
		t.Fatalf("catalogue size = %d, want %d", got, want)
	}

	hash := fnv.New64a()
	if _, err := hash.Write([]byte(strings.Join(tags, "\n"))); err != nil {
		t.Fatalf("hash catalogue: %v", err)
	}
	if got, want := hash.Sum64(), uint64(0x5a751f77a5ee754c); got != want {
		t.Fatalf("catalogue fingerprint = %016x, want %016x", got, want)
	}
}
