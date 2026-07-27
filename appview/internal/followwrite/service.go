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

type followRecord struct {
	Type    string     `json:"$type"`
	Subject syntax.DID `json:"subject"`
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

// HasDeterministicFollow distinguishes a replay of CraftSky's stable operation
// key from an unrelated follow that appeared before the worker completed.
func (s *Service) HasDeterministicFollow(
	ctx context.Context,
	owner syntax.DID,
	target syntax.DID,
	sessionID string,
	rkey syntax.RecordKey,
) (bool, error) {
	if s == nil || s.newPDS == nil || owner == "" || target == "" || rkey == "" {
		return false, ErrUnavailable
	}
	client, err := s.newPDS(ctx, owner, sessionID)
	if err != nil {
		return false, err
	}
	var record followRecord
	if _, err := client.GetRecord(ctx, owner, Collection, rkey.String(), &record); err != nil {
		if errors.Is(err, auth.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	if record.Type != Collection || record.Subject != target {
		return false, ErrUnavailable
	}
	return true, nil
}
