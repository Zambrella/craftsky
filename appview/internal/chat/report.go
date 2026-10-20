package chat

import (
	"errors"
	upstream "github.com/bluesky-social/indigo/api/chat"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"strings"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

var ErrUnsupportedReportReason = errors.New("unsupported message report reason")
var ErrInaccessibleConversation = errors.New("messaging conversation unavailable")

func validOpaqueID(value string) bool {
	if value == "" || len(value) > 512 || !utf8.ValidString(value) || strings.ContainsAny(value, "/\\?#% ") {
		return false
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func directPeer(owner syntax.DID, conversation *upstream.ConvoDefs_ConvoView) (syntax.DID, error) {
	if conversation == nil || !validOpaqueID(conversation.Id) || len(conversation.Members) != 2 || conversation.Status == nil || (*conversation.Status != "accepted" && *conversation.Status != "request") {
		return "", ErrInaccessibleConversation
	}
	if conversation.Kind != nil && (conversation.Kind.ConvoDefs_DirectConvo == nil || conversation.Kind.ConvoDefs_GroupConvo != nil) {
		return "", ErrInaccessibleConversation
	}
	var peer syntax.DID
	seen := false
	for _, member := range conversation.Members {
		if member == nil {
			return "", ErrInaccessibleConversation
		}
		did, err := syntax.ParseDID(member.Did)
		if err != nil {
			return "", ErrInaccessibleConversation
		}
		if did == owner {
			if seen {
				return "", ErrInaccessibleConversation
			}
			seen = true
		} else {
			peer = did
		}
	}
	if !seen || peer == "" {
		return "", ErrInaccessibleConversation
	}
	return peer, nil
}

// The caller must fetch the message from this conversation's authenticated
// getMessages response. Neither the reporter nor a consumer-supplied DID is
// used as the subject author; no transcript enters the reference.
func BuildReportEvidence(owner syntax.DID, conversation *upstream.ConvoDefs_ConvoView, message *upstream.ConvoDefs_MessageView) (*upstream.ConvoDefs_MessageRef, error) {
	peer, err := directPeer(owner, conversation)
	if err != nil {
		return nil, err
	}
	if message == nil || message.Sender == nil || !validOpaqueID(message.Id) {
		return nil, ErrInaccessibleConversation
	}
	sender, err := syntax.ParseDID(message.Sender.Did)
	if err != nil || (sender != owner && sender != peer) {
		return nil, ErrInaccessibleConversation
	}
	return &upstream.ConvoDefs_MessageRef{LexiconTypeID: "chat.bsky.convo.defs#messageRef", ConvoId: conversation.Id, Did: sender.String(), MessageId: message.Id}, nil
}

// ValidateReportInput accepts the six consumer report categories exposed by
// CraftSky. Appeals are not new-message reports. Optional prose is bounded
// independently in graphemes and UTF-8 bytes and is never modified.
func ValidateReportInput(reasonType string, reason *string) error {
	switch reasonType {
	case "com.atproto.moderation.defs#reasonSpam",
		"com.atproto.moderation.defs#reasonViolation",
		"com.atproto.moderation.defs#reasonMisleading",
		"com.atproto.moderation.defs#reasonSexual",
		"com.atproto.moderation.defs#reasonRude",
		"com.atproto.moderation.defs#reasonOther":
	default:
		return ErrUnsupportedReportReason
	}
	if reason != nil {
		if !utf8.ValidString(*reason) {
			return ErrTextInvalidUnicode
		}
		if len(*reason) > 20_000 {
			return ErrTextByteLimit
		}
		if uniseg.GraphemeClusterCount(*reason) > 2_000 {
			return ErrTextGraphemeLimit
		}
	}
	return nil
}
