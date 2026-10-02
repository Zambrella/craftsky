// appview/internal/auth/initialize_profile.go
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/ownerlifecycle"
)

type IdentityCacheRefresher interface {
	RefreshCurrentHandle(ctx context.Context, did syntax.DID) error
}

type BlueskyProfileProjector interface {
	ProjectBlueskyProfile(context.Context, syntax.DID, syntax.CID, map[string]any) error
}

type CraftskyProfileProjector interface {
	ProjectCraftskyProfile(context.Context, syntax.DID, syntax.CID, map[string]any) error
}

// OnboardingProfileWrite is the only PDS mutation admitted while a login
// callback still owns departed/onboarding authority. The command journal
// derives its stable operation key from the owner and lifecycle generation.
type OnboardingProfileWrite struct {
	Owner           syntax.DID
	OwnerGeneration int64
	Record          map[string]any
}

// OnboardingProfileWriter persists the command and dispatch before crossing
// the PDS boundary. Auth never imports the command journal directly.
type OnboardingProfileWriter interface {
	PutOnboardingProfile(context.Context, PDSClient, OnboardingProfileWrite) (syntax.CID, error)
}

// ErrProfileInitFailed wraps any non-404 PDS failure during onboarding-
// on-login. Callers surface this as a profile_init_failed error page.
var ErrProfileInitFailed = errors.New("profile: init failed")

// ErrProfileDataInvalid indicates the fetched social.craftsky.actor.profile
// record fails lexicon validation. Callers surface this as a
// profile_data_invalid error page.
var ErrProfileDataInvalid = errors.New("profile: data invalid")

var ErrProfileCreationConflict = errors.New("profile: changed during creation")
var ErrProfileWriteUnresolved = errors.New("profile: write outcome unresolved")
var ErrProfileWriteRejected = errors.New("profile: write rejected")

// profileInitStageError preserves the internal cause while exposing only a
// fixed operation name to callback diagnostics.
type profileInitStageError struct {
	stage string
	err   error
}

func (e *profileInitStageError) Error() string { return e.err.Error() }
func (e *profileInitStageError) Unwrap() error { return e.err }

func profileInitializationFailureStage(err error) string {
	var staged *profileInitStageError
	if errors.As(err, &staged) {
		return staged.stage
	}
	return "profile_initialization"
}

// Only fixed classifications are emitted; upstream error text may contain
// credentials, callback parameters, or private provider response bodies.
func callbackFailureReason(err error) string {
	switch {
	case errors.Is(err, ownerlifecycle.ErrOwnerNotOnboarding):
		return "owner_not_onboarding"
	case errors.Is(err, ownerlifecycle.ErrGenerationChanged):
		return "owner_generation_changed"
	case errors.Is(err, ownerlifecycle.ErrTerminalOwner):
		return "terminal_owner"
	case errors.Is(err, ErrOAuthOwnerIneligible):
		return "owner_ineligible"
	case errors.Is(err, ErrCallbackAttemptInvalid):
		return "callback_authority_invalid"
	case errors.Is(err, ErrProfileWriteUnresolved):
		return "profile_write_unresolved"
	case errors.Is(err, ErrProfileCreationConflict):
		return "profile_creation_conflict"
	case errors.Is(err, ErrProfileWriteRejected):
		return "profile_write_rejected"
	case errors.Is(err, ErrProfileDataInvalid):
		return "profile_data_invalid"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "operation_failed"
	}
}

const (
	blueskyProfileNSID       = "app.bsky.actor.profile"
	craftskyProfileNSID      = "social.craftsky.actor.profile"
	profileRecordKey         = "self"
	profileProjectionTimeout = 2 * time.Second
)

type fetchedProfile struct {
	cid    syntax.CID
	record map[string]any
}

type initializedProfiles struct {
	bluesky  *fetchedProfile
	craftsky fetchedProfile
}

// initializeProfile performs onboarding-on-login side effects against
// the user's PDS:
//
//  1. Fetch app.bsky.actor.profile (non-404 errors fail).
//  2. Fetch social.craftsky.actor.profile.
//     - If present, validate it.
//     - If missing, write an empty {crafts: []} record.
//
// Called by the OAuth callback after ProcessCallback + SaveSession and
// before the Craftsky session token is returned. Per
// docs/superpowers/specs/2026-04-23-profile-onboarding-design.md §4, on
// any failure we fail the whole callback — the user is sent to an error
// page, their Craftsky session is not created.
func initializeProfile(
	ctx context.Context,
	client PDSClient,
	attempt CallbackAttempt,
	writer OnboardingProfileWriter,
) (*initializedProfiles, error) {
	if client == nil || !attempt.validFor(attempt.Owner, attempt.State) ||
		!attempt.permitsOrdinaryOnboarding() {
		return nil, fmt.Errorf("%w: invalid onboarding authority", ErrProfileInitFailed)
	}
	did := attempt.Owner
	// 1. Bluesky profile: presence is optional; only non-404 errors fail.
	var bskyRecord map[string]any
	bskyCID, err := client.GetRecord(ctx, did, blueskyProfileNSID, profileRecordKey, &bskyRecord)
	if err != nil {
		if !errors.Is(err, ErrRecordNotFound) {
			return nil, &profileInitStageError{"bluesky_profile_read", fmt.Errorf("%w: get %s: %w", ErrProfileInitFailed, blueskyProfileNSID, err)}
		}
	}
	var fetchedBluesky *fetchedProfile
	if err == nil {
		fetchedBluesky = &fetchedProfile{cid: syntax.CID(bskyCID), record: bskyRecord}
	}

	// 2. Craftsky profile: present → validate; missing → write empty.
	var cskyRecord map[string]any
	cskyCID, err := client.GetRecord(ctx, did, craftskyProfileNSID, profileRecordKey, &cskyRecord)
	switch {
	case err == nil:
		if vErr := validateCraftskyProfile(cskyRecord); vErr != nil {
			return nil, &profileInitStageError{"craftsky_profile_validation", fmt.Errorf("%w: %v", ErrProfileDataInvalid, vErr)}
		}
		return &initializedProfiles{
			bluesky:  fetchedBluesky,
			craftsky: fetchedProfile{cid: syntax.CID(cskyCID), record: cskyRecord},
		}, nil
	case errors.Is(err, ErrRecordNotFound):
		if writer == nil {
			return nil, fmt.Errorf("%w: durable onboarding writer unavailable", ErrProfileInitFailed)
		}
		empty := map[string]any{
			"$type":  craftskyProfileNSID,
			"crafts": []string{},
		}
		createdCID, putErr := writer.PutOnboardingProfile(ctx, client, OnboardingProfileWrite{
			Owner: did, OwnerGeneration: attempt.OwnerGeneration, Record: empty,
		})
		if errors.Is(putErr, ErrProfileCreationConflict) {
			// A repository-CAS conflict may reveal a profile created elsewhere.
			// Read and validate that record instead of overwriting it or replaying
			// a previously accepted creation after a later deletion.
			cskyCID, readErr := client.GetRecord(ctx, did, craftskyProfileNSID, profileRecordKey, &cskyRecord)
			if readErr == nil {
				if err := validateCraftskyProfile(cskyRecord); err != nil {
					return nil, &profileInitStageError{"craftsky_profile_validation", fmt.Errorf("%w: %w", ErrProfileDataInvalid, err)}
				}
				return &initializedProfiles{bluesky: fetchedBluesky, craftsky: fetchedProfile{cid: syntax.CID(cskyCID), record: cskyRecord}}, nil
			}
			if !errors.Is(readErr, ErrRecordNotFound) {
				return nil, &profileInitStageError{"craftsky_profile_read", fmt.Errorf("%w: %w", ErrProfileInitFailed, readErr)}
			}
		}
		if putErr != nil {
			return nil, &profileInitStageError{"craftsky_profile_write", fmt.Errorf("%w: put %s: %w", ErrProfileInitFailed, craftskyProfileNSID, putErr)}
		}
		return &initializedProfiles{
			bluesky:  fetchedBluesky,
			craftsky: fetchedProfile{cid: createdCID, record: empty},
		}, nil
	default:
		return nil, &profileInitStageError{"craftsky_profile_read", fmt.Errorf("%w: get %s: %w", ErrProfileInitFailed, craftskyProfileNSID, err)}
	}
}

func InitializeProfileAndIdentityCache(
	ctx context.Context,
	client PDSClient,
	attempt CallbackAttempt,
	writer OnboardingProfileWriter,
	blueskyProjector BlueskyProfileProjector,
	craftskyProjector CraftskyProfileProjector,
	updater IdentityCacheRefresher,
	logger *slog.Logger,
) error {
	profiles, err := initializeProfile(ctx, client, attempt, writer)
	if err != nil {
		return err
	}
	did := attempt.Owner
	if craftskyProjector != nil {
		projectionCtx, cancel := context.WithTimeout(ctx, profileProjectionTimeout)
		err := craftskyProjector.ProjectCraftskyProfile(
			projectionCtx, did, profiles.craftsky.cid, profiles.craftsky.record,
		)
		cancel()
		if err != nil {
			return &profileInitStageError{"craftsky_profile_projection", fmt.Errorf("%w: project %s: %v", ErrProfileInitFailed, craftskyProfileNSID, err)}
		}
	}
	if profiles.bluesky != nil && blueskyProjector != nil {
		projectionCtx, cancel := context.WithTimeout(ctx, profileProjectionTimeout)
		err := blueskyProjector.ProjectBlueskyProfile(
			projectionCtx, did, profiles.bluesky.cid, profiles.bluesky.record,
		)
		cancel()
		if err != nil && logger != nil {
			logger.Warn("Bluesky profile projection after profile initialization failed",
				authLogErrorAttrs("", "profile_init.bluesky_projection", "store")...)
		}
	}
	if updater == nil {
		return nil
	}
	if err := updater.RefreshCurrentHandle(ctx, did); err != nil {
		if logger != nil {
			logger.Warn("identity cache upsert after profile initialization failed",
				authLogErrorAttrs("", "profile_init.identity_cache", "store")...)
		}
	}
	return nil
}

// validateCraftskyProfile does a minimal shape check against
// social.craftsky.actor.profile. Stricter lexicon validation is future
// work; for now we just confirm crafts, if present, is an array of strings.
func validateCraftskyProfile(rec map[string]any) error {
	raw, ok := rec["crafts"]
	if !ok {
		return nil // crafts is optional per the lexicon.
	}
	arr, ok := raw.([]any)
	if !ok {
		return fmt.Errorf("crafts is not an array (got %T)", raw)
	}
	for i, item := range arr {
		if _, ok := item.(string); !ok {
			return fmt.Errorf("crafts[%d] is not a string (got %T)", i, item)
		}
	}
	return nil
}
