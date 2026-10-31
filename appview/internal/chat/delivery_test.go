package chat

import (
	"context"
	"errors"
	"fmt"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"io"
	"net/http"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/push"
	"strings"
	"testing"
	"time"
)

type chatPushFunc func(context.Context, push.SendRequest) (push.ProviderResult, error)

func (f chatPushFunc) Send(ctx context.Context, request push.SendRequest) (push.ProviderResult, error) {
	return f(ctx, request)
}

// IT-011: the application worker drains durable private jobs through the
// shared sender and joins finalization during graceful shutdown.
func TestDeliveryWorkerSendsQueuedJobAndStops(t *testing.T) {
	service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"chatDisabled":false}`
		if strings.HasSuffix(r.URL.Path, "getConvo") {
			body = `{"convo":{"id":"direct-1","status":"accepted","unreadCount":1,"members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:peer"}],"lastMessage":{"$type":"chat.bsky.convo.defs#messageView","id":"incoming-1","rev":"0002","sender":{"did":"did:plc:peer"},"text":"private text","sentAt":"2026-10-19T12:00:00Z"}}}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	seedDeliveryFixture(t, store, info)
	sent := make(chan struct{}, 1)
	dispatcher, err := NewDeliveryDispatcher(service, backgroundSessionFunc(func(context.Context, syntax.DID, []string) (string, error) { return info.SessionID, nil }), chatPushFunc(func(context.Context, push.SendRequest) (push.ProviderResult, error) {
		sent <- struct{}{}
		return push.ProviderResult{Class: push.ResultSuccess}, nil
	}), DeliveryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- dispatcher.Run(ctx) }()
	select {
	case <-sent:
	case <-time.After(3 * time.Second):
		t.Fatal("durable delivery worker did not send")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("shutdown=%v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("delivery worker did not join finalization")
	}
	var state string
	if err := store.pool.QueryRow(t.Context(), `SELECT status FROM chat_push_deliveries`).Scan(&state); err != nil || state != "succeeded" {
		t.Fatalf("final status=%s err=%v", state, err)
	}
}

// IT-011 / FR-014: an invalid-token result deactivates the attempted token;
// a token rotated during the provider request must remain active.
func TestDeliveryInvalidTokenCannotDeactivateRotatedToken(t *testing.T) {
	for _, rotated := range []bool{false, true} {
		t.Run(fmt.Sprintf("rotated=%v", rotated), func(t *testing.T) {
			service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
				body := `{"chatDisabled":false}`
				if strings.HasSuffix(r.URL.Path, "getConvo") {
					body = `{"convo":{"id":"direct-1","status":"accepted","unreadCount":1,"members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:peer"}],"lastMessage":{"$type":"chat.bsky.convo.defs#messageView","id":"incoming-1","rev":"0002","sender":{"did":"did:plc:peer"},"text":"private text","sentAt":"2026-10-19T12:00:00Z"}}}`
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			}))
			seedDeliveryFixture(t, store, info)
			dispatcher, err := NewDeliveryDispatcher(service, backgroundSessionFunc(func(context.Context, syntax.DID, []string) (string, error) { return info.SessionID, nil }), chatPushFunc(func(context.Context, push.SendRequest) (push.ProviderResult, error) {
				if rotated {
					if _, err := store.pool.Exec(t.Context(), `UPDATE push_installations SET fcm_token='new-rotated-token'`); err != nil {
						t.Fatal(err)
					}
				}
				return push.ProviderResult{Class: push.ResultInvalidToken}, errors.New("synthetic invalid token")
			}), DeliveryOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if processed, _ := dispatcher.ProcessOne(t.Context()); !processed {
				t.Fatal("delivery not claimed")
			}
			var active, subscribed bool
			var token string
			if err := store.pool.QueryRow(t.Context(), `SELECT installation.active,subscription.active,installation.fcm_token FROM push_installations installation JOIN push_account_subscriptions subscription ON subscription.installation_id=installation.id`).Scan(&active, &subscribed, &token); err != nil {
				t.Fatal(err)
			}
			if active != rotated || subscribed != rotated {
				t.Fatalf("active=%v subscribed=%v rotated=%v", active, subscribed, rotated)
			}
			if rotated && token != "new-rotated-token" {
				t.Fatal("rotated token overwritten")
			}
		})
	}
}

func seedDeliveryFixture(t *testing.T, store *Store, info auth.AuthInfo) {
	t.Helper()
	if _, err := store.pool.Exec(t.Context(), `WITH installation AS (INSERT INTO push_installations(id,device_id,platform,fcm_token) VALUES(gen_random_uuid(),'synthetic-delivery-device','ios','synthetic-delivery-token') RETURNING id) INSERT INTO push_account_subscriptions(id,installation_id,account_did,routing_id) SELECT gen_random_uuid(),id,$1,gen_random_uuid() FROM installation`, info.DID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(t.Context(), `WITH notification AS(INSERT INTO chat_notifications(owner_did,id,logical_key,kind,conversation_id,message_id,messaging_generation,open_reference) VALUES($1,gen_random_uuid(),'incoming-1','normal','direct-1','incoming-1',2,gen_random_uuid()) RETURNING owner_did,id) INSERT INTO chat_push_deliveries(owner_did,id,notification_id,account_subscription_id,deadline_at) SELECT notification.owner_did,gen_random_uuid(),notification.id,subscription.id,now()+interval '24 hours' FROM notification JOIN push_account_subscriptions subscription ON subscription.account_did=notification.owner_did`, info.DID); err != nil {
		t.Fatal(err)
	}
}

// IT-011 / RULE-005, FR-013: a profile mute cannot suppress a DM;
// authoritative shared conversation mute suppresses it before provider send.
func TestDeliveryIgnoresProfileMuteAndChecksCurrentConversationMute(t *testing.T) {
	for _, muted := range []bool{false, true} {
		t.Run(fmt.Sprintf("conversation muted=%v", muted), func(t *testing.T) {
			service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
				body := `{"chatDisabled":false}`
				switch r.URL.Path {
				case "/xrpc/chat.bsky.actor.getStatus":
				case "/xrpc/chat.bsky.convo.getConvo":
					body = fmt.Sprintf(`{"convo":{"id":"direct-1","status":"accepted","muted":%v,"unreadCount":1,"members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:peer","displayName":"Alex","handle":"peer.example"}],"lastMessage":{"$type":"chat.bsky.convo.defs#messageView","id":"incoming-1","rev":"0002","sender":{"did":"did:plc:peer"},"text":"private-dispatch-canary","sentAt":"2026-10-19T12:00:00Z"}}}`, muted)
				default:
					t.Fatalf("unexpected dispatch operation %s", r.URL.Path)
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			}))
			if _, err := store.pool.Exec(t.Context(), `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'synthetic-cid') ON CONFLICT DO NOTHING`, info.DID); err != nil {
				t.Fatal(err)
			}
			if _, err := store.pool.Exec(t.Context(), `INSERT INTO actor_mutes(owner_did,subject_did) VALUES($1,'did:plc:peer')`, info.DID); err != nil {
				t.Fatal(err)
			}
			if _, err := store.pool.Exec(t.Context(), `WITH installation AS (INSERT INTO push_installations(id,device_id,platform,fcm_token) VALUES(gen_random_uuid(),'synthetic-delivery-device','ios','synthetic-delivery-token') RETURNING id) INSERT INTO push_account_subscriptions(id,installation_id,account_did,routing_id) SELECT gen_random_uuid(),id,$1,gen_random_uuid() FROM installation`, info.DID); err != nil {
				t.Fatal(err)
			}
			if _, err := store.pool.Exec(t.Context(), `WITH notification AS(INSERT INTO chat_notifications(owner_did,id,logical_key,kind,conversation_id,message_id,messaging_generation,open_reference) VALUES($1,gen_random_uuid(),'incoming-1','normal','direct-1','incoming-1',2,gen_random_uuid()) RETURNING owner_did,id) INSERT INTO chat_push_deliveries(owner_did,id,notification_id,account_subscription_id,deadline_at) SELECT notification.owner_did,gen_random_uuid(),notification.id,subscription.id,now()+interval '24 hours' FROM notification JOIN push_account_subscriptions subscription ON subscription.account_did=notification.owner_did`, info.DID); err != nil {
				t.Fatal(err)
			}
			sends := 0
			dispatcher, err := NewDeliveryDispatcher(service, backgroundSessionFunc(func(context.Context, syntax.DID, []string) (string, error) { return info.SessionID, nil }), chatPushFunc(func(_ context.Context, request push.SendRequest) (push.ProviderResult, error) {
				sends++
				if request.Chat == nil || request.Chat.OpenReference == "" || request.Category != "" || request.ActorDisplayName != "Alex" || strings.Contains(fmt.Sprintf("%+v", request), "private-dispatch-canary") {
					t.Fatal("provider input lost private routing/copy boundary")
				}
				return push.ProviderResult{Class: push.ResultSuccess}, nil
			}), DeliveryOptions{})
			if err != nil {
				t.Fatal(err)
			}
			processed, err := dispatcher.ProcessOne(t.Context())
			if err != nil || !processed {
				t.Fatalf("processed=%v err=%v", processed, err)
			}
			var state string
			if err := store.pool.QueryRow(t.Context(), `SELECT status FROM chat_push_deliveries`).Scan(&state); err != nil {
				t.Fatal(err)
			}
			expected := "succeeded"
			expectedSends := 1
			if muted {
				expected = "cancelled"
				expectedSends = 0
			}
			if state != expected || sends != expectedSends {
				t.Fatalf("state=%s sends=%d", state, sends)
			}
			if processed, err := dispatcher.ProcessOne(t.Context()); err != nil || processed || sends != expectedSends {
				t.Fatal("completed subscription delivered twice")
			}
		})
	}
}
