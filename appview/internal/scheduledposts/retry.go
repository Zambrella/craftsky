package scheduledposts

import "time"

var retryOffsets = [...]time.Duration{
	0,
	time.Minute,
	3 * time.Minute,
	7 * time.Minute,
	15 * time.Minute,
	30 * time.Minute,
}

func RetryAttemptAt(due time.Time, attempt int, jitter time.Duration) (time.Time, bool) {
	if due.IsZero() || attempt < 0 || attempt >= len(retryOffsets) {
		return time.Time{}, false
	}

	at := due.Add(retryOffsets[attempt])
	if attempt > 0 && attempt < len(retryOffsets)-1 {
		at = at.Add(jitter)
	}
	if at.Before(due) {
		at = due
	}
	deadline := due.Add(retryOffsets[len(retryOffsets)-1])
	if at.After(deadline) {
		at = deadline
	}
	return at, true
}
