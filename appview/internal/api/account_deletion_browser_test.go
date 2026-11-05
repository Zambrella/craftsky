package api_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"net/http/httptest"
	"strings"
	"testing"

	"social.craftsky/appview/internal/accountdeletion"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/ctxkeys"
)

type browserDeletionService struct {
	calls  int
	params accountdeletion.CreateIntentParams
}

func (s *browserDeletionService) CreateIntent(_ context.Context, p accountdeletion.CreateIntentParams) (accountdeletion.IntentResult, error) {
	s.calls++
	s.params = p
	return accountdeletion.IntentResult{}, nil
}
func (*browserDeletionService) Accept(context.Context, accountdeletion.AcceptParams) error {
	return nil
}
func (*browserDeletionService) CancelIntent(context.Context, string, syntax.DID) error { return nil }

func TestBrowserDeletionIntentRejectsUnboundRedirectBeforeServiceMutation(t *testing.T) {
	origin := "http://127.0.0.1:25319"
	nonce := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	for _, target := range []string{origin + "/account-deletion/browser/reauth-complete?state=" + nonce, origin + "/auth/browser/complete?state=" + nonce, "https://evil.example/account-deletion/browser/reauth-complete?state=" + nonce} {
		service := &browserDeletionService{}
		raw, _ := json.Marshal(map[string]string{"browserRedirectUri": target})
		request := httptest.NewRequest("POST", "/v1/account-deletion/intents", strings.NewReader(string(raw)))
		ctx := ctxkeys.WithDID(request.Context(), "did:plc:alice")
		ctx = ctxkeys.WithDeviceID(ctx, "browser-device")
		response := httptest.NewRecorder()
		api.CreateAccountDeletionIntentHandler(service, origin).ServeHTTP(response, request.WithContext(ctx))
		if strings.HasPrefix(target, origin+"/account-deletion/browser/") {
			if response.Code != 201 || service.calls != 1 || service.params.BrowserRedirectURI != target {
				t.Fatalf("browser intent not forwarded: %d", response.Code)
			}
		} else if response.Code != 400 || service.calls != 0 {
			t.Fatal("unreviewed deletion callback reached service")
		}
	}
}
