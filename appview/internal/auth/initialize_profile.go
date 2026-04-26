// appview/internal/auth/initialize_profile.go
package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

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
func InitializeProfile(ctx context.Context, client PDSClient, did syntax.DID) error {
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
		empty := map[string]any{
			"$type":  craftskyProfileNSID,
			"crafts": []string{},
		}
		if putErr := client.PutRecord(ctx, did, craftskyProfileNSID, profileRecordKey, empty); putErr != nil {
			return fmt.Errorf("%w: put %s: %v", ErrProfileInitFailed, craftskyProfileNSID, putErr)
		}
		return nil
	default:
		return fmt.Errorf("%w: get %s: %v", ErrProfileInitFailed, craftskyProfileNSID, err)
	}
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
