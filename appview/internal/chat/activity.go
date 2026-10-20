package chat

import (
	"context"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/chatdb"
	"time"
)

type ViewActivity struct {
	View           string         `json:"view"`
	ConversationID ConversationID `json:"conversationId,omitempty"`
	KnownRevision  *int64         `json:"knownRevision,omitempty"`
}

func (ViewActivity) String() string   { return "ChatViewActivity([REDACTED])" }
func (ViewActivity) GoString() string { return "ChatViewActivity([REDACTED])" }

type SyncSnapshot struct {
	Revision      int64      `json:"revision"`
	State         string     `json:"state"`
	LastSuccessAt *time.Time `json:"lastSuccessAt,omitempty"`
}

func (service *Service) Heartbeat(ctx context.Context, info auth.AuthInfo, activity ViewActivity) (SyncSnapshot, error) {
	var result SyncSnapshot
	if activity.KnownRevision != nil && *activity.KnownRevision < 0 {
		return result, ErrInvalidIdentifier
	}
	methods := []syntax.NSID{}
	switch activity.View {
	case "conversation":
		if !validOpaqueID(string(activity.ConversationID)) {
			return result, ErrInvalidIdentifier
		}
		methods = append(methods, "chat.bsky.convo.getConvo")
	case "inbox", "none":
		if activity.ConversationID != "" {
			return result, ErrInvalidIdentifier
		}
	default:
		return result, ErrInvalidIdentifier
	}
	err := service.withRPC(ctx, info, methods, func(ctx context.Context, transport *Transport) error {
		if activity.View == "conversation" {
			if _, err := currentConversation(ctx, transport, info.DID, activity.ConversationID); err != nil {
				return err
			}
		}
		return service.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
			q := chatdb.New(tx)
			if err := q.EnsureChatSync(ctx, info.DID.String()); err != nil {
				return err
			}
			if activity.View == "none" {
				if err := q.ClearChatViewActivity(ctx, chatdb.ClearChatViewActivityParams{OwnerDid: info.DID.String(), ChildTokenHash: info.ChildTokenHash[:]}); err != nil {
					return err
				}
			} else {
				if err := q.SetChatViewActivity(ctx, chatdb.SetChatViewActivityParams{OwnerDid: info.DID.String(), ChildTokenHash: info.ChildTokenHash[:], OauthSessionID: info.SessionID, View: activity.View, ConversationID: string(activity.ConversationID), ExpiresAt: dbTime(service.now().Add(12 * time.Second))}); err != nil {
					return err
				}
				if activity.View == "conversation" {
					if err := q.ScheduleChatForeground(ctx, chatdb.ScheduleChatForegroundParams{OwnerDid: info.DID.String(), NextDueAt: dbTime(service.now().Add(4 * time.Second))}); err != nil {
						return err
					}
				}
			}
			snapshot, err := q.ChatSyncSnapshot(ctx, info.DID.String())
			result = SyncSnapshot{Revision: snapshot.Version, State: snapshot.SafeState}
			if snapshot.LastSuccessAt.Valid {
				at := snapshot.LastSuccessAt.Time.UTC()
				result.LastSuccessAt = &at
			}
			return err
		})
	})
	return result, err
}
