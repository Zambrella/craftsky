package instagrammeta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPClientLookupUsernameUsesVersionedInstagramGraphRequest(t *testing.T) {
	t.Parallel()

	const (
		accessToken = "synthetic-private-access-token"
		senderIGSID = "synthetic-sender-id"
		username    = "synthetic_user"
	)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v99.0/"+senderIGSID {
			t.Errorf("path = %q, want versioned profile path", r.URL.Path)
		}
		if got := r.URL.Query().Get("fields"); got != "username" {
			t.Errorf("fields = %q, want username", got)
		}
		if got := r.URL.Query().Get("access_token"); got != "" {
			t.Errorf("access token appeared in URL query: %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+accessToken {
			t.Errorf("Authorization = %q, want bearer token", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":                  senderIGSID,
			"username":            username,
			"name":                "synthetic unrelated profile name",
			"profile_picture_url": "https://example.invalid/private-picture",
		})
	}))
	t.Cleanup(server.Close)

	client, err := NewHTTPClient(HTTPClientConfig{
		HTTPClient:        server.Client(),
		BaseURL:           server.URL,
		APIVersion:        "v99.0",
		AccessToken:       accessToken,
		OfficialAccountID: "synthetic-official-id",
	})
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	var _ Client = client

	got, err := client.LookupUsername(context.Background(), senderIGSID)
	if err != nil {
		t.Fatalf("LookupUsername: %v", err)
	}
	if got != username {
		t.Fatalf("LookupUsername = %q, want %q", got, username)
	}
}

func TestHTTPClientSendReplyUsesOfficialMessagesEndpoint(t *testing.T) {
	t.Parallel()

	const (
		accessToken = "synthetic-private-access-token"
		officialID  = "synthetic-official-id"
		senderIGSID = "synthetic-sender-id"
		replyText   = "Synthetic verification response."
	)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v99.0/"+officialID+"/messages" {
			t.Errorf("path = %q, want official messages path", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+accessToken {
			t.Errorf("Authorization = %q, want bearer token", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		var payload struct {
			Recipient struct {
				ID string `json:"id"`
			} `json:"recipient"`
			Message struct {
				Text string `json:"text"`
			} `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if payload.Recipient.ID != senderIGSID || payload.Message.Text != replyText {
			t.Errorf("reply payload = %+v", payload)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"recipient_id": senderIGSID,
			"message_id":   "synthetic-reply-message-id",
		})
	}))
	t.Cleanup(server.Close)

	client := newTestHTTPClient(t, server, HTTPClientConfig{
		APIVersion:        "v99.0",
		AccessToken:       accessToken,
		OfficialAccountID: officialID,
	})
	if err := client.SendReply(context.Background(), senderIGSID, replyText); err != nil {
		t.Fatalf("SendReply: %v", err)
	}
}

func TestHTTPClientClassifiesRateLimitWithBoundedRedactedError(t *testing.T) {
	t.Parallel()

	const (
		tokenCanary    = "synthetic-private-token"
		identityCanary = "synthetic-private-igsid"
		bodyCanary     = "synthetic-private-upstream-body"
	)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "86400")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"error":{"message":"`+bodyCanary+`"}}`)
	}))
	t.Cleanup(server.Close)
	client := newTestHTTPClient(t, server, HTTPClientConfig{
		APIVersion:        "v99.0",
		AccessToken:       tokenCanary,
		OfficialAccountID: "official",
	})

	_, err := client.LookupUsername(context.Background(), identityCanary)
	if err == nil {
		t.Fatal("LookupUsername unexpectedly succeeded")
	}
	kind, retryAfter, ok := ProviderErrorDetails(err)
	if !ok || kind != ProviderErrorRateLimited {
		t.Fatalf("ProviderErrorDetails = (%q, %s, %t), want rate limited", kind, retryAfter, ok)
	}
	if retryAfter != maxProviderRetryAfter {
		t.Fatalf("retry after = %s, want capped %s", retryAfter, maxProviderRetryAfter)
	}
	if !IsRetryableProviderError(err) {
		t.Fatal("rate-limited error was not retryable")
	}
	diagnostic := fmt.Sprintf("error=%v detailed=%+v go=%#v", err, err, err)
	for _, private := range []string{tokenCanary, identityCanary, bodyCanary} {
		if strings.Contains(diagnostic, private) {
			t.Fatalf("provider error leaked %q: %s", private, diagnostic)
		}
	}
}

func TestHTTPClientCancellationIsNotProviderFailure(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(started)
		<-time.After(time.Second)
	}))
	t.Cleanup(server.Close)
	client := newTestHTTPClient(t, server, HTTPClientConfig{
		APIVersion:        "v99.0",
		AccessToken:       "synthetic-token",
		OfficialAccountID: "official",
	})
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := client.LookupUsername(ctx, "sender")
		errCh <- err
	}()
	<-started
	cancel()
	if err := <-errCh; !errors.Is(err, context.Canceled) {
		t.Fatalf("LookupUsername cancellation = %v, want context.Canceled", err)
	}
}

func TestHTTPClientDoesNotFollowProviderRedirects(t *testing.T) {
	t.Parallel()

	targetCalled := make(chan struct{}, 1)
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		targetCalled <- struct{}{}
		_ = json.NewEncoder(w).Encode(map[string]string{"username": "should_not_be_reached"})
	}))
	t.Cleanup(target.Close)
	redirector := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/capture", http.StatusTemporaryRedirect)
	}))
	t.Cleanup(redirector.Close)

	client := newTestHTTPClient(t, redirector, HTTPClientConfig{
		APIVersion:        "v99.0",
		AccessToken:       "synthetic-private-token",
		OfficialAccountID: "official",
	})
	_, err := client.LookupUsername(context.Background(), "synthetic-private-igsid")
	if err == nil {
		t.Fatal("LookupUsername followed a redirect and unexpectedly succeeded")
	}
	select {
	case <-targetCalled:
		t.Fatal("provider redirect reached another origin")
	default:
	}
}

func TestHTTPClientValidatesAndNormalizesOnlyTheUsernameField(t *testing.T) {
	t.Parallel()

	responses := map[string]string{
		"uppercase": `{"username":"Synthetic.User_9","name":"ignored"}`,
		"missing":   `{"id":"sender"}`,
		"space":     `{"username":"synthetic user"}`,
		"unicode":   `{"username":"synthetïc"}`,
		"too-long":  `{"username":"` + strings.Repeat("a", 31) + `"}`,
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, responses[strings.TrimPrefix(r.URL.Path, "/v99.0/")])
	}))
	t.Cleanup(server.Close)
	client := newTestHTTPClient(t, server, HTTPClientConfig{
		APIVersion:        "v99.0",
		AccessToken:       "synthetic-token",
		OfficialAccountID: "official",
	})

	username, err := client.LookupUsername(context.Background(), "uppercase")
	if err != nil || username != "synthetic.user_9" {
		t.Fatalf("LookupUsername(uppercase) = (%q, %v)", username, err)
	}
	for _, sender := range []string{"missing", "space", "unicode", "too-long"} {
		if _, err := client.LookupUsername(context.Background(), sender); err == nil {
			t.Errorf("LookupUsername(%s) unexpectedly succeeded", sender)
		} else if kind, _, ok := ProviderErrorDetails(err); !ok || kind != ProviderErrorInvalidResponse {
			t.Errorf("LookupUsername(%s) error = %v, want invalid response", sender, err)
		}
	}
}

func TestHTTPClientSuppressesSensitiveTransportErrorsAndSetsFiveSecondDeadline(t *testing.T) {
	t.Parallel()

	const (
		tokenCanary     = "synthetic-private-token"
		identityCanary  = "synthetic-private-igsid"
		transportCanary = "synthetic-private-transport-error"
	)
	client, err := NewHTTPClient(HTTPClientConfig{
		HTTPClient: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			deadline, ok := request.Context().Deadline()
			if !ok {
				t.Error("provider request has no deadline")
			} else if remaining := time.Until(deadline); remaining <= 4*time.Second || remaining > MaxProviderTimeout {
				t.Errorf("provider request deadline remaining = %s, want at most five seconds", remaining)
			}
			if strings.Contains(request.URL.String(), tokenCanary) {
				t.Error("access token appeared in provider URL")
			}
			return nil, errors.New(transportCanary + "/" + identityCanary + "/" + tokenCanary)
		})},
		APIVersion:        "v99.0",
		AccessToken:       tokenCanary,
		OfficialAccountID: "official",
	})
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	_, err = client.LookupUsername(context.Background(), identityCanary)
	if err == nil {
		t.Fatal("LookupUsername unexpectedly succeeded")
	}
	diagnostic := fmt.Sprintf("%v/%+v/%#v", err, err, err)
	for _, private := range []string{tokenCanary, identityCanary, transportCanary} {
		if strings.Contains(diagnostic, private) {
			t.Fatalf("provider error leaked %q: %s", private, diagnostic)
		}
	}
}

func newTestHTTPClient(t *testing.T, server *httptest.Server, config HTTPClientConfig) *HTTPClient {
	t.Helper()
	config.HTTPClient = server.Client()
	config.BaseURL = server.URL
	client, err := NewHTTPClient(config)
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	return client
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
