package chat

import (
	"bytes"
	"encoding/json"
	"github.com/google/uuid"
	"io"
	"net/http"
	"social.craftsky/appview/internal/auth"
	"strconv"
	"strings"
	"testing"
)

// IT-004 / FR-010, BR-003: authenticated message reads establish the
// evidence. No consumer-supplied sender/reference or copied transcript exists.
func TestReportUsesCurrentMessageReferenceAndSingleAttempt(t *testing.T) {
	reports := 0
	var submitted map[string]any
	service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{}`
		switch r.URL.Path {
		case "/xrpc/chat.bsky.actor.getStatus":
			body = `{"chatDisabled":false}`
		case "/xrpc/chat.bsky.convo.getConvo":
			body = `{"convo":{"id":"direct-1","status":"accepted","members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:sender"}]}}`
		case "/xrpc/chat.bsky.convo.getMessages":
			body = `{"messages":[{"$type":"chat.bsky.convo.defs#messageView","id":"message-1","rev":"rev-1","sender":{"did":"did:plc:sender"},"text":"private-transcript-canary","sentAt":"2026-10-19T12:00:00Z"}]}`
		case "/xrpc/com.atproto.moderation.createReport":
			reports++
			if r.Header.Get("Atproto-Proxy") != auth.ChatReportAudience {
				t.Fatal("report sent to wrong audience")
			}
			if err := json.NewDecoder(r.Body).Decode(&submitted); err != nil {
				t.Fatal(err)
			}
			body = `{"id":17}`
		default:
			t.Fatalf("unexpected upstream method %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	ledger, err := NewOperationLedger(store, 1, map[int][]byte{1: bytes.Repeat([]byte{1}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	service.operations = ledger
	id := uuid.New()
	reason := "bounded explanation"
	receipt, err := service.Report(t.Context(), info, id, "direct-1", "message-1", "com.atproto.moderation.defs#reasonOther", &reason)
	if err != nil || receipt.State != "confirmed" {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
	if _, err := service.Report(t.Context(), info, id, "direct-1", "message-1", "com.atproto.moderation.defs#reasonOther", &reason); err != nil {
		t.Fatal(err)
	}
	if reports != 1 {
		t.Fatalf("report calls=%d", reports)
	}
	subject, ok := submitted["subject"].(map[string]any)
	if !ok || subject["did"] != "did:plc:sender" || subject["messageId"] != "message-1" || subject["convoId"] != "direct-1" || subject["$type"] != "chat.bsky.convo.defs#messageRef" {
		t.Fatalf("invalid message reference: %v", subject)
	}
	raw, _ := json.Marshal(submitted)
	if strings.Contains(string(raw), "private-transcript-canary") {
		t.Fatal("transcript submitted")
	}
	if _, err := service.Report(t.Context(), info, uuid.New(), "direct-1", "forged-message", "com.atproto.moderation.defs#reasonOther", nil); err == nil || reports != 1 {
		t.Fatal("forged reference submitted")
	}
	var social int
	if err := store.pool.QueryRow(t.Context(), `SELECT count(*) FROM moderation_reports`).Scan(&social); err != nil || social != 0 {
		t.Fatalf("social report count=%d err=%v", social, err)
	}
}

// IT-004 / FR-010: a message loaded deep in history remains reportable
// without an unbounded scan or a client-supplied sender reference.
func TestReportRevalidatesMessageOnPreviouslyLoadedOlderPage(t *testing.T) {
	reports := 0
	service, _, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"chatDisabled":false}`
		switch r.URL.Path {
		case "/xrpc/chat.bsky.actor.getStatus":
		case "/xrpc/chat.bsky.convo.getConvo":
			body = `{"convo":{"id":"direct-1","status":"accepted","members":[{"did":"did:plc:chat-service-owner"},{"did":"did:plc:sender"}]}}`
		case "/xrpc/chat.bsky.convo.getMessages":
			page, _ := strconv.Atoi(r.URL.Query().Get("cursor"))
			if page == 6 {
				body = `{"messages":[{"$type":"chat.bsky.convo.defs#messageView","id":"old-message","rev":"old-rev","sender":{"did":"did:plc:sender"},"text":"private old text","sentAt":"2026-10-19T11:00:00Z"}]}`
			} else {
				body = `{"messages":[],"cursor":"` + strconv.Itoa(page+1) + `"}`
			}
		case "/xrpc/com.atproto.moderation.createReport":
			reports++
			body = `{"id":18}`
		default:
			t.Fatal("unexpected method")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	var cursor Continuation
	for range 7 {
		page, err := service.History(t.Context(), info, "direct-1", cursor, 50)
		if err != nil {
			t.Fatal(err)
		}
		cursor = page.Cursor
	}
	receipt, err := service.Report(t.Context(), info, uuid.New(), "direct-1", "old-message", "com.atproto.moderation.defs#reasonOther", nil)
	if err != nil || receipt.State != "confirmed" || reports != 1 {
		t.Fatalf("receipt=%+v reports=%d err=%v", receipt, reports, err)
	}
}
