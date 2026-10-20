package chat

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// IT-009 / FR-012: preserve progress for unknown event variants without
// retaining their arbitrary payload; delete events retain identity only.
func TestLogDecoderPreservesUnknownProgressWithoutPrivatePayload(t *testing.T) {
	raw := []byte(`{"cursor":"head-private","logs":[{"$type":"future.event","rev":"0002","private":"private-log-canary"},{"$type":"chat.bsky.convo.defs#logDeleteMessage","rev":"0003","convoId":"direct-private","message":{"$type":"chat.bsky.convo.defs#deletedMessageView","id":"message-private","text":"deleted-text-canary","sender":{"did":"did:plc:peer"}}}]}`)
	page, err := decodeLogPage(raw)
	if err != nil || page.Cursor != "head-private" || len(page.Events) != 2 || page.Events[0].Revision != "0002" || page.Events[0].Kind != "unknown" || page.Events[1].Kind != "delete" || page.Events[1].MessageID != "message-private" {
		t.Fatalf("decoded=%+v err=%v", page, err)
	}
	encoded, _ := json.Marshal(page)
	printed := fmt.Sprintf("%+v %#v", page, page)
	for _, canary := range []string{"private-log-canary", "deleted-text-canary"} {
		if strings.Contains(string(encoded)+printed, canary) {
			t.Fatal("raw event payload retained")
		}
	}
	for _, canary := range []string{"head-private", "direct-private", "message-private", "did:plc:peer"} {
		if strings.Contains(printed, canary) {
			t.Fatal("private event routing in diagnostic format")
		}
	}
}
