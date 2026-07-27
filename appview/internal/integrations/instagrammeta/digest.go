package instagrammeta

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
)

const (
	DigestVersion     = 1
	digestKeyMinBytes = 32
)

const (
	messageIDDigestDomain = "craftsky:instagram-message-id:v1\x00"
	challengeDigestDomain = "craftsky:instagram-challenge:v1\x00"
)

var ErrInvalidDigestInput = errors.New("invalid Instagram webhook digest input")

type ChallengeCanonicalizer func(input string) (string, error)

// KeyedDigest is safe for private persistence. Its diagnostic formats are
// deliberately redacted because even a digest is sensitive correlation data.
type KeyedDigest struct {
	Version int               `json:"-"`
	Value   [sha256.Size]byte `json:"-"`
}

func (d KeyedDigest) Equal(other KeyedDigest) bool {
	versionEqual := subtle.ConstantTimeEq(int32(d.Version), int32(other.Version))
	valueEqual := subtle.ConstantTimeCompare(d.Value[:], other.Value[:])
	return versionEqual&valueEqual == 1
}

func (KeyedDigest) String() string {
	return "instagram keyed digest [REDACTED]"
}

func (KeyedDigest) GoString() string {
	return "instagram keyed digest [REDACTED]"
}

// DigestCodec reduces upstream identifiers before they cross the durable-work
// boundary. Separate domains prevent a message ID from being confused with a
// challenge even if their source text is equal.
type DigestCodec struct {
	key                   []byte
	canonicalizeChallenge ChallengeCanonicalizer
}

func (*DigestCodec) String() string {
	return "instagram digest codec [REDACTED]"
}

func (*DigestCodec) GoString() string {
	return "instagram digest codec [REDACTED]"
}

func NewDigestCodec(key []byte, canonicalizeChallenge ChallengeCanonicalizer) (*DigestCodec, error) {
	if len(key) < digestKeyMinBytes {
		return nil, errors.New("Instagram digest key must be at least 32 bytes")
	}
	if canonicalizeChallenge == nil {
		return nil, errors.New("Instagram challenge canonicalizer is required")
	}
	return &DigestCodec{
		key:                   append([]byte(nil), key...),
		canonicalizeChallenge: canonicalizeChallenge,
	}, nil
}

func (c *DigestCodec) MessageID(input string) (KeyedDigest, error) {
	if input == "" {
		return KeyedDigest{}, ErrInvalidDigestInput
	}
	return c.digest(messageIDDigestDomain, input), nil
}

func (c *DigestCodec) Challenge(input string) (KeyedDigest, error) {
	if input == "" {
		return KeyedDigest{}, ErrInvalidDigestInput
	}
	canonical, err := c.canonicalizeChallenge(input)
	if err != nil || canonical == "" {
		return KeyedDigest{}, ErrInvalidDigestInput
	}
	return c.digest(challengeDigestDomain, canonical), nil
}

func (c *DigestCodec) digest(domain, input string) KeyedDigest {
	mac := hmac.New(sha256.New, c.key)
	_, _ = mac.Write([]byte(domain))
	_, _ = mac.Write([]byte(input))
	var value [sha256.Size]byte
	copy(value[:], mac.Sum(nil))
	return KeyedDigest{Version: DigestVersion, Value: value}
}
