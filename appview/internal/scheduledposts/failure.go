package scheduledposts

import "errors"

var (
	ErrAuthUnavailable         = errors.New("scheduled publication authentication unavailable")
	ErrObjectUnavailable       = errors.New("scheduled publication object unavailable")
	ErrPDSUnavailable          = errors.New("scheduled publication PDS unavailable")
	ErrPolicyInvalid           = errors.New("scheduled publication policy invalid")
	ErrMediaInvalid            = errors.New("scheduled publication media invalid")
	ErrRecordConflict          = errors.New("scheduled publication record conflict")
	ErrPublicationAmbiguous    = errors.New("scheduled publication outcome is ambiguous")
	ErrManualPublicationFailed = errors.New("manual scheduled publication failed")
)

type FailureDisposition string

const (
	FailureRetry          FailureDisposition = "retry"
	FailureNeedsAttention FailureDisposition = "needs_attention"
)

type FailureDecision struct {
	Disposition FailureDisposition
	SafeCode    string
}

func ClassifyPublicationFailure(err error) FailureDecision {
	switch {
	case errors.Is(err, ErrPolicyInvalid):
		return FailureDecision{Disposition: FailureNeedsAttention, SafeCode: "policy_invalid"}
	case errors.Is(err, ErrMediaInvalid):
		return FailureDecision{Disposition: FailureNeedsAttention, SafeCode: "media_invalid"}
	case errors.Is(err, ErrRecordConflict):
		return FailureDecision{Disposition: FailureNeedsAttention, SafeCode: "record_conflict"}
	case errors.Is(err, ErrAuthUnavailable):
		return FailureDecision{Disposition: FailureRetry, SafeCode: "auth_unavailable"}
	case errors.Is(err, ErrObjectUnavailable):
		return FailureDecision{Disposition: FailureRetry, SafeCode: "object_unavailable"}
	case errors.Is(err, ErrPDSUnavailable):
		return FailureDecision{Disposition: FailureRetry, SafeCode: "pds_unavailable"}
	default:
		return FailureDecision{Disposition: FailureRetry, SafeCode: "dependency_unavailable"}
	}
}
