package main

import (
	"time"

	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/middleware"
)

func httpAdmissionConfigFromApp(cfg app.Config) HTTPAdmissionConfig {
	return HTTPAdmissionConfig{
		MaxConnections:      cfg.HTTPMaxConnections,
		MaxInFlightRequests: cfg.HTTPMaxInFlightRequests,
		ReadHeaderTimeout:   cfg.HTTPReadHeaderTimeout,
		ReadTimeout:         cfg.HTTPReadTimeout,
		WriteTimeout:        cfg.HTTPWriteTimeout,
		IdleTimeout:         cfg.HTTPIdleTimeout,
		MaxHeaderBytes:      cfg.HTTPMaxHeaderBytes,
	}
}

func handlerAdmissionConfigFromApp(cfg app.Config) HandlerAdmissionConfig {
	now := time.Now
	return HandlerAdmissionConfig{
		MaxInFlightRequests: cfg.HTTPMaxInFlightRequests,
		TrustedProxyCIDRs:   append([]string(nil), cfg.HTTPTrustedProxyCIDRs...),
		IPv6PrefixBits:      cfg.HTTPClientIPv6PrefixBits,
		OuterRateLimits: middleware.RateLimitConfig{Classes: map[middleware.RateClass]middleware.ClassLimit{
			middleware.RateClassOuter: {
				Window:    cfg.HTTPOuterRateWindow,
				Global:    cfg.HTTPOuterGlobalLimit,
				PerClient: cfg.HTTPOuterClientLimit,
			},
		}},
		LimiterCapacity: cfg.HTTPLimiterCapacity,
		LimiterIdleTTL:  cfg.HTTPLimiterIdleTTL,
		Now:             now,
	}
}
