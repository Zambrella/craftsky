package scheduledposts

import (
	"testing"
	"time"
)

func TestRetryAttemptAt(t *testing.T) {
	t.Parallel()

	due := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	wantOffsets := []time.Duration{
		0,
		time.Minute,
		3 * time.Minute,
		7 * time.Minute,
		15 * time.Minute,
		30 * time.Minute,
	}

	for attempt, wantOffset := range wantOffsets {
		got, ok := RetryAttemptAt(due, attempt, 0)
		if !ok {
			t.Fatalf("RetryAttemptAt(attempt %d) was not eligible", attempt)
		}
		if want := due.Add(wantOffset); !got.Equal(want) {
			t.Fatalf("RetryAttemptAt(attempt %d) = %s, want %s", attempt, got, want)
		}
	}

	t.Run("applies deterministic jitter to intermediate attempts", func(t *testing.T) {
		got, ok := RetryAttemptAt(due, 2, 30*time.Second)
		if !ok {
			t.Fatal("RetryAttemptAt() was not eligible")
		}
		if want := due.Add(3*time.Minute + 30*time.Second); !got.Equal(want) {
			t.Fatalf("RetryAttemptAt() = %s, want %s", got, want)
		}
	})

	t.Run("never jitters the due attempt early", func(t *testing.T) {
		got, ok := RetryAttemptAt(due, 0, -time.Minute)
		if !ok || !got.Equal(due) {
			t.Fatalf("RetryAttemptAt() = %s, %t, want %s, true", got, ok, due)
		}
	})

	t.Run("never jitters the final attempt beyond thirty minutes", func(t *testing.T) {
		got, ok := RetryAttemptAt(due, 5, time.Minute)
		want := due.Add(30 * time.Minute)
		if !ok || !got.Equal(want) {
			t.Fatalf("RetryAttemptAt() = %s, %t, want %s, true", got, ok, want)
		}
	})

	for _, attempt := range []int{-1, len(wantOffsets)} {
		if _, ok := RetryAttemptAt(due, attempt, 0); ok {
			t.Fatalf("RetryAttemptAt(attempt %d) was eligible, want exhausted", attempt)
		}
	}
}

func TestAutomaticPublicationCutoffIncludesExactlyThirtyMinutes(t *testing.T) {
	t.Parallel()

	due := time.Date(2026, time.September, 4, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		at   time.Time
		want bool
	}{
		{name: "29:59 remains eligible", at: due.Add(29*time.Minute + 59*time.Second), want: true},
		{name: "30:00 is the final eligible instant", at: due.Add(30 * time.Minute), want: true},
		{name: "30:01 is too late", at: due.Add(30*time.Minute + time.Second), want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := AutomaticPublicationEligible(due, test.at); got != test.want {
				t.Fatalf("AutomaticPublicationEligible() = %t, want %t", got, test.want)
			}
		})
	}
}
