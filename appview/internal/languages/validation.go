package languages

import (
	"errors"
	"fmt"

	"golang.org/x/text/language"
)

var ErrInvalidPreferences = errors.New("invalid language preferences")
var ErrInvalidPostTags = errors.New("invalid post language tags")

func ValidatePostTags(tags []string) error {
	if len(tags) > 3 {
		return fmt.Errorf("%w: exceeds three entries", ErrInvalidPostTags)
	}
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		if !hasBCP47WireShape(tag) {
			return fmt.Errorf("%w: invalid tag", ErrInvalidPostTags)
		}
		if _, err := language.Parse(tag); err != nil {
			return fmt.Errorf("%w: invalid tag", ErrInvalidPostTags)
		}
		if _, exists := seen[tag]; exists {
			return fmt.Errorf("%w: duplicate tag", ErrInvalidPostTags)
		}
		seen[tag] = struct{}{}
	}
	return nil
}

func hasBCP47WireShape(tag string) bool {
	if tag == "" || tag[0] == '-' || tag[len(tag)-1] == '-' {
		return false
	}
	previousHyphen := false
	for index := 0; index < len(tag); index++ {
		character := tag[index]
		if character == '-' {
			if previousHyphen {
				return false
			}
			previousHyphen = true
			continue
		}
		previousHyphen = false
		if (character < 'a' || character > 'z') &&
			(character < 'A' || character > 'Z') &&
			(character < '0' || character > '9') {
			return false
		}
	}
	return true
}

func ValidatePreferences(preferences Preferences) error {
	if _, err := language.Parse(preferences.PrimaryLanguage); err != nil ||
		!IsSupportedBaseLanguage(preferences.PrimaryLanguage) {
		return fmt.Errorf("%w: invalid primaryLanguage", ErrInvalidPreferences)
	}

	seen := make(map[string]struct{}, len(preferences.ContentLanguages))
	for _, tag := range preferences.ContentLanguages {
		if _, err := language.Parse(tag); err != nil || !IsSupportedBaseLanguage(tag) {
			return fmt.Errorf("%w: invalid contentLanguages", ErrInvalidPreferences)
		}
		if _, exists := seen[tag]; exists {
			return fmt.Errorf("%w: duplicate contentLanguages", ErrInvalidPreferences)
		}
		seen[tag] = struct{}{}
	}
	return nil
}
