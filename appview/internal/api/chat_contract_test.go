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

type chatDetailsReader struct {
	ChatConsumer
	calls int
}

func (reader *chatDetailsReader) Details(_ context.Context, info auth.AuthInfo, id chat.ConversationID) (chat.ConversationDetails, error) {
	reader.calls++
	if info.ChildTokenHash == ([32]byte{}) || id != "direct-1" {
		panic("wrong details authority")
	}
	return chat.ConversationDetails{Conversation: chat.Conversation{ID: id, Status: "request"}, Messages: chat.MessagePage{Items: []chat.Message{}}}, nil
}

func TestChatDetailsFixedPrivateContract(t *testing.T) {
	reader := &chatDetailsReader{}
	handler := ChatConsumerHandler(reader, "details")
	for _, test := range []struct {
		query, body string
		want        int
	}{
		{"", "", 200}, {"?limit=1", "", 400}, {"?cursor=private-cursor", "", 400}, {"?ownerDid=did:plc:other", "", 400}, {"", "{}", 400},
	} {
		w := httptest.NewRecorder()
		r := chatContractRequest("GET", "/v1/chat/conversations/direct-1/details"+test.query, test.body)
		r.SetPathValue("id", "direct-1")
		handler.ServeHTTP(w, r)
		if w.Code != test.want {
			t.Fatalf("status=%d want=%d", w.Code, test.want)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("details cacheable")
		}
		if test.want == 200 && (!strings.Contains(w.Body.String(), `"conversation":`) || !strings.Contains(w.Body.String(), `"messages":{"items":[]}`)) {
			t.Fatal("details wire shape changed")
		}
	}
	if reader.calls != 1 {
		t.Fatalf("details calls=%d", reader.calls)
	}
}

type chatReportReader struct {
	ChatConsumer
	conversationReports int
	messageReports      int
}

type chatDeleteReader struct {
	ChatConsumer
	calls int
}

func (reader *chatDeleteReader) DeleteMessage(_ context.Context, info auth.AuthInfo, conversation chat.ConversationID, message chat.MessageID) error {
	if info.ChildTokenHash == ([32]byte{}) || conversation != "direct-1" || message != "own-1" {
		panic("wrong delete authority or target")
	}
	reader.calls++
	return nil
}

func TestChatDeleteMessageContract(t *testing.T) {
	reader := &chatDeleteReader{}
	handler := ChatConsumerHandler(reader, "deleteMessage")
	for _, tc := range []struct {
		query, body, message string
		status               int
	}{
		{"", "", "own-1", 200},
		{"?senderDid=did:plc:other", "", "own-1", 400},
		{"", `{"forEveryone":true}`, "own-1", 400},
		{"", "", "", 400},
		{"", "", "bad/id", 400},
	} {
		r := chatContractRequest("DELETE", "/v1/chat/conversations/direct-1/messages/own-1"+tc.query, tc.body)
		r.SetPathValue("id", "direct-1")
		r.SetPathValue("messageId", tc.message)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.status || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("status=%d expected=%d body=%s", w.Code, tc.status, w.Body.String())
		}
	}
	if reader.calls != 1 {
		t.Fatalf("delete calls=%d", reader.calls)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("DELETE", "/v1/chat/conversations/direct-1/messages/own-1", nil))
	if w.Code != 401 || reader.calls != 1 {
		t.Fatal("unauthenticated delete dispatched")
	}
}

func (reader *chatReportReader) ReportConversation(_ context.Context, info auth.AuthInfo, operation uuid.UUID, conversation chat.ConversationID, reasonType string, reason *string) (chat.OperationReceipt, error) {
	if info.ChildTokenHash == ([32]byte{}) || conversation != "direct-1" || reasonType != "com.atproto.moderation.defs#reasonSpam" {
		panic("wrong conversation report contract")
	}
	reader.conversationReports++
	return chat.OperationReceipt{OperationID: operation, Kind: "report", State: "confirmed"}, nil
}

func (reader *chatReportReader) Report(_ context.Context, _ auth.AuthInfo, operation uuid.UUID, _ chat.ConversationID, _ chat.MessageID, _ string, _ *string) (chat.OperationReceipt, error) {
	reader.messageReports++
	return chat.OperationReceipt{OperationID: operation, Kind: "report", State: "confirmed"}, nil
}

// IT-004: the consumer selects a fixed target kind, never upstream evidence.
func TestChatConversationReportContract(t *testing.T) {
	reader := &chatReportReader{}
	handler := ChatConsumerHandler(reader, "report")
	base := `"operationId":"` + uuid.NewString() + `","conversationId":"direct-1","reasonType":"com.atproto.moderation.defs#reasonSpam"`
	for _, tc := range []struct {
		extra  string
		status int
	}{
		{`,"subjectType":"conversation"`, 200},
		{`,"messageId":"message-1"`, 200},
		{`,"subjectType":"message","messageId":"message-1"`, 200},
		{`,"subjectType":"conversation","messageId":"message-1"`, 400},
		{`,"subjectType":"unknown"`, 400},
		{``, 400},
		{`,"subjectType":"conversation","did":"did:plc:forged"`, 400},
		{`,"subjectType":"conversation","transcript":"private-canary"`, 400},
		{`,"subjectType":"conversation","subject":{"did":"did:plc:forged"}`, 400},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, chatContractRequest("POST", "/v1/chat/reports", "{"+base+tc.extra+"}"))
		if w.Code != tc.status {
			t.Fatalf("case %s: status=%d body=%s", tc.extra, w.Code, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("report response cacheable")
		}
	}
	if reader.conversationReports != 1 || reader.messageReports != 2 {
		t.Fatalf("unexpected report dispatch counts: %d/%d", reader.conversationReports, reader.messageReports)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("POST", "/v1/chat/reports", strings.NewReader("{"+base+`,"subjectType":"conversation"}`)))
	if w.Code != 401 || reader.conversationReports != 1 {
		t.Fatal("unauthenticated conversation report dispatched")
	}
}
