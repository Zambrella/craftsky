package instagram

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"net/url"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

var (
	ErrVerificationUnavailable = errors.New("Instagram verification unavailable")
	ErrInstagramLinkConflict   = errors.New("Instagram link conflict")
)

type CreatedVerification struct {
	Attempt   VerificationAttempt
	Challenge string
	DMURL     string
}

func (CreatedVerification) String() string {
	return "created Instagram verification [REDACTED]"
}

type AccountView struct {
	State                InstagramLinkState
	Username             string
	Discoverable         bool
	ConflictPending      bool
	ReactivationRequired bool
	VerifiedAt           time.Time
}

func (AccountView) String() string {
	return "Instagram account view [REDACTED]"
}

type ConfirmationResult struct {
	State   VerificationAttemptState
	Account AccountView
}

type VerificationRepository interface {
	CreateVerificationAttempt(context.Context, CreateVerificationAttemptParams) (*VerificationAttempt, error)
	GetVerificationAttempt(context.Context, syntax.DID, uuid.UUID, time.Time) (*VerificationAttempt, error)
	CancelVerificationAttempt(context.Context, syntax.DID, uuid.UUID, time.Time) error
	ConfirmVerificationAttempt(context.Context, ConfirmVerificationAttemptParams) (ConfirmationResult, error)
}

type VerificationServiceOptions struct {
	Store     VerificationRepository
	Codec     *ChallengeCodec
	Now       func() time.Time
	NewID     func() uuid.UUID
	TTL       time.Duration
	DMURL     *url.URL
	HMACKey   []byte
	Available bool
}

type VerificationService struct {
	store     VerificationRepository
	codec     *ChallengeCodec
	now       func() time.Time
	newID     func() uuid.UUID
	ttl       time.Duration
	dmURL     string
	hmacKey   []byte
	available bool
}

func NewVerificationService(options VerificationServiceOptions) (*VerificationService, error) {
	service := &VerificationService{
		store:     options.Store,
		codec:     options.Codec,
		now:       options.Now,
		newID:     options.NewID,
		ttl:       options.TTL,
		hmacKey:   append([]byte(nil), options.HMACKey...),
		available: options.Available,
	}
	if service.now == nil {
		service.now = time.Now
	}
	if service.newID == nil {
		service.newID = uuid.New
	}
	if options.DMURL != nil {
		service.dmURL = options.DMURL.String()
	}
	if !service.available {
		return service, nil
	}
	if service.store == nil || service.codec == nil || service.ttl <= 0 || service.dmURL == "" || len(service.hmacKey) < challengeKeyMinBytes {
		return nil, errors.New("complete Instagram verification dependencies are required when enabled")
	}
	return service, nil
}

func (s *VerificationService) CreateVerification(ctx context.Context, owner syntax.DID) (CreatedVerification, error) {
	if s == nil || !s.available {
		return CreatedVerification{}, ErrVerificationUnavailable
	}
	issued, err := s.codec.Generate()
	if err != nil {
		return CreatedVerification{}, err
	}
	now := s.now().UTC()
	attempt, err := s.store.CreateVerificationAttempt(ctx, CreateVerificationAttemptParams{
		ID:        s.newID(),
		OwnerDID:  owner,
		Digest:    issued.Stored().Digest,
		ExpiresAt: now.Add(s.ttl),
		Now:       now,
	})
	if err != nil {
		return CreatedVerification{}, err
	}
	return CreatedVerification{Attempt: *attempt, Challenge: issued.Display(), DMURL: s.dmURL}, nil
}

func (s *VerificationService) GetVerification(ctx context.Context, owner syntax.DID, id uuid.UUID) (*VerificationAttempt, error) {
	if s == nil || s.store == nil {
		return nil, ErrVerificationUnavailable
	}
	return s.store.GetVerificationAttempt(ctx, owner, id, s.now().UTC())
}

func (s *VerificationService) CancelVerification(ctx context.Context, owner syntax.DID, id uuid.UUID) error {
	if s == nil || s.store == nil {
		return nil
	}
	return s.store.CancelVerificationAttempt(ctx, owner, id, s.now().UTC())
}

func (s *VerificationService) ConfirmVerification(ctx context.Context, owner syntax.DID, id uuid.UUID, discoverable bool) (ConfirmationResult, error) {
	if s == nil || !s.available || s.store == nil {
		return ConfirmationResult{}, ErrVerificationUnavailable
	}
	attempt, err := s.store.GetVerificationAttempt(ctx, owner, id, s.now().UTC())
	if err != nil {
		return ConfirmationResult{}, err
	}
	if attempt.State != AttemptPendingConfirmation && attempt.State != AttemptConfirmed {
		return ConfirmationResult{}, ErrInstagramStateTransition
	}
	digest := digestPrivateIdentifier(s.hmacKey, "igsid", attempt.CandidateIGSID)
	now := s.now().UTC()
	return s.store.ConfirmVerificationAttempt(ctx, ConfirmVerificationAttemptParams{
		AttemptID:         id,
		OwnerDID:          owner,
		LinkID:            s.newID(),
		ClaimID:           s.newID(),
		ConflictID:        s.newID(),
		IGSID:             attempt.CandidateIGSID,
		IGSIDDigest:       digest,
		Username:          attempt.CandidateUsername,
		Discoverable:      discoverable,
		Now:               now,
		ConflictExpiresAt: now.Add(365 * 24 * time.Hour),
	})
}

func digestPrivateIdentifier(key []byte, domain, value string) ChallengeDigest {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("craftsky:instagram-private:" + domain + ":v1\x00"))
	_, _ = mac.Write([]byte(value))
	var digest ChallengeDigest
	digest.Version = ChallengeDigestVersion
	copy(digest.Value[:], mac.Sum(nil))
	return digest
}
