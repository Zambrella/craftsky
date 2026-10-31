package chat

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"io"
	"net/http"
	"social.craftsky/appview/internal/auth"
	"strings"
	"testing"
	"time"
)

// IT-009 / IT-012 / FR-012, FR-013: transient failures keep the saved
// cursor; missed accepted messages coalesce into one durable catch-up.
func TestRecoveryResumesSavedCursorAndCoalescesAcceptedConversation(t *testing.T) {
	headCalls := 0
	fail := false
	resume := false
	service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"chatDisabled":false}`
		switch r.URL.Path {
		case "/xrpc/chat.bsky.actor.getStatus":
		case "/xrpc/chat.bsky.convo.listConvos":
			body = `{"convos":[]}`
		case "/xrpc/chat.bsky.convo.listConvoRequests":
			body = `{"requests":[]}`
		case "/xrpc/chat.bsky.convo.getLog":
			cursor := r.URL.Query().Get("cursor")
			if cursor == "" {
				headCalls++
				body = `{"cursor":"0001","logs":[]}`
			} else {
				if fail {
					return nil, errors.New("synthetic transport interruption")
				}
				if resume && cursor == "0001" {
					body = `{"logs":[{"$type":"chat.bsky.convo.defs#logCreateMessage","rev":"0002","convoId":"accepted-1","message":{"id":"missed-1","sender":{"did":"did:plc:peer"}}},{"$type":"chat.bsky.convo.defs#logCreateMessage","rev":"0003","convoId":"accepted-1","message":{"id":"missed-2","sender":{"did":"did:plc:peer"}}}]}`
				} else {
					body = `{"logs":[]}`
				}
			}
		case "/xrpc/chat.bsky.convo.getConvo":
			body = `{"convo":{"id":"accepted-1","status":"accepted","unreadCount":2,"members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:peer"}]}}`
		default:
			t.Fatalf("unexpected method %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	now := time.Now().UTC()
	worker, err := NewSynchronizer(service, backgroundSessionFunc(func(context.Context, syntax.DID, []string) (string, error) { return info.SessionID, nil }), SyncOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	poll := func() error { now = now.Add(31 * time.Second); return worker.Poll(t.Context(), info.DID) }
	for range 3 {
		if err := poll(); err != nil {
			t.Fatal(err)
		}
	}
	fail = true
	if err := poll(); err == nil {
		t.Fatal("interruption ignored")
	}
	fail = false
	resume = true
	for range 2 {
		if err := poll(); err != nil {
			t.Fatal(err)
		}
	}
	var total int
	var kind, cursor string
	if err := store.pool.QueryRow(t.Context(), `SELECT count(*),coalesce(min(kind),'') FROM chat_notifications`).Scan(&total, &kind); err != nil {
		t.Fatal(err)
	}
	if err := store.pool.QueryRow(t.Context(), `SELECT cursor FROM chat_sync_state WHERE owner_did=$1`, info.DID).Scan(&cursor); err != nil {
		t.Fatal(err)
	}
	if total != 1 || kind != "catchUp" || cursor != "0003" || headCalls != 1 {
		t.Fatalf("alerts=%d kind=%s cursor=%s headCalls=%d", total, kind, cursor, headCalls)
	}
}

// IT-009 / IT-012: an explicitly expired cursor reconciles all direct
// pages before replacing progress, preserving baseline suppression.
func TestExpiredCursorReconcilesPagesBeforeInstallingNewHead(t *testing.T) {
	service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"chatDisabled":false}`
		status := 200
		switch r.URL.Path {
		case "/xrpc/chat.bsky.actor.getStatus":
		case "/xrpc/chat.bsky.convo.getLog":
			if r.URL.Query().Get("cursor") == "expired" {
				status = 400
				body = `{"error":"InvalidCursor","message":"private expired prose"}`
			} else {
				body = `{"cursor":"0010","logs":[]}`
			}
		case "/xrpc/chat.bsky.convo.listConvos":
			id := "new-message"
			next := `,"cursor":"second-private-page"`
			if r.URL.Query().Get("cursor") == "second-private-page" {
				id = "baseline-message"
				next = ""
			}
			body = `{"convos":[{"id":"accepted-1","status":"accepted","unreadCount":2,"members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:peer"}],"lastMessage":{"$type":"chat.bsky.convo.defs#messageView","id":"` + id + `","rev":"0010","sender":{"did":"did:plc:peer"},"text":"private-recovery-canary","sentAt":"2026-10-19T12:00:00Z"}}]` + next + `}`
		case "/xrpc/chat.bsky.convo.listConvoRequests":
			body = `{"requests":[]}`
		default:
			t.Fatalf("unexpected recovery method %s", r.URL.Path)
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	if _, err := store.pool.Exec(t.Context(), `INSERT INTO chat_sync_state(owner_did,phase,cursor) VALUES($1,'ready','expired');`, info.DID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(t.Context(), `INSERT INTO chat_message_dedupe(owner_did,conversation_id,message_id,disposition,observed_revision) VALUES($1,'accepted-1','baseline-message','baseline','0001')`, info.DID); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	worker, err := NewSynchronizer(service, backgroundSessionFunc(func(context.Context, syntax.DID, []string) (string, error) { return info.SessionID, nil }), SyncOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	for step := 0; step < 5; step++ {
		now = now.Add(31 * time.Second)
		if err := worker.Poll(t.Context(), info.DID); err != nil {
			t.Fatal(err)
		}
		var cursor string
		if err := store.pool.QueryRow(t.Context(), `SELECT cursor FROM chat_sync_state WHERE owner_did=$1`, info.DID).Scan(&cursor); err != nil {
			t.Fatal(err)
		}
		if step < 4 && cursor != "expired" {
			t.Fatal("new head installed before reconciliation finished")
		}
		if step == 4 && cursor != "0010" {
			t.Fatalf("recovered cursor=%s", cursor)
		}
	}
	var total int
	var kind string
	if err := store.pool.QueryRow(t.Context(), `SELECT count(*),min(kind) FROM chat_notifications`).Scan(&total, &kind); err != nil || total != 1 || kind != "catchUp" {
		t.Fatalf("recovery total=%d kind=%s err=%v", total, kind, err)
	}
}
