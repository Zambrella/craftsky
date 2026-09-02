package business

import (
	"errors"
	"strings"
	"testing"
)

func TestBusinessTextValidation(t *testing.T) {
	tests := []struct {
		field        TextField
		maxGraphemes int
		maxBytes     int
	}{
		{TextFieldTagline, 100, 1000},
		{TextFieldHoursNote, 300, 3000},
		{TextFieldServiceArea, 200, 2000},
		{TextFieldProductTitle, 150, 1500},
		{TextFieldEventName, 200, 2000},
		{TextFieldEventSummary, 1000, 10000},
		{TextFieldVenueName, 200, 2000},
	}

	for _, tt := range tests {
		t.Run(string(tt.field), func(t *testing.T) {
			if err := ValidateText(tt.field, strings.Repeat("a", tt.maxGraphemes)); err != nil {
				t.Fatalf("grapheme boundary rejected: %v", err)
			}
			assertTextFieldError(t, tt.field, ValidateText(tt.field, strings.Repeat("a", tt.maxGraphemes+1)))

			combining := strings.Repeat("e\u0301", tt.maxGraphemes)
			if err := ValidateText(tt.field, combining); err != nil {
				t.Fatalf("combining grapheme boundary rejected: %v", err)
			}

			largeGrapheme := "👨‍👩‍👧‍👦"
			byteOver := strings.Repeat(largeGrapheme, tt.maxBytes/len(largeGrapheme)+1)
			assertTextFieldError(t, tt.field, ValidateText(tt.field, byteOver))
		})
	}
}

func assertTextFieldError(t *testing.T, field TextField, err error) {
	t.Helper()
	if !errors.Is(err, ErrInvalidText) {
		t.Fatalf("validation error = %v, want ErrInvalidText", err)
	}
	var fieldErr *TextValidationError
	if !errors.As(err, &fieldErr) {
		t.Fatalf("validation error %T, want *TextValidationError", err)
	}
	if fieldErr.Field != field {
		t.Fatalf("validation field = %q, want %q", fieldErr.Field, field)
	}
}
