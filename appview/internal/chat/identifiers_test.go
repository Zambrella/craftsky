package chat

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"testing"
)

// IT-019 / NFR-003, NFR-002: bounded opaque identifiers keep their wire
// value while accidental formatting cannot reveal private routing.
func TestPrivateIdentifiersValidateAndRedactDiagnostics(t *testing.T) {
	const value = "private-conversation-canary"
	id, err := ParseConversationID(value)
	if err != nil {
		t.Fatal(err)
	}
	message, err := ParseMessageID("private-message-canary")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(struct {
		Conversation ConversationID `json:"conversationId"`
		Message      MessageID      `json:"messageId"`
	}{id, message})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), value) {
		t.Fatal("wire identifier lost")
	}
	if strings.Contains(fmt.Sprintf("%v %#v %v %#v", id, id, message, message), "canary") {
		t.Fatal("private routing formatted")
	}
	for _, invalid := range []string{"", strings.Repeat("x", 513), "a/b", "a?target=x", "a%2Fb", "a\nb", "https://evil.example"} {
		if _, err := ParseConversationID(invalid); err == nil {
			t.Fatal("invalid conversation ID accepted")
		}
		if _, err := ParseMessageID(invalid); err == nil {
			t.Fatal("invalid message ID accepted")
		}
	}
}

func TestOperationReceiptRedactsDiagnosticIdentity(t *testing.T) {
	id := uuid.MustParse("00000000-0000-4000-8000-000000000123")
	receipt := OperationReceipt{OperationID: id, Kind: "send", State: "uncertain"}
	for _, value := range []any{receipt, SendOutcome{OperationReceipt: receipt, Message: &Message{Text: "private-body-canary"}}} {
		formatted := fmt.Sprintf("%v %#v", value, value)
		if strings.Contains(formatted, id.String()) || strings.Contains(formatted, "private-body-canary") {
			t.Fatal("private receipt formatted")
		}
	}
	raw, _ := json.Marshal(receipt)
	if !strings.Contains(string(raw), id.String()) {
		t.Fatal("wire receipt identity lost")
	}
}
