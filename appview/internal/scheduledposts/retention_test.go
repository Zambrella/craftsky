package scheduledposts

import (
	"testing"
	"time"
)

func TestRetentionDeadlines(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		got  time.Time
		want time.Time
	}{
		{name: "unclaimed media", got: UnclaimedMediaExpiresAt(at), want: at.Add(24 * time.Hour)},
		{name: "Needs attention", got: NeedsAttentionExpiresAt(at), want: at.Add(30 * 24 * time.Hour)},
		{name: "published tombstone", got: PublicationTombstoneExpiresAt(at), want: at.Add(30 * 24 * time.Hour)},
		{name: "successful publication cleanup", got: SuccessfulPublicationCleanupAt(at), want: at},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if !test.got.Equal(test.want) {
				t.Fatalf("deadline = %s, want %s", test.got, test.want)
			}
		})
	}
}
