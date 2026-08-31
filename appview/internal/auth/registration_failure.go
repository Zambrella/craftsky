package auth

import (
	"encoding/json"
	"errors"
	"time"
)

type RegistrationFailureCode string

const (
	RegistrationFailureCanceled            RegistrationFailureCode = "canceled"
	RegistrationFailureProviderUnavailable RegistrationFailureCode = "providerUnavailable"
	RegistrationFailureIncomplete          RegistrationFailureCode = "registrationIncomplete"
)

func (code RegistrationFailureCode) valid() bool {
	switch code {
	case RegistrationFailureCanceled, RegistrationFailureProviderUnavailable, RegistrationFailureIncomplete:
		return true
	default:
		return false
	}
}

func (code RegistrationFailureCode) MarshalJSON() ([]byte, error) {
	if !code.valid() {
		return nil, errors.New("invalid registration failure code")
	}
	return json.Marshal(string(code))
}

func (code RegistrationFailureCode) MarshalText() ([]byte, error) {
	if !code.valid() {
		return nil, errors.New("invalid registration failure code")
	}
	return []byte(code), nil
}

type TrustedRegistrationFailure struct {
	Metadata AuthRequestMetadata
	Code     RegistrationFailureCode
	cause    error
}

func newTrustedRegistrationFailure(
	metadata AuthRequestMetadata,
	code RegistrationFailureCode,
	cause error,
) (*TrustedRegistrationFailure, error) {
	if !code.valid() || cause == nil || !metadata.valid() ||
		metadata.Purpose != RegistrationOAuthPurpose || metadata.RequestState != AuthRequestReady ||
		metadata.ExpiresAt.IsZero() || !time.Now().Before(metadata.ExpiresAt) {
		return nil, ErrOAuthFlowInvalid
	}
	return &TrustedRegistrationFailure{Metadata: metadata, Code: code, cause: cause}, nil
}

func (*TrustedRegistrationFailure) Error() string {
	return "registration could not be completed"
}

func (failure *TrustedRegistrationFailure) Unwrap() error {
	return failure.cause
}
