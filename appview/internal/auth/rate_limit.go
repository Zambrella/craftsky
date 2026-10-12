package auth

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RateLimitError is a definite refusal to execute, rather than response loss.
// Cause retains the original XRPC error without exposing it in public guidance.
type RateLimitError struct {
	RetryAfter time.Duration
	Cause      error
}

func (err *RateLimitError) Error() string { return "PDS request was rate limited" }
func (err *RateLimitError) Unwrap() error { return err.Cause }
func (err *RateLimitError) RetrySeconds() int {
	return max(1, int(math.Ceil(err.RetryAfter.Seconds())))
}

func RetryAfter(header string, now time.Time) time.Duration {
	if seconds, err := strconv.ParseInt(strings.TrimSpace(header), 10, 32); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if until, err := http.ParseTime(header); err == nil && until.After(now) {
		return until.Sub(now)
	}
	return 5 * time.Second
}
