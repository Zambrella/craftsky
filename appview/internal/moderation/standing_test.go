package moderation

import (
	"testing"
	"time"
)

func TestStandingCountsCasesOnceAndKeepsSuspensionBasesIndependent(t *testing.T) {
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	expired := now
	overturned := now
	tests := []struct {
		name       string
		strikes    []StrikeProjection
		severe     bool
		wantCount  int
		wantThresh bool
		wantActive bool
	}{
		{name: "none"},
		{name: "duplicate case counts once", strikes: []StrikeProjection{{CaseID: "one"}, {CaseID: "one"}}, wantCount: 1},
		{name: "expired and overturned excluded", strikes: []StrikeProjection{{CaseID: "active"}, {CaseID: "expired", ExpiredAt: &expired}, {CaseID: "overturned", OverturnedAt: &overturned}}, wantCount: 1},
		{name: "three reaches threshold", strikes: []StrikeProjection{{CaseID: "one"}, {CaseID: "two"}, {CaseID: "three"}}, wantCount: 3, wantThresh: true, wantActive: true},
		{name: "severe independent below threshold", strikes: []StrikeProjection{{CaseID: "one"}}, severe: true, wantCount: 1, wantActive: true},
		{name: "severe overlaps threshold", strikes: []StrikeProjection{{CaseID: "one"}, {CaseID: "two"}, {CaseID: "three"}}, severe: true, wantCount: 3, wantThresh: true, wantActive: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := DeriveStanding(test.strikes, test.severe)
			if got.ActiveStrikeCount != test.wantCount || got.ThresholdSuspended != test.wantThresh || got.SevereSuspended != test.severe || got.EffectiveSuspended != test.wantActive {
				t.Fatalf("DeriveStanding() = %+v", got)
			}
		})
	}
}
