package middleware

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type RateClass string

const (
	RateClassOuter       RateClass = "outer"
	RateClassAuth        RateClass = "auth"
	RateClassRead        RateClass = "read"
	RateClassWrite       RateClass = "write"
	RateClassSearch      RateClass = "expensive_search"
	RateClassUpload      RateClass = "upload"
	RateClassLinkPreview RateClass = "link_preview"
)

type RateLimitConfig struct {
	Classes map[RateClass]ClassLimit
}

type ClassLimit struct {
	Window    time.Duration
	Global    int
	PerClient int
	PerToken  int
	PerDevice int
}

type RateKeys struct {
	GlobalKey string
	ClientKey string
	TokenKey  string
	DeviceID  string
}

type Decision struct {
	Allowed    bool
	RetryAfter time.Duration
	KeyType    string
}

type Limiter interface {
	Allow(RateClass, RateKeys) Decision
}

type LocalLimiterOptions struct {
	Capacity int
	IdleTTL  time.Duration
	Now      func() time.Time
}

type LocalRateLimiter struct {
	mu          sync.Mutex
	config      RateLimitConfig
	now         func() time.Time
	capacity    int
	idleTTL     time.Duration
	nextCleanup time.Time
	buckets     map[string]bucket
	overflow    map[string]bucket
}

type bucket struct {
	windowStart time.Time
	window      time.Duration
	lastSeen    time.Time
	count       int
}

func NewLocalRateLimiter(config RateLimitConfig, now func() time.Time) *LocalRateLimiter {
	idleTTL := 10 * time.Minute
	for _, limit := range config.Classes {
		if candidate := 2 * limit.Window; candidate > idleTTL {
			idleTTL = candidate
		}
	}
	limiter, err := NewBoundedLocalRateLimiter(config, LocalLimiterOptions{
		Capacity: 4096,
		IdleTTL:  idleTTL,
		Now:      now,
	})
	if err != nil {
		panic(err)
	}
	return limiter
}

func NewBoundedLocalRateLimiter(config RateLimitConfig, options LocalLimiterOptions) (*LocalRateLimiter, error) {
	if options.Capacity <= 0 {
		return nil, errors.New("rate limiter capacity must be positive")
	}
	if options.IdleTTL <= 0 {
		return nil, errors.New("rate limiter idle TTL must be positive")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	overflow := make(map[string]bucket)
	for class, limit := range config.Classes {
		if limit.Window <= 0 {
			return nil, fmt.Errorf("rate limiter window for %s must be positive", class)
		}
		limits := []struct {
			keyType string
			value   int
		}{
			{keyType: "global", value: limit.Global},
			{keyType: "client", value: limit.PerClient},
			{keyType: "token", value: limit.PerToken},
			{keyType: "device", value: limit.PerDevice},
		}
		for _, item := range limits {
			if item.value < 0 {
				return nil, fmt.Errorf("rate limiter %s limit for %s cannot be negative", item.keyType, class)
			}
			if item.value > 0 && item.keyType != "global" {
				overflow[overflowBucketKey(class, item.keyType)] = bucket{window: limit.Window}
			}
		}
	}
	return &LocalRateLimiter{
		config:   config,
		now:      options.Now,
		capacity: options.Capacity,
		idleTTL:  options.IdleTTL,
		buckets:  make(map[string]bucket),
		overflow: overflow,
	}, nil
}

func (l *LocalRateLimiter) Allow(class RateClass, keys RateKeys) Decision {
	limit, ok := l.config.Classes[class]
	if !ok || limit.Window <= 0 {
		return Decision{Allowed: true}
	}
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cleanupIdle(now)

	type requestedBucket struct {
		key       string
		keyType   string
		max       int
		overflow  bool
		newBucket bool
		value     bucket
	}
	requests := make([]requestedBucket, 0, 4)
	appendRequest := func(keyType, key string, max int) {
		if max <= 0 || key == "" {
			return
		}
		requests = append(requests, requestedBucket{
			key:     fmt.Sprintf("%s:%s:%s", class, keyType, key),
			keyType: keyType,
			max:     max,
		})
	}
	globalKey := keys.GlobalKey
	if globalKey == "" {
		globalKey = "process"
	}
	appendRequest("global", globalKey, limit.Global)
	appendRequest("client", keys.ClientKey, limit.PerClient)
	appendRequest("token", keys.TokenKey, limit.PerToken)
	appendRequest("device", keys.DeviceID, limit.PerDevice)

	available := l.capacity - len(l.buckets)
	for index := range requests {
		requested := &requests[index]
		value, exists := l.buckets[requested.key]
		if !exists {
			if available > 0 {
				requested.newBucket = true
				available--
			} else {
				requested.overflow = true
				requested.key = overflowBucketKey(class, requested.keyType)
				value = l.overflow[requested.key]
			}
		}
		value = advanceBucket(value, now, limit.Window)
		value.lastSeen = now
		requested.value = value
		if value.count >= requested.max {
			if requested.overflow {
				requested.keyType += "_overflow"
				l.overflow[requested.key] = value
			} else if !requested.newBucket {
				l.buckets[requested.key] = value
			}
			return Decision{
				Allowed:    false,
				RetryAfter: value.windowStart.Add(limit.Window).Sub(now),
				KeyType:    requested.keyType,
			}
		}
	}

	for _, requested := range requests {
		requested.value.count++
		if requested.overflow {
			l.overflow[requested.key] = requested.value
		} else {
			l.buckets[requested.key] = requested.value
		}
	}
	return Decision{Allowed: true}
}

func (l *LocalRateLimiter) cleanupIdle(now time.Time) {
	if !l.nextCleanup.IsZero() && now.Before(l.nextCleanup) && len(l.buckets) < l.capacity {
		return
	}
	for key, value := range l.buckets {
		if now.Sub(value.lastSeen) >= l.idleTTL && !now.Before(value.windowStart.Add(value.window)) {
			delete(l.buckets, key)
		}
	}
	interval := l.idleTTL / 4
	if interval > time.Minute {
		interval = time.Minute
	}
	if interval < time.Second {
		interval = time.Second
	}
	l.nextCleanup = now.Add(interval)
}

func advanceBucket(value bucket, now time.Time, window time.Duration) bucket {
	if value.windowStart.IsZero() || !now.Before(value.windowStart.Add(window)) {
		return bucket{windowStart: now, window: window, lastSeen: now}
	}
	value.window = window
	return value
}

func overflowBucketKey(class RateClass, keyType string) string {
	return fmt.Sprintf("%s:%s:overflow", class, keyType)
}

func (l *LocalRateLimiter) EntryCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
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
