package observability

import (
	"strconv"

	"firebase.google.com/go/v4/errorutils"
	"firebase.google.com/go/v4/messaging"
)

// Provider error names are untrusted response data. Only these reviewed machine
// codes may enter diagnostics; neither provider prose nor response bodies do.
var providerDiagnosticCodes = map[string]struct{}{
	"pds.InvalidRequest": {}, "pds.InvalidRecord": {}, "pds.InvalidSwap": {},
	"pds.InvalidToken": {}, "pds.ExpiredToken": {}, "pds.AuthRequired": {},
	"pds.AuthenticationRequired": {}, "pds.InvalidRepo": {}, "pds.RepoNotFound": {},
	"pds.RecordNotFound": {}, "pds.RepoDeactivated": {}, "pds.RepoTakendown": {},
	"pds.AccountTakedown": {}, "pds.AccountDeactivated": {}, "pds.InvalidBlob": {},
	"pds.BlobNotFound": {}, "pds.BlobTooLarge": {}, "pds.RateLimitExceeded": {},
	"pds.InternalServerError": {}, "pds.UpstreamFailure": {}, "pds.NotEnoughResources": {},
	"pds.MethodNotFound": {}, "pds.NotImplemented": {},
	"fcm.THIRD_PARTY_AUTH_ERROR": {}, "fcm.SENDER_ID_MISMATCH": {}, "fcm.UNREGISTERED": {},
	"fcm.INVALID_ARGUMENT": {}, "fcm.QUOTA_EXCEEDED": {}, "fcm.INTERNAL": {}, "fcm.UNAVAILABLE": {},
	"firebase.INVALID_ARGUMENT": {}, "firebase.FAILED_PRECONDITION": {}, "firebase.OUT_OF_RANGE": {},
	"firebase.UNAUTHENTICATED": {}, "firebase.PERMISSION_DENIED": {}, "firebase.NOT_FOUND": {},
	"firebase.CONFLICT": {}, "firebase.ABORTED": {}, "firebase.ALREADY_EXISTS": {},
	"firebase.RESOURCE_EXHAUSTED": {}, "firebase.CANCELLED": {}, "firebase.DATA_LOSS": {},
	"firebase.UNKNOWN": {}, "firebase.INTERNAL": {}, "firebase.UNAVAILABLE": {}, "firebase.DEADLINE_EXCEEDED": {},
}

func knownProviderDiagnosticCode(code string) bool {
	_, ok := providerDiagnosticCodes[code]
	return ok
}

// Public SDK predicates expose fixed error classes without exposing the SDK's
// internal FirebaseError, its response body, recipient routing or credentials.
func firebaseDiagnostic(err error) (string, int) {
	status := 0
	if response := errorutils.HTTPResponse(err); response != nil && response.StatusCode >= 100 && response.StatusCode <= 599 {
		status = response.StatusCode
	}
	for _, candidate := range []struct {
		code    string
		matches func(error) bool
	}{
		{"fcm.THIRD_PARTY_AUTH_ERROR", messaging.IsThirdPartyAuthError},
		{"fcm.SENDER_ID_MISMATCH", messaging.IsSenderIDMismatch},
		{"fcm.UNREGISTERED", messaging.IsUnregistered},
		{"fcm.INVALID_ARGUMENT", messaging.IsInvalidArgument},
		{"fcm.QUOTA_EXCEEDED", messaging.IsQuotaExceeded},
		{"fcm.INTERNAL", messaging.IsInternal},
		{"fcm.UNAVAILABLE", messaging.IsUnavailable},
		{"firebase.INVALID_ARGUMENT", errorutils.IsInvalidArgument},
		{"firebase.FAILED_PRECONDITION", errorutils.IsFailedPrecondition},
		{"firebase.OUT_OF_RANGE", errorutils.IsOutOfRange},
		{"firebase.UNAUTHENTICATED", errorutils.IsUnauthenticated},
		{"firebase.PERMISSION_DENIED", errorutils.IsPermissionDenied},
		{"firebase.NOT_FOUND", errorutils.IsNotFound},
		{"firebase.CONFLICT", errorutils.IsConflict},
		{"firebase.ABORTED", errorutils.IsAborted},
		{"firebase.ALREADY_EXISTS", errorutils.IsAlreadyExists},
		{"firebase.RESOURCE_EXHAUSTED", errorutils.IsResourceExhausted},
		{"firebase.CANCELLED", errorutils.IsCancelled},
		{"firebase.DATA_LOSS", errorutils.IsDataLoss},
		{"firebase.UNKNOWN", errorutils.IsUnknown},
		{"firebase.INTERNAL", errorutils.IsInternal},
		{"firebase.UNAVAILABLE", errorutils.IsUnavailable},
		{"firebase.DEADLINE_EXCEEDED", errorutils.IsDeadlineExceeded},
	} {
		if candidate.matches(err) {
			return candidate.code, status
		}
	}
	return "", status
}

func providerDiagnosticFields(causes []DiagnosticCause) EventContext {
	fields := EventContext{}
	for _, cause := range causes {
		if _, found := fields["provider_code"]; !found && knownProviderDiagnosticCode(cause.ProviderCode) {
			fields["provider_code"] = cause.ProviderCode
		}
		if _, found := fields["upstream_status"]; !found {
			if status, err := strconv.Atoi(cause.Code); err == nil && len(cause.Code) == 3 && status >= 100 && status <= 599 {
				fields["upstream_status"] = status
			}
		}
	}
	return fields
}
