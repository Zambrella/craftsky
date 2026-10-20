package auth

import (
	"testing"
)

// IT-015 / RULE-004, NFR-001: optional consent is a distinct purpose;
// it may hand off credentials without running ordinary onboarding.
func TestChatConsentPurposeIsSeparateFromOnboarding(t *testing.T) {
	attempt := CallbackAttempt{Purpose: OAuthPurpose("chat")}
	if attempt.permitsOrdinaryOnboarding() {
		t.Fatal("chat consent must not initialize or reconcile a public profile")
	}
	attempt.State = "synthetic-state"
	attempt.Owner = "did:plc:synthetic-chat-owner"
	attempt.OwnerGeneration, attempt.AuthEpoch = 1, 1
	attempt.AttemptID[0] = 1
	if !attempt.validFor(attempt.Owner, attempt.State) {
		t.Fatal("chat consent cannot create its separately bound pending credential")
	}
}
