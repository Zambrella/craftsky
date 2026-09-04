package video

import (
	"context"
	"errors"

	appbsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/ipfs/go-cid"
)

const maxVideoBlobBytes int64 = 300_000_000

type Blob struct {
	CID      syntax.CID
	MIMEType string
	Size     int64
}

type JobStatusClient interface {
	GetJobStatus(context.Context, string) (*appbsky.VideoDefs_JobStatus, error)
}

type VerificationErrorKind string

const (
	VerificationRejected    VerificationErrorKind = "rejected"
	VerificationUnavailable VerificationErrorKind = "unavailable"
)

type VerificationError struct {
	Kind VerificationErrorKind
}

func (e *VerificationError) Error() string {
	return "video completion verification " + string(e.Kind)
}

type CompletionVerifier struct {
	client JobStatusClient
}

func NewCompletionVerifier(client JobStatusClient) *CompletionVerifier {
	return &CompletionVerifier{client: client}
}

func (v *CompletionVerifier) Verify(ctx context.Context, owner syntax.DID, jobID string, submitted Blob) (Blob, error) {
	if v == nil || v.client == nil {
		return Blob{}, &VerificationError{Kind: VerificationUnavailable}
	}
	status, err := v.client.GetJobStatus(ctx, jobID)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return Blob{}, err
		}
		return Blob{}, &VerificationError{Kind: VerificationUnavailable}
	}
	if !validTerminalStatus(status) || status.JobId != jobID {
		return Blob{}, &VerificationError{Kind: VerificationRejected}
	}
	statusOwner, err := syntax.ParseDID(status.Did)
	if err != nil || statusOwner != owner {
		return Blob{}, &VerificationError{Kind: VerificationRejected}
	}
	if !blobsMatch(status, submitted) {
		return Blob{}, &VerificationError{Kind: VerificationRejected}
	}
	return submitted, nil
}

func validTerminalStatus(status *appbsky.VideoDefs_JobStatus) bool {
	if status == nil || status.Blob == nil {
		return false
	}
	if status.State == "JOB_STATE_COMPLETED" && status.Error == nil {
		return true
	}
	return status.Error != nil && *status.Error == "already_exists"
}

func blobsMatch(status *appbsky.VideoDefs_JobStatus, submitted Blob) bool {
	if submitted.MIMEType != "video/mp4" || submitted.Size <= 0 || submitted.Size > maxVideoBlobBytes {
		return false
	}
	submittedCID, err := cid.Parse(string(submitted.CID))
	if err != nil || submittedCID.String() != string(submitted.CID) {
		return false
	}
	remote := status.Blob
	remoteCID := cid.Cid(remote.Ref)
	return remoteCID.Defined() &&
		remoteCID.Equals(submittedCID) &&
		remote.MimeType == submitted.MIMEType &&
		remote.Size == submitted.Size
}
