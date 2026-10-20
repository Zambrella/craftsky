package api

import (
	"context"
	"crypto/sha256"
	"errors"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chat"
	"social.craftsky/appview/internal/ctxkeys"
	"strings"
	"testing"
)

type chatSendFunc func(context.Context, auth.AuthInfo, uuid.UUID, chat.ConversationID, string) (chat.SendOutcome, error)

func (f chatSendFunc) Send(ctx context.Context, info auth.AuthInfo, id uuid.UUID, conversation chat.ConversationID, text string) (chat.SendOutcome, error) {
	return f(ctx, info, id, conversation, text)
}

type chatListReader struct {
	ChatConsumer
	calls int
}

func (reader *chatListReader) ListConversations(_ context.Context, info auth.AuthInfo, status string, cursor chat.Continuation, limit int) (chat.ConversationPage, error) {
	reader.calls++
	if info.ChildTokenHash == ([32]byte{}) || status != "request" || cursor != "" || limit != 50 {
		panic("wrong consumer list contract")
	}
	return chat.ConversationPage{Items: []chat.Conversation{}}, nil
}

// IT-019 / NFR-003: automatic reads admit only their fixed, bounded query
// contract and never permit caller authority or upstream cursors.
func TestChatConsumerStrictPaginationContract(t *testing.T) {
	reader := &chatListReader{}
	handler := ChatConsumerHandler(reader, "list")
	for _, test := range []struct {
		query string
		want  int
	}{
		{"?status=request", 200}, {"?status=request&ownerDid=did:plc:other", 400}, {"?status=request&limit=101", 400}, {"?status=request&limit=0", 400}, {"?status=request&cursor=raw-provider-cursor", 400}, {"?status=request&status=accepted", 400},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, chatContractRequest("GET", "/v1/chat/conversations"+test.query, ""))
		if w.Code != test.want {
			t.Fatalf("query=%s status=%d body=%s", test.query, w.Code, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("private read cacheable")
		}
	}
	if reader.calls != 1 {
		t.Fatalf("list calls=%d", reader.calls)
	}
}

func chatContractRequest(method, path, body string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	ctx := ctxkeys.WithDID(r.Context(), "did:plc:chat-contract-owner")
	ctx = ctxkeys.WithOAuthSessionID(ctx, "synthetic-parent")
	ctx = ctxkeys.WithChildTokenHash(ctx, sha256.Sum256([]byte("synthetic-bearer")))
	return r.WithContext(ctx)
}

// IT-019 / NFR-003: clients submit only the fixed consumer contract.
// Unknown authority controls and invalid text never reach the service.
func TestChatSendContractRejectsArbitraryControls(t *testing.T) {
	id := uuid.New()
	calls := 0
	h := ChatSendHandler(chatSendFunc(func(_ context.Context, info auth.AuthInfo, operation uuid.UUID, conversation chat.ConversationID, text string) (chat.SendOutcome, error) {
		calls++
		if info.DID != "did:plc:chat-contract-owner" || info.ChildTokenHash == ([32]byte{}) || string(conversation) != "direct-1" || text != "keep my exact text" {
			t.Fatal("consumer authority/text changed")
		}
		return chat.SendOutcome{OperationReceipt: chat.OperationReceipt{OperationID: operation, Kind: "send", State: "uncertain"}}, errors.New("private-provider-canary")
	}))
	for _, test := range []struct {
		body string
		want int
	}{
		{`{"operationId":"` + id.String() + `","text":"keep my exact text"}`, 202},
		{`{"operationId":"` + id.String() + `","text":"keep my exact text","senderDid":"did:plc:other"}`, 400},
		{`{"operationId":"` + id.String() + `","text":"keep my exact text","audience":"https://evil.example"}`, 400},
		{`{"operationId":"` + id.String() + `","text":"   "}`, 400},
		{`{"text":"missing operation"}`, 400},
		{`{"operationId":"` + id.String() + `","text":"\ud800"}`, 400},
		{`{"operationId":"` + id.String() + `","text":"\udc00"}`, 400},
		{`{"operationId":"` + id.String() + `","text":"\ud800x"}`, 400},
	} {
		r := chatContractRequest("POST", "/v1/chat/conversations/direct-1/messages", test.body)
		r.SetPathValue("id", "direct-1")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != test.want {
			t.Fatalf("status=%d expected=%d body=%s", w.Code, test.want, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "private-provider-canary") || strings.Contains(w.Body.String(), "keep my exact text") {
			t.Fatal("private text/prose copied into receipt")
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("private response is cacheable")
		}
	}
	if calls != 1 {
		t.Fatalf("service calls=%d", calls)
	}
}
