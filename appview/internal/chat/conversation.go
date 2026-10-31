package chat

import (
	"context"
	upstream "github.com/bluesky-social/indigo/api/chat"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
)

type Actor struct {
	DID         syntax.DID    `json:"did"`
	Handle      syntax.Handle `json:"handle,omitempty"`
	DisplayName string        `json:"displayName,omitempty"`
	Avatar      string        `json:"avatar,omitempty"`
}

func (Actor) String() string   { return "ChatActor([REDACTED])" }
func (Actor) GoString() string { return "ChatActor([REDACTED])" }

type Conversation struct {
	ID          ConversationID `json:"id"`
	Revision    string         `json:"revision"`
	Status      string         `json:"status"`
	Muted       bool           `json:"muted"`
	UnreadCount int64          `json:"unreadCount"`
	Peer        Actor          `json:"peer"`
	Members     []Actor        `json:"members"`
	LastMessage *Message       `json:"lastMessage,omitempty"`
	CanSend     bool           `json:"canSend"`
}

func (service *Service) Conversation(ctx context.Context, info auth.AuthInfo, id ConversationID) (Conversation, error) {
	var result Conversation
	if !validOpaqueID(string(id)) {
		return result, ErrInvalidIdentifier
	}
	err := service.withRPC(ctx, info, []syntax.NSID{"chat.bsky.convo.getConvo"}, func(ctx context.Context, transport *Transport) error {
		current, err := service.supportedConversation(ctx, transport, info.DID, id)
		if err != nil {
			return err
		}
		result, err = conversationFromView(info.DID, current)
		if err != nil {
			return err
		}
		blocked, err := service.blocked(ctx, info.DID, result.Peer.DID)
		result.CanSend = result.Status == "accepted" && !blocked
		return err
	})
	return result, err
}

func (Conversation) String() string   { return "Conversation([REDACTED])" }
func (Conversation) GoString() string { return "Conversation([REDACTED])" }
func (Message) String() string        { return "Message([REDACTED])" }
func (Message) GoString() string      { return "Message([REDACTED])" }

func actorFromView(view *upstream.ActorDefs_ProfileViewBasic) Actor {
	did, _ := syntax.ParseDID(view.Did)
	handle, _ := syntax.ParseHandle(view.Handle)
	actor := Actor{DID: did, Handle: handle}
	if view.DisplayName != nil && len(*view.DisplayName) <= 640 {
		actor.DisplayName = *view.DisplayName
	}
	if view.Avatar != nil && len(*view.Avatar) <= 2048 {
		actor.Avatar = *view.Avatar
	}
	return actor
}

func messageFromView(text *upstream.ConvoDefs_MessageView, deleted *upstream.ConvoDefs_DeletedMessageView, system *upstream.ConvoDefs_SystemMessageView) *Message {
	result := &Message{Kind: "unsupported"}
	if text != nil {
		id, err := ParseMessageID(text.Id)
		if err != nil {
			return result
		}
		result.ID, result.Revision, result.SentAt = id, text.Rev, text.SentAt
		if text.Sender != nil {
			result.SenderDID, _ = syntax.ParseDID(text.Sender.Did)
		}
		if result.SenderDID != "" && ValidateMessageText(text.Text) == nil {
			result.Kind = "text"
			result.Text = text.Text
		}
	} else if deleted != nil {
		result.ID, _ = ParseMessageID(deleted.Id)
		result.Revision, result.SentAt, result.Kind = deleted.Rev, deleted.SentAt, "deleted"
		if deleted.Sender != nil {
			result.SenderDID, _ = syntax.ParseDID(deleted.Sender.Did)
		}
	} else if system != nil {
		result.ID, _ = ParseMessageID(system.Id)
		result.Revision, result.SentAt = system.Rev, system.SentAt
	}
	return result
}

func conversationFromView(owner syntax.DID, view *upstream.ConvoDefs_ConvoView) (Conversation, error) {
	peer, err := directPeer(owner, view)
	if err != nil {
		return Conversation{}, err
	}
	if view.UnreadCount < 0 {
		return Conversation{}, ErrProviderResponse
	}
	result := Conversation{ID: ConversationID(view.Id), Revision: view.Rev, Status: *view.Status, Muted: view.Muted, UnreadCount: view.UnreadCount, Members: make([]Actor, 0, 2)}
	for _, member := range view.Members {
		actor := actorFromView(member)
		result.Members = append(result.Members, actor)
		if actor.DID == peer {
			result.Peer = actor
		}
	}
	if view.LastMessage != nil {
		result.LastMessage = messageFromView(view.LastMessage.ConvoDefs_MessageView, view.LastMessage.ConvoDefs_DeletedMessageView, view.LastMessage.ConvoDefs_SystemMessageView)
	}
	return result, nil
}
