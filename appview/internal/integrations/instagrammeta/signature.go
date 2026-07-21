package instagrammeta

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

const signaturePrefix = "sha256="

var ErrInvalidSignature = errors.New("invalid Instagram webhook signature")

// VerifySignatureValues rejects missing or duplicated signature headers.
func VerifySignatureValues(secret, body []byte, values []string) error {
	if len(values) != 1 {
		return ErrInvalidSignature
	}
	return VerifySignature(secret, body, values[0])
}

// VerifySignature verifies Meta's X-Hub-Signature-256 value over the exact
// request bytes. The header grammar is intentionally narrow so malformed or
// alternative algorithm values never reach payload decoding.
func VerifySignature(secret, body []byte, header string) error {
	if len(secret) == 0 || len(header) != len(signaturePrefix)+sha256.Size*2 {
		return ErrInvalidSignature
	}
	if header[:len(signaturePrefix)] != signaturePrefix {
		return ErrInvalidSignature
	}

	encoded := header[len(signaturePrefix):]
	for _, b := range []byte(encoded) {
		if (b < '0' || b > '9') && (b < 'a' || b > 'f') {
			return ErrInvalidSignature
		}
	}
	provided := make([]byte, sha256.Size)
	if _, err := hex.Decode(provided, []byte(encoded)); err != nil {
		return ErrInvalidSignature
	}

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	if !hmac.Equal(provided, mac.Sum(nil)) {
		return ErrInvalidSignature
	}
	return nil
}
