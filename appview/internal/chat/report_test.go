package chat

import (
	"encoding/json"
	upstream "github.com/bluesky-social/indigo/api/chat"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"strings"
	"testing"
)

// UT-005 / FR-010 / AC-010.
func TestReportInputValidation(t *testing.T) {
	byteBoundary := "a" + strings.Repeat("\u0301", 9999)
	for _, test := range []struct {
		name, reasonType string
		reason           *string
		valid            bool
	}{
		{"no explanation", "com.atproto.moderation.defs#reasonSpam", nil, true},
		{"empty optional explanation", "com.atproto.moderation.defs#reasonOther", reportText(""), true},
		{"missing reason type", "", nil, false},
		{"unsupported reason type", "example.invalid#private-canary", nil, false},
		{"1999 graphemes", "com.atproto.moderation.defs#reasonRude", reportText(strings.Repeat("x", 1999)), true},
		{"2000 graphemes", "com.atproto.moderation.defs#reasonRude", reportText(strings.Repeat("x", 2000)), true},
		{"2001 graphemes", "com.atproto.moderation.defs#reasonRude", reportText(strings.Repeat("x", 2001)), false},
		{"19999 bytes", "com.atproto.moderation.defs#reasonOther", &byteBoundary, true},
		{"20000 bytes", "com.atproto.moderation.defs#reasonOther", reportText(byteBoundary + "b"), true},
		{"20001 bytes", "com.atproto.moderation.defs#reasonOther", reportText(byteBoundary + "bc"), false},
		{"malformed Unicode", "com.atproto.moderation.defs#reasonOther", reportText(string([]byte{0xff})), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := ValidateReportInput(test.reasonType, test.reason) == nil; got != test.valid {
				t.Fatalf("valid=%t want=%t", got, test.valid)
			}
		})
	}
}

// UT-005 / FR-010: the authoritative sender differs from the reporter;
// evidence is a message reference, never a copied transcript.
func TestReportEvidenceUsesAccessibleMessageSender(t *testing.T) {
	owner := syntax.DID("did:plc:reporter")
	status := "accepted"
	conversation := &upstream.ConvoDefs_ConvoView{Id: "direct-1", Status: &status, Members: []*upstream.ActorDefs_ProfileViewBasic{{Did: owner.String()}, {Did: "did:plc:sender"}}}
	message := &upstream.ConvoDefs_MessageView{Id: "message-1", Text: "private-transcript-canary", Sender: &upstream.ConvoDefs_MessageViewSender{Did: "did:plc:sender"}}
	evidence, err := BuildReportEvidence(owner, conversation, message)
	if err != nil || evidence.Did != "did:plc:sender" || evidence.ConvoId != "direct-1" || evidence.MessageId != "message-1" {
		t.Fatalf("incorrect evidence: %+v %v", evidence, err)
	}
	raw, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), message.Text) || !strings.Contains(string(raw), `"$type":"chat.bsky.convo.defs#messageRef"`) {
		t.Fatal("report evidence has copied or untyped content")
	}
	message.Sender.Did = "did:plc:unrelated"
	if _, err := BuildReportEvidence(owner, conversation, message); err == nil {
		t.Fatal("forged sender accepted")
	}
	message.Sender.Did = "did:plc:sender"
	if _, err := BuildReportEvidence("did:plc:non-member", conversation, message); err == nil {
		t.Fatal("inaccessible conversation accepted")
	}
	conversation.Kind = &upstream.ConvoDefs_ConvoView_Kind{ConvoDefs_GroupConvo: &upstream.ConvoDefs_GroupConvo{}}
	if _, err := BuildReportEvidence(owner, conversation, message); err == nil {
		t.Fatal("group report entered the direct-only flow")
	}
}

func reportText(value string) *string { return &value }
