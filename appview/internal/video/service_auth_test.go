package video

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
)

type fakeSessionRunner struct {
	session   *oauth.ClientSession
	owner     syntax.DID
	sessionID string
	calls     int
}

func (f *fakeSessionRunner) WithActiveSession(ctx context.Context, owner syntax.DID, sessionID string, operation auth.OAuthSessionOperation) error {
	f.calls++
	f.owner = owner
	f.sessionID = sessionID
	return operation(ctx, f.session)
}

func TestUploadAuthorizationIssuer_IssuesOnlyPurposeBoundAuthorization(t *testing.T) {
	t.Parallel()
	fixedNow := time.Date(2026, 9, 3, 12, 0, 0, 987_000_000, time.UTC)
	runner := &fakeSessionRunner{session: &oauth.ClientSession{Data: &oauth.ClientSessionData{
		HostURL: "https://pds.example:8443",
	}}}
	var gotAudience, gotMethod string
	var gotExpiry int64
	issuer, err := NewUploadAuthorizationIssuer(UploadAuthorizationIssuerOptions{
		Sessions:  runner,
		PDSClient: &http.Client{},
		Now:       func() time.Time { return fixedNow },
		Lifetime:  45 * time.Minute,
		request: func(_ context.Context, session *oauth.ClientSession, audience string, expiry int64, method string) (string, error) {
			if session != runner.session {
				t.Fatal("request escaped the fenced session")
			}
			gotAudience, gotExpiry, gotMethod = audience, expiry, method
			return "service-jwt", nil
		},
	})
	if err != nil {
		t.Fatalf("NewUploadAuthorizationIssuer: %v", err)
	}

	authorization, err := issuer.IssueUpload(context.Background(), syntax.DID("did:plc:alice"), "session-alice")
	if err != nil {
		t.Fatalf("IssueUpload: %v", err)
	}
	wantExpiry := time.Unix(fixedNow.Add(30*time.Minute).Unix(), 0).UTC()
	if authorization.Token != "service-jwt" || !authorization.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("authorization = %+v, want expiry %s", authorization, wantExpiry)
	}
	if runner.calls != 1 || runner.owner != syntax.DID("did:plc:alice") || runner.sessionID != "session-alice" {
		t.Fatalf("runner calls=%d owner=%q session=%q", runner.calls, runner.owner, runner.sessionID)
	}
	if gotAudience != "did:web:pds.example" || gotMethod != "com.atproto.repo.uploadBlob" || gotExpiry != wantExpiry.Unix() {
		t.Fatalf("claims audience=%q method=%q expiry=%d", gotAudience, gotMethod, gotExpiry)
	}
}

func TestUploadAuthorizationIssuer_FailsClosedWithoutSensitiveDetail(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		hostURL    string
		token      string
		requestErr error
	}{
		{name: "invalid PDS host", hostURL: "https://user:pass@pds.example/path"},
		{name: "empty token", hostURL: "https://pds.example"},
		{name: "upstream failure", hostURL: "https://pds.example", requestErr: errors.New("oauth-token dpop-key provider-message")},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			runner := &fakeSessionRunner{session: &oauth.ClientSession{Data: &oauth.ClientSessionData{HostURL: test.hostURL}}}
			issuer, err := NewUploadAuthorizationIssuer(UploadAuthorizationIssuerOptions{
				Sessions:  runner,
				PDSClient: &http.Client{},
				request: func(context.Context, *oauth.ClientSession, string, int64, string) (string, error) {
					return test.token, test.requestErr
				},
			})
			if err != nil {
				t.Fatalf("NewUploadAuthorizationIssuer: %v", err)
			}
			_, err = issuer.IssueUpload(context.Background(), syntax.DID("did:plc:alice"), "session-alice")
			if !errors.Is(err, ErrServiceAuthorizationUnavailable) {
				t.Fatalf("error = %v", err)
			}
			for _, secret := range []string{"oauth-token", "dpop-key", "provider-message", "user:pass"} {
				if strings.Contains(err.Error(), secret) {
					t.Fatalf("error leaked %q: %v", secret, err)
				}
			}
		})
	}
}
