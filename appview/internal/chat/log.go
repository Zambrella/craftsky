package chat

import (
	"encoding/json"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"sort"
)

// Events contain reviewed operational fields only; unknown prose and message
// text never survive decoding. Revisions are private provider cursors.
type logEvent struct {
	Kind           string
	Revision       string
	ConversationID ConversationID
	MessageID      MessageID
	SenderDID      syntax.DID
}
type logPage struct {
	Cursor string
	Events []logEvent
}

func (logEvent) String() string   { return "ChatLogEvent([REDACTED])" }
func (logEvent) GoString() string { return "ChatLogEvent([REDACTED])" }
func (logPage) String() string    { return "ChatLogPage([REDACTED])" }
func (logPage) GoString() string  { return "ChatLogPage([REDACTED])" }

func decodeLogPage(raw []byte) (logPage, error) {
	var wire struct {
		Cursor string            `json:"cursor"`
		Logs   []json.RawMessage `json:"logs"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return logPage{}, errorsProviderCause(err)
	}
	if wire.Logs == nil || len(wire.Cursor) > 8192 {
		return logPage{}, ErrProviderResponse
	}
	result := logPage{Cursor: wire.Cursor, Events: make([]logEvent, 0, len(wire.Logs))}
	for _, item := range wire.Logs {
		var envelope struct {
			Type         string `json:"$type"`
			Revision     string `json:"rev"`
			Conversation string `json:"convoId"`
			Message      *struct {
				ID     string `json:"id"`
				Sender *struct {
					DID string `json:"did"`
				} `json:"sender"`
			} `json:"message"`
		}
		if err := json.Unmarshal(item, &envelope); err != nil {
			return logPage{}, errorsProviderCause(err)
		}
		if envelope.Revision == "" || len(envelope.Revision) > 512 {
			return logPage{}, ErrProviderResponse
		}
		event := logEvent{Kind: "unknown", Revision: envelope.Revision}
		switch envelope.Type {
		case "chat.bsky.convo.defs#logCreateMessage":
			event.Kind = "message"
		case "chat.bsky.convo.defs#logDeleteMessage":
			event.Kind = "delete"
		case "chat.bsky.convo.defs#logAcceptConvo":
			event.Kind = "accept"
		case "chat.bsky.convo.defs#logLeaveConvo":
			event.Kind = "leave"
		case "chat.bsky.convo.defs#logBeginConvo", "chat.bsky.convo.defs#logMuteConvo", "chat.bsky.convo.defs#logUnmuteConvo", "chat.bsky.convo.defs#logReadConvo", "chat.bsky.convo.defs#logReadMessage":
			event.Kind = "refresh"
		}
		if event.Kind != "unknown" {
			id, err := ParseConversationID(envelope.Conversation)
			if err != nil {
				return logPage{}, ErrProviderResponse
			}
			event.ConversationID = id
			if envelope.Message != nil {
				event.MessageID, err = ParseMessageID(envelope.Message.ID)
				if err != nil {
					return logPage{}, ErrProviderResponse
				}
				if envelope.Message.Sender != nil {
					event.SenderDID, err = syntax.ParseDID(envelope.Message.Sender.DID)
					if err != nil {
						return logPage{}, ErrProviderResponse
					}
				}
			}
			if (event.Kind == "message" || event.Kind == "delete") && event.MessageID == "" {
				return logPage{}, ErrProviderResponse
			}
		}
		result.Events = append(result.Events, event)
	}
	sort.SliceStable(result.Events, func(i, j int) bool { return result.Events[i].Revision < result.Events[j].Revision })
	return result, nil
}
