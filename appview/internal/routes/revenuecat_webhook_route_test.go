package routes

import (
	"net/http"
	"testing"
)

type routePatternRecorder struct {
	patterns []string
}

func (recorder *routePatternRecorder) Handle(pattern string, _ http.Handler) {
	recorder.patterns = append(recorder.patterns, pattern)
}

func TestRevenueCatWebhookRouteIsPublicAndConditional(t *testing.T) {
	for _, test := range []struct {
		name    string
		handler http.Handler
		want    int
	}{
		{name: "disabled", want: 0},
		{name: "configured", handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), want: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := &routePatternRecorder{}
			registerRevenueCatWebhookRoute(recorder, func(handler http.Handler) http.Handler { return handler }, test.handler)
			if len(recorder.patterns) != test.want {
				t.Fatalf("registered patterns = %v, want %d", recorder.patterns, test.want)
			}
			if test.want == 1 && recorder.patterns[0] != "POST /integrations/revenuecat/webhook" {
				t.Fatalf("RevenueCat route = %q", recorder.patterns[0])
			}
		})
	}
}
