package instagram

import (
	"errors"
	"fmt"
	"strings"
)

const (
	MaxInstagramUsernameLength = 30
	MaxImportEntries           = 10_000
)

var (
	ErrInvalidInstagramUsername = errors.New("invalid Instagram username")
	ErrInvalidImportDirection   = errors.New("invalid Instagram import direction")
	ErrTooManyImportEntries     = errors.New("too many Instagram import entries")
)

type ImportDirection string

const (
	DirectionFollowing ImportDirection = "following"
	DirectionFollower  ImportDirection = "follower"
)

func (d ImportDirection) Valid() bool {
	return d == DirectionFollowing || d == DirectionFollower
}

type ImportEntry struct {
	Username  string          `json:"username"`
	Direction ImportDirection `json:"direction"`
}

func (ImportEntry) String() string {
	return "Instagram import entry [REDACTED]"
}
func (e ImportEntry) GoString() string { return e.String() }

func NormalizeInstagramUsername(input string) (string, error) {
	username := strings.TrimSpace(input)
	if strings.HasPrefix(username, "@") {
		username = username[1:]
	}
	if len(username) == 0 || len(username) > MaxInstagramUsernameLength {
		return "", ErrInvalidInstagramUsername
	}

	normalized := make([]byte, len(username))
	for i := range username {
		character := username[i]
		if character >= 'A' && character <= 'Z' {
			character += 'a' - 'A'
		}
		if !((character >= 'a' && character <= 'z') ||
			(character >= '0' && character <= '9') ||
			character == '.' || character == '_') {
			return "", ErrInvalidInstagramUsername
		}
		normalized[i] = character
	}
	return string(normalized), nil
}

func NormalizeImportEntries(entries []ImportEntry) ([]ImportEntry, error) {
	result := make([]ImportEntry, 0, min(len(entries), MaxImportEntries))
	seen := make(map[string]struct{}, min(len(entries), MaxImportEntries))
	for index, entry := range entries {
		if !entry.Direction.Valid() {
			return nil, fmt.Errorf("%w at entry %d", ErrInvalidImportDirection, index)
		}
		username, err := NormalizeInstagramUsername(entry.Username)
		if err != nil {
			return nil, fmt.Errorf("%w at entry %d", err, index)
		}
		key := string(entry.Direction) + "\x00" + username
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, ImportEntry{Username: username, Direction: entry.Direction})
		if len(result) > MaxImportEntries {
			return nil, ErrTooManyImportEntries
		}
	}
	return result, nil
}
