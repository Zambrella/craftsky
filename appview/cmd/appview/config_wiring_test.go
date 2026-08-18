package main

import (
	"testing"
	"time"

	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/middleware"
)

func TestHTTPAdmissionConfigFromAppConfig(t *testing.T) {
	cfg := app.Config{
		HTTPMaxConnections:       41,
		HTTPMaxInFlightRequests:  23,
		HTTPReadHeaderTimeout:    time.Second,
		HTTPReadTimeout:          2 * time.Second,
		HTTPWriteTimeout:         3 * time.Second,
		HTTPIdleTimeout:          4 * time.Second,
		HTTPMaxHeaderBytes:       8192,
		HTTPTrustedProxyCIDRs:    []string{"10.0.0.0/8"},
		HTTPClientIPv6PrefixBits: 56,
		HTTPOuterRateWindow:      5 * time.Second,
		HTTPOuterGlobalLimit:     17,
		HTTPOuterClientLimit:     7,
		HTTPLimiterCapacity:      13,
		HTTPLimiterIdleTTL:       6 * time.Second,
	}

	httpCfg := httpAdmissionConfigFromApp(cfg)
	if httpCfg.MaxConnections != 41 || httpCfg.MaxInFlightRequests != 23 ||
		httpCfg.ReadHeaderTimeout != time.Second || httpCfg.ReadTimeout != 2*time.Second ||
		httpCfg.WriteTimeout != 3*time.Second || httpCfg.IdleTimeout != 4*time.Second ||
		httpCfg.MaxHeaderBytes != 8192 {
		t.Fatalf("HTTP config = %+v", httpCfg)
	}
	handlerCfg := handlerAdmissionConfigFromApp(cfg)
	outer := handlerCfg.OuterRateLimits.Classes[middleware.RateClassOuter]
	if handlerCfg.MaxInFlightRequests != 23 || len(handlerCfg.TrustedProxyCIDRs) != 1 ||
		handlerCfg.IPv6PrefixBits != 56 || outer.Window != 5*time.Second ||
		outer.Global != 17 || outer.PerClient != 7 || handlerCfg.LimiterCapacity != 13 ||
		handlerCfg.LimiterIdleTTL != 6*time.Second {
		t.Fatalf("handler config = %+v", handlerCfg)
	}
}
