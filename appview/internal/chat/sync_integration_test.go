package chat

import (
	"context"
	"fmt"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"io"
	"net/http"
	"social.craftsky/appview/internal/auth"
	"strings"
	"testing"
	"time"
)

type backgroundSessionFunc func(context.Context, syntax.DID, []string) (string, error)

func (f backgroundSessionFunc) SelectForCapability(ctx context.Context, owner syntax.DID, required []string) (string, error) {
	return f(ctx, owner, required)
}

// IT-009 / IT-010 / FR-012, FR-013: baseline is silent; a crash cannot
// commit a cursor ahead of its dedupe/logical notification transaction.
func TestSyncBaselineIsSilentAndProgressCommitsWithNotification(t *testing.T) {
	deleted := false
	service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"chatDisabled":false}`
		switch r.URL.Path {
		case "/xrpc/chat.bsky.actor.getStatus":
		case "/xrpc/chat.bsky.convo.getLog":
			if r.URL.Query().Get("limit") != "" {
				t.Fatal("unsupported getLog limit")
			}
			if deleted {
				body = `{"cursor":"0003","logs":[{"$type":"chat.bsky.convo.defs#logDeleteMessage","rev":"0003","convoId":"direct-1","message":{"id":"new-1"}}]}`
			} else if r.URL.Query().Get("cursor") == "" {
				body = `{"cursor":"0001","logs":[]}`
			} else {
				body = `{"cursor":"0002","logs":[{"$type":"chat.bsky.convo.defs#logCreateMessage","rev":"0002","convoId":"direct-1","message":{"id":"new-1","sender":{"did":"did:plc:peer"},"text":"private-sync-canary"}}]}`
			}
		case "/xrpc/chat.bsky.convo.listConvos":
			body = `{"convos":[{"id":"direct-1","status":"accepted","unreadCount":1,"members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:peer"}],"lastMessage":{"$type":"chat.bsky.convo.defs#messageView","id":"old-1","rev":"0001","sender":{"did":"did:plc:peer"},"text":"pre-enable-canary","sentAt":"2026-10-19T10:00:00Z"}}]}`
		case "/xrpc/chat.bsky.convo.listConvoRequests":
			body = `{"requests":[]}`
		case "/xrpc/chat.bsky.convo.getConvo":
			body = `{"convo":{"id":"direct-1","status":"accepted","unreadCount":1,"members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:peer"}],"lastMessage":{"$type":"chat.bsky.convo.defs#messageView","id":"new-1","rev":"0002","sender":{"did":"did:plc:peer"},"text":"private-sync-canary","sentAt":"2026-10-19T10:01:00Z"}}}`
		default:
			t.Fatalf("unexpected sync method %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	now := time.Now().UTC()
	for i := range 3 {
		if _, err := store.pool.Exec(t.Context(), `WITH installation AS (INSERT INTO push_installations(id,device_id,platform,fcm_token,active) VALUES(gen_random_uuid(),$2,'ios',$3,$4) RETURNING id) INSERT INTO push_account_subscriptions(id,installation_id,account_did,routing_id,active) SELECT gen_random_uuid(),id,$1,gen_random_uuid(),true FROM installation`, info.DID, fmt.Sprintf("synthetic-device-%d", i), fmt.Sprintf("synthetic-provider-token-%d", i), i < 2); err != nil {
			t.Fatal(err)
		}
	}
	worker, err := NewSynchronizer(service, backgroundSessionFunc(func(_ context.Context, owner syntax.DID, _ []string) (string, error) {
		if owner != info.DID {
			t.Fatal("wrong background owner")
		}
		return info.SessionID, nil
	}), SyncOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	poll := func() {
		t.Helper()
		now = now.Add(31 * time.Second)
		if err := worker.Poll(t.Context(), info.DID); err != nil {
			t.Fatal(err)
		}
	}
	for range 3 {
		poll()
	}
	var cursor string
	var notifications int
	if err := store.pool.QueryRow(t.Context(), `SELECT cursor FROM chat_sync_state WHERE owner_did=$1`, info.DID).Scan(&cursor); err != nil || cursor != "0001" {
		t.Fatalf("baseline cursor=%s err=%v", cursor, err)
	}
	if err := store.pool.QueryRow(t.Context(), `SELECT count(*) FROM chat_notifications`).Scan(&notifications); err != nil || notifications != 0 {
		t.Fatal("baseline notified", err)
	}
	if _, err := store.pool.Exec(t.Context(), `CREATE FUNCTION reject_chat_notification() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'simulated local crash'; END $$; CREATE TRIGGER reject_chat_notification BEFORE INSERT ON chat_notifications FOR EACH ROW EXECUTE FUNCTION reject_chat_notification()`); err != nil {
		t.Fatal(err)
	}
	now = now.Add(31 * time.Second)
	if err := worker.Poll(t.Context(), info.DID); err == nil {
		t.Fatal("injected crash ignored")
	}
	if err := store.pool.QueryRow(t.Context(), `SELECT cursor FROM chat_sync_state WHERE owner_did=$1`, info.DID).Scan(&cursor); err != nil || cursor != "0001" {
		t.Fatal("cursor advanced ahead of outbox", err)
	}
	if _, err := store.pool.Exec(t.Context(), `DROP TRIGGER reject_chat_notification ON chat_notifications`); err != nil {
		t.Fatal(err)
	}
	poll()
	poll()
	if err := store.pool.QueryRow(t.Context(), `SELECT cursor,(SELECT count(*) FROM chat_notifications) FROM chat_sync_state WHERE owner_did=$1`, info.DID).Scan(&cursor, &notifications); err != nil || cursor != "0002" || notifications != 1 {
		t.Fatalf("committed cursor=%s notifications=%d err=%v", cursor, notifications, err)
	}
	var stored string
	if err := store.pool.QueryRow(t.Context(), `SELECT coalesce(string_agg(row_to_json(n)::text,''),'') FROM chat_notifications n`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored, "private-sync-canary") || strings.Contains(stored, "pre-enable-canary") {
		t.Fatal("message text persisted")
	}
	var deliveries, social int
	if err := store.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM chat_push_deliveries),(SELECT count(*) FROM notification_events)`).Scan(&deliveries, &social); err != nil || deliveries != 2 || social != 0 {
		t.Fatalf("private fanout=%d social events=%d err=%v", deliveries, social, err)
	}

	deleted = true
	poll()
	var disposition string
	if err := store.pool.QueryRow(t.Context(), `SELECT cursor,(SELECT disposition FROM chat_message_dedupe WHERE owner_did=$1 AND conversation_id='direct-1' AND message_id='new-1'),(SELECT count(*) FROM chat_push_deliveries WHERE status IN('pending','retry','leased')) FROM chat_sync_state WHERE owner_did=$1`, info.DID).Scan(&cursor, &disposition, &deliveries); err != nil || cursor != "0003" || disposition != "deleted" || deliveries != 0 {
		t.Fatalf("delete checkpoint cursor=%s disposition=%s pending=%d err=%v", cursor, disposition, deliveries, err)
	}
}
