package api

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/scheduledposts"
	"social.craftsky/appview/internal/subscriptions"
)

type subscriptionBlockedScheduledList struct{ items []scheduledposts.Resource }

type subscriptionBlockedScheduledDetail struct{ item scheduledposts.Resource }

func (s subscriptionBlockedScheduledDetail) Get(context.Context, syntax.DID, uuid.UUID) (scheduledposts.Resource, error) {
	return s.item, nil
}

type scheduledAccessStub struct{ state subscriptions.SelfAccess }

func (s scheduledAccessStub) SelfAccess(context.Context, syntax.DID, time.Time) (subscriptions.SelfAccess, error) {
	return s.state, nil
}

func (f subscriptionBlockedScheduledList) List(context.Context, syntax.DID) ([]scheduledposts.Resource, error) {
	return f.items, nil
}

func TestScheduledOwnerListExposesSubscriptionErrorForMissedWork(t *testing.T) {
	payload, err := scheduledposts.EncodePayload(scheduledposts.Payload{Kind: scheduledposts.PostKindStandard, Text: "missed"})
	if err != nil {
		t.Fatal(err)
	}
	list := subscriptionBlockedScheduledList{items: []scheduledposts.Resource{{ScheduledPost: scheduledposts.ScheduledPost{ID: uuid.New(), Status: scheduledposts.StatusNeedsAttention, ScheduledAt: time.Now()}, PayloadBytes: payload, LastErrorCode: "subscription_required"}}}
	response := serveScheduledPostRequest(t, ListScheduledPostsHandler(list, nil), http.MethodGet, "/v1/scheduled-posts", "", "did:plc:owner")
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Items []struct {
			LastErrorCode string `json:"lastErrorCode"`
		} `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || len(body.Items) != 1 || body.Items[0].LastErrorCode != "subscription_required" {
		t.Fatalf("response=%s err=%v", response.Body.String(), err)
	}
}

func TestScheduledOwnerListSignalsFuturePendingWorkDuringLapse(t *testing.T) {
	payload, err := scheduledposts.EncodePayload(scheduledposts.Payload{Kind: scheduledposts.PostKindStandard, Text: "future"})
	if err != nil {
		t.Fatal(err)
	}
	list := subscriptionBlockedScheduledList{items: []scheduledposts.Resource{{ScheduledPost: scheduledposts.ScheduledPost{ID: uuid.New(), Status: scheduledposts.StatusScheduled, ScheduledAt: time.Now().Add(time.Hour)}, PayloadBytes: payload}}}
	response := serveScheduledPostRequest(t, ListScheduledPostsHandler(list, nil, scheduledAccessStub{state: subscriptions.SelfAccess{EffectiveTier: subscriptions.TierFree}}), http.MethodGet, "/v1/scheduled-posts", "", "did:plc:owner")
	var body struct {
		Items []struct {
			SubscriptionRequired bool `json:"subscriptionRequired"`
		} `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || len(body.Items) != 1 || !body.Items[0].SubscriptionRequired {
		t.Fatalf("response=%s err=%v", response.Body.String(), err)
	}
}

func TestScheduledOwnerDetailSignalsPendingSubscriptionDuringLapse(t *testing.T) {
	payload, err := scheduledposts.EncodePayload(scheduledposts.Payload{Kind: scheduledposts.PostKindStandard, Text: "future"})
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	reader := subscriptionBlockedScheduledDetail{item: scheduledposts.Resource{ScheduledPost: scheduledposts.ScheduledPost{ID: id, OperationID: uuid.New(), Status: scheduledposts.StatusScheduled, ScheduledAt: time.Now().Add(time.Hour)}, PayloadBytes: payload}}
	r := serveScheduledPostPathRequest(t, GetScheduledPostHandler(reader, nil, scheduledAccessStub{state: subscriptions.SelfAccess{EffectiveTier: subscriptions.TierFree}}), http.MethodGet, id.String(), "", "did:plc:owner")
	var body struct {
		SubscriptionRequired bool `json:"subscriptionRequired"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil || !body.SubscriptionRequired {
		t.Fatalf("detail=%s err=%v", r.Body.String(), err)
	}
}

func serveScheduledPostRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	body string,
	owner syntax.DID,
) *httptest.ResponseRecorder {
	t.Helper()
	request := scheduledPostRequest(method, path, body, owner)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func serveScheduledPostPathRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	id string,
	body string,
	owner syntax.DID,
) *httptest.ResponseRecorder {
	t.Helper()
	request := scheduledPostRequest(method, "/v1/scheduled-posts/"+id, body, owner)
	request.SetPathValue("id", id)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func scheduledPostRequest(method, path, body string, owner syntax.DID) *http.Request {
	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}
	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Content-Type", "application/json")
	return request.WithContext(middleware.WithDID(request.Context(), owner))
}

func decodeResponseJSON[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(recorder.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode response JSON: %v; body=%q", err, recorder.Body.String())
	}
	return value
}

func assertScheduledPostError(t *testing.T, recorder *httptest.ResponseRecorder, code string) {
	t.Helper()
	body := decodeResponseJSON[envelope.Error](t, recorder)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &fields); err != nil {
		t.Fatalf("decode error envelope fields: %v", err)
	}
	for _, key := range []string{"error", "message", "requestId"} {
		if _, ok := fields[key]; !ok {
			t.Fatalf("error envelope missing %q: %s", key, recorder.Body.String())
		}
	}
	if body.Error != code || body.Message == "" {
		t.Fatalf("error response=%+v, want code %q and standard envelope", body, code)
	}
}

func TestScheduledPostRequestPreservesBody(t *testing.T) {
	t.Parallel()
	const body = `{"payload":{"text":"missing sponsored"}}`
	request := scheduledPostRequest(http.MethodPost, "/v1/scheduled-posts", body, "did:plc:alice")
	got, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	if string(got) != body {
		t.Fatalf("request body = %q, want exact fixture %q", got, body)
	}
}
