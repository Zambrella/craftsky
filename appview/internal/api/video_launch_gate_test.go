package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"social.craftsky/appview/internal/api"
)

func TestVideoLaunchDisabledHandlerAlwaysRefusesOperation(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		request := httptest.NewRequest(method, "/v1/blobs/videos/authorization", nil)
		response := httptest.NewRecorder()
		api.VideoLaunchDisabledHandler().ServeHTTP(response, request)
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("method=%s status=%d body=%s", method, response.Code, response.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["error"] != "video_disabled" {
			t.Fatalf("method=%s body=%v err=%v", method, body, err)
		}
	}
}
