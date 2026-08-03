package scheduledposts

import (
	"errors"
	"testing"
	"time"
)

func TestValidateScheduledAt(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		scheduledAt time.Time
		wantErr     bool
	}{
		{
			name:        "rejects four minutes fifty-nine seconds",
			scheduledAt: now.Add(4*time.Minute + 59*time.Second),
			wantErr:     true,
		},
		{
			name:        "accepts inclusive five minute boundary",
			scheduledAt: now.Add(5 * time.Minute),
		},
		{
			name:        "rejects a non-whole minute",
			scheduledAt: now.Add(5*time.Minute + time.Second),
			wantErr:     true,
		},
		{
			name:        "accepts the last minute before the maximum",
			scheduledAt: now.Add(28*24*time.Hour - time.Minute),
		},
		{
			name:        "accepts inclusive twenty-eight day boundary",
			scheduledAt: now.Add(28 * 24 * time.Hour),
		},
		{
			name:        "rejects twenty-eight days plus one minute",
			scheduledAt: now.Add(28*24*time.Hour + time.Minute),
			wantErr:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateScheduledAt(now, test.scheduledAt)
			if test.wantErr {
				if !errors.Is(err, ErrInvalidScheduledAt) {
					t.Fatalf("ValidateScheduledAt() error = %v, want ErrInvalidScheduledAt", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateScheduledAt() error = %v, want nil", err)
			}
		})
	}
}

func TestScheduleEligibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		shape   PostShape
		wantErr bool
	}{
		{
			name:  "original standard post",
			shape: PostShape{Kind: PostKindStandard},
		},
		{
			name:  "standalone project post",
			shape: PostShape{Kind: PostKindProject},
		},
		{
			name:    "quote post",
			shape:   PostShape{Kind: PostKindStandard, HasQuoteEmbed: true},
			wantErr: true,
		},
		{
			name:    "reply or comment",
			shape:   PostShape{Kind: PostKindStandard, HasReplyReference: true},
			wantErr: true,
		},
		{
			name:    "project with quote",
			shape:   PostShape{Kind: PostKindProject, HasQuoteEmbed: true},
			wantErr: true,
		},
		{
			name:    "unknown post kind",
			shape:   PostShape{Kind: PostKind("unknown")},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateScheduleEligibility(test.shape)
			if test.wantErr {
				if !errors.Is(err, ErrIneligibleScheduledPost) {
					t.Fatalf("ValidateScheduleEligibility() error = %v, want ErrIneligibleScheduledPost", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateScheduleEligibility() error = %v, want nil", err)
			}
		})
	}
}
