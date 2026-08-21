// appview/internal/auth/initialize_profile.go
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type IdentityCacheUpdater interface {
	UpsertCurrentHandle(ctx context.Context, did syntax.DID) error
}

type RepositoryTracker interface {
	AddRepo(context.Context, syntax.DID) error
}

// OnboardingProfileWrite is the only PDS mutation admitted while a login
// callback still owns departed/onboarding authority. The stable identity is
// scoped to the owner lifecycle generation rather than to one browser callback
// attempt, so a fresh callback cannot repeat an outcome-uncertain Put.
type OnboardingProfileWrite struct {
	OperationID     string
	MutationKey     string
	Owner           syntax.DID
	OwnerGeneration int64
	Record          map[string]any
}

// OnboardingProfileWriter persists the durable no-repeat attempt before it
// crosses the PDS boundary. It is implemented by the application composition
// layer so auth never imports the ordinary PDS-effect package.
type OnboardingProfileWriter interface {
	PutOnboardingProfile(context.Context, PDSClient, OnboardingProfileWrite) error
}

// ErrProfileInitFailed wraps any non-404 PDS failure during onboarding-
// on-login. Callers surface this as a profile_init_failed error page.
var ErrProfileInitFailed = errors.New("profile: init failed")

// ErrProfileDataInvalid indicates the fetched social.craftsky.actor.profile
// record fails lexicon validation. Callers surface this as a
// profile_data_invalid error page.
var ErrProfileDataInvalid = errors.New("profile: data invalid")

const (
	blueskyProfileNSID  = "app.bsky.actor.profile"
	craftskyProfileNSID = "social.craftsky.actor.profile"
	profileRecordKey    = "self"
)

// InitializeProfile performs onboarding-on-login side effects against
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
func InitializeProfile(
	ctx context.Context,
	client PDSClient,
	attempt CallbackAttempt,
	writer OnboardingProfileWriter,
) error {
	if client == nil || !attempt.validFor(attempt.Owner, attempt.State) ||
		attempt.Purpose != LoginOAuthPurpose {
		return fmt.Errorf("%w: invalid onboarding authority", ErrProfileInitFailed)
	}
	did := attempt.Owner
	// 1. Bluesky profile: presence is optional; only non-404 errors fail.
	var bskyRecord map[string]any
	if _, err := client.GetRecord(ctx, did, blueskyProfileNSID, profileRecordKey, &bskyRecord); err != nil {
		if !errors.Is(err, ErrRecordNotFound) {
			return fmt.Errorf("%w: get %s: %v", ErrProfileInitFailed, blueskyProfileNSID, err)
		}
	}

	// 2. Craftsky profile: present → validate; missing → write empty.
	var cskyRecord map[string]any
	_, err := client.GetRecord(ctx, did, craftskyProfileNSID, profileRecordKey, &cskyRecord)
	switch {
	case err == nil:
		if vErr := validateCraftskyProfile(cskyRecord); vErr != nil {
			return fmt.Errorf("%w: %v", ErrProfileDataInvalid, vErr)
		}
		return nil
	case errors.Is(err, ErrRecordNotFound):
		if writer == nil {
			return fmt.Errorf("%w: durable onboarding writer unavailable", ErrProfileInitFailed)
		}
		empty := map[string]any{
			"$type":  craftskyProfileNSID,
			"crafts": []string{},
		}
		identity := fmt.Sprintf("oauth-onboarding-profile:%s:%d", did, attempt.OwnerGeneration)
		if putErr := writer.PutOnboardingProfile(ctx, client, OnboardingProfileWrite{
			OperationID: identity, MutationKey: identity,
			Owner: did, OwnerGeneration: attempt.OwnerGeneration, Record: empty,
		}); putErr != nil {
			return fmt.Errorf("%w: put %s: %v", ErrProfileInitFailed, craftskyProfileNSID, putErr)
		}
		return nil
	default:
		return fmt.Errorf("%w: get %s: %v", ErrProfileInitFailed, craftskyProfileNSID, err)
	}
}

func InitializeProfileAndIdentityCache(
	ctx context.Context,
	client PDSClient,
	attempt CallbackAttempt,
	writer OnboardingProfileWriter,
	updater IdentityCacheUpdater,
	logger *slog.Logger,
	repositoryTrackers ...RepositoryTracker,
) error {
	if err := InitializeProfile(ctx, client, attempt, writer); err != nil {
		return err
	}
	did := attempt.Owner
	for _, tracker := range repositoryTrackers {
		if tracker == nil {
			continue
		}
		if err := tracker.AddRepo(ctx, did); err != nil && logger != nil {
			logger.Warn("Tap repository tracking request after profile initialization failed",
				authLogErrorAttrs("", "profile_init.repository_tracking", "tap")...)
		}
	}
	if updater == nil {
		return nil
	}
	if err := updater.UpsertCurrentHandle(ctx, did); err != nil {
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
