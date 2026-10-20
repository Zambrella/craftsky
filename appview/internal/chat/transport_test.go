package chat

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"social.craftsky/appview/internal/auth"
)

// IT-019 / UT-002: no replacement-character normalization at JSON boundaries.
func TestTransportRejectsUnpairedUnicodeWithoutChangingValidText(t *testing.T) {
	for index, test := range []struct {
		raw, text string
		valid     bool
	}{
		{`{"text":"\ud800"}`, "", false},
		{`{"text":"\udc00"}`, "", false},
		{`{"text":"\ud800\u0041"}`, "", false},
		{`{"text":"\ud83e\uddf6"}`, "🧶", true},
		{`{"text":"\\ud800"}`, `\ud800`, true},
		{`{"text":"�"}`, "�", true},
	} {
		client := &atclient.APIClient{Host: "https://pds.example", Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(test.raw)), Request: r}, nil
		})}}
		var result struct {
			Text string `json:"text"`
		}
		err := NewTransport(client).Query(t.Context(), "chat.bsky.actor.getStatus", nil, &result)
		if test.valid && (err != nil || result.Text != test.text) {
			t.Fatalf("case %d valid text changed or rejected", index)
		}
		if !test.valid && !errors.Is(err, ErrProviderResponse) {
			t.Fatalf("case %d silently repaired malformed Unicode", index)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// IT-015 / RULE-004: destinations come from the fixed method profile, never
// from a caller-provided service URL. Nonces/tokens remain the SDK's concern.
func TestTransportPinsChatAndReportProxy(t *testing.T) {
	var routes []string
	client := &atclient.APIClient{Host: "https://pds.example", Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "pds.example" {
			t.Fatal("request bypassed the PDS")
		}
		routes = append(routes, r.Header.Get("Atproto-Proxy"))
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`)), Request: r}, nil
	})}}
	transport := NewTransport(client)
	if err := transport.Query(context.Background(), "chat.bsky.actor.getStatus", nil, &struct{}{}); err != nil {
		t.Fatal(err)
	}
	if err := transport.Procedure(context.Background(), "com.atproto.moderation.createReport", map[string]any{"reasonType": "com.atproto.moderation.defs#reasonOther"}, &struct{}{}); err != nil {
		t.Fatal(err)
	}
	if len(routes) != 2 || routes[0] != auth.ChatServiceAudience || routes[1] != auth.ChatReportAudience {
		t.Fatalf("wrong fixed audiences: %v", routes)
	}
	if err := transport.Procedure(context.Background(), "chat.bsky.convo.deleteMessageForSelf", map[string]any{}, nil); err == nil || len(routes) != 2 {
		t.Fatal("unapproved method dispatched")
	}
}
