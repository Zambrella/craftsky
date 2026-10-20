package auth_test

import (
	"context"
	"crypto/sha256"
	"net/http/httptest"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ctxkeys"
	"strings"
	"testing"
)

type chatAuthorizationRecorder struct {
	calls int
	info  auth.AuthInfo
	push  *bool
}

func (recorder *chatAuthorizationRecorder) StartChatAuthorization(_ context.Context, info auth.AuthInfo, bearer string, mode auth.HandoffMode, loopback, device string, push *bool) (string, error) {
	recorder.calls++
	recorder.info = info
	recorder.push = push
	if bearer != "synthetic-bearer" || mode != auth.HandoffVerifiedLink || loopback != "" || device != "synthetic-device" {
		panic("invalid server authority")
	}
	return "https://auth.example/authorize?request_uri=urn:synthetic", nil
}

// IT-019 / RULE-004: the consumer can choose a reviewed handoff mode and
// initial push preference, never a different account, audience or return URL.
func TestChatAuthorizationCannotSelectOwnerOrArbitraryTarget(t *testing.T) {
	recorder := &chatAuthorizationRecorder{}
	handler := auth.ChatAuthorizationHandler(recorder, false)
	for _, body := range []string{
		`{"handoffMode":"verified_link","pushEnabled":false}`,
		`{"handoffMode":"verified_link","owner":"did:plc:other"}`,
		`{"handoffMode":"verified_link","audience":"did:web:evil.example"}`,
		`{"handoffMode":"verified_link","returnUrl":"https://evil.example"}`,
		`{"handoffMode":"loopback","loopbackRedirectUri":"https://evil.example"}`,
		`{"handoffMode":"verified_link","loopbackRedirectUri":"http://127.0.0.1:1234"}`,
		`{"handoffMode":"dev_scheme"}`,
	} {
		r := httptest.NewRequest("POST", "/v1/chat/authorization", strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer synthetic-bearer")
		ctx := ctxkeys.WithDID(r.Context(), "did:plc:authenticated-owner")
		ctx = ctxkeys.WithOAuthSessionID(ctx, "synthetic-parent")
		ctx = ctxkeys.WithDeviceID(ctx, "synthetic-device")
		ctx = ctxkeys.WithChildTokenHash(ctx, sha256.Sum256([]byte("synthetic-bearer")))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r.WithContext(ctx))
		want := 400
		if strings.Contains(body, `"pushEnabled"`) {
			want = 200
		}
		if w.Code != want {
			t.Fatalf("status=%d expected=%d", w.Code, want)
		}
	}
	if recorder.calls != 1 || recorder.info.DID != "did:plc:authenticated-owner" || recorder.info.SessionID != "synthetic-parent" || recorder.push == nil || *recorder.push {
		t.Fatal("authorization did not use exact server authority")
	}
}
