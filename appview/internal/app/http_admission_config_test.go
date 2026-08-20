package app

import (
	"strings"
	"testing"
	"time"
)

func TestHTTPAdmissionConfigDefaults(t *testing.T) {
	const base = "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n"
	cfg, err := LoadConfig(EnvDev, testConfigFile(t, base))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPMaxConnections != 512 || cfg.HTTPMaxInFlightRequests != 256 ||
		cfg.HTTPReadHeaderTimeout != 5*time.Second || cfg.HTTPReadTimeout != 90*time.Second ||
		cfg.HTTPWriteTimeout != 2*time.Minute || cfg.HTTPIdleTimeout != time.Minute ||
		cfg.HTTPMaxHeaderBytes != 32<<10 || cfg.HTTPClientIPv6PrefixBits != 64 ||
		cfg.HTTPOuterRateWindow != time.Minute || cfg.HTTPOuterGlobalLimit != 6000 ||
		cfg.HTTPOuterClientLimit != 600 || cfg.HTTPLimiterCapacity != 4096 ||
		cfg.HTTPLimiterIdleTTL != 10*time.Minute || cfg.HTTPJSONBodyReadTimeout != 10*time.Second ||
		cfg.HTTPUploadBodyReadTimeout != time.Minute || cfg.OAuthPendingAuthRequestCapacity != 4096 ||
		cfg.OAuthAuthRequestTerminalRetention != 24*time.Hour ||
		cfg.OAuthAuthRequestSweepInterval != time.Minute || cfg.OAuthAuthRequestSweepBatch != 100 {
		t.Fatalf("HTTP/auth admission defaults = %+v", cfg)
	}
}

func TestHTTPAdmissionConfigRejectsUnsafeGeometryAndTrust(t *testing.T) {
	const base = "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n"
	for _, test := range []struct {
		name     string
		override string
		want     string
	}{
		{name: "request ceiling exceeds connections", override: "HTTP_MAX_CONNECTIONS=10\nHTTP_MAX_IN_FLIGHT_REQUESTS=11\n", want: "HTTP_MAX_IN_FLIGHT_REQUESTS"},
		{name: "write timeout above maximum", override: "HTTP_WRITE_TIMEOUT=20m1ns\n", want: "HTTP_WRITE_TIMEOUT"},
		{name: "write deadline cannot cover read", override: "HTTP_READ_TIMEOUT=2m\nHTTP_WRITE_TIMEOUT=2m\n", want: "HTTP_WRITE_TIMEOUT"},
		{name: "JSON body budget exceeds read deadline", override: "HTTP_READ_TIMEOUT=5s\nHTTP_JSON_BODY_READ_TIMEOUT=6s\n", want: "HTTP_JSON_BODY_READ_TIMEOUT"},
		{name: "upload body budget exceeds read deadline", override: "HTTP_READ_TIMEOUT=30s\nHTTP_UPLOAD_BODY_READ_TIMEOUT=31s\n", want: "HTTP_UPLOAD_BODY_READ_TIMEOUT"},
		{name: "trust all IPv4", override: "HTTP_TRUSTED_PROXY_CIDRS=0.0.0.0/0\n", want: "HTTP_TRUSTED_PROXY_CIDRS"},
		{name: "trust all IPv6", override: "HTTP_TRUSTED_PROXY_CIDRS=::/0\n", want: "HTTP_TRUSTED_PROXY_CIDRS"},
		{name: "invalid trusted proxy", override: "HTTP_TRUSTED_PROXY_CIDRS=not-a-prefix\n", want: "HTTP_TRUSTED_PROXY_CIDRS"},
		{name: "zero IPv6 aggregation", override: "HTTP_CLIENT_IPV6_PREFIX_BITS=0\n", want: "HTTP_CLIENT_IPV6_PREFIX_BITS"},
		{name: "sweep batch exceeds capacity", override: "OAUTH_PENDING_AUTH_REQUEST_CAPACITY=10\nOAUTH_AUTH_REQUEST_SWEEP_BATCH=11\n", want: "OAUTH_AUTH_REQUEST_SWEEP_BATCH"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadConfig(EnvDev, testConfigFile(t, base+test.override))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("LoadConfig error = %v, want %s", err, test.want)
			}
		})
	}
}

func TestHTTPAdmissionWriteTimeoutMustCoverUploadAndMediaPut(t *testing.T) {
	const base = "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n" +
		"HTTP_READ_TIMEOUT=1m10s\n" +
		"HTTP_UPLOAD_BODY_READ_TIMEOUT=1m\n" +
		"SCHEDULED_MEDIA_PUT_TIMEOUT=30s\n"
	_, err := LoadConfig(EnvDev, testConfigFile(t, base+"HTTP_WRITE_TIMEOUT=1m35s\n"))
	if err == nil {
		t.Fatal("LoadConfig error = nil, want unsafe write-budget rejection")
	}
	for _, field := range []string{
		"HTTP_WRITE_TIMEOUT",
		"HTTP_UPLOAD_BODY_READ_TIMEOUT",
		"SCHEDULED_MEDIA_PUT_TIMEOUT",
	} {
		if !strings.Contains(err.Error(), field) {
			t.Fatalf("LoadConfig error = %v, want %s", err, field)
		}
	}
	if _, err := LoadConfig(EnvDev, testConfigFile(t, base+"HTTP_WRITE_TIMEOUT=1m35s1ns\n")); err != nil {
		t.Fatalf("LoadConfig one nanosecond above required write budget: %v", err)
	}
	if _, err := LoadConfig(EnvDev, testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n"+
		"HTTP_READ_TIMEOUT=5m\n"+
		"HTTP_UPLOAD_BODY_READ_TIMEOUT=5m\n"+
		"SCHEDULED_MEDIA_PUT_TIMEOUT=10m\n"+
		"HTTP_WRITE_TIMEOUT=20m\n")); err != nil {
		t.Fatalf("LoadConfig coherent duration maxima: %v", err)
	}
}

func TestHTTPTrustedProxyCIDRsAreCanonicalized(t *testing.T) {
	const base = "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\nTAP_WS_URL=ws://tap\n" +
		"HTTP_TRUSTED_PROXY_CIDRS=10.0.0.9/24, 2001:db8::1/48\n"
	cfg, err := LoadConfig(EnvDev, testConfigFile(t, base))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"10.0.0.0/24", "2001:db8::/48"}
	if len(cfg.HTTPTrustedProxyCIDRs) != len(want) {
		t.Fatalf("trusted proxies = %v", cfg.HTTPTrustedProxyCIDRs)
	}
	for i := range want {
		if cfg.HTTPTrustedProxyCIDRs[i] != want[i] {
			t.Fatalf("trusted proxies = %v, want %v", cfg.HTTPTrustedProxyCIDRs, want)
		}
	}
}
