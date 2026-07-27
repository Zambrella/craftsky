package instagrammeta

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/url"
)

const (
	callbackModeKey      = "hub.mode"
	callbackTokenKey     = "hub.verify_token"
	callbackChallengeKey = "hub.challenge"
)

// VerifyCallbackQuery rejects ambiguous duplicate query parameters before
// applying the callback decision.
func VerifyCallbackQuery(query url.Values, expectedToken string) (string, bool) {
	mode := query[callbackModeKey]
	token := query[callbackTokenKey]
	challenge := query[callbackChallengeKey]
	if len(mode) != 1 || len(token) != 1 || len(challenge) != 1 {
		return "", false
	}
	return VerifyCallback(mode[0], token[0], challenge[0], expectedToken)
}

// VerifyCallback decides Meta's webhook-subscription callback without ever
// reflecting the supplied challenge for an invalid request.
func VerifyCallback(mode, token, challenge, expectedToken string) (string, bool) {
	if mode != "subscribe" || challenge == "" || expectedToken == "" {
		return "", false
	}
	providedDigest := sha256.Sum256([]byte(token))
	expectedDigest := sha256.Sum256([]byte(expectedToken))
	if subtle.ConstantTimeCompare(providedDigest[:], expectedDigest[:]) != 1 {
		return "", false
	}
	return challenge, true
}
