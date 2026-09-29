package index

import (
	"social.craftsky/appview/internal/sourcevalidation"
	"social.craftsky/appview/internal/tap"
)

type ValidationStatus = sourcevalidation.Status

const (
	ValidationPending = sourcevalidation.Pending
	ValidationValid   = sourcevalidation.Valid
	ValidationInvalid = sourcevalidation.Invalid
)

type SourceValidation = sourcevalidation.Result

func validateSourceRecord(event tap.Event) SourceValidation {
	return sourcevalidation.Validate(event)
}
