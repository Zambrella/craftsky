package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/integrations/instagrammeta"
)

// InstagramPersistentLimiter is the narrow shared-counter contract used at
// request boundaries. PostgresRateLimiter is the production implementation.
type InstagramPersistentLimiter interface {
	Key(instagram.RateLimitScope, []byte) (instagram.RateLimitKey, error)
	Allow(context.Context, instagram.RateLimitKey, time.Duration, int) (instagram.RateLimitDecision, error)
}

type InstagramRateIdentity uint8

const (
	InstagramRateIdentityDID InstagramRateIdentity = iota + 1
	InstagramRateIdentityDevice
	InstagramRateIdentityClientIP
	InstagramRateIdentityGlobal
)

type InstagramRateLimitRule struct {
	Scope    instagram.RateLimitScope
	Identity InstagramRateIdentity
	Window   time.Duration
	Limit    int
}

// InstagramWebhookRateLimiter adapts the shared keyed limiter to the Meta
// handler's fixed pre-auth source-IP and post-signature global boundaries.
type InstagramWebhookRateLimiter struct {
	limiter                    InstagramPersistentLimiter
	trustedProxyCIDRs          []netip.Prefix
	ipPerMinute                int
	globalPerMinute            int
	invalidIPPerFifteenMinutes int
}

var _ instagrammeta.WebhookRequestLimiter = (*InstagramWebhookRateLimiter)(nil)
var _ instagrammeta.WebhookInvalidRedemptionLimiter = (*instagramWebhookInvalidRedemptionLimiter)(nil)

func NewInstagramWebhookRateLimiter(limiter InstagramPersistentLimiter, trustedProxyCIDRs []netip.Prefix, ipPerMinute, globalPerMinute, invalidIPPerFifteenMinutes int) (*InstagramWebhookRateLimiter, error) {
	if limiter == nil || ipPerMinute <= 0 || globalPerMinute <= 0 || invalidIPPerFifteenMinutes <= 0 {
		return nil, errors.New("Instagram webhook rate limiter configuration is incomplete")
	}
	return &InstagramWebhookRateLimiter{
		limiter:                    limiter,
		trustedProxyCIDRs:          append([]netip.Prefix(nil), trustedProxyCIDRs...),
		ipPerMinute:                ipPerMinute,
		globalPerMinute:            globalPerMinute,
		invalidIPPerFifteenMinutes: invalidIPPerFifteenMinutes,
	}, nil
}

func (l *InstagramWebhookRateLimiter) AllowSourceIP(ctx context.Context, r *http.Request) (instagrammeta.WebhookLimitDecision, error) {
	clientIP, err := TrustedClientIP(r, l.trustedProxyCIDRs)
	if err != nil {
		return instagrammeta.WebhookLimitDecision{}, err
	}
	return l.allow(ctx, instagram.RateLimitWebhookIP, []byte(clientIP.String()), l.ipPerMinute)
}

func (l *InstagramWebhookRateLimiter) AllowGlobal(ctx context.Context) (instagrammeta.WebhookLimitDecision, error) {
	return l.allow(ctx, instagram.RateLimitWebhookGlobal, nil, l.globalPerMinute)
}

// InvalidRedemptionSourceIP resolves the same trusted client identity as the
// pre-auth ingress bucket, then immediately reduces it to a keyed digest. The
// returned object retains no raw IP and consumes no quota until the durable
// sink classifies a newly inserted event as an invalid redemption.
func (l *InstagramWebhookRateLimiter) InvalidRedemptionSourceIP(r *http.Request) (instagrammeta.WebhookInvalidRedemptionLimiter, error) {
	clientIP, err := TrustedClientIP(r, l.trustedProxyCIDRs)
	if err != nil {
		return nil, err
	}
	key, err := l.limiter.Key(instagram.RateLimitInvalidRedemptionIP, []byte(clientIP.String()))
	if err != nil {
		return nil, err
	}
	return &instagramWebhookInvalidRedemptionLimiter{
		limiter: l.limiter,
		key:     key,
		limit:   l.invalidIPPerFifteenMinutes,
	}, nil
}

type instagramWebhookInvalidRedemptionLimiter struct {
	limiter InstagramPersistentLimiter
	key     instagram.RateLimitKey
	limit   int
}

func (l *instagramWebhookInvalidRedemptionLimiter) AllowInvalidRedemption(ctx context.Context) (instagrammeta.WebhookLimitDecision, error) {
	decision, err := l.limiter.Allow(ctx, l.key, 15*time.Minute, l.limit)
	if err != nil {
		return instagrammeta.WebhookLimitDecision{}, err
	}
	return instagrammeta.WebhookLimitDecision{Allowed: decision.Allowed, RetryAfter: decision.RetryAfter}, nil
}

func (*instagramWebhookInvalidRedemptionLimiter) String() string {
	return "Instagram invalid-redemption source limiter [REDACTED]"
}

func (l *instagramWebhookInvalidRedemptionLimiter) GoString() string {
	return l.String()
}

func (l *InstagramWebhookRateLimiter) allow(ctx context.Context, scope instagram.RateLimitScope, identifier []byte, limit int) (instagrammeta.WebhookLimitDecision, error) {
	key, err := l.limiter.Key(scope, identifier)
	if err != nil {
		return instagrammeta.WebhookLimitDecision{}, err
	}
	decision, err := l.limiter.Allow(ctx, key, time.Minute, limit)
	if err != nil {
		return instagrammeta.WebhookLimitDecision{}, err
	}
	return instagrammeta.WebhookLimitDecision{Allowed: decision.Allowed, RetryAfter: decision.RetryAfter}, nil
}

// InstagramPersistentRateLimit applies all supplied shared fixed-window
// buckets. Raw identifiers are handed directly to the keyed limiter and are
// never logged or retained by this middleware.
func InstagramPersistentRateLimit(limiter InstagramPersistentLimiter, rules []InstagramRateLimitRule, trustedProxyCIDRs []netip.Prefix, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter == nil {
				writeInstagramLimiterUnavailable(w, r, logger)
				return
			}
			for _, rule := range rules {
				identifier, err := instagramRateIdentifier(r, rule.Identity, trustedProxyCIDRs)
				if err != nil {
					writeInstagramLimiterUnavailable(w, r, logger)
					return
				}
				key, err := limiter.Key(rule.Scope, identifier)
				if err != nil {
					writeInstagramLimiterUnavailable(w, r, logger)
					return
				}
				decision, err := limiter.Allow(r.Context(), key, rule.Window, rule.Limit)
				if err != nil {
					writeInstagramLimiterUnavailable(w, r, logger)
					return
				}
				if !decision.Allowed {
					retrySeconds := int((decision.RetryAfter + time.Second - 1) / time.Second)
					if retrySeconds < 1 {
						retrySeconds = 1
					}
					w.Header().Set("Retry-After", strconv.Itoa(retrySeconds))
					if logger != nil {
						logger.Warn("Instagram operation rate limited",
							slog.String("scope", string(rule.Scope)),
							slog.String("run_id", GetRunID(r.Context())))
					}
					envelope.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests", GetRunID(r.Context()), nil)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func instagramRateIdentifier(r *http.Request, identity InstagramRateIdentity, trustedProxyCIDRs []netip.Prefix) ([]byte, error) {
	switch identity {
	case InstagramRateIdentityDID:
		did, ok := GetDID(r.Context())
		if !ok {
			return nil, errors.New("authenticated DID unavailable")
		}
		return []byte(did.String()), nil
	case InstagramRateIdentityDevice:
		deviceID, ok := GetDeviceID(r.Context())
		if !ok || deviceID == "" {
			return nil, errors.New("device identifier unavailable")
		}
		return []byte(deviceID), nil
	case InstagramRateIdentityClientIP:
		clientIP, err := TrustedClientIP(r, trustedProxyCIDRs)
		if err != nil {
			return nil, err
		}
		return []byte(clientIP.String()), nil
	case InstagramRateIdentityGlobal:
		return nil, nil
	default:
		return nil, errors.New("Instagram rate identity unavailable")
	}
}

func writeInstagramLimiterUnavailable(w http.ResponseWriter, r *http.Request, logger *slog.Logger) {
	if logger != nil {
		logger.Error("Instagram persistent rate limiter unavailable",
			slog.String("run_id", GetRunID(r.Context())))
	}
	envelope.WriteError(w, http.StatusServiceUnavailable, "instagram_unavailable", "Instagram migration unavailable", GetRunID(r.Context()), nil)
}

// TrustedClientIP returns the socket peer unless that peer belongs to an
// explicitly configured proxy CIDR. Only then does it inspect Forwarded (or
// the legacy X-Forwarded-For fallback) and walk right-to-left to select the
// first untrusted hop.
func TrustedClientIP(r *http.Request, trustedProxyCIDRs []netip.Prefix) (netip.Addr, error) {
	peer, err := parsePeerAddr(r.RemoteAddr)
	if err != nil {
		return netip.Addr{}, err
	}
	if !instagramTrustedProxy(peer, trustedProxyCIDRs) {
		return peer, nil
	}

	var chain []netip.Addr
	if forwarded := strings.Join(r.Header.Values("Forwarded"), ","); forwarded != "" {
		chain, err = parseForwardedForChain(forwarded)
	} else if forwardedFor := strings.Join(r.Header.Values("X-Forwarded-For"), ","); forwardedFor != "" {
		chain, err = parseXForwardedForChain(forwardedFor)
	}
	// A malformed or absent trusted header deliberately collapses callers into
	// the peer bucket. It cannot grant a caller a chosen rate-limit identity.
	if err != nil || len(chain) == 0 {
		return peer, nil
	}
	chain = append(chain, peer)
	for i := len(chain) - 1; i >= 0; i-- {
		if !instagramTrustedProxy(chain[i], trustedProxyCIDRs) {
			return chain[i], nil
		}
	}
	return peer, nil
}

func parsePeerAddr(raw string) (netip.Addr, error) {
	if addrPort, err := netip.ParseAddrPort(raw); err == nil {
		return addrPort.Addr().Unmap(), nil
	}
	host, _, err := net.SplitHostPort(raw)
	if err == nil {
		raw = host
	}
	addr, err := netip.ParseAddr(strings.Trim(raw, "[]"))
	if err != nil {
		return netip.Addr{}, errors.New("request socket peer is invalid")
	}
	return addr.Unmap(), nil
}

func parseForwardedForChain(raw string) ([]netip.Addr, error) {
	elements := strings.Split(raw, ",")
	chain := make([]netip.Addr, 0, len(elements))
	for _, element := range elements {
		var value string
		for _, parameter := range strings.Split(element, ";") {
			name, candidate, ok := strings.Cut(strings.TrimSpace(parameter), "=")
			if !ok || !strings.EqualFold(name, "for") {
				continue
			}
			if value != "" {
				return nil, errors.New("ambiguous Forwarded for parameter")
			}
			value = strings.TrimSpace(candidate)
		}
		if value == "" {
			return nil, errors.New("Forwarded element has no for parameter")
		}
		if strings.HasPrefix(value, `"`) || strings.HasSuffix(value, `"`) {
			unquoted, err := strconv.Unquote(value)
			if err != nil {
				return nil, errors.New("invalid quoted Forwarded value")
			}
			value = unquoted
		}
		addr, err := parseForwardedAddr(value)
		if err != nil {
			return nil, err
		}
		chain = append(chain, addr)
	}
	return chain, nil
}

func parseXForwardedForChain(raw string) ([]netip.Addr, error) {
	values := strings.Split(raw, ",")
	chain := make([]netip.Addr, 0, len(values))
	for _, value := range values {
		addr, err := parseForwardedAddr(strings.TrimSpace(value))
		if err != nil {
			return nil, err
		}
		chain = append(chain, addr)
	}
	return chain, nil
}

func parseForwardedAddr(raw string) (netip.Addr, error) {
	if raw == "" || strings.EqualFold(raw, "unknown") || strings.HasPrefix(raw, "_") {
		return netip.Addr{}, errors.New("Forwarded address is not an IP literal")
	}
	if addrPort, err := netip.ParseAddrPort(raw); err == nil {
		return addrPort.Addr().Unmap(), nil
	}
	if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		raw = strings.TrimSuffix(strings.TrimPrefix(raw, "["), "]")
	}
	addr, err := netip.ParseAddr(raw)
	if err != nil {
		return netip.Addr{}, errors.New("Forwarded address is invalid")
	}
	return addr.Unmap(), nil
}

func instagramTrustedProxy(addr netip.Addr, prefixes []netip.Prefix) bool {
	addr = addr.Unmap()
	for _, prefix := range prefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
