package app

import (
	"container/list"
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
)

type oauthMetadataSource interface {
	ResolveAuthServerURL(context.Context, string) (string, error)
	ResolveAuthServerMetadata(context.Context, string) (*oauth.AuthServerMetadata, error)
}

type oauthAuthorityMetadataResolver interface {
	ResolveIssuer(context.Context, string) (string, error)
}

type freshOAuthAuthorityMetadataResolver struct {
	source oauthMetadataSource
}

func (resolver *freshOAuthAuthorityMetadataResolver) ResolveIssuer(ctx context.Context, pds string) (string, error) {
	if resolver == nil || resolver.source == nil {
		return "", errors.New("OAuth authority metadata resolver is unavailable")
	}
	issuer, err := resolver.source.ResolveAuthServerURL(ctx, pds)
	if err != nil {
		return "", err
	}
	metadata, err := resolver.source.ResolveAuthServerMetadata(ctx, issuer)
	if err != nil {
		return "", err
	}
	if metadata == nil || metadata.Issuer == "" {
		return "", errors.New("current OAuth authorization metadata is invalid")
	}
	return metadata.Issuer, nil
}

type oauthMetadataCacheObserver interface {
	ObserveOAuthMetadataCache(context.Context, string, string)
}

type oauthMetadataCacheKey struct {
	stage  string
	origin string
}

type oauthMetadataCacheEntry struct {
	key       oauthMetadataCacheKey
	value     string
	expiresAt time.Time
}

type oauthMetadataFlight struct {
	done    chan struct{}
	cancel  context.CancelFunc
	waiters int
	value   string
	err     error
}

type cachedOAuthAuthorityMetadataResolver struct {
	source       oauthMetadataSource
	ttl          time.Duration
	capacity     int
	fetchTimeout time.Duration
	now          func() time.Time
	observer     oauthMetadataCacheObserver

	mu          sync.Mutex
	entries     map[oauthMetadataCacheKey]*list.Element
	order       *list.List
	flights     map[oauthMetadataCacheKey]*oauthMetadataFlight
	activeLoads int
}

func newCachedOAuthAuthorityMetadataResolver(
	source oauthMetadataSource,
	ttl time.Duration,
	capacity int,
	fetchTimeout time.Duration,
	now func() time.Time,
	observer oauthMetadataCacheObserver,
) (*cachedOAuthAuthorityMetadataResolver, error) {
	if source == nil || ttl <= 0 || ttl > maxOAuthAuthorityMetadataCacheTTL || capacity <= 0 || fetchTimeout <= 0 {
		return nil, errors.New("OAuth authority metadata cache configuration is invalid")
	}
	if now == nil {
		now = time.Now
	}
	return &cachedOAuthAuthorityMetadataResolver{
		source: source, ttl: ttl, capacity: capacity, fetchTimeout: fetchTimeout,
		now: now, observer: observer, entries: make(map[oauthMetadataCacheKey]*list.Element),
		order: list.New(), flights: make(map[oauthMetadataCacheKey]*oauthMetadataFlight),
	}, nil
}

func (resolver *cachedOAuthAuthorityMetadataResolver) ResolveIssuer(ctx context.Context, pds string) (string, error) {
	issuer, err := resolver.resolve(ctx, "protected_resource", pds, func(loadCtx context.Context) (string, error) {
		return resolver.source.ResolveAuthServerURL(loadCtx, pds)
	})
	if err != nil {
		return "", err
	}
	return resolver.resolve(ctx, "authorization_server", issuer, func(loadCtx context.Context) (string, error) {
		metadata, err := resolver.source.ResolveAuthServerMetadata(loadCtx, issuer)
		if err != nil {
			return "", err
		}
		if metadata == nil || metadata.Issuer == "" {
			return "", errors.New("current OAuth authorization metadata is invalid")
		}
		return metadata.Issuer, nil
	})
}

func (resolver *cachedOAuthAuthorityMetadataResolver) resolve(
	ctx context.Context,
	stage string,
	rawOrigin string,
	load func(context.Context) (string, error),
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	origin, err := canonicalOAuthMetadataOrigin(rawOrigin)
	if err != nil {
		return "", err
	}
	key := oauthMetadataCacheKey{stage: stage, origin: origin}
	now := resolver.now()

	resolver.mu.Lock()
	if element := resolver.entries[key]; element != nil {
		entry := element.Value.(oauthMetadataCacheEntry)
		if entry.expiresAt.After(now) {
			resolver.order.MoveToFront(element)
			resolver.mu.Unlock()
			resolver.observe(ctx, stage, "hit")
			return entry.value, nil
		}
		resolver.removeElement(element)
	}
	if flight := resolver.flights[key]; flight != nil {
		flight.waiters++
		resolver.mu.Unlock()
		return resolver.waitForFlight(ctx, key, flight, "coalesced")
	}
	for len(resolver.entries)+resolver.activeLoads >= resolver.capacity && resolver.order.Len() > 0 {
		resolver.removeElement(resolver.order.Back())
	}
	if len(resolver.entries)+resolver.activeLoads >= resolver.capacity {
		resolver.mu.Unlock()
		resolver.observe(ctx, stage, "error")
		return "", errors.New("OAuth authority metadata cache capacity exceeded")
	}
	loadCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), resolver.fetchTimeout)
	flight := &oauthMetadataFlight{done: make(chan struct{}), cancel: cancel, waiters: 1}
	resolver.flights[key] = flight
	resolver.activeLoads++
	resolver.mu.Unlock()

	go func() {
		defer cancel()
		value, loadErr := load(loadCtx)
		if loadErr == nil {
			value, loadErr = canonicalOAuthMetadataOrigin(value)
		}
		resolver.mu.Lock()
		resolver.activeLoads--
		flight.value, flight.err = value, loadErr
		if resolver.flights[key] == flight {
			delete(resolver.flights, key)
			if loadErr == nil {
				resolver.insert(key, value, resolver.now().Add(resolver.ttl))
			}
		}
		close(flight.done)
		resolver.mu.Unlock()
	}()
	return resolver.waitForFlight(ctx, key, flight, "miss")
}

func (resolver *cachedOAuthAuthorityMetadataResolver) waitForFlight(
	ctx context.Context,
	key oauthMetadataCacheKey,
	flight *oauthMetadataFlight,
	result string,
) (string, error) {
	select {
	case <-ctx.Done():
		resolver.mu.Lock()
		if resolver.flights[key] == flight {
			flight.waiters--
			if flight.waiters == 0 {
				delete(resolver.flights, key)
				flight.cancel()
			}
		}
		resolver.mu.Unlock()
		resolver.observe(ctx, key.stage, "error")
		return "", ctx.Err()
	case <-flight.done:
		if flight.err != nil {
			result = "error"
		}
		resolver.observe(ctx, key.stage, result)
		return flight.value, flight.err
	}
}

func (resolver *cachedOAuthAuthorityMetadataResolver) insert(key oauthMetadataCacheKey, value string, expiresAt time.Time) {
	if element := resolver.entries[key]; element != nil {
		resolver.order.Remove(element)
	}
	element := resolver.order.PushFront(oauthMetadataCacheEntry{key: key, value: value, expiresAt: expiresAt})
	resolver.entries[key] = element
	for resolver.order.Len() > resolver.capacity {
		resolver.removeElement(resolver.order.Back())
	}
}

func (resolver *cachedOAuthAuthorityMetadataResolver) removeElement(element *list.Element) {
	if element == nil {
		return
	}
	entry := element.Value.(oauthMetadataCacheEntry)
	delete(resolver.entries, entry.key)
	resolver.order.Remove(element)
}

func (resolver *cachedOAuthAuthorityMetadataResolver) observe(ctx context.Context, stage, result string) {
	if resolver.observer != nil {
		resolver.observer.ObserveOAuthMetadataCache(ctx, stage, result)
	}
}

func canonicalOAuthMetadataOrigin(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", errors.New("invalid OAuth metadata origin")
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Path = ""
	parsed.RawPath = ""
	return parsed.String(), nil
}
