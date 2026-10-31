package chat

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

type TextValidationError uint8

const (
	ErrTextBlank TextValidationError = iota + 1
	ErrTextGraphemeLimit
	ErrTextByteLimit
	ErrTextInvalidUnicode
)

func (err TextValidationError) Error() string {
	switch err {
	case ErrTextBlank:
		return "text must not be blank"
	case ErrTextGraphemeLimit:
		return "text exceeds grapheme limit"
	case ErrTextByteLimit:
		return "text exceeds UTF-8 byte limit"
	default:
		return "text contains invalid Unicode"
	}
}

// ValidateMessageText validates upstream text limits without modifying input.
func ValidateMessageText(text string) error {
	if !utf8.ValidString(text) {
		return ErrTextInvalidUnicode
	}
	// Match Dart String.trim, which also treats the Unicode BOM as blank.
	if strings.TrimFunc(text, func(r rune) bool { return unicode.IsSpace(r) || r == '\ufeff' }) == "" {
		return ErrTextBlank
	}
	if len(text) > 10_000 {
		return ErrTextByteLimit
	}
	if uniseg.GraphemeClusterCount(text) > 1_000 {
		return ErrTextGraphemeLimit
	}
	return nil
}
