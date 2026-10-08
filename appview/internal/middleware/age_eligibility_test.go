package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/middleware"
)

type restrictedEligibilityReader struct{ restricted bool }

func (reader restrictedEligibilityReader) Restricted(context.Context, syntax.DID) (bool, error) {
	return reader.restricted, nil
}

func TestAgeEligibilityEnforcementDeniesOrdinaryAndPreservesRetainedRoutes(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	for _, test := range []struct {
		name     string
		retained bool
		want     int
	}{
		{name: "ordinary product route", want: http.StatusForbidden},
		{name: "retained safety route", retained: true, want: http.StatusNoContent},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/v1/test", nil)
			request = request.WithContext(middleware.WithDID(request.Context(), syntax.DID("did:plc:alice")))
			response := httptest.NewRecorder()
			middleware.AgeEligibilityEnforcement(restrictedEligibilityReader{restricted: true}, test.retained, nil)(next).ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status=%d body=%s want=%d", response.Code, response.Body.String(), test.want)
			}
		})
	}
}
