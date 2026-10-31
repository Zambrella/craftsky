package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
)

var ErrUnsupportedMethod = errors.New("unsupported messaging operation")
var ErrProviderResponse = errors.New("messaging provider response unavailable")

type Transport struct{ client *atclient.APIClient }

func NewTransport(client *atclient.APIClient) *Transport {
	if client == nil {
		return &Transport{}
	}
	copy := *client
	copy.Headers = client.Headers.Clone()
	if copy.Headers == nil {
		copy.Headers = make(http.Header)
	}
	return &Transport{client: &copy}
}

func (transport *Transport) Query(ctx context.Context, method syntax.NSID, params map[string]any, out any) error {
	return transport.call(ctx, http.MethodGet, method, params, nil, out)
}
func (transport *Transport) Procedure(ctx context.Context, method syntax.NSID, body any, out any) error {
	return transport.call(ctx, http.MethodPost, method, nil, body, out)
}

func methodAudience(verb string, method syntax.NSID) (string, bool) {
	if verb == http.MethodGet {
		switch method {
		case "chat.bsky.actor.getStatus", "chat.bsky.convo.listConvos", "chat.bsky.convo.listConvoRequests",
			"chat.bsky.convo.getConvo", "chat.bsky.convo.getMessages", "chat.bsky.convo.getConvoAvailability",
			"chat.bsky.convo.getConvoForMembers", "chat.bsky.convo.getUnreadCounts", "chat.bsky.convo.getLog":
			return auth.ChatServiceAudience, true
		}
	} else if verb == http.MethodPost {
		switch method {
		case "chat.bsky.convo.sendMessage", "chat.bsky.convo.updateRead", "chat.bsky.convo.acceptConvo",
			"chat.bsky.convo.muteConvo", "chat.bsky.convo.unmuteConvo", "chat.bsky.convo.leaveConvo", "chat.bsky.convo.deleteMessageForSelf":
			return auth.ChatServiceAudience, true
		case "com.atproto.moderation.createReport":
			return auth.ChatReportAudience, true
		}
	}
	return "", false
}

// This boundary deliberately has no general retry policy. In particular send
// and report ambiguity must reach their durable operation receipt unchanged.
func (transport *Transport) call(ctx context.Context, verb string, method syntax.NSID, params map[string]any, body, out any) error {
	audience, ok := methodAudience(verb, method)
	if !ok {
		return ErrUnsupportedMethod
	}
	return transport.callFixed(ctx, verb, method, audience, params, body, out)
}

func (transport *Transport) callFixed(ctx context.Context, verb string, method syntax.NSID, audience string, params map[string]any, body, out any) error {
	if transport == nil || transport.client == nil {
		return ErrProviderResponse
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var encoded []byte
	var err error
	if body != nil {
		encoded, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	request := atclient.NewAPIRequest(verb, method, bytes.NewReader(encoded))
	request.Headers.Set("Accept", "application/json")
	if body != nil {
		request.Headers.Set("Content-Type", "application/json")
	}
	if params != nil {
		request.QueryParams, err = atclient.ParseParams(params)
		if err != nil {
			return err
		}
	}
	client := transport.client.WithService(audience)
	if audience == "" {
		client.Headers.Del("Atproto-Proxy")
	}
	response, err := client.Do(ctx, request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	const maxResponse = 2 << 20
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxResponse+1))
	if err != nil {
		return err
	}
	if len(raw) > maxResponse || !ValidJSONUnicode(raw) {
		return ErrProviderResponse
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var envelope atclient.ErrorBody
		_ = json.Unmarshal(raw, &envelope)
		cause := envelope.APIError(response.StatusCode)
		if response.StatusCode == http.StatusTooManyRequests {
			return &auth.RateLimitError{RetryAfter: auth.RetryAfter(response.Header.Get("Retry-After"), time.Now()), Cause: cause}
		}
		return cause
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return errors.Join(ErrProviderResponse, err)
	}
	return nil
}
