package instagram

import (
	"context"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

var (
	ErrInstagramFollowWriteUnavailable     = errors.New("Instagram follow write unavailable")
	ErrInvalidInstagramSuggestionPageLimit = errors.New("invalid Instagram suggestion page limit")
)

type SuggestionRepository interface {
	ListPendingSuggestions(context.Context, syntax.DID, int, *SuggestionCursor) ([]SuggestionEvidence, *SuggestionCursor, error)
	DismissSuggestion(context.Context, syntax.DID, uuid.UUID, time.Time) error
	ClaimSuggestionAcceptance(context.Context, syntax.DID, uuid.UUID, string, time.Time) (AcceptanceClaim, error)
	CompleteSuggestionAcceptance(context.Context, syntax.DID, uuid.UUID, InstagramSuggestionState, time.Time) (Suggestion, error)
	ResetSuggestionAcceptance(context.Context, syntax.DID, uuid.UUID, string, time.Time) error
	InvalidateSuggestion(context.Context, syntax.DID, uuid.UUID, time.Time) error
}

type InstagramFollowWriter interface {
	PutFollow(context.Context, syntax.DID, syntax.DID, syntax.RecordKey, time.Time) error
}

type SuggestionServiceOptions struct {
	Repository      SuggestionRepository
	Policy          InstagramSuggestionEligibilityPolicy
	Now             func() time.Time
	NewRkey         func(time.Time, uuid.UUID) syntax.RecordKey
	DefaultPageSize int
	MaxPageSize     int
}

type SuggestionService struct {
	repository      SuggestionRepository
	policy          InstagramSuggestionEligibilityPolicy
	now             func() time.Time
	newRkey         func(time.Time, uuid.UUID) syntax.RecordKey
	defaultPageSize int
	maxPageSize     int
}

func NewSuggestionService(options SuggestionServiceOptions) (*SuggestionService, error) {
	if options.Repository == nil || options.Policy == nil {
		return nil, errors.New("Instagram suggestion repository and policy are required")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.NewRkey == nil {
		options.NewRkey = newSuggestionFollowRkey
	}
	if options.DefaultPageSize == 0 {
		options.DefaultPageSize = 20
	}
	if options.MaxPageSize == 0 {
		options.MaxPageSize = 50
	}
	if options.DefaultPageSize < 1 || options.MaxPageSize < options.DefaultPageSize || options.MaxPageSize > 50 {
		return nil, errors.New("invalid Instagram suggestion page limits")
	}
	return &SuggestionService{
		repository: options.Repository, policy: options.Policy, now: options.Now,
		newRkey: options.NewRkey, defaultPageSize: options.DefaultPageSize,
		maxPageSize: options.MaxPageSize,
	}, nil
}

func (s *SuggestionService) ListSuggestions(ctx context.Context, owner syntax.DID, limit int, cursor *SuggestionCursor) ([]Suggestion, *SuggestionCursor, error) {
	if s == nil || s.repository == nil || s.policy == nil || owner == "" || limit < 0 {
		return nil, nil, ErrInvalidInstagramSuggestionPageLimit
	}
	if limit == 0 {
		limit = s.defaultPageSize
	}
	if limit > s.maxPageSize {
		limit = s.maxPageSize
	}
	evidence, next, err := s.repository.ListPendingSuggestions(ctx, owner, limit, cursor)
	if err != nil {
		return nil, nil, err
	}
	now := s.now().UTC()
	items := make([]Suggestion, 0, len(evidence))
	for _, item := range evidence {
		decision, err := s.policy.Evaluate(ctx, EligibilityAtList, SuggestionEligibilityRequest{
			ImporterDID: item.Suggestion.ImporterDID, TargetDID: item.Suggestion.TargetDID,
			ImportedUsername: item.ImportedUsername, Direction: item.Direction,
		})
		if err != nil {
			return nil, nil, err
		}
		if !decision.Eligible {
			if decision.Reason == EligibilitySafetyUnavailable {
				continue
			}
			if err := s.repository.InvalidateSuggestion(ctx, owner, item.Suggestion.ID, now); err != nil {
				return nil, nil, err
			}
			continue
		}
		items = append(items, item.Suggestion)
	}
	return items, next, nil
}

func (s *SuggestionService) DismissSuggestion(ctx context.Context, owner syntax.DID, id uuid.UUID) error {
	if s == nil || s.repository == nil {
		return errors.New("Instagram suggestion service unavailable")
	}
	if owner == "" || id == uuid.Nil {
		return nil
	}
	return s.repository.DismissSuggestion(ctx, owner, id, s.now().UTC())
}

func (s *SuggestionService) AcceptSuggestion(ctx context.Context, owner syntax.DID, id uuid.UUID, writer InstagramFollowWriter) (Suggestion, error) {
	if s == nil || s.repository == nil || s.policy == nil || owner == "" || id == uuid.Nil {
		return Suggestion{}, ErrInstagramResourceNotFound
	}
	now := s.now().UTC()
	rkey := s.newRkey(now, id)
	claim, err := s.repository.ClaimSuggestionAcceptance(ctx, owner, id, rkey.String(), now)
	if err != nil {
		return Suggestion{}, err
	}
	if claim.Suggestion.State == SuggestionAccepted || claim.Suggestion.State == SuggestionAlreadyFollowing {
		return claim.Suggestion, nil
	}
	decision, err := s.policy.Evaluate(ctx, EligibilityAtAccept, SuggestionEligibilityRequest{
		ImporterDID: claim.Suggestion.ImporterDID, TargetDID: claim.Suggestion.TargetDID,
		ImportedUsername: claim.ImportedUsername, Direction: claim.Direction,
	})
	if err != nil {
		_ = s.repository.ResetSuggestionAcceptance(context.WithoutCancel(ctx), owner, id, "eligibilityUnavailable", s.now().UTC())
		return Suggestion{}, err
	}
	if decision.Reason == EligibilityAlreadyFollowing {
		return s.repository.CompleteSuggestionAcceptance(ctx, owner, id, SuggestionAlreadyFollowing, s.now().UTC())
	}
	if !decision.Eligible {
		if decision.Reason == EligibilitySafetyUnavailable {
			_ = s.repository.ResetSuggestionAcceptance(context.WithoutCancel(ctx), owner, id, "eligibilityUnavailable", s.now().UTC())
			return Suggestion{}, ErrInstagramSuggestionIneligible
		}
		if err := s.repository.InvalidateSuggestion(ctx, owner, id, s.now().UTC()); err != nil {
			return Suggestion{}, err
		}
		return Suggestion{}, ErrInstagramSuggestionIneligible
	}
	if writer == nil || writer.PutFollow(ctx, owner, claim.Suggestion.TargetDID, claim.Operation.Rkey, claim.Operation.CreatedAt) != nil {
		_ = s.repository.ResetSuggestionAcceptance(context.WithoutCancel(ctx), owner, id, "followWriteUnavailable", s.now().UTC())
		return Suggestion{}, ErrInstagramFollowWriteUnavailable
	}
	return s.repository.CompleteSuggestionAcceptance(ctx, owner, id, SuggestionAccepted, s.now().UTC())
}

func newSuggestionFollowRkey(now time.Time, id uuid.UUID) syntax.RecordKey {
	clockID := (uint(id[0])<<2 | uint(id[1])>>6) & 0x3ff
	return syntax.RecordKey(syntax.NewTIDFromTime(now.UTC(), clockID))
}

var _ SuggestionRepository = (*SuggestionStore)(nil)
