package accountdeletion

import (
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

var ErrStatusUnauthorized = errors.New("status credential unauthorized")

type StatusAction string

const (
	StatusRead                  StatusAction = "read"
	StatusStartReauthentication StatusAction = "startReauthentication"
	StatusRetry                 StatusAction = "retry"
	StatusOrdinaryAPI           StatusAction = "ordinaryAPI"
	StatusPDSAPI                StatusAction = "pdsAPI"
)

type StatusGrant struct {
	JobID     string
	Owner     syntax.DID
	ExpiresAt time.Time
	Revoked   bool
}

func AuthorizeStatus(grant StatusGrant, now time.Time, jobID string, owner syntax.DID, action StatusAction) error {
	if grant.Revoked || !now.Before(grant.ExpiresAt) || grant.JobID != jobID || grant.Owner != owner {
		return ErrStatusUnauthorized
	}
	switch action {
	case StatusRead, StatusStartReauthentication, StatusRetry:
		return nil
	default:
		return ErrStatusUnauthorized
	}
}

type DeletionStatusView struct {
	JobID                 string `json:"jobId"`
	Status                Status `json:"status"`
	Phase                 Phase  `json:"phase"`
	RetryAllowed          bool   `json:"retryAllowed"`
	NeedsReauthentication bool   `json:"needsReauthentication"`
}

func ProjectDeletionStatus(jobID string, status Status, phase Phase, retryAllowed bool, needsReauthentication ...bool) DeletionStatusView {
	needsReauth := false
	if len(needsReauthentication) > 0 {
		needsReauth = needsReauthentication[0]
	}
	return DeletionStatusView{
		JobID: jobID, Status: status, Phase: phase,
		RetryAllowed: retryAllowed, NeedsReauthentication: needsReauth,
	}
}
