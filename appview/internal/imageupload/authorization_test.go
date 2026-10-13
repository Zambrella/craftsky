package imageupload

import (
	"context"
	"errors"
	"fmt"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
	"testing"
	"time"
)

type sessionRunner struct {
	session *oauth.ClientSession
	err     error
	owner   syntax.DID
	id      string
}

func (r *sessionRunner) WithActiveSession(ctx context.Context, owner syntax.DID, id string, operation auth.OAuthSessionOperation) error {
	r.owner, r.id = owner, id
	if r.err != nil {
		return r.err
	}
	return operation(ctx, r.session)
}

func TestIssuerBindsOwnerDestinationMethodAndFiveMinuteExpiry(t *testing.T) {
	now := time.Date(2026, 10, 13, 12, 0, 0, 123, time.UTC)
	runner := &sessionRunner{session: &oauth.ClientSession{Data: &oauth.ClientSessionData{AccountDID: "did:plc:alice", HostURL: "https://pds.example"}}}
	issuer := &Issuer{sessions: runner, now: func() time.Time { return now }, request: func(_ context.Context, session *oauth.ClientSession, audience string, expiry int64, method string) (string, error) {
		if session != runner.session || audience != "did:web:pds.example" || expiry != now.Add(5*time.Minute).Unix() || method != "com.atproto.repo.uploadBlob" {
			t.Fatal("incorrect fenced authorization")
		}
		return "service-secret", nil
	}}
	result, err := issuer.IssueUpload(context.Background(), "did:plc:alice", "session-alice")
	if err != nil {
		t.Fatal(err)
	}
	if result.Token != "service-secret" || result.UploadEndpoint != "https://pds.example/xrpc/com.atproto.repo.uploadBlob" || result.ExpiresAt.Unix() != now.Add(Lifetime).Unix() {
		t.Fatal("incorrect authorization result")
	}
	if runner.owner != "did:plc:alice" || runner.id != "session-alice" {
		t.Fatal("wrong session identity")
	}
	for _, format := range []string{"%v", "%+v", "%#v"} {
		if fmt.Sprintf(format, result) != "ImageUploadAuthorization(<private>)" {
			t.Fatal("debug output exposes capability")
		}
	}
}

func TestIssuerRejectsInvalidSessionsAndDestinations(t *testing.T) {
	for _, host := range []string{"http://pds.example", "https://user:secret@pds.example", "https://pds.example/path", "https://pds.example?capability=secret", "https://pds.example#fragment", "https://"} {
		t.Run(host, func(t *testing.T) {
			runner := &sessionRunner{session: &oauth.ClientSession{Data: &oauth.ClientSessionData{AccountDID: "did:plc:alice", HostURL: host}}}
			issuer := &Issuer{sessions: runner, now: time.Now, request: func(context.Context, *oauth.ClientSession, string, int64, string) (string, error) {
				t.Fatal("unsafe request issued")
				return "", nil
			}}
			if _, err := issuer.IssueUpload(context.Background(), "did:plc:alice", "session"); !errors.Is(err, ErrUnavailable) {
				t.Fatalf("error = %v", err)
			}
		})
	}
	for _, owner := range []syntax.DID{"", "did:plc:bob"} {
		runner := &sessionRunner{session: &oauth.ClientSession{Data: &oauth.ClientSessionData{AccountDID: "did:plc:alice", HostURL: "https://pds.example"}}}
		issuer := &Issuer{sessions: runner, now: time.Now, request: func(context.Context, *oauth.ClientSession, string, int64, string) (string, error) {
			t.Fatal("wrong owner authorized")
			return "", nil
		}}
		if _, err := issuer.IssueUpload(context.Background(), owner, "session"); !errors.Is(err, ErrUnavailable) {
			t.Fatal(err)
		}
	}
}

func TestIssuerPreservesCancellationAndOriginalCauses(t *testing.T) {
	for _, cause := range []error{context.Canceled, auth.ErrPDSSessionExpired} {
		runner := &sessionRunner{err: cause}
		issuer := &Issuer{sessions: runner, now: time.Now, request: func(context.Context, *oauth.ClientSession, string, int64, string) (string, error) {
			t.Fatal("request issued after invalidation")
			return "", nil
		}}
		if _, err := issuer.IssueUpload(context.Background(), "did:plc:alice", "session"); !errors.Is(err, cause) {
			t.Fatalf("lost cause: %v", err)
		}
	}
}
