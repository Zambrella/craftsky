package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/testdb"
	"strings"
	"testing"
)

type browserDeviceStore struct {
	platform string
	calls    int
}

func TestBrowserPushMigrationSupportsBrowserAlongsideNative(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	for index, platform := range []string{"ios", "android", "web"} {
		_, err := pool.Exec(t.Context(), `INSERT INTO push_installations(id,device_id,platform,fcm_token) VALUES(gen_random_uuid(),$1,$2,$3)`, platform+"-device", platform, platform+"-test-token")
		if err != nil {
			t.Fatalf("platform %d failed after production migrations: %v", index, err)
		}
	}
	_, err := pool.Exec(t.Context(), `INSERT INTO push_installations(id,device_id,platform,fcm_token) VALUES(gen_random_uuid(),'invalid-device','desktop','invalid-test-token')`)
	if err == nil {
		t.Fatal("production migration accepted unsupported platform")
	}
}

func (s *browserDeviceStore) RegisterNotificationDevice(_ context.Context, _, _, platform, _ string) (string, error) {
	s.platform = platform
	s.calls++
	return "00000000-0000-4000-8000-000000000001", nil
}
func (*browserDeviceStore) RemoveNotificationSubscription(context.Context, string, string, string) error {
	return nil
}
func TestNotificationDeviceRegistrationAcceptsSupportedPlatformsWithoutEchoingTokens(t *testing.T) {
	for _, platform := range []string{"ios", "android", "web", "desktop"} {
		t.Run(platform, func(t *testing.T) {
			store := &browserDeviceStore{}
			handler := api.RegisterNotificationDeviceHandler(store, nil)
			request := authedReq(http.MethodPost, "/v1/notifications/devices", `{"platform":"`+platform+`","token":"PRIVATE_REGISTRATION_TOKEN"}`, "did:plc:viewer")
			request = request.WithContext(middleware.WithDeviceID(request.Context(), "00000000-0000-4000-8000-000000000002"))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if platform == "desktop" {
				if response.Code != http.StatusBadRequest || store.calls != 0 {
					t.Fatalf("unsupported platform status=%d calls=%d", response.Code, store.calls)
				}
				return
			}
			if response.Code != http.StatusOK || store.platform != platform || strings.Contains(response.Body.String(), "PRIVATE_REGISTRATION_TOKEN") {
				t.Fatalf("status=%d platform=%s body=%s", response.Code, store.platform, response.Body.String())
			}
		})
	}
}
