package moderation

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

const caseReferencePrefix = "MOD-"

var ErrInvalidCaseReference = errors.New("invalid moderation case reference")

type CaseReference struct {
	id uuid.UUID
}

func ParseCaseReference(raw string) (CaseReference, error) {
	if len(raw) != len(caseReferencePrefix)+36 ||
		!strings.EqualFold(raw[:len(caseReferencePrefix)], caseReferencePrefix) {
		return CaseReference{}, ErrInvalidCaseReference
	}

	rawID := raw[len(caseReferencePrefix):]
	id, err := uuid.Parse(rawID)
	if err != nil || !strings.EqualFold(id.String(), rawID) || !validCaseUUID(id) {
		return CaseReference{}, ErrInvalidCaseReference
	}
	return CaseReference{id: id}, nil
}

func FormatCaseReference(id uuid.UUID) (string, error) {
	if !validCaseUUID(id) {
		return "", ErrInvalidCaseReference
	}
	return caseReferencePrefix + id.String(), nil
}

func (r CaseReference) String() string {
	formatted, _ := FormatCaseReference(r.id)
	return formatted
}

func (r CaseReference) UUID() uuid.UUID {
	return r.id
}

func validCaseUUID(id uuid.UUID) bool {
	return id != uuid.Nil && id.Version() == 4 && id.Variant() == uuid.RFC4122
}
