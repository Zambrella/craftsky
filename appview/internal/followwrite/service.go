package followwrite

import (
	"context"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
)

const Collection = "app.bsky.graph.follow"

var (
	ErrUnavailable = errors.New("follow writer unavailable")
	ErrSelfFollow  = errors.New("self follow is not allowed")
)

// Service owns the common follow record shape and PDS write semantics. A nil
// rkey creates an ordinary record; a supplied rkey performs the deterministic
// PutRecord used by replay-safe Instagram acceptance.
type Service struct {
	newPDS auth.PDSClientFactory
}

func NewService(newPDS auth.PDSClientFactory) *Service {
	return &Service{newPDS: newPDS}
}

func (s *Service) Write(ctx context.Context, owner, target syntax.DID, sessionID string, rkey *syntax.RecordKey, createdAt time.Time) error {
	if s == nil || s.newPDS == nil || owner == "" || target == "" || createdAt.IsZero() {
		return ErrUnavailable
	}
	if owner == target {
		return ErrSelfFollow
	}
	client, err := s.newPDS(ctx, owner, sessionID)
	if err != nil {
		return err
	}
	record := map[string]any{
		"$type":     Collection,
		"subject":   target.String(),
		"createdAt": createdAt.UTC().Format(time.RFC3339),
	}
	if rkey != nil {
		return client.PutRecord(ctx, owner, Collection, rkey.String(), record)
	}
	_, _, err = client.CreateRecord(ctx, owner, Collection, record)
	return err
}
