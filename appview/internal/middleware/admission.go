package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
)

type clientPrefixContextKey struct{}

func ClientPrefix(ctx context.Context) (string, bool) {
	prefix, ok := ctx.Value(clientPrefixContextKey{}).(string)
	return prefix, ok && prefix != ""
}

func RequestConcurrencyLimit(capacity int, logger *slog.Logger) (func(http.Handler) http.Handler, error) {
	if capacity <= 0 {
		return nil, errors.New("HTTP request concurrency capacity must be positive")
	}
	permits := make(chan struct{}, capacity)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case permits <- struct{}{}:
				defer func() { <-permits }()
				next.ServeHTTP(w, r)
			default:
				RejectBodyWithoutDrain(w, r)
				w.Header().Set("Retry-After", "1")
				if logger != nil {
					logger.Warn("HTTP request concurrency saturated",
						slog.String("component", "http"),
						slog.String("admission_stage", "concurrency"),
						slog.String("run_id", ctxkeys.GetRunID(r.Context())))
				}
				envelope.WriteError(w, http.StatusServiceUnavailable, "service_unavailable", "service is temporarily unavailable", ctxkeys.GetRunID(r.Context()), nil)
			}
		})
	}, nil
}

func OuterRateLimit(
	resolver *ClientAddressResolver,
	limiter Limiter,
	logger *slog.Logger,
) (func(http.Handler) http.Handler, error) {
	if resolver == nil {
		return nil, errors.New("HTTP client address resolver is required")
	}
	if limiter == nil {
		return nil, errors.New("HTTP outer rate limiter is required")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientKey := "unknown"
			if prefix, err := resolver.Resolve(r); err == nil {
				clientKey = prefix.String()
			} else if logger != nil {
				logger.Warn("HTTP peer address could not be classified",
					slog.String("component", "http"),
					slog.String("admission_stage", "client_address"),
					slog.String("run_id", ctxkeys.GetRunID(r.Context())))
			}
			decision := limiter.Allow(RateClassOuter, RateKeys{
				GlobalKey: "process",
				ClientKey: clientKey,
			})
			if !decision.Allowed {
				RejectBodyWithoutDrain(w, r)
				seconds := int((decision.RetryAfter + time.Second - 1) / time.Second)
				if seconds < 1 {
					seconds = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(seconds))
				if logger != nil {
					logger.Warn("HTTP request rate limited",
						slog.String("component", "http"),
						slog.String("admission_stage", "outer_rate"),
						slog.String("key_type", decision.KeyType),
						slog.String("run_id", ctxkeys.GetRunID(r.Context())))
				}
				envelope.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests", ctxkeys.GetRunID(r.Context()), nil)
				return
			}
			ctx := context.WithValue(r.Context(), clientPrefixContextKey{}, clientKey)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}, nil
}
