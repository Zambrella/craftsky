package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
)

type RateClass string

const (
	RateClassAuth   RateClass = "auth"
	RateClassRead   RateClass = "read"
	RateClassWrite  RateClass = "write"
	RateClassSearch RateClass = "expensive_search"
	RateClassUpload RateClass = "upload"
)

type RateLimitConfig struct {
	Classes map[RateClass]ClassLimit
}

type ClassLimit struct {
	Window    time.Duration
	PerToken  int
	PerDevice int
}

type RateKeys struct {
	TokenKey string
	DeviceID string
}

type Decision struct {
	Allowed    bool
	RetryAfter time.Duration
	KeyType    string
}

type LocalRateLimiter struct {
	mu      sync.Mutex
	config  RateLimitConfig
	now     func() time.Time
	buckets map[string]bucket
}

type bucket struct {
	windowStart time.Time
	count       int
}

func NewLocalRateLimiter(config RateLimitConfig, now func() time.Time) *LocalRateLimiter {
	if now == nil {
		now = time.Now
	}
	return &LocalRateLimiter{config: config, now: now, buckets: map[string]bucket{}}
}

func (l *LocalRateLimiter) Allow(class RateClass, keys RateKeys) Decision {
	limit, ok := l.config.Classes[class]
	if !ok || limit.Window <= 0 {
		return Decision{Allowed: true}
	}
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	if keys.TokenKey != "" && limit.PerToken > 0 {
		if d := l.allowKey(now, limit.Window, "token", class, keys.TokenKey, limit.PerToken); !d.Allowed {
			return d
		}
	}
	if keys.DeviceID != "" && limit.PerDevice > 0 {
		if d := l.allowKey(now, limit.Window, "device", class, keys.DeviceID, limit.PerDevice); !d.Allowed {
			return d
		}
	}
	return Decision{Allowed: true}
}

func (l *LocalRateLimiter) allowKey(now time.Time, window time.Duration, keyType string, class RateClass, key string, max int) Decision {
	bucketKey := fmt.Sprintf("%s:%s:%s", class, keyType, key)
	b := l.buckets[bucketKey]
	if b.windowStart.IsZero() || now.Sub(b.windowStart) >= window {
		b = bucket{windowStart: now}
	}
	if b.count >= max {
		return Decision{Allowed: false, RetryAfter: b.windowStart.Add(window).Sub(now), KeyType: keyType}
	}
	b.count++
	l.buckets[bucketKey] = b
	return Decision{Allowed: true}
}

func (l *LocalRateLimiter) DebugKeys() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	keys := make([]string, 0, len(l.buckets))
	for key := range l.buckets {
		keys = append(keys, key)
	}
	return keys
}

func RateLimit(limiter *LocalRateLimiter, class RateClass, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			keys := RateKeys{DeviceID: r.Header.Get("X-Craftsky-Device-Id")}
			if sid, ok := ctxkeys.GetOAuthSessionID(r.Context()); ok {
				keys.TokenKey = sid
			}
			decision := limiter.Allow(class, keys)
			if !decision.Allowed {
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
