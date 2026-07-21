package api

import (
	"testing"
	"time"
)

func TestSavedPostTimestampEffects(t *testing.T) {
	existing := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	now := existing.Add(time.Hour)

	if got := savedAtForMutation(nil, now); !got.Equal(now) {
		t.Fatalf("new save time = %s, want %s", got, now)
	}
	if got := savedAtForMutation(&existing, now); !got.Equal(existing) {
		t.Fatalf("existing save time = %s, want %s", got, existing)
	}
	if got := savedAtForMutation(nil, now.Add(time.Hour)); !got.Equal(now.Add(time.Hour)) {
		t.Fatalf("resave time = %s, want %s", got, now.Add(time.Hour))
	}

	for _, test := range []struct {
		name    string
		renamed bool
		want    time.Time
	}{
		{name: "rename advances folder updatedAt", renamed: true, want: now},
		{name: "add save preserves folder updatedAt", want: existing},
		{name: "move save preserves folder updatedAt", want: existing},
		{name: "unfile save preserves folder updatedAt", want: existing},
		{name: "remove save preserves folder updatedAt", want: existing},
		{name: "delete folder does not synthesize update", want: existing},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := folderUpdatedAtForMutation(existing, now, test.renamed); !got.Equal(test.want) {
				t.Fatalf("updatedAt = %s, want %s", got, test.want)
			}
		})
	}
}
