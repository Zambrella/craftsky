package routes

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/testdb"
)

func TestInstagramVerificationRoutePoliciesRequireAuthDeviceAndCurrentMember(t *testing.T) {
	t.Parallel()

	want := map[string]struct {
		rate RateClass
		body BodyKind
	}{
		"POST /v1/migrations/instagram/verifications":                          {rate: RateClassWrite, body: BodyDefaultJSON},
		"GET /v1/migrations/instagram/verifications/current":                   {rate: RateClassRead, body: BodyNoBody},
		"GET /v1/migrations/instagram/verifications/{verificationId}":          {rate: RateClassRead, body: BodyNoBody},
		"DELETE /v1/migrations/instagram/verifications/{verificationId}":       {rate: RateClassWrite, body: BodyNoBody},
		"POST /v1/migrations/instagram/verifications/{verificationId}/confirm": {rate: RateClassWrite, body: BodyDefaultJSON},
		"GET /v1/migrations/instagram/account":                                 {rate: RateClassRead, body: BodyNoBody},
		"DELETE /v1/migrations/instagram/account":                              {rate: RateClassWrite, body: BodyNoBody},
		"PATCH /v1/migrations/instagram/settings":                              {rate: RateClassWrite, body: BodyDefaultJSON},
		"POST /v1/migrations/instagram/imports":                                {rate: RateClassWrite, body: BodyDefaultJSON},
		"GET /v1/migrations/instagram/imports":                                 {rate: RateClassRead, body: BodyNoBody},
		"GET /v1/migrations/instagram/imports/{importId}":                      {rate: RateClassRead, body: BodyNoBody},
		"PATCH /v1/migrations/instagram/imports/{importId}":                    {rate: RateClassWrite, body: BodyDefaultJSON},
		"DELETE /v1/migrations/instagram/imports/{importId}":                   {rate: RateClassWrite, body: BodyNoBody},
		"GET /v1/migrations/instagram/suggestions":                             {rate: RateClassRead, body: BodyNoBody},
		"POST /v1/migrations/instagram/suggestions/{suggestionId}/accept":      {rate: RateClassWrite, body: BodyNoBody},
		"DELETE /v1/migrations/instagram/suggestions/{suggestionId}":           {rate: RateClassWrite, body: BodyNoBody},
	}
	for _, policy := range V1RoutePolicies(app.EnvProd, app.Config{Env: app.EnvProd}) {
		key := policy.Method + " " + policy.PathPattern
		expected, ok := want[key]
		if !ok {
			continue
		}
		if !policy.AuthRequired || !policy.CurrentMemberRequired || policy.RateClass != expected.rate || policy.BodyKind != expected.body {
			t.Errorf("policy %s = %+v", key, policy)
		}
		delete(want, key)
	}
	for missing := range want {
		t.Errorf("missing Instagram route policy %s", missing)
	}
}

func TestInstagramVerificationRoutesEnforceMembershipBeforeDisabledService(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
		INSERT INTO craftsky_profiles (did) VALUES ('did:plc:synthetic-current');
	`)
	disabled, err := instagram.NewVerificationService(instagram.VerificationServiceOptions{})
	if err != nil {
		t.Fatalf("disabled service: %v", err)
	}
	deps := &app.Deps{
		Config:                app.Config{Env: app.EnvDev},
		DB:                    pool,
		Logger:                slog.New(slog.NewTextHandler(io.Discard, nil)),
		AuthService:           &auth.MockAuthService{DefaultDID: "did:plc:synthetic-current"},
		InstagramMembership:   instagram.NewMembershipStore(pool),
		InstagramVerification: disabled,
	}
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	request := func(did string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/migrations/instagram/verifications", strings.NewReader("{}"))
		req.Header.Set("Authorization", "Bearer synthetic")
		req.Header.Set("X-Dev-DID", did)
		req.Header.Set("X-Craftsky-Device-Id", "synthetic-device")
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		return rr
	}

	departed := request("did:plc:synthetic-departed")
	if departed.Code != http.StatusNotFound {
		t.Fatalf("departed status = %d; body=%s", departed.Code, departed.Body.String())
	}
	var departedError envelope.Error
	if err := json.Unmarshal(departed.Body.Bytes(), &departedError); err != nil || departedError.Error != "profile_not_found" {
		t.Fatalf("departed error = %+v, %v", departedError, err)
	}

	current := request("did:plc:synthetic-current")
	if current.Code != http.StatusServiceUnavailable {
		t.Fatalf("current disabled status = %d; body=%s", current.Code, current.Body.String())
	}
	var currentError envelope.Error
	if err := json.Unmarshal(current.Body.Bytes(), &currentError); err != nil || currentError.Error != "instagram_unavailable" {
		t.Fatalf("current error = %+v, %v", currentError, err)
	}

	currentReadRequest := httptest.NewRequest(
		http.MethodGet,
		"/v1/migrations/instagram/verifications/current",
		nil,
	)
	currentReadRequest.Header.Set("Authorization", "Bearer synthetic")
	currentReadRequest.Header.Set("X-Dev-DID", "did:plc:synthetic-current")
	currentReadRequest.Header.Set("X-Craftsky-Device-Id", "synthetic-device")
	currentReadResponse := httptest.NewRecorder()
	mux.ServeHTTP(currentReadResponse, currentReadRequest)
	if currentReadResponse.Code != http.StatusServiceUnavailable {
		t.Fatalf("current read status = %d; body=%s", currentReadResponse.Code, currentReadResponse.Body.String())
	}
	var currentReadError envelope.Error
	if err := json.Unmarshal(currentReadResponse.Body.Bytes(), &currentReadError); err != nil || currentReadError.Error != "instagram_verification_unavailable" {
		t.Fatalf("current read error = %+v, %v", currentReadError, err)
	}

	deleteRequest := httptest.NewRequest(
		http.MethodDelete,
		"/v1/migrations/instagram/verifications/00000000-0000-0000-0000-000000000001",
		nil,
	)
	deleteRequest.Header.Set("Authorization", "Bearer synthetic")
	deleteRequest.Header.Set("X-Dev-DID", "did:plc:synthetic-current")
	deleteRequest.Header.Set("X-Craftsky-Device-Id", "synthetic-device")
	deleteResponse := httptest.NewRecorder()
	mux.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("privacy delete status = %d; body=%s", deleteResponse.Code, deleteResponse.Body.String())
	}
}

func TestInstagramChallengeRouteUsesSharedDIDDeviceAndIPLimits(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
		INSERT INTO craftsky_profiles (did) VALUES ('did:plc:synthetic-current');
		CREATE TABLE instagram_rate_limit_buckets (
			bucket_scope TEXT NOT NULL,
			key_version SMALLINT NOT NULL,
			key_digest BYTEA NOT NULL CHECK (octet_length(key_digest) = 32),
			window_start TIMESTAMPTZ NOT NULL,
			window_end TIMESTAMPTZ NOT NULL,
			count INTEGER NOT NULL CHECK (count >= 0),
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			PRIMARY KEY (bucket_scope, key_version, key_digest, window_start)
		);
	`)
	disabled, err := instagram.NewVerificationService(instagram.VerificationServiceOptions{})
	if err != nil {
		t.Fatalf("disabled service: %v", err)
	}
	now := time.Date(2026, 7, 19, 19, 30, 0, 0, time.UTC)
	limiter, err := instagram.NewPostgresRateLimiter(pool, []byte("synthetic-route-rate-key-32-bytes-long"), func() time.Time { return now })
	if err != nil {
		t.Fatalf("rate limiter: %v", err)
	}
	deps := &app.Deps{
		Config: app.Config{
			Env: app.EnvDev,
			InstagramLimits: app.InstagramLimits{
				ChallengeDIDPer15Minutes:    1,
				ChallengeDevicePer15Minutes: 1,
				ChallengeIPPer15Minutes:     1,
			},
		},
		DB:                    pool,
		Logger:                slog.New(slog.NewTextHandler(io.Discard, nil)),
		AuthService:           &auth.MockAuthService{DefaultDID: "did:plc:synthetic-current"},
		InstagramMembership:   instagram.NewMembershipStore(pool),
		InstagramRateLimiter:  limiter,
		InstagramVerification: disabled,
	}
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, deps)

	request := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/migrations/instagram/verifications", strings.NewReader("{}"))
		req.Header.Set("Authorization", "Bearer synthetic")
		req.Header.Set("X-Dev-DID", "did:plc:synthetic-current")
		req.Header.Set("X-Craftsky-Device-Id", "synthetic-device")
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		return rr
	}

	if first := request(); first.Code != http.StatusServiceUnavailable {
		t.Fatalf("first status = %d; body=%s", first.Code, first.Body.String())
	}
	second := request()
	if second.Code != http.StatusTooManyRequests || second.Header().Get("Retry-After") == "" {
		t.Fatalf("second response = status %d retry %q body=%s", second.Code, second.Header().Get("Retry-After"), second.Body.String())
	}
	var body envelope.Error
	if err := json.Unmarshal(second.Body.Bytes(), &body); err != nil || body.Error != "rate_limited" {
		t.Fatalf("second error = %+v, %v", body, err)
	}
}

func TestInstagramWebhookRoutesAreAbsentUntilCompleteHandlerIsWired(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	disabledMux := http.NewServeMux()
	AddRoutes(context.Background(), disabledMux, &app.Deps{Config: app.Config{Env: app.EnvDev}, Logger: logger})
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		rr := httptest.NewRecorder()
		disabledMux.ServeHTTP(rr, httptest.NewRequest(method, "/integrations/instagram/webhook", nil))
		if rr.Code != http.StatusNotFound {
			t.Fatalf("disabled %s status = %d, want 404", method, rr.Code)
		}
	}

	calls := make(map[string]int)
	enabledMux := http.NewServeMux()
	AddRoutes(context.Background(), enabledMux, &app.Deps{
		Config: app.Config{Env: app.EnvDev},
		Logger: logger,
		InstagramWebhook: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls[r.Method]++
			w.WriteHeader(http.StatusOK)
		}),
	})
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		rr := httptest.NewRecorder()
		enabledMux.ServeHTTP(rr, httptest.NewRequest(method, "/integrations/instagram/webhook", nil))
		if rr.Code != http.StatusOK || calls[method] != 1 {
			t.Fatalf("enabled %s = status %d calls %d", method, rr.Code, calls[method])
		}
	}
}
