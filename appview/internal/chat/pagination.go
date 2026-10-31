package chat

import (
	"context"
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"social.craftsky/appview/internal/chatdb"
)

var ErrInvalidCursor = errors.New("messaging continuation is invalid or expired")

type Continuation string

func (Continuation) String() string                 { return "Continuation([REDACTED])" }
func (Continuation) GoString() string               { return "Continuation([REDACTED])" }
func (c Continuation) MarshalText() ([]byte, error) { return []byte(c), nil }
func ParseContinuation(raw string) (Continuation, error) {
	if raw == "" {
		return "", nil
	}
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return "", ErrInvalidCursor
	}
	return Continuation(id.String()), nil
}

func (service *Service) resolveCursor(ctx context.Context, owner syntax.DID, generation int64, scope string, cursor Continuation) (string, error) {
	if cursor == "" {
		return "", nil
	}
	id, err := uuid.Parse(string(cursor))
	if err != nil {
		return "", ErrInvalidCursor
	}
	var result string
	err = service.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		result, err = chatdb.New(tx).GetChatContinuation(ctx, chatdb.GetChatContinuationParams{OwnerDid: owner.String(), ID: pgtype.UUID{Bytes: id, Valid: true}, Scope: scope, Generation: generation})
		return err
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInvalidCursor
	}
	return result, err
}

func (service *Service) retainCursor(ctx context.Context, owner syntax.DID, generation int64, scope, raw string) (Continuation, error) {
	if raw == "" {
		return "", nil
	}
	if len(raw) > 8192 {
		return "", ErrProviderResponse
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	err = service.store.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
		q := chatdb.New(tx)
		connection, err := q.LockChatAccount(ctx, owner.String())
		if err != nil {
			return err
		}
		if !connection.Enabled || connection.Generation != generation {
			return ErrConnectionChanged
		}
		if err := q.DeleteExpiredChatContinuations(ctx, owner.String()); err != nil {
			return err
		}
		if err := q.TrimChatContinuations(ctx, owner.String()); err != nil {
			return err
		}
		return q.CreateChatContinuation(ctx, chatdb.CreateChatContinuationParams{OwnerDid: owner.String(), ID: pgtype.UUID{Bytes: id, Valid: true}, Scope: scope, Generation: generation, UpstreamCursor: raw})
	})
	if err != nil {
		return "", err
	}
	return Continuation(id.String()), nil
}
