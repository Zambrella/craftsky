package instagram

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	ChallengeAlphabet      = "23456789ABCDEFGHJKMNPQRSTVWXYZ"
	ChallengeDigestVersion = 1
	challengeSymbolCount   = 13
	challengeKeyMinBytes   = 32
	challengeByteLimit     = 240 // largest multiple of len(ChallengeAlphabet) below 256
)

const challengeDigestDomain = "craftsky:instagram-challenge:v1\x00"

var ErrInvalidChallenge = errors.New("invalid Instagram verification challenge")

// ChallengeCodec generates uniformly distributed display challenges and
// creates versioned keyed digests suitable for private persistence.
type ChallengeCodec struct {
	random io.Reader
	key    []byte
}

func NewChallengeCodec(random io.Reader, key []byte) (*ChallengeCodec, error) {
	if random == nil {
		return nil, errors.New("challenge entropy source is required")
	}
	if len(key) < challengeKeyMinBytes {
		return nil, fmt.Errorf("challenge digest key must be at least %d bytes", challengeKeyMinBytes)
	}
	return &ChallengeCodec{random: random, key: append([]byte(nil), key...)}, nil
}

// IssuedChallenge intentionally separates the short-lived display value from
// StoredChallenge, which contains only the keyed digest.
type IssuedChallenge struct {
	display string
	digest  ChallengeDigest
}

func (c IssuedChallenge) Display() string {
	return c.display
}

func (c IssuedChallenge) Stored() StoredChallenge {
	return StoredChallenge{Digest: c.digest}
}

func (IssuedChallenge) String() string {
	return "instagram challenge [REDACTED]"
}

// StoredChallenge is the complete persistence-safe challenge representation.
type StoredChallenge struct {
	Digest ChallengeDigest
}

func (StoredChallenge) String() string {
	return "stored instagram challenge [REDACTED]"
}

type ChallengeDigest struct {
	Version int
	Value   [sha256.Size]byte
}

func (d ChallengeDigest) Equal(other ChallengeDigest) bool {
	versionEqual := subtle.ConstantTimeEq(int32(d.Version), int32(other.Version))
	valueEqual := subtle.ConstantTimeCompare(d.Value[:], other.Value[:])
	return versionEqual&valueEqual == 1
}

func (d ChallengeDigest) IsZero() bool {
	var zero [sha256.Size]byte
	return subtle.ConstantTimeCompare(d.Value[:], zero[:]) == 1
}

func (ChallengeDigest) String() string {
	return "instagram challenge digest [REDACTED]"
}

func (c *ChallengeCodec) Generate() (IssuedChallenge, error) {
	symbols := make([]byte, 0, challengeSymbolCount)
	var randomByte [1]byte
	for len(symbols) < challengeSymbolCount {
		if _, err := io.ReadFull(c.random, randomByte[:]); err != nil {
			return IssuedChallenge{}, fmt.Errorf("read challenge entropy: %w", err)
		}
		if randomByte[0] >= challengeByteLimit {
			continue
		}
		symbols = append(symbols, ChallengeAlphabet[int(randomByte[0])%len(ChallengeAlphabet)])
	}

	display := "CSKY-" + string(symbols[:4]) + "-" + string(symbols[4:8]) + "-" + string(symbols[8:12]) + "-" + string(symbols[12:])
	digest, err := c.Digest(display)
	if err != nil {
		return IssuedChallenge{}, err
	}
	return IssuedChallenge{display: display, digest: digest}, nil
}

func (c *ChallengeCodec) Digest(input string) (ChallengeDigest, error) {
	canonical, err := CanonicalizeChallenge(input)
	if err != nil {
		return ChallengeDigest{}, err
	}
	mac := hmac.New(sha256.New, c.key)
	_, _ = mac.Write([]byte(challengeDigestDomain))
	_, _ = mac.Write([]byte(canonical))
	var value [sha256.Size]byte
	copy(value[:], mac.Sum(nil))
	return ChallengeDigest{Version: ChallengeDigestVersion, Value: value}, nil
}

func CanonicalizeChallenge(input string) (string, error) {
	input = strings.Trim(input, " \t\r\n\v\f")
	if len(input) != 21 {
		return "", ErrInvalidChallenge
	}

	canonical := make([]byte, len(input))
	for i := range input {
		b := input[i]
		if b >= 'a' && b <= 'z' {
			b -= 'a' - 'A'
		}
		canonical[i] = b
	}
	if string(canonical[:5]) != "CSKY-" || canonical[9] != '-' || canonical[14] != '-' || canonical[19] != '-' {
		return "", ErrInvalidChallenge
	}
	for _, index := range [...]int{5, 6, 7, 8, 10, 11, 12, 13, 15, 16, 17, 18, 20} {
		if !strings.ContainsRune(ChallengeAlphabet, rune(canonical[index])) {
			return "", ErrInvalidChallenge
		}
	}
	return string(canonical), nil
}
