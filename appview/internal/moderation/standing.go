package moderation

import "time"

type StrikeProjection struct {
	CaseID       string
	DueAt        time.Time
	ExpiredAt    *time.Time
	OverturnedAt *time.Time
}

func (s StrikeProjection) Active() bool {
	return s.ExpiredAt == nil && s.OverturnedAt == nil
}

func (s StrikeProjection) Due(now time.Time) bool {
	return s.Active() && !now.UTC().Before(s.DueAt.UTC())
}

type Standing struct {
	ActiveStrikeCount  int
	ThresholdSuspended bool
	SevereSuspended    bool
	EffectiveSuspended bool
}

func StrikeDeadline(issuedAt time.Time) time.Time {
	issued := issuedAt.UTC()
	year := issued.Year() + 1
	day := issued.Day()
	lastDay := time.Date(year, issued.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(year, issued.Month(), day, issued.Hour(), issued.Minute(), issued.Second(), issued.Nanosecond(), time.UTC)
}

func DeriveStanding(strikes []StrikeProjection, severeSuspended bool) Standing {
	activeCases := make(map[string]struct{}, len(strikes))
	for _, strike := range strikes {
		if strike.Active() {
			activeCases[strike.CaseID] = struct{}{}
		}
	}
	count := len(activeCases)
	threshold := count >= 3
	return Standing{
		ActiveStrikeCount: count, ThresholdSuspended: threshold,
		SevereSuspended: severeSuspended, EffectiveSuspended: threshold || severeSuspended,
	}
}
