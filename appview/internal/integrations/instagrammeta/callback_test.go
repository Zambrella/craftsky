package instagrammeta

import (
	"net/url"
	"testing"
)

func TestVerifyCallbackRequiresExactModeTokenAndNonEmptyChallenge(t *testing.T) {
	t.Parallel()

	const (
		verifyToken = "synthetic-verify-token"
		challenge   = "synthetic-callback-challenge"
	)
	if got, ok := VerifyCallback("subscribe", verifyToken, challenge, verifyToken); !ok || got != challenge {
		t.Fatalf("VerifyCallback(valid) = (%q, %t), want (%q, true)", got, ok, challenge)
	}

	for name, input := range map[string]struct {
		mode, token, challenge, expected string
	}{
		"wrong mode":      {"unsubscribe", verifyToken, challenge, verifyToken},
		"mode case":       {"Subscribe", verifyToken, challenge, verifyToken},
		"wrong token":     {"subscribe", "wrong", challenge, verifyToken},
		"token suffix":    {"subscribe", verifyToken + " ", challenge, verifyToken},
		"empty token":     {"subscribe", "", challenge, verifyToken},
		"empty expected":  {"subscribe", verifyToken, challenge, ""},
		"empty challenge": {"subscribe", verifyToken, "", verifyToken},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got, ok := VerifyCallback(input.mode, input.token, input.challenge, input.expected); ok || got != "" {
				t.Fatalf("VerifyCallback(invalid) = (%q, %t), want (\"\", false)", got, ok)
			}
		})
	}
}

func TestVerifyCallbackQueryRejectsMissingAndDuplicateParameters(t *testing.T) {
	t.Parallel()

	const token = "synthetic-verify-token"
	valid := url.Values{
		"hub.mode":         {"subscribe"},
		"hub.verify_token": {token},
		"hub.challenge":    {"synthetic-challenge"},
	}
	if got, ok := VerifyCallbackQuery(valid, token); !ok || got != "synthetic-challenge" {
		t.Fatalf("VerifyCallbackQuery(valid) = (%q, %t)", got, ok)
	}
	for _, key := range []string{"hub.mode", "hub.verify_token", "hub.challenge"} {
		missing := cloneValues(valid)
		delete(missing, key)
		if got, ok := VerifyCallbackQuery(missing, token); ok || got != "" {
			t.Errorf("missing %s returned (%q, %t)", key, got, ok)
		}
		duplicate := cloneValues(valid)
		duplicate[key] = append(duplicate[key], duplicate[key][0])
		if got, ok := VerifyCallbackQuery(duplicate, token); ok || got != "" {
			t.Errorf("duplicate %s returned (%q, %t)", key, got, ok)
		}
	}
}

func cloneValues(input url.Values) url.Values {
	output := make(url.Values, len(input))
	for key, values := range input {
		output[key] = append([]string(nil), values...)
	}
	return output
}
