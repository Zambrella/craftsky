package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/routes"
)

type HandlerAdmissionConfig struct {
	MaxInFlightRequests int
	TrustedProxyCIDRs   []string
	IPv6PrefixBits      int
	OuterRateLimits     middleware.RateLimitConfig
	LimiterCapacity     int
	LimiterIdleTTL      time.Duration
	Now                 func() time.Time
}

func DefaultHandlerAdmissionConfig() HandlerAdmissionConfig {
	return HandlerAdmissionConfig{
		MaxInFlightRequests: defaultMaxInFlightRequest,
		IPv6PrefixBits:      64,
		OuterRateLimits: middleware.RateLimitConfig{Classes: map[middleware.RateClass]middleware.ClassLimit{
			middleware.RateClassOuter: {
				Window:    time.Minute,
				Global:    6000,
				PerClient: 600,
			},
		},
		},
		LimiterCapacity: 4096,
		LimiterIdleTTL:  10 * time.Minute,
		Now:             time.Now,
	}
}

// NewServer constructs the App View's HTTP handler. main.go wraps it in
// a *http.Server; this function stays focused on routing and middleware.
//
// Middleware stack (outside-in):
//
//	Logging (assigns run_id, logs every request)
//	HTTPMetrics (records route/status/duration/size)
//	Recovery (turns panics into safe v1 error envelopes where possible)
//	Global request concurrency
//	Trusted client-address and global/IP rate admission
//	ExpectedHost (rejects untrusted authority before route/CORS/auth/body work)
//	[canonical-path validation added by AV-032]
//	CORS (origin check, preflight handling)
//	mux (routing — Authenticated is applied per-route)
func NewServer(ctx context.Context, deps *app.Deps) http.Handler {
	handler, err := NewServerWithAdmission(ctx, deps, DefaultHandlerAdmissionConfig())
	if err != nil {
		panic(fmt.Sprintf("invalid default HTTP admission configuration: %v", err))
	}
	return handler
}

func NewServerWithAdmission(ctx context.Context, deps *app.Deps, cfg HandlerAdmissionConfig) (http.Handler, error) {
	if deps == nil {
		return nil, fmt.Errorf("AppView dependencies are required")
	}
	routeDeps := app.RouteDependencies(deps)
	resolver, err := middleware.NewClientAddressResolver(cfg.TrustedProxyCIDRs, cfg.IPv6PrefixBits)
	if err != nil {
		return nil, err
	}
	limiter, err := middleware.NewBoundedLocalRateLimiter(cfg.OuterRateLimits, middleware.LocalLimiterOptions{
		Capacity: cfg.LimiterCapacity,
		IdleTTL:  cfg.LimiterIdleTTL,
		Now:      cfg.Now,
	})
	if err != nil {
		return nil, err
	}
	outerRate, err := middleware.OuterRateLimit(resolver, limiter, deps.Logger)
	if err != nil {
		return nil, err
	}
	concurrency, err := middleware.RequestConcurrencyLimit(cfg.MaxInFlightRequests, deps.Logger)
	if err != nil {
		return nil, err
	}

	catalogue, err := routes.NewV1Catalogue(routes.V1RoutePolicies(routeDeps.Config.Env, routeDeps.Config))
	if err != nil {
		return nil, err
	}
	mux := routes.NewPolicyMux(http.NewServeMux(), catalogue)
	routes.AddRoutes(ctx, mux, routeDeps)
	if err := mux.Validate(); err != nil {
		return nil, err
	}

	var h http.Handler = mux
	h = middleware.CORS(routeDeps.Config.AllowedOrigins, catalogue)(h)
	h = catalogue.RoutingHandler(h)
	if len(routeDeps.Config.ExpectedHosts) > 0 {
		h = middleware.ExpectedHost(middleware.ExpectedHostPolicy{
			Authorities:  routeDeps.Config.ExpectedHosts,
			AllowAnyPort: routeDeps.Config.ExpectedHostAllowAnyPort,
		})(h)
	}
	h = outerRate(h)
	h = concurrency(h)
	h = middleware.Recovery(deps.Logger, deps.Observability)(h)
	h = middleware.HTTPMetrics(deps.Observability)(h)
	h = middleware.Logging(deps.Logger)(h)
	return h, nil
}
