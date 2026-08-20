package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/observability"
)

func TestV1RoutePoliciesUseExplicitAccessClasses(t *testing.T) {
	t.Parallel()

	wantOverrides := map[string]AccessClass{
		"POST /v1/auth/login":                         AccessAnonymous,
		"POST /v1/auth/handoffs/exchange":             AccessAnonymous,
		"POST /v1/auth/handoffs/confirm":              AccessAnonymous,
		"GET /v1/whoami":                              AccessAuthenticatedRecovery,
		"POST /v1/auth/logout":                        AccessAuthenticatedRecovery,
		"DELETE /v1/account-deletion/intents/{jobId}": AccessAuthenticatedRecovery,
		"POST /v1/account-deletions/{jobId}":          AccessAuthenticatedRecovery,
		"GET /v1/dev/media/{name}":                    AccessAnonymous,
		"GET /v1/dev/panic":                           AccessAnonymous,
		"POST /v1/dev/moderation/ozone-events":        AccessAnonymous,
	}

	policies := V1RoutePolicies(EnvDev, Config{
		Env:                 EnvDev,
		EnableDevModeration: true,
		DevModerationToken:  "configured",
	})
	for _, policy := range policies {
		key := policy.Method + " " + policy.PathPattern
		want := AccessCurrentMember
		if override, ok := wantOverrides[key]; ok {
			want = override
			delete(wantOverrides, key)
		}
		if !policy.AccessClass.Valid() {
			t.Errorf("%s access class = %d, want one explicit non-zero class", key, policy.AccessClass)
		}
		if policy.AccessClass != want {
			t.Errorf("%s access class = %v, want %v", key, policy.AccessClass, want)
		}
	}
	if len(wantOverrides) != 0 {
		t.Fatalf("missing explicitly classified routes: %v", wantOverrides)
	}
}

func TestV1MiddlewareDispatchesAccessClass(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name        string
		accessClass AccessClass
		rateClass   RateClass
		wantCalls   []string
	}{
		{name: "anonymous auth route", accessClass: AccessAnonymous, rateClass: RateClassAuth, wantCalls: []string{"device", "handler"}},
		{name: "anonymous dev route", accessClass: AccessAnonymous, rateClass: RateClassDevOnly, wantCalls: []string{"handler"}},
		{name: "authenticated recovery", accessClass: AccessAuthenticatedRecovery, rateClass: RateClassRead, wantCalls: []string{"recovery", "device", "handler"}},
		{name: "current member", accessClass: AccessCurrentMember, rateClass: RateClassRead, wantCalls: []string{"ordinary", "device", "member", "handler"}},
		{name: "unspecified fails closed", accessClass: 0, rateClass: RateClassRead, wantCalls: []string{"ordinary", "device", "member", "handler"}},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var calls []string
			probe := func(name string) func(http.Handler) http.Handler {
				return func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls = append(calls, name)
						next.ServeHTTP(w, r)
					})
				}
			}
			mw := v1Middleware{
				authCurrentMember: probe("ordinary"),
				authRecovery:      probe("recovery"),
				deviceID:          probe("device"),
				member:            probe("member"),
				rateLimit:         map[RateClass]func(http.Handler) http.Handler{},
				observer:          observability.New(observability.Config{}),
			}
			handler := mw.wrap(RoutePolicy{
				Method:      http.MethodGet,
				PathPattern: "/v1/probe",
				RateClass:   test.rateClass,
				BodyKind:    BodyExempt,
				AccessClass: test.accessClass,
			}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls = append(calls, "handler")
				w.WriteHeader(http.StatusNoContent)
			}))

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/probe", nil))
			if response.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want 204", response.Code)
			}
			if !reflect.DeepEqual(calls, test.wantCalls) {
				t.Fatalf("middleware calls = %v, want %v", calls, test.wantCalls)
			}
		})
	}
}

func TestAddRoutesRequiresRecoveryAuthenticationContract(t *testing.T) {
	t.Parallel()

	deps := testDeps()
	deps.AuthService = ordinaryOnlyAuthService{}
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("AddRoutes did not panic for auth service without recovery authentication")
		}
	}()
	AddRoutes(context.Background(), http.NewServeMux(), deps)
}

type ordinaryOnlyAuthService struct{}

func (ordinaryOnlyAuthService) Authenticate(context.Context, string) (auth.AuthInfo, error) {
	return auth.AuthInfo{}, auth.ErrAuthTokenInvalid
}

var _ auth.AuthService = ordinaryOnlyAuthService{}
