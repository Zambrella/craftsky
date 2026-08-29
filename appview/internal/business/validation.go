package business

import (
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

var ErrInvalidText = errors.New("business: invalid text")

type TextField string

const (
	TextFieldTagline      TextField = "tagline"
	TextFieldHoursNote    TextField = "hoursNote"
	TextFieldServiceArea  TextField = "serviceArea"
	TextFieldLocality     TextField = "locality"
	TextFieldProductTitle TextField = "productTitle"
	TextFieldEventName    TextField = "eventName"
	TextFieldEventSummary TextField = "eventSummary"
	TextFieldVenueName    TextField = "venueName"
	TextFieldImageAlt     TextField = "imageAlt"
)

type textBounds struct {
	graphemes int
	bytes     int
}

var textRules = map[TextField]textBounds{
	TextFieldTagline:      {graphemes: 100, bytes: 1000},
	TextFieldHoursNote:    {graphemes: 300, bytes: 3000},
	TextFieldServiceArea:  {graphemes: 200, bytes: 2000},
	TextFieldLocality:     {graphemes: 100, bytes: 1000},
	TextFieldProductTitle: {graphemes: 150, bytes: 1500},
	TextFieldEventName:    {graphemes: 200, bytes: 2000},
	TextFieldEventSummary: {graphemes: 1000, bytes: 10000},
	TextFieldVenueName:    {graphemes: 200, bytes: 2000},
	TextFieldImageAlt:     {graphemes: 1000, bytes: 1000},
}

type TextValidationError struct {
	Field TextField
}

func (e *TextValidationError) Error() string {
	return fmt.Sprintf("%s: %v", e.Field, ErrInvalidText)
}

func (e *TextValidationError) Unwrap() error {
	return ErrInvalidText
}

func ValidateText(field TextField, value string) error {
	bounds, ok := textRules[field]
	if !ok || !utf8.ValidString(value) || len(value) > bounds.bytes || uniseg.GraphemeClusterCount(value) > bounds.graphemes {
		return &TextValidationError{Field: field}
	}
	return nil
}
