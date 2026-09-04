package video

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestUploadLimitsService_UsesInternalPurposeBoundAuthorization(t *testing.T) {
	t.Parallel()
	fixedNow := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	runner := &fakeSessionRunner{session: &oauth.ClientSession{Data: &oauth.ClientSessionData{
		HostURL: "https://pds.example",
	}}}
	var gotAudience, gotMethod, gotEndpoint, gotToken string
	var gotExpiry int64
	service, err := NewUploadLimitsService(UploadLimitsServiceOptions{
		Sessions:    runner,
		PDSClient:   &http.Client{},
		VideoClient: &http.Client{},
		ServiceDID:  syntax.DID("did:web:video.bsky.app"),
		ServiceURL:  "https://video.bsky.app",
		Now:         func() time.Time { return fixedNow },
		authorize: func(_ context.Context, session *oauth.ClientSession, audience string, expiry int64, method string) (string, error) {
			if session != runner.session {
				t.Fatal("authorization escaped the fenced session")
			}
			gotAudience, gotExpiry, gotMethod = audience, expiry, method
			return "limits-service-jwt", nil
		},
		requestLimits: func(_ context.Context, _ *http.Client, endpoint, token string) (UploadLimitsResponse, error) {
			gotEndpoint, gotToken = endpoint, token
			videos, bytes := int64(4), int64(900_000_000)
			return UploadLimitsResponse{
				CanUpload: true, RemainingDailyVideos: &videos, RemainingDailyBytes: &bytes,
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewUploadLimitsService: %v", err)
	}

	limits, err := service.Get(context.Background(), syntax.DID("did:plc:alice"), "session-alice")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !limits.CanUpload || limits.RemainingDailyVideos == nil || *limits.RemainingDailyVideos != 4 ||
		limits.RemainingDailyBytes == nil || *limits.RemainingDailyBytes != 900_000_000 || limits.Reason != "" {
		t.Fatalf("limits = %+v", limits)
	}
	if runner.calls != 1 || runner.owner != syntax.DID("did:plc:alice") || runner.sessionID != "session-alice" {
		t.Fatalf("runner calls=%d owner=%q session=%q", runner.calls, runner.owner, runner.sessionID)
	}
	if gotAudience != "did:web:video.bsky.app" || gotMethod != "app.bsky.video.getUploadLimits" ||
		gotExpiry != fixedNow.Add(time.Minute).Unix() {
		t.Fatalf("authorization audience=%q method=%q expiry=%d", gotAudience, gotMethod, gotExpiry)
	}
	if gotEndpoint != "https://video.bsky.app" || gotToken != "limits-service-jwt" {
		t.Fatalf("request endpoint=%q token=%q", gotEndpoint, gotToken)
	}
}

func TestUploadLimitsService_NormalizesEligibilityAndFailures(t *testing.T) {
	t.Parallel()
	one, two := int64(1), int64(2)
	lowBytes, enoughBytes := int64(299_999_999), int64(300_000_000)
	emailError := "EmailNotConfirmed"
	providerError := "UnsupportedPds"
	unknownError := "FutureProviderState"
	tests := []struct {
		name       string
		response   UploadLimitsResponse
		requestErr error
		want       UploadLimits
		wantErr    error
	}{
		{name: "eligible", response: UploadLimitsResponse{CanUpload: true, RemainingDailyVideos: &two, RemainingDailyBytes: &enoughBytes}, want: UploadLimits{CanUpload: true, RemainingDailyVideos: &two, RemainingDailyBytes: &enoughBytes}},
		{name: "email", response: UploadLimitsResponse{Error: &emailError}, want: UploadLimits{Reason: "email_unverified"}},
		{name: "provider", response: UploadLimitsResponse{Error: &providerError}, want: UploadLimits{Reason: "provider_unsupported"}},
		{name: "video quota", response: UploadLimitsResponse{CanUpload: true, RemainingDailyVideos: &one, RemainingDailyBytes: &enoughBytes}, want: UploadLimits{RemainingDailyVideos: &one, RemainingDailyBytes: &enoughBytes, Reason: "quota_exhausted"}},
		{name: "byte quota", response: UploadLimitsResponse{CanUpload: true, RemainingDailyVideos: &two, RemainingDailyBytes: &lowBytes}, want: UploadLimits{RemainingDailyVideos: &two, RemainingDailyBytes: &lowBytes, Reason: "quota_exhausted"}},
		{name: "unknown", response: UploadLimitsResponse{Error: &unknownError}, want: UploadLimits{Reason: "unknown"}},
		{name: "upstream unavailable", requestErr: errors.New("provider secret detail"), wantErr: ErrUploadLimitsUnavailable},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			runner := &fakeSessionRunner{session: &oauth.ClientSession{Data: &oauth.ClientSessionData{HostURL: "https://pds.example"}}}
			service, err := NewUploadLimitsService(UploadLimitsServiceOptions{
				Sessions: runner, PDSClient: &http.Client{}, VideoClient: &http.Client{},
				ServiceDID: syntax.DID("did:web:video.bsky.app"), ServiceURL: "https://video.bsky.app",
				authorize: func(context.Context, *oauth.ClientSession, string, int64, string) (string, error) {
					return "limits-token", nil
				},
				requestLimits: func(context.Context, *http.Client, string, string) (UploadLimitsResponse, error) {
					return test.response, test.requestErr
				},
			})
			if err != nil {
				t.Fatalf("NewUploadLimitsService: %v", err)
			}
			got, err := service.Get(context.Background(), syntax.DID("did:plc:alice"), "session-alice")
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}
			if got.CanUpload != test.want.CanUpload || got.Reason != test.want.Reason ||
				!equalInt64Pointers(got.RemainingDailyVideos, test.want.RemainingDailyVideos) ||
				!equalInt64Pointers(got.RemainingDailyBytes, test.want.RemainingDailyBytes) {
				t.Fatalf("limits = %+v, want %+v", got, test.want)
			}
			if err != nil && strings.Contains(err.Error(), "provider secret detail") {
				t.Fatalf("error leaked provider detail: %v", err)
			}
		})
	}
}

func equalInt64Pointers(left, right *int64) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

func TestRequestUploadLimits_UsesExactAuthenticatedEndpoint(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/xrpc/app.bsky.video.getUploadLimits" || r.URL.RawQuery != "" {
			t.Errorf("request = %s %s", r.Method, r.URL.String())
		}
		if authorization := r.Header.Get("Authorization"); authorization != "Bearer limits-service-jwt" {
			t.Errorf("Authorization = %q", authorization)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"canUpload":true,"remainingDailyVideos":3,"remainingDailyBytes":600000000}`))
	}))
	t.Cleanup(server.Close)

	response, err := requestUploadLimits(context.Background(), server.Client(), server.URL, "limits-service-jwt")
	if err != nil {
		t.Fatalf("requestUploadLimits: %v", err)
	}
	if !response.CanUpload || response.RemainingDailyVideos == nil || *response.RemainingDailyVideos != 3 ||
		response.RemainingDailyBytes == nil || *response.RemainingDailyBytes != 600_000_000 {
		t.Fatalf("response = %+v", response)
	}
}
