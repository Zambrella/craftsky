package middleware

import (
	"log/slog"
	"net/http"
	"strconv"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
)

func RateLimit(limiter *LocalRateLimiter, class RateClass, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			keys := RateKeys{DeviceID: r.Header.Get("X-Craftsky-Device-Id")}
			if sid, ok := ctxkeys.GetOAuthSessionID(r.Context()); ok {
				keys.TokenKey = sid
			}
			decision := limiter.Allow(class, keys)
			if !decision.Allowed {
				RejectBodyWithoutDrain(w, r)
				seconds := int(decision.RetryAfter.Seconds())
				if seconds < 1 {
					seconds = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(seconds))
				if logger != nil {
					logger.Warn("request rate limited", slog.String("class", string(class)), slog.String("key_type", decision.KeyType), slog.String("run_id", GetRunID(r.Context())))
				}
				envelope.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests", GetRunID(r.Context()), nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
